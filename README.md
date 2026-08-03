# shiboq

jq クエリを繰り返し試しながら結果を確認するための、5ペイン構成のデスクトップアプリです。

JSON を貼り付けて jq クエリを書くと、入力が止まった 500ms 後に自動で実行され、
結果・スキーマ・実行履歴が並べて表示されます。試行錯誤しながら目的のクエリに
辿り着くための道具です。

名前は「絞る」と jq の q を掛けた造語です。

## 主な機能

- **5ペイン構成** — 左に実行履歴、中央上に元の JSON、中央下にスキーマ、
  右上に jq クエリ、右下に実行結果。
- **自動実行** — 入力が一定時間止まると自動でパース・実行します。ボタンを押す必要はありません。
- **エラー時は前回の結果を保持** — クエリが構文エラーや実行エラーになっても結果ペインは
  消えず、直前の成功結果が残ります（陳腐化していることはバッジで示されます）。
- **スキーマ表示と使用キーのハイライト** — JSON からキー名と型のツリーを推論し、
  現在のクエリが参照しているキーを色付きで示します。参照キーの抽出は
  文字列補間 `"\(.name)"` や分割代入 `. as {a: $x}` にも対応しています。
- **キー名の補完** — クエリペインで、現在の JSON に含まれるキー名と
  jq の組み込み関数を候補表示します。
- **実行履歴** — 実行が成功するたびに自動保存され、クリックすると
  JSON・クエリ・結果をまとめて復元します。
- **jq バイナリ不要** — [gojq](https://github.com/itchyny/gojq) を組み込んでいるため、
  別途 jq をインストールする必要はありません。

## データの保存場所

実行履歴は次のディレクトリに 1 件 1 ファイルで保存されます。

```
~/.shiboq/history/{UUID}.json
```

件数が 200 を超えると古いものから自動的に削除されます。

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
```

> clone 直後は `go test` が `pattern all:frontend/dist: no matching files found` で
> 失敗します。`//go:embed` がビルド生成物を要求するためで、先に一度
> `wails build` を実行すれば解消します。詳細は [`CLAUDE.md`](./CLAUDE.md) を参照してください。

Windows で実行するには WebView2 ランタイムが必要です（Windows 11 は標準搭載）。

## ライセンス

MIT License（[`LICENSE`](./LICENSE)）。

同梱するサードパーティソフトウェアのライセンスは
[`THIRD_PARTY_LICENSES.md`](./THIRD_PARTY_LICENSES.md) を参照してください。
