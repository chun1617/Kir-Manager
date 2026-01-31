//go:build !windows

package kiropath

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// getRunningProcessPathInternal 從運行中的 Kiro 進程取得安裝路徑 (非 Windows 平台)
func getRunningProcessPathInternal() (string, error) {
	// 使用 pgrep 找到 Kiro 進程
	cmd := exec.Command("pgrep", "-l", "Kiro")
	output, err := cmd.Output()
	if err != nil {
		return "", errors.New("kiro process not found")
	}

	// 解析 pgrep 輸出取得 PID
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", errors.New("kiro process not found")
	}

	parts := strings.SplitN(lines[0], " ", 2)
	if len(parts) < 1 {
		return "", errors.New("failed to parse pgrep output")
	}

	pid := parts[0]

	// 使用 lsof 或 /proc 取得執行檔路徑
	// macOS: lsof -p PID -Fn
	// Linux: readlink /proc/PID/exe
	var exePath string

	// 嘗試 Linux 方式
	procExe := "/proc/" + pid + "/exe"
	if linkCmd := exec.Command("readlink", "-f", procExe); linkCmd != nil {
		if linkOutput, err := linkCmd.Output(); err == nil {
			exePath = strings.TrimSpace(string(linkOutput))
		}
	}

	// 如果 Linux 方式失敗，嘗試 macOS 方式
	if exePath == "" {
		lsofCmd := exec.Command("lsof", "-p", pid, "-Fn")
		if lsofOutput, err := lsofCmd.Output(); err == nil {
			for _, line := range strings.Split(string(lsofOutput), "\n") {
				if strings.HasPrefix(line, "n") && strings.Contains(line, "Kiro") {
					exePath = strings.TrimPrefix(line, "n")
					break
				}
			}
		}
	}

	if exePath == "" {
		return "", errors.New("failed to get kiro executable path")
	}

	// 返回目錄部分
	return filepath.Dir(exePath), nil
}

// getDarwinSpotlightPath 使用 Spotlight 搜索 Kiro.app (macOS)
func getDarwinSpotlightPath() (string, error) {
	// 使用 mdfind 搜索 Kiro.app
	cmd := exec.Command("mdfind", "kMDItemKind == 'Application' && kMDItemDisplayName == 'Kiro'")
	output, err := cmd.Output()
	if err != nil {
		return "", errors.New("spotlight search failed")
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, "Kiro.app") {
			return line, nil
		}
	}

	return "", errors.New("kiro not found via spotlight")
}

// getLinuxWhichPath 使用 which 命令搜索 kiro 執行檔 (Linux)
func getLinuxWhichPath() (string, error) {
	cmd := exec.Command("which", "kiro")
	output, err := cmd.Output()
	if err != nil {
		return "", errors.New("kiro not found in PATH")
	}

	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", errors.New("kiro not found in PATH")
	}

	// 返回目錄部分
	return filepath.Dir(path), nil
}

// getWindowsRegistryPath 非 Windows 平台不支援
func getWindowsRegistryPath() (string, error) {
	return "", errors.New("windows registry not available on this platform")
}
