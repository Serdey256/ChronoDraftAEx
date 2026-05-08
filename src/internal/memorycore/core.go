// Package memorycore 是系统的记忆索引内核
// 负责整合变动检测、AI 摘要生成和知识库索引，提供统一的记忆管理接口
package memorycore

import (
	"ChronoDraftAEx/internal/agentswriter"
	"ChronoDraftAEx/internal/changeorganize"
	"ChronoDraftAEx/internal/changedetect"
	"ChronoDraftAEx/internal/kbindex"
	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
	"context"
	"fmt"
	"time"
)

// MemoryCore 记忆内核
type MemoryCore struct {
	detector     *changedetect.Detector
	organizer    *changeorganize.Organizer
	kbIndex      *kbindex.KBIndex
	agentsWriter *agentswriter.Writer
	ctx          context.Context
}

// NewMemoryCore 创建记忆内核实例
func NewMemoryCore(projectRoot, aiAPIKey, aiBaseURL, aiModel string) (*MemoryCore, error) {
	detector := changedetect.NewDetector(projectRoot)
	organizer := changeorganize.NewOrganizer(aiAPIKey, aiBaseURL, aiModel)
	kbi, err := kbindex.NewKBIndex(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("初始化知识库索引失败: %w", err)
	}

	// 配置嵌入模型 API（与 chat API 共享 key 和 base）
	kbi.SetEmbeddingConfig(aiAPIKey, aiBaseURL, "")

	ctx := context.Background()
	if err := kbi.Init(ctx); err != nil {
		return nil, fmt.Errorf("初始化知识库失败: %w", err)
	}

	m := &MemoryCore{
		detector:     detector,
		organizer:    organizer,
		kbIndex:      kbi,
		agentsWriter: agentswriter.NewWriter(projectRoot),
		ctx:          ctx,
	}
	return m, nil
}

// CaptureAndIndex 捕获变动、生成摘要并索引到知识库（完整工作流）
func (m *MemoryCore) CaptureAndIndex(oldSnap, newSnap map[string]changedetect.FileSnapshot, sessionID string) (*models.StructuredEntry, error) {
	// 1. 检测变动
	record := m.detector.DetectChanges(oldSnap, newSnap, sessionID)
	if len(record.Changes) == 0 {
		return nil, fmt.Errorf("未检测到文件变动")
	}

	// 2. AI 生成结构化摘要
	entry, err := m.organizer.Organize(record)
	if err != nil {
		return nil, fmt.Errorf("生成结构化摘要失败: %w", err)
	}

	// 3. 索引到知识库
	if err := m.kbIndex.IndexEntry(m.ctx, entry); err != nil {
		return nil, fmt.Errorf("索引知识库失败: %w", err)
	}

	// 自动写入 AGENTS.md
	if results, err := m.kbIndex.Search(m.ctx, "最新变更", 10); err == nil {
		_ = m.agentsWriter.Write(results)
	}

	return entry, nil
}

// IndexProject 全量索引：将项目现有代码当作一次初始变更处理
func (m *MemoryCore) IndexProject() (*models.StructuredEntry, error) {
	snapshot, err := m.detector.ScanSnapshot()
	if err != nil {
		return nil, fmt.Errorf("扫描项目文件失败: %w", err)
	}

	var changes []models.FileChange
	for path := range snapshot {
		changes = append(changes, models.FileChange{
			Path:       path,
			ChangeType: "add",
		})
	}

	if len(changes) == 0 {
		return nil, fmt.Errorf("项目中未发现任何文件")
	}

	record := &models.ChangeRecord{
		ID:        utils.GenerateID(),
		Timestamp: time.Now(),
		SessionID: "full-index",
		Changes:   changes,
	}

	entry, err := m.organizer.Organize(record)
	if err != nil {
		return nil, fmt.Errorf("生成结构化摘要失败: %w", err)
	}

	if err := m.kbIndex.IndexEntry(m.ctx, entry); err != nil {
		return nil, fmt.Errorf("索引知识库失败: %w", err)
	}

	if results, err := m.kbIndex.Search(m.ctx, "最新变更", 10); err == nil {
		_ = m.agentsWriter.Write(results)
	}

	return entry, nil
}

// SearchKnowledge 搜索知识库
func (m *MemoryCore) SearchKnowledge(query string, topK int) ([]models.SearchResult, error) {
	return m.kbIndex.Search(m.ctx, query, topK)
}

// GetGraphData 获取图谱数据
func (m *MemoryCore) GetGraphData(limit int) ([]models.KnowledgeNode, []models.KnowledgeEdge, error) {
	return m.kbIndex.GetGraphData(m.ctx, limit)
}

// ListEntries 列出知识条目
func (m *MemoryCore) ListEntries(offset, limit int) ([]models.StructuredEntry, error) {
	return m.kbIndex.ListEntries(offset, limit)
}

// CreateSnapshot 创建当前项目快照
func (m *MemoryCore) CreateSnapshot(version string, dependencies []string) (*models.ProjectSnapshot, error) {
	snapshot := &models.ProjectSnapshot{
		ID:           utils.GenerateID(),
		Timestamp:    time.Now(),
		Version:      version,
		Dependencies: dependencies,
		Metadata:     map[string]string{},
	}
	if err := m.kbIndex.SaveSnapshot(snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

// ListSnapshots 列出所有快照
func (m *MemoryCore) ListSnapshots() ([]models.ProjectSnapshot, error) {
	return m.kbIndex.ListSnapshots()
}

// GenerateAgentsMD 手动触发 AGENTS.md 生成
func (m *MemoryCore) GenerateAgentsMD() error {
	results, err := m.kbIndex.Search(m.ctx, "最新变更", 10)
	if err != nil {
		return err
	}
	return m.agentsWriter.Write(results)
}

// IndexEntry 直接将结构化条目索引到知识库（不经过 AI 处理）
func (m *MemoryCore) IndexEntry(entry *models.StructuredEntry) error {
	return m.kbIndex.IndexEntry(m.ctx, entry)
}

// RefreshAgentsMD 刷新 AGENTS.md 文件
func (m *MemoryCore) RefreshAgentsMD() error {
	results, err := m.kbIndex.Search(m.ctx, "最新变更", 10)
	if err != nil {
		return err
	}
	return m.agentsWriter.Write(results)
}

// Close 关闭记忆内核，释放资源
func (m *MemoryCore) Close() error {
	return m.kbIndex.Close()
}
