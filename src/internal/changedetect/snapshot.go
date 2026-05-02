package changedetect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SnapshotsDir 返回项目快照存储目录
func SnapshotsDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".chronodraft")
}

// SaveLastSnap 将文件快照持久化到磁盘
func SaveLastSnap(projectRoot string, snap map[string]FileSnapshot) error {
	dir := SnapshotsDir(projectRoot)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建快照目录失败: %w", err)
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化快照失败: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "last_snap.json"), data, 0644)
}

// LoadLastSnap 从磁盘加载文件快照
func LoadLastSnap(projectRoot string) (map[string]FileSnapshot, error) {
	path := filepath.Join(SnapshotsDir(projectRoot), "last_snap.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err // caller handles os.IsNotExist
	}
	var snap map[string]FileSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("解析快照文件失败: %w", err)
	}
	return snap, nil
}
