package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestHistoryServiceCRUD(t *testing.T) {
	svc := NewHistoryService(t.TempDir())

	saved, err := svc.SaveHistory(HistoryEntry{JSON: `{"a":1}`, Query: ".a", Result: "1"})
	if err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}
	if saved.ID == "" {
		t.Fatalf("expected generated ID")
	}

	list, err := svc.ListHistory()
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(list) != 1 || list[0].ID != saved.ID {
		t.Fatalf("list = %+v", list)
	}

	if err := svc.DeleteHistory(saved.ID); err != nil {
		t.Fatalf("DeleteHistory: %v", err)
	}
	list, _ = svc.ListHistory()
	if len(list) != 0 {
		t.Fatalf("expected empty list after delete, got %+v", list)
	}
}

func TestHistoryServiceOrder(t *testing.T) {
	svc := NewHistoryService(t.TempDir())
	_, err := svc.SaveHistory(HistoryEntry{JSON: "1", Query: ".", Result: "1"})
	if err != nil {
		t.Fatalf("SaveHistory (first): %v", err)
	}
	second, err := svc.SaveHistory(HistoryEntry{JSON: "2", Query: ".", Result: "2"})
	if err != nil {
		t.Fatalf("SaveHistory (second): %v", err)
	}
	list, err := svc.ListHistory()
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %+v", list)
	}
	if list[0].ID != second.ID {
		t.Errorf("expected newest first: %+v", list)
	}
}

func TestHistoryServiceInvalidID(t *testing.T) {
	svc := NewHistoryService(t.TempDir())
	if err := svc.DeleteHistory("not-a-uuid"); err == nil {
		t.Fatalf("expected error for invalid ID")
	}
}

func TestHistoryServiceSkipsDuplicateOfNewest(t *testing.T) {
	// jqはキーが無ければnullを返すだけでエラーにしないため、対策が無いと
	// `.address.city`と打つ間の`.a`・`.add`・`.addr`…の全てが履歴に残ってしまう。
	// 直近の(JSON, Query)と一致する保存はスキップされ、新規ファイルは作られない。
	svc := NewHistoryService(t.TempDir())
	first, err := svc.SaveHistory(HistoryEntry{JSON: `{"a":1}`, Query: ".a", Result: "1"})
	if err != nil {
		t.Fatalf("SaveHistory (first): %v", err)
	}
	second, err := svc.SaveHistory(HistoryEntry{JSON: `{"a":1}`, Query: ".a", Result: "1"})
	if err != nil {
		t.Fatalf("SaveHistory (duplicate): %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected duplicate save to return the existing entry (%s), got %s", first.ID, second.ID)
	}
	list, err := svc.ListHistory()
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected duplicate save to not create a new file, list = %+v", list)
	}
}

func TestHistoryServiceSavesGenuinelyDifferentContentAfterDuplicate(t *testing.T) {
	svc := NewHistoryService(t.TempDir())
	if _, err := svc.SaveHistory(HistoryEntry{JSON: `{"a":1}`, Query: ".a", Result: "1"}); err != nil {
		t.Fatalf("SaveHistory (first): %v", err)
	}
	if _, err := svc.SaveHistory(HistoryEntry{JSON: `{"a":1}`, Query: ".a", Result: "1"}); err != nil {
		t.Fatalf("SaveHistory (duplicate): %v", err)
	}
	if _, err := svc.SaveHistory(HistoryEntry{JSON: `{"a":1}`, Query: ".b", Result: "null"}); err != nil {
		t.Fatalf("SaveHistory (different query): %v", err)
	}
	list, err := svc.ListHistory()
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected the genuinely different save to add a new entry, list = %+v", list)
	}
}

func TestHistoryServicePrunesOldEntries(t *testing.T) {
	svc := NewHistoryService(t.TempDir())
	svc.MaxEntries = 3
	base := time.Now()
	var ids []string
	for i := 0; i < 5; i++ {
		e, err := svc.SaveHistory(HistoryEntry{
			JSON:      fmt.Sprintf("%d", i),
			Query:     ".",
			Result:    fmt.Sprintf("%d", i),
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatalf("SaveHistory(%d): %v", i, err)
		}
		ids = append(ids, e.ID)
	}
	list, err := svc.ListHistory()
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected pruning to keep MaxEntries=3 entries, got %d: %+v", len(list), list)
	}
	survivors := map[string]bool{}
	for _, e := range list {
		survivors[e.ID] = true
	}
	for _, wantID := range ids[2:] { // 新しい3件（インデックス2,3,4）
		if !survivors[wantID] {
			t.Fatalf("expected newest entry %s to survive pruning, list = %+v", wantID, list)
		}
	}
}

func TestListHistoryFiltersInvalidID(t *testing.T) {
	dir := t.TempDir()
	svc := NewHistoryService(dir)
	good, err := svc.SaveHistory(HistoryEntry{JSON: "1", Query: ".", Result: "1"})
	if err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}
	// idフィールドが空文字の壊れたファイルを直接書き込む
	// （パースはできるがisValidHistoryIDに失敗する = 実運用でも起こり得る破損状態）。
	broken := []byte(`{"id":"","json":"2","query":".","result":"2","createdAt":"2020-01-01T00:00:00Z"}`)
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), broken, 0600); err != nil {
		t.Fatalf("write broken file: %v", err)
	}
	list, err := svc.ListHistory()
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(list) != 1 || list[0].ID != good.ID {
		t.Fatalf("expected the invalid-ID entry to be filtered out, got %+v", list)
	}
}

func TestHistoryServiceOrderTiebreak(t *testing.T) {
	// CreatedAtが同時刻（Windowsのwall clockの粗さ等）でも、SliceStable+IDの
	// タイブレークで順序が決定的になることを確認する。
	svc := NewHistoryService(t.TempDir())
	ts := time.Now()
	a, err := svc.SaveHistory(HistoryEntry{JSON: "a", Query: ".", Result: "a", CreatedAt: ts})
	if err != nil {
		t.Fatalf("SaveHistory (a): %v", err)
	}
	b, err := svc.SaveHistory(HistoryEntry{JSON: "b", Query: ".", Result: "b", CreatedAt: ts})
	if err != nil {
		t.Fatalf("SaveHistory (b): %v", err)
	}
	list, err := svc.ListHistory()
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %+v", list)
	}
	wantFirst := a.ID
	if b.ID > a.ID {
		wantFirst = b.ID
	}
	if list[0].ID != wantFirst {
		t.Fatalf("tie-break order = %+v, want ID-descending first entry %s", list, wantFirst)
	}
}

func TestHistoryServicePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not meaningful on windows")
	}
	// t.TempDir()自体は既に存在するディレクトリなのでMkdirAllのパーミッションが
	// 検証できない（既存ディレクトリに対してはno-op）。NewHistoryServiceに
	// 新規に作らせるため、まだ存在しないサブパスを渡す。
	historyDir := filepath.Join(t.TempDir(), "history")
	svc := NewHistoryService(historyDir)

	dirInfo, err := os.Stat(historyDir)
	if err != nil {
		t.Fatalf("stat historyDir: %v", err)
	}
	if mode := dirInfo.Mode().Perm(); mode != 0700 {
		t.Fatalf("historyDir mode = %o, want 0700", mode)
	}

	entry, err := svc.SaveHistory(HistoryEntry{JSON: "1", Query: ".", Result: "1"})
	if err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}
	fileInfo, err := os.Stat(filepath.Join(historyDir, entry.ID+".json"))
	if err != nil {
		t.Fatalf("stat entry file: %v", err)
	}
	if mode := fileInfo.Mode().Perm(); mode != 0600 {
		t.Fatalf("entry file mode = %o, want 0600", mode)
	}
}

func TestHistoryServiceLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	svc := NewHistoryService(dir)
	if _, err := svc.SaveHistory(HistoryEntry{JSON: "1", Query: ".", Result: "1"}); err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			t.Fatalf("writeFileAtomic left a stray file behind: %s", e.Name())
		}
	}
}
