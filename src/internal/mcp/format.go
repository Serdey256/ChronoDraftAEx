package mcp

import (
	"ChronoDraftAEx/pkg/models"
	"fmt"
)

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
