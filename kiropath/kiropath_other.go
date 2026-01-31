//go:build !windows

package kiropath

import "errors"

// getRunningProcessPathInternal 從運行中的 Kiro 進程取得安裝路徑 (非 Windows 平台 stub)
func getRunningProcessPathInternal() (string, error) {
	// TODO: 實作 macOS 和 Linux 的進程偵測
	return "", errors.New("not implemented on this platform")
}

// getDarwinSpotlightPath 使用 Spotlight 搜索 Kiro.app (macOS)
func getDarwinSpotlightPath() (string, error) {
	// TODO: 實作 macOS Spotlight 搜索
	return "", errors.New("spotlight search not implemented")
}

// getLinuxWhichPath 使用 which 命令搜索 kiro 執行檔 (Linux)
func getLinuxWhichPath() (string, error) {
	// TODO: 實作 Linux which 搜索
	return "", errors.New("which search not implemented")
}
