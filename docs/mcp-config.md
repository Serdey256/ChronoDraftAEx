# ChronoDraftAEx MCP 配置指南

## 概述

ChronoDraftAEx 提供独立的 MCP (Model Context Protocol) 服务器二进制文件 `chronodraft-mcp`，支持通过 stdio 传输与 Claude Desktop、Cursor 等 AI 编辑器连接。

## 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `CHRONODRAFT_PROJECT_ROOT` | 要监控的项目根目录 | 当前工作目录 |
| `CHRONODRAFT_AI_KEY` | AI API Key（用于 capture_changes） | （空） |
| `CHRONODRAFT_AI_BASE` | AI API Base URL | `https://api.openai.com/v1` |
| `CHRONODRAFT_AI_MODEL` | AI 模型名称 | `gpt-4o` |

## Claude Desktop 配置

编辑 `~/.claude/claude_desktop_config.json`（macOS/Linux）或 `%APPDATA%\Claude\claude_desktop_config.json`（Windows）：

```json
{
  "mcpServers": {
    "chronodraft": {
      "command": "/path/to/chronodraft-mcp",
      "env": {
        "CHRONODRAFT_PROJECT_ROOT": "/path/to/your/project",
        "CHRONODRAFT_AI_KEY": "sk-..."
      }
    }
  }
}
```

### Windows 示例

```json
{
  "mcpServers": {
    "chronodraft": {
      "command": "C:\\Users\\YourName\\bin\\chronodraft-mcp.exe",
      "env": {
        "CHRONODRAFT_PROJECT_ROOT": "C:\\Users\\YourName\\Projects\\MyApp",
        "CHRONODRAFT_AI_KEY": "sk-..."
      }
    }
  }
}
```

## Cursor 配置

在项目根目录创建 `.cursor/mcp.json`：

```json
{
  "mcpServers": {
    "chronodraft": {
      "command": "/path/to/chronodraft-mcp",
      "env": {
        "CHRONODRAFT_PROJECT_ROOT": "/path/to/your/project",
        "CHRONODRAFT_AI_KEY": "sk-..."
      }
    }
  }
}
```

## 可用工具

| 工具名 | 说明 | 参数 |
|--------|------|------|
| `search_knowledge` | 搜索项目知识库 | `query` (string, required), `top_k` (int, default 10) |
| `get_snapshot` | 获取项目知识快照 | 无 |
| `get_graph` | 获取知识图谱数据 | `limit` (int, default 100) |
| `list_entries` | 分页列出知识条目 | `offset` (int, default 0), `limit` (int, default 20) |
| `capture_changes` | 捕获文件变动 | `session_id` (string, default "mcp-auto") |

## 构建

```bash
cd src
go build -o chronodraft-mcp ./cmd/mcp
```

## 故障排查

1. **MCP 服务器无响应**: 检查 `CHRONODRAFT_PROJECT_ROOT` 路径是否正确
2. **capture_changes 失败**: 检查 `CHRONODRAFT_AI_KEY` 是否有效
3. **空结果**: 项目可能尚未索引数据，先通过 GUI 应用运行一次"捕获变更"
