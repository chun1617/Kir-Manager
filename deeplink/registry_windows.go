//go:build windows

package deeplink

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"kiro-manager/internal/cmdutil"
)

// Registry 路徑常數
const (
	// schemeRegPath 是 URL Scheme 的 Registry 路徑
	schemeRegPath = `HKCU\Software\Classes\` + URLScheme

	// commandRegPath 是 shell open command 的 Registry 路徑
	commandRegPath = schemeRegPath + `\shell\open\command`

	// schemeDescription 是 URL Scheme 的描述
	schemeDescription = "URL:Kiro Manager Protocol"
)

// IsDeepLinkSupported 檢查當前平台是否支援 Deep Link
// Windows 平台返回 true
func IsDeepLinkSupported() bool {
	return true
}

// IsURLSchemeRegistered 檢查 URL Scheme 是否已註冊
func IsURLSchemeRegistered() (bool, error) {
	// 查詢 scheme key 是否存在
	args := buildRegQueryArgs(schemeRegPath)
	cmd := exec.Command("reg", args...)
	cmdutil.HideWindow(cmd)

	output, err := cmd.Output()
	if err != nil {
		// 如果 key 不存在，reg query 會返回錯誤
		return false, nil
	}

	// 檢查輸出是否包含有效值
	_, parseErr := parseRegistryDefaultValue(string(output))
	if parseErr != nil {
		return false, nil
	}

	return true, nil
}

// GetRegisteredExePath 取得已註冊的執行檔路徑
func GetRegisteredExePath() (string, error) {
	// 查詢 command key
	args := buildRegQueryArgs(commandRegPath)
	cmd := exec.Command("reg", args...)
	cmdutil.HideWindow(cmd)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("URL scheme not registered: %w", err)
	}

	// 解析 command 值
	commandVal, err := parseRegistryDefaultValue(string(output))
	if err != nil {
		return "", fmt.Errorf("failed to parse registry value: %w", err)
	}

	// 從 command 值提取執行檔路徑
	exePath, err := extractExePathFromCommand(commandVal)
	if err != nil {
		return "", fmt.Errorf("failed to extract exe path: %w", err)
	}

	return exePath, nil
}

// EnsureURLSchemeRegistered 確保 URL Scheme 已註冊
// 若未註冊則自動註冊，若路徑不同則更新
func EnsureURLSchemeRegistered() error {
	// 取得當前執行檔路徑
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// 檢查是否已註冊
	registered, err := IsURLSchemeRegistered()
	if err != nil {
		return err
	}

	if registered {
		// 檢查路徑是否相同
		registeredPath, err := GetRegisteredExePath()
		if err == nil && registeredPath == exePath {
			// 已註冊且路徑相同，無需更新
			return nil
		}
		// 路徑不同，需要更新
	}

	// 註冊 URL Scheme
	return registerURLScheme(exePath)
}

// registerURLScheme 寫入 Registry 註冊 URL Scheme
func registerURLScheme(exePath string) error {
	// 1. 設定 scheme key 的 default value
	if err := regAdd(schemeRegPath, schemeDescription); err != nil {
		return fmt.Errorf("%w: failed to set scheme description: %v", ErrRegistryFailed, err)
	}

	// 2. 設定 URL Protocol 值（空字串表示這是 URL Protocol）
	if err := regAddValue(schemeRegPath, "URL Protocol", ""); err != nil {
		return fmt.Errorf("%w: failed to set URL Protocol: %v", ErrRegistryFailed, err)
	}

	// 3. 設定 command key 的 default value
	commandVal := buildCommandValue(exePath)
	if err := regAdd(commandRegPath, commandVal); err != nil {
		return fmt.Errorf("%w: failed to set command: %v", ErrRegistryFailed, err)
	}

	return nil
}

// regAdd 使用 reg add 設定 Registry key 的 default value
func regAdd(regPath, value string) error {
	args := buildRegAddArgs(regPath, value)
	cmd := exec.Command("reg", args...)
	cmdutil.HideWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reg add failed: %s, output: %s", err, string(output))
	}

	return nil
}

// regAddValue 使用 reg add 設定 Registry key 的指定值
func regAddValue(regPath, valueName, value string) error {
	args := []string{
		"add",
		regPath,
		"/v", valueName,
		"/t", "REG_SZ",
		"/d", value,
		"/f",
	}
	cmd := exec.Command("reg", args...)
	cmdutil.HideWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reg add failed: %s, output: %s", err, string(output))
	}

	return nil
}

// buildRegAddArgs 建構 reg add 命令參數（設定 default value）
func buildRegAddArgs(regPath, value string) []string {
	return []string{
		"add",
		regPath,
		"/ve", // 設定 default value
		"/t", "REG_SZ",
		"/d", value,
		"/f", // 強制覆寫
	}
}

// buildRegQueryArgs 建構 reg query 命令參數（查詢 default value）
func buildRegQueryArgs(regPath string) []string {
	return []string{
		"query",
		regPath,
		"/ve", // 查詢 default value
	}
}

// buildCommandValue 建構 shell open command 的值
func buildCommandValue(exePath string) string {
	return fmt.Sprintf(`"%s" "%%1"`, exePath)
}

// parseRegistryDefaultValue 解析 reg query 輸出，提取 default value
// 輸出格式範例:
//
//	HKEY_CURRENT_USER\Software\Classes\kiro
//	    (Default)    REG_SZ    URL:Kiro Manager Protocol
func parseRegistryDefaultValue(output string) (string, error) {
	if output == "" {
		return "", errors.New("empty registry output")
	}

	// 檢查是否有錯誤訊息
	if strings.Contains(output, "ERROR:") {
		return "", errors.New("registry key not found")
	}

	// 正則表達式匹配 default value
	// 格式: "    (Default)    REG_SZ    ValueData"
	// 注意：(Default) 在不同語言的 Windows 可能顯示不同，但 REG_SZ 是固定的
	defaultValueRe := regexp.MustCompile(`(?i)^\s*\(Default\)\s+REG_SZ\s+(.*)$`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		if matches := defaultValueRe.FindStringSubmatch(line); len(matches) > 1 {
			value := strings.TrimSpace(matches[1])
			return value, nil
		}
	}

	return "", errors.New("no default value found in registry output")
}

// extractExePathFromCommand 從 command 值提取執行檔路徑
// command 值格式: "C:\Path\To\exe.exe" "%1"
func extractExePathFromCommand(commandVal string) (string, error) {
	if commandVal == "" {
		return "", errors.New("empty command value")
	}

	// 檢查是否以引號開頭
	if !strings.HasPrefix(commandVal, `"`) {
		return "", errors.New("invalid command format: expected quoted path")
	}

	// 找到第二個引號的位置
	endQuote := strings.Index(commandVal[1:], `"`)
	if endQuote == -1 {
		return "", errors.New("invalid command format: missing closing quote")
	}

	// 提取引號內的路徑
	exePath := commandVal[1 : endQuote+1]

	return exePath, nil
}
