# Microsoft Store 提出内容の控え

Partner Center（製品「shiboq」、種類: **MSIX または PWA アプリ**）に入力した内容の控え。
次回の更新提出や、製品を作り直したときにそのまま再利用する。
状況と注意点は [`CLAUDE.md`](../CLAUDE.md) の「Microsoft Storeへの提出（Partner Center）」を参照。

## 製品 ID（Partner Center「製品 ID の表示」）

| 項目 | 値 |
|---|---|
| Package/Identity/Name | `morststs.shiboq` |
| Package/Identity/Publisher | `CN=CE782894-A6A6-48CC-8F6D-36D71DA87B9A` |
| Package/Properties/PublisherDisplayName | `morststs` |
| Package Family Name (PFN) | `morststs.shiboq_q9kp0rnkmck24` |
| Microsoft Store ID | `9MV10MVFX78Z` |

最初の3つは `build/msix/AppxManifest.xml` に設定済み。

## 価格と提供の状況

- 市場: 世界中のすべての市場（新しいマーケットにも自動で提供: オン）
- 対象ユーザー: 一般ユーザー / 見つかりやすさ: 検索可能
- スケジュール: できるだけ早く / 購入の停止: 終了しない
- 価格: Default 市場グループで JPY・¥0（無料）。無料試用版・販売価格・組織のライセンスは既定のまま

## プロパティ

- カテゴリ: 開発者ツール / ユーティリティ（＋ユーティリティ & ツール）
- 個人情報: 「いいえ、製品では個人情報を使用しません」
- Web サイト: `https://github.com/morststs/shiboq`
- サポートの連絡先情報: `https://github.com/morststs/shiboq/issues`
- 電話番号・住所: **空欄**（入力すると Store ページで公開される）
- 表示モード（Mixed Reality）: PC / HoloLens ともにチェックなし
- 製品の公表: 「代替ドライブやリムーバブル ストレージへのインストール」のみチェック。
  OneDrive バックアップは外す（保存データはパッケージのデータフォルダーではなく
  `%USERPROFILE%\.shiboq` にあり、実際にはバックアップされないため）。
  録画とブロードキャストは外す（ゲーム向け）。アクセシビリティのテスト済み・
  ペン入力・生成 AI・Store 外の購入はチェックなし
- システム要件: キーボードとマウスの「最小ハードウェア要件」のみチェック、他はすべて未指定

## 年齢区分（IARC）

- 「IARC のアンケートへ回答する準備ができています」
- アプリの種類: その他のすべてのアプリの種類
- 質問はすべて「いいえ」（性・暴力などのコンテンツ、ユーザー コンテンツの共有、
  オンライン コンテンツ、年齢制限商品、現在地の共有、デジタル商品の購入、
  現金のリワード・暗号資産・NFT、ウェブブラウザ/検索エンジン、ニュース/教育）
- 評価機関から直接取得した評価・物理メディア: いいえ

## パッケージ

- `shiboq.msix`（v1.0.0.0, X64, Windows.Desktop min 10.0.22000.0）をアップロード済み・Validated
- デバイス ファミリ: Windows 10/11 Desktop のみ。「将来のデバイス ファミリを Microsoft に任せる」は外す
- アップロード時の警告「restricted capabilities require approval: runFullTrust」は想定どおり
  （申請オプションで理由を書く）

## Store 登録情報（日本語のみ。追加言語なし）

### 説明

```
shiboq（シボク）は、JSON に対して jq クエリを繰り返し実行し、結果をその場で確認できるデスクトップアプリです。

JSON とクエリを入力すると、手を止めてから約 0.5 秒で自動的に実行され、結果が表示されます。jq の書き方を試行錯誤しながら、目的のデータを「絞り込む」作業に向いています。

・jq エンジンを内蔵しているため、jq コマンドのインストールは不要です
・JSON から推論したキーの一覧（スキーマ）を表示し、クエリで使っているキーを強調表示します
・クエリ入力中にキー名や jq の関数を補完します
・よく使う JSON とクエリの組み合わせを保存し、ワンクリックで呼び出せます
・JSON の整形や、スキーマを JSON Schema 形式でコピーする機能があります
・jq の -r に相当する生出力モードに対応しています

すべての処理はお使いの PC 内で完結し、インターネットへの通信は行いません。入力した JSON や保存したデータが外部に送信されることはありません。
```

読み方は「シボク」を推奨した（`q` は「Iraq → イラク」と同じく「ク」と読むのが自然なため）。
ユーザーが最終決定したかは未確認。実際に入力した説明文の表記は Partner Center で確認すること。

### 短い説明（推奨 270 文字以下）

```
JSON に jq クエリを繰り返し実行して、結果をその場で確認できるデスクトップアプリです。入力を止めると約 0.5 秒で自動実行。キーの一覧表示と使用中キーの強調、入力補完、クエリの保存に対応。jq のインストールは不要で、データを外部に送信しません。
```

### 製品の機能

```
jq クエリの自動実行と結果の即時表示
jq エンジン内蔵（jq コマンドのインストール不要）
JSON のキー一覧（スキーマ）表示と、使用中のキーの強調
キー名と jq 関数の入力補完
JSON とクエリの組み合わせの保存と呼び出し
JSON の整形、スキーマの JSON Schema 形式でのコピー
完全オフライン動作
```

### キーワード（最大 7 個）

単独の `jq` と `JSON` は入力欄が受け付けなかった（原因不明。短すぎる／他のキーワードと重複、のどちらかと推測）。
説明文に何度も出てくるので検索上の実害は小さい。代わりに次を使う。

```
jq クエリ / JSON Schema / JSON整形 / JSON 加工 / 開発者ツール / データ抽出 / フィルター
```

### その他

- 著作権と商標の情報: `© 2026 morststs`
- 開発元: `morststs`
- 短いタイトル・ボイス タイトル: 空（Xbox 用）
- 追加のライセンス条項（任意）:
  ```
  本アプリのソースコードは MIT License で公開しています。https://github.com/morststs/shiboq/blob/main/LICENSE
  同梱するサードパーティ製ソフトウェアのライセンスは https://github.com/morststs/shiboq/blob/main/THIRD_PARTY_LICENSES.md を参照してください。
  ```
- プライバシー ポリシーの URL: `https://github.com/morststs/shiboq/blob/main/PRIVACY.md`
- トレーラー: なし。Xbox 画像: なし

### ロゴ・画像

`build/appicon.svg` から `rsvg-convert`（`librsvg2-bin`）で生成した（生成物はリポジトリに含めない）。

| 枠 | サイズ | 作り方 |
|---|---|---|
| 1:1 ボックス アート | 2160×2160 | `rsvg-convert -w 2160 -h 2160 build/appicon.svg` |
| 1:1 アプリ タイル アイコン | 300×300 / 150×150 / 71×71 | 同上でサイズ違い |
| 9:16 ポスター アート | 1440×2160 | 全面 `#E8890C` の矩形に、appicon.svg のじょうごの `path` を `translate(3.2,315.6) scale(1.4)` で中央配置した SVG を変換 |
| 16:9 スーパー ヒーロー アート（任意） | 3840×2160 | 同様に `translate(1100.8,206.4) scale(1.6)` |

じょうごのパス（appicon.svg と同一）:
`M188 268 h648 L586 612 v186 a26 26 0 0 1 -26 26 h-96 a26 26 0 0 1 -26 -26 V612 L188 268 Z`（fill `#ffffff`）

### スクリーンショット用のデータ

アプリと同じ `RunQuery` で結果を確認済み。スキーマペインで 8 キーが「使用中」になり、
使っていない `id`・`country`・`osaka`・`store`・`updatedAt` との対比が出る。

JSON:

```json
{
  "store": "シボク書店",
  "updatedAt": "2026-10-03",
  "books": [
    { "id": 101, "title": "はじめての jq", "author": { "name": "山田 花子", "country": "JP" }, "price": 2640, "tags": ["jq", "json", "cli"], "stock": { "tokyo": 12, "osaka": 3 } },
    { "id": 102, "title": "JSON 設計パターン", "author": { "name": "佐藤 一郎", "country": "JP" }, "price": 3300, "tags": ["json", "api"], "stock": { "tokyo": 0, "osaka": 5 } },
    { "id": 103, "title": "シェル芸レシピ集", "author": { "name": "鈴木 次郎", "country": "JP" }, "price": 1980, "tags": ["shell", "jq"], "stock": { "tokyo": 7, "osaka": 0 } },
    { "id": 104, "title": "Data Wrangling at Scale", "author": { "name": "Alex Smith", "country": "US" }, "price": 4950, "tags": ["data", "json"], "stock": { "tokyo": 2, "osaka": 1 } }
  ]
}
```

クエリ:

```
.books
| map(select(.tags | index("json")))
| sort_by(-.price)
| map({title, author: .author.name, price, inTokyo: (.stock.tokyo > 0)})
```

（貼り付けたあと「整形」を押すと見やすくなる。）Store のスクリーンショットは横 1366×縦 768 以上の PNG。
アプリの初期ウィンドウは 1280×800 なので、最大化してから撮る。

## 申請オプション

- 公開の保留: 「認定されたらすぐに…公開する」
- 制限付き機能 runFullTrust の理由（**上限 500 文字**。最初の長文は途中で切れた）:

```
Win32 desktop app (Go + Wails) packaged as MSIX, so runFullTrust is required to launch shiboq.exe. Full trust is used only for normal desktop behavior: showing the UI with WebView2, opening a JSON file the user selects, saving the user's items under %USERPROFILE%\.shiboq, and copying to the clipboard on request. No drivers, services, background tasks, admin rights, or network access.
```

## 追加のテスト情報（認定の注意書き。顧客には非表示）

左メニュー「追加のテスト情報」で入力する（申請一覧には出てこないので忘れやすい）。資格情報は空。

```
shiboq is a desktop tool for running jq queries against JSON and viewing the results.

How to test:
1. Launch the app. A sample JSON and a sample query (.name) are pre-filled, and the result appears automatically.
2. Edit the JSON (middle-top pane) or the query (right-top pane). The result (right-bottom pane) updates about 0.5 seconds after you stop typing.
3. Press "+ 保存" (Save) in the left pane or Ctrl+S to save the current JSON/query/result. Click a saved item to restore it; hover it and press "x" to delete it.
4. "整形" formats the JSON. "コピー" in the schema pane copies the inferred JSON Schema to the clipboard.

Notes:
- No account, sign-in, or license key is required. All features are available immediately.
- The app works fully offline and does not send any data to any server. It does not collect personal information.
- Saved items are stored only on the local device, under %USERPROFILE%\.shiboq\saved.
- The UI language is Japanese.
- Dependencies: Microsoft Edge WebView2 Runtime (included in Windows 11). No drivers, NT services, or other products are required.
- runFullTrust is declared because this is a packaged Win32 desktop application (built with Go and Wails).
```
