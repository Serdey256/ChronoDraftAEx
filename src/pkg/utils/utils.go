// Package utils 提供 ChronoDraftAEx 的通用工具函数
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
)

// GenerateID 生成一个随机的 16 字节十六进制字符串作为唯一 ID
func GenerateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// EnsureDir 确保指定目录存在，不存在则创建
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// GetProjectRoot 获取当前工作目录作为项目根目录
func GetProjectRoot() (string, error) {
	return os.Getwd()
}

// GetDataDir 获取应用数据存储目录（项目根目录下的 .chronodraft 目录）
func GetDataDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".chronodraft")
}
