//go:build !js

// Web版（GOOS=js）ではファイルシステム・Wailsランタイムを使えないため除外する（wasm_main.go参照）。

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

// SavedItem はユーザーが明示的に保存した1件。ID未指定でSaveItemに渡すとUUIDを発行する。
// Name は空を許容し、その場合はフロントエンド側が代わりに Query を表示に使う。
type SavedItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	JSON      string    `json:"json"`
	Query     string    `json:"query"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"createdAt"`
}

// SavedService は保存項目を `{savedDir}/{UUID}.json` として1件1ファイルで永続化する。
//
// かつては実行のたびに自動保存する「履歴」で、際限なく増えるため件数上限による
// 自動間引きと、直近と同一内容ならスキップする重複排除を持っていた。現在は
// ユーザーが明示的に保存する方式に変えたため、どちらも行わない。意図して
// 保存したものを黙って消したり、押した保存を無視したりしないため。
type SavedService struct {
	savedDir string
}

// savedDirPerm/savedFilePerm: 保存項目には貼り付けたJSON（トークンやPIIを
// 含み得る）がそのまま入るため、他ユーザーから読めないようにする。
const (
	savedDirPerm  os.FileMode = 0700
	savedFilePerm os.FileMode = 0600
)

func NewSavedService(savedDir string) *SavedService {
	os.MkdirAll(savedDir, savedDirPerm)
	return &SavedService{savedDir: savedDir}
}

// migrateLegacyHistoryDir は旧「履歴」時代のディレクトリを保存先へ移行する。
// legacyDir が存在し、かつ savedDir がまだ存在しない場合に限りリネームする。
// 既に savedDir がある場合は何もしない（利用者の現行データを壊さないため）。
//
// NewSavedService より前に呼ぶ必要がある。NewSavedService が savedDir を
// 作ってしまうと、この関数は「移行済み」とみなして何もしなくなる。
// 戻り値は移行を実際に行ったかどうか。失敗しても呼び出し側は起動を続ける。
func migrateLegacyHistoryDir(legacyDir, savedDir string) (bool, error) {
	if _, err := os.Stat(savedDir); err == nil {
		return false, nil
	}
	if _, err := os.Stat(legacyDir); err != nil {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(savedDir), savedDirPerm); err != nil {
		return false, err
	}
	if err := os.Rename(legacyDir, savedDir); err != nil {
		return false, err
	}
	return true, nil
}

func isValidSavedID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// SaveItem は保存項目を書き込む。IDが空文字ならUUIDを新規発行し、
// CreatedAtが未設定なら現在時刻を設定して返す。
//
// 内容が直近の項目と同一でも必ず新規保存する。明示的な操作は押したとおりに
// 実行されるべきで、黙って無視すると「保存したのに増えない」ことになるため。
func (s *SavedService) SaveItem(item SavedItem) (SavedItem, error) {
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.New().String()
	} else if !isValidSavedID(item.ID) {
		return SavedItem{}, fmt.Errorf("invalid saved item ID: %s", item.ID)
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return SavedItem{}, fmt.Errorf("failed to marshal saved item: %w", err)
	}
	path := filepath.Join(s.savedDir, item.ID+".json")
	if err := writeFileAtomic(path, data, savedFilePerm); err != nil {
		return SavedItem{}, fmt.Errorf("failed to write saved item: %w", err)
	}
	return item, nil
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

// ListSaved は保存項目を新しい順（CreatedAt降順、同時刻はID降順でタイブレーク）で返す。
// IDが空文字・不正な形式（UUID以外）の項目は、Svelte側の`{#each items as i (i.id)}`
// キーが重複してSvelteが例外を投げるのを防ぐため除外する。
func (s *SavedService) ListSaved() ([]SavedItem, error) {
	entries, err := os.ReadDir(s.savedDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read saved dir: %w", err)
	}
	var result []SavedItem
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.savedDir, e.Name()))
		if err != nil {
			continue
		}
		var item SavedItem
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}
		if !isValidSavedID(item.ID) {
			continue
		}
		result = append(result, item)
	}
	// sort.Sliceではなくsort.SliceStableを使い、CreatedAtが同時刻の場合はIDで
	// タイブレークする。Windowsのwall clockは粗く、同時刻の項目が並んだ
	// 際にsort.Sliceの非安定性がリスト表示順をぐらつかせ得るため。
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID > result[j].ID
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *SavedService) DeleteSaved(id string) error {
	if !isValidSavedID(id) {
		return fmt.Errorf("invalid saved item ID: %s", id)
	}
	path := filepath.Join(s.savedDir, id+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("saved item not found: %s", id)
	}
	return os.Remove(path)
}
