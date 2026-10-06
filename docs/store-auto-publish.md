# Microsoft Store への自動公開

> **現在は使っていません（2026-10-06）。** Store への更新は半自動（タグの push で MSIX を
> 自動ビルドし、Partner Center へは手でアップロードする）で運用しています（README 参照）。
> この仕組みには Microsoft Entra のテナント（`○○@○○.onmicrosoft.com` のような組織用の
> アカウント）が必要で、個人の Microsoft アカウントだけでは準備できないためです。
> 下記の準備を済ませてリポジトリ変数を設定すれば、そのまま使えます（設定するまでは申請しません）。

`v*` タグを push すると、GitHub Actions（[`msix.yml`](../.github/workflows/msix.yml)）が
MSIX をビルドし、Microsoft Store Developer CLI（`msstore`）で Store へ申請して認定に出します。

Store への認証には **GitHub Actions の OIDC トークン**（Microsoft Entra の
フェデレーション資格情報）を使います。クライアントシークレットを使わないので、
期限切れで公開が止まることも、リポジトリに秘密情報を置くこともありません。

以下の準備（1〜4）が終わるまでは、タグを push しても **MSIX のビルドだけ**行い、
申請はしません（ビルド自体は成功します）。

## 全体の流れ

```
v0.5.0 タグを push
  ├─ release.yml … exe の zip を GitHub Release に添付（従来どおり）
  └─ msix.yml
       ├─ build-msix   … MSIX 1.5.0 をビルド（タグ vX.Y.Z → (X+1).Y.Z）
       └─ publish-store … GitHub の OIDC トークンで Entra にサインイン
                          → msstore publish で申請を作り、認定に出す
                          → 審査（数時間〜数日）を通ると Store に公開
```

MSIX のバージョンはタグの先頭に 1 を足したものです（`v0.4.0` → `1.4.0`、`v1.0.0` → `2.0.0`）。
Store は先頭 0 を受け付けず、公開中より大きい番号が必要なため、この対応で常に増えていきます。

## 準備（初回だけ）

画面の項目名は Partner Center の英語表記です（`msstore` CLI の案内と同じ）。日本語表示では
「アカウント設定」「ユーザー管理」「Microsoft Entra アプリケーション」のような名前になっています。

### 1. Partner Center のアカウントを Microsoft Entra テナントに関連付ける

Partner Center の **Account settings → Organization profile → Tenants** を開きます。

- テナントが表示されていれば、そのまま次へ進みます。
- 何も無ければ「Microsoft Entra ID を関連付ける」か「新しい Microsoft Entra ID を作成する」を
  選びます。個人の開発者アカウントでは、新しく作るのが簡単です。作成時に設定する
  管理者アカウント（`○○@○○.onmicrosoft.com`）は、手順 3 でも使うので控えておきます。

### 2. Store を操作するアプリ（Entra アプリケーション）を作る

Partner Center の **Account settings → User management → Microsoft Entra applications** を開き、
手順 1 のテナントの管理者でサインインした状態で次のようにします。

1. **Add Microsoft Entra Application** を押す
2. **Create Microsoft Entra Application** を選んで **Continue**
3. フォームに入力する（例）
   - Name: `shiboq-github-actions`
   - Reply URI / App ID URI: `https://<テナントのドメイン>/shiboq-github-actions`
     （このアプリはブラウザでのサインインに使わないので、形式が合っていれば何でもかまいません）
4. **Next** を押し、ロールに **Manager (Windows)** を選んで **Create**

作成したアプリの画面に表示される次の 2 つを控えます。

- **Client ID**（アプリの GUID）
- **Tenant ID**（テナントの GUID）

### 3. GitHub Actions からのサインインを許可する（フェデレーション資格情報）

[Microsoft Entra 管理センター](https://entra.microsoft.com/)に手順 1 のテナントの管理者でサインインし、
**アプリの登録 → すべてのアプリケーション → `shiboq-github-actions`** を開きます。

**証明書とシークレット → フェデレーション資格情報 → 資格情報の追加** で、次のように入力します。

| 項目 | 値 |
|---|---|
| フェデレーション資格情報のシナリオ | GitHub Actions による Azure リソースのデプロイ |
| 組織 | `morststs` |
| リポジトリ | `shiboq` |
| エンティティ型 | 環境（Environment） |
| GitHub 環境名 | `microsoft-store` |
| 名前 | `shiboq-msix`（任意） |

保存すると、サブジェクト識別子が `repo:morststs/shiboq:environment:microsoft-store`、
対象ユーザー（Audience）が `api://AzureADTokenExchange` になっていることを確認します。
**この 2 つが 1 文字でも違うとサインインに失敗します。**

### 4. Seller ID を控え、リポジトリ変数に登録する

Partner Center の **Account settings → Organization profile → Legal info** にある
**Seller ID**（数字）を控えます。

手順 2・4 で控えた 3 つを、GitHub のリポジトリ変数に登録します。どれも秘密情報ではないので、
Secrets ではなく Variables に置きます（ログにもそのまま表示されます）。

```bash
gh variable set MSSTORE_TENANT_ID --body "<Tenant ID>"
gh variable set MSSTORE_CLIENT_ID --body "<Client ID>"
gh variable set MSSTORE_SELLER_ID --body "<Seller ID>"
```

GitHub の画面からは **Settings → Secrets and variables → Actions → Variables** で登録できます。

### 5. 下書きで試す

最初は認定に出さず、下書きで動作を確かめます。

```bash
gh workflow run msix.yml -f version=1.4.0 -f publish=draft
```

成功すると、Partner Center の shiboq に新しい申請（パッケージ 1.4.0.0）が下書きとして
できています。内容を確認して、問題なければ Partner Center で **送信** します
（または `-f publish=submit` で実行し直します）。

以後は `v*` タグを push するだけで、自動で認定に出ます。

## 運用

- **公開まで**: 申請は認定（審査）を通ってから Store に反映されます。数時間〜数日かかるため、
  GitHub Release より遅れて更新されます（Release の本文にもそう書いてあります）。
- **承認してから申請したい場合**: GitHub の **Settings → Environments → microsoft-store** で
  **Required reviewers** に自分を追加すると、申請の前に Actions の画面で承認を求められます。
- **申請だけやり直す**: Actions の「Build MSIX for Microsoft Store」を手動実行し、
  バージョンと `publish`（`none` / `draft` / `submit`）を選びます。

## 注意

- **Partner Center で作業中の申請は削除されます。** `msstore publish` は、送信前の申請
  （下書き）があるとそれを削除して作り直します。登録情報（説明文・スクリーンショット等）は
  公開中の申請から引き継がれますが、下書きで手直ししていた内容は失われます。
  登録情報を手で直すときは、タグを push する前に送信を済ませてください。
- **古いパッケージは差し替えられます。** 新しい `.msix` をアップロードし、前回の `.msix` は
  削除扱いにします。
- **同じバージョンは出せません。** 公開中より大きい MSIX のバージョンが必要です。タグから
  自動で決まるので、通常は意識する必要はありません。
- 環境 `microsoft-store` は `main` ブランチと `v*` タグからしか使えないようにしてあります
  （別のブランチから Store へ申請されるのを防ぐため）。

## うまくいかないとき

| 症状 | 確認すること |
|---|---|
| `AADSTS70021` / `No matching federated identity record found` | 手順 3 のサブジェクト識別子が `repo:morststs/shiboq:environment:microsoft-store` になっているか |
| `AADSTS700016` / アプリが見つからない | `MSSTORE_CLIENT_ID` と `MSSTORE_TENANT_ID` の取り違え |
| 401 / 403（Store API） | 手順 2 でロールに **Manager (Windows)** を選んだか。Partner Center の User management に表示されているか |
| `リポジトリ変数が未設定です` | 手順 4 の 3 つの変数 |
| パッケージのバージョンエラー | 公開中より大きいバージョンか（手動実行時の `version`） |

## 参考

- [microsoft/msstore-cli](https://github.com/microsoft/msstore-cli)（`msstore` CLI。OIDC 対応は v0.4.2 から。
  ワークフローでは v0.4.3 に固定）
- [microsoft/microsoft-store-apppublisher](https://github.com/microsoft/microsoft-store-apppublisher)（CLI を
  セットアップする GitHub Action）
