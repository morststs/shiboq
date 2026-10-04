//go:build !js

// Web版（GOOS=js）ではファイルシステム・Wailsランタイムを使えないため除外する（wasm_main.go参照）。

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestSavedServiceCRUD(t *testing.T) {
	svc := NewSavedService(t.TempDir())

	saved, err := svc.SaveItem(SavedItem{JSON: `{"a":1}`, Query: ".a", Result: "1"})
	if err != nil {
		t.Fatalf("SaveItem: %v", err)
	}
	if saved.ID == "" {
		t.Fatalf("expected generated ID")
	}

	list, err := svc.ListSaved()
	if err != nil {
		t.Fatalf("ListSaved: %v", err)
	}
	if len(list) != 1 || list[0].ID != saved.ID {
		t.Fatalf("list = %+v", list)
	}

	if err := svc.DeleteSaved(saved.ID); err != nil {
		t.Fatalf("DeleteSaved: %v", err)
	}
	list, _ = svc.ListSaved()
	if len(list) != 0 {
		t.Fatalf("expected empty list after delete, got %+v", list)
	}
}

func TestSavedServiceOrder(t *testing.T) {
	svc := NewSavedService(t.TempDir())
	_, err := svc.SaveItem(SavedItem{JSON: "1", Query: ".", Result: "1"})
	if err != nil {
		t.Fatalf("SaveItem (first): %v", err)
	}
	second, err := svc.SaveItem(SavedItem{JSON: "2", Query: ".", Result: "2"})
	if err != nil {
		t.Fatalf("SaveItem (second): %v", err)
	}
	list, err := svc.ListSaved()
	if err != nil {
		t.Fatalf("ListSaved: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %+v", list)
	}
	if list[0].ID != second.ID {
		t.Errorf("expected newest first: %+v", list)
	}
}

func TestSavedServiceInvalidID(t *testing.T) {
	svc := NewSavedService(t.TempDir())
	if err := svc.DeleteSaved("not-a-uuid"); err == nil {
		t.Fatalf("expected error for invalid ID")
	}
}

// 明示的な保存に変えたので、同じ内容を続けて保存しても両方とも残る。
// 自動保存時代は直近と同一の(JSON, Query)をスキップしていたが、
// ユーザーが押した保存を黙って無視するのは正しくない。
func TestSavedServiceSavesDuplicatesExplicitly(t *testing.T) {
	svc := NewSavedService(t.TempDir())
	first, err := svc.SaveItem(SavedItem{JSON: `{"a":1}`, Query: ".a", Result: "1"})
	if err != nil {
		t.Fatalf("SaveItem (first): %v", err)
	}
	second, err := svc.SaveItem(SavedItem{JSON: `{"a":1}`, Query: ".a", Result: "1"})
	if err != nil {
		t.Fatalf("SaveItem (same content): %v", err)
	}
	if second.ID == first.ID {
		t.Fatalf("expected a distinct new item, got the same ID %s", first.ID)
	}
	list, err := svc.ListSaved()
	if err != nil {
		t.Fatalf("ListSaved: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected both explicit saves to persist, list = %+v", list)
	}
}

// 自動間引きを廃止したので、多数保存しても1件も消えない。
// ユーザーが意図して保存したものを黙って削除しないため。
func TestSavedServiceDoesNotPrune(t *testing.T) {
	svc := NewSavedService(t.TempDir())
	base := time.Now()
	const n = 250 // かつての上限200を明確に超える件数
	for i := 0; i < n; i++ {
		_, err := svc.SaveItem(SavedItem{
			JSON:      fmt.Sprintf("%d", i),
			Query:     ".",
			Result:    fmt.Sprintf("%d", i),
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatalf("SaveItem(%d): %v", i, err)
		}
	}
	list, err := svc.ListSaved()
	if err != nil {
		t.Fatalf("ListSaved: %v", err)
	}
	if len(list) != n {
		t.Fatalf("expected all %d items to survive (no pruning), got %d", n, len(list))
	}
}

func TestSavedServiceStoresName(t *testing.T) {
	svc := NewSavedService(t.TempDir())
	saved, err := svc.SaveItem(SavedItem{Name: "本番のフィルタ", JSON: "1", Query: ".", Result: "1"})
	if err != nil {
		t.Fatalf("SaveItem: %v", err)
	}
	if saved.Name != "本番のフィルタ" {
		t.Fatalf("returned Name = %q", saved.Name)
	}
	list, err := svc.ListSaved()
	if err != nil {
		t.Fatalf("ListSaved: %v", err)
	}
	if len(list) != 1 || list[0].Name != "本番のフィルタ" {
		t.Fatalf("persisted Name not read back: %+v", list)
	}
}

// Nameは空を許容する（フロントエンド側がQueryを代わりに表示する）。
func TestSavedServiceAllowsEmptyName(t *testing.T) {
	svc := NewSavedService(t.TempDir())
	if _, err := svc.SaveItem(SavedItem{JSON: "1", Query: ".foo", Result: "1"}); err != nil {
		t.Fatalf("SaveItem: %v", err)
	}
	list, err := svc.ListSaved()
	if err != nil {
		t.Fatalf("ListSaved: %v", err)
	}
	if len(list) != 1 || list[0].Name != "" {
		t.Fatalf("expected empty Name, got %+v", list)
	}
}

// 旧「履歴」時代のディレクトリを保存先へ移行する。
func TestMigrateLegacyHistoryDir(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "history")
	saved := filepath.Join(root, "saved")

	// 旧ディレクトリに、name フィールドを持たない当時の形式のファイルを置く。
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	const id = "11111111-1111-4111-8111-111111111111"
	old := []byte(`{"id":"` + id + `","json":"1","query":".","result":"1","createdAt":"2020-01-01T00:00:00Z"}`)
	if err := os.WriteFile(filepath.Join(legacy, id+".json"), old, 0600); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	migrated, err := migrateLegacyHistoryDir(legacy, saved)
	if err != nil {
		t.Fatalf("migrateLegacyHistoryDir: %v", err)
	}
	if !migrated {
		t.Fatalf("expected migration to happen")
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy dir should be gone after migration")
	}

	// 移行後、name が無い旧データも読める（ゼロ値の空文字になる）。
	svc := NewSavedService(saved)
	list, err := svc.ListSaved()
	if err != nil {
		t.Fatalf("ListSaved: %v", err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Fatalf("migrated item not readable: %+v", list)
	}
	if list[0].Name != "" {
		t.Errorf("legacy item Name = %q, want empty", list[0].Name)
	}
}

// 保存先が既にある場合は移行しない（現行データを壊さない）。
func TestMigrateLegacyHistoryDirSkipsWhenSavedExists(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "history")
	saved := filepath.Join(root, "saved")
	for _, d := range []string{legacy, saved} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}
	migrated, err := migrateLegacyHistoryDir(legacy, saved)
	if err != nil {
		t.Fatalf("migrateLegacyHistoryDir: %v", err)
	}
	if migrated {
		t.Fatalf("expected no migration when saved dir already exists")
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Errorf("legacy dir should be left untouched: %v", err)
	}
}

// 旧ディレクトリが無ければ何もしない（新規インストール時）。
func TestMigrateLegacyHistoryDirNoLegacy(t *testing.T) {
	root := t.TempDir()
	migrated, err := migrateLegacyHistoryDir(filepath.Join(root, "history"), filepath.Join(root, "saved"))
	if err != nil {
		t.Fatalf("migrateLegacyHistoryDir: %v", err)
	}
	if migrated {
		t.Fatalf("expected no migration when there is no legacy dir")
	}
}

func TestListSavedFiltersInvalidID(t *testing.T) {
	dir := t.TempDir()
	svc := NewSavedService(dir)
	good, err := svc.SaveItem(SavedItem{JSON: "1", Query: ".", Result: "1"})
	if err != nil {
		t.Fatalf("SaveItem: %v", err)
	}
	// idフィールドが空文字の壊れたファイルを直接書き込む
	// （パースはできるがisValidSavedIDに失敗する = 実運用でも起こり得る破損状態）。
	broken := []byte(`{"id":"","json":"2","query":".","result":"2","createdAt":"2020-01-01T00:00:00Z"}`)
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), broken, 0600); err != nil {
		t.Fatalf("write broken file: %v", err)
	}
	list, err := svc.ListSaved()
	if err != nil {
		t.Fatalf("ListSaved: %v", err)
	}
	if len(list) != 1 || list[0].ID != good.ID {
		t.Fatalf("expected the invalid-ID item to be filtered out, got %+v", list)
	}
}

func TestSavedServiceOrderTiebreak(t *testing.T) {
	// CreatedAtが同時刻（Windowsのwall clockの粗さ等）でも、SliceStable+IDの
	// タイブレークで順序が決定的になることを確認する。
	svc := NewSavedService(t.TempDir())
	ts := time.Now()
	a, err := svc.SaveItem(SavedItem{JSON: "a", Query: ".", Result: "a", CreatedAt: ts})
	if err != nil {
		t.Fatalf("SaveItem (a): %v", err)
	}
	b, err := svc.SaveItem(SavedItem{JSON: "b", Query: ".", Result: "b", CreatedAt: ts})
	if err != nil {
		t.Fatalf("SaveItem (b): %v", err)
	}
	list, err := svc.ListSaved()
	if err != nil {
		t.Fatalf("ListSaved: %v", err)
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

func TestSavedServicePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not meaningful on windows")
	}
	// t.TempDir()自体は既に存在するディレクトリなのでMkdirAllのパーミッションが
	// 検証できない（既存ディレクトリに対してはno-op）。NewSavedServiceに
	// 新規に作らせるため、まだ存在しないサブパスを渡す。
	savedDir := filepath.Join(t.TempDir(), "saved")
	svc := NewSavedService(savedDir)

	dirInfo, err := os.Stat(savedDir)
	if err != nil {
		t.Fatalf("stat savedDir: %v", err)
	}
	if mode := dirInfo.Mode().Perm(); mode != 0700 {
		t.Fatalf("savedDir mode = %o, want 0700", mode)
	}

	item, err := svc.SaveItem(SavedItem{JSON: "1", Query: ".", Result: "1"})
	if err != nil {
		t.Fatalf("SaveItem: %v", err)
	}
	fileInfo, err := os.Stat(filepath.Join(savedDir, item.ID+".json"))
	if err != nil {
		t.Fatalf("stat item file: %v", err)
	}
	if mode := fileInfo.Mode().Perm(); mode != 0600 {
		t.Fatalf("item file mode = %o, want 0600", mode)
	}
}

func TestSavedServiceLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	svc := NewSavedService(dir)
	if _, err := svc.SaveItem(SavedItem{JSON: "1", Query: ".", Result: "1"}); err != nil {
		t.Fatalf("SaveItem: %v", err)
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
