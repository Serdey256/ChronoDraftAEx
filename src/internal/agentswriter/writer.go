// Package agentswriter 提供 AGENTS.md 文件的自动生成与追加能力
package agentswriter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ChronoDraftAEx/pkg/models"
	"ChronoDraftAEx/pkg/utils"
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

// WriteContent 将任意内容直接写入 AGENTS.md（覆盖模式）
func (w *Writer) WriteContent(content string) error {
	if err := os.MkdirAll(filepath.Dir(w.outputPath), 0o755); err != nil {
		return fmt.Errorf("agentswriter: create dir: %w", err)
	}
	return os.WriteFile(w.outputPath, []byte(content), 0o644)
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

// ---------------------------------------------------------------------------
// Smart AGENTS.md generation (≤1500 tokens, standalone data-driven function)
// ---------------------------------------------------------------------------

// GenerateSmartMarkdown 生成 ≤1500 tokens 的智能 AGENTS.md 内容。
//
// 参数:
//   - commits:    最近的 git commit 记录（按时间降序排列，最新的在前）
//   - entries:    结构化知识条目（按时间降序排列）
//   - dirStructure: 预计算的目录树字符串
//
// 内容包含 4 个章节: 项目概览 / 关键设计决策 / 最近动态 / 项目结构。
// 超出 token 预算时按优先级截断: 项目结构 → 最近动态 → 关键决策 → 项目概览。
func GenerateSmartMarkdown(commits []models.CommitRecord, entries []models.StructuredEntry, dirStructure string) string {
	const maxTokens = 1500

	overviewContent := buildOverview(entries)
	toolsContent := buildToolsSection()
	decisionsContent := buildDecisionsSection(entries, 5)
	recentContent := buildRecentChanges(commits, 3)
	structureContent := buildStructureSection(dirStructure)

	// assembleContent 将当前各章节拼接为完整文档
	assembleContent := func() string {
		var sb strings.Builder
		sb.WriteString("# 项目上下文（≤1500 tokens）\n\n")
		sb.WriteString(overviewContent)
		sb.WriteString(toolsContent)
		sb.WriteString(decisionsContent)
		sb.WriteString(recentContent)
		sb.WriteString(structureContent)
		return sb.String()
	}

	content := assembleContent()

	// 超额截断：从低优先级到高优先级依次缩减
	if utils.EstimateTokens(content) > maxTokens {
		structureContent = utils.TruncateToBudget(structureContent, 200)
		content = assembleContent()
	}
	if utils.EstimateTokens(content) > maxTokens {
		recentContent = buildRecentChanges(commits, 1)
		content = assembleContent()
	}
	if utils.EstimateTokens(content) > maxTokens {
		decisionsContent = buildDecisionsSection(entries, 2)
		content = assembleContent()
	}
	if utils.EstimateTokens(content) > maxTokens {
		overviewContent = utils.TruncateToBudget(overviewContent, 50)
		content = assembleContent()
	}

	return content
}

// buildOverview 构建"项目概览"章节
func buildOverview(entries []models.StructuredEntry) string {
	var sb strings.Builder
	sb.WriteString("## 项目概览\n\n")
	sb.WriteString("由 ChronoDraftAEx 自动维护的项目级记忆。\n\n")
	if len(entries) > 0 {
		sb.WriteString(fmt.Sprintf("**最新条目**: %s\n\n", entries[0].Summary))
	}
	return sb.String()
}

// buildDecisionsSection 构建"关键设计决策"章节，列出 topN 条压缩后的决策
func buildDecisionsSection(entries []models.StructuredEntry, topN int) string {
	var sb strings.Builder
	sb.WriteString("## 关键设计决策\n\n")

	if len(entries) == 0 || topN <= 0 {
		sb.WriteString("暂无设计决策。Agent 可通过 record_change(what, why, problem, files, tags) 记录。\n\n")
		return sb.String()
	}

	if topN > len(entries) {
		topN = len(entries)
	}
	for i := 0; i < topN; i++ {
		compressed := utils.CompressEntry(entries[i], 80)
		sb.WriteString(fmt.Sprintf("- %s\n", compressed))
	}
	sb.WriteString("\n")
	return sb.String()
}

// buildRecentChanges 构建"最近动态"章节，列出 limit 条最近的 commit
func buildRecentChanges(commits []models.CommitRecord, limit int) string {
	var sb strings.Builder
	sb.WriteString("## 最近动态\n\n")

	if len(commits) == 0 || limit <= 0 {
		sb.WriteString("暂无最近的代码变更。\n\n")
		return sb.String()
	}

	if limit > len(commits) {
		limit = len(commits)
	}
	for i := 0; i < limit; i++ {
		c := commits[i]
		hash := c.Hash
		if len(hash) > 8 {
			hash = hash[:8]
		}
		sb.WriteString(fmt.Sprintf("- [`%s`] %s — %s (%s)\n", hash, c.Message, c.Author, c.Timestamp))
	}
	sb.WriteString("\n")
	return sb.String()
}

// buildStructureSection 构建"项目结构"章节
func buildStructureSection(dirStructure string) string {
	var sb strings.Builder
	sb.WriteString("## 项目结构\n\n")

	if dirStructure == "" {
		sb.WriteString("暂无项目结构信息。\n\n")
		return sb.String()
	}

	sb.WriteString(dirStructure)
	sb.WriteString("\n")
	return sb.String()
}

// buildToolsSection 构建"可用工具"章节，告知 Agent 可用的 MCP 工具
func buildToolsSection() string {
	return "## 💡 可用工具\n\n" +
		"> 📖 完整使用指南见项目根目录 `MCP工具指南.md` 和 `致AI-Agent.md`\n\n" +
		"- `get_context(files=\"a.go,b.go\")` — 获取文件的变更历史、代码结构、相关决策和关联文件\n" +
		"- `get_code_entities(files=\"a.go\")` — 获取文件的函数签名、类型定义和导入关系\n" +
		"- `search_knowledge(query)` — 语义搜索历史设计决策和变更记录\n" +
		"- `get_graph(query, top_k)` — 关键词查询关联图谱\n" +
		"- `get_snapshot()` — 获取最新的项目全局上下文快照\n" +
		"- `list_entries(compact=\"true\")` — 分页列出知识条目\n" +
		"- `record_change(what, why, problem, files, tags)` — ⭐ 记录本次修改的设计决策\n\n" +
		"### 核心原则：改完代码必须 record_change\n\n" +
		"其他 Agent 需要通过它理解你的设计意图。不记录 = 知识丢失。详见 `致AI-Agent.md`。\n\n"
}
