package main

import (
	"encoding/json"
	"fmt"
)

// callJqService はWeb版（wasm_main.go）からの呼び出しを JqService のメソッドへ
// 振り分ける。引数 argsJSON はメソッド引数のJSON配列、戻り値は結果のJSON文字列。
// Wailsのバインディングと同じく、メソッドが返したエラーは error として返す
// （フロントエンドではPromiseのrejectになる）。
//
// syscall/js に依存しない形でここに切り出しているのは、通常の go test で
// 検証できるようにするため（wasm_main.go はGOOS=jsでしかビルドされない）。
func callJqService(s *JqService, method, argsJSON string) (string, error) {
	var args []json.RawMessage
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("引数の解析に失敗しました: %w", err)
	}
	str := func(i int) (string, error) {
		var v string
		if i >= len(args) {
			return "", fmt.Errorf("%s: 引数が不足しています", method)
		}
		if err := json.Unmarshal(args[i], &v); err != nil {
			return "", fmt.Errorf("%s: 引数%dが文字列ではありません", method, i)
		}
		return v, nil
	}

	var out any
	switch method {
	case "RunQuery":
		jsonText, err := str(0)
		if err != nil {
			return "", err
		}
		query, err := str(1)
		if err != nil {
			return "", err
		}
		var raw bool
		if len(args) > 2 {
			if err := json.Unmarshal(args[2], &raw); err != nil {
				return "", fmt.Errorf("RunQuery: 引数2が真偽値ではありません")
			}
		}
		out = s.RunQuery(jsonText, query, raw)
	case "ExtractUsedKeys":
		query, err := str(0)
		if err != nil {
			return "", err
		}
		keys, err := s.ExtractUsedKeys(query)
		if err != nil {
			return "", err
		}
		out = keys
	case "InferSchema":
		jsonText, err := str(0)
		if err != nil {
			return "", err
		}
		nodes, err := s.InferSchema(jsonText)
		if err != nil {
			return "", err
		}
		out = nodes
	case "SchemaJSON":
		jsonText, err := str(0)
		if err != nil {
			return "", err
		}
		schema, err := s.SchemaJSON(jsonText)
		if err != nil {
			return "", err
		}
		out = schema
	case "EngineInfo":
		out = s.EngineInfo()
	default:
		return "", fmt.Errorf("未知のメソッドです: %s", method)
	}

	data, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
