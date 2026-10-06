package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCallJqServiceRunQuery(t *testing.T) {
	out, err := callJqService(NewJqService(), "RunQuery", `["{\"a\":\"x\"}", ".a", true]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var r RunResult
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Result != "x" || r.Error != "" {
		t.Fatalf("got %+v, want raw result x", r)
	}
}

func TestCallJqServiceRunQueryWithoutRaw(t *testing.T) {
	out, err := callJqService(NewJqService(), "RunQuery", `["{\"a\":\"x\"}", ".a"]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"result":"\"x\""`) {
		t.Fatalf("out = %s, want quoted result", out)
	}
}

func TestCallJqServiceRunQueryErrorIsResultNotError(t *testing.T) {
	// RunQueryのエラーはWails版と同じくRunResult.Errorに入る（rejectしない）。
	out, err := callJqService(NewJqService(), "RunQuery", `["{", ".a", false]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var r RunResult
	json.Unmarshal([]byte(out), &r)
	if r.ErrorKind != ErrorKindJSON {
		t.Fatalf("ErrorKind = %q, want json", r.ErrorKind)
	}
}

func TestCallJqServiceExtractUsedKeys(t *testing.T) {
	out, err := callJqService(NewJqService(), "ExtractUsedKeys", `[".foo"]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != `["foo"]` {
		t.Fatalf("out = %s", out)
	}
}

func TestCallJqServiceInferSchemaError(t *testing.T) {
	_, err := callJqService(NewJqService(), "InferSchema", `["{"]`)
	if err == nil || !strings.Contains(err.Error(), "JSONの解析に失敗しました") {
		t.Fatalf("err = %v, want JSON parse error", err)
	}
}

func TestCallJqServiceSchemaJSONReturnsJSONString(t *testing.T) {
	out, err := callJqService(NewJqService(), "SchemaJSON", `["{\"a\":1}"]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var s string
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("out is not a JSON string: %s", out)
	}
	if !strings.Contains(s, `"type": "object"`) {
		t.Fatalf("schema = %s", s)
	}
}

func TestCallJqServiceUnknownMethod(t *testing.T) {
	if _, err := callJqService(NewJqService(), "Nope", `[]`); err == nil {
		t.Fatal("expected error for unknown method")
	}
}

func TestCallJqServiceMissingArgs(t *testing.T) {
	if _, err := callJqService(NewJqService(), "RunQuery", `["{}"]`); err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestCallJqServiceEngineInfo(t *testing.T) {
	out, err := callJqService(NewJqService(), "EngineInfo", `[]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var info EngineInfo
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if info.Engine != "gojq" || info.EngineVersion != gojqVersion {
		t.Fatalf("info = %+v", info)
	}
}
