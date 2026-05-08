package mcp

import (
	"ChronoDraftAEx/pkg/models"
	"fmt"
)

func GenerateAgentsMarkdown(results []models.SearchResult) string {
	md := "# ChronoDraftAEx 项目知识快照\n\n"
	md += "> 由 ChronoDraftAEx 自动生成，用于为 AI 代理提供项目全局上下文。\n\n"

	if len(results) == 0 {
		md += "## 📊 状态\n\n"
		md += "当前知识库中暂无变更记录。Agent 可通过 `record_change` 工具上报代码变更。\n\n"
		md += "### 可用 MCP 工具\n\n"
		md += "- `record_change` — 记录代码变更（必填：what, why, problem）\n"
		md += "- `search_knowledge` — 语义搜索知识库\n"
		md += "- `get_graph` — 获取知识图谱\n"
		md += "- `get_project_structure` — 获取项目文件结构\n"
		md += "- `list_entries` — 列出知识条目\n"
		md += "- `index_project` — 全量索引项目（首次接入时使用）\n\n"
		md += "### 💡 使用提示\n\n"
		md += "Agent 完成代码修改后，请调用 `record_change` 告知 ChronoDraftAEx：\n"
		md += "```\n"
		md += "record_change(\n"
		md += "  what: \"添加了用户认证模块，包括登录/注册页面和 JWT token 管理\",\n"
		md += "  why: \"为了支持多用户系统，需要独立的认证流程\",\n"
		md += "  problem: \"原有系统没有用户认证机制，任何人可以直接访问管理页面\",\n"
		md += "  files: \"src/auth/login.ts,src/auth/register.ts,src/middleware/jwt.ts\",\n"
		md += "  tags: \"认证,安全,JWT\"\n"
		md += ")\n"
		md += "```\n"
	} else {
		md += "## 近期变更摘要\n\n"
		for _, r := range results {
			md += fmt.Sprintf("### %s\n\n", r.Entry.Summary)
			md += fmt.Sprintf("- **时间**: %s\n", r.Entry.Timestamp.Format("2006-01-02 15:04"))
			md += fmt.Sprintf("- **设计决策**: %s\n", r.Entry.DesignDecision)
			md += fmt.Sprintf("- **问题**: %s\n", r.Entry.ImpactAnalysis)
			if len(r.Entry.AffectedFiles) > 0 {
				md += "- **涉及文件**:\n"
				for _, f := range r.Entry.AffectedFiles {
					md += fmt.Sprintf("  - `%s`\n", f.Path)
				}
			}
			if len(r.Entry.Tags) > 0 {
				md += fmt.Sprintf("- **标签**: %v\n", r.Entry.Tags)
			}
			md += "\n"
		}
	}

	md += "---\n\n"
	md += "*本文件由 ChronoDraftAEx 自动生成，请勿手动修改。*\n"
	return md
}
