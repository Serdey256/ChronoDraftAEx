# ChronoDraftAEx

> AI 编码时代的「第二大脑」——把 Agent 开发过程中的设计决策、变更脉络和可复用上下文沉淀为项目资产。

## 产品定位

ChronoDraftAEx 的核心不是“让 AI 总结整个代码库”，而是：

- 记住为什么这样改
- 把最近的真实变更接到当前上下文里
- 让后续 Agent 能沿着文件、提交、决策继续工作

这次更新后，项目初始化从“全量 AI 索引”调整为“项目脚手架”模式，避免在大项目上变成负担。

## 这次更新

### 从全量索引改为脚手架

首次接入项目时，推荐调用：

```text
scaffold_project()
```

它现在只做这些事：

1. 扫描项目文件结构
2. 构建目录/文件层级图
3. 预计算代码 AST 实体
4. 导入最近 Git 历史
5. 让 Agent 之后可通过 `get_snapshot()` 获取项目快照

特点：

- 零 AI 调用
- 适合大项目
- 不尝试“总结整个代码库”
- 让后续知识通过 `record_change` 和 Git 增量积累

兼容性说明：

- `index_project()` 仍可调用
- 但现在它只是 `scaffold_project()` 的兼容别名

### 知识图谱改为分层结构

图谱现在优先表达：

- 目录
- 文件
- 知识条目
- 标签

前端图谱支持目录节点折叠/展开，避免大项目图谱坍缩为一个中心团块。

### 尊重 `.gitignore`

脚手架扫描和 AST 分析会自动跳过 `.gitignore` 中的路径，减少无意义数据进入图谱。

### Git 历史只导入元数据

初始化时导入最近 commit 的元数据，但**不会**回头对历史版本重做整库 AST。
后续 AST 由 Git Hook 和增量更新维护。

## 三层记忆架构

```text
Layer 1: get_snapshot()
  项目概览、最近动态、关键决策、结构快照（通过 MCP 获取，不落盘写文件）

Layer 2: get_context / get_code_entities
  文件级上下文、代码实体、关联文件、相关决策

Layer 3: search_knowledge / get_graph
  语义搜索、关系图谱、知识追溯
```

## 技术栈

| 层级 | 技术 | 说明 |
|:---|:---|:---|
| 桌面框架 | Wails v3 | 跨平台 GUI |
| 后端 | Go | 单二进制、零 CGO |
| 前端 | React + TypeScript + D3.js | 仪表盘 + 图谱可视化 |
| 存储 | SQLite | metadata + vector + graph |
| MCP | mark3labs/mcp-go | stdio 通信 |

## 快速开始

### 构建

```bash
cd src
go build -o chronodraft-mcp.exe ./cmd/mcp
wails3 dev
```

### 环境变量

| 变量 | 说明 | 默认值 |
|:---|:---|:---|
| `CHRONODRAFT_AI_KEY` | AI API Key | 必填（仅 AI 功能需要） |
| `CHRONODRAFT_AI_BASE` | API Base URL | `https://api.openai.com/v1` |
| `CHRONODRAFT_AI_MODEL` | 对话模型 | `gpt-4o` |
| `CHRONODRAFT_EMBEDDING_MODEL` | 嵌入模型 | `text-embedding-3-small` |
| `CHRONODRAFT_PROJECT_ROOT` | 被监控项目根目录 | 当前工作目录 |
| `CHRONODRAFT_BINARY_PATH` | MCP 二进制路径（Git Hook 用） | 空=跳过 Hook |

## MCP 工具工作流

### 初始化

```text
scaffold_project()
→ 扫描项目结构
→ 构建目录层级图谱
→ AST 预计算
→ 导入最近 50 条 Git 提交
→ 后续可通过 get_snapshot() 获取项目快照
```

### 日常开发

```text
1. Agent 修改代码
2. git commit（Hook 自动捕获 commit + 增量更新 AST）
3. Agent 调用 record_change 记录 why / problem / files / tags
4. ChronoDraftAEx 刷新知识条目，Agent 需要时调用 get_snapshot()
```

### 核心工具

| 工具 | 说明 |
|:---|:---|
| `scaffold_project` | 初始化项目脚手架（零 AI） |
| `record_change` | 记录设计决策，是最重要的高层入口 |
| `get_context` | 获取文件级上下文 |
| `get_code_entities` | 获取 AST 实体 |
| `search_knowledge` | 搜索历史决策 |
| `get_graph` | 查询关联图谱 |
| `get_snapshot` | 获取当前项目知识快照（Markdown，经 MCP 返回） |
| `list_entries` | 浏览知识条目 |
| `capture_commit` | 手动记录提交 |
| `get_project_structure` | 获取项目结构 |

## GUI 页面

| 页面 | 功能 |
|:---|:---|
| 仪表盘 | 搜索、条目浏览、项目脚手架进度 |
| 代码结构 | AST 实体查看 |
| 知识图谱 | 目录/文件/条目关系图，可折叠目录节点 |
| 项目快照 | 快照管理 |
| 项目管理 | 添加/切换被监控项目 |

## 忽略规则

默认忽略：

- `.git`
- `.chronodraft`
- `node_modules`
- `vendor`
- `build` / `dist` / `target` / `out` / `bin` / `obj`
- `__pycache__` / `.next` / `.nuxt`

同时会读取项目根目录的 `.gitignore` 追加忽略规则。

## 项目结构

参考 [docs/项目结构.md](docs/项目结构.md)。
