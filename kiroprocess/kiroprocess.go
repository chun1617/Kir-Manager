package kiroprocess

import (
	"errors"
	"runtime"
)

var (
	ErrUnsupportedPlatform = errors.New("unsupported platform: " + runtime.GOOS)
	ErrProcessNotFound     = errors.New("kiro process not found")
)

// ProcessInfo 包含進程的基本資訊
type ProcessInfo struct {
	PID     int    `json:"pid"`
	Name    string `json:"name"`
	ExePath string `json:"exePath"` // 執行檔完整路徑
}

// IsKiroRunning 檢查 Kiro 是否正在運行
func IsKiroRunning() bool {
	processes, err := GetKiroProcesses()
	if err != nil {
		return false
	}
	return len(processes) > 0
}

// GetKiroProcesses 取得所有正在運行的 Kiro 進程
func GetKiroProcesses() ([]ProcessInfo, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsKiroProcesses()
	case "darwin":
		return getDarwinKiroProcesses()
	case "linux":
		return getLinuxKiroProcesses()
	default:
		return nil, ErrUnsupportedPlatform
	}
}

// GetKiroProcessCount 取得 Kiro 進程數量
func GetKiroProcessCount() int {
	processes, err := GetKiroProcesses()
	if err != nil {
		return 0
	}
	return len(processes)
}

// KillKiroProcesses 關閉所有 Kiro 進程
// Windows 使用原生 API，其他平台使用 kill 命令
// 回傳被關閉的進程數量和錯誤
func KillKiroProcesses() (int, error) {
	processes, err := GetKiroProcesses()
	if err != nil {
		return 0, err
	}

	if len(processes) == 0 {
		return 0, nil
	}

	killed := 0
	for _, p := range processes {
		var killErr error
		switch runtime.GOOS {
		case "windows":
			killErr = killWindowsProcess(p.PID)
		default:
			killErr = killUnixProcess(p.PID)
		}

		if killErr == nil {
			killed++
		}
	}

	return killed, nil
}

// GetKiroExecutablePath 從運行中的 Kiro 進程取得執行檔完整路徑
// 如果 Kiro 未運行，返回 ErrProcessNotFound
func GetKiroExecutablePath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsKiroExecutablePath()
	case "darwin":
		return getDarwinKiroExecutablePath()
	case "linux":
		return getLinuxKiroExecutablePath()
	default:
		return "", ErrUnsupportedPlatform
	}
}
