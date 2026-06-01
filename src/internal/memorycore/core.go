// Package memorycore 是系统的记忆索引内核
// 负责整合变动检测、AI 摘要生成和知识库索引，提供统一的记忆管理接口
package memorycore

import (
	"ChronoDraftAEx/internal/changedetect"
	"ChronoDraftAEx/internal/changeorganize"
	"ChronoDraftAEx/internal/codeanalysis"
	"ChronoDraftAEx/internal/githook"
	"ChronoDraftAEx/internal/kbindex"
	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MemoryCore 记忆内核
type MemoryCore struct {
	detector            *changedetect.Detector
	organizer           *changeorganize.Organizer
	kbIndex             *kbindex.KBIndex
	ctx                 context.Context
	projectRoot         string
	aiAnnotationEnabled bool
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
		embeddingModel = "text-embedding-3-small"
	}
	kbi.SetEmbeddingConfig(aiAPIKey, aiBaseURL, embeddingModel)

	ctx := context.Background()
	if err := kbi.Init(ctx); err != nil {
		return nil, fmt.Errorf("初始化知识库失败: %w", err)
	}

	return &MemoryCore{
		detector:            detector,
		organizer:           organizer,
		kbIndex:             kbi,
		ctx:                 ctx,
		projectRoot:         projectRoot,
		aiAnnotationEnabled: false,
	}, nil
}

// CaptureAndIndex 捕获变动、生成摘要并索引到知识库（完整工作流）
func (m *MemoryCore) CaptureAndIndex(oldSnap, newSnap map[string]changedetect.FileSnapshot, sessionID string) (*models.StructuredEntry, error) {
	record := m.detector.DetectChanges(oldSnap, newSnap, sessionID)
	if len(record.Changes) == 0 {
		return nil, fmt.Errorf("未检测到文件变动")
	}

	entry, err := m.organizer.Organize(record)
	if err != nil {
		return nil, fmt.Errorf("生成结构化摘要失败: %w", err)
	}

	if err := m.kbIndex.IndexEntry(m.ctx, entry); err != nil {
		return nil, fmt.Errorf("索引知识库失败: %w", err)
	}

	return entry, nil
}

// IndexProject 全量索引（兼容别名，实际走脚手架流程）
func (m *MemoryCore) IndexProject() (*models.StructuredEntry, error) {
	return m.ScaffoldProject()
}

// ScaffoldProject 轻量脚手架：零 AI 成本的结构扫描 + 目录层级 + Git 历史导入
func (m *MemoryCore) ScaffoldProject() (*models.StructuredEntry, error) {
	m.indexPhase = "扫描文件"
	snapshot, err := m.detector.ScanSnapshot()
	if err != nil {
		m.indexPhase = ""
		return nil, fmt.Errorf("扫描项目文件失败: %w", err)
	}

	var filePaths []string
	for path := range snapshot {
		filePaths = append(filePaths, path)
	}
	if len(filePaths) == 0 {
		m.indexPhase = ""
		return nil, fmt.Errorf("项目中未发现任何文件")
	}

	m.indexPhase = "构建目录层级"
	log.Printf("构建目录层级：%d 个文件...", len(filePaths))
	if err := m.kbIndex.UpdateDirectoryHierarchy(m.ctx, filePaths); err != nil {
		log.Printf("警告: 构建目录层级失败: %v", err)
	}

	m.indexPhase = "AST代码分析"
	if err := codeanalysis.AnalyzeProject(m.projectRoot, m.AnnotateAndSaveEntities); err != nil {
		log.Printf("警告: AST 代码分析失败: %v", err)
	}

	m.indexPhase = "构建导入关系"
	if err := m.refreshAllImportEdges(); err != nil {
		log.Printf("警告: 构建 IMPORTS 关系失败: %v", err)
	}

	m.indexPhase = "安装GitHook"
	binaryPath := os.Getenv("CHRONODRAFT_BINARY_PATH")
	if binaryPath != "" {
		if err := githook.InstallHook(m.projectRoot, binaryPath); err != nil {
			log.Printf("警告: 安装 Git Hook 失败: %v", err)
		}
	} else {
		log.Println("提示: 环境变量 CHRONODRAFT_BINARY_PATH 未设置，跳过 Git Hook 安装")
	}

	m.indexPhase = "导入Git历史"
	if err := m.ImportGitHistory(50); err != nil {
		log.Printf("警告: 导入 Git 历史失败: %v", err)
	}

	m.indexPhase = ""
	return &models.StructuredEntry{
		ID:        utils.GenerateID(),
		Timestamp: time.Now(),
		SessionID: "scaffold",
		Summary:   fmt.Sprintf("项目脚手架构建完成：扫描了 %d 个文件", len(filePaths)),
	}, nil
}

// ImportGitHistory 导入最近 N 条 git commit 记录（仅元数据，不做 AST 分析）
func (m *MemoryCore) ImportGitHistory(limit int) error {
	cmd := exec.Command("git", "log", "--oneline", "--format=%H|%s|%an|%ai", "-n", strconv.Itoa(limit))
	cmd.Dir = m.projectRoot
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git log 执行失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		hash := strings.TrimSpace(parts[0])
		message := strings.TrimSpace(parts[1])
		author := strings.TrimSpace(parts[2])
		timestamp := strings.TrimSpace(parts[3])
		files, ins, dels := m.readCommitFilesAndStats(hash)
		if err := m.kbIndex.SaveCommit(hash, message, author, timestamp, files, ins, dels); err != nil {
			log.Printf("保存 commit %s 失败: %v", hash, err)
		}
	}

	log.Printf("已导入 %d 条 git commit 记录", len(lines))
	return nil
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
	if nodes, edges, ok := m.searchStructureGraph(query, topK); ok {
		return nodes, edges, nil
	}

	results, err := m.kbIndex.Search(m.ctx, query, topK)
	if err != nil || len(results) == 0 {
		entries, _ := m.kbIndex.ListEntries(0, 50)
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Summary), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(e.DesignDecision), strings.ToLower(query)) {
				results = append(results, models.SearchResult{Entry: e, Score: 0.5})
			}
		}
		if len(results) > topK {
			results = results[:topK]
		}
	}
	nodeMap := make(map[string]models.KnowledgeNode)
	edgeMap := make(map[string]models.KnowledgeEdge)
	for _, r := range results {
		nodes, edges, _ := m.kbIndex.QueryRelated(m.ctx, r.Entry.ID)
		for _, n := range nodes {
			nodeMap[n.ID] = n
		}
		for _, e := range edges {
			k := e.SourceID + "|" + e.TargetID + "|" + e.Relation
			edgeMap[k] = e
		}
	}
	nodes := make([]models.KnowledgeNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}
	var edgeList []models.KnowledgeEdge
	for _, e := range edgeMap {
		edgeList = append(edgeList, e)
	}
	return nodes, edgeList, nil
}

func (m *MemoryCore) searchStructureGraph(query string, topK int) ([]models.KnowledgeNode, []models.KnowledgeEdge, bool) {
	if strings.TrimSpace(query) == "" {
		return nil, nil, false
	}
	matches, err := m.kbIndex.SearchNodesByLabel(m.ctx, query, topK)
	if err != nil || len(matches) == 0 {
		return nil, nil, false
	}

	nodeMap := make(map[string]models.KnowledgeNode)
	edgeMap := make(map[string]models.KnowledgeEdge)
	for _, match := range matches {
		var nodes []models.KnowledgeNode
		var edges []models.KnowledgeEdge
		if match.Type == "directory" || match.Type == "module" {
			dirPath := strings.TrimPrefix(match.ID, "dir:")
			nodes, edges, err = m.kbIndex.GetModuleGraph(m.ctx, dirPath, topK*20)
		} else {
			nodes, edges, err = m.kbIndex.QueryRelated(m.ctx, match.ID)
		}
		if err != nil {
			continue
		}
		for _, n := range nodes {
			nodeMap[n.ID] = n
		}
		for _, e := range edges {
			k := e.SourceID + "|" + e.TargetID + "|" + e.Relation
			edgeMap[k] = e
		}
	}
	if len(nodeMap) == 0 {
		return nil, nil, false
	}
	nodes := make([]models.KnowledgeNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}
	edges := make([]models.KnowledgeEdge, 0, len(edgeMap))
	for _, e := range edgeMap {
		edges = append(edges, e)
	}
	return nodes, edges, true
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

// GenerateAgentsMD 兼容保留：纯 MCP 模式下不再落盘生成文件
func (m *MemoryCore) GenerateAgentsMD() error {
	return nil
}

// IndexEntry 直接将结构化条目索引到知识库（不经过 AI 处理）
func (m *MemoryCore) IndexEntry(entry *models.StructuredEntry) error {
	return m.kbIndex.IndexEntry(m.ctx, entry)
}

// RefreshAgentsMD 兼容保留：纯 MCP 模式下不再刷新文件
func (m *MemoryCore) RefreshAgentsMD() error {
	return nil
}

func (m *MemoryCore) refreshAllImportEdges() error {
	files, err := m.kbIndex.ListCodeEntityFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	existing := make(map[string]bool, len(files))
	for _, f := range files {
		existing[strings.ReplaceAll(f, "\\", "/")] = true
	}
	modulePath := readGoModulePath(m.projectRoot)
	imports := make(map[string][]string)
	for _, file := range files {
		imports[file] = []string{}
		entities, err := m.kbIndex.GetCodeEntities(file)
		if err != nil {
			continue
		}
		seen := make(map[string]bool)
		for _, e := range entities {
			if e.EntityType != "import" {
				continue
			}
			if target, ok := resolveImportTarget(file, e.Name, existing, modulePath); ok && target != file && !seen[target] {
				imports[file] = append(imports[file], target)
				seen[target] = true
			}
		}
	}
	return m.kbIndex.UpdateImportEdges(m.ctx, imports)
}

func (m *MemoryCore) readCommitFilesAndStats(hash string) (string, int, int) {
	filesCmd := exec.Command("git", "show", "--format=", "--name-only", "-M", hash)
	filesCmd.Dir = m.projectRoot
	filesOut, err := filesCmd.Output()
	if err != nil {
		return "", 0, 0
	}
	var files []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(filesOut)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		line = strings.ReplaceAll(line, "\\", "/")
		if !seen[line] {
			seen[line] = true
			files = append(files, line)
		}
	}

	statsCmd := exec.Command("git", "show", "--format=", "--numstat", "-M", hash)
	statsCmd.Dir = m.projectRoot
	statsOut, err := statsCmd.Output()
	if err != nil {
		return strings.Join(files, ","), 0, 0
	}
	ins, dels := 0, 0
	statsScanner := bufio.NewScanner(strings.NewReader(string(statsOut)))
	for statsScanner.Scan() {
		parts := strings.Fields(statsScanner.Text())
		if len(parts) < 3 {
			continue
		}
		if parts[0] != "-" {
			if n, err := strconv.Atoi(parts[0]); err == nil {
				ins += n
			}
		}
		if parts[1] != "-" {
			if n, err := strconv.Atoi(parts[1]); err == nil {
				dels += n
			}
		}
	}

	return strings.Join(files, ","), ins, dels
}

func readGoModulePath(projectRoot string) string {
	data, err := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func resolveImportTarget(sourceFile, importName string, existing map[string]bool, modulePath string) (string, bool) {
	importName = strings.TrimSpace(strings.Trim(importName, `"'`))
	if importName == "" {
		return "", false
	}

	addCandidates := func(base string) []string {
		base = strings.ReplaceAll(base, "\\", "/")
		base = strings.TrimPrefix(base, "./")
		base = strings.TrimPrefix(base, "/")
		var candidates []string
		exts := []string{"", ".ts", ".tsx", ".js", ".jsx", ".vue", ".py", ".go", ".java", ".rs", ".kt", ".kts", ".c", ".cpp", ".cc", ".h", ".hpp", ".cs"}
		for _, ext := range exts {
			candidates = append(candidates, base+ext)
		}
		for _, ext := range exts[1:] {
			candidates = append(candidates, strings.TrimSuffix(base, "/")+"/index"+ext)
		}
		return candidates
	}

	var bases []string
	if strings.HasPrefix(importName, "./") || strings.HasPrefix(importName, "../") {
		bases = append(bases, filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(sourceFile), importName))))
	} else {
		bases = append(bases, strings.ReplaceAll(importName, "\\", "/"))
		if modulePath != "" && strings.HasPrefix(importName, modulePath+"/") {
			bases = append(bases, strings.TrimPrefix(importName, modulePath+"/"))
		}
	}

	for _, base := range bases {
		for _, candidate := range addCandidates(base) {
			if existing[candidate] {
				return candidate, true
			}
		}
	}
	return "", false
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

// SetAgentsMDEnabled 兼容保留：纯 MCP 模式下无效果
func (m *MemoryCore) SetAgentsMDEnabled(enabled bool) {}

// IsAgentsMDEnabled 兼容保留：纯 MCP 模式下始终为 false
func (m *MemoryCore) IsAgentsMDEnabled() bool {
	return false
}

// refreshSmartAgentsMD 兼容保留：纯 MCP 模式下不再写入任何文件
func (m *MemoryCore) refreshSmartAgentsMD() {}

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

// GetIndexPhase 返回当前脚手架阶段
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
	if err := m.kbIndex.SaveCodeEntities(filePath, entities); err != nil {
		return err
	}
	if err := m.refreshAllImportEdges(); err != nil {
		log.Printf("刷新 IMPORTS 关系失败 %s: %v", filePath, err)
	}
	return nil
}

// Close 关闭记忆内核，释放资源
func (m *MemoryCore) Close() error {
	return m.kbIndex.Close()
}
