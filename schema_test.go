package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

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

// assertJSONSchema は InferJSONSchema の出力を、キー順や空白に依らず意味的に比較する。
func assertJSONSchema(t *testing.T, input, want string) {
	t.Helper()
	got, err := InferJSONSchema(input)
	if err != nil {
		t.Fatalf("InferJSONSchema: %v", err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal([]byte(got), &gotValue); err != nil {
		t.Fatalf("出力が有効なJSONではない: %v\n%s", err, got)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("want が有効なJSONではない: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("schema =\n%s\nwant\n%s", got, want)
	}
}

func TestInferJSONSchemaObject(t *testing.T) {
	assertJSONSchema(t,
		`{"name":"taro","age":20,"active":true,"note":null}`,
		`{"type":"object","properties":{
			"name":{"type":"string"},
			"age":{"type":"number"},
			"active":{"type":"boolean"},
			"note":{"type":"null"}
		}}`)
}

func TestInferJSONSchemaNestedObject(t *testing.T) {
	assertJSONSchema(t,
		`{"address":{"city":"tokyo"}}`,
		`{"type":"object","properties":{
			"address":{"type":"object","properties":{"city":{"type":"string"}}}
		}}`)
}

func TestInferJSONSchemaArrayUsesFirstElement(t *testing.T) {
	// スキーマペイン（InferSchema）と同じく、先頭要素を配列要素の代表とする。
	assertJSONSchema(t,
		`{"tags":["a","b"],"items":[{"id":1},{"id":2,"extra":true}]}`,
		`{"type":"object","properties":{
			"tags":{"type":"array","items":{"type":"string"}},
			"items":{"type":"array","items":{"type":"object","properties":{"id":{"type":"number"}}}}
		}}`)
}

func TestInferJSONSchemaEmptyArrayHasNoItems(t *testing.T) {
	// 要素が無ければ型を推論できないので items 自体を出力しない。
	assertJSONSchema(t,
		`{"items":[]}`,
		`{"type":"object","properties":{"items":{"type":"array"}}}`)
}

func TestInferJSONSchemaEmptyObjectKeepsProperties(t *testing.T) {
	assertJSONSchema(t, `{}`, `{"type":"object","properties":{}}`)
}

func TestInferJSONSchemaRootArray(t *testing.T) {
	// キーツリー（SchemaNode）と違い、ルートが配列であることを失わない。
	assertJSONSchema(t,
		`[{"id":1},{"id":2}]`,
		`{"type":"array","items":{"type":"object","properties":{"id":{"type":"number"}}}}`)
}

func TestInferJSONSchemaNestedArray(t *testing.T) {
	assertJSONSchema(t,
		`[[1,2],[3]]`,
		`{"type":"array","items":{"type":"array","items":{"type":"number"}}}`)
}

func TestInferJSONSchemaRootScalar(t *testing.T) {
	assertJSONSchema(t, `"hello"`, `{"type":"string"}`)
}

func TestInferJSONSchemaFormatting(t *testing.T) {
	// 貼り付けて読むための出力なので、2スペースインデントで、type を先頭に、
	// properties のキーはアルファベット順に並べる。
	got, err := InferJSONSchema(`{"b":1,"a":"x"}`)
	if err != nil {
		t.Fatalf("InferJSONSchema: %v", err)
	}
	want := `{
  "type": "object",
  "properties": {
    "a": {
      "type": "string"
    },
    "b": {
      "type": "number"
    }
  }
}`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestInferJSONSchemaDoesNotEscapeHTML(t *testing.T) {
	// encoding/json 既定のHTMLエスケープ（< → <）はコピー結果を読みにくくするだけなので行わない。
	got, err := InferJSONSchema(`{"a<b>&c":1}`)
	if err != nil {
		t.Fatalf("InferJSONSchema: %v", err)
	}
	if !strings.Contains(got, `"a<b>&c"`) {
		t.Errorf("キーがHTMLエスケープされている:\n%s", got)
	}
}

func TestInferJSONSchemaInvalidJSON(t *testing.T) {
	if _, err := InferJSONSchema(`{invalid`); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}
