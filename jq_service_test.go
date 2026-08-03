package main

import (
	"math"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestRunQueryOK(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`{"name":"taro","age":20}`, `.name`)
	if r.Error != "" {
		t.Fatalf("unexpected error: %s", r.Error)
	}
	if r.Result != `"taro"` {
		t.Fatalf("Result = %q, want %q", r.Result, `"taro"`)
	}
}

func TestRunQueryMultipleOutputs(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`{"a":1,"b":2}`, `.a, .b`)
	if r.Error != "" {
		t.Fatalf("unexpected error: %s", r.Error)
	}
	if r.Result != "1\n2" {
		t.Fatalf("Result = %q, want %q", r.Result, "1\n2")
	}
}

func TestRunQueryInvalidJSON(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`{invalid`, `.`)
	if r.Error == "" {
		t.Fatalf("expected error for invalid JSON")
	}
	if r.Result != "" {
		t.Fatalf("Result should stay empty on error, got %q", r.Result)
	}
	if r.ErrorKind != ErrorKindJSON {
		t.Fatalf("ErrorKind = %q, want %q", r.ErrorKind, ErrorKindJSON)
	}
}

func TestRunQueryParseError(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`{}`, `.foo.`)
	if r.Error == "" {
		t.Fatalf("expected parse error, got none")
	}
	if r.Result != "" {
		t.Fatalf("Result should stay empty on error, got %q", r.Result)
	}
	if r.ErrorKind != ErrorKindSyntax {
		t.Fatalf("ErrorKind = %q, want %q", r.ErrorKind, ErrorKindSyntax)
	}
}

func TestRunQueryCompileError(t *testing.T) {
	// 構文としては正しいが未定義の関数呼び出しはgojq.Compileの段階で失敗する。
	// パースエラー（構文エラー）とは区別し、「コンパイルエラー: 」を接頭辞にする。
	s := NewJqService()
	r := s.RunQuery(`{}`, `this_function_does_not_exist`)
	if r.Error == "" {
		t.Fatalf("expected compile error, got none")
	}
	if !strings.HasPrefix(r.Error, "コンパイルエラー: ") {
		t.Fatalf("Error = %q, want prefix %q", r.Error, "コンパイルエラー: ")
	}
	if r.ErrorKind != ErrorKindCompile {
		t.Fatalf("ErrorKind = %q, want %q", r.ErrorKind, ErrorKindCompile)
	}
}

func TestRunQueryRuntimeError(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`"hello"`, `.foo`)
	if r.Error == "" {
		t.Fatalf("expected runtime error, got none")
	}
	if r.ErrorKind != ErrorKindRuntime {
		t.Fatalf("ErrorKind = %q, want %q", r.ErrorKind, ErrorKindRuntime)
	}
}

func TestRunQueryTimeout(t *testing.T) {
	// `repeat(1)` は無限に値を生成し続ける。デフォルトの5秒を待たされないよう、
	// Timeoutを短く差し替えてテストを高速に保つ。MaxOutputsは十分大きくして
	// 出力件数の上限ではなくタイムアウトの方が先に発火するようにする。
	s := &JqService{Timeout: 20 * time.Millisecond, MaxOutputs: 1_000_000_000, MaxOutputBytes: 1_000_000_000}
	start := time.Now()
	r := s.RunQuery(`1`, `repeat(1)`)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("RunQuery took too long: %s (want well under the 5s default, timeout was 20ms)", elapsed)
	}
	if r.Error == "" {
		t.Fatalf("expected timeout error, got none")
	}
	if r.Result != "" {
		t.Fatalf("Result should stay empty on timeout, got %q", r.Result)
	}
	if r.ErrorKind != ErrorKindRuntime {
		t.Fatalf("ErrorKind = %q, want %q", r.ErrorKind, ErrorKindRuntime)
	}
	if !strings.Contains(r.Error, "タイムアウト") {
		t.Fatalf("Error = %q, want a message mentioning タイムアウト", r.Error)
	}
}

func TestRunQueryOutputCountCap(t *testing.T) {
	// `repeat(1)` は無限に出力するので、MaxOutputsを超えたらタイムアウトを
	// 待たずに即座に打ち切られるはずである。
	s := &JqService{Timeout: 5 * time.Second, MaxOutputs: 5, MaxOutputBytes: 1_000_000_000}
	start := time.Now()
	r := s.RunQuery(`1`, `repeat(1)`)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("RunQuery took too long: %s (output cap should trigger almost instantly)", elapsed)
	}
	if r.Error == "" {
		t.Fatalf("expected output-count-cap error, got none")
	}
	if r.Result != "" {
		t.Fatalf("Result should stay empty when the cap is hit, got %q", r.Result)
	}
	if r.ErrorKind != ErrorKindRuntime {
		t.Fatalf("ErrorKind = %q, want %q", r.ErrorKind, ErrorKindRuntime)
	}
}

func TestRunQueryOutputByteCap(t *testing.T) {
	// 1回あたりの出力は大きいが件数は少ない場合でも、累積バイト数の上限で
	// 打ち切られることを確認する。
	s := &JqService{Timeout: 5 * time.Second, MaxOutputs: 1_000_000_000, MaxOutputBytes: 100}
	start := time.Now()
	r := s.RunQuery(`1`, `repeat("x" * 1000)`)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("RunQuery took too long: %s (byte cap should trigger almost instantly)", elapsed)
	}
	if r.Error == "" {
		t.Fatalf("expected output-byte-cap error, got none")
	}
	if r.Result != "" {
		t.Fatalf("Result should stay empty when the cap is hit, got %q", r.Result)
	}
	if r.ErrorKind != ErrorKindRuntime {
		t.Fatalf("ErrorKind = %q, want %q", r.ErrorKind, ErrorKindRuntime)
	}
}

func TestRunQueryNormalizesNaN(t *testing.T) {
	// jqの`nan`組み込み関数はfloat64のNaNを生成する。encoding/jsonはNaNを
	// 直接Marshalできないため、本物のjqと同じく`null`として出力する必要がある。
	s := NewJqService()
	r := s.RunQuery(`null`, `nan`)
	if r.Error != "" {
		t.Fatalf("unexpected error: %s", r.Error)
	}
	if r.Result != "null" {
		t.Fatalf("Result = %q, want %q", r.Result, "null")
	}
}

func TestRunQueryNormalizesInfinite(t *testing.T) {
	s := NewJqService()
	r := s.RunQuery(`null`, `infinite, -infinite`)
	if r.Error != "" {
		t.Fatalf("unexpected error: %s", r.Error)
	}
	if r.Result != "null\nnull" {
		t.Fatalf("Result = %q, want %q", r.Result, "null\nnull")
	}
}

func TestNormalizeNonFiniteFloatsNested(t *testing.T) {
	in := map[string]any{
		"a": math.NaN(),
		"b": []any{math.Inf(1), math.Inf(-1), 1.5},
	}
	out := normalizeNonFiniteFloats(in).(map[string]any)
	if out["a"] != nil {
		t.Fatalf(`out["a"] = %v, want nil`, out["a"])
	}
	arr := out["b"].([]any)
	if arr[0] != nil || arr[1] != nil {
		t.Fatalf("arr = %v, want [nil, nil, 1.5]", arr)
	}
	if arr[2] != 1.5 {
		t.Fatalf("arr[2] = %v, want 1.5", arr[2])
	}
}

func TestExtractUsedKeys(t *testing.T) {
	s := NewJqService()
	keys, err := s.ExtractUsedKeys(`select(.active) | {name: .name, age} | .user.id`)
	if err != nil {
		t.Fatalf("ExtractUsedKeys: %v", err)
	}
	sort.Strings(keys)
	want := []string{"active", "age", "id", "name", "user"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys = %v, want %v", keys, want)
		}
	}
}

func TestExtractUsedKeysParseError(t *testing.T) {
	s := NewJqService()
	if _, err := s.ExtractUsedKeys(`.foo.`); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestExtractUsedKeysIgnoresBracketString(t *testing.T) {
	// `.["weird key"]` のようなブラケット記法は対象外（ドット記法のみを扱う既知の制約）。
	s := NewJqService()
	keys, err := s.ExtractUsedKeys(`.["weird key"]`)
	if err != nil {
		t.Fatalf("ExtractUsedKeys: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("keys = %v, want empty", keys)
	}
}

func TestExtractUsedKeysStringInterpolation(t *testing.T) {
	s := NewJqService()
	keys, err := s.ExtractUsedKeys(`"\(.name) is \(.age)"`)
	if err != nil {
		t.Fatalf("ExtractUsedKeys: %v", err)
	}
	sort.Strings(keys)
	want := []string{"age", "name"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys = %v, want %v", keys, want)
		}
	}
}

func TestExtractUsedKeysUnaryOperator(t *testing.T) {
	s := NewJqService()
	keys, err := s.ExtractUsedKeys(`-.foo`)
	if err != nil {
		t.Fatalf("ExtractUsedKeys: %v", err)
	}
	want := []string{"foo"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	if keys[0] != want[0] {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
}

func TestExtractUsedKeysDestructuringSimple(t *testing.T) {
	s := NewJqService()
	keys, err := s.ExtractUsedKeys(`. as {a: $x} | $x`)
	if err != nil {
		t.Fatalf("ExtractUsedKeys: %v", err)
	}
	want := []string{"a"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	if keys[0] != want[0] {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
}

func TestExtractUsedKeysDestructuringArray(t *testing.T) {
	s := NewJqService()
	keys, err := s.ExtractUsedKeys(`. as [{b: $y}] | $y`)
	if err != nil {
		t.Fatalf("ExtractUsedKeys: %v", err)
	}
	want := []string{"b"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	if keys[0] != want[0] {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
}

func TestExtractUsedKeysReduceWithPattern(t *testing.T) {
	s := NewJqService()
	keys, err := s.ExtractUsedKeys(`reduce .items[] as {id: $i} (0; . + $i)`)
	if err != nil {
		t.Fatalf("ExtractUsedKeys: %v", err)
	}
	sort.Strings(keys)
	want := []string{"id", "items"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys = %v, want %v", keys, want)
		}
	}
}

func TestJqServiceInferSchema(t *testing.T) {
	s := NewJqService()
	nodes, err := s.InferSchema(`{"a":1}`)
	if err != nil {
		t.Fatalf("InferSchema: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Key != "a" {
		t.Fatalf("nodes = %+v", nodes)
	}
}
