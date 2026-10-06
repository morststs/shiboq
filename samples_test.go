//go:build !js

package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// frontend/src/samples.json（学習用サンプル）の整合性を検査する。
// サンプルはフロントエンドのデータだが、実際に内蔵エンジンで実行して
// エラーにならないことをここで保証する（gojq更新で壊れたら検出する）。

type sampleFile struct {
	Datasets map[string]json.RawMessage `json:"datasets"`
	Chapters []struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Samples []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Data    string `json:"data"`
			Query   string `json:"query"`
			Raw     bool   `json:"raw"`
			Explain string `json:"explain"`
			Try     string `json:"try"`
		} `json:"samples"`
	} `json:"chapters"`
}

func loadSamples(t *testing.T) sampleFile {
	t.Helper()
	data, err := os.ReadFile("frontend/src/samples.json")
	if err != nil {
		t.Fatalf("read samples.json: %v", err)
	}
	var f sampleFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("parse samples.json: %v", err)
	}
	return f
}

func TestSamplesRunWithoutError(t *testing.T) {
	f := loadSamples(t)
	s := NewJqService()
	count := 0
	for _, ch := range f.Chapters {
		for _, sm := range ch.Samples {
			ds, ok := f.Datasets[sm.Data]
			if !ok {
				t.Errorf("%s/%s: データセット %q がありません", ch.ID, sm.ID, sm.Data)
				continue
			}
			r := s.RunQuery(string(ds), sm.Query, sm.Raw)
			if r.Error != "" {
				t.Errorf("%s/%s: %s\nquery: %s", ch.ID, sm.ID, r.Error, sm.Query)
				continue
			}
			if strings.TrimSpace(r.Result) == "" {
				t.Errorf("%s/%s: 結果が空です\nquery: %s", ch.ID, sm.ID, sm.Query)
			}
			if testing.Verbose() {
				t.Logf("[%s/%s] %s\n%s", ch.ID, sm.ID, sm.Query, r.Result)
			}
			count++
		}
	}
	if count == 0 {
		t.Fatal("サンプルが1件もありません")
	}
}

func TestSamplesHaveUniqueIDsAndText(t *testing.T) {
	f := loadSamples(t)
	seen := map[string]bool{}
	for _, ch := range f.Chapters {
		if ch.ID == "" || ch.Title == "" || len(ch.Samples) == 0 {
			t.Errorf("章 %q: id・title・サンプルが必要です", ch.ID)
		}
		for _, sm := range ch.Samples {
			key := ch.ID + "/" + sm.ID
			if sm.ID == "" || seen[key] {
				t.Errorf("サンプルIDが空か重複しています: %q", key)
			}
			seen[key] = true
			if sm.Title == "" || sm.Query == "" || sm.Explain == "" {
				t.Errorf("%s: title・query・explainが必要です", key)
			}
		}
	}
}
