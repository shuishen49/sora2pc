package main

import (
	"embed"
	"flag"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 解析命令行参数
	devMode := flag.Bool("dev", false, "启用开发模式（连接到开发服务器）")
	debugMode := flag.Bool("debug", false, "启用调试模式")
	flag.Parse()

	// Create an instance of the app structure
	app := NewApp()

	// 配置 AssetServer
	assetServerOptions := &assetserver.Options{
		Assets: assets,
	}

	// 如果启用开发模式，连接到开发服务器
	if *devMode || *debugMode {
		// 开发模式：连接到 Vite 开发服务器
		assetServerOptions = &assetserver.Options{
			Handler: nil, // 使用默认处理器连接到开发服务器
		}
		// 设置开发服务器 URL（Wails 会自动检测）
		if devServerURL := os.Getenv("WAILS_DEV_SERVER_URL"); devServerURL == "" {
			// 默认开发服务器地址
			os.Setenv("WAILS_DEV_SERVER_URL", "http://localhost:34115")
		}
	}

	// Create application with options（Windows 默认最大化）
	err := wails.Run(&options.App{
		Title:            "sorapc",
		Width:            1024,
		Height:           768,
		WindowStartState: options.Maximised,
		AssetServer:      assetServerOptions,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Debug: options.Debug{
			OpenInspectorOnStartup: *debugMode || *devMode,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
