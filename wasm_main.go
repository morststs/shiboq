//go:build js && wasm

package main

import "syscall/js"

// Web版（GitHub Pages）のエントリポイント。GOOS=js GOARCH=wasm でビルドし、
// フロントエンドのWeb Worker（frontend/src/backend/jq.worker.js）内で動かす。
//
// グローバル関数 shiboqCall(method, argsJSON, callback) を登録する。
// 結果は callback(resultJSON, errorMessage) で非同期に返す。
// js.FuncOf のコールバック内でブロックするとデッドロックするため
// （syscall/jsの制約）、処理は必ず別goroutineで行う。
func main() {
	jq := NewJqService()
	js.Global().Set("shiboqCall", js.FuncOf(func(this js.Value, args []js.Value) any {
		method := args[0].String()
		argsJSON := args[1].String()
		callback := args[2]
		go func() {
			result, err := callJqService(jq, method, argsJSON)
			if err != nil {
				callback.Invoke(js.Null(), err.Error())
				return
			}
			callback.Invoke(result, js.Null())
		}()
		return nil
	}))
	select {}
}
