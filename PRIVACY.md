# Privacy Policy / プライバシーポリシー

Last updated / 最終更新日: 2026-10-04

## English

shiboq is a desktop application for running jq queries against JSON on your own computer.

### Information we collect

None. shiboq does not collect, transmit, or share any personal information.

- The application does not communicate with any server operated by the developer or by third parties.
- It has no user accounts, analytics, advertising, or crash reporting.

### Data stored on your device

- Items you explicitly save (the JSON, the query, and its result) are stored only on your device, in the `.shiboq/saved` folder under your user profile (for example `C:\Users\<name>\.shiboq\saved` on Windows, `~/.shiboq/saved` on Linux).
- Display preferences (such as pane width) are stored locally on your device.
- This data never leaves your device. You can delete it at any time from within the app or by removing the `.shiboq` folder.

### Files and clipboard

- shiboq reads a file only when you choose it in the "open file" dialog.
- shiboq writes to the clipboard only when you press a copy button. It does not read the clipboard.

### Third-party components

On Windows, shiboq displays its user interface with the Microsoft Edge WebView2 Runtime, which is a component of Windows. WebView2 may send diagnostic data to Microsoft independently of shiboq. That data is governed by the [Microsoft Privacy Statement](https://privacy.microsoft.com/privacystatement).

### Web version

shiboq is also available as a web page at <https://morststs.github.io/shiboq/>.

- The JSON, queries, and results are processed entirely inside your browser (by a WebAssembly build of the same jq engine). They are never sent to any server.
- Items you save and display preferences are stored in your browser's local storage (IndexedDB and localStorage) on your device only. You can delete them from within the app or by clearing the site data in your browser.
- The page is hosted on GitHub Pages. When your browser downloads the page, GitHub receives the usual request information (such as your IP address) as part of serving it. The developer has no access to it, and it is governed by the [GitHub Privacy Statement](https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement). The page loads nothing from any other domain.

### Changes to this policy

If this policy changes, the updated version will be published at this location with a new "Last updated" date.

### Contact

Questions can be raised at <https://github.com/morststs/shiboq/issues>.

---

## 日本語

shiboqは、お使いのコンピューター上でJSONに対してjqクエリを実行するデスクトップアプリです。

### 収集する情報

ありません。shiboqは個人情報を収集・送信・共有しません。

- 開発者や第三者が運用するサーバーとは一切通信しません。
- ユーザーアカウント、利用状況の解析、広告、クラッシュレポートの機能はありません。

### お使いの端末に保存されるデータ

- 明示的に保存した項目（JSON・クエリ・実行結果）は、お使いの端末のユーザープロファイル配下の`.shiboq/saved`フォルダーにのみ保存されます（Windowsでは`C:\Users\<ユーザー名>\.shiboq\saved`、Linuxでは`~/.shiboq/saved`）。
- 表示設定（ペインの幅など）は端末内に保存されます。
- これらのデータが端末の外に送信されることはありません。アプリ内の削除操作、または`.shiboq`フォルダーの削除により、いつでも消去できます。

### ファイルとクリップボード

- ファイルを読み込むのは、「ファイルを開く」ダイアログで利用者が選択した場合のみです。
- クリップボードへ書き込むのは、利用者がコピーボタンを押した場合のみです。クリップボードの内容を読み取ることはありません。

### 第三者のコンポーネント

Windowsでは、画面表示にWindowsのコンポーネントであるMicrosoft Edge WebView2ランタイムを使用します。WebView2は、shiboqとは独立してMicrosoftへ診断データを送信する場合があります。そのデータの取り扱いは[Microsoft のプライバシーに関する声明](https://privacy.microsoft.com/privacystatement)に従います。

### Web版

shiboqは <https://morststs.github.io/shiboq/> でWebページとしても利用できます。

- JSON・クエリ・実行結果は、すべてブラウザ内で処理されます（同じjqエンジンをWebAssemblyにしたものを使用）。サーバーへ送信されることはありません。
- 保存した項目と表示設定は、お使いの端末のブラウザ内（IndexedDBおよびlocalStorage）にのみ保存されます。アプリ内の削除操作、またはブラウザのサイトデータの消去により、いつでも消去できます。
- ページはGitHub Pagesで配信しています。ブラウザがページを読み込む際、配信の過程でGitHubが通常のリクエスト情報（IPアドレスなど）を受け取ります。開発者はこれにアクセスできず、その取り扱いは[GitHub のプライバシーに関する声明](https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement)に従います。GitHub以外のドメインからは何も読み込みません。

### 本ポリシーの変更

本ポリシーを変更した場合は、「最終更新日」を更新したうえで、この場所に公開します。

### お問い合わせ

ご質問は <https://github.com/morststs/shiboq/issues> へお寄せください。
