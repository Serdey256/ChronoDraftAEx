# ChronoDraftAEx MCP 工具指南

> 10 个工具，三层记忆架构。Agent 会话开始时建议先调用 `get_snapshot()` 获取项目快照，开发中按需调用以下工具。

---

## 初始化（首次使用，执行一次）

### `scaffold_project()`

扫描项目结构，构建目录层级图谱，导入 Git 历史。零 AI 调用，轻量快速。

```
无参数。首次接入项目时调用一次即可。
```

---

## 核心工作流（每次修改代码后）

### `record_change(what, why, problem, files?, tags?)`

记录一次设计决策。**这是唯一的高层知识入口**——只有 Agent 知道"为什么改"。

| 参数 | 必填 | 说明 |
|:---|:--:|:---|
| `what` | ✅ | 改了什么 |
| `why` | ✅ | 为什么这样改（设计决策与理由） |
| `problem` | ✅ | 解决了什么问题 |
| `files` | | 涉及的文件，逗号分隔 |
| `tags` | | 标签，逗号分隔 |

```
示例：
record_change(
  what:    "将认证从 Session 迁移到 JWT",
  why:     "需要支持无状态横向扩展，Session 需要粘性路由",
  problem: "多实例部署时用户频繁掉线",
  files:   "src/auth/jwt.ts,src/middleware/auth.ts",
  tags:    "认证,JWT,重构"
)
```

---

## 上下文检索

### `get_context(files)`

获取文件的变更历史、代码结构、相关决策、关联文件。**修改文件前调用**。

| 参数 | 必填 | 说明 |
|:---|:--:|:---|
| `files` | ✅ | 文件路径，逗号分隔，最多 3 个 |

```
get_context(files="src/components/LoginForm.vue")
```

返回 Markdown，含 4 个章节：最近变更 · 代码结构 · 相关决策 · 关联文件。

### `get_code_entities(files)`

获取文件的 AST 实体（函数签名、类型定义、导入关系）。纯结构数据，已预计算。

```
get_code_entities(files="src/stores/UserPinia.ts")
```

返回 `[{name, entity_type, signature, metadata}, ...]`

---

## 语义搜索

### `search_knowledge(query, top_k?)`

搜索历史设计决策。先向量语义搜索，失败时回退到文本匹配。

| 参数 | 必填 | 说明 |
|:---|:--:|:---|
| `query` | ✅ | 搜索关键词 |
| `top_k` | | 返回数量，默认 10 |

```
search_knowledge(query="JWT 过期处理", top_k=5)
```

> ⚠️ 需要有 `record_change` 写入的条目才能搜到结果。

### `get_graph(query, top_k?, compact?)`

查询知识图谱，返回匹配节点及其关联子图。

| 参数 | 必填 | 说明 |
|:---|:--:|:---|
| `query` | ✅ | 搜索关键词 |
| `top_k` | | 关联节点数，默认 5，最大 20 |
| `compact` | | `"true"` 精简模式，省略 metadata |

```
get_graph(query="用户模块", top_k=5, compact="true")
```

---

## 快速查询

### `get_snapshot()`

获取当前项目知识快照（Markdown，通过 MCP 返回，不写入文件）。

```
无参数。返回 Markdown，≤1500 tokens。
```

### `list_entries(offset?, limit?, compact?)`

分页列出知识条目。

| 参数 | 必填 | 说明 |
|:---|:--:|:---|
| `offset` | | 起始位置，默认 0 |
| `limit` | | 返回数量，默认 20 |
| `compact` | | `"true"` 精简模式（仅 summary + timestamp + tags） |

```
list_entries(limit=5, compact="true")
```

### `get_project_structure()`

获取当前项目的文件结构信息。

---

## Git 集成

### `capture_commit(hash, message, author?, files?, insertions?, deletions?)`

手动捕获一次 commit（通常由 Git Hook 自动调用，也可手动触发）。

| 参数 | 必填 | 说明 |
|:---|:--:|:---|
| `hash` | ✅ | Git commit hash |
| `message` | ✅ | Commit message |
| `author` | | 提交者 |
| `files` | | 变更文件，逗号分隔 |
| `insertions` | | 新增行数 |
| `deletions` | | 删除行数 |

---

## 最佳实践

```
会话开始 ──→ get_snapshot()
修改文件前 ──→ get_context(files="...")
修改完成后 ──→ record_change(what, why, problem, files, tags)
搜索历史 ──→ search_knowledge(query="关键词")
图谱关联 ──→ get_graph(query="关键词", top_k=5)
```
