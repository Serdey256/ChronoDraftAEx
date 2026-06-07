# Agent 一键接入 ChronoDraft 模板

> 这是一份给 Agent 看的接入模板。目标是：让用户在**自己的项目仓库**里，通过一次指令，把 ChronoDraft 作为 OpenCode MCP 工具接进去，并立即可用。

## 结论先说

- `MCP` 配置推荐写在项目根 `opencode.json` 或 `opencode.jsonc` 的 `mcp` 字段下。
- `.opencode/` 可以放 agent、commands、plugins、skills 等扩展内容，但**不建议**把 ChronoDraft 的 MCP 注册主配置放在那里。
- 如果要把规则交给 Agent 长期遵守，推荐额外写入项目根 `AGENTS.md`，或让 `opencode.jsonc` 的 `instructions` 引用一份规则文档。

## 适用场景

当用户对 Agent 说下面这类话时，可以使用本模板：

- “把 ChronoDraft 接到当前项目里”
- “给这个仓库加上 ChronoDraft MCP”
- “让 opencode 在当前项目里能用 ChronoDraft”

## Agent 目标

Agent 需要在**用户当前项目**中完成以下结果：

1. 保留现有配置，不破坏用户已有 `opencode.json` / `opencode.jsonc`。
2. 在项目根 OpenCode 配置中注册 `chronodraft` MCP。
3. 让 `CHRONODRAFT_PROJECT_ROOT` 指向当前项目根目录。
4. 可选写入 `CHRONODRAFT_BINARY_PATH`，以便 `scaffold_project()` 安装 Git Hook。
5. 如用户项目有 `AGENTS.md`，追加 ChronoDraft 工作流说明；没有则按需新建。
6. 接入完成后，提示用户下一步执行 `scaffold_project()`。

## 前置条件

### 如果用户要从源码构建 ChronoDraft

ChronoDraftAEx 当前仓库的 MCP 二进制构建命令为：

```bash
cd src
go build -o chronodraft-mcp.exe ./cmd/mcp
```

非 Windows 可改为：

```bash
cd src
go build -o chronodraft-mcp ./cmd/mcp
```

### Agent 在目标项目里需要知道的变量

部署时至少要确认这两个值：

- `CHRONODRAFT_MCP_PATH`: ChronoDraft MCP 二进制绝对路径
- `PROJECT_ROOT`: 当前用户项目根目录绝对路径

可选：

- `CHRONODRAFT_AI_KEY`
- `CHRONODRAFT_AI_BASE`
- `CHRONODRAFT_AI_MODEL`
- `CHRONODRAFT_EMBEDDING_MODEL`
- `CHRONODRAFT_BINARY_PATH`（通常与 `CHRONODRAFT_MCP_PATH` 相同）

## 推荐落盘结果

### 1. 项目根 `opencode.jsonc`

推荐让 Agent 在目标项目根写入或合并以下配置：

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "chronodraft": {
      "type": "local",
      "enabled": true,
      "command": [
        "C:/ABSOLUTE/PATH/TO/chronodraft-mcp.exe"
      ],
      "environment": {
        "CHRONODRAFT_PROJECT_ROOT": "C:/ABSOLUTE/PATH/TO/TARGET_PROJECT",
        "CHRONODRAFT_BINARY_PATH": "C:/ABSOLUTE/PATH/TO/chronodraft-mcp.exe",
        "CHRONODRAFT_AI_KEY": "{env:CHRONODRAFT_AI_KEY}",
        "CHRONODRAFT_AI_BASE": "{env:CHRONODRAFT_AI_BASE}",
        "CHRONODRAFT_AI_MODEL": "{env:CHRONODRAFT_AI_MODEL}",
        "CHRONODRAFT_EMBEDDING_MODEL": "{env:CHRONODRAFT_EMBEDDING_MODEL}"
      }
    }
  }
}
```

说明：

- Windows 路径建议统一写成正斜杠绝对路径，避免转义噪音。
- 如果用户不需要 AI 相关能力，可以先不设置 `CHRONODRAFT_AI_KEY`；`scaffold_project()` 仍可工作。
- 如果用户已有 `mcp` 配置，只追加 `chronodraft`，不要覆盖其他服务器。

### 2. 可选 `AGENTS.md` 片段

如果用户愿意让 Agent 在后续开发中持续使用 ChronoDraft，可追加以下片段：

```md
## ChronoDraft Workflow

- Session start: call `get_snapshot()` first.
- Before editing important files: call `get_context(files="...")`.
- After meaningful code changes: call `record_change(what, why, problem, files, tags)`.
- First time enabling ChronoDraft in this repo: call `scaffold_project()` once.
- Use `search_knowledge()` when you need past decisions.
- Use `get_code_entities()` for AST-level structure lookup.
```

如果用户不想改 `AGENTS.md`，也可以把同样内容放进单独文档，再通过 `opencode.jsonc` 的 `instructions` 引用。

## Agent 执行步骤

当用户要求“把 ChronoDraft 部署到当前项目”时，Agent 应按这个顺序做：

1. 确认当前仓库根目录。
2. 确认 ChronoDraft MCP 二进制绝对路径是否已存在。
3. 查找项目根是否已有 `opencode.json` 或 `opencode.jsonc`。
4. 若已有配置，合并 `mcp.chronodraft`，不要覆盖用户原有字段。
5. 若没有配置，新建 `opencode.jsonc`。
6. 按用户偏好决定是否追加 `AGENTS.md` 片段。
7. 完成后提示用户重启 OpenCode 或重新加载配置。
8. 最后提示在该项目中调用一次：

```text
scaffold_project()
```

## Agent 必须遵守的约束

- 不要覆盖已有 `mcp` 服务器定义。
- 不要把主 MCP 注册配置写进 `.opencode/` 来替代 `opencode.jsonc`。
- 不要假设用户一定需要 AI Key；无 Key 也可以先完成脚手架接入。
- 不要写入当前仓库无关的 provider、model、permission 配置，除非用户明确要求。
- 如果找不到 ChronoDraft 二进制，先停止并报告缺少 `CHRONODRAFT_MCP_PATH`，不要编造路径。

## 接入后可立即使用的工具

ChronoDraftAEx 当前 MCP 入口实际提供的核心工具包括：

- `scaffold_project()`
- `index_project()`（兼容别名）
- `get_snapshot()`
- `record_change(what, why, problem, files?, tags?)`
- `get_context(files)`
- `get_code_entities(files)`
- `search_knowledge(query, top_k?)`
- `get_graph(query, top_k?, compact?)`
- `list_entries(offset?, limit?, compact?)`
- `get_project_structure()`
- `capture_commit(hash, message, author?, files?, insertions?, deletions?)`

## 给用户的一键指令模板

用户可以把下面这段话直接发给 Agent：

```text
请把 ChronoDraft 部署到当前项目，并按以下规则执行：

1. 使用项目根 opencode.jsonc 注册 ChronoDraft MCP，不要把主 MCP 配置写进 .opencode/。
2. 如果当前项目已有 opencode.json 或 opencode.jsonc，只做合并，不要覆盖现有配置。
3. 将 chronodraft MCP 指向这个二进制路径：<在这里填入 ChronoDraft MCP 绝对路径>
4. 将 CHRONODRAFT_PROJECT_ROOT 设置为当前项目根目录。
5. 将 CHRONODRAFT_BINARY_PATH 设置为同一个 MCP 二进制路径。
6. 如果项目已有 AGENTS.md，就追加 ChronoDraft 工作流；如果没有，请先问我要不要创建。
7. 完成后告诉我改了哪些文件，并提醒我下一步运行 scaffold_project()。
```

## 推荐默认策略

如果没有额外偏好，推荐 Agent 采用以下默认策略：

1. `opencode.jsonc` 作为唯一必改文件。
2. `AGENTS.md` 作为可选增强，不强制创建。
3. `.opencode/` 只放本地私有 agent/commands/skills，不承载 ChronoDraft MCP 主注册。

## 为什么这么放

原因很简单：

- `opencode.jsonc` 是 OpenCode 官方文档明确推荐的项目级 JSON 配置入口。
- `mcp` 本来就是 OpenCode Config 的标准字段。
- `.opencode/` 虽然可加载部分扩展内容，但更适合 agent/commands/plugins/skills 这类目录化扩展，不适合承担团队共享的 MCP 主配置。
- 当前仓库的 `.gitignore` 已忽略 `.opencode/`，这也说明它更像本地工作区资产，而不是稳定分发入口。
