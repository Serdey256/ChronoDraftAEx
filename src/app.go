// Package main 的 Service 层，暴露给 Wails 前端调用的绑定方法
package main

import (
	"ChronoDraftAEx/internal/changedetect"
	"ChronoDraftAEx/internal/config"
	"ChronoDraftAEx/internal/depsdetect"
	"ChronoDraftAEx/internal/filewatcher"
	"ChronoDraftAEx/internal/memorycore"
	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ChronoService 提供前端可调用的核心业务方法
type ChronoService struct {
	ctx           context.Context
	core          *memorycore.MemoryCore
	projectRoot   string
	lastSnap      map[string]changedetect.FileSnapshot
	configManager *config.Manager
	watcher       *filewatcher.Watcher
	apiKey        string
	apiBase       string
	apiModel      string
}

func NewChronoService() *ChronoService {
	return &ChronoService{}
}

func (s *ChronoService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx

	// 初始化配置管理器
	configDir := filepath.Join(os.TempDir(), "chronodraftaex")
	s.configManager = config.NewManager(configDir)
	if err := s.configManager.Load(); err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
	}

	// 读取 API 配置
	s.apiKey = os.Getenv("CHRONODRAFT_AI_KEY")
	s.apiBase = os.Getenv("CHRONODRAFT_AI_BASE")
	if s.apiBase == "" {
		s.apiBase = "https://api.openai.com/v1"
	}
	s.apiModel = os.Getenv("CHRONODRAFT_AI_MODEL")
	if s.apiModel == "" {
		s.apiModel = "gpt-4o"
	}

	// 尝试加载活跃项目
	activeProject := s.configManager.GetActiveProject()
	if activeProject != nil {
		s.projectRoot = activeProject.Path
	} else {
		// 默认使用当前工作目录
		root, err := os.Getwd()
		if err != nil {
			root = "."
		}
		s.projectRoot = findProjectRoot(root)
	}

	if err := s.initCore(); err != nil {
		return err
	}

	return nil
}

// initCore 初始化记忆内核
func (s *ChronoService) initCore() error {
	if s.core != nil {
		_ = s.core.Close()
	}

	core, err := memorycore.NewMemoryCore(s.projectRoot, s.apiKey, s.apiBase, s.apiModel)
	if err != nil {
		return fmt.Errorf("记忆内核初始化失败: %w", err)
	}
	s.core = core

	loaded, err := changedetect.LoadLastSnap(s.projectRoot)
	if err != nil || loaded == nil {
		detector := changedetect.NewDetector(s.projectRoot)
		s.lastSnap, _ = detector.ScanSnapshot()
		_ = changedetect.SaveLastSnap(s.projectRoot, s.lastSnap)
	} else {
		s.lastSnap = loaded
	}

	// 初始化文件监听器（不自动启动）
	s.watcher = filewatcher.NewWatcher(s.projectRoot, func() {
		s.CaptureChanges("auto-" + fmt.Sprintf("%d", time.Now().UnixMilli()))
	})

	return nil
}

func (s *ChronoService) ServiceShutdown(ctx context.Context) error {
	if s.watcher != nil {
		s.watcher.Stop()
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
	_ = changedetect.SaveLastSnap(s.projectRoot, s.lastSnap)
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
	// Auto-detect dependencies if none provided
	if len(dependencies) == 0 {
		detected, err := depsdetect.DetectDependencies(s.projectRoot)
		if err == nil && len(detected) > 0 {
			dependencies = detected
		}
	}
	return s.core.CreateSnapshot(version, dependencies)
}

// DetectDependencies 前端调用：检测项目依赖
func (s *ChronoService) DetectDependencies() ([]string, error) {
	return depsdetect.DetectDependencies(s.projectRoot)
}

// GenerateAgentsMD 兼容保留：纯 MCP 快照模式下不再生成文件
func (s *ChronoService) GenerateAgentsMD() error {
	if s.core == nil {
		return fmt.Errorf("记忆内核未初始化")
	}
	return s.core.GenerateAgentsMD()
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

// SearchGraph 前端调用：关键词查询关联图谱
func (s *ChronoService) SearchGraph(query string, topK int) (*models.GraphData, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	if topK <= 0 {
		topK = 5
	}
	nodes, edges, err := s.core.SearchGraph(query, topK)
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

// DeleteEntry 前端调用：删除知识条目，并返回删除前内容用于撤销
func (s *ChronoService) DeleteEntry(entryID string) (*models.StructuredEntry, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	return s.core.DeleteEntry(entryID)
}

// RestoreEntry 前端调用：恢复此前删除的知识条目
func (s *ChronoService) RestoreEntry(entry models.StructuredEntry) error {
	if s.core == nil {
		return fmt.Errorf("记忆内核未初始化")
	}
	return s.core.RestoreEntry(&entry)
}

// GetCodeEntities 前端调用：获取指定文件的代码实体（AST 分析结果）
func (s *ChronoService) GetCodeEntities(filePath string) ([]models.CodeEntity, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	return s.core.GetCodeEntities(filePath)
}

// ListCodeEntityFiles 前端调用：获取所有已分析的文件路径
func (s *ChronoService) ListCodeEntityFiles() ([]string, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	return s.core.ListCodeEntityFiles()
}

// AnnotateAllCodeEntities 前端调用：对所有已有代码实体进行批量 AI 语义标注
func (s *ChronoService) AnnotateAllCodeEntities() error {
	if s.core == nil {
		return fmt.Errorf("记忆内核未初始化")
	}
	return s.core.AnnotateAllCodeEntities()
}

// GetAllCodeEntities 前端调用：获取所有代码实体（AST 分析结果）
func (s *ChronoService) GetAllCodeEntities() ([]models.CodeEntity, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	return s.core.GetAllCodeEntities()
}

// GetAnnotationProgress 前端调用：获取 AI 标注进度 (current, total)
func (s *ChronoService) GetAnnotationProgress() map[string]int {
	if s.core == nil {
		return map[string]int{"current": 0, "total": 0}
	}
	c, t := s.core.GetAnnotationProgress()
	return map[string]int{"current": c, "total": t}
}

// GetIndexPhase 前端调用：获取全量索引当前阶段
func (s *ChronoService) GetIndexPhase() string {
	if s.core == nil {
		return ""
	}
	return s.core.GetIndexPhase()
}

// SetAIAnnotation 前端调用：启用/禁用 AI 语义标注
func (s *ChronoService) SetAIAnnotation(enabled bool) {
	if s.core != nil {
		s.core.SetAIAnnotation(enabled)
	}
}

// IsAIAnnotationEnabled 前端调用：查询 AI 语义标注是否启用
func (s *ChronoService) IsAIAnnotationEnabled() bool {
	if s.core == nil {
		return false
	}
	return s.core.IsAIAnnotationEnabled()
}

// SetAgentsMDEnabled 兼容保留：纯 MCP 快照模式下无实际效果
func (s *ChronoService) SetAgentsMDEnabled(enabled bool) {
	if s.core != nil {
		s.core.SetAgentsMDEnabled(enabled)
	}
}

// IsAgentsMDEnabled 兼容保留：纯 MCP 快照模式下始终为 false
func (s *ChronoService) IsAgentsMDEnabled() bool {
	if s.core == nil {
		return true
	}
	return s.core.IsAgentsMDEnabled()
}

// IndexProject 前端调用：项目脚手架（零AI成本的轻量结构扫描）
func (s *ChronoService) IndexProject() (*models.StructuredEntry, error) {
	if s.core == nil {
		return nil, fmt.Errorf("记忆内核未初始化")
	}
	return s.core.IndexProject()
}

// IsKnowledgeBaseEmpty 前端调用：检查知识库是否为空
func (s *ChronoService) IsKnowledgeBaseEmpty() bool {
	if s.core == nil {
		return true
	}
	entries, err := s.core.ListEntries(0, 1)
	return err != nil || len(entries) == 0
}

// GetProjectRoot 返回当前项目根目录
func (s *ChronoService) GetProjectRoot() string {
	return s.projectRoot
}

// GetCurrentProject 返回当前活跃项目信息
func (s *ChronoService) GetCurrentProject() *config.ProjectConfig {
	if s.configManager == nil {
		return nil
	}
	return s.configManager.GetActiveProject()
}

// ListProjects 列出所有已配置的项目
func (s *ChronoService) ListProjects() []config.ProjectConfig {
	if s.configManager == nil {
		return []config.ProjectConfig{}
	}
	return s.configManager.ListProjects()
}

// AddProject 添加新项目并切换
func (s *ChronoService) AddProject(name, path, description string) (*config.ProjectConfig, error) {
	if s.configManager == nil {
		return nil, fmt.Errorf("配置管理器未初始化")
	}

	// 验证路径是否存在
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("项目路径不存在: %w", err)
	}

	project := config.ProjectConfig{
		ID:          utils.GenerateID(),
		Name:        name,
		Path:        path,
		Description: description,
		IsActive:    true,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	if err := s.configManager.AddProject(project); err != nil {
		return nil, err
	}

	// 切换到新项目
	if err := s.SwitchProject(project.ID); err != nil {
		return nil, err
	}

	return &project, nil
}

// SwitchProject 切换到指定项目
func (s *ChronoService) SwitchProject(projectID string) error {
	if s.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	project := s.configManager.GetProjectByID(projectID)
	if project == nil {
		return fmt.Errorf("项目不存在: %s", projectID)
	}

	// 更新配置
	if err := s.configManager.SetActiveProject(projectID); err != nil {
		return err
	}

	// 切换项目根目录
	s.projectRoot = project.Path

	// 停止旧的文件监听器
	if s.watcher != nil {
		s.watcher.Stop()
	}

	// 重新初始化内核
	if err := s.initCore(); err != nil {
		return fmt.Errorf("切换项目失败: %w", err)
	}

	return nil
}

// RemoveProject 删除项目
func (s *ChronoService) RemoveProject(projectID string) error {
	if s.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	return s.configManager.RemoveProject(projectID)
}

// StartWatcher 前端调用：启动文件监控
func (s *ChronoService) StartWatcher() error {
	if s.watcher == nil {
		return fmt.Errorf("监听器未初始化")
	}
	return s.watcher.Start()
}

// StopWatcher 前端调用：停止文件监控
func (s *ChronoService) StopWatcher() {
	if s.watcher != nil {
		s.watcher.Stop()
	}
}

// IsWatcherRunning 前端调用：查询监控状态
func (s *ChronoService) IsWatcherRunning() bool {
	if s.watcher == nil {
		return false
	}
	return s.watcher.IsRunning()
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
