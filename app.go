package main

import (
	"context"
	"fmt"
	"os"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// maxOpenJSONFileSize はOpenJSONFileが読み込むファイルサイズの上限。
// 上限が無いと、誤って数百MBのファイルを選択した際にUIが応答不能になる。
const maxOpenJSONFileSize int64 = 20 * 1024 * 1024 // 20MB

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
	return readJSONFileWithLimit(path, maxOpenJSONFileSize)
}

// readJSONFileWithLimit はpathのサイズがmaxSizeを超えないか確認してから
// 内容を読み込む。ダイアログ操作から切り離してテスト可能にするために
// OpenJSONFileから分離してある。
func readJSONFileWithLimit(path string, maxSize int64) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Size() > maxSize {
		return "", fmt.Errorf(
			"ファイルサイズが大きすぎます（%.1fMB、上限%dMB）。より小さいファイルを選択してください。",
			float64(info.Size())/(1024*1024), maxSize/(1024*1024),
		)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
