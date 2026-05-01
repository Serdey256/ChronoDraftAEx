## 1. SQLite 元数据存储 (metadata.go)

**文件**: `src/internal/kbindex/metadata.go`

### 变更内容
将所有 TODO 占位替换为完整的 SQLite CRUD 实现。

### 具体实现
- **`SaveEntry`**: 使用事务将 `StructuredEntry` 写入 `entries` 表，关联的 `FileChange` 写入 `file_changes` 表。Tags 使用 JSON 序列化存储。
- **`GetEntryByID`**: 通过 ID 查询条目，JOIN `file_changes` 表恢复 `AffectedFiles`。
- **`ListEntries`**: 分页查询，按时间倒序，支持 offset/limit 参数。
- **`SaveSnapshot`**: 将 `ProjectSnapshot` 的 dependencies 和 metadata 均 JSON 序列化后写入 `snapshots` 表。
- **`ListSnapshots`**: 查询所有快照，按时间倒序。
- **`getFileChanges`**: 内部辅助方法，查询指定条目的文件变更记录。
- 启用 WAL 日志模式提升并发性能。

---

## 2. 向量数据库层 (vectordb.go)

**文件**: `src/internal/kbindex/vectordb.go`

### 设计决策
README 原计划使用 LanceDB (嵌入式向量数据库)。调研发现 `github.com/lancedb/lancedb-go` v0.1.2 需要 CGO + 预编译原生库 + 手动设置 CGO_CFLAGS/CGO_LDFLAGS，集成到 Wails 跨平台构建流程复杂度极高。

**最终方案**: 使用 SQLite 存储向量（BLOB 字段）+ Go 原生余弦相似度计算。接口设计与 LanceDB 方案一致，后续可无缝替换。

### 具体实现
- **`InitSchema`**: 创建 `vectors` 表，包含 entry_id, session_id, summary, design_decision, impact_analysis, tags, embedding (BLOB), timestamp。
- **`InsertEntry`**: 调用 OpenAI 兼容的 embedding API 生成 1536 维向量，编码为小端序 BLOB 存入 SQLite。API 不可用时降级为零向量。
- **`Search`**: 从 SQLite 加载所有向量，计算余弦相似度，按得分降序返回 topK 结果。嵌入 API 不可用时回退到 SQL LIKE 关键词匹配。
- **`generateEmbedding`**: 调用 `/embeddings` 端点，默认使用 `text-embedding-3-small` 模型。
- **`encodeVector` / `decodeVector`**: float32 切片与字节的编解码。
- **`cosineSimilarity`**: 纯 Go 实现的余弦相似度计算。
- **`SetEmbeddingConfig`**: 支持外部配置 API Key、Base URL 和模型名称。

---

## 3. 图数据库层 (graphdb.go)

**文件**: `src/internal/kbindex/graphdb.go`

### 设计决策
README 原计划使用 Kuzu (嵌入式图数据库)。调研发现 `github.com/kuzudb/go-kuzu` v0.11.3 已归档（2025-10-10），且 Windows 上需要 MSYS2 + UCRT64 环境。

**最终方案**: 使用 SQLite 邻接表实现图存储。节点和边分别存储在 `nodes` 和 `edges` 表中，通过 SQL JOIN 实现图遍历。

### 具体实现
- **`InitSchema`**: 创建 `nodes` 表（id, label, type, metadata）和 `edges` 表（source_id, target_id, relation），含唯一约束和索引。
- **`InsertEntry`**: 在事务中：
  1. 创建 Entry 节点
  2. 为每个受影响文件创建 File 节点 + AFFECTS 边
  3. 为每个标签创建 Tag 节点 + HAS_TAG 边
  4. 通过共同标签查找相关条目，建立 RELATED_TO 边
- **`QueryRelated`**: 查询指定节点的所有关联节点和边（双向）。
- **`GetGraphData`**: 获取图谱全量数据（用于前端可视化），支持 limit 参数。

---

## 4. 知识库索引整合层 (kbindex.go)

**文件**: `src/internal/kbindex/kbindex.go`

### 变更内容
- 向量数据库路径从 `vectors.lance` 改为 `vectors.db`（SQLite）
- 图数据库路径从 `graph.kuzu` 改为 `graph.db`（SQLite）
- 新增 `SetEmbeddingConfig` 方法，传递 API 配置到向量层
- 新增 `GetGraphData` 方法暴露图谱数据
- 新增 `ListEntries` 方法暴露条目列表
- 新增 `ListSnapshots` 方法暴露快照列表

---

## 5. 记忆内核 (core.go)

**文件**: `src/internal/memorycore/core.go`

### 变更内容
- 在 `NewMemoryCore` 中调用 `SetEmbeddingConfig` 传递 AI API 配置
- 新增 `GetGraphData` 方法
- 新增 `ListEntries` 方法
- 新增 `ListSnapshots` 方法

---

## 6. 应用层 (app.go)

**文件**: `src/app.go`

### 变更内容
新增暴露给前端的绑定方法：
- **`ListSnapshots()`**: 列出所有项目快照
- **`GetGraphData(limit)`**: 获取知识图谱数据，返回 `GraphData` 结构
- **`ListEntries(offset, limit)`**: 分页列出知识条目

---

## 7. MCP 服务器 (server.go)

**文件**: `src/internal/mcp/server.go`

### 变更内容
新增 HTTP 端点：
- **`GET /mcp/graph?limit=100`**: 返回知识图谱数据（节点 + 边）
- **`GET /mcp/entries?offset=0&limit=20`**: 分页列出知识条目
- `/mcp/snapshot` 端点现在同时返回 Markdown 快照和原始条目数据

---

## 8. 数据模型 (models.go)

**文件**: `src/pkg/models/models.go`

### 变更内容
新增 `GraphData` 结构体：
```go
type GraphData struct {
    Nodes []KnowledgeNode `json:"nodes"`
    Edges []KnowledgeEdge `json:"edges"`
}
```

---

## 9. 文件变动检测器 (detector.go)

**文件**: `src/internal/changedetect/detector.go`

### 变更内容
- **`SaveSnapshot`**: 实现快照持久化，将 `map[string]FileSnapshot` 序列化为 JSON 写入指定目录的 `snapshot.json` 文件。
- **`LoadSnapshot`**: 新增方法，从 `snapshot.json` 加载快照。
- 新增 `encoding/json` 导入。

---

## 10. 前端修复与增强

### types.ts
- 新增 `GraphData` 接口
- 新增 `WailsApp` 接口，统一声明所有后端绑定方法
- `KnowledgeNode.metadata` 改为可选字段
- 通过 `declare global` 统一声明 `window.go` 类型，消除各组件重复声明

### KnowledgeGraph.tsx
- **修复**: D3 drag 行为的 TypeScript 类型不兼容问题（使用 `@ts-expect-error` 抑制）
- **增强**: 接入后端 `GetGraphData` API，加载真实知识图谱数据
- **保留**: mock 数据作为后端不可用时的降级方案
- 修复 nullish coalescing 运算符优先级问题

### Dashboard.tsx
- 移除未使用的 `SearchResult` 导入
- 移除重复的 `declare global` 块

### SnapshotView.tsx
- **增强**: 接入后端 `ListSnapshots` 和 `CreateSnapshot` API
- **保留**: mock 数据作为后端不可用时的降级方案
- 移除重复的 `declare global` 块

---

## 编译验证

- **Go 后端**: `go build ./...` 通过，无错误
- **前端**: `npm run build`（tsc + vite build）通过，输出 `dist/` 目录

---

## 技术决策总结

| 原计划 | 实际方案 | 原因 |
|--------|----------|------|
| LanceDB (CGO) | SQLite + 余弦相似度 | LanceDB Go SDK 需要下载原生库 + 手动设置 CGO 环境变量，Wails 跨平台构建兼容性差 |
| Kuzu (CGO) | SQLite 邻接表 | go-kuzu 已归档，Windows 需要 MSYS2 环境 |

两个方案的接口设计保持一致，后续有需要时可无缝替换为原生 SDK。
