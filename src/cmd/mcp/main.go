// chronodraft-mcp: ChronoDraftAEx 的 MCP stdio 服务器
// 供 Claude Desktop、Cursor 等 AI 编辑器通过 MCP 协议访问项目知识库
package main

import (
	"ChronoDraftAEx/internal/memorycore"
	"ChronoDraftAEx/internal/mcp"
	"log"
	"os"
)

func main() {
	log.SetOutput(os.Stderr) // MCP 协议占用 stdout

	// 从环境变量获取项目路径
	projectRoot := os.Getenv("CHRONODRAFT_PROJECT_ROOT")
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			log.Fatalf("获取工作目录失败: %v", err)
		}
	}

	// 读取 AI API 配置
	apiKey := os.Getenv("CHRONODRAFT_AI_KEY")
	apiBase := os.Getenv("CHRONODRAFT_AI_BASE")
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	apiModel := os.Getenv("CHRONODRAFT_AI_MODEL")
	if apiModel == "" {
		apiModel = "gpt-4o"
	}

	// 初始化 MemoryCore
	core, err := memorycore.NewMemoryCore(projectRoot, apiKey, apiBase, apiModel)
	if err != nil {
		log.Fatalf("初始化记忆内核失败: %v", err)
	}
	defer core.Close()

	// 启动 MCP stdio 服务器
	server := mcp.NewStdioServer(core, projectRoot)
	if err := server.Start(); err != nil {
		log.Fatalf("MCP 服务器启动失败: %v", err)
	}
}
