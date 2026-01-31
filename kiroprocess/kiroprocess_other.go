//go:build !windows

package kiroprocess

import (
	"os/exec"
	"strconv"
	"strings"
)

// getWindowsKiroProcesses 非 Windows 平台不支援
func getWindowsKiroProcesses() ([]ProcessInfo, error) {
	return nil, ErrUnsupportedPlatform
}

// killWindowsProcess 非 Windows 平台不支援
func killWindowsProcess(pid int) error {
	return ErrUnsupportedPlatform
}

func getDarwinKiroProcesses() ([]ProcessInfo, error) {
	cmd := exec.Command("pgrep", "-l", "Kiro")
	output, err := cmd.Output()
	if err != nil {
		return []ProcessInfo{}, nil
	}
	return parseUnixPgrep(string(output))
}

func getLinuxKiroProcesses() ([]ProcessInfo, error) {
	cmd := exec.Command("pgrep", "-l", "-i", "kiro")
	output, err := cmd.Output()
	if err != nil {
		return []ProcessInfo{}, nil
	}
	return parseUnixPgrep(string(output))
}

func parseUnixPgrep(output string) ([]ProcessInfo, error) {
	var processes []ProcessInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 2 {
			pid, err := strconv.Atoi(parts[0])
			if err == nil {
				processes = append(processes, ProcessInfo{
					PID:  pid,
					Name: parts[1],
				})
			}
		}
	}

	return processes, nil
}

// killUnixProcess 使用 kill 命令終止進程
func killUnixProcess(pid int) error {
	cmd := exec.Command("kill", "-9", strconv.Itoa(pid))
	return cmd.Run()
}

// getWindowsKiroExecutablePath 非 Windows 平台不支援
func getWindowsKiroExecutablePath() (string, error) {
	return "", ErrUnsupportedPlatform
}

// getDarwinKiroExecutablePath 使用 lsof 取得 Kiro 進程的執行檔路徑 (macOS)
func getDarwinKiroExecutablePath() (string, error) {
	processes, err := getDarwinKiroProcesses()
	if err != nil {
		return "", err
	}
	if len(processes) == 0 {
		return "", ErrProcessNotFound
	}

	// 使用 lsof 取得進程的執行檔路徑
	cmd := exec.Command("lsof", "-p", strconv.Itoa(processes[0].PID), "-Fn")
	output, err := cmd.Output()
	if err != nil {
		return "", ErrProcessNotFound
	}

	// 解析 lsof 輸出，找到 txt 類型的執行檔
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "n") && strings.Contains(line, "Kiro") {
			return strings.TrimPrefix(line, "n"), nil
		}
	}

	return "", ErrProcessNotFound
}

// getLinuxKiroExecutablePath 使用 /proc 取得 Kiro 進程的執行檔路徑 (Linux)
func getLinuxKiroExecutablePath() (string, error) {
	processes, err := getLinuxKiroProcesses()
	if err != nil {
		return "", err
	}
	if len(processes) == 0 {
		return "", ErrProcessNotFound
	}

	// 讀取 /proc/[pid]/exe 符號連結
	exePath := "/proc/" + strconv.Itoa(processes[0].PID) + "/exe"
	cmd := exec.Command("readlink", "-f", exePath)
	output, err := cmd.Output()
	if err != nil {
		return "", ErrProcessNotFound
	}

	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", ErrProcessNotFound
	}

	return path, nil
}
