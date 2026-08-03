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
