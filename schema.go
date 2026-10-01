package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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

// jsonSchemaNode は InferJSONSchema が出力するJSON Schemaの1ノード。
// フィールドの宣言順がそのまま出力順になる（type を先頭にする）。
// Properties は map なので encoding/json がキーをアルファベット順に並べる。
// omitzero は nil のときだけ省略する（omitempty と違い、空オブジェクト `{}` の
// `"properties": {}` は残る）。
type jsonSchemaNode struct {
	Type       string                     `json:"type"`
	Properties map[string]*jsonSchemaNode `json:"properties,omitzero"`
	Items      *jsonSchemaNode            `json:"items,omitzero"`
}

// InferJSONSchema はJSONテキストから、整形済みのJSON Schema文字列を推論する。
// 推論ルールは InferSchema（スキーマペイン）と揃えている: 配列は先頭要素を
// 要素の代表とし（空配列は items を出力しない）、数値は全て number とする。
// 出力するのは type / properties / items のみで、required 等は推論しない。
// SchemaNode のツリーからではなく元のJSONから直接作るのは、SchemaNode が
// 「ルートが配列かどうか」を保持していないため。
func InferJSONSchema(jsonText string) (string, error) {
	var value any
	if err := json.Unmarshal([]byte(jsonText), &value); err != nil {
		return "", fmt.Errorf("JSONの解析に失敗しました: %w", err)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// 既定のHTMLエスケープ（< → < 等）はキー名を読みにくくするだけなので切る。
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(inferJSONSchemaNode(value)); err != nil {
		return "", fmt.Errorf("スキーマの生成に失敗しました: %w", err)
	}
	// Encoder は末尾に改行を付けるので落とす。
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

func inferJSONSchemaNode(value any) *jsonSchemaNode {
	switch v := value.(type) {
	case map[string]any:
		props := make(map[string]*jsonSchemaNode, len(v))
		for k, child := range v {
			props[k] = inferJSONSchemaNode(child)
		}
		return &jsonSchemaNode{Type: "object", Properties: props}
	case []any:
		node := &jsonSchemaNode{Type: "array"}
		if len(v) > 0 {
			node.Items = inferJSONSchemaNode(v[0])
		}
		return node
	case nil:
		return &jsonSchemaNode{Type: "null"}
	case bool:
		return &jsonSchemaNode{Type: "boolean"}
	case float64:
		return &jsonSchemaNode{Type: "number"}
	default:
		return &jsonSchemaNode{Type: "string"}
	}
}
