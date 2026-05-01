// Package main 的 Service 层，暴露给 Wails 前端调用的绑定方法
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

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ChronoService 提供前端可调用的核心业务方法
type ChronoService struct {
	ctx         context.Context
	core        *memorycore.MemoryCore
	mcpServer   *mcp.MCPServer
	projectRoot string
	lastSnap    map[string]changedetect.FileSnapshot
}

func NewChronoService() *ChronoService {
	return &ChronoService{}
}

func (s *ChronoService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx

	root, err := os.Getwd()
	if err != nil {
		root = "."
	}
	s.projectRoot = findProjectRoot(root)

	apiKey := os.Getenv("CHRONODRAFT_AI_KEY")
	apiBase := os.Getenv("CHRONODRAFT_AI_BASE")
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	model := os.Getenv("CHRONODRAFT_AI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	core, err := memorycore.NewMemoryCore(s.projectRoot, apiKey, apiBase, model)
	if err != nil {
		return fmt.Errorf("记忆内核初始化失败: %w", err)
	}
	s.core = core

	s.mcpServer = mcp.NewMCPServer(core, ":8787")
	_ = s.mcpServer.Start()

	detector := changedetect.NewDetector(s.projectRoot)
	s.lastSnap, _ = detector.ScanSnapshot()
	return nil
}

func (s *ChronoService) ServiceShutdown(ctx context.Context) error {
	if s.mcpServer != nil {
		_ = s.mcpServer.Stop(ctx)
	}
	if s.core != nil {
		_ = s.core.Close()
	}
	return nil
}

// CaptureChanges 前端调用：手动触发变动捕获与索引
func (s *ChronoService) CaptureChanges(sessionID string) (*models.StructuredEntry, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	newSnap, err := changedetect.NewDetector(s.projectRoot).ScanSnapshot()
	if err != nil {
		return nil, err
	}
	entry, err := s.core.CaptureAndIndex(s.lastSnap, newSnap, sessionID)
	if err != nil {
		return nil, err
	}
	s.lastSnap = newSnap
	return entry, nil
}

// SearchKnowledge 前端调用：搜索知识库
func (s *ChronoService) SearchKnowledge(query string, topK int) ([]models.SearchResult, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	if topK == 0 {
		topK = 10
	}
	return s.core.SearchKnowledge(query, topK)
}

// CreateSnapshot 前端调用：创建项目快照
func (s *ChronoService) CreateSnapshot(version string, dependencies []string) (*models.ProjectSnapshot, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	return s.core.CreateSnapshot(version, dependencies)
}

// ListSnapshots 前端调用：列出所有快照
func (s *ChronoService) ListSnapshots() ([]models.ProjectSnapshot, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	return s.core.ListSnapshots()
}

// GetGraphData 前端调用：获取知识图谱数据
func (s *ChronoService) GetGraphData(limit int) (*models.GraphData, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	if limit <= 0 {
		limit = 100
	}
	nodes, edges, err := s.core.GetGraphData(limit)
	if err != nil {
		return nil, err
	}
	return &models.GraphData{Nodes: nodes, Edges: edges}, nil
}

// ListEntries 前端调用：分页列出知识条目
func (s *ChronoService) ListEntries(offset, limit int) ([]models.StructuredEntry, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	if limit <= 0 {
		limit = 20
	}
	return s.core.ListEntries(offset, limit)
}

// GetProjectRoot 返回当前项目根目录
func (s *ChronoService) GetProjectRoot() string {
	return s.projectRoot
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
