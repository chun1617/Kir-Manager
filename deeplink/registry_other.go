//go:build !windows

package deeplink

// IsURLSchemeRegistered 檢查 URL Scheme 是否已註冊
// 非 Windows 平台不支援，返回 false 和 ErrNotWindows
func IsURLSchemeRegistered() (bool, error) {
	return false, ErrNotWindows
}

// GetRegisteredExePath 取得已註冊的執行檔路徑
// 非 Windows 平台不支援
func GetRegisteredExePath() (string, error) {
	return "", ErrNotWindows
}

// EnsureURLSchemeRegistered 確保 URL Scheme 已註冊
// 非 Windows 平台不支援
func EnsureURLSchemeRegistered() error {
	return ErrNotWindows
}

// IsDeepLinkSupported 檢查當前平台是否支援 Deep Link
// 非 Windows 平台返回 false
func IsDeepLinkSupported() bool {
	return false
}
