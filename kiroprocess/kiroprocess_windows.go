//go:build windows

package kiroprocess

import (
	"os/exec"
	"strconv"
	"strings"

	"kiro-manager/internal/cmdutil"
)

// getWindowsKiroProcesses 使用 tasklist 命令取得 Kiro 進程列表
// 使用系統內建工具避免防毒軟體誤報
func getWindowsKiroProcesses() ([]ProcessInfo, error) {
	// tasklist /FI "IMAGENAME eq Kiro.exe" /FO CSV /NH
	// 輸出格式: "Kiro.exe","12345","Console","1","123,456 K"
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq Kiro.exe", "/FO", "CSV", "/NH")
	cmdutil.HideWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		// tasklist 在找不到進程時會回傳錯誤，這是正常的
		// 檢查輸出是否包含 "INFO: No tasks"
		if strings.Contains(string(output), "INFO:") {
			return []ProcessInfo{}, nil
		}
		// 如果是其他錯誤，嘗試解析輸出
		if len(output) == 0 {
			return []ProcessInfo{}, nil
		}
	}

	return parseTasklistOutput(string(output))
}

// parseTasklistOutput 解析 tasklist CSV 輸出
func parseTasklistOutput(output string) ([]ProcessInfo, error) {
	var processes []ProcessInfo

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 跳過 INFO 訊息（例如 "INFO: No tasks are running..."）
		if strings.HasPrefix(line, "INFO:") {
			continue
		}

		// CSV 格式: "Kiro.exe","12345","Console","1","123,456 K"
		// 移除引號並分割
		fields := parseCSVLine(line)
		if len(fields) < 2 {
			continue
		}

		name := fields[0]
		pidStr := fields[1]

		// 確認是 Kiro 進程
		if !strings.EqualFold(name, "Kiro.exe") {
			continue
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		processes = append(processes, ProcessInfo{
			PID:  pid,
			Name: name,
		})
	}

	return processes, nil
}

// parseCSVLine 解析 CSV 行，處理引號
func parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, r := range line {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if inQuotes {
				current.WriteRune(r)
			} else {
				fields = append(fields, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	// 加入最後一個欄位
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}

// killWindowsProcess 使用 taskkill 命令終止指定 PID 的進程
// 使用系統內建工具避免防毒軟體誤報
func killWindowsProcess(pid int) error {
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F")
	cmdutil.HideWindow(cmd)
	return cmd.Run()
}

// getDarwinKiroProcesses Windows 平台不支援
func getDarwinKiroProcesses() ([]ProcessInfo, error) {
	return nil, ErrUnsupportedPlatform
}

// getLinuxKiroProcesses Windows 平台不支援
func getLinuxKiroProcesses() ([]ProcessInfo, error) {
	return nil, ErrUnsupportedPlatform
}

// killUnixProcess Windows 平台不支援
func killUnixProcess(pid int) error {
	return ErrUnsupportedPlatform
}
