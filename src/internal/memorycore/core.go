// Package memorycore 是系统的记忆索引内核
// 负责整合变动检测、AI 摘要生成和知识库索引，提供统一的记忆管理接口
package memorycore

import (
	"ChronoDraftAEx/internal/agentswriter"
	"ChronoDraftAEx/internal/changeorganize"
	"ChronoDraftAEx/internal/changedetect"
	"ChronoDraftAEx/internal/codeanalysis"
	"ChronoDraftAEx/internal/githook"
	"ChronoDraftAEx/internal/kbindex"
	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MemoryCore 记忆内核
type MemoryCore struct {
	detector            *changedetect.Detector
	organizer           *changeorganize.Organizer
	kbIndex             *kbindex.KBIndex
	agentsWriter        *agentswriter.Writer
	ctx                 context.Context
	projectRoot         string
	aiAnnotationEnabled bool
	agentsMDEnabled     bool // AGENTS.md 自动生成开关，默认 true
	annotationCurrent   int
	annotationTotal     int
	indexPhase          string
}

// NewMemoryCore 创建记忆内核实例
func NewMemoryCore(projectRoot, aiAPIKey, aiBaseURL, aiModel string) (*MemoryCore, error) {
	detector := changedetect.NewDetector(projectRoot)
	organizer := changeorganize.NewOrganizer(aiAPIKey, aiBaseURL, aiModel)
	kbi, err := kbindex.NewKBIndex(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("初始化知识库索引失败: %w", err)
	}

	// 配置嵌入模型 API（与 chat API 共享 key 和 base，模型可单独配置）
	embeddingModel := os.Getenv("CHRONODRAFT_EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = "text-embedding-3-small" // OpenAI 默认
	}
	kbi.SetEmbeddingConfig(aiAPIKey, aiBaseURL, embeddingModel)

	ctx := context.Background()
	if err := kbi.Init(ctx); err != nil {
		return nil, fmt.Errorf("初始化知识库失败: %w", err)
	}

	m := &MemoryCore{
		detector:            detector,
		organizer:           organizer,
		kbIndex:             kbi,
		agentsWriter:        agentswriter.NewWriter(projectRoot),
		ctx:                 ctx,
		projectRoot:         projectRoot,
		aiAnnotationEnabled: false,
		agentsMDEnabled:     true,
	}
	// Auto-refresh AGENTS.md on startup to ensure latest data
	m.refreshSmartAgentsMD()

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

	// 自动刷新 AGENTS.md（智能格式）
	m.refreshSmartAgentsMD()

	return entry, nil
}

// IndexProject 全量索引：将项目现有代码当作一次初始变更处理
func (m *MemoryCore) IndexProject() (*models.StructuredEntry, error) {
	m.indexPhase = "扫描文件"
	snapshot, err := m.detector.ScanSnapshot()
	if err != nil {
		m.indexPhase = ""
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

	// Phase 2: Zero-cost operations (run FIRST, before AI)
	// These should always succeed regardless of AI availability
	// 2a. Install Git Hook
	m.indexPhase = "安装GitHook"
	binaryPath := os.Getenv("CHRONODRAFT_BINARY_PATH")
	if binaryPath != "" {
		if err := githook.InstallHook(m.projectRoot, binaryPath); err != nil {
			log.Printf("警告: 安装 Git Hook 失败: %v", err)
		}
	} else {
		log.Println("警告: 环境变量 CHRONODRAFT_BINARY_PATH 未设置，跳过 Git Hook 安装")
	}
	// 2b. AST code analysis
	m.indexPhase = "AST代码分析"
	if err := codeanalysis.AnalyzeProject(m.projectRoot, m.AnnotateAndSaveEntities); err != nil {
		log.Printf("警告: AST 代码分析失败: %v", err)
	}

	// Phase 3: AI-dependent step (may fail, we continue regardless)
	m.indexPhase = "AI摘要"
	entry, aiErr := m.organizer.Organize(record)
	if aiErr != nil {
		log.Printf("警告: AI 组织摘要失败（将跳过知识条目索引）: %v", aiErr)
		// Create a minimal entry so callers get something back
		entry = &models.StructuredEntry{
			ID:        utils.GenerateID(),
			Timestamp: time.Now(),
			SessionID: "full-index",
			Summary:   "项目全量索引（AI 摘要未生成）",
			AffectedFiles: changes,
		}
	}

	// Try to index the entry (may fail if AI didn't produce embeddings, that's OK)
	if err := m.kbIndex.IndexEntry(m.ctx, entry); err != nil {
		log.Printf("警告: 索引知识条目失败: %v", err)
	}

	// Phase 4: Generate smart AGENTS.md (always, using all available data)
	// NOTE: do NOT call old Write() anymore - only smart format
	m.indexPhase = "生成AGENTS.md"
	m.refreshSmartAgentsMD()

	m.indexPhase = ""
	if aiErr != nil {
		return entry, nil // return successfully even without AI
	}
	return entry, nil
}

// SearchKnowledge 搜索知识库
func (m *MemoryCore) SearchKnowledge(query string, topK int) ([]models.SearchResult, error) {
	return m.kbIndex.Search(m.ctx, query, topK)
}

// GetRelatedNodes 查询与指定条目相关的知识节点
func (m *MemoryCore) GetRelatedNodes(entryID string) ([]models.KnowledgeNode, []models.KnowledgeEdge, error) {
	return m.kbIndex.QueryRelated(m.ctx, entryID)
}

// GetGraphData 获取图谱数据
func (m *MemoryCore) GetGraphData(limit int) ([]models.KnowledgeNode, []models.KnowledgeEdge, error) {
	return m.kbIndex.GetGraphData(m.ctx, limit)
}

// SearchGraph 查询图谱：根据关键词搜索，返回关联子图
func (m *MemoryCore) SearchGraph(query string, topK int) ([]models.KnowledgeNode, []models.KnowledgeEdge, error) {
	results, err := m.kbIndex.Search(m.ctx, query, topK)
	if err != nil || len(results) == 0 {
		entries, _ := m.kbIndex.ListEntries(0, 50)
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Summary), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(e.DesignDecision), strings.ToLower(query)) {
				results = append(results, models.SearchResult{Entry: e, Score: 0.5})
			}
		}
		if len(results) > topK { results = results[:topK] }
	}
	nodeMap := make(map[string]models.KnowledgeNode)
	edgeMap := make(map[string]bool)
	for _, r := range results {
		nodes, edges, _ := m.kbIndex.QueryRelated(m.ctx, r.Entry.ID)
		for _, n := range nodes { nodeMap[n.ID] = n }
		for _, e := range edges {
			k := e.SourceID + "->" + e.TargetID
			if !edgeMap[k] { edgeMap[k] = true }
		}
	}
	nodes := make([]models.KnowledgeNode, 0, len(nodeMap))
	for _, n := range nodeMap { nodes = append(nodes, n) }
	var edgeList []models.KnowledgeEdge
	for k := range edgeMap {
		parts := strings.SplitN(k, "->", 2)
		if len(parts) == 2 { edgeList = append(edgeList, models.KnowledgeEdge{SourceID: parts[0], TargetID: parts[1]}) }
	}
	return nodes, edgeList, nil
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

// GenerateAgentsMD 手动触发 AGENTS.md 生成（使用智能格式）
func (m *MemoryCore) GenerateAgentsMD() error {
	m.refreshSmartAgentsMD()
	return nil
}

// IndexEntry 直接将结构化条目索引到知识库（不经过 AI 处理）
func (m *MemoryCore) IndexEntry(entry *models.StructuredEntry) error {
	return m.kbIndex.IndexEntry(m.ctx, entry)
}

// RefreshAgentsMD 刷新 AGENTS.md 文件
func (m *MemoryCore) RefreshAgentsMD() error {
	m.refreshSmartAgentsMD()
	return nil
}

// buildDirStructure 构建项目目录结构（2层深度），返回 markdown 格式字符串
func (m *MemoryCore) buildDirStructure() string {
	var sb strings.Builder
	sb.WriteString("```\n")

	entries, err := os.ReadDir(m.projectRoot)
	if err != nil {
		return ""
	}

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "build" || name == "dist" {
			continue
		}
		if e.IsDir() {
			sb.WriteString(fmt.Sprintf("%s/\n", name))
			subs, err := os.ReadDir(filepath.Join(m.projectRoot, name))
			if err != nil {
				continue
			}
			count := 0
			for _, s := range subs {
				if count >= 10 {
					sb.WriteString("  ...\n")
					break
				}
				if strings.HasPrefix(s.Name(), ".") {
					continue
				}
				count++
				if s.IsDir() {
					sb.WriteString(fmt.Sprintf("  %s/\n", s.Name()))
				} else {
					sb.WriteString(fmt.Sprintf("  %s\n", s.Name()))
				}
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s\n", name))
		}
	}

	sb.WriteString("```\n")
	return sb.String()
}

// SetAgentsMDEnabled 启用/禁用 AGENTS.md 自动生成
func (m *MemoryCore) SetAgentsMDEnabled(enabled bool) {
	m.agentsMDEnabled = enabled
}

// IsAgentsMDEnabled 查询 AGENTS.md 自动生成是否启用
func (m *MemoryCore) IsAgentsMDEnabled() bool {
	return m.agentsMDEnabled
}

// refreshSmartAgentsMD 使用智能方式刷新 AGENTS.md
func (m *MemoryCore) refreshSmartAgentsMD() {
	if !m.agentsMDEnabled {
		return
	}
	commits, err := m.kbIndex.ListCommits(3)
	if err != nil {
		log.Printf("警告: 获取 commit 记录失败: %v", err)
		commits = nil
	}

	entries, err := m.kbIndex.ListEntries(0, 5)
	if err != nil {
		log.Printf("警告: 获取知识条目失败: %v", err)
		entries = nil
	}

	dirStructure := m.buildDirStructure()

	content := agentswriter.GenerateSmartMarkdown(commits, entries, dirStructure)

	if err := m.agentsWriter.WriteContent(content); err != nil {
		log.Printf("警告: 写入智能 AGENTS.md 失败: %v", err)
	}
}

// SaveCommit 保存 git commit 记录
func (m *MemoryCore) SaveCommit(hash, message, author, timestamp, files string, ins, dels int) error {
	return m.kbIndex.SaveCommit(hash, message, author, timestamp, files, ins, dels)
}

// GetCodeEntities 查询指定文件的所有代码实体
func (m *MemoryCore) GetCodeEntities(filePath string) ([]models.CodeEntity, error) {
	return m.kbIndex.GetCodeEntities(filePath)
}

// ListCodeEntityFiles 返回所有已分析的文件路径
func (m *MemoryCore) ListCodeEntityFiles() ([]string, error) {
	return m.kbIndex.ListCodeEntityFiles()
}

// GetAllCodeEntities 返回所有代码实体
func (m *MemoryCore) GetAllCodeEntities() ([]models.CodeEntity, error) {
	return m.kbIndex.GetAllCodeEntities()
}

// ListCommits 列出 git commit 记录
func (m *MemoryCore) ListCommits(limit int) ([]models.CommitRecord, error) {
	return m.kbIndex.ListCommits(limit)
}

// SaveCodeEntities 保存文件的所有代码实体
func (m *MemoryCore) SaveCodeEntities(filePath string, entities []models.CodeEntity) error {
	return m.kbIndex.SaveCodeEntities(filePath, entities)
}

// GetAnnotationProgress 返回 AI 标注进度 (current, total)
func (m *MemoryCore) GetAnnotationProgress() (int, int) {
	return m.annotationCurrent, m.annotationTotal
}

// GetIndexPhase 返回全量索引当前阶段
func (m *MemoryCore) GetIndexPhase() string {
	return m.indexPhase
}

// SetAIAnnotation 启用/禁用 AI 语义标注
func (m *MemoryCore) SetAIAnnotation(enabled bool) {
	m.aiAnnotationEnabled = enabled
	if enabled {
		log.Println("AI 语义标注已启用 — 代码实体将附带 AI 生成的描述")
	} else {
		log.Println("AI 语义标注已禁用")
	}
}

// IsAIAnnotationEnabled 查询 AI 语义标注是否启用
func (m *MemoryCore) IsAIAnnotationEnabled() bool {
	return m.aiAnnotationEnabled
}

// annotateCodeEntities 对代码实体进行 AI 语义标注（内部辅助方法）
func (m *MemoryCore) annotateCodeEntities(entities []models.CodeEntity) ([]models.CodeEntity, error) {
	return m.organizer.AnnotateEntities(entities)
}

// AnnotateAllCodeEntities 对所有已有代码实体进行批量 AI 语义标注
// 只标注没有 description 的实体，避免重复 API 调用
func (m *MemoryCore) AnnotateAllCodeEntities() error {
	if !m.aiAnnotationEnabled {
		return fmt.Errorf("AI 语义标注未启用")
	}
	all, err := m.kbIndex.GetAllCodeEntities()
	if err != nil {
		return fmt.Errorf("获取代码实体失败: %w", err)
	}
	if len(all) == 0 {
		return nil
	}

	// Filter to entities without description
	var toAnnotate []models.CodeEntity
	for _, e := range all {
		if e.Metadata == "" || !strings.Contains(e.Metadata, `"description"`) {
			toAnnotate = append(toAnnotate, e)
		}
	}
	if len(toAnnotate) == 0 {
		log.Println("所有代码实体已有 AI 描述，跳过标注")
		return nil
	}

	log.Printf("开始 AI 语义标注 %d 个代码实体...", len(toAnnotate))

	// Batch in groups of 10 to avoid oversized prompts
	batchSize := 10
	m.annotationTotal = len(toAnnotate)
	m.annotationCurrent = 0
	for i := 0; i < len(toAnnotate); i += batchSize {
		end := i + batchSize
		if end > len(toAnnotate) {
			end = len(toAnnotate)
		}
		batch := toAnnotate[i:end]
		annotated, err := m.annotateCodeEntities(batch)
		if err != nil {
			log.Printf("标注批次 %d-%d 失败: %v", i, end, err)
			continue
		}
		// Save each annotated entity back
		for _, ae := range annotated {
			_ = m.kbIndex.UpdateCodeEntityMetadata(ae.FilePath, ae.Name, ae.EntityType, ae.Metadata)
		}
		m.annotationCurrent = end
	}
	log.Println("AI 语义标注完成")
	m.annotationCurrent = 0
	m.annotationTotal = 0
	return nil
}

// AnnotateAndSaveEntities 对代码实体进行 AI 语义标注后保存
// 仅在 aiAnnotationEnabled 为 true 时调用 AI 标注；标注失败不影响保存
func (m *MemoryCore) AnnotateAndSaveEntities(filePath string, entities []models.CodeEntity) error {
	if m.aiAnnotationEnabled && len(entities) > 0 {
		annotated, err := m.organizer.AnnotateEntities(entities)
		if err != nil {
			log.Printf("AI 语义标注失败（将保存无标注实体）: %v", err)
		} else {
			entities = annotated
		}
	}
	return m.kbIndex.SaveCodeEntities(filePath, entities)
}

// Close 关闭记忆内核，释放资源
func (m *MemoryCore) Close() error {
	return m.kbIndex.Close()
}
