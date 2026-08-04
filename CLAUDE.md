# CLAUDE.md — shiboq

## プロジェクト概要

jqコマンドを繰り返し実行して結果を確認できる、Wails v2 + Svelte 5 で構築した
デスクトップアプリ（アプリ名: shiboq。「絞る」＋jqの「q」）。左から履歴・JSON/
スキーマ・クエリ/結果の5ペイン構成（履歴ペインはドラッグで幅を調整できる。
後述「App.svelteの並行処理」の下、「履歴ペインの幅（スプリッター）」参照）。
JSON/クエリの編集後500ms（デバウンス）でスキーマ再推論・使用キー抽出・
クエリ実行を自動的にまとめて行い、クエリ実行が成功し、かつ直近の履歴エントリと
(JSON, クエリ)が同一でない場合のみ履歴として自動保存する。実行履歴は
`~/.shiboq/history/{UUID}.json` に保存される
（保存件数は200件を上限に、古いものから自動的に間引かれる）。

参考プロジェクト: [morststs/sirusita](https://github.com/morststs/sirusita)（Wails v2 + Svelte 5 + Monaco構成のメモアプリ）。
技術スタック・開発環境・永続化パターンを踏襲している。

## 技術スタック

- **バックエンド:** Go 1.24（`go.mod`: `go 1.24.0` / `toolchain go1.24.13`）+ Wails v2.12.0
- **jq実行エンジン:** `github.com/itchyny/gojq` v0.12.19（システムのjqコマンドに依存しない純Go実装）
- **フロントエンド:** Svelte 5（runes）+ Vite 7
- **エディタ:** monaco-editor 0.55.1。`vs/language/json/monaco.contribution` を読み込んだ
  フル機能のJSON言語サービス（検証・スキーマ対応補完・ホバー）+ 独自言語ID `jq` による
  簡易補完（後述「Monaco構成の注意点」参照）
- **UIフレームワーク:** flowbite-svelte 1.x（`Button`, `Badge` を使用）+ TailwindCSS 4
- **その他Go依存:** `github.com/google/uuid`（履歴ファイル名）
- **フォント:** Google Fonts（Noto Sans JP / Source Code Pro）へのCDN参照は使わない。
  デスクトップアプリが起動のたびに外部ドメインへ通信するのは望ましくなく
  （オフライン時に失敗する・利用シグナルが漏れる）、`style.css`のフォント
  フォールバックスタック（`-apple-system, BlinkMacSystemFont, "Segoe UI",
  Roboto, sans-serif`等）にそのまま委ねる。フォントファイルをバンドルする
  選択肢もあったが、数MB単位のバイナリをアプリに含めるコストの方が
  フォールバック採用より大きいと判断した

## 開発環境

### Docker必須（コンテナ内で完結）

ホストにGo/Node/Wailsはインストールしない。全てのビルド・テストは
`./scripts/dev-run.sh <command>` 経由でコンテナ（`shiboq-dev`イメージ）内で実行する。

```bash
# 開発用イメージのビルド（初回 or Dockerfile変更時）
docker build -t shiboq-dev .

# 【重要】clone直後は必ず先にこれを実行する（理由は下の注記）
./scripts/dev-run.sh wails build

# Goのテスト実行
./scripts/dev-run.sh go test -v ./...

# Wailsビルド（Linux。出力: build/bin/shiboq）
./scripts/dev-run.sh wails build

# Wailsビルド（Windows。出力: build/bin/shiboq.exe）
# Dockerfileが同梱するmingw-w64でクロスコンパイルする。Windows実機は不要
./scripts/dev-run.sh wails build -platform windows/amd64

# フロントエンドのみビルド確認
./scripts/dev-run.sh sh -c "cd frontend && npm run build"

# Wailsバインディング再生成（Go側のメソッドシグネチャ変更時）
./scripts/dev-run.sh wails generate module
```

> **clone直後は `go test` / `go vet` が失敗する。** `main.go` の
> `//go:embed all:frontend/dist` はコンパイル時に `frontend/dist` の実体を要求するが、
> このディレクトリはビルド生成物で `.gitignore` 済みのため、cloneした時点では存在しない。
> エラーは `pattern all:frontend/dist: no matching files found`。
> 一度 `./scripts/dev-run.sh wails build`（または
> `./scripts/dev-run.sh sh -c "cd frontend && npm run build"`）を実行して
> `frontend/dist` を生成すれば、以降は通常どおりテストできる。

> Podman環境では `docker` を `podman` に読み替えても同じコマンドで動く。
> `scripts/dev-run.sh` は非rootユーザー（ホストと同じUID/GID）でコンテナを実行し、
> `GOPATH`（`.devcache/gopath`）・`GOCACHE`（`.devcache/gobuild`）・npmキャッシュ
> （`.devcache/npm`）を `/tmp` 直下の1階層パスにバインドマウントする
> （`/go`・`/root` はイメージ内でroot所有のため、そのままでは書き込めない。
> `/tmp/foo` のように `/tmp` のすぐ下の1階層ならDockerが中間ディレクトリを
> rootで自動生成する問題を避けられる）。`.devcache/` は`.gitignore`済み。

### イメージ内容（Dockerfile）

- `golang:1.24-bookworm` ベース、Node.js 22 LTS（NodeSource）、Wails CLI v2.12.0
  （`go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0` でgo.modと明示的に揃えている）
- libgtk-3-dev, libwebkit2gtk-4.0-dev/4.1-dev（Linuxビルド用）
- gcc-mingw-w64-x86-64, nsis（Windowsクロスコンパイル用。
  `wails build -platform windows/amd64` で `build/bin/shiboq.exe` を生成できる。
  生成物はWindows x86-64のGUIサブシステム実行ファイル。
  実行にはWindows側にWebView2ランタイムが必要（Win11は標準搭載、Win10は要インストール））

## プロジェクト構成

```
shiboq/
├── main.go                # Wails エントリポイント、App/JqService/HistoryServiceのBind
├── app.go                 # App構造体（ライフサイクル + OpenJSONFile）
├── jq_service.go          # gojqによるクエリ実行・エラー整形・使用キー抽出(AST解析)
├── schema.go              # JSON値からキーツリー（型付き）を推論
├── history_service.go     # 履歴CRUD（~/.shiboq/history/*.json）
├── jq_service_test.go     # jq_service.goのテスト（テストケースごとに個別のfunc、テーブル駆動ではない）
├── schema_test.go         # schema.goのテスト（同上）
├── history_service_test.go # history_service.goのテスト（同上）
├── app_test.go            # app.goの一部（readJSONFileWithLimit）のテスト（同上）
│                          #   ※ main.goにはテストが無い。app.goのOpenJSONFile自体は
│                          #   ネイティブダイアログに依存するためテスト不可、サイズ
│                          #   チェックのロジックだけreadJSONFileWithLimitへ切り出してテストする
├── Dockerfile              # 開発用コンテナイメージ定義（golang:1.24-bookworm）
├── scripts/dev-run.sh      # コンテナ内でコマンドを実行するヘルパー
├── .devcache/               # go/npmキャッシュのバインドマウント先（.gitignore済み、初回実行時に自動生成）
├── docs/superpowers/
│   ├── specs/2026-08-02-jqrepl-design.md    # 設計仕様書（アプリソースではない。
│   │         # アプリ名がjqreplだった当時の日付入り記録として、ファイル名・
│   │         # 本文とも意図的に当時のまま変更していない。shiboqへの改名は
│   │         # このファイルより後の出来事）
│   └── plans/2026-08-02-jqrepl-implementation.md # 実装計画書（アプリソースではない、同上）
├── frontend/
│   ├── svelte.config.js    # vitePreprocess({ script: true })（flowbite-svelte対応、削除不可）
│   ├── vite.config.js      # Vite + svelte + @tailwindcss/vite
│   ├── src/
│   │   ├── main.js             # Svelte マウント
│   │   ├── style.css           # グローバルスタイル（Tailwind + Flowbite）
│   │   ├── monaco.js           # Monaco Editorの構成 + 'jq'言語ID登録（後述「Monaco構成の注意点」）
│   │   ├── jqCompletions.js    # jqクエリペイン用のMonaco補完プロバイダ
│   │   ├── App.svelte          # ルート（状態管理 + 5ペインのレイアウト + 自動実行パイプライン。後述「App.svelteの並行処理」）
│   │   ├── HistoryPane.svelte  # 左ペイン：実行履歴一覧
│   │   ├── JsonPane.svelte     # 中央上ペイン：元のJSON（編集可、Monaco）
│   │   ├── SchemaPane.svelte   # 中央下ペイン：キーツリー（使用中キーをハイライト）
│   │   ├── SchemaTreeNode.svelte # スキーマツリーの再帰描画コンポーネント
│   │   ├── QueryPane.svelte    # 右上ペイン：jqクエリ入力（Monaco + 補完）
│   │   └── ResultPane.svelte   # 右下ペイン：実行結果（読み取り専用Monaco）
│   └── wailsjs/             # Wails自動生成バインディング（編集不可・`wails generate module`で再生成）
├── build/bin/               # ビルド出力先（shiboq、.gitignore済み）
└── wails.json
```

## Go Backend API

### App（app.go）

| メソッド | 説明 |
|---------|------|
| `OpenJSONFile()` | ネイティブのファイル選択ダイアログを開き、選択された`.json`の内容を返す（キャンセル時は空文字とnilエラー）。選択されたファイルが`maxOpenJSONFileSize`（20MB）を超える場合は読み込まず、日本語のエラーメッセージを返す（巨大ファイル選択でUIが応答不能になるのを防ぐ） |

### JqService（jq_service.go）

| メソッド | 説明 |
|---------|------|
| `RunQuery(jsonText, query, raw)` | jqクエリを実行し `{result, error, errorKind}` を返す。エラー時は `error` に日本語メッセージ、`result` は空文字。複数出力（`.a, .b`等）は改行区切りで連結する。`raw` は jq の `-r` 相当で、true のとき結果が**文字列の場合に限り**引用符とエスケープを外す（文字列以外は raw によらずJSON表現のまま）。詳細は下記「RunQueryの実行制限とエラー種別」参照 |
| `ExtractUsedKeys(query)` | クエリのASTを解析し、参照されているキー名一覧を返す（重複排除、順不同）。ドット記法（`.foo`）に加え、文字列補間（`"\(.name)"`）・単項演算子（`-.foo`）・分解パターン（`. as {a: $x}` や `reduce ... as {id: $i}`）内のキー参照も対象。`.["foo"]` のようなブラケット記法によるキー参照のみ対象外（既知の制約） |
| `InferSchema(jsonText)` | JSONからキーツリー（型付き）を推論する（`schema.go`のInferSchemaを呼ぶだけ） |

#### RunQueryの実行制限とエラー種別

500msデバウンスで自動実行される都合上、`repeat(.)`や`range(1e9)`のような
編集途中のクエリでも毎回goroutineが起動する。無制限に実行させると
終了しないgoroutineの蓄積・メモリの際限ない消費につながるため、以下の
制限を設けている（`JqService`のフィールドで、いずれもテストから小さい値に
差し替え可能。ゼロ値の場合は`NewJqService()`のデフォルトが使われる）。

- `Timeout`（デフォルト5秒）: `gojq.Code.RunWithContext`にcontextを渡し、
  タイムアウトしたら打ち切る。gojqはcontextのキャンセルをIterのエラー値
  として返すため、既存の`if err, ok := v.(error)`分岐で拾える
- `MaxOutputs`（デフォルト10,000件）・`MaxOutputBytes`（デフォルト10MB）:
  タイムアウトを待たず、生成された出力そのものが多すぎる／大きすぎる
  場合に打ち切る（`repeat(.)`のような高速に出力し続けるクエリがタイムアウト
  猶予中にメモリを食い潰すのを防ぐ）

`RunResult.ErrorKind`（`"json"` / `"syntax"` / `"compile"` / `"runtime"`）で
エラーの出処を判別できる。フロントエンド（App.svelte）は`"json"`（入力JSON
自体の不正）をクエリペインのエラーバーには出さない。jqクエリ側の構文エラーは
`gojq.Parse`失敗時に`"構文エラー: "`、`gojq.Compile`失敗時（未定義関数など）
は`"コンパイルエラー: "`を接頭辞にして区別する。タイムアウト・出力上限超過は
`ErrorKind: "runtime"`として扱う。

`nan`/`infinite`のようなjqの組み込み関数が返すfloat64のNaN/±Infは
`encoding/json`がMarshalできないため、`normalizeNonFiniteFloats`で
再帰的に`null`へ正規化してから整形する（本物のjqの`nan`表示に合わせている）。

### HistoryService（history_service.go）

| メソッド | 説明 |
|---------|------|
| `SaveHistory(entry)` | 履歴を保存する。IDが空ならUUIDを発行する（空でない場合はUUID形式を検証）。直近（最新）のエントリと`(JSON, Query)`が完全一致する場合は新規ファイルを書き込まずスキップし、既存エントリをそのまま返す（jqはキー不在時に`null`を返すだけでエラーにしないため、対策が無いと`.address.city`と打つ間の`.a`・`.add`・`.addr`…が全て履歴に残ってしまう）。保存後は`MaxEntries`（デフォルト200件、`defaultMaxHistoryEntries`）を超えた古いエントリを間引く（`pruneOldEntries`）。書き込みは同ディレクトリへの一時ファイル作成＋`os.Rename`によるアトミック書き込み |
| `ListHistory()` | 履歴を新しい順（CreatedAt降順、同時刻は ID降順でタイブレーク＝`sort.SliceStable`）で返す。IDが空・不正な形式（UUID以外）のエントリは、Svelteの`{#each ... (entry.id)}`キー重複による例外を防ぐため除外する |
| `DeleteHistory(id)` | 履歴を削除する（削除UIは未実装、APIのみ） |

履歴ディレクトリは`0700`、ファイルは`0600`で作成する（貼り付けたJSONに
トークンやPIIが含まれ得るため、他ユーザーから読めないようにする）。

## Monaco構成の注意点（frontend/src/monaco.js）

`monaco.js` は `monaco-editor/esm/vs/language/json/monaco.contribution` を読み込んでおり、
JSONペイン（`JsonPane.svelte`）はJSONの構文チェックやホバー等を含む
**フル機能のJSON言語サービス**を使っている（「JSON構文のみのスリム構成」ではない）。
このJSON言語サービスは専用の `json.worker` を必要とするため、
`MonacoEnvironment.getWorker(_moduleId, label)` は `label === 'json'` のときだけ
`JsonWorker`（`monaco-editor/esm/vs/language/json/json.worker?worker`）を返し、
それ以外（`'jq'` 言語のQueryPane、`'jq-result'` 言語のResultPane等）は通常の
`EditorWorker` を返すよう振り分けている。

**このlabel分岐は動作に必須。** 過去に `getWorker()` がlabelを無視して常に
`EditorWorker` を返す実装になっており、JSON言語サービスがworker経由で失敗する
不具合が実際に発生した（修正コミット: `fix: wire the JSON language worker in Monaco`）。
分岐を削除・簡略化すると、ビルドは通ってもJSONの検証・補完等が実行時に静かに壊れるため注意。

結果ペイン（`ResultPane.svelte`）は `'json'` 言語を使っていない。`RunQuery` は
複数出力（`.a, .b` や `.[]` 等）を改行区切りで連結するため、出力全体としては
有効なJSON文書にならないことがある（jqとしては正しい仕様）。`'json'` 言語だと
フル機能のJSON言語サービスが構文検証してしまい、正当な複数出力に
「End of file expected」の赤波線が付く。そのため結果ペイン専用に、検証を
持たない軽量なMonarchトークナイザ（文字列・数値・true/false/null・括弧の
ハイライトのみ）を独自言語ID `'jq-result'` として登録している
（`monaco.languages.setMonarchTokensProvider('jq-result', {...})`）。
JSON入力ペインの検証は意図した機能なので、こちらは変更しないこと。

jqクエリ用の言語ID `jq` は `monaco.languages.register({ id: 'jq' })` で登録されているのみで、
独自トークナイザ（シンタックスハイライト）は持たない。`jqCompletions.js` の
`registerJqCompletions()` が、現在のJSONから収集したフラットなキー名一覧と代表的な
jq組み込み関数（`map`, `select`, `keys` 等）を候補として返す補完プロバイダを登録する。

## App.svelteの並行処理（フロントエンド）

`evaluate()`（JSON/スキーマ/使用キー/クエリ実行をまとめて行う自動実行パイプライン）は、
JSON/クエリの変更のたびに500ms後に呼ばれるが、連続編集時には複数回の`evaluate()`が
同時に飛行中になり得る。古い呼び出しが新しい呼び出しの結果を上書きしないよう、
以下のガードを設けている。将来編集する際もこれらのガードを外さないこと。

- **`evaluateToken`:** `evaluate()`開始時にインクリメントして捕捉するモノトニックなトークン。
  各`await`の直後（catchブロックも含む）で `if (token !== evaluateToken) return;` を行い、
  トークンが変わっていれば（＝自分より新しい呼び出しが既に始まっていれば）状態を一切書き込まずに中断する。
- **`historyRequestToken`:** `refreshHistory()`自身の呼び出しごとに採番するトークン。
  `onMount`と`evaluate()`の両方から呼ばれるため、古い呼び出しの`ListHistory()`応答が
  新しい呼び出しの応答より後に返ってきても、一覧を古い内容で上書きしないように自己ガードする。
- **デバウンスの`$effect`:** `setTimeout(evaluate, 500)`をセットし、`clearTimeout`する
  ティアダウン関数を返す。エフェクトの再実行・アンマウント時に前回のタイマーを確実に破棄する。
- **`restoredSignature`（履歴復元後の再保存抑制）:** 履歴から復元した直後、その内容のまま
  評価しても再度履歴に保存しないようにするための仕組み。単純なbooleanフラグだと
  「復元後、次に実行される`evaluate()`」の内容がどうであれ無条件に保存をスキップしてしまい、
  復元後に内容を編集して実行した正当な結果まで保存されずに失われるバグになる。
  そのため、復元した内容そのものの署名（`signatureOf(json, query)` = `JSON.stringify([json, query])`）
  を保持し、`evaluate()`が実際に評価する内容の署名と一致する場合に限りスキップする。
  内容が変化すれば自然に「復元済み」とはみなされなくなるため、明示的にnullへ戻す処理は不要。
  **初期値はSAMPLE_JSON＋初期クエリ（`.name`）の署名にしてある。** `null`のままだと、
  起動直後（未操作）にマウント時の最初の`evaluate()`がこの初期状態を新規実行結果として
  履歴に保存してしまい、アプリを起動するだけで（何も操作しなくても）履歴が1件増える
  （起動50回で同一内容の履歴が50件になる）。この初期化は、後から偶然まったく同じ内容
  （SAMPLE_JSON＋`.name`）に戻した場合の保存も同様にスキップする、という既知のトレード
  オフを伴う（`restoreHistory()`後に元の内容へ戻した場合と同じ性質の制約であり、新規に
  持ち込んだものではない）。

### エラー表示の振り分けと結果ペインの陳腐化表示

- `RunQuery`が失敗した場合、`errorMessage`（クエリペインのエラーバー）には
  `r.errorKind === 'json'` のときは何も入れない。入力JSON自体が不正なときの
  詳細は`schemaError`（JSON/スキーマペイン側）に既に出ているため、クエリ自体は
  悪くないのにクエリペインへもエラーが出る「同じ原因で2箇所にエラーが出て、
  しかも片方は見当違いのペインに出る」問題を避けるため。
- `resultStale`は「直近の`RunQuery`が失敗し、結果ペインの内容が最新ではない」
  ことを示す独立したフラグ。`errorMessage`と連動させていない（`errorKind === 'json'`
  のときは`errorMessage`を空にするが、結果ペインの内容はやはり古いままなので）。
  `ResultPane`は`stale`propが真の間、ヘッダーに「前回の成功結果」バッジを出し、
  エディタを薄く表示する。
- `schemaError`にはバックエンド（`schema.go`）のエラーメッセージ（`invalid character
  'x' at offset N`等）をそのまま表示する。汎用文言に潰さない。

### 履歴ペインの幅（スプリッター）

`historyWidth`（`$state`、初期値260px）をドラッグでリサイズできる。
参考プロジェクト [sirusita](https://github.com/morststs/sirusita) の`App.svelte`の
`startDrag`/`onDrag`/`stopDrag`パターンを踏襲しており、`mousedown`で
`window`に`mousemove`/`mouseup`リスナーを登録し、`mouseup`（`stopDrag`相当）で
リスナーを解除して幅を`localStorage`（キー: `shiboq.historyWidth`）に保存する。
`onDestroy`でも`stopHistoryDrag()`を呼び、ドラッグ中にアンマウントされても
`window`リスナーが残留しないようにする。幅は`HISTORY_MIN_WIDTH`(160)〜
`HISTORY_MAX_WIDTH`(500)にクランプする。中央/右列の上下比率（JSON/スキーマ、
クエリ/結果）は引き続き固定（`flex: 1`で等分、リサイズ不可）。

### 履歴保存の失敗通知

`SaveHistory`が失敗した場合、実行結果自体は表示され続ける（評価パイプラインを
壊さない）が、気づけないとユーザーは履歴が保存されていると誤解し続ける。
そのため`showToast('履歴を保存できませんでした')`で3秒間のトースト通知を
画面右下に出す（モーダルにはしない）。

## 既知の制約（スコープ外）

- 複数JSONドキュメントのタブ管理は無い
- 履歴の削除UIは無い（バックエンドAPIのみ）
- jqクエリ（QueryPane）のシンタックスハイライトは無い（独自トークナイザ未実装、補完のみ）。
  結果ペイン（ResultPane）は検証無しの簡易Monarchトークナイザ（`'jq-result'`）を持つ
- 補完・使用キーハイライトはフラットなキー名一致であり、jqのパイプ位置に応じた
  文脈依存の絞り込みは行わない
- `.["キー名"]` のようなブラケット記法によるキー参照は、使用キー抽出の対象外
- **正規表現フラグは `g` / `i` / `m` の3つのみ**（`test` / `match` / `capture` / `scan`
  / `splits` / `sub` / `gsub`）。jq本家（Oniguruma）が対応する `x`（空白とコメントを
  無視する拡張モード）・`s`・`n`・`p`・`l` は使えない。内蔵する gojq が Go の
  `regexp` を使っており、Go に拡張モードが無いことによる上流の制約
  （gojq v0.12.19 の `compileRegexp` が `g`/`i`/`m` 以外を弾く）。
  `gojq.WithFunction` で `test` 等を差し替えても組み込みが優先されるため、
  アプリ側で回避することはできない。該当フラグを渡した場合は、対応フラグを
  併記した日本語エラー（`unsupportedRegexFlagMessage`）を表示する
- リサイズ可能なのは履歴ペインの幅のみ（スプリッター）。中央/右列の上下比率
  （JSON/スキーマ、クエリ/結果）は`flex: 1`で等分固定、リサイズ不可
- 履歴ペインのスプリッターはマウスドラッグのみで、キーボード操作の代替手段は無い
- 単一の`RunWithContext`呼び出し内で大きな値を1回で構築するクエリ（例:
  `[range(100000000)]`のように出力自体は1件だが構築コストが大きいもの）は、
  Timeoutで最終的には打ち切られるが、打ち切りまでの間にメモリを消費し得る。
  `MaxOutputs`/`MaxOutputBytes`は「出力として確定した値」の件数・サイズを
  対象にしたガードであり、単一の巨大な値の構築中は効かない

## コーディング規約

- Go: 標準フォーマット（`gofmt`）。全て `package main`（sirusitaに合わせたflat構成）
- Svelte 5（runes）: propsは `$props()`、状態は `$state`/`$derived`/`$effect`、
  親への通知はコールバックprops（`onXxx`）、イベントは `onclick` 等のネイティブ属性
- コミットメッセージ: `feat:` / `fix:` / `chore:` プレフィックス
- 言語: コード内コメントは日本語OK、識別子は英語

## 起動時の注意点（main.go）

`os.UserHomeDir()` の取得に失敗した場合、相対パスへのフォールバックはせず、
日本語のエラーメッセージを出力して `os.Exit(1)` でただちに起動を中断する
（履歴保存先が意図せず作業ディレクトリ配下になることを防ぐため）。
