# ChronoDraftAEx

> AI 编码时代项目知识的「黑匣子」— 结构化记忆与上下文管理系统

## 💡 产品愿景

AI 代理进行多轮开发时，上下文窗口爆满、项目记忆丢失是核心痛点。ChronoDraftAEx 采用 **Agent 驱动的知识管理**模式：Agent 完成代码修改后，主动告知 ChronoDraftAEx"改了什么、为什么改、解决了什么问题"，ChronoDraftAEx 将其转化为**可索引、可追溯、可重用的工程资产**。

知识图谱记录项目的**文件结构**（目录/文件节点 + CONTAINS 边），而非文件名间的关系，让 AI 能理解项目架构而非零散文件名。

## 🏗️ 双轨架构

```
┌─────────────────────────┐     ┌──────────────────────────┐
│   Wails 桌面应用          │     │  MCP stdio 服务器         │
│   (人类用户)              │     │  (AI 代理)                │
├─────────────────────────┤     ├──────────────────────────┤
│ • 仪表盘/知识图谱          │     │ • 7 个 MCP 工具           │
│ • 项目管理/快照            │     │ • record_change (核心)     │
│ • 项目结构扫描 + 依赖检测    │     │ • index_project          │
│ • Agent 驱动变更展示        │     │ • search/get/list/结构    │
└───────────┬─────────────┘     └───────────┬──────────────┘
            │                               │
            └───────────────┬───────────────┘
                            ▼
                ┌─────────────────────┐
                │   SQLite 共享存储    │
                │ .chronodraft/       │
                │  元数据 + 图谱 + 快照  │
                └─────────────────────┘
```

## ⚙️ 技术栈

| 层级 | 技术 | 说明 |
|:---|:---|:---|
| 桌面框架 | Wails v3 (Go + Web) | 跨平台、轻量级 |
| 后端语言 | Go 1.25.5 | 单二进制、高性能 |
| 前端框架 | React + TypeScript + D3.js | 组件化 + 图谱可视化 |
| AI 能力 | 云端 API (OpenAI 兼容) | 仅需 API Key |
| 存储 | SQLite (modernc.org/sqlite) | 纯 Go 实现，无需 CGO |
| MCP SDK | mark3labs/mcp-go v0.50 | stdio 传输 |
| 文件监控 | fsnotify v1.10 | (可选，次要用例) |

## 🚀 快速开始

### 1. 构建

```bash
cd src

# 前端
cd frontend && npm install && cd ..

# GUI 桌面应用（开发模式）
wails3 dev

# MCP 独立二进制
go build -o chronodraft-mcp.exe ./cmd/mcp
```

### 2. 环境变量

| 变量 | 说明 | 默认值 |
|:---|:---|:---|
| `CHRONODRAFT_AI_KEY` | AI API Key | （必须设置） |
| `CHRONODRAFT_AI_BASE` | API Base URL | `https://api.openai.com/v1` |
| `CHRONODRAFT_AI_MODEL` | 模型名称 | `gpt-4o` |
| `CHRONODRAFT_PROJECT_ROOT` | 要监控的项目根目录 | 当前工作目录 |

> ⚠️ `CHRONODRAFT_PROJECT_ROOT` 必须指向**要监控的项目根目录**，不是 ChronoDraftAEx 自身。

### 3. 使用流程

**首次接入已有项目：**
1. 启动 GUI → 添加项目 → 选择项目目录
2. 仪表盘显示"知识库为空" → 点击「扫描项目结构」
3. `index_project` 扫描目录树，为关键文件生成作用描述，构建初始知识图谱
4. Agent 即可通过 MCP 工具查询知识库

**日常开发（Agent 驱动 — 核心工作流）：**
```
Agent 修改代码
  ↓
Agent 调用 record_change(what, why, problem, files, tags)
  ↓
ChronoDraftAEx 直接索引为结构化条目 + 刷新 AGENTS.md
  ↓
前端仪表盘实时显示新条目
```

Agent 是知识的主要生产者 — `record_change` 是核心工具，三个参数必须提供：
- **what**：改了什么（具体修改内容）
- **why**：为什么改（设计决策与理由）
- **problem**：解决了什么问题

这比自动检测文件变更更高效，因为 Agent 了解修改的上下文和意图。

## 📡 集成 AI 编辑器

### OpenCode

在项目根目录的 `opencode.json` 中配置：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "chronodraft": {
      "type": "local",
      "command": ["F:\\path\\to\\chronodraft-mcp.exe"],
      "enabled": true,
      "environment": {
        "CHRONODRAFT_PROJECT_ROOT": "F:\\path\\to\\your\\project",
        "CHRONODRAFT_AI_KEY": "sk-..."
      }
    }
  }
}
```

### Claude Desktop / Cursor

详见 [MCP 配置指南](docs/mcp-config.md)。

### 可用 MCP 工具（7 个）

| 工具 | 说明 | 关键参数 |
|:---|:---|:---|
| `record_change` | **Agent 报告代码变更**（核心驱动） | `what`(必填), `why`(必填), `problem`(必填), `files`, `tags` |
| `search_knowledge` | 语义搜索知识库 | `query`, `top_k`(默认 10) |
| `get_snapshot` | 获取项目知识快照（AGENTS.md 格式） | — |
| `get_graph` | 获取知识图谱数据（节点和边） | `limit`(默认 100) |
| `get_project_structure` | 获取项目文件结构 | — |
| `list_entries` | 分页列出知识条目 | `offset`(默认 0), `limit`(默认 20) |
| `index_project` | 扫描项目文件结构，构建初始知识图谱 | — |

### `record_change` 使用示例

```json
record_change(
  what: "添加了用户认证模块，包括登录/注册页面和 JWT token 管理",
  why: "为了支持多用户系统，需要独立的认证流程来保护敏感页面",
  problem: "原有系统没有用户认证机制，任何人可以直接访问管理页面",
  files: "src/auth/login.ts,src/auth/register.ts,src/middleware/jwt.ts",
  tags: "认证,安全,JWT"
)
```

调用后 ChronoDraftAEx 自动：
1. 索引为结构化知识条目
2. 刷新 AGENTS.md（含工具列表、使用指南、变更内容）
3. 前端仪表盘立即显示新条目

## 🧠 知识图谱模型

知识图谱记录项目的**文件结构关系**而非文件名间的关系：

```
项目根目录/
├── src/                (目录节点)
│   ├── main.go         (文件节点 + 作用描述)
│   ├── app.go          (文件节点 + 作用描述)
│   └── internal/
│       ├── mcp/        (目录节点)
│       │   ├── stdio.go  (文件节点)
│       │   └── format.go (文件节点)
│       └── memorycore/
│           └── core.go   (文件节点)
```

边类型：`CONTAINS`（目录包含文件/子目录）

`index_project` 扫描时为关键文件调用 AI 生成作用描述，构建图谱。

## 🚫 忽略规则

扫描项目时自动排除以下构建产物和配置目录：

| 类型 | 列表 |
|:---|:---|
| VCS | `.git` |
| 数据 | `.chronodraft`, `node_modules`, `vendor` |
| 构建产物 | `build`, `dist`, `target`, `out`, `bin`, `obj` |
| 缓存 | `__pycache__`, `.next`, `.nuxt`, `.output` |
| IDE | `.gradle`, `.idea`, `.vscode`, `.vs` |
| 系统 | `.DS_Store`, `Thumbs.db` |
| 文件扩展名 | `.class`, `.pyc`, `.pyo`, `.o`, `.so`, `.dll`, `.exe`, `.bin`, `.flat`, `.dex` |

## 📁 项目结构

```
ChronoDraftAEx/
├── src/
│   ├── main.go                    # Wails 桌面应用入口
│   ├── app.go                     # 服务层（绑定的前端调用方法）
│   ├── cmd/mcp/main.go            # MCP 独立二进制入口
│   ├── internal/
│   │   ├── mcp/                   # MCP stdio 服务器（7 个工具注册与处理）
│   │   │   ├── stdio.go           #   工具注册 + handler 实现
│   │   │   ├── format.go          #   AGENTS.md 格式化生成
│   │   │   └── stdio_test.go      #   单元测试
│   │   ├── memorycore/            # 记忆内核（整合层：索引、查询、知识管理）
│   │   ├── kbindex/               # 知识库索引（向量、图结构、元数据）
│   │   ├── changedetect/          # 文件变动检测 + 快照 + 忽略规则
│   │   ├── changeorganize/        # (辅助) AI 结构化摘要生成
│   │   ├── agentswriter/          # AGENTS.md 写入磁盘
│   │   ├── depsdetect/            # 依赖自动检测
│   │   ├── filewatcher/           # (辅助) fsnotify 文件监控
│   │   └── config/                # 配置管理（项目切换/持久化）
│   ├── pkg/
│   │   ├── models/                # 核心数据结构（StructuredEntry, FileChange 等）
│   │   └── utils/                 # 工具函数（ID 生成等）
│   └── frontend/                  # React 前端
│       └── src/
│           ├── components/        # Dashboard, KnowledgeGraph, SnapshotView, ProjectManager
│           ├── contexts/          # ToastContext
│           └── types/             # TypeScript 类型定义
├── docs/
│   ├── 使用说明.md                # 人类使用 + Agent 使用完整指南
│   └── mcp-config.md             # Claude Desktop / Cursor 配置示例
└── dev.ps1                        # 开发模式启动脚本
```

## 📖 文档

- [使用说明](docs/使用说明.md) — 人类使用 + Agent 使用完整指南
- [MCP 配置](docs/mcp-config.md) — Claude Desktop / Cursor 配置示例
- [ChronoDraftAEx 设计文档](docs/ChronoDraftAEx.md) — 系统架构与设计细节
