# 致 AI Agent — ChronoDraftAEx 使用须知

> 你不是在空白上下文里工作。这个项目里，之前的 Agent 做过决策、改过文件、踩过坑。ChronoDraftAEx 的作用，就是把这些记忆留给你——前提是你也把自己的决策写进去。

## 先记住这次更新

### 不再做“全量 AI 索引”

如果你是第一次接入一个大项目，请调用：

```text
scaffold_project()
```

它现在表示：

- 扫描项目结构
- 构建目录/文件层级图
- 预计算 AST
- 导入最近 Git 历史
- 让你后续可以通过 `get_snapshot()` 拉取项目快照

它**不会**：

- 调用 AI 总结整个项目
- 试图理解所有实现细节
- 生成一条空洞的“全量索引”大条目

兼容说明：`index_project()` 仍然可用，但只是兼容别名。

## 你为什么要用它

| 不用 ChronoDraftAEx | 使用 ChronoDraftAEx |
|:---|:---|
| 每次会话都从零理解项目 | 会话开始先调 `get_snapshot()` 拿到项目快照 |
| 自己翻 Git log 和多个文件 | `get_context()` 和 `search_knowledge()` 直接给你线索 |
| 改完只剩 diff，没有 why | `record_change()` 把设计决策留下来 |

一句话：它帮你保存“为什么这样改”，而不是替你理解整座代码山。

### 0. 会话开始，先拉快照

```text
get_snapshot()
```

这会返回 ChronoDraft 维护的项目快照 Markdown。
如果项目根目录本来就有用户自己的 `AGENTS.md`，请把它视为**用户规则文件**；ChronoDraft 不再写它。

## 你必须养成的习惯

### 1. 改完代码，立刻记 `record_change`

```text
record_change(
  what: "改了什么",
  why: "为什么这样改",
  problem: "解决了什么问题",
  files: "涉及的文件",
  tags: "关键词,技术栈"
)
```

### 2. 改文件前，先查上下文

```text
get_context(files="src/components/LoginForm.vue")
```

你会得到：

- 最近变更
- 代码结构
- 相关决策
- 关联文件

### 3. 不确定时，搜历史

```text
search_knowledge(query="登录流程", top_k=5)
```

### 4. 看整体关系时，用图谱

```text
get_graph(query="用户模块", top_k=5, compact="true")
```

图谱现在是分层的，目录节点支持折叠/展开，不再默认坍缩成一团。

## 工具速查

| 时机 | 工具 | 一句话 |
|:---|:---|:---|
| 刚接入项目 | `scaffold_project()` | 初始化脚手架：结构扫描 + AST + Git 历史 |
| 改完代码后 | `record_change(...)` | 记录这次修改的 why |
| 改文件前 | `get_context(files="...")` | 看这个文件的来龙去脉 |
| 查代码结构 | `get_code_entities(files="...")` | 看函数、类型、导入 |
| 搜历史决策 | `search_knowledge(query="...")` | 看前人怎么想的 |
| 查关联图谱 | `get_graph(query="...", top_k=5)` | 看模块和谁有关 |
| 看项目快照 | `get_snapshot()` | 获取 ChronoDraft 项目快照 |
| 浏览条目 | `list_entries(compact="true")` | 快速看知识记录 |
| 手动记提交 | `capture_commit(...)` | 一般不需要，Hook 会自动做 |
| 看项目结构 | `get_project_structure()` | 获取目录树 |

## 关于项目快照

当前 `get_snapshot()` 默认返回这 5 个部分：

1. 项目概览
2. 关键设计决策
3. 最近动态（Git 提交）
4. 项目结构
5. 可用工具

如果你看到“关键设计决策”一栏很空，通常说明前面的 Agent 没认真调用 `record_change()`。

## 这次架构调整对你的影响

- `scaffold_project()` 只负责初始化结构与历史上下文
- 真正的高层知识入口仍然是 `record_change()`
- 扫描和 AST 会遵守项目 `.gitignore`
- Git 历史导入只保留提交元数据，不回头重做历史 AST
- ChronoDraft 不再写项目根目录 `AGENTS.md`

## 一句话总结

**ChronoDraftAEx 是你的外部记忆，不是整库自动讲解器。你负责写下为什么改，它负责把这些理由留给下一个 Agent。**
