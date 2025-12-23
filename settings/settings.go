package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const (
	// 設定檔名稱
	SettingsFileName = "settings.json"
	// 預設低餘額閾值（20%）
	DefaultLowBalanceThreshold = 0.2
	// 預設 Kiro IDE 版本號
	DefaultKiroVersion = "0.7.5"
)

// Settings 全域設定結構
type Settings struct {
	// LowBalanceThreshold 低餘額閾值（0.0 ~ 1.0）
	// 當餘額比率低於此值時，顯示低餘額警告
	LowBalanceThreshold float64 `json:"lowBalanceThreshold"`
	// KiroVersion Kiro IDE 版本號
	// 用於 API 請求的 User-Agent header（當 UseAutoDetect 為 false 時使用）
	KiroVersion string `json:"kiroVersion,omitempty"`
	// UseAutoDetect 是否使用自動偵測的版本號
	// true: 每次 API 請求時自動偵測 Kiro 執行檔版本
	// false: 使用 KiroVersion 欄位的自定義值
	UseAutoDetect bool `json:"useAutoDetect"`
	// CustomKiroInstallPath 自定義 Kiro 安裝路徑
	// 當自動偵測失敗時，使用此路徑
	// 空字串表示使用自動偵測
	CustomKiroInstallPath string `json:"customKiroInstallPath,omitempty"`
}

var (
	currentSettings *Settings
	settingsMutex   sync.RWMutex
)

// GetSettingsPath 取得設定檔路徑（執行檔同層）
func GetSettingsPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	execDir := filepath.Dir(execPath)
	return filepath.Join(execDir, SettingsFileName), nil
}

// LoadSettings 載入設定
// 如果設定檔不存在，返回預設設定
func LoadSettings() (*Settings, error) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	settingsPath, err := GetSettingsPath()
	if err != nil {
		return getDefaultSettings(), nil
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			currentSettings = getDefaultSettings()
			return currentSettings, nil
		}
		return getDefaultSettings(), nil
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return getDefaultSettings(), nil
	}

	// 驗證並修正設定值
	settings = validateSettings(settings)
	currentSettings = &settings
	return currentSettings, nil
}

// SaveSettings 儲存設定
func SaveSettings(settings *Settings) error {
	if settings == nil {
		return nil
	}

	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	// 驗證並修正設定值
	validated := validateSettings(*settings)
	settings = &validated

	settingsPath, err := GetSettingsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return err
	}

	currentSettings = settings
	return nil
}

// GetCurrentSettings 取得當前設定（快取）
// 如果尚未載入，會自動載入
func GetCurrentSettings() *Settings {
	settingsMutex.RLock()
	if currentSettings != nil {
		defer settingsMutex.RUnlock()
		return currentSettings
	}
	settingsMutex.RUnlock()

	// 尚未載入，執行載入
	settings, _ := LoadSettings()
	return settings
}

// GetLowBalanceThreshold 取得低餘額閾值
func GetLowBalanceThreshold() float64 {
	settings := GetCurrentSettings()
	if settings == nil {
		return DefaultLowBalanceThreshold
	}
	return settings.LowBalanceThreshold
}

// GetKiroVersion 取得 Kiro IDE 版本號（自定義值）
// 注意：此函數僅返回設定中的版本號，不處理自動偵測邏輯
// 呼叫端應先檢查 IsAutoDetectEnabled() 決定是否使用自動偵測
func GetKiroVersion() string {
	settings := GetCurrentSettings()
	if settings == nil || settings.KiroVersion == "" {
		return DefaultKiroVersion
	}
	return settings.KiroVersion
}

// IsAutoDetectEnabled 檢查是否啟用自動偵測版本號
func IsAutoDetectEnabled() bool {
	settings := GetCurrentSettings()
	if settings == nil {
		return true // 預設啟用自動偵測
	}
	return settings.UseAutoDetect
}

// GetCustomKiroInstallPath 取得自定義 Kiro 安裝路徑
// 返回空字串表示使用自動偵測
func GetCustomKiroInstallPath() string {
	settings := GetCurrentSettings()
	if settings == nil {
		return ""
	}
	return settings.CustomKiroInstallPath
}

// getDefaultSettings 取得預設設定
func getDefaultSettings() *Settings {
	return &Settings{
		LowBalanceThreshold: DefaultLowBalanceThreshold,
		KiroVersion:         DefaultKiroVersion,
		UseAutoDetect:       true, // 預設使用自動偵測
	}
}

// validateSettings 驗證並修正設定值
func validateSettings(settings Settings) Settings {
	// LowBalanceThreshold 必須在 0.0 ~ 1.0 之間
	if settings.LowBalanceThreshold < 0 {
		settings.LowBalanceThreshold = 0
	}
	if settings.LowBalanceThreshold > 1 {
		settings.LowBalanceThreshold = 1
	}
	// KiroVersion 為空時使用預設值
	if settings.KiroVersion == "" {
		settings.KiroVersion = DefaultKiroVersion
	}
	return settings
}
