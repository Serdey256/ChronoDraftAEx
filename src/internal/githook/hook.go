// Package githook 提供 Git hook 的安装与卸载功能
// 用于在 git post-commit 时自动捕获 commit 信息到 ChronoDraftAEx 知识库
package githook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hookScriptTemplate 是 post-commit hook 脚本模板
// BINARY_PATH 占位符会在 InstallHook 时替换为实际路径
const hookScriptTemplate = `#!/bin/bash
HASH=$(git rev-parse HEAD)
MSG=$(git log -1 --pretty=%B | head -1)
AUTHOR=$(git log -1 --pretty=%an)
FILES=$(git diff-tree --no-commit-id --name-only -r HEAD | tr '\n' ',' | sed 's/,$//')
STATS=$(git diff-tree --no-commit-id --numstat -r HEAD)
INS=$(echo "$STATS" | awk '{s+=$1} END {print s+0}')
DELS=$(echo "$STATS" | awk '{s+=$2} END {print s+0}')
BINARY_PATH capture-commit --hash "$HASH" --message "$MSG" --author "$AUTHOR" --files "$FILES" --ins "$INS" --dels "$DELS"
`

const hookFileName = "post-commit"
const hookBackupSuffix = ".bak"

// InstallHook 在项目根目录的 .git/hooks/ 下安装 post-commit hook
// projectRoot: 项目根目录（应包含 .git 子目录）
// binaryPath: chronodraft-mcp 可执行文件的完整路径
//
// 如果 post-commit hook 已存在，会先备份为 post-commit.bak
func InstallHook(projectRoot, binaryPath string) error {
	hooksDir := filepath.Join(projectRoot, ".git", "hooks")

	// 确保 hooks 目录存在
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("创建 hooks 目录失败: %w", err)
	}

	hookPath := filepath.Join(hooksDir, hookFileName)
	backupPath := hookPath + hookBackupSuffix

	// 检查是否已安装（内容相同的跳过）
	if existing, err := os.ReadFile(hookPath); err == nil {
		expected := generateHookScript(binaryPath)
		if string(existing) == expected {
			return nil // 已安装且内容一致，无需操作
		}

		// 备份现有 hook
		if err := os.Rename(hookPath, backupPath); err != nil {
			return fmt.Errorf("备份已有 hook 失败: %w", err)
		}
	}

	// 写入新 hook 脚本
	script := generateHookScript(binaryPath)
	if err := os.WriteFile(hookPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("写入 hook 脚本失败: %w", err)
	}

	return nil
}

// UninstallHook 删除 .git/hooks/post-commit hook
// projectRoot: 项目根目录（应包含 .git 子目录）
func UninstallHook(projectRoot string) error {
	hookPath := filepath.Join(projectRoot, ".git", "hooks", hookFileName)

	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return nil // 不存在则无需删除
	}

	if err := os.Remove(hookPath); err != nil {
		return fmt.Errorf("删除 hook 失败: %w", err)
	}

	return nil
}

// generateHookScript 生成最终 hook 脚本内容，替换 BINARY_PATH 占位符
func generateHookScript(binaryPath string) string {
	return strings.ReplaceAll(hookScriptTemplate, "BINARY_PATH", binaryPath)
}
