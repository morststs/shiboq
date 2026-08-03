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
	historyDir := filepath.Join(homeDir, ".shiboq", "history")
	historyService := NewHistoryService(historyDir)

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
			historyService,
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
