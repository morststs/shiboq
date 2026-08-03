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
