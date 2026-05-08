package mcp

import (
	"ChronoDraftAEx/internal/changedetect"
	"ChronoDraftAEx/internal/memorycore"
	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
			mcpgo.WithDescription("获取知识图谱数据（节点和边）"),
			mcpgo.WithNumber("limit", mcpgo.Description("最大节点数，默认 100")),
		),
		s.handleGetGraph,
	)

	// 4. list_entries
	srv.AddTool(
		mcpgo.NewTool("list_entries",
			mcpgo.WithDescription("分页列出知识条目"),
			mcpgo.WithNumber("offset", mcpgo.Description("起始位置，默认 0")),
			mcpgo.WithNumber("limit", mcpgo.Description("返回数量，默认 20")),
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
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("搜索失败: %v", err)), nil
	}

	data, _ := json.Marshal(results)
	return mcpgo.NewToolResultText(string(data)), nil
}

func (s *StdioServer) handleGetSnapshot(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	results, err := s.core.SearchKnowledge("最新变更", 10)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("获取快照失败: %v", err)), nil
	}

	md := GenerateAgentsMarkdown(results)
	return mcpgo.NewToolResultText(md), nil
}

func (s *StdioServer) handleGetGraph(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	limit := int(req.GetFloat("limit", 100))
	if limit <= 0 {
		limit = 100
	}

	nodes, edges, err := s.core.GetGraphData(limit)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("获取图谱失败: %v", err)), nil
	}

	data, _ := json.Marshal(map[string]interface{}{"nodes": nodes, "edges": edges})
	return mcpgo.NewToolResultText(string(data)), nil
}

func (s *StdioServer) handleListEntries(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	offset := int(req.GetFloat("offset", 0))
	limit := int(req.GetFloat("limit", 20))
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

	data, _ := json.Marshal(entries)
	return mcpgo.NewToolResultText(string(data)), nil
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
