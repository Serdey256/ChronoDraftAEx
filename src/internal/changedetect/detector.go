// Package changedetect 负责对话变动实时监控
// 通过对比文件系统快照，生成结构化的改动记录
package changedetect

import (
	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Detector 文件变动检测器
type Detector struct {
	projectRoot string
	ignoreList  []string
}

// FileSnapshot 记录文件在某个时间点的元信息
type FileSnapshot struct {
	Path    string `json:"path"`
	ModTime int64  `json:"mod_time"`
	Size    int64  `json:"size"`
	Hash    string `json:"hash"`
}

// NewDetector 创建一个变动检测器
func NewDetector(projectRoot string) *Detector {
	return &Detector{
		projectRoot: projectRoot,
		ignoreList:  []string{".git", ".chronodraft", "node_modules", "vendor"},
	}
}

// SetIgnoreList 自定义忽略目录列表
func (d *Detector) SetIgnoreList(ignores []string) {
	d.ignoreList = ignores
}

// shouldIgnore 判断路径是否在忽略列表中
func (d *Detector) shouldIgnore(path string) bool {
	for _, ig := range d.ignoreList {
		if strings.Contains(path, ig) {
			return true
		}
	}
	return false
}

// ScanSnapshot 扫描项目目录并生成当前文件快照
func (d *Detector) ScanSnapshot() (map[string]FileSnapshot, error) {
	snapshot := make(map[string]FileSnapshot)
	err := filepath.Walk(d.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(d.projectRoot, path)
		if d.shouldIgnore(rel) || info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		snapshot[rel] = FileSnapshot{
			Path:    rel,
			ModTime: info.ModTime().Unix(),
			Size:    info.Size(),
			Hash:    hash,
		}
		return nil
	})
	return snapshot, err
}

// DetectChanges 对比两个快照，生成变更记录
func (d *Detector) DetectChanges(oldSnap, newSnap map[string]FileSnapshot, sessionID string) *models.ChangeRecord {
	record := &models.ChangeRecord{
		ID:        utils.GenerateID(),
		Timestamp: time.Now(),
		SessionID: sessionID,
		Changes:   []models.FileChange{},
	}

	// 检测新增和修改
	for path, newInfo := range newSnap {
		oldInfo, exists := oldSnap[path]
		if !exists {
			record.Changes = append(record.Changes, models.FileChange{
				Path:       path,
				ChangeType: "add",
			})
		} else if oldInfo.Hash != newInfo.Hash {
			record.Changes = append(record.Changes, models.FileChange{
				Path:       path,
				ChangeType: "modify",
			})
		}
	}

	// 检测删除
	for path := range oldSnap {
		if _, exists := newSnap[path]; !exists {
			record.Changes = append(record.Changes, models.FileChange{
				Path:       path,
				ChangeType: "delete",
			})
		}
	}

	return record
}

// SaveSnapshot 将快照持久化到本地（JSON 格式）
func (d *Detector) SaveSnapshot(snap map[string]FileSnapshot, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "snapshot.json"), data, 0644)
}

// LoadSnapshot 从本地加载快照
func (d *Detector) LoadSnapshot(dir string) (map[string]FileSnapshot, error) {
	data, err := os.ReadFile(filepath.Join(dir, "snapshot.json"))
	if err != nil {
		return nil, err
	}
	var snap map[string]FileSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return snap, nil
}
