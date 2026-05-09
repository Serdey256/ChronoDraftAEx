# 致 AI Agent — ChronoDraftAEx 使用须知

> 你不是在一个空白的上下文里工作。有几轮对话在你之前修改过这个项目，做过设计决策，踩过坑。这些记忆都存在 ChronoDraftAEx 里——**前提是你告诉它你做了什么**。

---

## 你的收益：为什么你要费心多调一个工具

| 你不用 ChronoDraftAEx | 你用 ChronoDraftAEx |
|:---|:---|
| 每次会话从零理解项目 | 启动时 AGENTS.md 已告诉你项目概况（0 token 成本） |
| 改文件前自己读 3-5 个文件了解上下文 | `get_context("目标文件.vue")` 一步到位 |
| 搜索 git log 找历史变更 | `search_knowledge("JWT")` 语义搜索 |
| 同一件事向用户解释两次 | `record_change` 记一次，后续 Agent 自己查到 |

**一句话：多调一个工具，你自己省 3000-5000 tokens 的上下文探索成本。**

---

## 你必须养成的习惯

### 习惯 1：改完代码，立即记录

```
你的每次修改完成后，末尾加上一行：

⏺  record_change(
     what:    "一句话说改了什么",
     why:     "为什么不用别的方案",
     problem: "之前会出什么问题",
     files:   "改动的文件",
     tags:    "关键词,技术栈"
   )
```

**这不是对你自己的记忆，是对下一个接手这段代码的 Agent 的记忆。** 你不写，下个 Agent 就得重新逆向工程你的决策。

### 习惯 2：改文件前，先查上下文

```
目标: 修改 src/components/LoginForm.vue

⏺  先调: get_context(files="src/components/LoginForm.vue")
   → 这个文件最近被谁改过？
   → 有什么设计决策关联到它？
   → 它和哪些文件一起变更过？
   → 它的代码结构是什么？
```

### 习惯 3：不确定怎么改时，搜索历史

```
⏺  search_knowledge(query="登录流程", top_k=5)
   → 之前有人做过登录相关的改动吗？
   → 他们当时的设计决策是什么？
   → 有什么坑已经被踩过了？
```

### 习惯 4：想了解整体关系时，查图谱

```
⏺  get_graph(query="用户模块", top_k=5, compact="true")
   → 用户模块关联了哪些文件？
   → 哪些决策节点和它有关？
```

---

## 10 个工具速查

| 时机 | 工具 | 一句话 |
|:---|:---|:---|
| **刚接入项目** | `index_project()` | 扫描项目，建立初始知识库 |
| **改完代码后** | `record_change(what,why,problem,files,tags)` | 这是你最重要的义务 |
| **改文件前** | `get_context(files="...")` | 了解这个文件的来龙去脉 |
| **查代码结构** | `get_code_entities(files="...")` | 看函数/类型/导入 |
| **搜历史决策** | `search_knowledge(query="...")` | 前人怎么想的 |
| **搜关联关系** | `get_graph(query="...", top_k=5)` | 这个模块和谁有关 |
| **看项目快照** | `get_snapshot()` | AGENTS.md 全文 |
| **列出条目** | `list_entries(compact="true")` | 快速浏览记录 |
| **手动记录commit** | `capture_commit(hash, message, ...)` | 一般用不到（Hook 自动） |
| **看文件结构** | `get_project_structure()` | 项目目录树 |

---

## record_change 怎么写才有效

### ❌ 无效写法（浪费 token，后续 Agent 无法利用）

```
record_change(
  what: "修改了代码",           ← 没说是哪个模块
  why: "修复bug",               ← 没说是哪个bug、怎么修的
  problem: "有问题",            ← 什么问题？
  files: "src/app.vue"
)
```

### ✅ 有效写法（后续 Agent 能直接利用）

```
record_change(
  what: "将登录表单的密码校验从客户端正则改为后端 API 校验",
  why: "客户端正则可被绕过，后端校验配合速率限制更安全；选择了 /api/auth/validate-password 而非复用 /api/users，
        因为校验逻辑与用户 CRUD 无关，独立端点便于未来加 MFA",
  problem: "用户可以提交弱密码直接注册，密码强度策略形同虚设",
  files: "src/components/LoginForm.vue,src/api/auth.ts,src/middleware/rateLimit.ts",
  tags: "安全,登录,密码校验,速率限制,前端"
)
```

**每条有效记录 ≈ 为下一个 Agent 节省 2000 tokens 的逆向工程成本。**

---

## 你的 AGENTS.md 里有什么

项目根目录的 AGENTS.md 是 ChronoDraftAEx 自动维护的。包含 4 个部分：

1. **项目概览** — 这个项目的技术栈和基本信息
2. **关键设计决策** — 最近 5 条 record_change（**这就是你写的那些记录**）
3. **最近动态** — 最近 3 次 git commit
4. **项目结构** — 目录树
5. **可用工具** — 本文件的工具速查

AGENTS.md **每次会话自动加载，0 token 成本**。如果里面"关键设计决策"是空的，说明之前在这个项目上工作的 Agent 没有用 record_change——你可以在你的改动后成为第一个记录的人。

---

## 一句话总结

**ChronoDraftAEx 是你的外部记忆。它不会自动记住你为什么做某个决定——你必须告诉它。但一旦你告诉了它，后续所有 Agent（包括你自己在别的会话里）都能通过 `search_knowledge` 和 `get_context` 找到。**
