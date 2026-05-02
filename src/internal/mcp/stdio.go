package mcp

import (
	"ChronoDraftAEx/internal/changedetect"
	"ChronoDraftAEx/internal/memorycore"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

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

	// 5. capture_changes
	srv.AddTool(
		mcpgo.NewTool("capture_changes",
			mcpgo.WithDescription("捕获文件变动并生成结构化知识条目"),
			mcpgo.WithString("session_id", mcpgo.Description("会话 ID，默认 mcp-auto")),
		),
		s.handleCaptureChanges,
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

func (s *StdioServer) handleCaptureChanges(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	sessionID := req.GetString("session_id", "mcp-auto")

	if s.lastSnap == nil {
		loaded, err := changedetect.LoadLastSnap(s.projectRoot)
		if err == nil && loaded != nil {
			s.lastSnap = loaded
		} else {
			detector := changedetect.NewDetector(s.projectRoot)
			s.lastSnap, _ = detector.ScanSnapshot()
		}
	}

	newSnap, err := changedetect.NewDetector(s.projectRoot).ScanSnapshot()
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("扫描文件失败: %v", err)), nil
	}

	entry, err := s.core.CaptureAndIndex(s.lastSnap, newSnap, sessionID)
	if err != nil {
		return mcpgo.NewToolResultError(fmt.Sprintf("捕获变更失败: %v", err)), nil
	}

	s.lastSnap = newSnap
	_ = changedetect.SaveLastSnap(s.projectRoot, s.lastSnap)

	data, _ := json.Marshal(entry)
	return mcpgo.NewToolResultText(string(data)), nil
}
