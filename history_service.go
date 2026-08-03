package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// HistoryEntry は1回のjq実行の記録。ID未指定でSaveHistoryに渡すとUUIDを発行する。
type HistoryEntry struct {
	ID        string    `json:"id"`
	JSON      string    `json:"json"`
	Query     string    `json:"query"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"createdAt"`
}

// HistoryService は履歴を `{historyDir}/{UUID}.json` として1件1ファイルで永続化する。
type HistoryService struct {
	historyDir string
	// MaxEntries はSaveHistory後に保持する最大件数。0以下ならdefaultMaxHistoryEntries。
	// テストから小さい値に差し替えられるようフィールドにしている。
	MaxEntries int
}

// defaultMaxHistoryEntries は保存後にpruneOldEntriesが保持するエントリ数の上限。
// 履歴は自動保存かつ削除UIが無いため、上限を設けないとディスクを際限なく消費する。
const defaultMaxHistoryEntries = 200

// historyDirPerm/historyFilePerm: 履歴には貼り付けたJSON（トークンやPIIを
// 含み得る）がそのまま保存されるため、他ユーザーから読めないようにする。
const (
	historyDirPerm  os.FileMode = 0700
	historyFilePerm os.FileMode = 0600
)

func NewHistoryService(historyDir string) *HistoryService {
	os.MkdirAll(historyDir, historyDirPerm)
	return &HistoryService{historyDir: historyDir}
}

func isValidHistoryID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// SaveHistory は履歴を保存する。IDが空文字ならUUIDを新規発行し、
// CreatedAtが未設定なら現在時刻を設定して返す。
//
// 直近（最新）のエントリと (JSON, Query) の組が完全に一致する場合は
// 新規ファイルを書き込まずスキップし、既存のエントリをそのまま返す。
// jqはキーが無ければ`null`を返しエラーにしないため、対策が無いと
// `.address.city`と打つ間に`.a`・`.add`・`.addr`…の全てが「成功」扱いで
// 保存されてしまう。
//
// 保存後はpruneOldEntriesで古いエントリを間引き、ディスク使用量に上限を設ける。
func (s *HistoryService) SaveHistory(entry HistoryEntry) (HistoryEntry, error) {
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = uuid.New().String()
	} else if !isValidHistoryID(entry.ID) {
		return HistoryEntry{}, fmt.Errorf("invalid history ID: %s", entry.ID)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	if existing, err := s.ListHistory(); err == nil && len(existing) > 0 {
		newest := existing[0]
		if newest.JSON == entry.JSON && newest.Query == entry.Query {
			return newest, nil
		}
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return HistoryEntry{}, fmt.Errorf("failed to marshal history entry: %w", err)
	}
	path := filepath.Join(s.historyDir, entry.ID+".json")
	if err := writeFileAtomic(path, data, historyFilePerm); err != nil {
		return HistoryEntry{}, fmt.Errorf("failed to write history entry: %w", err)
	}

	s.pruneOldEntries()

	return entry, nil
}

// pruneOldEntries はMaxEntries（未設定ならdefaultMaxHistoryEntries）を
// 超えた分の古いエントリファイルを削除する。個々の削除失敗は無視する
// （ベストエフォート。次回保存時にまた試みられる）。
func (s *HistoryService) pruneOldEntries() {
	max := s.MaxEntries
	if max <= 0 {
		max = defaultMaxHistoryEntries
	}
	entries, err := s.ListHistory()
	if err != nil || len(entries) <= max {
		return
	}
	for _, e := range entries[max:] {
		_ = os.Remove(filepath.Join(s.historyDir, e.ID+".json"))
	}
}

// writeFileAtomic は同じディレクトリに一時ファイルを作成して書き込み、
// os.Renameでpathへ置き換える。os.WriteFileの直接上書きは、書き込み途中で
// プロセスが落ちた場合に壊れたファイルを残す（非アトミック）ため使わない。
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // Renameが成功していれば対象は既に無いのでno-op

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// ListHistory は履歴を新しい順（CreatedAt降順、同時刻はID降順でタイブレーク）で返す。
// IDが空文字・不正な形式（UUID以外）のエントリは、Svelte側の`{#each entries as e (e.id)}`
// キーが重複してSvelteが例外を投げるのを防ぐため除外する。
func (s *HistoryService) ListHistory() ([]HistoryEntry, error) {
	entries, err := os.ReadDir(s.historyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read history dir: %w", err)
	}
	var result []HistoryEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.historyDir, e.Name()))
		if err != nil {
			continue
		}
		var entry HistoryEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		if !isValidHistoryID(entry.ID) {
			continue
		}
		result = append(result, entry)
	}
	// sort.Sliceではなくsort.SliceStableを使い、CreatedAtが同時刻の場合はIDで
	// タイブレークする。Windowsのwall clockは粗く、同時刻のエントリが並んだ
	// 際にsort.Sliceの非安定性がリスト表示順をぐらつかせ得るため。
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID > result[j].ID
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *HistoryService) DeleteHistory(id string) error {
	if !isValidHistoryID(id) {
		return fmt.Errorf("invalid history ID: %s", id)
	}
	path := filepath.Join(s.historyDir, id+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("history not found: %s", id)
	}
	return os.Remove(path)
}
