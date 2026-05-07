# ChronoDraftAEx

> AI 编码时代项目知识的「黑匣子」— 结构化记忆与上下文管理系统

## 💡 产品愿景

在 AI 代理进行多轮开发时，上下文窗口爆满、项目记忆丢失、改动文档混乱是核心痛点。ChronoDraftAEx 将每次对话产出的改动和决策理由，转化为**可索引、可追溯、可重用的工程资产**。

## 🏗️ 双轨架构

```
┌─────────────────────┐     ┌─────────────────────┐
│   Wails 桌面应用     │     │  MCP stdio 服务器    │
│   (人类用户)         │     │  (AI 代理)           │
├─────────────────────┤     ├─────────────────────┤
│ • 仪表盘/知识图谱    │     │ • 6 个 MCP 工具      │
│ • 项目管理/快照      │     │ • Claude/Cursor      │
│ • 文件监控/全量索引   │     │ • OpenCode 集成      │
└─────────┬───────────┘     └─────────┬───────────┘
          │                           │
          └───────────┬───────────────┘
                      ▼
          ┌─────────────────────┐
          │   SQLite 共享存储    │
          │ .chronodraft/       │
          └─────────────────────┘
```

## ⚙️ 技术栈

| 层级 | 技术 | 说明 |
|:---|:---|:---|
| 桌面框架 | Wails v3 (Go + Web) | 跨平台、轻量级 |
| 后端语言 | Go 1.25+ | 单二进制、高性能 |
| 前端框架 | React + TypeScript + D3.js | 组件化 + 图谱可视化 |
| AI 能力 | 云端 API (OpenAI 兼容) | 仅需 API Key |
| 存储 | SQLite | 向量/图/元数据统一存储 |
| MCP SDK | mark3labs/mcp-go | stdio 传输 |

## 🚀 快速开始

### 1. 构建

```bash
cd src

# 前端
cd frontend && npm install && cd ..

# GUI 桌面应用
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

 `CHRONODRAFT_PROJECT_ROOT`为监控项目路径，应当指向项目根目录，以便 ChronoDraftAEx 监控

### 3. 使用流程

**首次接入已有项目：**
1. 启动 GUI → 添加项目 → 选择项目目录
2. 仪表盘显示"知识库为空" → 点击「📦 全量索引」
3. 等待 AI 生成摘要 → 知识库初始化完成

**日常开发：**
1. 启动 GUI → 点击「启动监控」
2. 正常编码 → 文件变动自动捕获 → AI 生成知识条目
3. 仪表盘实时更新、知识图谱可视化

## 📡 接入 OpenCode

在项目根目录创建 `opencode.json`：

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

> ⚠️ `CHRONODRAFT_PROJECT_ROOT` 必须指向**要监控的项目目录**，不是 ChronoDraftAEx 自身。

### 可用 MCP 工具

| 工具 | 说明 |
|:---|:---|
| `search_knowledge` | 语义搜索知识库 |
| `get_snapshot` | 获取项目知识快照 |
| `get_graph` | 获取知识图谱数据 |
| `list_entries` | 列出知识条目 |
| `capture_changes` | 捕获文件变动 |
| `full_index` | 全量索引（首次接入时使用） |

## 📁 项目结构

```
ChronoDraftAEx/
├── src/
│   ├── main.go                    # Wails 桌面应用入口
│   ├── app.go                     # 服务层（暴露给前端的绑定方法）
│   ├── cmd/mcp/main.go            # MCP 独立二进制入口
│   ├── internal/
│   │   ├── changedetect/          # 文件变动检测 + 快照持久化
│   │   ├── changeorganize/        # AI 结构化摘要生成
│   │   ├── kbindex/               # 知识库索引（向量/图/元数据）
│   │   ├── memorycore/            # 记忆内核（整合层）
│   │   ├── mcp/                   # MCP stdio 服务器
│   │   ├── agentswriter/          # AGENTS.md 自动写入
│   │   ├── depsdetect/            # 依赖自动检测
│   │   ├── filewatcher/           # fsnotify 文件监控
│   │   └── config/                # 配置管理
│   ├── pkg/
│   │   ├── models/                # 核心数据结构
│   │   └── utils/                 # 工具函数
│   └── frontend/                  # React 前端
│       └── src/
│           ├── components/        # Dashboard, KnowledgeGraph, SnapshotView, ProjectManager
│           └── contexts/          # ToastContext
├── docs/
│   ├── 使用说明.md                # 完整使用指南
│   └── mcp-config.md             # MCP 配置指南
└── dev.ps1                        # 开发模式启动脚本
```

## 📖 文档

- [使用说明](docs/使用说明.md) — 人类使用 + Agent 使用完整指南
- [MCP 配置](docs/mcp-config.md) — Claude Desktop / Cursor 配置示例
