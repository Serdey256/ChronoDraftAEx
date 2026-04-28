# ChronoDraftAEx

### 💡 产品愿景与核心场景：将提示词工程转变为“工程资产”
在OpenClaw等代理进行多轮开发时，上下文窗口资源的剧烈竞争与项目记忆丢失混乱是核心痛点。ChronoDraftAEx的设计逻辑是**嵌入式工作、结构化产出**，它不改变开发者习惯，而是将每次对话产出的改动提示词和决策理由，转化为可索引、可追溯、可重用的“工程资产”，从而在根本上解决上述问题。

---

### 🛠️ ChronoDraftAEx 产品框架设计

#### 1. 核心架构：“记忆内核 + 工具链插件”
*   **记忆索引内核 (Backend)**：系统核心，负责结构化记录项目信息（版本号、依赖等），并通过向量嵌入和知识图谱构建高维语义记忆网络。它既记录代码变更（如`src/auth/google-oauth.ts`修改），也关联其设计意图与决策链（如“选择`PKCE`流程的原因”）。
*   **工具链与IDE插件 (Agents)**：通过MCP协议与OpenClaw等主流工具安全交互。它将生成一个轻量级的结构化项目知识快照文件（如`AGENTS.md`），AI代理只需读取此文件即可快速获取当前项目的全局上下文与近期改动摘要，大幅提升问答准确率和相关性。

#### 2. 核心工作流：“变动记录-智能整理-索引更新”
*   **对话变动实时监控 (`change-detect`)**：每次对话结束后，ChronoDraftAEx能通过MCP协议（Model Context Protocol，一种允许AI代理与外部工具安全交互的标准协议）捕获并对比文件系统变更，生成结构化的改动记录。
*   **AI生成结构化摘要 (`change-organize`)**：这是产品核心。内置的AI将对原始记录进行提炼，生成结构化的更新条目，包含**代码变更概要**、**设计决策与理由**及**潜在影响面**，而非简单的diff信息。
*   **知识库索引更新 (`kb-index`)**：将上述结构化的知识单元，通过**向量嵌入（用于语义检索）** 和**知识图谱（用于关系推理）** 融合存入本地索引数据库，构建高维语义记忆网络。

#### 3. 可视化交互界面 (GUI)
为了满足跨平台与可视化需求，界面设计结合现代web技术实现。
*   **跨平台GUI方案**：推荐使用 **Go + React** 结合 **Wails v3** 框架（打包后应用约15MB，内存占用约10MB，远优于Electron）。React能提供优秀的UI交互体验，并结合D3.js等库进行关系图谱、时间线等复杂数据的可视化。
*   **可视化仪表盘 (Dashboard)**：清晰展示项目演进历史（时间线视图）与知识关联图谱。例如，展示`auth模块`的开发时间线，并高亮其与`security`和`database`模块的依赖关系。
*   **项目“快照” (Snapshot)**：以图形化方式对比项目不同版本间的依赖、接口变化等，让宏观变化一目了然。
*   **知识库浏览器 (KB Explorer)**：用户可通过关键词、文件路径或语义搜索查询知识库，并用知识图谱浏览实体间的关联。

---

### ⚙️ 技术栈

| 层级 | 技术选项 | 选择理由 |
| :--- | :--- | :--- |
| **桌面框架** | Wails v3 (Go + Web) | 跨平台、性能优异、体积小、内存占用低 |
| **后端语言** | Go | 高性能、并发支持好、编译为单一二进制文件，部署简单 |
| **前端框架** | React / Vue + D3.js | 生态丰富、组件化开发效率高，D3.js适合复杂数据可视化 |
| **AI能力** | 调用云端API (如OpenAI / Claude) | 仅需API Key，完全不受本地设备限制 |
| **向量数据库** | LanceDB (嵌入式) | 文件级存储，无需额外服务，支持向量+元数据混合检索 |
| **图数据库** | Kuzu (嵌入式) | 与LanceDB搭配，为知识图谱提供原生图查询能力 |
| **本地存储** | SQLite | 作为结构化元数据的主存储，轻量可靠，与LanceDB/Kuzu互补 |

---

### 💎 总结

“ChronoDraftAEx”不只是文档生成器，它是为AI编码时代设计的“**结构化记忆与上下文管理系统**”，解决了从短期上下文爆满到长期项目知识沉淀的核心问题。它以嵌入、辅助、无感的姿态，帮助AI代理有效延续每次对话的累积价值。

AI 编码时代项目知识的「黑匣子」——结构化记忆与上下文管理系统。

## 技术栈

- **桌面框架**: Wails v3 (Go + Web)
- **后端语言**: Go
- **前端框架**: React + D3.js
- **AI 能力**: 云端 API (OpenAI / Claude 兼容)
- **向量数据库**: LanceDB (嵌入式)
- **图数据库**: Kuzu (嵌入式)
- **本地存储**: SQLite

## 项目结构

```
src/
├── main.go                      # 应用入口 (Wails v3)
├── app.go                       # App 层，暴露给前端的绑定方法
├── internal/
│   ├── changedetect/            # 对话变动实时监控
│   ├── changeorganize/          # AI 生成结构化摘要
│   ├── kbindex/                 # 知识库索引更新 (向量 + 图 + 元数据)
│   ├── memorycore/              # 记忆索引内核（整合层）
│   └── mcp/                     # MCP 协议服务器（与 AI 代理交互）
├── pkg/
│   ├── models/                  # 核心数据结构
│   └── utils/                   # 通用工具函数
└── frontend/                    # React 前端
    ├── src/
    │   ├── components/
    │   │   ├── Dashboard.tsx    # 仪表盘
    │   │   ├── KnowledgeGraph.tsx # 知识图谱 (D3.js)
    │   │   └── SnapshotView.tsx   # 项目快照
    │   ├── App.tsx
    │   ├── main.tsx
    │   ├── types.ts
    │   └── styles.css
    └── package.json
```

## 核心工作流

1. **change-detect**: 对比文件系统快照，检测新增/修改/删除
2. **change-organize**: 调用 AI API 提炼结构化知识（变更概要、设计决策、影响面）
3. **kb-index**: 将知识单元写入 LanceDB（语义检索）、Kuzu（关系推理）、SQLite（元数据）

## 开发指南

### 环境变量

- `CHRONODRAFT_AI_KEY`: AI API Key
- `CHRONODRAFT_AI_BASE`: AI API Base URL (默认: https://api.openai.com/v1)
- `CHRONODRAFT_AI_MODEL`: 模型名称 (默认: gpt-4o)

### 构建

```bash
cd src

# 安装前端依赖
cd frontend && npm install && cd ..

# 开发模式 (Wails v3)
wails dev

# 构建生产版本
wails build
```

### MCP 接口

启动后 MCP 服务器监听 `:8787`：

- `GET /mcp/snapshot` — 获取项目知识快照
- `POST /mcp/search` — 语义搜索知识库
- `GET /mcp/health` — 健康检查
