基于你提供的两份文档，我的评估结论是：**这个项目不应该二选一，而是需要"双轨架构"——人类用户走 Wails 原生绑定，AI 代理走真正的 MCP 协议。当前的普通 HTTP REST API 应该被废弃或降级为内部调试接口，不应作为对外协议。**

以下是详细评估：

---

## 一、先厘清一个根本问题：你的项目有两个截然不同的"用户"

ChronoDraftAEx 的定位是**"AI 编码时代的项目知识黑匣子"**，这意味着它同时服务两类消费者：

| 消费者 | 需求特征 | 合适的接口形态 |
|--------|---------|-------------|
| **人类开发者** | 需要可视化仪表盘、知识图谱、时间线、点击交互 | **Wails 桌面 GUI**（当前 React 前端） |
| **AI 代理/编辑器** | 需要结构化上下文、工具发现、自主调用、无头交互 | **MCP 协议**（stdio/SSE） |

这两类消费者的交互模式完全不同：**人类是"拉取信息、浏览决策"**，**AI 是"自主发现工具、调用获取上下文"**。试图用同一种协议同时满足两者，是当前代码陷入"伪 MCP"困境的根本原因。

---

## 二、普通 HTTP REST API 的适用性评估：对 AI 集成价值很低

当前 `mcp/server.go` 实现的 HTTP API 对项目目标而言**性价比极低**：

### 1. 无法解决核心痛点
AI 编辑器（Claude Desktop、Cursor、OpenClaw）**不会**去调用一个自定义的 REST API 来获取项目上下文。它们只认原生支持的协议：
- Claude Desktop 只认 **MCP over stdio**
- Cursor 支持 **MCP over SSE**
- 没有 AI 编辑器会为了你的工具单独写一个 HTTP 客户端适配器

这意味着你当前的 HTTP API 对"AI 代理交互"这个核心愿景**几乎零贡献**。

### 2. 与产品愿景直接矛盾
README 明确说"通过 MCP 协议与 OpenClaw 等主流工具安全交互"，但实现却是普通 HTTP。这不仅是技术债务，更是**产品承诺欺诈**——用户按 MCP 标准配置编辑器后发现无法连接。

### 3. 对前端也没有必要
当前前端在 Wails 窗口内完全可以通过 `window.go.main.App` 调用 Go 后端，性能更好、类型更安全、无序列化开销。额外维护一套 HTTP 层只是增加了：
- 端口占用与冲突风险
- CORS 配置麻烦
- 序列化/反序列化开销
- 安全边界模糊（谁可以访问这个 HTTP 端口？）

**唯一合理的例外**：如果你计划未来做 SaaS 版或 Web 版，可以保留一个**内部 REST API** 供前端 fetch 调用，但它应该被明确标记为 `Internal API` 或 `Web Preview API`，**绝不应被称为 MCP**。

---

## 三、MCP 协议的适用性评估：符合愿景，但需认清成本

MCP 是**唯一正确**的 AI 集成接口选择，但需要理性看待实现成本：

### 为什么必须上 MCP？

| 维度 | 评估 |
|------|------|
| **生态位** | MCP 正在快速成为 AI 工具集成的"USB-C 接口"。Claude、Cursor、OpenClaw 等已原生支持，不上 MCP 等于主动退出 AI 工具链 |
| **产品定位** | 你的产品叫"AI 编码时代的黑匣子"，如果 AI 无法原生读取这个黑匣子，产品逻辑就不成立 |
| **N×M 问题** | 未来如果有 N 个 AI 编辑器要读取 M 个项目知识库，MCP 让集成成本从 N×M 降到 N+M。HTTP API 则要求每个编辑器单独适配 |
| **工具发现** | MCP 的 `tools/list` 让 AI 代理能自主发现"捕获变更"、"搜索知识"、"获取快照"等能力，这是 REST API 无法提供的语义层 |

### 实现成本与风险（基于 2026 年 5 月现状）

| 成本项 | 实际情况 | 应对策略 |
|--------|---------|---------|
| **协议实现** | Go 语言的 MCP SDK 生态不如 TypeScript/Python 成熟，但核心协议（JSON-RPC 2.0 over stdio/SSE）手写难度中等 | 先实现 stdio 传输（Claude Desktop 用），再扩展 SSE |
| **协议演进** | MCP 仍在快速迭代（Linux Foundation 刚接手，2026 路线图聚焦企业就绪），存在 Breaking Change 风险 | 锁定一个稳定版本（如 2025.3），不追新特性 |
| **企业级缺口** | 审计日志、多租户、网关行为等尚未标准化 | 对 ChronoDraftAEx（个人/小团队桌面工具）**不构成阻碍**，这些缺口主要影响 SaaS 平台 |
| **调试复杂度** | stdio 模式调试比 HTTP 麻烦，需要日志重定向到 stderr | 用 MCP Inspector 工具辅助调试 |

**结论**：对于 ChronoDraftAEx 这种**桌面级 AI 工具**（而非企业 SaaS 平台），MCP 的协议缺口基本不产生影响。实现成本可控，长期收益极高。

---

## 四、具体架构建议

我建议将接口层重构为以下三层架构：

```
┌─────────────────────────────────────────────────────┐
│              ChronoDraftAEx 应用进程                 │
│  ┌─────────────────┐      ┌─────────────────────┐  │
│  │   React GUI     │      │   MCP Server        │  │
│  │  (Wails 绑定)   │◄────►│  (stdio / SSE)      │  │
│  │                 │      │  - tools/list       │  │
│  │  Dashboard      │      │  - tools/call       │  │
│  │  KnowledgeGraph │      │  - resources/list   │  │
│  │  SnapshotView   │      │                     │  │
│  └─────────────────┘      └─────────────────────┘  │
│           │                      │                  │
│           └──────────┬───────────┘                  │
│                      ▼                              │
│           ┌─────────────────┐                       │
│           │   Go Core       │                       │
│           │  MemoryCore     │                       │
│           │  (SQLite +      │                       │
│           │   LanceDB +     │                       │
│           │   Kuzu)         │                       │
│           └─────────────────┘                       │
└─────────────────────────────────────────────────────┘
```

### 各层职责与协议选择

| 层级 | 协议/技术 | 服务对象 | 理由 |
|------|----------|---------|------|
| **GUI ↔ Core** | **Wails 原生绑定** (`window.go.main.App`) | 人类用户 | 类型安全、无序列化、无需端口。修复当前 `window.go` 未定义问题即可（确保在 Wails 窗口内运行） |
| **AI Editor ↔ Core** | **真正的 MCP** (stdio 为主，SSE 为辅) | Claude, Cursor, OpenClaw | 原生支持、工具发现、符合产品愿景 |
| **调试/未来 Web** | **内部 REST API** (可选，重命名) | 开发者浏览器调试或未来 SaaS | 仅在需要时保留，明确与 MCP 区分 |

### 对当前代码的具体处置

| 当前模块 | 建议操作 | 说明 |
|---------|---------|------|
| `mcp/server.go` (HTTP API) | **废弃或重构** | 停止对外暴露。如需保留，重命名为 `internal/api/server.go`，明确为"内部开发接口" |
| `mcp/server.go` (MCP 实现) | **重写** | 基于 JSON-RPC 2.0 实现 stdio/SSE 传输，注册 `capture_changes`、`search_knowledge`、`get_snapshot` 等工具 |
| `Dashboard.tsx` 等前端 | **修复 Wails 绑定** | 解决 `window.go.main.App` 未定义问题，添加后端未连接提示。不要转向 HTTP API 作为默认方案 |

---

## 五、实施优先级建议

结合你的《问题清单与改进方向》，接口协议的选择应该这样落地：

### P0：立刻停止"伪 MCP"的误导
- 将 README 中"通过 MCP 协议..."的描述改为"计划通过 MCP 协议..."或暂时移除，避免用户误解
- 将 `mcp/server.go` 的 HTTP 路由从 `/mcp/*` 改为 `/api/*`，切断与 MCP 的虚假关联

### P1：修复人类用户通道（Wails 绑定）
- 这是当前最紧急的——**所有按钮静默失效**（问题 1.1）比协议选择更致命
- 确保 `wails3 dev` 启动的桌面窗口中 `window.go.main.App` 正常注入
- 添加全局连接状态检测（问题清单 P0 第 1 条）

### P2：实现真正的 MCP 服务器（AI 通道）
- 基于 Go 实现 MCP stdio 服务器（参考官方 TypeScript SDK 的逻辑，迁移到 Go）
- 注册 3-4 个核心工具：
  - `get_project_context` → 返回 AGENTS.md 内容
  - `search_knowledge` → 语义搜索知识库
  - `capture_changes` → 触发变更检测
  - `get_knowledge_graph` → 返回图谱数据
- 验证 Claude Desktop 可通过 `claude_desktop_config.json` 连接

### P3：按需保留内部 HTTP API
- 如果确实需要浏览器独立调试前端，再考虑恢复 HTTP 层
- 或者等未来做 SaaS 版时再建设

---

## 六、一句话总结

> **普通 HTTP REST API 对 AI 集成没有价值，MCP 对 AI 集成不可替代。你的项目应该让"人类走 Wails，AI 走 MCP"，当前的 HTTP API 是架构杂音，建议废弃或降级。**

ChronoDraftAEx 的核心竞争力是"让 AI 代理读懂项目历史"，如果 AI 代理无法通过标准协议（MCP）访问它，这个产品就不成立。