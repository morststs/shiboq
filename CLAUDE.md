# CLAUDE.md — shiboq

## プロジェクト概要

jqコマンドを繰り返し実行して結果を確認できる、Wails v2 + Svelte 5 で構築した
デスクトップアプリ（アプリ名: shiboq。「絞る」＋jqの「q」）。左から保存・JSON/
スキーマ・クエリ/結果の5ペイン構成（保存ペインはドラッグで幅を調整できる。
後述「App.svelteの並行処理」の下、「保存ペインの幅（スプリッター）」参照）。
JSON/クエリの編集後500ms（デバウンス）でスキーマ再推論・使用キー抽出・
クエリ実行を自動的にまとめて行う。**保存は自動ではなくユーザーの明示的な操作**で、
保存項目は `~/.shiboq/saved/{UUID}.json` に保存される（件数の上限も自動間引きも無い。
ユーザーが意図して保存したものを黙って消さないため）。

以前は実行成功のたびに自動保存する「履歴」だったが、jqはキーが無くても`null`を返し
エラーにしないため`.a`→`.add`→`.address`のような入力途中がほぼ全て「成功」として
残ってしまい、一覧が意図しない項目で埋まっていた。そのため明示的な保存に変更した
（旧`~/.shiboq/history/`は起動時に`saved/`へ自動移行する。後述`migrateLegacyHistoryDir`）。

同じフロントエンドをブラウザで動かす**Web版**もあり、GitHub Pages
（<https://shiboq.e17.click/>）で公開している。jqエンジンはGoのコードを
WebAssemblyにビルドしたもの（後述「Web版（GitHub Pages）」参照）。

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
- **その他Go依存:** `github.com/google/uuid`（保存ファイル名）
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

# Wailsビルド（Linux。出力: build/bin/shiboq。WebKitGTK 4.1にリンクする。後述の注記）
./scripts/dev-run.sh wails build

# Wailsビルド（Windows。出力: build/bin/shiboq.exe）
# Dockerfileが同梱するmingw-w64でクロスコンパイルする。Windows実機は不要
./scripts/dev-run.sh wails build -platform windows/amd64

# フロントエンドのみビルド確認
./scripts/dev-run.sh sh -c "cd frontend && npm run build"

# Web版のビルド（出力: frontend/dist-web。wasmのビルドも含む）
./scripts/dev-run.sh sh -c "cd frontend && npm run build:web"

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
- libgtk-3-dev, libwebkit2gtk-4.0-dev/4.1-dev（Linuxビルド用）。実際に使うのは4.1で、
  `wails.json`の`"build:tags": "webkit2_41"`で指定している（Wailsはこのタグと
  CLIの`-tags`を合算する）。Wails v2の既定はWebKitGTK 4.0だが、Ubuntu 24.04以降は
  `libwebkit2gtk-4.0`を提供しておらず、既定のままだと
  `error while loading shared libraries: libwebkit2gtk-4.0.so.37`で起動できないため。
  4.1はUbuntu 22.04にもあるので、4.1に寄せて失う対象は無い。実行時に必要なのは
  `libwebkit2gtk-4.1-0`（Ubuntu 26.04の`2.52.6-0ubuntu0.26.04.1`で起動確認済み）。
  `go test`/`go vet`はこのタグを使わず4.0側でリンクするが、コンテナには両方の
  `-dev`があるので問題ない
- gcc-mingw-w64-x86-64, nsis（Windowsクロスコンパイル用。
  `wails build -platform windows/amd64` で `build/bin/shiboq.exe` を生成できる。
  生成物はWindows x86-64のGUIサブシステム実行ファイル。
  実行にはWindows側にWebView2ランタイムが必要（Win11は標準搭載、Win10は要インストール））

## プロジェクト構成

```
shiboq/
├── main.go                # Wails エントリポイント（//go:build !js。下記のapp.go等も同様）、App/JqService/SavedServiceのBind + 旧履歴ディレクトリの移行
├── app.go                 # App構造体（ライフサイクル + OpenJSONFile）
├── jq_service.go          # gojqによるクエリ実行・エラー整形・使用キー抽出(AST解析)
├── schema.go              # JSON値からキーツリー（型付き）とJSON Schemaを推論
├── saved_service.go       # 保存項目のCRUD（~/.shiboq/saved/*.json）+ 旧history/からの移行
├── wasm_main.go           # Web版のエントリポイント（//go:build js && wasm）。shiboqCallをJSへ公開
├── web_bridge.go          # Web版の呼び出しをJqServiceへ振り分ける（syscall/js非依存でテスト可能）
├── web_bridge_test.go     # web_bridge.goのテスト
├── jq_service_test.go     # jq_service.goのテスト（テストケースごとに個別のfunc、テーブル駆動ではない）
├── schema_test.go         # schema.goのテスト（同上）
├── saved_service_test.go  # saved_service.goのテスト（同上）
├── app_test.go            # app.goの一部（readJSONFileWithLimit）のテスト（同上）
│                          #   ※ main.goにはテストが無い。app.goのOpenJSONFile自体は
│                          #   ネイティブダイアログに依存するためテスト不可、サイズ
│                          #   チェックのロジックだけreadJSONFileWithLimitへ切り出してテストする
├── Dockerfile              # 開発用コンテナイメージ定義（golang:1.24-bookworm）
├── scripts/dev-run.sh      # コンテナ内でコマンドを実行するヘルパー
├── scripts/build-wasm.sh   # Web版のwasm + wasm_exec.jsをfrontend/src/backend/generated/へ出力
├── .github/workflows/pages.yml # mainへのpushでWeb版をGitHub Pagesへ公開
├── scripts/build-msix.ps1  # Microsoft Store提出用MSIXのビルド（Windows専用。後述「MSIX（Microsoft Store提出用）」）
├── build/msix/AppxManifest.xml # MSIXのマニフェスト（Partner Centerの製品IDを書き込む）
├── PRIVACY.md              # プライバシーポリシー（Store提出時にURLを入力する）
├── docs/store-submission.md # Partner Centerに入力した内容の控え（説明文・審査メモ・画像の作り方等）
├── .devcache/               # go/npmキャッシュのバインドマウント先（.gitignore済み、初回実行時に自動生成）
├── docs/superpowers/
│   ├── specs/2026-08-02-jqrepl-design.md    # 設計仕様書（アプリソースではない。
│   │         # アプリ名がjqreplだった当時の日付入り記録として、ファイル名・
│   │         # 本文とも意図的に当時のまま変更していない。shiboqへの改名は
│   │         # このファイルより後の出来事）
│   └── plans/2026-08-02-jqrepl-implementation.md # 実装計画書（アプリソースではない、同上）
├── frontend/
│   ├── svelte.config.js    # vitePreprocess({ script: true })（flowbite-svelte対応、削除不可）
│   ├── vite.config.js      # Vite + svelte + @tailwindcss/vite。`--mode web`でWeb版（'$backend'の切替）
│   ├── public/favicon.png  # build/appicon.pngを64pxに縮小したもの
│   ├── src/
│   │   ├── main.js             # Svelte マウント
│   │   ├── backend/            # バックエンド呼び出しの実装（App.svelteは'$backend'からimport）
│   │   │   ├── wails.js        #   デスクトップ版: wailsjsのバインディングを再exportするだけ
│   │   │   ├── web.js          #   Web版: wasm Worker・ファイル選択・navigator.clipboard
│   │   │   ├── webSaved.js     #   Web版の保存項目（IndexedDB）
│   │   │   ├── jq.worker.js    #   Web版: wasmを動かすWorker
│   │   │   └── generated/      #   scripts/build-wasm.shの出力（.gitignore済み）
│   │   ├── style.css           # グローバルスタイル（Tailwind + Flowbite）
│   │   ├── monaco.js           # Monaco Editorの構成 + 'jq'言語ID登録（後述「Monaco構成の注意点」）
│   │   ├── jqCompletions.js    # jqクエリペイン用のMonaco補完プロバイダ
│   │   ├── App.svelte          # ルート（状態管理 + 5ペインのレイアウト + 自動実行パイプライン。後述「App.svelteの並行処理」）
│   │   ├── SavedPane.svelte    # 左ペイン：保存項目一覧（＋保存ボタン・各行の削除ボタン）
│   │   ├── JsonPane.svelte     # 中央上ペイン：元のJSON（編集可、Monaco。「整形」ボタン）
│   │   ├── SchemaPane.svelte   # 中央下ペイン：キーツリー（使用中キーをハイライト。JSON Schemaの「コピー」ボタン）
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
| `SchemaJSON(jsonText)` | JSONから推論したスキーマを、整形済み（2スペース）のJSON Schema文字列で返す（`schema.go`のInferJSONSchemaを呼ぶだけ）。スキーマペインの「コピー」用。詳細は下記「JSON Schemaの推論ルール」参照 |

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

#### JSON Schemaの推論ルール（InferJSONSchema）

スキーマペイン（`InferSchema`）と表示が食い違わないよう、推論ルールを揃えている。

- 出力するのは`type` / `properties` / `items`のみ。`required`・`$schema`・
  `additionalProperties`等は推論しない（1件のサンプルからは判断できないため）
- 配列は先頭要素を`items`にする。空配列は`{"type": "array"}`のみ（`items`無し）
- 数値は全て`number`（`integer`と区別しない）
- `type`を先頭に出し、`properties`のキーはアルファベット順
- キー名をHTMLエスケープしない（`<`を`\u003c`にしない）
- `SchemaNode`のツリーからではなく**元のJSONから直接**生成する。`SchemaNode`は
  「ルートが配列かどうか」を保持しておらず（先頭要素の形状をルート扱いにする）、
  そこから作るとルート配列が`object`になってしまうため

### SavedService（saved_service.go）

`SavedItem` は `{id, name, json, query, result, createdAt}`。`name` は任意入力で、
空の場合はフロントエンド側が代わりに `query` を一覧の見出しに使う。

| メソッド | 説明 |
|---------|------|
| `SaveItem(item)` | 保存項目を書き込む。IDが空ならUUIDを発行する（空でない場合はUUID形式を検証）。**内容が既存と同一でも必ず新規保存する**（明示的な操作を黙って無視しないため）。書き込みは同ディレクトリへの一時ファイル作成＋`os.Rename`によるアトミック書き込み |
| `ListSaved()` | 保存項目を新しい順（CreatedAt降順、同時刻は ID降順でタイブレーク＝`sort.SliceStable`）で返す。IDが空・不正な形式（UUID以外）の項目は、Svelteの`{#each ... (item.id)}`キー重複による例外を防ぐため除外する |
| `DeleteSaved(id)` | 保存項目を削除する |

`migrateLegacyHistoryDir(legacyDir, savedDir)`（Wailsにはバインドしない内部関数）は、
旧「履歴」時代の`~/.shiboq/history`を`saved`へ`os.Rename`で移行する。
**`saved`が未作成で、かつ`history`が存在する場合のみ**実行する。`main.go`で
`NewSavedService`より**前に**呼ぶ必要がある（先に`NewSavedService`が`saved`を
作ってしまうと移行済みと判定される）。失敗しても警告を出して起動は継続する。

かつて持っていた「直近と同一内容ならスキップ」「200件を超えたら古いものを間引く」は
どちらも削除した。自動保存が前提の仕組みであり、明示的な保存では
「押したのに増えない」「意図して残したものが消える」という不具合になるため。

保存ディレクトリは`0700`、ファイルは`0600`で作成する（貼り付けたJSONに
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

`evaluate()`（スキーマ推論・使用キー抽出・クエリ実行をまとめて行う自動実行パイプライン）は、
JSON/クエリの変更のたびに500ms後に呼ばれるが、連続編集時には複数回の`evaluate()`が
同時に飛行中になり得る。古い呼び出しが新しい呼び出しの結果を上書きしないよう、
以下のガードを設けている。将来編集する際もこれらのガードを外さないこと。

- **`evaluateToken`:** `evaluate()`開始時にインクリメントして捕捉するモノトニックなトークン。
  各`await`の直後（catchブロックも含む）で `if (token !== evaluateToken) return;` を行い、
  トークンが変わっていれば（＝自分より新しい呼び出しが既に始まっていれば）状態を一切書き込まずに中断する。
- **`savedRequestToken`:** `refreshSaved()`自身の呼び出しごとに採番するトークン。
  `onMount`と保存/削除の完了後から呼ばれるため、古い呼び出しの`ListSaved()`応答が
  新しい呼び出しの応答より後に返ってきても、一覧を古い内容で上書きしないように自己ガードする。
- **デバウンスの`$effect`:** `setTimeout(evaluate, 500)`をセットし、`clearTimeout`する
  ティアダウン関数を返す。エフェクトの再実行・アンマウント時に前回のタイマーを確実に破棄する。
`evaluate()`は保存を行わない（保存はユーザーの明示操作のみ）。かつて自動保存だった頃は
「復元直後の内容を再保存しない」ための`restoredSignature`という仕組みを持っていたが、
自動保存の廃止に伴い不要になったため削除した。

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

### 保存ペインの幅（スプリッター）

`savedWidth`（`$state`、初期値260px）をドラッグでリサイズできる。
参考プロジェクト [sirusita](https://github.com/morststs/sirusita) の`App.svelte`の
`startDrag`/`onDrag`/`stopDrag`パターンを踏襲しており、`mousedown`で
`window`に`mousemove`/`mouseup`リスナーを登録し、`mouseup`（`stopDrag`相当）で
リスナーを解除して幅を`localStorage`（キー: `shiboq.savedWidth`）に保存する。
`onDestroy`でも`stopSavedDrag()`を呼び、ドラッグ中にアンマウントされても
`window`リスナーが残留しないようにする。幅は`SAVED_MIN_WIDTH`(160)〜
`SAVED_MAX_WIDTH`(500)にクランプする。中央/右列の上下比率（JSON/スキーマ、
クエリ/結果）は引き続き固定（`flex: 1`で等分、リサイズ不可）。

### JSONの整形とスキーマのコピー

- **整形**（JSONペインのヘッダー「整形」ボタン）: Monaco標準の
  `editor.action.formatDocument`（JSON言語サービスのフォーマッタ）を呼ぶ。
  `JSON.parse`→`JSON.stringify`にはしない。大きな整数の精度落ちや数値表記の変化
  （`1.0`→`1`）が起き、Ctrl+Zで戻せなくなるため。Monacoは読み込んだ内容から
  インデント幅を自動検出するので、実行前にモデルを`tabSize: 2`に固定している。
  整形は通常の編集として扱われ、500ms後に自動再実行される。
- **スキーマのコピー**（スキーマペインのヘッダー「コピー」ボタン）:
  `SchemaJSON(jsonText)`の結果をWailsランタイムの`ClipboardSetText`で書き込む
  （`navigator.clipboard`はWebKitGTK等のWebViewで許可されないことがあるため使わない）。
  デバウンス後の`schemaNodes`ではなく**押した時点の`jsonText`**から作る。
  結果はトーストで通知する（「スキーマをコピーしました」／「コピーできませんでした」）。
  JSONが不正な間（`schemaError`あり）はボタンを無効化する。

### 保存・削除の失敗通知

`SaveItem`/`DeleteSaved`が失敗した場合、気づけないとユーザーは保存/削除できたと
誤解し続ける。
そのため`showToast('保存できませんでした')` /
`showToast('削除できませんでした')`で3秒間のトースト通知を画面右下に出す
（モーダルにはしない）。

### 保存・削除のUI

保存ペインのヘッダー右端の「＋ 保存」ボタン、または`Ctrl+S`（`svelte:window`の
`onkeydown`。WebViewの既定動作は`preventDefault`で抑止）で名前入力ダイアログを開く。
名前は空でもよく、その場合は一覧にクエリが表示される（`SavedPane.displayLabel()`）。

**エラー中は保存できない**（`canSave = !resultStale && errorMessage === '' && schemaError === ''`）。
表示中の結果は前回成功時のもので、いま画面にあるJSON/クエリと対応しないため、
保存すると内容が食い違うから。ボタンは無効化され、`title`で理由を示す。

削除は各行のホバーで現れる×から確認ダイアログを経て実行する。削除したのが
選択中の項目なら`selectedSavedId`を`null`に戻す（存在しないIDを指したままにしない）。
両ダイアログとも`Esc`で閉じられる。

## Web版（GitHub Pages）

デスクトップ版と同じ`App.svelte`をブラウザで動かす。`vite.config.js`の`'$backend'`
エイリアスが、通常は`src/backend/wails.js`、`--mode web`では`src/backend/web.js`を指す。
両者は同じ名前・同じ形の関数（`RunQuery`等。エラーはrejectする）を提供するので、
App.svelte側に分岐は無い。Web版の出力先は`frontend/dist-web`（`go:embed`される
`frontend/dist`を上書きしないため）、`base: './'`（相対パスにしておけば、独自ドメインのルートでも`morststs.github.io/shiboq/`のようなサブパスでも動く）。

- **jqエンジン:** ルートパッケージを`GOOS=js GOARCH=wasm`でビルドする（`main.go`・
  `app.go`・`saved_service.go`とそのテストは`//go:build !js`で除外、代わりに
  `wasm_main.go`がmainになる）。`jq_service.go`・`schema.go`はそのまま共用。
  `wasm_exec.js`はビルドしたGoの`$(go env GOROOT)/lib/wasm/`から必ずコピーする
  （バージョンが合わないと動かない）。gzip後約1.6MB。
- **Workerで動かし、メインスレッドで打ち切る:** wasmにはgoroutineのプリエンプションが
  無く、`def f: f; f`のようにCPUを占有し続けるクエリでは**Go側の`Timeout`が発火しない**
  （Nodeで実測）。そのため`web.js`がWorkerごと`terminate()`して打ち切り、
  Go側と同じタイムアウト文言の`RunResult`を返す。さらに、前の`RunQuery`が未完了の
  まま次の`RunQuery`が来たら、古い方をWorkerごと打ち切る（古い結果は`evaluateToken`で
  捨てられるので待つ意味が無い）。巻き添えを避けるため、クエリ実行用と
  スキーマ推論等の解析用でWorkerを分けている。コンパイル済みの`WebAssembly.Module`
  はメインスレッドで1回だけ作り、Workerを作り直すたびに`postMessage`で渡す。
- **保存項目:** IndexedDB（DB `shiboq`、ストア`saved`、keyPath `id`）。localStorageは
  容量が約5MBしかなく、JSONを丸ごと保存すると溢れるため使わない。`saved_service.go`と
  同じ規則（UUID発行・同一内容でも新規保存・新しい順・UUID以外は除外）。
- **ファイルを開く:** `<input type="file">`。20MB上限と文言はデスクトップ版と同じ。
- **クリップボード:** `navigator.clipboard.writeText`（Pagesはhttpsなので使える）。
- **公開:** `.github/workflows/pages.yml`がmainへのpushごとにビルドして公開する。
  リポジトリのPages設定はSourceが「GitHub Actions」。CIでは`go test`を実行しない
  （Wailsのcgo依存と`frontend/dist`が必要なため）。`GOOS=js go vet`のみ。
  アクションは2026-10-06に全ワークフロー（pages/release/msix）でNode 24版へ更新した
  （checkout@v7・setup-go@v7・setup-node@v7・configure-pages@v6・upload-pages-artifact@v5・
  deploy-pages@v5・upload-artifact@v7・softprops/action-gh-release@v3）。
  upload-pages-artifact@v4以降はドットファイルを含めない（dist-webには無いので影響なし）。
  更新後、pages.yml（push）とmsix.yml（手動実行、`version=1.0.0`。Artifactを作るだけで
  Storeには送らない）が成功し、Node 20の警告は消えた。**release.ymlは更新後まだ一度も
  実行していない**（Releaseを作成・変更するため確認目的では動かさなかった）。次に
  `v*`タグをpushしたときに、`action-gh-release@v3`で添付まで通るか確認すること。
  `ubuntu-latest`は2026-10-19からUbuntu 26へ移行するとの通知が出ている。pages.ymlは
  GoとNodeしか使わないため影響は無い見込み（release/msixは`windows-latest`）。deployジョブがGitHub側で`queued`のまま
  7時間以上進まなくなったことがある（2026-10-05。環境の保護ルールや承認待ちではなかった）。
  `concurrency: pages`のため後続の実行も`pending`で止まるので、詰まった実行を
  `gh run cancel <run-id>`で取り消すと後続が進む（内容は後続の方が新しいので失うものは無い）。
- **独自ドメイン:** `shiboq.e17.click`（2026-10-05設定）。Route53のホストゾーン`e17.click`
  （`Z003907937M42XWPZX222`）に`shiboq` → `morststs.github.io`のCNAME（TTL 300）を置き、
  Pages設定のカスタムドメインに指定している（Actionsでの公開なので`CNAME`ファイルは
  不要。Pages設定側が正。`gh api -X PUT repos/morststs/shiboq/pages -f cname=...`）。
  証明書は発行済みでHTTPS強制。`morststs.github.io/shiboq/`は独自ドメインへ301で転送される。
  リポジトリのWebsite欄もこのURL。同ゾーンの`app`・`numvil`等も同じ構成。
  Route53はaws-mcp経由で操作する。`TOKEN_EXPIRED`が出たら、認証し直すだけでは足りず
  `/mcp`でaws-mcpを再接続する必要があった。
- **動作確認:** `shiboq-dev`イメージにrootで`chromium`を入れ、`playwright-core`
  （`executablePath: '/usr/bin/chromium'`）で`vite preview --mode web`を操作して確認した
  （結果表示・スキーマ・タイムアウト打ち切り・打ち切り後の復帰・保存とリロード後の復元）。
  同じスクリプトで公開中の`morststs.github.io/shiboq/`も確認済み（独自ドメイン化の前）。
  スクリプトは`.devcache/e2e/`（.gitignore済み、リポジトリには無い）。Monacoの
  `innerText`には行番号が混ざる（`"1\n\"taro\""`）ので、比較時は注意。
  **開発環境のネットワークは許可されたホスト名以外への接続を遮断する**ため、
  `shiboq.e17.click`はここから名前解決も接続もできない（GitHubのIPを指定しても
  SNIで切られる）。独自ドメインでの表示確認はユーザーのブラウザで行う
  （2026-10-05時点で未確認）。

## MSIX（Microsoft Store提出用）

署名の無い`shiboq.exe`はWindows 11のスマート アプリ コントロールにブロックされる
（アプリ単位の例外登録はできない）。Store経由ならMicrosoftが署名するため、
有料のコード署名証明書なしで回避できる。そのための仕組み。

- **`build/msix/AppxManifest.xml`**: `runFullTrust`のデスクトップアプリ構成。
  `Identity`の`Name`・`Publisher`と`PublisherDisplayName`はPartner Centerの
  「製品 ID の表示」の値（設定済み。秘密情報ではない）で、完全に一致しないと
  Storeに受け付けられない。`REPLACE_WITH_`で始まる仮の値だとスクリプトがエラーで止まる。
  `MinVersion`はWindows 11（`10.0.22000.0`）。WebView2が標準搭載の環境に絞るため。
- **`scripts/build-msix.ps1 -Version X.Y.Z`**: `build/bin/shiboq.exe`・マニフェスト・
  ロゴ・ライセンス類を`build/msix/staging`に集め、Windows SDKの`MakeAppx.exe`で
  `build/bin/shiboq.msix`を作る。**Windows専用**で、開発用コンテナ（Linux）では
  実行できない。ロゴ（44px・150px・50px）は`build/appicon.png`からその場で
  縮小生成する（画像をリポジトリに重複して持たない）。**署名はしない**（Storeが
  審査後に署名する）。
- **バージョン**: `X.Y.Z` → `X.Y.Z.0`に変換する。Storeの規則で4桁目は0固定、
  **先頭は0にできない**。exeのタグは`v0.x.y`なので、MSIXのバージョンはタグとは
  別に`1.0.0`以上を指定する。
- **スクリプトはASCIIのみ**で書く。Windows PowerShell 5.1はBOM無しのスクリプトを
  ANSIとして読むため、日本語を入れると構文解析が壊れ得る。
- **`.github/workflows/msix.yml`**: 手動実行（`workflow_dispatch`）専用。成果物は
  Releaseではなく実行のArtifact（`shiboq-msix`）に出す。`release.yml`とは独立。
- ロゴは修飾子無しのファイル名のみ（`targetsize-*`・`scale-*`の派生は持たない）。
  派生を使うには`resources.pri`（MakePri）が必要になるため、今は入れていない。

## Microsoft Storeへの提出（Partner Center）

### 状況（2026-10-03時点）

- 製品「shiboq」（Store ID `9MV10MVFX78Z`）は**下書き段階で、まだ「送信して認定を受ける」を押していない**。
- 完了済み: パッケージ（`shiboq.msix` v1.0.0.0をアップロード、Validated）、プロパティ、
  Store登録情報（日本語のみ）、申請オプション（runFullTrustの理由）、価格（無料、保存済み）。
- **残り**: 年齢区分（IARCがアンケートを改訂したため回答し直し。MSIX製品を作り直した際に
  回答が引き継がれず「アプリの種類」から未選択だった）。「追加のテスト情報」の説明欄が
  保存されたかも要確認（最後に見た時点では空だった）。全項目が「完了」になったら送信する。
- 入力した値・文章はすべて[`docs/store-submission.md`](./docs/store-submission.md)に控えてある。
  再提出や作り直しのときはそこから転記する。

### 提出で分かった注意点

- **製品の種類は必ず「MSIX または PWA アプリ」**。最初に誤って「EXE または MSI アプリ」で
  作ってしまった。EXE/MSI型はインストーラーのURLを登録する方式で、署名は自分で行う前提
  （Storeは署名しない）ため、スマート アプリ コントロール回避という目的を果たせない。
  種類は作成後に変更できないので、作り直して名前を移した。
- **更新を出すときはMSIXのバージョンを上げる**（`1.0.0`の次は`1.0.1`以上）。
  `.github/workflows/msix.yml`を`gh workflow run msix.yml -f version=X.Y.Z`で実行し、
  Artifact `shiboq-msix`をダウンロードしてアップロードする。Chromeはzip内の未署名
  実行ファイルを理由にダウンロードをブロックすることがあり、`Ctrl+J`から「保存」で通せる
  （`gh run download <run-id> -n shiboq-msix`でも取得できる）。
- Partner Centerの入力欄の制約: runFullTrustの理由は**500文字まで**。キーワードは
  単独の`jq`・`JSON`を受け付けなかった（`jq クエリ`等の組み合わせで代替）。
- 個人アカウントでは電話番号・住所は公開されないが、プロパティの「Support info」に
  入れたものはStoreページに**公開される**ので、電話番号・住所は空欄にしている。
- Partner Centerの「問題が発生しました…関連付け ID」は原因を示さない汎用エラー。
  時間を置く・再読み込み・別ブラウザーで再試行する。

## 実機での動作確認（Windows / WSL）

開発用コンテナではGUIを確認できないので、ユーザーの実機で確認する。注意点:

- **Windows 11のスマート アプリ コントロールが未署名の`shiboq.exe`をブロックする**
  （ユーザーのPCで発生。以前は動いていたので、評価モードからオンに切り替わったと推測）。
  アプリ単位の例外は無く、自己署名も通らない。解決策はStore経由の配布（Microsoftが署名）。
- **Windowsサンドボックスでの確認手順**（スマート アプリ コントロールの影響を受けない）:
  サンドボックスにはWebView2が無く、Wailsの「Missing Requirements」が出る。自動インストールは
  失敗し、インストーラー画面も文字化けした（日本語フォントはサンドボックスに入っているので
  フォントが原因ではない。原因不明）。次の手順で動いた:
  1. Evergreen Standalone Installer（x64）を管理者PowerShellで`/silent /install`付きで実行
  2. Wailsの判定は`HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\ClientState\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
     の`EBWebView`（インストール先フォルダー）と、その下の`EBWebView\x64\EmbeddedBrowserWebView.dll`の
     実在で行う（go-webview2の`find_dll_installed.go`）。`Clients\{…}`の`pv`ではない。サンドボックスでは
     この登録が実在するフォルダーとずれていたので、`EBWebView`を
     `C:\Program Files (x86)\Microsoft\EdgeWebView\Application\<実在するバージョン>`に書き換えて起動した
     （サンドボックス内だけの応急処置。通常のWindowsでは行わない）
- **WSL（Ubuntu 26.04）でLinux版を動かす場合**: `libwebkit2gtk-4.1-0`に加えて`fonts-noto-cjk`が
  必要（無いと日本語が□になる。フォントは同梱しない方針）。WSLgでスクリーンショットを
  撮るには`GDK_BACKEND=x11 ./shiboq`で起動してImageMagickの`import`を使う。ただし
  `GDK_BACKEND=x11`だとWindowsからのクリップボード貼り付けが不安定だった
  （JSONは「ファイルを開く」で読み込めば回避できる）。
- コンテナ内での起動確認は、`ubuntu:26.04`イメージに`libwebkit2gtk-4.1-0 xvfb`を入れて
  `Xvfb :99 & DISPLAY=:99 ./build/bin/shiboq`で行える（`imagemagick`の`import -window root`で
  画面を撮って描画まで確認できる。日本語を見るなら`fonts-noto-cjk`も入れる）。

## 既知の制約（スコープ外）

- Web版の保存項目はブラウザごとのIndexedDBで、デスクトップ版の`~/.shiboq/saved`とは共有されない
- 複数JSONドキュメントのタブ管理は無い
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
- リサイズ可能なのは保存ペインの幅のみ（スプリッター）。中央/右列の上下比率
  （JSON/スキーマ、クエリ/結果）は`flex: 1`で等分固定、リサイズ不可
- 保存ペインのスプリッターはマウスドラッグのみで、キーボード操作の代替手段は無い
- 保存項目のリネーム・並べ替え・検索・タグ付け・エクスポートは無い
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
（保存先が意図せず作業ディレクトリ配下になることを防ぐため）。
