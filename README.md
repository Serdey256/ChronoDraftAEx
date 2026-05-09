# ChronoDraftAEx

> AI 编码时代项目知识的「黑匣子」— 结构化记忆与上下文管理系统

## 💡 产品愿景

AI 代理进行多轮开发时，上下文窗口爆满、项目记忆丢失是核心痛点。ChronoDraftAEx 通过 **三层记忆架构** 解决：

- **Layer 1: AGENTS.md**（项目级记忆，会话启动时免费加载，0 token 成本）
- **Layer 2: get_context()**（任务级记忆，按需获取当前文件的变更历史+代码结构，≤1500 tokens）
- **Layer 3: 知识库**（冷存储，语义搜索+图关系推理，深度查询）

知识自动捕获（Git Hook + AST 分析）+ 上下文按需注入（Token 预算控制）= **节省 60-70% 上下文收集成本**。

## 🏗️ 架构

```
┌─────────────────────────────────────────────────────┐
│                  Agent 上下文窗口                     │
├─────────────────────────────────────────────────────┤
│  Layer 1: AGENTS.md（免费加载）                       │
│  项目概览 · 关键设计决策 · 最近动态 · 项目结构 · 可用工具 │
├─────────────────────────────────────────────────────┤
│  Layer 2: get_context / get_code_entities（按需）    │
│  文件变更历史 · 代码实体 · 关联文件 · 相关决策           │
├─────────────────────────────────────────────────────┤
│  Layer 3: search_knowledge / get_graph（深度查询）    │
│  SQLite + 向量语义搜索 + 图关系推理                   │
└─────────────────────────────────────────────────────┘
```

## ⚙️ 技术栈

| 层级 | 技术 | 说明 |
|:---|:---|:---|
| 桌面框架 | Wails v3 | 跨平台 GUI |
| 后端 | Go | 单二进制、零 CGO |
| 前端 | React + TypeScript + D3.js | 仪表盘 + 知识图谱可视化 |
| AI | OpenAI 兼容 API | 支持 DeepSeek 等 |
| 存储 | SQLite | 元数据 + 向量 + 图，三库分立 |
| MCP | mark3labs/mcp-go | stdio 传输 |

## 🚀 快速开始

### 构建

```bash
cd src
go build -o chronodraft-mcp.exe ./cmd/mcp    # MCP 独立二进制
wails3 dev                                      # GUI 桌面应用（开发模式）
```

### 环境变量

| 变量 | 说明 | 默认值 |
|:---|:---|:---|
| `CHRONODRAFT_AI_KEY` | AI API Key | 必须设置 |
| `CHRONODRAFT_AI_BASE` | API Base URL | `https://api.openai.com/v1` |
| `CHRONODRAFT_AI_MODEL` | 对话模型 | `gpt-4o` |
| `CHRONODRAFT_EMBEDDING_MODEL` | 嵌入模型 | `text-embedding-3-small` |
| `CHRONODRAFT_PROJECT_ROOT` | 被监控的项目根目录 | 当前工作目录 |
| `CHRONODRAFT_BINARY_PATH` | chronodraft-mcp.exe 绝对路径（Git Hook 用） | 空=跳过 Hook |

在项目根目录的 `opencode.json` 中配置：
### DeepSeek 用户配置示例

```json
{
  "mcp": {
    "chronodraft": {
      "type": "local",
      "command": ["F:\\path\\to\\chronodraft-mcp.exe"],
      "enabled": true,
      "environment": {
        "CHRONODRAFT_PROJECT_ROOT": "F:\\path\\to\\your\\project",
        "CHRONODRAFT_AI_KEY": "sk-...",
        "CHRONODRAFT_AI_BASE": "https://api.deepseek.com/v1",
        "CHRONODRAFT_AI_MODEL": "deepseek-chat",
        "CHRONODRAFT_EMBEDDING_MODEL": "deepseek-chat"
      }
    }
  }
}
```

> ⚠️ `CHRONODRAFT_PROJECT_ROOT` 指向**被开发的项目**，不是 ChronoDraftAEx 自身。

---

## 📡 Agent 使用指南（MCP 工具）

### 初始化（每个项目首次使用执行一次）

```
index_project()
→ 扫描项目文件结构
→ AST 分析所有 Go/TS/Java/Python/Vue 等源文件
→ 安装 Git Hook（如已配置 CHRONODRAFT_BINARY_PATH）
→ 生成初始 AGENTS.md
```

### 日常开发核心流程

```
1. Agent 修改代码
2. git commit（Hook 自动捕获 commit + AST 增量更新）
3. Agent 调用 record_change 记录设计决策
   record_change(
     what:  "添加了用户认证模块",
     why:   "选用 JWT 替代 Session，支持无状态横向扩展",
     problem: "原系统无认证，管理员页面暴露",
     files: "src/auth/login.ts,src/middleware/jwt.ts",
     tags:  "认证,JWT,安全"
   )
→ ChronoDraftAEx: 索引条目 + 刷新 AGENTS.md + 更新向量语义索引
```

### 完整 MCP 工具清单（10 个）

| 工具 | 说明 | 关键参数 | Token 成本 |
|:---|:---|:---|:--:|
| `record_change` | 记录设计决策（**核心**） | `what`, `why`, `problem`(必填), `files`, `tags` | ~100 |
| `get_context` | 获取文件级任务上下文 | `files`(必填, 逗号分隔, ≤3个) | ≤1500 |
| `get_code_entities` | 获取文件 AST 实体 | `files`(必填, 逗号分隔) | ≤500 |
| `search_knowledge` | 语义搜索历史决策 | `query`(必填), `top_k`(默认10) | ≤1000 |
| `get_graph` | 查询关联图谱 | `query`(必填), `top_k`(默认5, 最大20) | ≤1500 |
| `get_snapshot` | 获取 AGENTS.md 内容 | — | ≤1500 |
| `list_entries` | 分页列出条目 | `offset`(默认0), `limit`(默认20), `compact` | ≤1000 |
| `index_project` | 全量扫描 + AST | — | 0（后台） |
| `capture_commit` | 手动捕获 commit | `hash`, `message`(必填) | 0 |
| `get_project_structure` | 项目文件结构 | — | ≤1000 |

### Agent 最佳实践

```
会话开始:
  → 读取 AGENTS.md（IDE 自动，0 token 成本）
    → 了解项目概况、最近变更、可用工具

修改文件前:
  → get_context(files="目标文件.vue")
    → 获取它的变更历史、代码结构、关联文件

记录决策:
  → record_change(what="...", why="...", problem="...", files="...", tags="...")
    → 让后续 Agent 理解"为什么这么改"

搜索历史:
  → search_knowledge(query="JWT认证")
    → 查找相关设计决策

图谱关联:
  → get_graph(query="用户模块", top_k=5)
    → 查看与用户模块关联的文件和决策节点
```

### ⚠️ 重要提示

1. **AGENTS.md 是免费记忆**——IDE 自动加载，不消耗 Agent 上下文预算。确保它启用（GUI 仪表盘可切换）
2. **record_change 是唯一的高层知识入口**——代码变更可以自动捕获（Git Hook + AST），但"为什么改"只有 Agent 知道
3. **search_knowledge 需要数据**——必须先有 record_change 调用，才有数据可搜
4. **token 预算可控**——可在 GUI 前端禁用 AGENTS.md 自动生成以节省 token

---

## 🖥️ GUI 桌面应用

启动 `wails3 dev` 后，桌面窗口提供：

| 页面 | 功能 |
|:---|:---|
| 📊 仪表盘 | 知识条目统计、搜索、全量索引进度条 |
| 🧬 代码结构 | AST 提取的函数/类型/导入，AI 语义标注开关 |
| 🕸️ 知识图谱 | 关键词查询 + top_k 关联子图 |
| 📸 项目快照 | 快照管理 |
| 📁 项目管理 | 添加/切换被监控项目 |

---

## 🚫 忽略规则

| 类型 | 列表 |
|:---|:---|
| VCS | `.git` |
| 数据 | `.chronodraft`, `node_modules`, `vendor` |
| 构建产物 | `build`, `dist`, `target`, `out`, `bin`, `obj` |
| 缓存 | `__pycache__`, `.next`, `.nuxt` |
| IDE | `.idea`, `.vscode`, `.vs` |
| 扩展名 | `.class`, `.pyc`, `.pyo`, `.o`, `.so`, `.dll`, `.exe`, `.bin` |

---

## 📁 项目结构

```
ChronoDraftAEx/src/
├── cmd/mcp/main.go            # MCP 独立二进制入口
├── app.go                     # Wails 服务层
├── internal/
│   ├── mcp/stdio.go           # MCP 工具注册 + handler（10 个工具）
│   ├── memorycore/core.go     # 记忆内核（整合层）
│   ├── kbindex/               # SQLite 知识库（metadata + vector + graph）
│   ├── codeanalysis/          # 多语言 AST 分析（Go/TS/Java/Python/C++/Rust/C#/Kotlin/JS/Vue）
│   ├── agentswriter/          # 智能 AGENTS.md 生成器（≤1500 tokens）
│   ├── githook/               # Git post-commit Hook 自动捕获
│   ├── changeorganize/        # AI 结构化摘要 + 代码实体语义标注
│   ├── changedetect/          # 文件变动检测 + 快照
│   └── config/                # 项目管理配置
├── pkg/
│   ├── models/                # 核心数据结构
│   └── utils/                 # Tokenizer 等工具
└── frontend/                  # React 前端
    └── src/components/        # Dashboard, CodeEntities, KnowledgeGraph 等
```
