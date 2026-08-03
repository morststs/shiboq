# jqrepl 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** jqコマンドを繰り返し実行して結果を確認できる、5ペイン構成のWails v2デスクトップアプリ（jqrepl）を作る。

**Architecture:** Go 1.24 + Wails v2.12.0 バックエンド（jq実行は `itchyny/gojq`、履歴は `~/.jqrepl/history/{UUID}.json` に1件1ファイルで永続化）+ Svelte 5 (runes) + Vite 7 フロントエンド（Monaco Editorでコード編集、TailwindCSS + Flowbite Svelteで見た目）。`/workspaces/sirusita`（Wails v2 + Svelte 5 + Monaco構成のメモアプリ）の技術スタック・開発環境・永続化パターンを踏襲する。

**Tech Stack:** Go 1.24 / Wails v2.12.0 / itchyny/gojq v0.12.19 / google/uuid v1.6.0 / Svelte 5 / Vite 7 / monaco-editor 0.55.1 / TailwindCSS 4 / flowbite-svelte 1.x / Docker（sirusitaはPodmanだが、本プロジェクトの開発環境ではDockerを使う。コマンドは同一のためどちらでも動く）

## Global Constraints

- 全てのGoビルド・テスト・Wailsコマンドは `./scripts/dev-run.sh <command>` 経由でコンテナ内で実行する（ホストにGo/Node/Wailsをインストールしない。GTK/WebKit開発パッケージがホストに無いため、コンテナ無しでは `wails build` はおろか `go test`（`main.go` を含むパッケージ全体のコンパイルが必要なため）も通らない）
- Go言語コードは全て `package main`（sirusitaの flat な構成に合わせる。サブパッケージに分割しない）
- 日本語UI: 各ペインの見出し・エラーメッセージは日本語で表示する
- 実行環境依存のコマンド例は `docker` を使う（Podman環境では `podman` に読み替え可能。動作するコマンドの形式は同一）
- 新規に追加するGoの依存はこの計画に明記したバージョンをピン留めする（`go get <module>@<version>` を使い、`@latest` は使わない）。フロントエンド（npm）の依存はsirusitaと同じ慣習でキャレット範囲表記（`^x.y.z`）のままでよい。`package-lock.json` をコミットすることで、実際に解決されたバージョンは再現可能になる

---

## 事前検証済みの事実（実装時の前提）

この計画を書く過程で、以下を実際にコンテナ内でビルド・実行して確認済み。計画中のコマンド・コードはこれに基づく。

- `github.com/itchyny/gojq` v0.12.19 の `Query`/`Term`/`Suffix`/`Index`/`ParseError`/`HaltError` のAST構造と挙動（`gojq.Parse` のエラー、`query.Run(input)` でのビルトイン関数動作、`.foo.bar` のようなフィールドアクセスの実行時エラー形式）
- Dockerイメージ `golang:1.24-bookworm` に GTK3/WebKit2GTK/mingw-w64/nsis を apt-get でインストールし、Wails CLI v2.12.0 を `go install` できること
- そのコンテナ内で `wails build`（GTKリンクを含む完全ビルド）が成功し、実行可能バイナリが生成されること
- コンテナを非rootユーザー（ホストと同じUID/GID）で実行する際、`GOPATH`/`GOCACHE`/`npm` のキャッシュ先を `/tmp` 直下の単一階層パスにバインドマウントしないと `permission denied` になること（`/go`・`/root` はイメージ内で root 所有のため）

---

### Task 1: 開発コンテナ・プロジェクトの土台（Wailsの空アプリがビルドできる状態）

**Files:**
- Create: `Dockerfile`
- Create: `scripts/dev-run.sh`
- Create: `.gitignore`（追記）
- Create: `go.mod`, `go.sum`
- Create: `main.go`
- Create: `app.go`
- Create: `wails.json`
- Create: `frontend/package.json`
- Create: `frontend/package-lock.json`（`npm install` で生成）
- Create: `frontend/vite.config.js`
- Create: `frontend/svelte.config.js`
- Create: `frontend/index.html`
- Create: `frontend/src/main.js`
- Create: `frontend/src/style.css`
- Create: `frontend/src/App.svelte`

**Interfaces:**
- Produces: `App` 構造体（`ctx context.Context`）、`func NewApp() *App`、`func (a *App) startup(ctx context.Context)`、`func (a *App) OpenJSONFile() (string, error)` — 以降のタスクはこの `App` に手を加えず、`JqService`・`HistoryService` を新設して `main.go` の `Bind` に追加していく

- [ ] **Step 1: `Dockerfile` を作成する**

```dockerfile
FROM golang:1.24-bookworm

# Wails 用のシステム依存パッケージ（sirusitaのDockerfileを流用）
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    libgtk-3-dev \
    libwebkit2gtk-4.0-dev \
    libwebkit2gtk-4.1-dev \
    build-essential \
    pkg-config \
    gcc-mingw-w64-x86-64 \
    nsis \
    && rm -rf /var/lib/apt/lists/*

# Node.js 22 LTS（Svelte 5 / Vite 7 は Node 20.19+ / 22.12+ が必要）
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/* \
    && node -v && npm -v

# Wails CLI（go.mod と揃えたバージョンに固定する）
RUN go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0

WORKDIR /app
CMD ["bash"]
```

- [ ] **Step 2: イメージをビルドする**

Run: `docker build -t jqrepl-dev .`
Expected: `docker images jqrepl-dev` に一覧が出る（数分かかる。`apt-get`が失敗する場合はネットワークポリシーで `deb.debian.org` / `deb.nodesource.com` へのアクセスが許可されているか確認する）

- [ ] **Step 3: コンテナ実行ヘルパー `scripts/dev-run.sh` を作成する**

```bash
#!/usr/bin/env bash
# jqrepl-dev イメージのコンテナ内でコマンドを実行するヘルパー。
# GOPATH/GOCACHE/npmキャッシュは /tmp 直下の単一階層パスにバインドマウントする。
# （/go, /root はイメージ内で root 所有のため、非rootユーザー実行時に書き込めない。
#   /tmp/foo のように /tmp のすぐ下の1階層なら、Dockerが中間ディレクトリを
#   rootで自動生成する問題を避けられる。）
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$ROOT_DIR/.devcache/gopath" "$ROOT_DIR/.devcache/gobuild" "$ROOT_DIR/.devcache/npm"

exec docker run --rm \
  --user "$(id -u):$(id -g)" \
  -e HOME=/tmp \
  -e GOPATH=/tmp/gopath \
  -e GOCACHE=/tmp/gobuild \
  -e http_proxy="${http_proxy:-}" \
  -e https_proxy="${https_proxy:-}" \
  -e no_proxy="${no_proxy:-}" \
  -v "$ROOT_DIR":/app \
  -v "$ROOT_DIR/.devcache/gopath":/tmp/gopath \
  -v "$ROOT_DIR/.devcache/gobuild":/tmp/gobuild \
  -v "$ROOT_DIR/.devcache/npm":/tmp/.npm \
  -w /app \
  jqrepl-dev "$@"
```

Run: `chmod +x scripts/dev-run.sh`

- [ ] **Step 4: `.gitignore` を更新する**

`.gitignore` の内容を次の通りにする（既存の `/tmp` に追記する形）:

```
/tmp
build/bin
node_modules
frontend/dist
.devcache/
```

- [ ] **Step 5: Goモジュールを初期化し、Wailsを依存に追加する**

Run: `./scripts/dev-run.sh go mod init jqrepl`
Run: `./scripts/dev-run.sh go get github.com/wailsapp/wails/v2@v2.12.0`

Expected: `go.mod` に `module jqrepl` と `go 1.24` が記載され、`go.sum` が生成される

- [ ] **Step 6: `main.go` を作成する**

```go
package main

import (
	"embed"
	"net/http"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// noCacheMiddleware はフロントエンド資産に no-store を付与する。
// WebView がキャッシュすると、ビルドし直しても古い画面が表示され続けるため、
// 毎回必ず再取得させる（sirusitaと同じ対策）。
func noCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "jqrepl",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets:     assets,
			Middleware: noCacheMiddleware,
		},
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 30, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			Theme: windows.Dark,
		},
		Linux: &linux.Options{
			ProgramName: "jqrepl",
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
```

- [ ] **Step 7: `app.go` を作成する**

```go
package main

import (
	"context"
	"os"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// OpenJSONFile はネイティブのファイル選択ダイアログを開き、
// 選択された .json ファイルの内容を文字列で返す。
// キャンセル時は空文字とnilエラーを返す。
func (a *App) OpenJSONFile() (string, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "JSONファイルを開く",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "JSON (*.json)", Pattern: "*.json"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

- [ ] **Step 8: `wails.json` を作成する**

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "jqrepl",
  "outputfilename": "jqrepl",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "auto",
  "author": {
    "name": "",
    "email": ""
  }
}
```

- [ ] **Step 9: フロントエンドの土台を作成する**

`frontend/package.json`:

```json
{
  "name": "frontend",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "devDependencies": {
    "@sveltejs/vite-plugin-svelte": "^6.2.1",
    "@tailwindcss/vite": "^4.3.1",
    "flowbite": "^4.0.2",
    "svelte": "^5.56.3",
    "tailwindcss": "^4.3.1",
    "vite": "^7.1.12"
  },
  "dependencies": {
    "flowbite-svelte": "^1.33.1",
    "monaco-editor": "^0.55.1"
  }
}
```

`frontend/vite.config.js`:

```js
import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [tailwindcss(), svelte()]
})
```

`frontend/svelte.config.js`:

```js
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

// flowbite-svelte はTypeScript入りの.svelteを配布しているため、
// Svelte 5 の組み込みTS除去だけではビルドが失敗する箇所がある。
// esbuildによるTSトランスパイルを明示的に有効化する（sirusitaと同じ対策）。
export default {
  preprocess: vitePreprocess({ script: true }),
}
```

`frontend/index.html`:

```html
<!DOCTYPE html>
<html lang="ja" class="dark">
<head>
    <meta charset="UTF-8"/>
    <meta content="width=device-width, initial-scale=1.0" name="viewport"/>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Noto+Sans+JP:wght@400;700&family=Source+Code+Pro:wght@400;700&display=swap" rel="stylesheet">
    <title>jqrepl</title>
</head>
<body>
<div id="app"></div>
<script src="./src/main.js" type="module"></script>
</body>
</html>
```

`frontend/src/main.js`:

```js
import './style.css'
import { mount } from 'svelte'
import App from './App.svelte'

const app = mount(App, {
  target: document.getElementById('app')
})

export default app
```

`frontend/src/style.css`:

```css
@import "tailwindcss";
@plugin "flowbite/plugin";
@custom-variant dark (&:where(.dark, .dark *));
@source "../node_modules/flowbite-svelte/dist";

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  height: 100%;
  font-family: "Noto Sans JP", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  font-size: 14px;
  color: #cccccc;
  background: #1e1e1e;
}

#app {
  height: 100%;
}
```

`frontend/src/App.svelte`（後続タスクで5ペイン構成に置き換える、動作確認用の仮実装）:

```svelte
<script>
  let message = 'jqrepl';
</script>

<main>
  <h1>{message}</h1>
</main>

<style>
  main {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #cccccc;
  }
</style>
```

- [ ] **Step 10: 依存関係をインストールする**

Run: `./scripts/dev-run.sh sh -c "cd frontend && npm install"`
Expected: `frontend/node_modules` と `frontend/package-lock.json` が生成される

- [ ] **Step 11: ビルドを実行して土台が動くことを確認する**

Run: `./scripts/dev-run.sh wails build`
Expected: `Built '/app/build/bin/jqrepl' in ...` と表示され、`build/bin/jqrepl` が生成される（GUIは表示できない環境でもビルド自体は成功する。実際にウィンドウを開いての目視確認は、GUI表示可能な環境で `wails dev` を使って別途行う）

- [ ] **Step 12: コミットする**

```bash
git add Dockerfile scripts/dev-run.sh .gitignore go.mod go.sum main.go app.go wails.json frontend
git commit -m "$(cat <<'EOF'
feat: Wailsアプリの土台を作成する

sirusitaの技術スタック（Wails v2 + Svelte 5 + Vite 7）を踏襲した
最小構成のWailsアプリを用意する。以降のタスクでjq実行・履歴・5ペインUIを追加する。
EOF
)"
```

---

### Task 2: 履歴の永続化（HistoryService）

**Files:**
- Create: `history_service.go`
- Create: `history_service_test.go`

**Interfaces:**
- Consumes: なし（このタスクは独立して動作する）
- Produces: `type HistoryEntry struct { ID string; JSON string; Query string; Result string; CreatedAt time.Time }`（JSONタグ: `id`/`json`/`query`/`result`/`createdAt`）、`type HistoryService struct{ historyDir string }`、`func NewHistoryService(historyDir string) *HistoryService`、`func (s *HistoryService) SaveHistory(entry HistoryEntry) (HistoryEntry, error)`、`func (s *HistoryService) ListHistory() ([]HistoryEntry, error)`、`func (s *HistoryService) DeleteHistory(id string) error` — Task 5 で `main.go` の `Bind` に追加する

- [ ] **Step 1: `google/uuid` を依存に追加する**

Run: `./scripts/dev-run.sh go get github.com/google/uuid@v1.6.0`

- [ ] **Step 2: 失敗するテストを書く（`history_service_test.go`）**

```go
package main

import "testing"

func TestHistoryServiceCRUD(t *testing.T) {
	svc := NewHistoryService(t.TempDir())

	saved, err := svc.SaveHistory(HistoryEntry{JSON: `{"a":1}`, Query: ".a", Result: "1"})
	if err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}
	if saved.ID == "" {
		t.Fatalf("expected generated ID")
	}

	list, err := svc.ListHistory()
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(list) != 1 || list[0].ID != saved.ID {
		t.Fatalf("list = %+v", list)
	}

	if err := svc.DeleteHistory(saved.ID); err != nil {
		t.Fatalf("DeleteHistory: %v", err)
	}
	list, _ = svc.ListHistory()
	if len(list) != 0 {
		t.Fatalf("expected empty list after delete, got %+v", list)
	}
}

func TestHistoryServiceOrder(t *testing.T) {
	svc := NewHistoryService(t.TempDir())
	_, err := svc.SaveHistory(HistoryEntry{JSON: "1", Query: ".", Result: "1"})
	if err != nil {
		t.Fatalf("SaveHistory (first): %v", err)
	}
	second, err := svc.SaveHistory(HistoryEntry{JSON: "2", Query: ".", Result: "2"})
	if err != nil {
		t.Fatalf("SaveHistory (second): %v", err)
	}
	list, err := svc.ListHistory()
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %+v", list)
	}
	if list[0].ID != second.ID {
		t.Errorf("expected newest first: %+v", list)
	}
}

func TestHistoryServiceInvalidID(t *testing.T) {
	svc := NewHistoryService(t.TempDir())
	if err := svc.DeleteHistory("not-a-uuid"); err == nil {
		t.Fatalf("expected error for invalid ID")
	}
}
```

- [ ] **Step 3: テストが失敗することを確認する**

Run: `./scripts/dev-run.sh go test -run TestHistoryService -v ./...`
Expected: `history_service.go` が無いためコンパイルエラーで失敗する

- [ ] **Step 4: `history_service.go` を実装する**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// HistoryEntry は1回のjq実行の記録。ID未指定でSaveHistoryに渡すとUUIDを発行する。
type HistoryEntry struct {
	ID        string    `json:"id"`
	JSON      string    `json:"json"`
	Query     string    `json:"query"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"createdAt"`
}

// HistoryService は履歴を `{historyDir}/{UUID}.json` として1件1ファイルで永続化する。
type HistoryService struct {
	historyDir string
}

func NewHistoryService(historyDir string) *HistoryService {
	os.MkdirAll(historyDir, 0755)
	return &HistoryService{historyDir: historyDir}
}

func isValidHistoryID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// SaveHistory は履歴を保存する。IDが空文字ならUUIDを新規発行し、
// CreatedAtが未設定なら現在時刻を設定して返す。
func (s *HistoryService) SaveHistory(entry HistoryEntry) (HistoryEntry, error) {
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = uuid.New().String()
	} else if !isValidHistoryID(entry.ID) {
		return HistoryEntry{}, fmt.Errorf("invalid history ID: %s", entry.ID)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return HistoryEntry{}, fmt.Errorf("failed to marshal history entry: %w", err)
	}
	path := filepath.Join(s.historyDir, entry.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return HistoryEntry{}, fmt.Errorf("failed to write history entry: %w", err)
	}
	return entry, nil
}

// ListHistory は履歴を新しい順（CreatedAt降順）で返す。
func (s *HistoryService) ListHistory() ([]HistoryEntry, error) {
	entries, err := os.ReadDir(s.historyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read history dir: %w", err)
	}
	var result []HistoryEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.historyDir, e.Name()))
		if err != nil {
			continue
		}
		var entry HistoryEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (s *HistoryService) DeleteHistory(id string) error {
	if !isValidHistoryID(id) {
		return fmt.Errorf("invalid history ID: %s", id)
	}
	path := filepath.Join(s.historyDir, id+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("history not found: %s", id)
	}
	return os.Remove(path)
}
```

- [ ] **Step 5: テストが通ることを確認する**

Run: `./scripts/dev-run.sh go test -run TestHistoryService -v ./...`
Expected: `PASS`（3件すべて）

- [ ] **Step 6: コミットする**

```bash
git add history_service.go history_service_test.go go.mod go.sum
git commit -m "$(cat <<'EOF'
feat: 履歴の永続化(HistoryService)を追加する

実行履歴を ~/.jqrepl/history/{UUID}.json として1件1ファイルで
保存・一覧・削除できるようにする。
EOF
)"
```

---

### Task 3: スキーマ推論（schema.go）

**Files:**
- Create: `schema.go`
- Create: `schema_test.go`

**Interfaces:**
- Consumes: なし
- Produces: `type SchemaNode struct { Key string; Type string; Children []SchemaNode }`（JSONタグ: `key`/`type`/`children,omitempty`）、`func InferSchema(jsonText string) ([]SchemaNode, error)` — Task 4 で `JqService.InferSchema` から呼び出す

- [ ] **Step 1: 失敗するテストを書く（`schema_test.go`）**

```go
package main

import "testing"

func TestInferSchemaNested(t *testing.T) {
	nodes, err := InferSchema(`{"name":"taro","age":20,"tags":["a","b"],"address":{"city":"tokyo"},"note":null,"items":[{"id":1},{"id":2}]}`)
	if err != nil {
		t.Fatalf("InferSchema: %v", err)
	}
	got := map[string]SchemaNode{}
	for _, n := range nodes {
		got[n.Key] = n
	}
	if got["name"].Type != "string" {
		t.Errorf("name type = %s", got["name"].Type)
	}
	if got["age"].Type != "number" {
		t.Errorf("age type = %s", got["age"].Type)
	}
	if got["tags"].Type != "array<string>" {
		t.Errorf("tags type = %s", got["tags"].Type)
	}
	if got["note"].Type != "null" {
		t.Errorf("note type = %s", got["note"].Type)
	}
	if got["address"].Type != "object" || len(got["address"].Children) != 1 || got["address"].Children[0].Key != "city" {
		t.Errorf("address = %+v", got["address"])
	}
	if got["items"].Type != "array<object>" || len(got["items"].Children) != 1 || got["items"].Children[0].Key != "id" {
		t.Errorf("items = %+v", got["items"])
	}
}

func TestInferSchemaRootArray(t *testing.T) {
	// ルートが配列の場合は、先頭要素の形状をルートの形状として扱う。
	nodes, err := InferSchema(`[{"id":1,"active":true},{"id":2,"active":false}]`)
	if err != nil {
		t.Fatalf("InferSchema: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes = %+v, want 2 keys (id, active)", nodes)
	}
}

func TestInferSchemaEmptyArray(t *testing.T) {
	nodes, err := InferSchema(`{"items":[]}`)
	if err != nil {
		t.Fatalf("InferSchema: %v", err)
	}
	if nodes[0].Type != "array" {
		t.Errorf("items type = %s, want array", nodes[0].Type)
	}
}

func TestInferSchemaInvalidJSON(t *testing.T) {
	if _, err := InferSchema(`{invalid`); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `./scripts/dev-run.sh go test -run TestInferSchema -v ./...`
Expected: `schema.go` が無いためコンパイルエラーで失敗する

- [ ] **Step 3: `schema.go` を実装する**

```go
package main

import (
	"encoding/json"
	"fmt"
	"sort"
)

// SchemaNode はJSON値から推論したキーとその型を表すノード。
// ルートが配列の場合は先頭要素の形状をルートの形状とみなす（InferSchema参照）。
type SchemaNode struct {
	Key      string       `json:"key"`
	Type     string       `json:"type"`
	Children []SchemaNode `json:"children,omitempty"`
}

// InferSchema はJSONテキストを解析し、キーツリー（型付き）を推論する。
func InferSchema(jsonText string) ([]SchemaNode, error) {
	var value any
	if err := json.Unmarshal([]byte(jsonText), &value); err != nil {
		return nil, fmt.Errorf("JSONの解析に失敗しました: %w", err)
	}
	return inferChildren(value), nil
}

func inferChildren(value any) []SchemaNode {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		nodes := make([]SchemaNode, 0, len(keys))
		for _, k := range keys {
			nodes = append(nodes, SchemaNode{
				Key:      k,
				Type:     typeOf(v[k]),
				Children: inferChildren(v[k]),
			})
		}
		return nodes
	case []any:
		if len(v) == 0 {
			return nil
		}
		// 配列の要素形状は先頭要素をサンプルとして採用する。
		return inferChildren(v[0])
	default:
		return nil
	}
}

func typeOf(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case map[string]any:
		return "object"
	case []any:
		if len(v) == 0 {
			return "array"
		}
		return "array<" + typeOf(v[0]) + ">"
	default:
		return "unknown"
	}
}
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `./scripts/dev-run.sh go test -run TestInferSchema -v ./...`
Expected: `PASS`（4件すべて）

- [ ] **Step 5: コミットする**

```bash
git add schema.go schema_test.go
git commit -m "$(cat <<'EOF'
feat: JSONのキーツリー推論(InferSchema)を追加する

スキーマペイン表示用に、JSONからキー名と型（ネスト対応）を推論する。
ルートが配列の場合は先頭要素の形状を採用する。
EOF
)"
```

---

### Task 4: jqクエリ実行とASTからのキー抽出（JqService）

**Files:**
- Create: `jq_service.go`
- Create: `jq_service_test.go`

**Interfaces:**
- Consumes: `InferSchema(jsonText string) ([]SchemaNode, error)`（Task 3・同一パッケージ）
- Produces: `type RunResult struct { Result string; Error string }`（JSONタグ: `result`/`error`）、`type JqService struct{}`、`func NewJqService() *JqService`、`func (s *JqService) RunQuery(jsonText, query string) RunResult`、`func (s *JqService) ExtractUsedKeys(query string) ([]string, error)`、`func (s *JqService) InferSchema(jsonText string) ([]SchemaNode, error)` — Task 5 で `main.go` の `Bind` に追加する

- [ ] **Step 1: `itchyny/gojq` を依存に追加する**

Run: `./scripts/dev-run.sh go get github.com/itchyny/gojq@v0.12.19`

- [ ] **Step 2: 失敗するテストを書く（`jq_service_test.go`）**

```go
package main

import (
	"sort"
	"testing"
)

func TestRunQueryOK(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`{"name":"taro","age":20}`, `.name`)
	if r.Error != "" {
		t.Fatalf("unexpected error: %s", r.Error)
	}
	if r.Result != `"taro"` {
		t.Fatalf("Result = %q, want %q", r.Result, `"taro"`)
	}
}

func TestRunQueryMultipleOutputs(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`{"a":1,"b":2}`, `.a, .b`)
	if r.Error != "" {
		t.Fatalf("unexpected error: %s", r.Error)
	}
	if r.Result != "1\n2" {
		t.Fatalf("Result = %q, want %q", r.Result, "1\n2")
	}
}

func TestRunQueryInvalidJSON(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`{invalid`, `.`)
	if r.Error == "" {
		t.Fatalf("expected error for invalid JSON")
	}
	if r.Result != "" {
		t.Fatalf("Result should stay empty on error, got %q", r.Result)
	}
}

func TestRunQueryParseError(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`{}`, `.foo.`)
	if r.Error == "" {
		t.Fatalf("expected parse error, got none")
	}
	if r.Result != "" {
		t.Fatalf("Result should stay empty on error, got %q", r.Result)
	}
}

func TestRunQueryRuntimeError(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`"hello"`, `.foo`)
	if r.Error == "" {
		t.Fatalf("expected runtime error, got none")
	}
}

func TestExtractUsedKeys(t *testing.T) {
	s := NewJqService()
	keys, err := s.ExtractUsedKeys(`select(.active) | {name: .name, age} | .user.id`)
	if err != nil {
		t.Fatalf("ExtractUsedKeys: %v", err)
	}
	sort.Strings(keys)
	want := []string{"active", "age", "id", "name", "user"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys = %v, want %v", keys, want)
		}
	}
}

func TestExtractUsedKeysParseError(t *testing.T) {
	s := NewJqService()
	if _, err := s.ExtractUsedKeys(`.foo.`); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestExtractUsedKeysIgnoresBracketString(t *testing.T) {
	// `.["weird key"]` のようなブラケット記法は対象外（ドット記法のみを扱う既知の制約）。
	s := NewJqService()
	keys, err := s.ExtractUsedKeys(`.["weird key"]`)
	if err != nil {
		t.Fatalf("ExtractUsedKeys: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("keys = %v, want empty", keys)
	}
}

func TestJqServiceInferSchema(t *testing.T) {
	s := NewJqService()
	nodes, err := s.InferSchema(`{"a":1}`)
	if err != nil {
		t.Fatalf("InferSchema: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Key != "a" {
		t.Fatalf("nodes = %+v", nodes)
	}
}
```

- [ ] **Step 3: テストが失敗することを確認する**

Run: `./scripts/dev-run.sh go test -run 'TestRunQuery|TestExtractUsedKeys|TestJqServiceInferSchema' -v ./...`
Expected: `jq_service.go` が無いためコンパイルエラーで失敗する

- [ ] **Step 4: `jq_service.go` を実装する**

```go
package main

import (
	"encoding/json"
	"strings"

	"github.com/itchyny/gojq"
)

// JqService は jq クエリの実行・解析をまとめる。状態を持たない。
type JqService struct{}

func NewJqService() *JqService {
	return &JqService{}
}

// RunResult は RunQuery の結果。Error が空文字なら成功。
type RunResult struct {
	Result string `json:"result"`
	Error  string `json:"error"`
}

// RunQuery は元のJSONに対してjqクエリを実行する。
// パースエラー・実行時エラーはそれぞれ日本語の接頭辞を付けてErrorに格納し、
// Resultは空のまま返す。呼び出し側はErrorが空でない場合、結果ペインを更新してはいけない。
// 複数の出力（`.a, .b` 等）は改行区切りで連結する。
func (s *JqService) RunQuery(jsonText, query string) RunResult {
	var input any
	if err := json.Unmarshal([]byte(jsonText), &input); err != nil {
		return RunResult{Error: "JSONの解析に失敗しました: " + err.Error()}
	}

	q, err := gojq.Parse(query)
	if err != nil {
		return RunResult{Error: "構文エラー: " + err.Error()}
	}

	iter := q.Run(input)
	var lines []string
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if haltErr, ok := err.(*gojq.HaltError); ok && haltErr.Value() == nil {
				// `halt` によるエラーで値がnilの場合は正常終了として扱う。
				break
			}
			return RunResult{Error: "実行エラー: " + err.Error()}
		}
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return RunResult{Error: "結果の整形に失敗しました: " + err.Error()}
		}
		lines = append(lines, string(b))
	}
	return RunResult{Result: strings.Join(lines, "\n")}
}

// InferSchema はJSONからキーツリー（型付き）を推論する（schema.go参照）。
func (s *JqService) InferSchema(jsonText string) ([]SchemaNode, error) {
	return InferSchema(jsonText)
}

// ExtractUsedKeys はjqクエリのAST（抽象構文木）を解析し、
// `.foo` のようなドットによるフィールドアクセスで参照されているキー名の
// 一覧を返す（重複排除、順不同）。`.["foo"]` のようなブラケット記法は対象外。
func (s *JqService) ExtractUsedKeys(query string) ([]string, error) {
	q, err := gojq.Parse(query)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	walkQueryForKeys(q, set)
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	return keys, nil
}

func walkQueryForKeys(q *gojq.Query, set map[string]bool) {
	if q == nil {
		return
	}
	walkTermForKeys(q.Term, set)
	for _, fd := range q.FuncDefs {
		walkQueryForKeys(fd.Body, set)
	}
	walkQueryForKeys(q.Left, set)
	walkQueryForKeys(q.Right, set)
}

func walkTermForKeys(t *gojq.Term, set map[string]bool) {
	if t == nil {
		return
	}
	collectIndexKey(t.Index, set)
	for _, sfx := range t.SuffixList {
		collectIndexKey(sfx.Index, set)
	}
	if t.Func != nil {
		for _, a := range t.Func.Args {
			walkQueryForKeys(a, set)
		}
	}
	if t.Object != nil {
		for _, kv := range t.Object.KeyVals {
			if kv.Val == nil {
				// `{age}` のような省略記法は `.age` の参照とみなす。
				if kv.Key != "" {
					set[kv.Key] = true
				}
				continue
			}
			walkQueryForKeys(kv.KeyQuery, set)
			walkQueryForKeys(kv.Val, set)
		}
	}
	if t.Array != nil {
		walkQueryForKeys(t.Array.Query, set)
	}
	if t.If != nil {
		walkQueryForKeys(t.If.Cond, set)
		walkQueryForKeys(t.If.Then, set)
		for _, e := range t.If.Elif {
			walkQueryForKeys(e.Cond, set)
			walkQueryForKeys(e.Then, set)
		}
		walkQueryForKeys(t.If.Else, set)
	}
	if t.Try != nil {
		walkQueryForKeys(t.Try.Body, set)
		walkQueryForKeys(t.Try.Catch, set)
	}
	if t.Reduce != nil {
		walkQueryForKeys(t.Reduce.Query, set)
		walkQueryForKeys(t.Reduce.Start, set)
		walkQueryForKeys(t.Reduce.Update, set)
	}
	if t.Foreach != nil {
		walkQueryForKeys(t.Foreach.Query, set)
		walkQueryForKeys(t.Foreach.Start, set)
		walkQueryForKeys(t.Foreach.Update, set)
		walkQueryForKeys(t.Foreach.Extract, set)
	}
	if t.Label != nil {
		walkQueryForKeys(t.Label.Body, set)
	}
	walkQueryForKeys(t.Query, set)
}

func collectIndexKey(idx *gojq.Index, set map[string]bool) {
	if idx == nil {
		return
	}
	if idx.Name != "" {
		set[idx.Name] = true
	}
	walkQueryForKeys(idx.Start, set)
	walkQueryForKeys(idx.End, set)
}
```

- [ ] **Step 5: テストが通ることを確認する**

Run: `./scripts/dev-run.sh go test -v ./...`
Expected: これまでの全タスク分を含め `PASS`（`TestHistoryService*`・`TestInferSchema*`・`TestRunQuery*`・`TestExtractUsedKeys*`・`TestJqServiceInferSchema`）

- [ ] **Step 6: コミットする**

```bash
git add jq_service.go jq_service_test.go go.mod go.sum
git commit -m "$(cat <<'EOF'
feat: jqクエリ実行とAST解析(JqService)を追加する

itchyny/gojqでクエリを実行し、パースエラー・実行時エラーを
日本語メッセージに整形して返す。ASTを解析して参照キー名を
抽出するExtractUsedKeysも追加する（スキーマペインのハイライト用）。
EOF
)"
```

---

### Task 5: バックエンドの結線とWailsバインディング生成

**Files:**
- Modify: `main.go`

**Interfaces:**
- Consumes: `NewJqService() *JqService`（Task 4）、`NewHistoryService(historyDir string) *HistoryService`（Task 2）
- Produces: `frontend/wailsjs/go/main/App.js`・`JqService.js`・`HistoryService.js`（`wails generate module` が生成、以降のフロントエンドタスクがここから import する）

- [ ] **Step 1: `main.go` の `Bind` に `JqService`・`HistoryService` を追加する**

`main.go` の `func main()` を次のように変更する:

```go
func main() {
	app := NewApp()
	jqService := NewJqService()

	homeDir, _ := os.UserHomeDir()
	historyDir := filepath.Join(homeDir, ".jqrepl", "history")
	historyService := NewHistoryService(historyDir)

	err := wails.Run(&options.App{
		Title:  "jqrepl",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets:     assets,
			Middleware: noCacheMiddleware,
		},
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 30, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
			jqService,
			historyService,
		},
		Windows: &windows.Options{
			Theme: windows.Dark,
		},
		Linux: &linux.Options{
			ProgramName: "jqrepl",
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
```

`import` に `"os"` と `"path/filepath"` を追加する（`net/http` の下、Wailsパッケージ群の上に並べる）。

- [ ] **Step 2: ビルドが通ることを確認する**

Run: `./scripts/dev-run.sh go build -o /tmp/jqrepl-check .`
Expected: エラー無く終了する（`embed` のため `frontend/dist` が必要。Task 1 で `wails build` 済みなら存在するはずだが、無ければ `./scripts/dev-run.sh wails build` を先に一度実行する）

- [ ] **Step 3: Wailsバインディングを生成する**

Run: `./scripts/dev-run.sh wails generate module`
Expected: `frontend/wailsjs/go/main/App.js`・`App.d.ts`・`JqService.js`・`JqService.d.ts`・`HistoryService.js`・`HistoryService.d.ts`・`frontend/wailsjs/go/models.ts` が生成される

- [ ] **Step 4: 最終ビルドが通ることを確認する**

Run: `./scripts/dev-run.sh wails build`
Expected: `Built '/app/build/bin/jqrepl' in ...`

- [ ] **Step 5: コミットする**

```bash
git add main.go frontend/wailsjs
git commit -m "$(cat <<'EOF'
feat: JqService/HistoryServiceをWailsにバインドする

フロントエンドから RunQuery/ExtractUsedKeys/InferSchema/
SaveHistory/ListHistory/DeleteHistory/OpenJSONFile を
呼び出せるようにバインディングを生成する。
EOF
)"
```

---

### Task 6: Monaco Editorの土台とJSON/結果ペイン

**Files:**
- Create: `frontend/src/monaco.js`
- Create: `frontend/src/jqCompletions.js`
- Create: `frontend/src/JsonPane.svelte`
- Create: `frontend/src/ResultPane.svelte`
- Modify: `frontend/src/App.svelte`

**Interfaces:**
- Consumes: `OpenJSONFile()`（`frontend/wailsjs/go/main/App`、Task 5で生成）
- Produces: `export default monaco`（`monaco.js`）、`export function registerJqCompletions(monaco, getKeys)`（`jqCompletions.js`、Task 8のQueryPaneが使う）、`JsonPane` props: `{ value, onChange, onOpenFile }`、`ResultPane` props: `{ value }`

- [ ] **Step 1: `frontend/src/monaco.js` を作成する**

```js
// Monaco Editor のスリム構成。エディタ本体 + JSON のシンタックスのみを取り込み、
// 全言語版を避けてバンドルサイズを抑える。jqクエリ用には専用の言語IDだけを登録し、
// 独自トークナイザは持たず補完機能のみ提供する（QueryPane.svelte / jqCompletions.js参照）。
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import 'monaco-editor/esm/vs/basic-languages/json/json.contribution';
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';

self.MonacoEnvironment = {
  getWorker() {
    return new EditorWorker();
  },
};

monaco.languages.register({ id: 'jq' });

export default monaco;
```

- [ ] **Step 2: `frontend/src/jqCompletions.js` を作成する**

```js
// jqクエリペイン用のMonaco補完プロバイダ。
// 現在のJSONから収集したキー名（フラット・重複排除）と、
// 代表的なjq組み込み関数を候補として表示する。
const BUILTIN_FUNCTIONS = [
  'map', 'select', 'keys', 'length', 'sort_by', 'group_by',
  'has', 'to_entries', 'from_entries', 'unique', 'flatten',
  'add', 'min_by', 'max_by', 'reverse', 'empty', 'not', 'type',
];

// getKeys: () => string[] を渡すと、補完要求のたびに最新のキー一覧を反映する。
export function registerJqCompletions(monaco, getKeys) {
  return monaco.languages.registerCompletionItemProvider('jq', {
    triggerCharacters: ['.'],
    provideCompletionItems(model, position) {
      const word = model.getWordUntilPosition(position);
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };
      const keySuggestions = getKeys().map((key) => ({
        label: key,
        kind: monaco.languages.CompletionItemKind.Field,
        insertText: key,
        detail: 'JSONのキー',
        range,
      }));
      const functionSuggestions = BUILTIN_FUNCTIONS.map((name) => ({
        label: name,
        kind: monaco.languages.CompletionItemKind.Function,
        insertText: name,
        detail: 'jq組み込み関数',
        range,
      }));
      return { suggestions: [...keySuggestions, ...functionSuggestions] };
    },
  });
}
```

- [ ] **Step 3: `frontend/src/JsonPane.svelte` を作成する**

```svelte
<script>
  import { onMount, onDestroy } from 'svelte';
  import monaco from './monaco.js';
  import { Button } from 'flowbite-svelte';

  let { value = '', onChange, onOpenFile } = $props();

  let container;
  let editor;
  let applyingExternal = false;

  onMount(() => {
    editor = monaco.editor.create(container, {
      value,
      language: 'json',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 14,
      fontFamily: '"Source Code Pro", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace',
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      tabSize: 2,
    });

    editor.onDidChangeModelContent(() => {
      if (applyingExternal) return;
      onChange?.(editor.getValue());
    });
  });

  onDestroy(() => {
    editor?.dispose();
  });

  // 親（履歴の復元・ファイル読み込み）からの反映。編集中の自己ループを避けるため、
  // 値が実際に異なる場合のみ setValue する。
  $effect(() => {
    const v = value;
    if (editor && editor.getValue() !== v) {
      applyingExternal = true;
      editor.setValue(v);
      applyingExternal = false;
    }
  });
</script>

<div class="pane">
  <div class="pane-header">
    <span>JSON</span>
    <Button size="xs" color="alternative" onclick={() => onOpenFile?.()}>ファイルを開く</Button>
  </div>
  <div class="editor" bind:this={container}></div>
</div>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }
  .pane-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 10px;
    background: #252526;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .editor {
    flex: 1;
    min-height: 0;
  }
</style>
```

- [ ] **Step 4: `frontend/src/ResultPane.svelte` を作成する**

```svelte
<script>
  import { onMount, onDestroy } from 'svelte';
  import monaco from './monaco.js';

  let { value = '' } = $props();

  let container;
  let editor;

  onMount(() => {
    editor = monaco.editor.create(container, {
      value,
      language: 'json',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 14,
      fontFamily: '"Source Code Pro", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace',
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      readOnly: true,
      tabSize: 2,
    });
  });

  onDestroy(() => {
    editor?.dispose();
  });

  $effect(() => {
    const v = value;
    if (editor && editor.getValue() !== v) {
      editor.setValue(v);
    }
  });
</script>

<div class="pane">
  <div class="pane-header">実行結果</div>
  <div class="editor" bind:this={container}></div>
</div>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }
  .pane-header {
    padding: 6px 10px;
    background: #252526;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .editor {
    flex: 1;
    min-height: 0;
  }
</style>
```

- [ ] **Step 5: `App.svelte` を仮のJSON/結果ペイン2枚並びに置き換える（後続タスクでスキーマ・クエリ・履歴を追加していく）**

```svelte
<script>
  import JsonPane from './JsonPane.svelte';
  import ResultPane from './ResultPane.svelte';
  import { OpenJSONFile } from '../wailsjs/go/main/App';

  let jsonText = $state('{\n  "name": "taro",\n  "age": 20\n}');
  let result = $state('');

  async function handleOpenFile() {
    try {
      const content = await OpenJSONFile();
      if (content) jsonText = content;
    } catch (e) {
      // ダイアログのキャンセル・エラーはUI状態を変えずに無視する
    }
  }
</script>

<div class="app-layout">
  <div class="col">
    <JsonPane value={jsonText} onChange={(v) => (jsonText = v)} onOpenFile={handleOpenFile} />
  </div>
  <div class="col">
    <ResultPane value={result} />
  </div>
</div>

<style>
  .app-layout {
    display: flex;
    height: 100vh;
  }
  .col {
    flex: 1;
    min-width: 0;
    border-right: 1px solid #3c3c3c;
  }
</style>
```

- [ ] **Step 6: フロントエンドがビルドできることを確認する**

Run: `./scripts/dev-run.sh sh -c "cd frontend && npm run build"`
Expected: エラー無く `frontend/dist` が生成される

- [ ] **Step 7: コミットする**

```bash
git add frontend/src/monaco.js frontend/src/jqCompletions.js frontend/src/JsonPane.svelte frontend/src/ResultPane.svelte frontend/src/App.svelte
git commit -m "$(cat <<'EOF'
feat: Monaco EditorによるJSON/結果ペインを追加する

スリム構成のMonaco Editorをセットアップし、JSONペイン（編集可・
ファイルを開くボタン付き）と結果ペイン（読み取り専用）を用意する。
jqクエリペイン用の補完プロバイダの土台(jqCompletions.js)も追加する。
EOF
)"
```

---

### Task 7: スキーマペイン

**Files:**
- Create: `frontend/src/SchemaTreeNode.svelte`
- Create: `frontend/src/SchemaPane.svelte`
- Modify: `frontend/src/App.svelte`

**Interfaces:**
- Consumes: `InferSchema(jsonText)`（`frontend/wailsjs/go/main/JqService`）、`SchemaNode` 形状 `{ key, type, children? }`
- Produces: `SchemaPane` props: `{ nodes, usedKeys, errorMessage }`（`usedKeys` は `Set<string>`）— Task 9 で `usedKeys` を実データに接続する

- [ ] **Step 1: `frontend/src/SchemaTreeNode.svelte` を作成する（再帰コンポーネント）**

```svelte
<script>
  import SchemaTreeNode from './SchemaTreeNode.svelte';
  import { Badge } from 'flowbite-svelte';

  let { node, usedKeys } = $props();
  let expanded = $state(true);
  let isUsed = $derived(usedKeys.has(node.key));
  let hasChildren = $derived(!!(node.children && node.children.length > 0));
</script>

<li>
  <div class="row" class:used={isUsed}>
    {#if hasChildren}
      <button class="toggle" onclick={() => (expanded = !expanded)}>{expanded ? '▾' : '▸'}</button>
    {:else}
      <span class="toggle-spacer"></span>
    {/if}
    <span class="key">{node.key}</span>
    <span class="type">: {node.type}</span>
    {#if isUsed}
      <Badge color="blue" class="ml-1">使用中</Badge>
    {/if}
  </div>
  {#if hasChildren && expanded}
    <ul>
      {#each node.children as child (child.key)}
        <SchemaTreeNode node={child} {usedKeys} />
      {/each}
    </ul>
  {/if}
</li>

<style>
  li {
    list-style: none;
  }
  ul {
    margin-left: 16px;
    padding-left: 8px;
    border-left: 1px dashed #3c3c3c;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 2px 4px;
    border-radius: 3px;
  }
  .row.used {
    background: #094771;
  }
  .toggle {
    background: none;
    border: none;
    color: #969696;
    cursor: pointer;
    width: 14px;
    flex-shrink: 0;
    padding: 0;
  }
  .toggle-spacer {
    width: 14px;
    flex-shrink: 0;
    display: inline-block;
  }
  .key {
    color: #9cdcfe;
    font-family: "Source Code Pro", monospace;
  }
  .type {
    color: #6a9955;
    font-family: "Source Code Pro", monospace;
  }
</style>
```

- [ ] **Step 2: `frontend/src/SchemaPane.svelte` を作成する**

```svelte
<script>
  import SchemaTreeNode from './SchemaTreeNode.svelte';

  let { nodes = [], usedKeys = new Set(), errorMessage = '' } = $props();
</script>

<div class="pane">
  <div class="pane-header">スキーマ</div>
  <div class="tree">
    {#if errorMessage}
      <div class="error-text">{errorMessage}</div>
    {:else if nodes.length === 0}
      <div class="empty">キーがありません</div>
    {:else}
      <ul>
        {#each nodes as node (node.key)}
          <SchemaTreeNode {node} {usedKeys} />
        {/each}
      </ul>
    {/if}
  </div>
</div>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background: #1e1e1e;
  }
  .pane-header {
    padding: 6px 10px;
    background: #252526;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .tree {
    flex: 1;
    overflow: auto;
    padding: 8px;
  }
  .empty,
  .error-text {
    color: #6a6a6a;
    font-size: 13px;
    padding: 8px;
  }
  .error-text {
    color: #f48771;
  }
</style>
```

- [ ] **Step 3: `App.svelte` にスキーマペインを組み込む（中央列を上下2段にする）**

`App.svelte` を次のように変更する:

```svelte
<script>
  import JsonPane from './JsonPane.svelte';
  import SchemaPane from './SchemaPane.svelte';
  import ResultPane from './ResultPane.svelte';
  import { OpenJSONFile } from '../wailsjs/go/main/App';
  import { InferSchema } from '../wailsjs/go/main/JqService';

  let jsonText = $state('{\n  "name": "taro",\n  "age": 20\n}');
  let result = $state('');
  let schemaNodes = $state([]);
  let schemaError = $state('');

  let debounceTimer = null;

  async function refreshSchema() {
    try {
      schemaNodes = (await InferSchema(jsonText)) ?? [];
      schemaError = '';
    } catch (e) {
      schemaNodes = [];
      schemaError = 'JSONが不正です';
    }
  }

  $effect(() => {
    const _ = jsonText;
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(refreshSchema, 500);
  });

  async function handleOpenFile() {
    try {
      const content = await OpenJSONFile();
      if (content) jsonText = content;
    } catch (e) {
      // ダイアログのキャンセル・エラーはUI状態を変えずに無視する
    }
  }
</script>

<div class="app-layout">
  <div class="col">
    <div class="pane-slot"><JsonPane value={jsonText} onChange={(v) => (jsonText = v)} onOpenFile={handleOpenFile} /></div>
    <div class="pane-slot"><SchemaPane nodes={schemaNodes} usedKeys={new Set()} errorMessage={schemaError} /></div>
  </div>
  <div class="col">
    <ResultPane value={result} />
  </div>
</div>

<style>
  .app-layout {
    display: flex;
    height: 100vh;
  }
  .col {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    border-right: 1px solid #3c3c3c;
  }
  .pane-slot {
    flex: 1;
    min-height: 0;
    border-bottom: 1px solid #3c3c3c;
  }
  .pane-slot:last-child {
    border-bottom: none;
  }
</style>
```

- [ ] **Step 4: フロントエンドがビルドできることを確認する**

Run: `./scripts/dev-run.sh sh -c "cd frontend && npm run build"`
Expected: エラー無く終了する

- [ ] **Step 5: コミットする**

```bash
git add frontend/src/SchemaTreeNode.svelte frontend/src/SchemaPane.svelte frontend/src/App.svelte
git commit -m "$(cat <<'EOF'
feat: スキーマペインを追加する

JSONペインの内容からキー名+型のツリーを推論して表示する。
JSON変更から500ms後にデバウンスして再推論する。
EOF
)"
```

---

### Task 8: jqクエリペイン

**Files:**
- Create: `frontend/src/QueryPane.svelte`

**Interfaces:**
- Consumes: `registerJqCompletions(monaco, getKeys)`（Task 6・`jqCompletions.js`）
- Produces: `QueryPane` props: `{ value, onChange, errorMessage, getKeys }` — Task 9 で `App.svelte` に組み込む

- [ ] **Step 1: `frontend/src/QueryPane.svelte` を作成する**

```svelte
<script>
  import { onMount, onDestroy } from 'svelte';
  import monaco from './monaco.js';
  import { registerJqCompletions } from './jqCompletions.js';

  let { value = '', onChange, errorMessage = '', getKeys } = $props();

  let container;
  let editor;
  let applyingExternal = false;
  let completionDisposable;

  onMount(() => {
    editor = monaco.editor.create(container, {
      value,
      language: 'jq',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 14,
      fontFamily: '"Source Code Pro", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace',
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      wordWrap: 'on',
      tabSize: 2,
    });

    completionDisposable = registerJqCompletions(monaco, getKeys ?? (() => []));

    editor.onDidChangeModelContent(() => {
      if (applyingExternal) return;
      onChange?.(editor.getValue());
    });
  });

  onDestroy(() => {
    editor?.dispose();
    completionDisposable?.dispose();
  });

  $effect(() => {
    const v = value;
    if (editor && editor.getValue() !== v) {
      applyingExternal = true;
      editor.setValue(v);
      applyingExternal = false;
    }
  });
</script>

<div class="pane">
  <div class="pane-header">jqクエリ</div>
  <div class="editor" bind:this={container}></div>
  {#if errorMessage}
    <div class="error-bar">⚠ {errorMessage}</div>
  {/if}
</div>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }
  .pane-header {
    padding: 6px 10px;
    background: #252526;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .editor {
    flex: 1;
    min-height: 0;
  }
  .error-bar {
    flex-shrink: 0;
    padding: 6px 10px;
    background: #3a1414;
    color: #f48771;
    font-size: 12px;
    white-space: pre-wrap;
    border-top: 1px solid #5a1d1d;
  }
</style>
```

- [ ] **Step 2: フロントエンドがビルドできることを確認する**

Run: `./scripts/dev-run.sh sh -c "cd frontend && npm run build"`
Expected: エラー無く終了する（この時点では `App.svelte` にまだ組み込んでいないため、見た目上の変化はない）

- [ ] **Step 3: コミットする**

```bash
git add frontend/src/QueryPane.svelte
git commit -m "$(cat <<'EOF'
feat: jqクエリペインのコンポーネントを追加する

Monaco Editor + 独自言語ID('jq')による補完・パースエラー表示欄を持つ
クエリ入力ペインを用意する（App.svelteへの組み込みは次のタスク）。
EOF
)"
```

---

### Task 9: App.svelteの本結線（JSON・スキーマ・クエリ・結果の自動実行パイプライン）

**Files:**
- Modify: `frontend/src/App.svelte`

**Interfaces:**
- Consumes: `RunQuery(jsonText, query)`・`ExtractUsedKeys(query)`・`InferSchema(jsonText)`（`frontend/wailsjs/go/main/JqService`）
- Produces: `App.svelte` 内の `evaluate()` 関数（Task 10 の履歴復元がこのパイプラインを再利用する）

- [ ] **Step 1: `App.svelte` を4ペイン構成（JSON・スキーマ・クエリ・結果）に組み替える**

```svelte
<script>
  import JsonPane from './JsonPane.svelte';
  import SchemaPane from './SchemaPane.svelte';
  import QueryPane from './QueryPane.svelte';
  import ResultPane from './ResultPane.svelte';
  import { OpenJSONFile } from '../wailsjs/go/main/App';
  import { RunQuery, ExtractUsedKeys, InferSchema } from '../wailsjs/go/main/JqService';

  const SAMPLE_JSON = '{\n  "name": "taro",\n  "age": 20,\n  "tags": ["admin", "user"],\n  "address": {\n    "city": "tokyo"\n  }\n}';

  let jsonText = $state(SAMPLE_JSON);
  let query = $state('.name');
  let result = $state('');
  let errorMessage = $state('');
  let schemaNodes = $state([]);
  let schemaError = $state('');
  let usedKeys = $state(new Set());

  let flatKeys = $derived(flattenKeys(schemaNodes));

  function flattenKeys(nodes) {
    const set = new Set();
    const walk = (list) => {
      for (const n of list) {
        set.add(n.key);
        if (n.children) walk(n.children);
      }
    };
    walk(nodes);
    return set;
  }

  let debounceTimer = null;

  // evaluate() はJSON/クエリの変更のたびに500ms後に1回だけ呼ばれる（下の$effect参照）。
  // スキーマ再推論・使用キー抽出・クエリ実行をまとめて行う。
  // パースエラー・実行時エラー時は result を更新しない（前回成功時の内容を保持する）。
  async function evaluate() {
    const currentJson = jsonText;
    const currentQuery = query;

    try {
      schemaNodes = (await InferSchema(currentJson)) ?? [];
      schemaError = '';
    } catch (e) {
      schemaNodes = [];
      schemaError = 'JSONが不正です';
    }

    try {
      const keys = await ExtractUsedKeys(currentQuery);
      usedKeys = new Set(keys ?? []);
    } catch (e) {
      usedKeys = new Set();
    }

    const r = await RunQuery(currentJson, currentQuery);
    if (r.error) {
      errorMessage = r.error;
      return;
    }
    errorMessage = '';
    result = r.result;
  }

  $effect(() => {
    const _ = jsonText + ' ' + query;
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(evaluate, 500);
  });

  async function handleOpenFile() {
    try {
      const content = await OpenJSONFile();
      if (content) jsonText = content;
    } catch (e) {
      // ダイアログのキャンセル・エラーはUI状態を変えずに無視する
    }
  }
</script>

<div class="app-layout">
  <div class="col">
    <div class="pane-slot"><JsonPane value={jsonText} onChange={(v) => (jsonText = v)} onOpenFile={handleOpenFile} /></div>
    <div class="pane-slot"><SchemaPane nodes={schemaNodes} {usedKeys} errorMessage={schemaError} /></div>
  </div>
  <div class="col">
    <div class="pane-slot"><QueryPane value={query} onChange={(v) => (query = v)} {errorMessage} getKeys={() => Array.from(flatKeys)} /></div>
    <div class="pane-slot"><ResultPane value={result} /></div>
  </div>
</div>

<style>
  .app-layout {
    display: flex;
    height: 100vh;
  }
  .col {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    border-right: 1px solid #3c3c3c;
  }
  .col:last-child {
    border-right: none;
  }
  .pane-slot {
    flex: 1;
    min-height: 0;
    border-bottom: 1px solid #3c3c3c;
  }
  .pane-slot:last-child {
    border-bottom: none;
  }
</style>
```

- [ ] **Step 2: フロントエンドがビルドできることを確認する**

Run: `./scripts/dev-run.sh sh -c "cd frontend && npm run build"`
Expected: エラー無く終了する

- [ ] **Step 3: `wails build` で結合ビルドを確認する**

Run: `./scripts/dev-run.sh wails build`
Expected: `Built '/app/build/bin/jqrepl' in ...`（この時点でJSON入力→スキーマ推論→クエリ実行→結果表示の一連の流れが完成している。GUIでの目視確認は表示可能な環境で別途行う）

- [ ] **Step 4: コミットする**

```bash
git add frontend/src/App.svelte
git commit -m "$(cat <<'EOF'
feat: JSON・スキーマ・クエリ・結果の自動実行パイプラインを結線する

JSON/クエリの変更から500ms後にスキーマ再推論・使用キー抽出・
クエリ実行をまとめて行うevaluate()を追加する。パースエラー・
実行時エラー時は結果ペインを更新しない。
EOF
)"
```

---

### Task 10: 履歴ペインと履歴の自動保存・復元

**Files:**
- Create: `frontend/src/HistoryPane.svelte`
- Modify: `frontend/src/App.svelte`

**Interfaces:**
- Consumes: `SaveHistory(entry)`・`ListHistory()`（`frontend/wailsjs/go/main/HistoryService`）
- Produces: 完成した5ペイン構成（履歴クリックでJSON・クエリ・結果を一括復元）

- [ ] **Step 1: `frontend/src/HistoryPane.svelte` を作成する**

```svelte
<script>
  let { entries = [], selectedId = null, onSelect } = $props();

  function formatTime(iso) {
    const d = new Date(iso);
    return d.toLocaleString('ja-JP', { hour12: false });
  }
</script>

<div class="pane">
  <div class="pane-header">履歴</div>
  <ul class="list">
    {#each entries as entry (entry.id)}
      <li>
        <button class="item" class:selected={entry.id === selectedId} onclick={() => onSelect?.(entry.id)}>
          <div class="time">{formatTime(entry.createdAt)}</div>
          <div class="query">{entry.query}</div>
        </button>
      </li>
    {:else}
      <li class="empty">まだ実行履歴がありません</li>
    {/each}
  </ul>
</div>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background: #252526;
  }
  .pane-header {
    padding: 6px 10px;
    border-bottom: 1px solid #3c3c3c;
    font-size: 12px;
    color: #969696;
    flex-shrink: 0;
  }
  .list {
    list-style: none;
    overflow-y: auto;
    flex: 1;
  }
  .item {
    width: 100%;
    text-align: left;
    background: none;
    border: none;
    border-bottom: 1px solid #333;
    padding: 8px 10px;
    cursor: pointer;
    color: #cccccc;
  }
  .item:hover {
    background: #2a2d2e;
  }
  .item.selected {
    background: #094771;
  }
  .time {
    font-size: 11px;
    color: #969696;
  }
  .query {
    font-family: "Source Code Pro", monospace;
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .empty {
    padding: 10px;
    color: #6a6a6a;
    font-size: 12px;
  }
</style>
```

- [ ] **Step 2: `App.svelte` に履歴ペインを組み込み、自動保存・復元を結線する**

`App.svelte` の内容全体を次のもので置き換える（Task 9の内容に、履歴のimport・state・`refreshHistory`/`restoreHistory`・`skipNextAutoSave`によるevaluate()の変更・履歴ペインの表示を加えたもの）:

```svelte
<script>
  import HistoryPane from './HistoryPane.svelte';
  import JsonPane from './JsonPane.svelte';
  import SchemaPane from './SchemaPane.svelte';
  import QueryPane from './QueryPane.svelte';
  import ResultPane from './ResultPane.svelte';
  import { onMount } from 'svelte';
  import { OpenJSONFile } from '../wailsjs/go/main/App';
  import { RunQuery, ExtractUsedKeys, InferSchema } from '../wailsjs/go/main/JqService';
  import { SaveHistory, ListHistory } from '../wailsjs/go/main/HistoryService';

  const SAMPLE_JSON = '{\n  "name": "taro",\n  "age": 20,\n  "tags": ["admin", "user"],\n  "address": {\n    "city": "tokyo"\n  }\n}';

  let jsonText = $state(SAMPLE_JSON);
  let query = $state('.name');
  let result = $state('');
  let errorMessage = $state('');
  let schemaNodes = $state([]);
  let schemaError = $state('');
  let usedKeys = $state(new Set());
  let historyEntries = $state([]);
  let selectedHistoryId = $state(null);

  let flatKeys = $derived(flattenKeys(schemaNodes));

  function flattenKeys(nodes) {
    const set = new Set();
    const walk = (list) => {
      for (const n of list) {
        set.add(n.key);
        if (n.children) walk(n.children);
      }
    };
    walk(nodes);
    return set;
  }

  let debounceTimer = null;
  // 履歴復元の直後に実行されるevaluate()では、同じ内容を新規履歴として
  // 保存し直さないようにするためのフラグ。
  let skipNextAutoSave = false;

  async function refreshHistory() {
    try {
      historyEntries = (await ListHistory()) ?? [];
    } catch (e) {
      historyEntries = [];
    }
  }

  async function evaluate() {
    const currentJson = jsonText;
    const currentQuery = query;
    const shouldSkipSave = skipNextAutoSave;
    skipNextAutoSave = false;

    try {
      schemaNodes = (await InferSchema(currentJson)) ?? [];
      schemaError = '';
    } catch (e) {
      schemaNodes = [];
      schemaError = 'JSONが不正です';
    }

    try {
      const keys = await ExtractUsedKeys(currentQuery);
      usedKeys = new Set(keys ?? []);
    } catch (e) {
      usedKeys = new Set();
    }

    const r = await RunQuery(currentJson, currentQuery);
    if (r.error) {
      errorMessage = r.error;
      return;
    }
    errorMessage = '';
    result = r.result;

    if (!shouldSkipSave) {
      try {
        const saved = await SaveHistory({ json: currentJson, query: currentQuery, result: r.result });
        selectedHistoryId = saved.id;
        await refreshHistory();
      } catch (e) {
        // 履歴保存に失敗しても実行結果自体は表示され続ける
      }
    }
  }

  $effect(() => {
    const _ = jsonText + ' ' + query;
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(evaluate, 500);
  });

  function restoreHistory(id) {
    const entry = historyEntries.find((e) => e.id === id);
    if (!entry) return;
    skipNextAutoSave = true;
    jsonText = entry.json;
    query = entry.query;
    result = entry.result;
    errorMessage = '';
    selectedHistoryId = entry.id;
  }

  async function handleOpenFile() {
    try {
      const content = await OpenJSONFile();
      if (content) jsonText = content;
    } catch (e) {
      // ダイアログのキャンセル・エラーはUI状態を変えずに無視する
    }
  }

  onMount(() => {
    refreshHistory();
  });
</script>

<div class="app-layout">
  <div class="history-col">
    <HistoryPane entries={historyEntries} selectedId={selectedHistoryId} onSelect={restoreHistory} />
  </div>
  <div class="col">
    <div class="pane-slot"><JsonPane value={jsonText} onChange={(v) => (jsonText = v)} onOpenFile={handleOpenFile} /></div>
    <div class="pane-slot"><SchemaPane nodes={schemaNodes} {usedKeys} errorMessage={schemaError} /></div>
  </div>
  <div class="col">
    <div class="pane-slot"><QueryPane value={query} onChange={(v) => (query = v)} {errorMessage} getKeys={() => Array.from(flatKeys)} /></div>
    <div class="pane-slot"><ResultPane value={result} /></div>
  </div>
</div>

<style>
  .app-layout {
    display: flex;
    height: 100vh;
  }
  .history-col {
    width: 260px;
    flex-shrink: 0;
    border-right: 1px solid #3c3c3c;
  }
  .col {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    border-right: 1px solid #3c3c3c;
  }
  .col:last-child {
    border-right: none;
  }
  .pane-slot {
    flex: 1;
    min-height: 0;
    border-bottom: 1px solid #3c3c3c;
  }
  .pane-slot:last-child {
    border-bottom: none;
  }
</style>
```

- [ ] **Step 3: フロントエンドがビルドできることを確認する**

Run: `./scripts/dev-run.sh sh -c "cd frontend && npm run build"`
Expected: エラー無く終了する

- [ ] **Step 4: `wails build` で結合ビルドを確認する**

Run: `./scripts/dev-run.sh wails build`
Expected: `Built '/app/build/bin/jqrepl' in ...`

- [ ] **Step 5: コミットする**

```bash
git add frontend/src/HistoryPane.svelte frontend/src/App.svelte
git commit -m "$(cat <<'EOF'
feat: 履歴ペインと自動保存・復元を追加する

クエリ実行が成功するたびに履歴を自動保存し、左ペインに新しい順で
一覧表示する。履歴クリックでJSON・クエリ・結果を一括復元する。
EOF
)"
```

---

### Task 11: CLAUDE.md の作成

**Files:**
- Create: `CLAUDE.md`

**Interfaces:**
- Consumes: なし
- Produces: なし（ドキュメントのみ）

- [ ] **Step 1: `/workspaces/sirusita/CLAUDE.md` を参考に、jqrepl用の `CLAUDE.md` を作成する**

```markdown
# CLAUDE.md — jqrepl

## プロジェクト概要

jqコマンドを繰り返し実行して結果を確認できる、Wails v2 + Svelte 5 で構築した
デスクトップアプリ（アプリ名: jqrepl）。左から履歴・JSON/スキーマ・クエリ/結果の
5ペイン構成。実行履歴は `~/.jqrepl/history/{UUID}.json` に保存される。

参考プロジェクト: `/workspaces/sirusita`（Wails v2 + Svelte 5 + Monaco構成のメモアプリ）。
技術スタック・開発環境・永続化パターンを踏襲している。

## 技術スタック

- **バックエンド:** Go 1.24 + Wails v2.12.0
- **jq実行エンジン:** `github.com/itchyny/gojq` v0.12.19（システムのjqコマンドに依存しない純Go実装）
- **フロントエンド:** Svelte 5（runes）+ Vite 7
- **エディタ:** monaco-editor 0.55.1（JSON表示用 + 独自言語ID `jq` による補完）
- **UIフレームワーク:** flowbite-svelte 1.x + TailwindCSS 4
- **その他Go依存:** `github.com/google/uuid`（履歴ファイル名）

## 開発環境

### Docker必須（コンテナ内で完結）

ホストにGo/Node/Wailsはインストールしない。全てのビルド・テストは
`./scripts/dev-run.sh <command>` 経由でコンテナ（`jqrepl-dev`イメージ）内で実行する。

```bash
# 開発用イメージのビルド（初回 or Dockerfile変更時）
docker build -t jqrepl-dev .

# Goのテスト実行
./scripts/dev-run.sh go test -v ./...

# Wailsビルド（Linux）
./scripts/dev-run.sh wails build

# フロントエンドのみビルド確認
./scripts/dev-run.sh sh -c "cd frontend && npm run build"

# Wailsバインディング再生成（Go側のメソッドシグネチャ変更時）
./scripts/dev-run.sh wails generate module
```

> Podman環境では `docker` を `podman` に読み替えても同じコマンドで動く。
> `scripts/dev-run.sh` は非rootユーザー（ホストと同じUID/GID）でコンテナを実行し、
> `GOPATH`/`GOCACHE`/npmキャッシュを `.devcache/` 配下にバインドマウントする
> （`/go`・`/root` はイメージ内でroot所有のため、そのままでは書き込めない）。

### イメージ内容（Dockerfile）

- Go 1.24, Node.js 22 LTS（NodeSource）, Wails CLI v2.12.0
- libgtk-3-dev, libwebkit2gtk-4.0/4.1-dev（Linuxビルド用）
- gcc-mingw-w64-x86-64, nsis（Windowsクロスコンパイル用、現時点では未使用）

## プロジェクト構成

```
jqrepl/
├── main.go                # Wails エントリポイント、App/JqService/HistoryServiceのBind
├── app.go                 # App構造体（ライフサイクル + OpenJSONFile）
├── jq_service.go          # gojqによるクエリ実行・エラー整形・使用キー抽出(AST解析)
├── schema.go              # JSON値からキーツリー（型付き）を推論
├── history_service.go     # 履歴CRUD（~/.jqrepl/history/*.json）
├── Dockerfile              # 開発用コンテナイメージ定義
├── scripts/dev-run.sh      # コンテナ内でコマンドを実行するヘルパー
├── frontend/
│   ├── svelte.config.js    # vitePreprocess({ script: true })（flowbite-svelte対応、削除不可）
│   ├── vite.config.js      # Vite + svelte + @tailwindcss/vite
│   ├── src/
│   │   ├── main.js             # Svelte マウント
│   │   ├── style.css           # グローバルスタイル（Tailwind + Flowbite）
│   │   ├── monaco.js            # Monaco Editor のスリム構成 + 'jq'言語ID登録
│   │   ├── jqCompletions.js     # jqクエリペイン用のMonaco補完プロバイダ
│   │   ├── App.svelte           # ルート（状態管理 + 5ペインのレイアウト + 自動実行パイプライン）
│   │   ├── HistoryPane.svelte   # 左ペイン：実行履歴一覧
│   │   ├── JsonPane.svelte      # 中央上ペイン：元のJSON（編集可）
│   │   ├── SchemaPane.svelte    # 中央下ペイン：キーツリー（使用中キーをハイライト）
│   │   ├── SchemaTreeNode.svelte# スキーマツリーの再帰描画コンポーネント
│   │   ├── QueryPane.svelte     # 右上ペイン：jqクエリ入力
│   │   └── ResultPane.svelte    # 右下ペイン：実行結果（読み取り専用）
│   └── wailsjs/             # Wails自動生成バインディング（編集不可・`wails generate module`で再生成）
├── build/bin/               # ビルド出力先（jqrepl）
└── wails.json
```

## Go Backend API

### App（app.go）

| メソッド | 説明 |
|---------|------|
| `OpenJSONFile()` | ネイティブのファイル選択ダイアログを開き、選択された`.json`の内容を返す（キャンセル時は空文字） |

### JqService（jq_service.go）

| メソッド | 説明 |
|---------|------|
| `RunQuery(jsonText, query)` | jqクエリを実行し `{result, error}` を返す。エラー時は `error` に日本語メッセージ、`result` は空文字 |
| `ExtractUsedKeys(query)` | クエリのASTを解析し、参照されているキー名一覧を返す（ドット記法のみ対応） |
| `InferSchema(jsonText)` | JSONからキーツリー（型付き）を推論する（`schema.go`のInferSchemaを呼ぶだけ） |

### HistoryService（history_service.go）

| メソッド | 説明 |
|---------|------|
| `SaveHistory(entry)` | 履歴を保存する。IDが空ならUUIDを発行する |
| `ListHistory()` | 履歴一覧を新しい順（CreatedAt降順）で返す |
| `DeleteHistory(id)` | 履歴を削除する（削除UIは未実装、APIのみ） |

## 既知の制約（スコープ外）

- 複数JSONドキュメントのタブ管理は無い
- 履歴の削除UIは無い（バックエンドAPIのみ）
- jqクエリのシンタックスハイライトは無い（独自トークナイザ未実装、補完のみ）
- 補完・使用キーハイライトはフラットなキー名一致であり、jqのパイプ位置に応じた
  文脈依存の絞り込みは行わない
- `.["キー名"]` のようなブラケット記法によるキー参照は、使用キー抽出の対象外
- 中央・右列の上下ペインの境界は固定（ドラッグでのリサイズは左の履歴ペイン幅のみ）

## コーディング規約

- Go: 標準フォーマット（`gofmt`）。全て `package main`（sirusitaに合わせたflat構成）
- Svelte 5（runes）: propsは `$props()`、状態は `$state`/`$derived`/`$effect`、
  親への通知はコールバックprops（`onXxx`）、イベントは `onclick` 等のネイティブ属性
- コミットメッセージ: `feat:` / `fix:` / `chore:` プレフィックス
- 言語: コード内コメントは日本語OK、識別子は英語
```

- [ ] **Step 2: コミットする**

```bash
git add CLAUDE.md
git commit -m "$(cat <<'EOF'
docs: CLAUDE.mdを追加する

sirusitaのCLAUDE.mdを参考に、技術スタック・開発環境・
プロジェクト構成・Go Backend API・既知の制約をまとめる。
EOF
)"
```

**補足（Claude Codeのスキルを作らない理由）:** 元の指示（`tmp/input01.md`）には「必要であれば、スキルなどを作成して」とあるが、
本プロジェクトでは `scripts/dev-run.sh` と本 `CLAUDE.md` が「どうビルド・テスト・実行するか」を
既に完結して文書化しており、これをそのままなぞるだけの `.claude/skills/` を追加しても
実質的な重複にしかならない（YAGNI）。将来、ビルド手順が複数ステップの定型作業に
発展した場合（例: リリース用のクロスビルド＋パッケージング＋GitHub Releaseへの添付を
一括で行う等）は、そのときに専用スキルの追加を検討する。

---

### Task 12: 最終検証

**Files:**
- なし（検証のみ）

**Interfaces:**
- Consumes: これまでの全タスクの成果物
- Produces: なし

- [ ] **Step 1: 全Goテストを実行する**

Run: `./scripts/dev-run.sh go test -v ./...`
Expected: 全テスト `PASS`（`jq_service_test.go`・`schema_test.go`・`history_service_test.go`）

- [ ] **Step 2: フロントエンドビルドを確認する**

Run: `./scripts/dev-run.sh sh -c "cd frontend && npm run build"`
Expected: エラー無く終了する

- [ ] **Step 3: 完全なWailsビルドを確認する**

Run: `./scripts/dev-run.sh wails build`
Expected: `Built '/app/build/bin/jqrepl' in ...` と表示され、`build/bin/jqrepl` が生成される

- [ ] **Step 4: 目視確認についての注記**

このタスクで自動検証できるのは「ビルドが通ること」「Goのビジネスロジックが仕様通り動くこと」まで。
実際に5ペインが意図通りレイアウトされ、JSON貼り付け→スキーマ反映→クエリ入力→
500ms後の結果反映→履歴復元→使用キーのハイライトが画面上で正しく動くかは、
GUIを表示できる環境（実際のデスクトップ）で `./scripts/dev-run.sh wails dev` 相当を
実行するか、`build/bin/jqrepl` を実行して確認する必要がある。この確認はユーザー自身の
環境で行うこと。

- [ ] **Step 5: 最終コミット（必要な場合のみ）**

Run: `git status`
Expected: 未コミットの変更が無い（Task 1〜11で全てコミット済みのはず）。もし残っていれば内容を確認してコミットする。
