package mcp

import (
	"ChronoDraftAEx/internal/memorycore"
	"context"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

func createTestCore(t *testing.T) *memorycore.MemoryCore {
	t.Helper()
	tmpDir := t.TempDir()
	core, err := memorycore.NewMemoryCore(tmpDir, "", "", "")
	if err != nil {
		t.Fatalf("创建测试 MemoryCore 失败: %v", err)
	}
	t.Cleanup(func() { core.Close() })
	return core
}

func TestSearchKnowledgeEmpty(t *testing.T) {
	core := createTestCore(t)
	srv := NewStdioServer(core, t.TempDir())

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"query": "test query",
		"top_k": 10.0,
	}

	result, err := srv.handleSearchKnowledge(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	text := getContentText(result)
	if strings.Contains(strings.ToLower(text), "error") {
		t.Errorf("expected empty result, got error: %s", text)
	}
}

func TestSearchKnowledgeNoQuery(t *testing.T) {
	core := createTestCore(t)
	srv := NewStdioServer(core, t.TempDir())

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := srv.handleSearchKnowledge(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	text := getContentText(result)
	if !strings.Contains(strings.ToLower(text), "query") && !strings.Contains(strings.ToLower(text), "空") {
		t.Errorf("expected query validation error, got: %s", text)
	}
}

func TestGetSnapshot(t *testing.T) {
	core := createTestCore(t)
	srv := NewStdioServer(core, t.TempDir())

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := srv.handleGetSnapshot(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestGetGraph(t *testing.T) {
	core := createTestCore(t)
	srv := NewStdioServer(core, t.TempDir())

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"limit": 50.0,
	}

	result, err := srv.handleGetGraph(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestListEntries(t *testing.T) {
	core := createTestCore(t)
	srv := NewStdioServer(core, t.TempDir())

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"offset": 0.0,
		"limit":  10.0,
	}

	result, err := srv.handleListEntries(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestToolRegistration(t *testing.T) {
	core := createTestCore(t)
	srv := NewStdioServer(core, t.TempDir())
	if srv.server == nil {
		t.Fatal("server is nil")
	}
}

func TestIndexProject(t *testing.T) {
	core := createTestCore(t)
	srv := NewStdioServer(core, t.TempDir())

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := srv.handleIndexProject(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestRecordChange(t *testing.T) {
	core := createTestCore(t)
	srv := NewStdioServer(core, t.TempDir())

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"what":    "添加了用户登录功能",
		"why":     "需要身份认证",
		"problem": "解决未授权访问问题",
		"files":   "src/login.ts,src/auth.ts",
		"tags":    "认证,安全",
	}

	result, err := srv.handleRecordChange(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestRecordChangeMissingParams(t *testing.T) {
	core := createTestCore(t)
	srv := NewStdioServer(core, t.TempDir())

	req := mcpgo.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := srv.handleRecordChange(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	text := getContentText(result)
	if !strings.Contains(text, "必填") && !strings.Contains(strings.ToLower(text), "required") {
		t.Errorf("expected validation error, got: %s", text)
	}
}

func getContentText(result *mcpgo.CallToolResult) string {
	for _, c := range result.Content {
		if tc, ok := c.(mcpgo.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}
