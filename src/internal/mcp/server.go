// Package mcp 实现 MCP (Model Context Protocol) 服务器
// 通过标准协议与 OpenClaw 等 AI 代理安全交互，提供项目知识快照服务
package mcp

import (
	"ChronoDraftAEx/internal/memorycore"
	"ChronoDraftAEx/pkg/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// MCPServer MCP 协议服务器
type MCPServer struct {
	core   *memorycore.MemoryCore
	server *http.Server
	addr   string
}

// NewMCPServer 创建 MCP 服务器
func NewMCPServer(core *memorycore.MemoryCore, addr string) *MCPServer {
	if addr == "" {
		addr = ":8787"
	}
	m := &MCPServer{
		core: core,
		addr: addr,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp/snapshot", m.handleSnapshot)
	mux.HandleFunc("/mcp/search", m.handleSearch)
	mux.HandleFunc("/mcp/graph", m.handleGraph)
	mux.HandleFunc("/mcp/entries", m.handleEntries)
	mux.HandleFunc("/mcp/health", m.handleHealth)
	m.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return m
}

// Start 启动 MCP 服务器（非阻塞）
func (m *MCPServer) Start() error {
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("MCP 服务器异常退出: %v\n", err)
		}
	}()
	fmt.Printf("MCP 服务器已启动，监听 %s\n", m.addr)
	return nil
}

// Stop 关闭 MCP 服务器
func (m *MCPServer) Stop(ctx context.Context) error {
	return m.server.Shutdown(ctx)
}

// handleSnapshot 返回轻量级的 AGENTS.md 风格项目知识快照
func (m *MCPServer) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	results, err := m.core.SearchKnowledge("最新变更", 10)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	md := GenerateAgentsMarkdown(results)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"snapshot_type": "agents_md_style",
		"markdown":      md,
		"entries":       results,
		"generated_at":  "now",
	})
}

// handleSearch 语义搜索知识库
func (m *MCPServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var req struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.TopK == 0 {
		req.TopK = 5
	}

	results, err := m.core.SearchKnowledge(req.Query, req.TopK)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleGraph 返回知识图谱数据
func (m *MCPServer) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	nodes, edges, err := m.core.GetGraphData(limit)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.GraphData{Nodes: nodes, Edges: edges})
}

// handleEntries 列出知识条目
func (m *MCPServer) handleEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")
	offset, limit := 0, 20
	if o, err := strconv.Atoi(offsetStr); err == nil {
		offset = o
	}
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	entries, err := m.core.ListEntries(offset, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// handleHealth 健康检查端点
func (m *MCPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"mcp":    "ChronoDraftAEx",
	})
}

// GenerateAgentsMarkdown 生成 AGENTS.md 风格的 Markdown 快照文件内容
func GenerateAgentsMarkdown(results []models.SearchResult) string {
	md := "# ChronoDraftAEx 项目知识快照\n\n"
	md += "> 由 ChronoDraftAEx 自动生成，用于为 AI 代理提供项目全局上下文。\n\n"
	md += "## 近期变更摘要\n\n"

	for _, r := range results {
		md += fmt.Sprintf("### %s\n\n", r.Entry.Summary)
		md += fmt.Sprintf("- **时间**: %s\n", r.Entry.Timestamp.Format("2006-01-02 15:04"))
		md += fmt.Sprintf("- **设计决策**: %s\n", r.Entry.DesignDecision)
		md += fmt.Sprintf("- **影响面**: %s\n", r.Entry.ImpactAnalysis)
		if len(r.Entry.AffectedFiles) > 0 {
			md += "- **涉及文件**:\n"
			for _, f := range r.Entry.AffectedFiles {
				md += fmt.Sprintf("  - `%s` (%s)\n", f.Path, f.ChangeType)
			}
		}
		if len(r.Entry.Tags) > 0 {
			md += fmt.Sprintf("- **标签**: %v\n", r.Entry.Tags)
		}
		md += "\n"
	}

	md += "---\n\n"
	md += "*本文件由 ChronoDraftAEx MCP 服务动态生成，请勿手动修改。*\n"
	return md
}
