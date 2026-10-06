# shiboq

jq クエリを繰り返し試しながら結果を確認するための、5ペイン構成のデスクトップアプリです。
ブラウザで動く Web 版もあります。

**Web 版: <https://shiboq.e17.click/>**（インストール不要。JSON はブラウザ内で処理され、どこにも送信されません）

JSON を貼り付けて jq クエリを書くと、入力が止まった 500ms 後に自動で実行され、
結果とスキーマが並べて表示されます。試行錯誤しながら目的のクエリに
辿り着くための道具です。

名前は「絞る」と jq の q を掛けた造語です。

## 主な機能

- **5ペイン構成** — 左に保存した項目、中央上に元の JSON、中央下にスキーマ、
  右上に jq クエリ、右下に実行結果。
- **自動実行** — 入力が一定時間止まると自動でパース・実行します。ボタンを押す必要はありません。
- **エラー時は前回の結果を保持** — クエリが構文エラーや実行エラーになっても結果ペインは
  消えず、直前の成功結果が残ります（陳腐化していることはバッジで示されます）。
- **スキーマ表示と使用キーのハイライト** — JSON からキー名と型のツリーを推論し、
  現在のクエリが参照しているキーを色付きで示します。参照キーの抽出は
  文字列補間 `"\(.name)"` や分割代入 `. as {a: $x}` にも対応しています。
- **キー名の補完** — クエリペインで、現在の JSON に含まれるキー名と
  jq の組み込み関数を候補表示します。
- **生出力モード（`-r` 相当）** — 結果ペインのチェックボックスで切り替えます。
  オンにすると、結果が文字列のときだけ引用符とエスケープを外して表示するので、
  `@csv` や `@tsv` の結果がそのまま読めます。設定は次回起動時も保たれます。
- **保存** — 残しておきたい状態を「＋ 保存」ボタン（または `Ctrl+S`）で保存します。
  名前を付けられ、省略するとクエリがそのまま見出しになります。クリックすると
  JSON・クエリ・結果をまとめて復元でき、不要になったものは削除できます。
  自動では保存されないので、一覧に並ぶのは自分で選んだものだけです。
- **学習用サンプル** — 左ペインの「サンプル」タブに、値の取り出し・配列・オブジェクト・
  絞り込み・集計・変数と reduce・文字列と正規表現・出力形式・更新と探索の 10 章、
  43 個の例題があります。選ぶと JSON とクエリに読み込まれ、解説と「やってみよう」を
  見ながら自分で書き換えて試せます。
- **ヘルプ** — クエリペインの「？ ヘルプ」から、構文の早見表・よく使う関数・
  困ったときの Q&A・使えない機能を確認できます。対応している jq の仕様のバージョン
  （jq 1.7 準拠。jq 1.8 の一部の関数も使用可）と、内蔵エンジン（gojq）のバージョンも表示します。
- **jq バイナリ不要** — [gojq](https://github.com/itchyny/gojq) を組み込んでいるため、
  別途 jq をインストールする必要はありません。

## Web 版

<https://shiboq.e17.click/> で使えます。デスクトップ版と同じ画面・同じ jq エンジン
（gojq を WebAssembly にビルドしたもの）で動き、JSON やクエリをサーバーへ送ることはありません。
デスクトップ版との違いは次のとおりです。

- 保存した項目はブラウザの IndexedDB に保存されます（ブラウザ・端末ごとに別。
  サイトデータを消去すると消えます）。デスクトップ版の保存項目とは共有されません。
- 初回表示時に jq エンジン（約 1.6MB、gzip 圧縮時）をダウンロードします。

`main` ブランチに push すると GitHub Actions（[`pages.yml`](./.github/workflows/pages.yml)）が
ビルドして GitHub Pages に公開します。

## データの保存場所（デスクトップ版）

保存した項目は次のディレクトリに 1 件 1 ファイルで保存されます。

```
~/.shiboq/saved/{UUID}.json
```

自動保存はしないため件数の上限もなく、削除するのは自分で削除したときだけです。

> 以前のバージョンで自動保存された `~/.shiboq/history/` がある場合は、
> 初回起動時に `~/.shiboq/saved/` へ自動で移行します。

## 技術スタック

- **バックエンド:** Go 1.24 + [Wails](https://wails.io) v2.12
- **jq エンジン:** [itchyny/gojq](https://github.com/itchyny/gojq)
- **フロントエンド:** Svelte 5（runes）+ Vite 7
- **エディタ:** Monaco Editor
- **UI:** Flowbite Svelte + TailwindCSS 4

## ビルド

ホストに Go / Node / Wails をインストールせず、すべて Docker コンテナ内で完結します。

```bash
# 開発用イメージのビルド（初回のみ）
docker build -t shiboq-dev .

# Linux 向けビルド（出力: build/bin/shiboq）
./scripts/dev-run.sh wails build

# Windows 向けビルド（出力: build/bin/shiboq.exe）
./scripts/dev-run.sh wails build -platform windows/amd64

# テスト
./scripts/dev-run.sh go test ./...

# Web 版のビルド（出力: frontend/dist-web）
./scripts/dev-run.sh sh -c "cd frontend && npm run build:web"
```

> clone 直後は `go test` が `pattern all:frontend/dist: no matching files found` で
> 失敗します。`//go:embed` がビルド生成物を要求するためで、先に一度
> `wails build` を実行すれば解消します。詳細は [`CLAUDE.md`](./CLAUDE.md) を参照してください。

Windows で実行するには WebView2 ランタイムが必要です（Windows 11 は標準搭載）。

Linux 版は WebKitGTK 4.1 を使います（`wails.json` の `build:tags` で `webkit2_41`
を指定）。Ubuntu 22.04 以降なら次のパッケージで動きます（Ubuntu 26.04 で起動を確認済み）。

```bash
sudo apt install libwebkit2gtk-4.1-0 fonts-noto-cjk
```

## ダウンロード（Windows）

**[Microsoft Store](https://apps.microsoft.com/detail/9MV10MVFX78Z)** からインストールしてください
（Microsoft が署名しているため、Windows 11 のスマート アプリ コントロールにブロックされません）。

署名の無い exe 単体も [Releases](https://github.com/morststs/shiboq/releases/latest) に置いています
（スマート アプリ コントロールがオンの環境では起動できないことがあります）。
`v*` タグ（例: `v0.1.0`）を push すると GitHub Actions が `windows-latest` 上で
exe をビルドし、`shiboq-windows-amd64.zip`（exe 本体・LICENSE・
THIRD_PARTY_LICENSES.md・README.md を同梱）を Release に自動添付します。
Store 版の更新はこれとは別に手動で行います（下記）。

## Microsoft Store 向けパッケージ（MSIX）

署名の無い exe は Windows 11 のスマート アプリ コントロールにブロックされることがあります。
Microsoft Store 経由で配布すると Store が署名するため、証明書を用意せずに回避できます。

1. [`build/msix/AppxManifest.xml`](./build/msix/AppxManifest.xml) の
   `Identity`（`Name`・`Publisher`）と `PublisherDisplayName` が、Partner Center の
   「製品 ID の表示」の値と一致していることを確認する（設定済み。秘密情報ではありません）。
2. GitHub の Actions タブで「Build MSIX for Microsoft Store」を手動実行し、
   バージョン（例: `1.0.0`）を入力する。Store の規則で先頭を `0` にはできないため、
   exe のタグ（`v0.x.y`）とは別の番号になります。
3. 実行結果の Artifact `shiboq-msix` から `shiboq.msix` をダウンロードし、
   Partner Center の提出画面にアップロードする。

MSIX は未署名のまま作ります（Store が審査後に署名します）。そのため、ダウンロードした
`shiboq.msix` をそのままダブルクリックしてもインストールできません。

Windows SDK を入れた Windows 上では、手元でも作れます。

```powershell
wails build -platform windows/amd64
pwsh scripts/build-msix.ps1 -Version 1.0.0   # 出力: build/bin/shiboq.msix
```

プライバシーポリシーは [`PRIVACY.md`](./PRIVACY.md) を参照してください。

## ライセンス

MIT License（[`LICENSE`](./LICENSE)）。

同梱するサードパーティソフトウェアのライセンスは
[`THIRD_PARTY_LICENSES.md`](./THIRD_PARTY_LICENSES.md) を参照してください。
