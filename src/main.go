// Package main 是 ChronoDraftAEx 桌面应用的入口
// 基于 Wails v3 框架构建，提供跨平台 GUI 能力
package main

import (
	"fmt"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	app := application.New(application.Options{
		Name:        "ChronoDraftAEx",
		Description: "AI 编码时代项目知识的黑匣子",
		Services: []application.Service{
			{
				Name:  "App",
				Setup: NewApp,
			},
		},
		Assets: application.AssetOptions{
			// Wails v3 默认从 frontend/dist 加载打包后的前端资源
			Handler: nil,
		},
		Windows: application.WindowsOptions{
			WebviewUserDataPath: "",
		},
	})

	// 创建主窗口
	mainWindow := app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:  "ChronoDraftAEx",
		Width:  1280,
		Height: 800,
		Center: true,
	})

	_ = mainWindow

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "应用运行失败: %v\n", err)
		os.Exit(1)
	}
}
