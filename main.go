package main

import (
	"embed"

	"kiro-manager/settings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	// 載入已保存的視窗大小
	s, _ := settings.LoadSettings()
	width := settings.DefaultWindowWidth
	height := settings.DefaultWindowHeight
	if s != nil && s.WindowWidth >= settings.MinWindowWidth {
		width = s.WindowWidth
	}
	if s != nil && s.WindowHeight >= settings.MinWindowHeight {
		height = s.WindowHeight
	}

	err := wails.Run(&options.App{
		Title:     "Kiro Manager",
		Width:     width,
		Height:    height,
		MinWidth:  settings.MinWindowWidth,
		MinHeight: settings.MinWindowHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 9, G: 9, B: 11, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			WebviewGpuIsDisabled: true, // 禁用 GPU 加速，解決 NVIDIA 藍屏問題
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
