package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadJSONFileWithLimitOK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "small.json")
	if err := os.WriteFile(path, []byte(`{"a":1}`), 0600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	content, err := readJSONFileWithLimit(path, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != `{"a":1}` {
		t.Fatalf("content = %q, want %q", content, `{"a":1}`)
	}
}

func TestReadJSONFileWithLimitTooLarge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("a", 2048)), 0600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	_, err := readJSONFileWithLimit(path, 1024)
	if err == nil {
		t.Fatalf("expected an error for an oversized file")
	}
	if !strings.Contains(err.Error(), "大きすぎます") {
		t.Fatalf("error = %q, want a message mentioning 大きすぎます", err.Error())
	}
}

func TestReadJSONFileWithLimitMissingFile(t *testing.T) {
	_, err := readJSONFileWithLimit(filepath.Join(t.TempDir(), "does-not-exist.json"), 1024)
	if err == nil {
		t.Fatalf("expected an error for a missing file")
	}
}
