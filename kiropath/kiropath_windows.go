//go:build windows

package kiropath

import (
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"kiro-manager/internal/cmdutil"
)

// Registry 查詢路徑（按優先級排序）
var registryPaths = []string{
	`HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall`,
	`HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall`,
	`HKLM\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
}

// getWindowsRegistryPath 從 Windows Registry 讀取 Kiro 安裝路徑
// 查詢順序：
// 1. HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\{*Kiro*}
// 2. HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall\{*Kiro*}
// 3. HKLM\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\{*Kiro*}
// 讀取欄位：InstallLocation 或 DisplayIcon
func getWindowsRegistryPath() (string, error) {
	for _, regPath := range registryPaths {
		output, err := queryRegistry(regPath)
		if err != nil {
			continue
		}

		path, err := parseRegistryOutput(output)
		if err == nil && path != "" {
			return path, nil
		}
	}

	return "", errors.New("kiro not found in Windows Registry")
}

// queryRegistry 執行 reg query 命令查詢 Registry
func queryRegistry(regPath string) (string, error) {
	// 使用 reg query 搜索包含 "Kiro" 的項目
	// /s: 遞迴搜索子項
	// /f: 搜索字串
	// /d: 只搜索資料（值）
	cmd := exec.Command("reg", "query", regPath, "/s", "/f", "Kiro", "/d")
	cmdutil.HideWindow(cmd)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// parseRegistryOutput 解析 reg query 輸出，提取 Kiro 安裝路徑
// 優先使用 InstallLocation，其次使用 DisplayIcon
func parseRegistryOutput(output string) (string, error) {
	if output == "" {
		return "", errors.New("empty registry output")
	}

	// 檢查是否有錯誤訊息
	if strings.Contains(output, "ERROR:") {
		return "", errors.New("registry query failed")
	}

	lines := strings.Split(output, "\n")

	// 正則表達式匹配 Registry 值
	// 格式: "    ValueName    REG_SZ    ValueData"
	installLocationRe := regexp.MustCompile(`(?i)^\s*InstallLocation\s+REG_SZ\s+(.+?)\s*$`)
	displayIconRe := regexp.MustCompile(`(?i)^\s*DisplayIcon\s+REG_SZ\s+(.+?)\s*$`)

	var installLocation, displayIcon string

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		// 優先匹配 InstallLocation
		if matches := installLocationRe.FindStringSubmatch(line); len(matches) > 1 {
			installLocation = strings.TrimSpace(matches[1])
			// 找到 InstallLocation 就直接返回
			if installLocation != "" {
				return installLocation, nil
			}
		}

		// 其次匹配 DisplayIcon
		if displayIcon == "" {
			if matches := displayIconRe.FindStringSubmatch(line); len(matches) > 1 {
				displayIcon = strings.TrimSpace(matches[1])
			}
		}
	}

	// 如果有 DisplayIcon，從中提取目錄
	if displayIcon != "" {
		path := extractInstallPath(displayIcon)
		if path != "" {
			return path, nil
		}
	}

	return "", errors.New("no valid install path found in registry output")
}

// extractInstallPath 從 DisplayIcon 值提取安裝目錄
// DisplayIcon 通常是 exe 路徑，可能帶有圖標索引（如 "C:\...\Kiro.exe,0"）
func extractInstallPath(displayIcon string) string {
	if displayIcon == "" {
		return ""
	}

	path := displayIcon

	// 移除引號
	path = strings.Trim(path, `"`)

	// 移除圖標索引（如 ",0" 或 ",1"）
	if idx := strings.LastIndex(path, ","); idx != -1 {
		// 確保逗號後面是數字（圖標索引）
		suffix := path[idx+1:]
		if isNumeric(suffix) {
			path = path[:idx]
		}
	}

	// 提取目錄部分
	dir := filepath.Dir(path)

	return dir
}

// isNumeric 檢查字串是否為數字
func isNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// getRunningProcessPathInternal 從運行中的 Kiro 進程取得安裝路徑 (Windows 實作)
func getRunningProcessPathInternal() (string, error) {
	// 使用 WMIC 查詢 Kiro 進程的執行檔路徑
	cmd := exec.Command("wmic", "process", "where", "name='Kiro.exe'", "get", "ExecutablePath", "/format:list")
	cmdutil.HideWindow(cmd)

	output, err := cmd.Output()
	if err != nil {
		return "", errors.New("failed to query running processes")
	}

	// 解析輸出，格式為 "ExecutablePath=C:\...\Kiro.exe"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ExecutablePath=") {
			exePath := strings.TrimPrefix(line, "ExecutablePath=")
			exePath = strings.TrimSpace(exePath)
			if exePath != "" {
				// 返回目錄部分
				return filepath.Dir(exePath), nil
			}
		}
	}

	return "", errors.New("kiro process not found")
}

// getDarwinSpotlightPath Windows 平台不需要此函數，返回錯誤
func getDarwinSpotlightPath() (string, error) {
	return "", errors.New("spotlight not available on Windows")
}

// getLinuxWhichPath Windows 平台不需要此函數，返回錯誤
func getLinuxWhichPath() (string, error) {
	return "", errors.New("which not available on Windows")
}
