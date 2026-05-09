package agentswriter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
)

func fixedTime() time.Time {
	return time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
}

func sampleEntry(summary string) models.StructuredEntry {
	return models.StructuredEntry{
		ID:             "test-id",
		Timestamp:      fixedTime(),
		Summary:        summary,
		DesignDecision: "采用模块化设计",
		ImpactAnalysis: "影响 auth 模块",
		AffectedFiles: []models.FileChange{
			{Path: "src/auth/login.go", ChangeType: "modify"},
		},
		Tags: []string{"auth", "refactor"},
	}
}

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	results := []models.SearchResult{
		{Entry: sampleEntry("重构登录模块"), Score: 0.95},
	}

	if err := w.Write(results); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "# ChronoDraftAEx 项目知识快照") {
		t.Error("missing header")
	}
	if !strings.Contains(content, "重构登录模块") {
		t.Error("missing summary")
	}
	if !strings.Contains(content, "采用模块化设计") {
		t.Error("missing design decision")
	}
	if !strings.Contains(content, "影响 auth 模块") {
		t.Error("missing impact analysis")
	}
	if !strings.Contains(content, "src/auth/login.go") {
		t.Error("missing affected file")
	}
	if !strings.Contains(content, "---") {
		t.Error("missing footer separator")
	}
	if !strings.Contains(content, "*本文件由 ChronoDraftAEx 自动生成*") {
		t.Error("missing footer text")
	}
}

func TestAppend(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	initial := []models.SearchResult{
		{Entry: sampleEntry("初始条目")},
	}
	if err := w.Write(initial); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	newEntry := sampleEntry("追加条目")
	newEntry.ID = "test-id-2"
	if err := w.Append(newEntry); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "初始条目") {
		t.Error("missing initial entry after append")
	}
	if !strings.Contains(content, "追加条目") {
		t.Error("missing appended entry")
	}

	separatorIdx := strings.Index(content, "---")
	footerIdx := strings.Index(content, "*本文件由 ChronoDraftAEx 自动生成*")
	if separatorIdx < 0 || footerIdx < separatorIdx {
		t.Error("footer text should appear after ---")
	}

	appendedIdx := strings.Index(content, "追加条目")
	if appendedIdx > separatorIdx {
		t.Error("appended entry should appear before --- separator")
	}
}

func TestWriteEmptyResults(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	if err := w.Write(nil); err != nil {
		t.Fatalf("Write with nil results failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# ChronoDraftAEx 项目知识快照") {
		t.Error("missing header for empty results")
	}
	if !strings.Contains(content, "---") {
		t.Error("missing footer for empty results")
	}
}

func TestCustomOutputPath(t *testing.T) {
	dir := t.TempDir()
	customPath := filepath.Join(dir, "custom", "OUTPUT.md")
	w := NewWriter(dir)
	w.SetOutputPath(customPath)

	if err := w.Write(nil); err != nil {
		t.Fatalf("Write to custom path failed: %v", err)
	}

	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Fatal("file not created at custom path")
	}

	data, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !strings.Contains(string(data), "# ChronoDraftAEx 项目知识快照") {
		t.Error("custom path file missing header")
	}
}

// ---------------------------------------------------------------------------
// GenerateSmartMarkdown tests
// ---------------------------------------------------------------------------

func TestGenerateSmartMarkdown_Basic(t *testing.T) {
	commits := []models.CommitRecord{
		{Hash: "abc123def456", Message: "添加用户认证模块", Author: "Alice", Timestamp: "2025-06-15"},
		{Hash: "def789abc012", Message: "修复登录页面样式", Author: "Bob", Timestamp: "2025-06-14"},
		{Hash: "ghi345jkl678", Message: "重构数据库层", Author: "Alice", Timestamp: "2025-06-13"},
	}
	entries := []models.StructuredEntry{
		sampleEntry("添加用户认证模块"),
		sampleEntry("修复登录页面样式"),
		sampleEntry("重构数据库层"),
		sampleEntry("性能优化"),
		sampleEntry("更新文档"),
	}
	dirStructure := "src/\n  main.go\n  internal/\n    auth/\n      login.go\n"

	md := GenerateSmartMarkdown(commits, entries, dirStructure)

	// Verify all 4 sections present
	if !strings.Contains(md, "# 项目上下文") {
		t.Error("missing document title")
	}
	if !strings.Contains(md, "## 项目概览") {
		t.Error("missing 项目概览 section")
	}
	if !strings.Contains(md, "## 关键设计决策") {
		t.Error("missing 关键设计决策 section")
	}
	if !strings.Contains(md, "## 最近动态") {
		t.Error("missing 最近动态 section")
	}
	if !strings.Contains(md, "## 项目结构") {
		t.Error("missing 项目结构 section")
	}

	// Verify commit hash appears (truncated to 8 chars)
	if !strings.Contains(md, "abc123de") {
		t.Error("missing commit hash")
	}

	// Verify token budget
	tokens := utils.EstimateTokens(md)
	if tokens > 1500 {
		t.Errorf("token count %d exceeds limit 1500", tokens)
	}
}

func TestGenerateSmartMarkdown_Empty(t *testing.T) {
	md := GenerateSmartMarkdown(nil, nil, "")

	if !strings.Contains(md, "# 项目上下文") {
		t.Error("missing document title")
	}
	if !strings.Contains(md, "## 项目概览") {
		t.Error("missing 项目概览 section")
	}
	if !strings.Contains(md, "暂无最近的代码变更") {
		t.Error("should show empty message for commits")
	}
	if !strings.Contains(md, "暂无设计决策") {
		t.Error("should show empty message for decisions")
	}
	if !strings.Contains(md, "暂无项目结构信息") {
		t.Error("should show empty message for structure")
	}

	tokens := utils.EstimateTokens(md)
	if tokens > 1500 {
		t.Errorf("token count %d exceeds limit 1500", tokens)
	}
}

func TestGenerateSmartMarkdown_OverBudget(t *testing.T) {
	// Force truncation by providing a very large directory structure
	largeDir := strings.Repeat("  some/deep/nested/directory/\n    file.go\n", 100)
	entries := []models.StructuredEntry{
		sampleEntry("条目1"),
		sampleEntry("条目2"),
		sampleEntry("条目3"),
		sampleEntry("条目4"),
		sampleEntry("条目5"),
	}
	commits := []models.CommitRecord{
		{Hash: "abc123def", Message: "提交一", Author: "A", Timestamp: "2025-06-15"},
		{Hash: "ghi456jkl", Message: "提交二", Author: "B", Timestamp: "2025-06-14"},
		{Hash: "mno789pqr", Message: "提交三", Author: "C", Timestamp: "2025-06-13"},
	}

	md := GenerateSmartMarkdown(commits, entries, largeDir)
	tokens := utils.EstimateTokens(md)
	if tokens > 1500 {
		t.Errorf("token count %d exceeds limit 1500 after truncation", tokens)
	}

	// All 4 sections must still be present after truncation
	if !strings.Contains(md, "## 项目概览") {
		t.Error("missing 项目概览 section after truncation")
	}
	if !strings.Contains(md, "## 关键设计决策") {
		t.Error("missing 关键设计决策 section after truncation")
	}
	if !strings.Contains(md, "## 最近动态") {
		t.Error("missing 最近动态 section after truncation")
	}
	if !strings.Contains(md, "## 项目结构") {
		t.Error("missing 项目结构 section after truncation")
	}
}

func TestGenerateSmartMarkdown_LessData(t *testing.T) {
	// Test with fewer entries and commits than the display limits
	commits := []models.CommitRecord{
		{Hash: "abc", Message: "仅有一次提交", Author: "Dev", Timestamp: "2025-06-15"},
	}
	entries := []models.StructuredEntry{
		sampleEntry("仅有一条条目"),
	}

	md := GenerateSmartMarkdown(commits, entries, "src/\n  main.go\n")

	if !strings.Contains(md, "仅有一次提交") {
		t.Error("missing single commit")
	}
	if !strings.Contains(md, "仅有一条条目") {
		t.Error("missing single entry")
	}

	tokens := utils.EstimateTokens(md)
	if tokens > 1500 {
		t.Errorf("token count %d exceeds limit 1500", tokens)
	}
}
