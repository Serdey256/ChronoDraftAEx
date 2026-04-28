// Package main 的 App 层，暴露给 Wails 前端调用的绑定方法
package main

import (
	"ChronoDraftAEx/internal/changedetect"
	"ChronoDraftAEx/internal/memorycore"
	"ChronoDraftAEx/internal/mcp"
	"ChronoDraftAEx/pkg/models"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// App 应用主结构
type App struct {
	ctx        context.Context
	core       *memorycore.MemoryCore
	mcpServer  *mcp.MCPServer
	projectRoot string
	lastSnap   map[string]changedetect.FileSnapshot
}

// NewApp 创建 App 实例
func NewApp() *App {
	return &App{}
}

// Startup 在应用启动时调用，初始化核心服务
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// 获取当前工作目录作为项目根目录
	root, err := os.Getwd()
	if err != nil {
		root = "."
	}
	// 向上寻找项目根（简单策略：包含 go.mod 或 package.json 的目录）
	a.projectRoot = findProjectRoot(root)

	// 从环境变量读取 AI 配置
	apiKey := os.Getenv("CHRONODRAFT_AI_KEY")
	apiBase := os.Getenv("CHRONODRAFT_AI_BASE")
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	model := os.Getenv("CHRONODRAFT_AI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	// 初始化记忆内核
	core, err := memorycore.NewMemoryCore(a.projectRoot, apiKey, apiBase, model)
	if err != nil {
		fmt.Printf("记忆内核初始化失败: %v\n", err)
		return
	}
	a.core = core

	// 启动 MCP 服务器
	a.mcpServer = mcp.NewMCPServer(core, ":8787")
	_ = a.mcpServer.Start()

	// 生成初始快照
	a.lastSnap, _ = changedetect.NewDetector(a.projectRoot).ScanSnapshot()
}

// Shutdown 在应用关闭时调用
func (a *App) Shutdown(ctx context.Context) {
	if a.mcpServer != nil {
		_ = a.mcpServer.Stop(ctx)
	}
	if a.core != nil {
		_ = a.core.Close()
	}
}

// CaptureChanges 前端调用：手动触发变动捕获与索引
func (a *App) CaptureChanges(sessionID string) (*models.StructuredEntry, error) {
	if a.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	newSnap, err := changedetect.NewDetector(a.projectRoot).ScanSnapshot()
	if err != nil {
		return nil, err
	}
	entry, err := a.core.CaptureAndIndex(a.lastSnap, newSnap, sessionID)
	if err != nil {
		return nil, err
	}
	a.lastSnap = newSnap
	return entry, nil
}

// SearchKnowledge 前端调用：搜索知识库
func (a *App) SearchKnowledge(query string, topK int) ([]models.SearchResult, error) {
	if a.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	if topK == 0 {
		topK = 10
	}
	return a.core.SearchKnowledge(query, topK)
}

// CreateSnapshot 前端调用：创建项目快照
func (a *App) CreateSnapshot(version string, dependencies []string) (*models.ProjectSnapshot, error) {
	if a.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	return a.core.CreateSnapshot(version, dependencies)
}

// GetProjectRoot 返回当前项目根目录
func (a *App) GetProjectRoot() string {
	return a.projectRoot
}

// findProjectRoot 向上查找包含 go.mod 或 package.json 的目录
func findProjectRoot(start string) string {
	for {
		if _, err := os.Stat(filepath.Join(start, "go.mod")); err == nil {
			return start
		}
		if _, err := os.Stat(filepath.Join(start, "package.json")); err == nil {
			return start
		}
		parent := filepath.Dir(start)
		if parent == start {
			break
		}
		start = parent
	}
	return start
}
