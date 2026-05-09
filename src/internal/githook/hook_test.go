package githook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo 在临时目录中创建 .git/hooks 目录结构
func setupTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("创建 .git/hooks 失败: %v", err)
	}
	return root
}

func TestInstallHook_CreatesHookFile(t *testing.T) {
	root := setupTestRepo(t)
	binaryPath := "/usr/local/bin/chronodraft-mcp"

	if err := InstallHook(root, binaryPath); err != nil {
		t.Fatalf("InstallHook 失败: %v", err)
	}

	hookPath := filepath.Join(root, ".git", "hooks", "post-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Fatal("post-commit hook 文件未被创建")
	}

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("读取 hook 文件失败: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, binaryPath) {
		t.Errorf("hook 内容应包含 binaryPath %q，实际为: %s", binaryPath, content)
	}

	if !strings.Contains(content, "capture-commit") {
		t.Errorf("hook 内容应包含 capture-commit 命令")
	}
}

func TestInstallHook_ReplacesBinaryPath(t *testing.T) {
	root := setupTestRepo(t)
	binaryPath := "C:\\tools\\chronodraft-mcp.exe"

	if err := InstallHook(root, binaryPath); err != nil {
		t.Fatalf("InstallHook 失败: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "post-commit"))
	if err != nil {
		t.Fatalf("读取 hook 文件失败: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "BINARY_PATH") {
		t.Error("hook 内容中仍包含未替换的 BINARY_PATH 占位符")
	}
}

func TestInstallHook_BacksUpExisting(t *testing.T) {
	root := setupTestRepo(t)
	binaryPath := "/usr/local/bin/chronodraft-mcp"
	hookPath := filepath.Join(root, ".git", "hooks", "post-commit")

	// 创建现存的 hook 文件
	oldContent := "#!/bin/bash\necho old hook\n"
	if err := os.WriteFile(hookPath, []byte(oldContent), 0755); err != nil {
		t.Fatalf("写入旧 hook 失败: %v", err)
	}

	if err := InstallHook(root, binaryPath); err != nil {
		t.Fatalf("InstallHook 失败: %v", err)
	}

	// 验证备份文件存在
	backupPath := hookPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("备份文件 post-commit.bak 不存在")
	}

	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("读取备份文件失败: %v", err)
	}
	if string(backupData) != oldContent {
		t.Errorf("备份内容与原始内容不匹配\n期望: %q\n实际: %q", oldContent, string(backupData))
	}

	// 验证新 hook 已写入
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("读取新 hook 失败: %v", err)
	}
	if !strings.Contains(string(data), binaryPath) {
		t.Error("新 hook 应包含 binaryPath")
	}
}

func TestInstallHook_SkipsIfSameContent(t *testing.T) {
	root := setupTestRepo(t)
	binaryPath := "/usr/local/bin/chronodraft-mcp"
	hookPath := filepath.Join(root, ".git", "hooks", "post-commit")

	// 先安装一次
	if err := InstallHook(root, binaryPath); err != nil {
		t.Fatalf("第一次 InstallHook 失败: %v", err)
	}

	// 第二次安装不应该有任何变化
	if err := InstallHook(root, binaryPath); err != nil {
		t.Fatalf("第二次 InstallHook 失败: %v", err)
	}

	// 验证没有生成备份文件
	backupPath := hookPath + ".bak"
	if _, err := os.Stat(backupPath); err == nil {
		t.Fatal("内容相同时不应创建备份文件")
	}
}

func TestUninstallHook_RemovesHook(t *testing.T) {
	root := setupTestRepo(t)
	binaryPath := "/usr/local/bin/chronodraft-mcp"
	hookPath := filepath.Join(root, ".git", "hooks", "post-commit")

	// 先安装
	if err := InstallHook(root, binaryPath); err != nil {
		t.Fatalf("InstallHook 失败: %v", err)
	}

	// 卸载
	if err := UninstallHook(root); err != nil {
		t.Fatalf("UninstallHook 失败: %v", err)
	}

	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatal("卸载后 hook 文件应被删除")
	}
}

func TestUninstallHook_NoErrorWhenNotExists(t *testing.T) {
	root := setupTestRepo(t)
	// 没有安装 hook，直接卸载不应报错
	if err := UninstallHook(root); err != nil {
		t.Fatalf("hook 不存在时 UninstallHook 不应报错: %v", err)
	}
}

func TestInstallHook_CreatesGitDirIfMissing(t *testing.T) {
	// 即使 .git 目录不存在，InstallHook 也应自动创建完整路径
	root := t.TempDir()
	binaryPath := "/usr/local/bin/chronodraft-mcp"

	if err := InstallHook(root, binaryPath); err != nil {
		t.Fatalf("InstallHook 应自动创建 .git/hooks 目录，但返回错误: %v", err)
	}

	hookPath := filepath.Join(root, ".git", "hooks", "post-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Fatal("post-commit hook 文件应被创建")
	}
}
