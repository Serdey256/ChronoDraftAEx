// Package agentswriter 提供 AGENTS.md 文件的自动生成与追加能力
package agentswriter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ChronoDraftAEx/pkg/models"
)

const (
	header = "# ChronoDraftAEx 项目知识快照\n\n" +
		"> 由 ChronoDraftAEx 自动生成，用于为 AI 代理提供项目全局上下文。\n\n" +
		"## 近期变更摘要\n\n"

	footer = "---\n\n*本文件由 ChronoDraftAEx 自动生成*\n"
)

// Writer 负责将结构化知识条目写入 AGENTS.md
type Writer struct {
	outputPath string
}

// NewWriter 创建 Writer，outputPath 默认为 projectRoot/AGENTS.md
func NewWriter(projectRoot string) *Writer {
	return &Writer{
		outputPath: filepath.Join(projectRoot, "AGENTS.md"),
	}
}

// SetOutputPath 覆盖默认输出路径
func (w *Writer) SetOutputPath(path string) {
	w.outputPath = path
}

// Write 将搜索结果全量写入 AGENTS.md（覆盖模式）
func (w *Writer) Write(results []models.SearchResult) error {
	if err := os.MkdirAll(filepath.Dir(w.outputPath), 0o755); err != nil {
		return fmt.Errorf("agentswriter: create dir: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(header)

	for _, r := range results {
		formatEntry(&sb, r.Entry)
	}

	sb.WriteString(footer)

	return os.WriteFile(w.outputPath, []byte(sb.String()), 0o644)
}

// Append 在已有 AGENTS.md 的 footer 行之前追加一条新 entry。
// 若文件不存在，则退化为 Write 单条记录。
func (w *Writer) Append(entry models.StructuredEntry) error {
	content, err := os.ReadFile(w.outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return w.Write([]models.SearchResult{{Entry: entry}})
		}
		return fmt.Errorf("agentswriter: read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	footerIdx := findFooter(lines)

	var entryBlock strings.Builder
	formatEntry(&entryBlock, entry)

	var result string
	if footerIdx >= 0 {
		before := strings.Join(lines[:footerIdx], "\n")
		remaining := strings.Join(lines[footerIdx:], "\n")
		result = before + entryBlock.String() + remaining
	} else {
		result = string(content) + entryBlock.String() + footer
	}

	if err := os.MkdirAll(filepath.Dir(w.outputPath), 0o755); err != nil {
		return fmt.Errorf("agentswriter: create dir: %w", err)
	}
	return os.WriteFile(w.outputPath, []byte(result), 0o644)
}

// findFooter 返回 footer 起始行（"---"）的索引，未找到返回 -1
func findFooter(lines []string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			return i
		}
	}
	return -1
}

// formatEntry 将单条 StructuredEntry 格式化并写入 strings.Builder
func formatEntry(sb *strings.Builder, entry models.StructuredEntry) {
	sb.WriteString(fmt.Sprintf("### %s\n\n", entry.Summary))
	sb.WriteString(fmt.Sprintf("- **时间**: %s\n", entry.Timestamp.Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("- **设计决策**: %s\n", entry.DesignDecision))
	sb.WriteString(fmt.Sprintf("- **影响面**: %s\n", entry.ImpactAnalysis))
	if len(entry.AffectedFiles) > 0 {
		sb.WriteString("- **涉及文件**:\n")
		for _, f := range entry.AffectedFiles {
			sb.WriteString(fmt.Sprintf("  - `%s` (%s)\n", f.Path, f.ChangeType))
		}
	}
	if len(entry.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("- **标签**: %v\n", entry.Tags))
	}
	sb.WriteString("\n")
}
