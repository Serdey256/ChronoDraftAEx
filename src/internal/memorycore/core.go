// Package memorycore 是系统的记忆索引内核
// 负责整合变动检测、AI 摘要生成和知识库索引，提供统一的记忆管理接口
package memorycore

import (
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
	detector  *changedetect.Detector
	organizer *changeorganize.Organizer
	kbIndex   *kbindex.KBIndex
	ctx       context.Context
}

// NewMemoryCore 创建记忆内核实例
func NewMemoryCore(projectRoot, aiAPIKey, aiBaseURL, aiModel string) (*MemoryCore, error) {
	detector := changedetect.NewDetector(projectRoot)
	organizer := changeorganize.NewOrganizer(aiAPIKey, aiBaseURL, aiModel)
	kbi, err := kbindex.NewKBIndex(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("初始化知识库索引失败: %w", err)
	}

	ctx := context.Background()
	if err := kbi.Init(ctx); err != nil {
		return nil, fmt.Errorf("初始化知识库失败: %w", err)
	}

	return &MemoryCore{
		detector:  detector,
		organizer: organizer,
		kbIndex:   kbi,
		ctx:       ctx,
	}, nil
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

	return entry, nil
}

// SearchKnowledge 搜索知识库
func (m *MemoryCore) SearchKnowledge(query string, topK int) ([]models.SearchResult, error) {
	return m.kbIndex.Search(m.ctx, query, topK)
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

// Close 关闭记忆内核，释放资源
func (m *MemoryCore) Close() error {
	return m.kbIndex.Close()
}
