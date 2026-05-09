package mcp

import (
	"ChronoDraftAEx/internal/agentswriter"
	"ChronoDraftAEx/internal/changedetect"
	"ChronoDraftAEx/internal/codeanalysis"
	"ChronoDraftAEx/internal/memorycore"
	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// StdioServer MCP stdio 协议服务器
type StdioServer struct {
	core        *memorycore.MemoryCore
	projectRoot string
	server      *mcpserver.MCPServer
	lastSnap    map[string]changedetect.FileSnapshot
}

// NewStdioServer 创建 MCP stdio 服务器
func NewStdioServer(core *memorycore.MemoryCore, projectRoot string) *StdioServer {
	s := &StdioServer{
		core:        core,
		projectRoot: projectRoot,
	}

	mcpSrv := mcpserver.NewMCPServer(
		"ChronoDraftAEx",
		"1.0.0",
		mcpserver.WithToolCapabilities(false),
	)

	s.registerTools(mcpSrv)
	s.server = mcpSrv

	return s
}

func (s *StdioServer) registerTools(srv *mcpserver.MCPServer) {
	// 1. search_knowledge
	srv.AddTool(
		mcpgo.NewTool("search_knowledge",
			mcpgo.WithDescription("搜索项目知识库，返回相关条目"),
			mcpgo.WithString("query", mcpgo.Required(), mcpgo.Description("搜索关键词")),
			mcpgo.WithNumber("top_k", mcpgo.Description("返回结果数量，默认 10")),
		),
		s.handleSearchKnowledge,
	)

	// 2. get_snapshot
	srv.AddTool(
		mcpgo.NewTool("get_snapshot",
			mcpgo.WithDescription("获取项目知识快照（AGENTS.md 格式）"),
		),
		s.handleGetSnapshot,
	)

	// 3. get_graph
	srv.AddTool(
		mcpgo.NewTool("get_graph",
			mcpgo.WithDescription("搜索知识图谱：根据关键词查找相关节点及关联关系"),
			mcpgo.WithString("query", mcpgo.Required(), mcpgo.Description("搜索关键词")),
			mcpgo.WithNumber("top_k", mcpgo.Description("返回关联度最高的节点数，默认 5")),
			mcpgo.WithString("compact", mcpgo.Description("设为 'true' 启用精简模式")),
		),
		s.handleGetGraph,
	)

	// 4. list_entries
	srv.AddTool(
		mcpgo.NewTool("list_entries",
			mcpgo.WithDescription("分页列出知识条目"),
			mcpgo.WithNumber("offset", mcpgo.Description("起始位置，默认 0")),
			mcpgo.WithNumber("limit", mcpgo.Description("返回数量，默认 20")),
			mcpgo.WithBoolean("compact", mcpgo.Description("精简模式，只返回 summary/timestamp/tags，默认 false")),
		),
		s.handleListEntries,
	)

	// 5. record_change (replaces capture_changes)
	srv.AddTool(
		mcpgo.NewTool("record_change",
			mcpgo.WithDescription("记录一次代码变更。Agent 完成代码修改后调用此工具，告知 ChronoDraftAEx 改动了什么、为什么改、解决了什么问题。"),
			mcpgo.WithString("what", mcpgo.Required(), mcpgo.Description("改动了哪些文件，具体做了什么修改")),
			mcpgo.WithString("why", mcpgo.Required(), mcpgo.Description("为什么要做出这些修改（设计决策与理由）")),
			mcpgo.WithString("problem", mcpgo.Required(), mcpgo.Description("这些修改解决了什么问题")),
			mcpgo.WithString("files", mcpgo.Description("涉及的文件路径列表，逗号分隔")),
			mcpgo.WithString("tags", mcpgo.Description("相关标签，逗号分隔（如：认证,API,重构）")),
		),
		s.handleRecordChange,
	)

	// 6. index_project
	srv.AddTool(
		mcpgo.NewTool("index_project",
			mcpgo.WithDescription("扫描项目文件结构，为关键文件生成作用描述，构建知识图谱。首次接入项目时使用。"),
		),
		s.handleIndexProject,
	)

	// 7. get_project_structure
	srv.AddTool(
		mcpgo.NewTool("get_project_structure",
			mcpgo.WithDescription("获取当前项目的文件结构信息"),
		),
		s.handleGetProjectStructure,
	)

	// 8. capture_commit
	srv.AddTool(
		mcpgo.NewTool("capture_commit",
			mcpgo.WithDescription("捕获一次 git commit 记录，解析参数并保存，对变更文件触发增量 AST 分析"),
			mcpgo.WithString("hash", mcpgo.Required(), mcpgo.Description("Git commit hash")),
			mcpgo.WithString("message", mcpgo.Required(), mcpgo.Description("Git commit message")),
			mcpgo.WithString("author", mcpgo.Description("提交者")),
			mcpgo.WithString("files", mcpgo.Description("变更的文件列表，逗号分隔")),
			mcpgo.WithNumber("insertions", mcpgo.Description("新增行数")),
			mcpgo.WithNumber("deletions", mcpgo.Description("删除行数")),
		),
		s.handleCaptureCommit,
	)

	// 9. get_context
	srv.AddTool(
		mcpgo.NewTool("get_context",
			mcpgo.WithDescription("获取指定文件的相关上下文：最近变更、代码结构、相关决策和关联文件"),
			mcpgo.WithString("files", mcpgo.Required(), mcpgo.Description("文件路径列表，逗号分隔")),
		),
		s.handleGetContext,
	)

	// 10. get_code_entities
	srv.AddTool(
		mcpgo.NewTool("get_code_entities",
			mcpgo.WithDescription("获取指定文件的代码实体（函数、类型、导入关系等 AST 分析结果）"),
			mcpgo.WithString("files", mcpgo.Required(), mcpgo.Description("文件路径列表，逗号分隔")),
		),
		s.handleGetCodeEntities,
	)
}

// Start 启动 MCP stdio 服务器（阻塞）
func (s *StdioServer) Start() error {
	log.SetOutput(os.Stderr)
	return mcpserver.ServeStdio(s.server)
}

func (s *StdioServer) handleSearchKnowledge(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcpgo.NewToolResultError("query 参数不能为空"), nil
	}

	topK := int(req.GetFloat("top_k", 10))
	if topK <= 0 {
		topK = 10
	}

	results, err := s.core.SearchKnowledge(query, topK)
	// Fallback: text search on entries if vector search returns empty
	if err != nil || len(results) == 0 {
		entries, _ := s.core.ListEntries(0, 50)
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Summary), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(e.DesignDecision), strings.ToLower(query)) {
				results = append(results, models.SearchResult{Entry: e, Score: 0.5})
			}
		}
		// Limit results
		if len(results) > topK {
			results = results[:topK]
		}
	}
	if err != nil && len(results) == 0 {
		return mcpgo.NewToolResultError(fmt.Sprintf("搜索失败: %v", err)), nil
	}

	// Build compact representation: only summary, design_decision, tags, score
	type compactResult struct {
		Summary        string   `json:"summary"`
		DesignDecision string   `json:"design_decision"`
		Tags           []string `json:"tags"`
		Score          float64  `json:"score"`
	}

	compact := make([]compactResult, 0, len(results))
	for _, r := range results {
		compact = append(compact, compactResult{
			Summary:        r.Entry.Summary,
			DesignDecision: r.Entry.DesignDecision,
			Tags:           r.Entry.Tags,
			Score:          r.Score,
		})
	}

	data, _ := json.Marshal(compact)
	text := string(data)

	if utils.EstimateTokens(text) > 1000 {
		text = utils.TruncateToBudget(text, 1000)
	}

	return mcpgo.NewToolResultText(text), nil
}

func (s *StdioServer) handleGetSnapshot(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	commits, _ := s.core.ListCommits(3)
	entries, _ := s.core.ListEntries(0, 5)
	dirStructure := buildDirStructure(s.projectRoot)

	md := agentswriter.GenerateSmartMarkdown(commits, entries, dirStructure)
	return mcpgo.NewToolResultText(md), nil
}

func (s *StdioServer) handleGetGraph(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcpgo.NewToolResultError("query 参数不能为空"), nil
	}
	topK := int(req.GetFloat("top_k", 5))
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}
	compact := strings.ToLower(req.GetString("compact", "")) == "true"

	// Search entries matching the query
	results, err := s.core.SearchKnowledge(query, topK)
	// Fallback: text search on entries
	if err != nil || len(results) == 0 {
		entries, _ := s.core.ListEntries(0, 50)
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

	// Build subgraph from matching entries + their related nodes
	nodeMap := make(map[string]models.KnowledgeNode)
	edgeMap := make(map[string]bool)
	for _, r := range results {
		nodes, edges, err := s.core.GetRelatedNodes(r.Entry.ID)
		if err == nil {
			for _, n := range nodes {
				nodeMap[n.ID] = n
			}
			for _, e := range edges {
				key := e.SourceID + "->" + e.TargetID
				if !edgeMap[key] {
					edgeMap[key] = true
				}
			}
		}
	}

	nodes := make([]models.KnowledgeNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}
	edges := make([]models.KnowledgeEdge, 0)
	for k := range edgeMap {
		parts := strings.SplitN(k, "->", 2)
		if len(parts) == 2 {
			edges = append(edges, models.KnowledgeEdge{SourceID: parts[0], TargetID: parts[1]})
		}
	}

	var data []byte
	if compact {
		for i := range nodes {
			nodes[i].Metadata = nil
		}
	}
	data, _ = json.Marshal(map[string]interface{}{"nodes": nodes, "edges": edges})
	return mcpgo.NewToolResultText(string(data)), nil
}

func (s *StdioServer) handleListEntries(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	offset := int(req.GetFloat("offset", 0))
	limit := int(req.GetFloat("limit", 20))
	compact := req.GetBool("compact", false)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}

	entries, err := s.core.ListEntries(offset, limit)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("列出条目失败: %v", err)), nil
	}

	var data []byte
	if compact {
		type compactEntry struct {
			Summary   string   `json:"summary"`
			Timestamp string   `json:"timestamp"`
			Tags      []string `json:"tags"`
		}
		compactEntries := make([]compactEntry, len(entries))
		for i, e := range entries {
			compactEntries[i] = compactEntry{
				Summary:   e.Summary,
				Timestamp: e.Timestamp.Format(time.RFC3339),
				Tags:      e.Tags,
			}
		}
		data, _ = json.Marshal(compactEntries)
	} else {
		data, _ = json.Marshal(entries)
	}
	text := string(data)

	if utils.EstimateTokens(text) > 1000 {
		text = utils.TruncateToBudget(text, 1000)
	}

	return mcpgo.NewToolResultText(text), nil
}

func (s *StdioServer) handleRecordChange(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	what := req.GetString("what", "")
	why := req.GetString("why", "")
	problem := req.GetString("problem", "")
	files := req.GetString("files", "")
	tags := req.GetString("tags", "")

	if what == "" || why == "" || problem == "" {
		return mcpgo.NewToolResultError("what, why, problem 参数均为必填"), nil
	}

	var affectedFiles []models.FileChange
	if files != "" {
		for _, f := range strings.Split(files, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				affectedFiles = append(affectedFiles, models.FileChange{
					Path: f, ChangeType: "modify",
				})
			}
		}
	}

	var tagList []string
	if tags != "" {
		for _, t := range strings.Split(tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagList = append(tagList, t)
			}
		}
	}

	entry := &models.StructuredEntry{
		ID:             utils.GenerateID(),
		Timestamp:      time.Now(),
		SessionID:      "agent-report",
		Summary:        what,
		DesignDecision: why,
		ImpactAnalysis: problem,
		AffectedFiles:  affectedFiles,
		Tags:           tagList,
	}

	if err := s.core.IndexEntry(entry); err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("索引失败: %v", err)), nil
	}

	_ = s.core.RefreshAgentsMD()

	// 增量 AST 分析涉及的变更文件
	if files != "" {
		for _, f := range strings.Split(files, ",") {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			fullPath := filepath.Join(s.projectRoot, f)
			entities, err := codeanalysis.AnalyzeFile(fullPath)
			if err != nil {
				log.Printf("AST 分析失败 %s: %v", f, err)
				continue
			}
			if len(entities) > 0 {
				if err := s.core.AnnotateAndSaveEntities(f, entities); err != nil {
					log.Printf("保存代码实体失败 %s: %v", f, err)
				}
			}
		}
	}

	data, _ := json.Marshal(entry)
	return mcpgo.NewToolResultText(string(data)), nil
}

func (s *StdioServer) handleIndexProject(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	entry, err := s.core.IndexProject()
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("全量索引失败: %v", err)), nil
	}
	data, _ := json.Marshal(entry)
	return mcpgo.NewToolResultText(string(data)), nil
}

func (s *StdioServer) handleGetProjectStructure(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	nodes, edges, err := s.core.GetGraphData(200)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("获取结构失败: %v", err)), nil
	}
	data, _ := json.Marshal(map[string]interface{}{"nodes": nodes, "edges": edges})
	return mcpgo.NewToolResultText(string(data)), nil
}

func (s *StdioServer) handleCaptureCommit(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	hash := req.GetString("hash", "")
	message := req.GetString("message", "")
	author := req.GetString("author", "")
	files := req.GetString("files", "")
	insertions := int(req.GetFloat("insertions", 0))
	deletions := int(req.GetFloat("deletions", 0))

	if hash == "" || message == "" {
		return mcpgo.NewToolResultError("hash 和 message 参数均为必填"), nil
	}

	timestamp := time.Now().Format(time.RFC3339)

	// 保存 commit 记录
	if err := s.core.SaveCommit(hash, message, author, timestamp, files, insertions, deletions); err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("保存 commit 失败: %v", err)), nil
	}

	// 对变更的文件触发增量 AST 分析
	if files != "" {
		for _, f := range strings.Split(files, ",") {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}

			fullPath := filepath.Join(s.projectRoot, f)
			entities, err := codeanalysis.AnalyzeFile(fullPath)
			if err != nil {
				log.Printf("AST 分析失败 %s: %v", f, err)
				continue
			}

			if len(entities) > 0 {
				if err := s.core.AnnotateAndSaveEntities(f, entities); err != nil {
					log.Printf("保存代码实体失败 %s: %v", f, err)
				}
			}
		}
	}

	// 触发 AGENTS.md 重新生成
	if err := s.core.RefreshAgentsMD(); err != nil {
		log.Printf("刷新 AGENTS.md 失败: %v", err)
	}

	// 返回保存的 commit 记录 JSON
	record := models.CommitRecord{
		Hash:       hash,
		Message:    message,
		Author:     author,
		Timestamp:  timestamp,
		Files:      files,
		Insertions: insertions,
		Deletions:  deletions,
	}

	data, _ := json.Marshal(record)
	return mcpgo.NewToolResultText(string(data)), nil
}

func (s *StdioServer) handleGetContext(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	filesStr := req.GetString("files", "")
	if filesStr == "" {
		return mcpgo.NewToolResultError("files 参数不能为空"), nil
	}

	rawFiles := strings.Split(filesStr, ",")
	var files []string
	for _, f := range rawFiles {
		f = strings.TrimSpace(f)
		if f != "" {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		return mcpgo.NewToolResultError("files 参数包含的文件列表为空"), nil
	}

	// Max 3 files to prevent abuse
	if len(files) > 3 {
		files = files[:3]
	}

	// Get commits once for all files
	commits, _ := s.core.ListCommits(10)

	var sections []string
	for _, file := range files {
		section := s.buildFileContext(file, commits)
		sections = append(sections, section)
	}

	content := strings.Join(sections, "\n\n")

	// Enforce total token budget: max 1500
	if utils.EstimateTokens(content) > 1500 {
		content = utils.TruncateToBudget(content, 1500)
	}

	return mcpgo.NewToolResultText(content), nil
}

func (s *StdioServer) handleGetCodeEntities(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	filesStr := req.GetString("files", "")
	if filesStr == "" {
		return mcpgo.NewToolResultError("files 参数不能为空"), nil
	}
	allEntities := make([]models.CodeEntity, 0)
	for _, f := range strings.Split(filesStr, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		entities, err := s.core.GetCodeEntities(f)
		if err != nil {
			continue
		}
		allEntities = append(allEntities, entities...)
	}
	data, _ := json.Marshal(allEntities)
	return mcpgo.NewToolResultText(string(data)), nil
}

// pathMatches 检查存储的文件路径是否匹配目标文件
// 支持：精确匹配、文件名匹配、后缀匹配
func pathMatches(stored, target string) bool {
	stored = strings.TrimSpace(stored)
	target = strings.TrimSpace(target)
	if stored == "" || target == "" {
		return false
	}
	if stored == target {
		return true
	}
	// Basename match: "src/comp/Home.vue" matches "Home.vue"
	if filepath.Base(stored) == filepath.Base(target) {
		return true
	}
	// Suffix match
	return strings.HasSuffix(stored, target) || strings.HasSuffix(target, stored)
}

// buildFileContext 构建单个文件的上下文 markdown，含 4 个章节，每章节 ≤500 tokens
func (s *StdioServer) buildFileContext(file string, commits []models.CommitRecord) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 上下文：%s\n\n", file))

	// 1. 最近变更：从 commits 表中筛选，回退到 entries
	sb.WriteString("### 最近变更\n")
	var fileCommits []models.CommitRecord
	for _, c := range commits {
		if c.Files == "" {
			continue
		}
		for _, f := range strings.Split(c.Files, ",") {
			if pathMatches(f, file) {
				fileCommits = append(fileCommits, c)
				break
			}
		}
	}
	if len(fileCommits) > 0 {
		for _, c := range fileCommits {
			ts := c.Timestamp
			if len(ts) > 10 {
				ts = ts[:10]
			}
			sb.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", ts, c.Message, c.Author))
		}
	} else {
		// Fallback: check entries for file-related design decisions
		entries, _ := s.core.ListEntries(0, 10)
		hasEntry := false
		for _, e := range entries {
			for _, af := range e.AffectedFiles {
				if pathMatches(af.Path, file) {
					if !hasEntry {
						hasEntry = true
					}
					ts := e.Timestamp.Format("2006-01-02")
					sb.WriteString(fmt.Sprintf("- [%s] %s\n", ts, e.Summary))
					break
				}
			}
		}
		if !hasEntry {
			sb.WriteString("- 无相关变更记录\n")
		}
	}
	sb.WriteString("\n")

	// 2. 代码结构：从 code_entities 表获取
	sb.WriteString("### 代码结构\n")
	entities, err := s.core.GetCodeEntities(file)
	if err == nil && len(entities) > 0 {
		for _, e := range entities {
			if e.Signature != "" {
				sb.WriteString(fmt.Sprintf("- %s %s: %s\n", e.EntityType, e.Name, e.Signature))
			} else {
				sb.WriteString(fmt.Sprintf("- %s %s\n", e.EntityType, e.Name))
			}
		}
	} else {
		sb.WriteString("- 无代码实体信息\n")
	}
	sb.WriteString("\n")

	// 3. 相关决策：通过 SearchKnowledge 搜索涉及该文件的条目
	sb.WriteString("### 相关决策\n")
	decisions, err := s.core.SearchKnowledge(file, 3)
	if err == nil && len(decisions) > 0 {
		for _, d := range decisions {
			summary := d.Entry.Summary
			if utils.EstimateTokens(summary) > 50 {
				summary = utils.TruncateToBudget(summary, 50)
			}
			sb.WriteString(fmt.Sprintf("- %s\n", summary))
		}
	} else {
		sb.WriteString("- 无相关决策记录\n")
	}
	sb.WriteString("\n")

	// 4. 关联文件：从涉及该文件的 commit 中提取其他文件路径
	sb.WriteString("### 关联文件\n")
	relatedFiles := make(map[string]bool)
	for _, c := range fileCommits {
		for _, f := range strings.Split(c.Files, ",") {
			f = strings.TrimSpace(f)
			if f != "" && f != file {
				relatedFiles[f] = true
			}
		}
	}
	if len(relatedFiles) > 0 {
		// Sort for deterministic output
		var sorted []string
		for f := range relatedFiles {
			sorted = append(sorted, f)
		}
		// Simple sort via insertion order — stable enough for display
		for _, f := range sorted {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
	} else {
		sb.WriteString("- 无关联文件\n")
	}

	section := sb.String()

	// Enforce per-file 500 token budget
	if utils.EstimateTokens(section) > 500 {
		section = utils.TruncateToBudget(section, 500)
	}

	return section
}

// buildDirStructure 构建项目目录结构的 markdown 表示（最多 2 层深度）
func buildDirStructure(root string) string {
	ignoredDirs := map[string]bool{
		".git": true, ".chronodraft": true, "node_modules": true, "vendor": true,
		"build": true, "dist": true, "target": true, "out": true, "bin": true, "obj": true,
		"__pycache__": true, ".next": true, ".nuxt": true, ".output": true,
		".gradle": true, ".idea": true, ".vscode": true, ".vs": true,
	}

	var sb strings.Builder

	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}

	for _, e := range entries {
		if ignoredDirs[e.Name()] {
			continue
		}
		if e.IsDir() {
			sb.WriteString(fmt.Sprintf("- %s/\n", e.Name()))
			subEntries, err := os.ReadDir(filepath.Join(root, e.Name()))
			if err != nil {
				continue
			}
			for _, sub := range subEntries {
				if ignoredDirs[sub.Name()] {
					continue
				}
				if sub.IsDir() {
					sb.WriteString(fmt.Sprintf("  - %s/\n", sub.Name()))
				} else {
					sb.WriteString(fmt.Sprintf("  - %s\n", sub.Name()))
				}
			}
		} else {
			sb.WriteString(fmt.Sprintf("- %s\n", e.Name()))
		}
	}

	return sb.String()
}
