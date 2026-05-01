package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	var assetHandler http.Handler

	// 开发模式：如果 FRONTEND_DEVSERVER_URL 存在，将请求代理到 Vite dev server
	if devURL := os.Getenv("FRONTEND_DEVSERVER_URL"); devURL != "" {
		target, err := url.Parse(devURL)
		if err == nil {
			proxy := httputil.NewSingleHostReverseProxy(target)
			// 修正 Host 头，避免 Vite 拒绝请求
			originalDirector := proxy.Director
			proxy.Director = func(req *http.Request) {
				originalDirector(req)
				req.Host = target.Host
			}
			assetHandler = proxy
			fmt.Printf("[DevMode] Assets proxied to: %s\n", devURL)
		} else {
			fmt.Fprintf(os.Stderr, "[DevMode] 无效的 FRONTEND_DEVSERVER_URL: %s\n", devURL)
		}
	}

	app := application.New(application.Options{
		Name:        "ChronoDraftAEx",
		Description: "AI 编码时代项目知识的黑匣子",
		Services: []application.Service{
			application.NewService(NewChronoService()),
		},
		Assets: application.AssetOptions{
			Handler: assetHandler,
		},
		Windows: application.WindowsOptions{
			WebviewUserDataPath: "",
		},
	})

	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "ChronoDraftAEx",
		Width:  1280,
		Height: 800,
	})
	mainWindow.Center()

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "应用运行失败: %v\n", err)
		os.Exit(1)
	}
}
