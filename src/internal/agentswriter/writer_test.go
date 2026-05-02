package agentswriter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ChronoDraftAEx/pkg/models"
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
