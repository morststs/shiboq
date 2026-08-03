# サードパーティライセンス

shiboq 本体は MIT ライセンス（[`LICENSE`](./LICENSE)）で配布されます。
ビルド成果物（`build/bin/shiboq`, `build/bin/shiboq.exe`）には以下の
サードパーティソフトウェアが含まれます。いずれも許容的ライセンスであり、
コピーレフト（GPL / AGPL / LGPL / SSPL）系のコードは含まれません。

各ライセンスの全文は、それぞれのリポジトリまたは配布物に同梱の
LICENSE ファイルを参照してください。

## Go（バックエンド）

| ライブラリ | バージョン | ライセンス |
|---|---|---|
| [github.com/wailsapp/wails/v2](https://github.com/wailsapp/wails) | v2.12.0 | MIT |
| [github.com/itchyny/gojq](https://github.com/itchyny/gojq) | v0.12.19 | MIT |
| [github.com/itchyny/timefmt-go](https://github.com/itchyny/timefmt-go) | v0.1.8 | MIT |
| [github.com/leaanthony/go-ansi-parser](https://github.com/leaanthony/go-ansi-parser) | v1.6.1 | MIT |
| [github.com/leaanthony/slicer](https://github.com/leaanthony/slicer) | v1.6.0 | MIT |
| [github.com/leaanthony/u](https://github.com/leaanthony/u) | v1.1.1 | MIT |
| [github.com/rivo/uniseg](https://github.com/rivo/uniseg) | v0.4.7 | MIT |
| [github.com/google/uuid](https://github.com/google/uuid) | v1.6.0 | BSD-3-Clause |
| [github.com/pkg/errors](https://github.com/pkg/errors) | v0.9.1 | BSD-2-Clause |

Windows 向けビルドでは、上記に加えて Wails が依存する
`github.com/go-ole/go-ole`（MIT）、`git.sr.ht/~jackmordaunt/go-toast/v2`（MIT）等が
リンクされます。

## JavaScript / CSS（フロントエンド）

| ライブラリ | バージョン | ライセンス |
|---|---|---|
| [monaco-editor](https://github.com/microsoft/monaco-editor) | 0.55.1 | MIT |
| [svelte](https://github.com/sveltejs/svelte) | 5.56.8 | MIT |
| [flowbite-svelte](https://github.com/themesberg/flowbite-svelte) | 1.33.1 | MIT |
| [flowbite](https://github.com/themesberg/flowbite) | 4.0.2 | MIT |
| [tailwindcss](https://github.com/tailwindlabs/tailwindcss) | 4.3.3 | MIT |
| [dompurify](https://github.com/cure53/DOMPurify) | 3.2.7 | MPL-2.0 **または** Apache-2.0（下記参照） |

### DOMPurify のライセンス選択

DOMPurify は MPL-2.0 と Apache-2.0 のデュアルライセンスで提供されています。
本プロジェクトでは **Apache-2.0 を選択** します。DOMPurify のソースは改変せず、
依存関係として取り込んだものをそのまま同梱しています。

DOMPurify は flowbite-svelte 経由で間接的に取り込まれ、ビルド成果物に含まれます。

## ビルド時のみ使用するツール（成果物には含まれません）

以下はビルドツールチェーンで使用されるもので、配布されるバイナリには
含まれないため、その条件が成果物に及ぶことはありません。

- [vite](https://github.com/vitejs/vite)（MIT）
- [lightningcss](https://github.com/parcel-bundler/lightningcss)（MPL-2.0）— TailwindCSS が内部で使用
- [aria-query](https://github.com/A11yance/aria-query), [axobject-query](https://github.com/A11yance/axobject-query)（Apache-2.0）— Svelte コンパイラが使用
- [source-map-js](https://github.com/7rulnik/source-map-js)（BSD-3-Clause）

## ApexCharts について（成果物には含まれません）

`apexcharts` は flowbite-svelte の依存関係として `npm install` 時に
ダウンロードされますが、**ビルド成果物には含まれません**。本プロジェクトが
flowbite-svelte から使用しているのは `Button` と `Badge` のみで、
チャート関連のコードは Vite のツリーシェイキングによって除外されます
（`dist/assets/*.js` を検索しても ApexCharts のコードは存在しません）。

ApexCharts は OSI 承認のオープンソースライセンスではなく、
年間売上 200 万 USD 未満の組織のみ無償で利用できる収益条件付きの
デュアルライセンスで提供されています。本プロジェクトの配布物には
含まれないため、この条件が shiboq の利用者に及ぶことはありませんが、
**flowbite-svelte のチャート系コンポーネントを新たに使用する場合は
この条件が適用される点に注意してください。**
