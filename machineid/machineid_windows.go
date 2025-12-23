//go:build windows

package machineid

import (
	"errors"
	"os/exec"
	"strings"

	"kiro-manager/internal/cmdutil"
)

// getWindowsMachineId 使用 reg query 命令讀取 Registry 中的 MachineGuid
// 使用系統內建工具避免防毒軟體誤報
func getWindowsMachineId() (string, error) {
	// reg query "HKLM\SOFTWARE\Microsoft\Cryptography" /v MachineGuid
	cmd := exec.Command("reg", "query",
		`HKLM\SOFTWARE\Microsoft\Cryptography`,
		"/v", "MachineGuid")
	cmdutil.HideWindow(cmd)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// 輸出格式:
	// HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography
	//     MachineGuid    REG_SZ    xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "MachineGuid") && strings.Contains(line, "REG_SZ") {
			// 分割並取得最後一個欄位（UUID）
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return strings.ToLower(parts[len(parts)-1]), nil
			}
		}
	}

	return "", errors.New("MachineGuid not found in registry")
}

// getDarwinMachineId Windows 平台不支援
func getDarwinMachineId() (string, error) {
	return "", errors.New("Darwin-only function called on Windows")
}

// getLinuxMachineId Windows 平台不支援
func getLinuxMachineId() (string, error) {
	return "", errors.New("Linux-only function called on Windows")
}
