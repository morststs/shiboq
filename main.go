//go:build !js

// Web版（GOOS=js）ではファイルシステム・Wailsランタイムを使えないため除外する（wasm_main.go参照）。

package main

import (
	"embed"
	"net/http"
	"os"
	"path/filepath"

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
	jqService := NewJqService()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		println("エラー: ホームディレクトリを取得できませんでした:", err.Error())
		os.Exit(1)
	}
	savedDir := filepath.Join(homeDir, ".shiboq", "saved")
	// 旧「履歴」時代の ~/.shiboq/history を保存先へ移行する（保存先が未作成の
	// 場合のみ）。NewSavedService より前に呼ぶ必要がある（先に呼ぶと保存先が
	// 作られてしまい、移行が「済み」と判定されてしまう）。
	// 失敗しても起動は続ける（保存先が空で始まるだけで致命的ではない）。
	legacyDir := filepath.Join(homeDir, ".shiboq", "history")
	if _, err := migrateLegacyHistoryDir(legacyDir, savedDir); err != nil {
		println("警告: 旧履歴データの移行に失敗しました:", err.Error())
	}
	savedService := NewSavedService(savedDir)

	err = wails.Run(&options.App{
		Title:  "shiboq",
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
			savedService,
		},
		Windows: &windows.Options{
			Theme: windows.Dark,
		},
		Linux: &linux.Options{
			ProgramName: "shiboq",
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
