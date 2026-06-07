// chronodraft-mcp: ChronoDraftAEx 的 MCP stdio 服务器
// 供 Claude Desktop、Cursor 等 AI 编辑器通过 MCP 协议访问项目知识库
package main

import (
	"ChronoDraftAEx/internal/codeanalysis"
	"ChronoDraftAEx/internal/mcp"
	"ChronoDraftAEx/internal/memorycore"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

func main() {
	log.SetOutput(os.Stderr) // MCP 协议占用 stdout

	// 处理子命令
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "capture-commit":
			runCaptureCommit()
			return
		}
	}

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

	// 启动 MCP stdio 服务器（带 panic 恢复，确保崩溃信息可被诊断）
	log.Printf("ChronoDraftAEx MCP 就绪 (项目: %s, 工具: 10)", projectRoot)
	server := mcp.NewStdioServer(core, projectRoot)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("FATAL PANIC: %v\n%s", r, string(debug.Stack()))
		}
	}()

	if err := server.Start(); err != nil {
		log.Fatalf("MCP 服务器异常退出: %v", err)
	}
}

// runCaptureCommit 处理 capture-commit 子命令
// 由 git post-commit hook 调用，捕获 commit 信息并存入知识库
func runCaptureCommit() {
	args := os.Args[2:] // 跳过程序名和子命令

	var hash, message, author, files string
	ins, dels := 0, 0

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--hash":
			if i+1 < len(args) {
				hash = args[i+1]
				i++
			}
		case "--message":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		case "--author":
			if i+1 < len(args) {
				author = args[i+1]
				i++
			}
		case "--files":
			if i+1 < len(args) {
				files = args[i+1]
				i++
			}
		case "--ins":
			if i+1 < len(args) {
				ins, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "--dels":
			if i+1 < len(args) {
				dels, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
	}

	if hash == "" || message == "" {
		log.Fatal("capture-commit 需要 --hash 和 --message 参数")
	}

	projectRoot := os.Getenv("CHRONODRAFT_PROJECT_ROOT")
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			log.Fatalf("获取工作目录失败: %v", err)
		}
	}

	core, err := memorycore.NewMemoryCore(projectRoot, "", "", "")
	if err != nil {
		log.Fatalf("初始化记忆内核失败: %v", err)
	}
	defer core.Close()

	timestamp := time.Now().Format(time.RFC3339)
	if err := core.SaveCommit(hash, message, author, timestamp, files, ins, dels); err != nil {
		log.Fatalf("保存 commit 记录失败: %v", err)
	}

	fmt.Fprintf(os.Stderr, "已捕获 commit %s: %s\n", hash[:min(len(hash), 8)], message)

	// 增量 AST 分析变更的文件
	if files != "" {
		for _, f := range strings.Split(files, ",") {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			fullPath := filepath.Join(projectRoot, f)
			entities, err := codeanalysis.AnalyzeFile(fullPath)
			if err != nil {
				log.Printf("AST 分析失败 %s: %v", f, err)
				continue
			}
			if len(entities) > 0 {
				if err := core.SaveCodeEntities(f, entities); err != nil {
					log.Printf("保存代码实体失败 %s: %v", f, err)
				}
			}
		}
	}

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
