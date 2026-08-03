# jqrepl 設計仕様書

- 作成日: 2026-08-02
- 元要求: `tmp/input01.md`

## 概要

jqコマンドを繰り返して実行して結果を確認するためのデスクトップアプリ。JSONを読み込み、
jqクエリを入力すると自動的に実行され、結果・スキーマ・実行履歴を並べて確認できる。

## 参考プロジェクト

`/workspaces/sirusita`（GitHub: `morststs/sirusita`）— Wails v2 (Go) + Svelte 5 +
Monaco Editor + TailwindCSS/Flowbite Svelte で構築されたマークダウンメモアプリ。
本プロジェクトは技術スタック・開発環境（Podmanコンテナ内完結のビルド）・
1エンティティ1ファイルの永続化方式をこのプロジェクトから踏襲する。

## 全体アーキテクチャ

- **アプリ形態:** Wails v2 デスクトップアプリ
- **バックエンド:** Go 1.23 + Wails v2、jq実行に `github.com/itchyny/gojq`
  （システムのjqバイナリに依存しない、純Go実装）
- **フロントエンド:** Svelte 5（runes）+ Vite 7、エディタは Monaco Editor、
  UIは TailwindCSS + Flowbite Svelte
- **開発/ビルド環境:** sirusitaの `Dockerfile` / `.devcontainer` をベースに流用
  （Podmanコンテナ内完結、Go/Node/Wailsをホストにインストールしない）
- **履歴の保存先:** `~/.jqrepl/history/{UUID}.json`（1履歴1ファイル）

### ディレクトリ構成

```
jqrepl/
├── main.go                # Wails エントリポイント
├── app.go                 # App構造体（ライフサイクル + フロント向けバインディング）
├── jq_service.go          # gojqによるクエリ実行・エラー整形・使用キー抽出
├── history_service.go     # 履歴CRUD（~/.jqrepl/history/*.json）
├── schema.go              # JSON値からキーツリー（型付き）を推論
├── Dockerfile / .devcontainer/   # sirusitaから流用・調整
├── frontend/
│   └── src/
│       ├── App.svelte           # 全体レイアウト（5ペイン + スプリッター）
│       ├── HistoryPane.svelte   # 左ペイン
│       ├── JsonPane.svelte      # 中央上ペイン
│       ├── SchemaPane.svelte    # 中央下ペイン
│       ├── QueryPane.svelte     # 右上ペイン（Monaco + intellisense）
│       ├── ResultPane.svelte    # 右下ペイン
│       └── jqCompletions.js     # Monaco補完プロバイダ
└── wails.json
```

## UI・操作フロー

**レイアウト:** 左ペイン（履歴一覧、幅可変）+ 中央（上:JSON／下:スキーマ、縦分割）+
右（上:指示／下:結果、縦分割）の3カラム。各ペインの見出しは日本語で明示する
（「JSON」「スキーマ」「jqクエリ」「実行結果」「履歴」）。

### JSONペイン（中央上）

- Monaco（JSON言語モード、シンタックスハイライト付き）で直接貼り付け・編集可能
- ヘッダーに「ファイルを開く」ボタン（`.json`ファイル選択ダイアログ、Wailsランタイム経由）
- JSONが変わるたびにスキーマペインの再推論とクエリの再実行をトリガーする

### スキーマペイン（中央下）

- JSONペインの内容から型付きキーツリーを推論して表示
  （例: `name: string`, `tags: array<string>`, `user.age: number`）
- 折りたたみ可能なツリー表示
- クエリペインの現在のクエリで参照されているキーを背景色でハイライトする。
  ハイライト対象は gojq のASTから抽出したキー名の集合と、ツリー上のキー名を
  **キー名の完全一致**で照合する（ネスト位置（パス）までは区別しない）。

### クエリペイン（右上）

- Monacoエディタ（jq専用の言語定義・トークナイザは持たず `plaintext` として扱い、
  補完機能のみ独自に提供する）
- 入力停止から **500ms** 後にデバウンスしてパース・実行する
  （数値は初期値。実装・使用感を見て調整可能とする）
- 補完プロバイダは以下を候補として表示する:
  - 現在のJSON全体から収集したキー名一覧（重複排除、フラット、ネスト位置は問わない）
  - jq組み込み関数の代表例: `map`, `select`, `keys`, `length`, `sort_by`,
    `group_by`, `has`, `to_entries`, `from_entries` など
- **パースエラー時:** エディタ下に赤字で `⚠ 構文エラー: {エラー内容}` を表示し、
  結果ペインは直前の内容のまま変更しない
- **実行時エラー時**（構文は正しいが評価に失敗する場合）も同様にエラーを表示し、
  結果ペインは更新しない

### 結果ペイン（右下）

- 実行結果のJSONを整形表示する（読み取り専用、シンタックスハイライトのみ）
- クエリが成功した場合のみ内容を更新する。エラー時は直前の成功結果を保持し続ける

### 履歴ペイン（左）

- クエリの実行が成功した場合のみ、新規履歴として自動的に一覧へ追加する
  （一覧は新しい順に並べる）
- 各項目には実行日時とクエリ文字列の先頭部分を表示する
- 項目をクリックすると、その時点の「元JSON・クエリ・実行結果」のセットを
  4つのペイン（JSON／スキーマ／クエリ／結果）へ一括で復元する

## バックエンドAPI・データ形式

```go
type HistoryEntry struct {
    ID        string    `json:"id"`
    JSON      string    `json:"json"`
    Query     string    `json:"query"`
    Result    string    `json:"result"`
    CreatedAt time.Time `json:"createdAt"`
}

type SchemaNode struct {
    Key      string       `json:"key"`
    Type     string       `json:"type"` // "string" | "number" | "boolean" | "null" | "object" | "array<...>"
    Children []SchemaNode `json:"children,omitempty"`
}

type RunResult struct {
    Result string `json:"result"`
    Error  string `json:"error"` // 空文字なら成功
}
```

### App（Wailsバインディング、フロントエンドから呼び出す関数）

| メソッド | 説明 |
|---|---|
| `RunQuery(jsonText, query string) RunResult` | gojqでクエリを実行する。パース/実行エラーは`RunResult.Error`に整形して返す。結果ペインを更新するかどうかはフロント側で`Error`の有無を見て判断する |
| `InferSchema(jsonText string) ([]SchemaNode, error)` | JSONからキーツリーを推論する |
| `ExtractUsedKeys(query string) ([]string, error)` | gojqのASTを解析し参照されているキー名一覧を返す。パース失敗時はエラーを返す |
| `OpenJSONFile() (string, error)` | ファイル選択ダイアログを開き、選択された`.json`ファイルの内容を文字列で返す |
| `SaveHistory(entry HistoryEntry) error` | `~/.jqrepl/history/{ID}.json`に保存する。IDが未指定の場合はUUIDを発行する |
| `ListHistory() ([]HistoryEntry, error)` | 履歴一覧を新しい順で返す |
| `DeleteHistory(id string) error` | 履歴を削除する（削除UIは今回のスコープ外だが、将来のためにAPIのみ用意する） |

### エラー整形方針

gojqのパースエラー（`gojq.ParseError`）・コンパイルエラー・実行時エラーを判別し、
それぞれ日本語の接頭辞（例:「構文エラー: 」「実行エラー: 」）を付けて返す。
行/列位置が取得できる場合は、将来的にMonacoのエラーマーカーにも反映できるようにしておく。

## テスト方針

### Go側（`go test`、sirusitaの`app_test.go`を参考にする）

- `jq_service_test.go`: 正常系クエリ実行、パースエラー、実行時エラーそれぞれの
  `RunResult`検証。`ExtractUsedKeys`のキー抽出精度
  （ネスト・配列・パイプ・複数キー参照などのケース）を検証する
- `schema_test.go`: ネストしたobject/array/null混在JSONからの`SchemaNode`推論を検証する
- `history_service_test.go`: 保存・一覧・削除のCRUDを検証する
  （一時ディレクトリを使用し、実際のホームディレクトリは汚さない）

### フロントエンド

sirusitaにもフロントエンドのユニットテストは無いため、同様にGoテストを中心とし、
フロントエンドは実装後に手動で確認する。実装完了後、`wails dev`で実アプリを起動し、
以下を目視確認する:

- JSON貼り付け → スキーマペインへの反映
- クエリ入力 → 500ms後の結果反映
- パースエラー時に結果ペインが変化しないこと
- スキーマペインのキーハイライトが指示ペインの内容と連動すること
- 履歴クリックで4ペインが復元されること

## スコープ外（今回は実装しない）

- 複数JSONドキュメントのタブ管理
- 履歴の削除UI（バックエンドAPIのみ用意する）
- jq構文の高度なシンタックスハイライト（トークナイザの自作）
- 文脈依存の補完（パイプ位置に応じた候補の絞り込み）

## プロジェクトドキュメント

実装完了時に、sirusitaの`CLAUDE.md`を参考にした`jqrepl`用の`CLAUDE.md`
（技術スタック・開発環境・プロジェクト構成・よく使うコマンドなど）を作成する。
