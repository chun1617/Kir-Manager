package kiropath

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"kiro-manager/settings"
)

var (
	ErrKiroNotFound      = errors.New("kiro installation not found")
	ErrUnsupportedPlatform = errors.New("unsupported platform: " + runtime.GOOS)
)

// GetKiroHomePath 取得 Kiro 的使用者設定目錄 (~/.kiro)
func GetKiroHomePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	kiroHome := filepath.Join(homeDir, ".kiro")
	return kiroHome, nil
}

// GetKiroConfigPath 取得 Kiro 應用程式的設定目錄
// Windows: %APPDATA%\Kiro
// macOS: ~/Library/Application Support/Kiro
// Linux: ~/.config/Kiro
func GetKiroConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Kiro"), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Kiro"), nil
	case "linux":
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(homeDir, ".config")
		}
		return filepath.Join(configDir, "Kiro"), nil
	default:
		return "", ErrUnsupportedPlatform
	}
}


// GetKiroInstallPath 取得 Kiro 的安裝路徑
// 優先使用自定義路徑，若未設定則自動偵測
// Windows: 檢查 %LOCALAPPDATA%\Programs\Kiro 和 %PROGRAMFILES%\Kiro
// macOS: /Applications/Kiro.app
// Linux: /usr/share/kiro, /opt/kiro, 或 /usr/local/bin/kiro
func GetKiroInstallPath() (string, error) {
	// 優先使用自定義路徑
	customPath := settings.GetCustomKiroInstallPath()
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			return customPath, nil
		}
		// 自定義路徑無效，繼續嘗試自動偵測
	}

	switch runtime.GOOS {
	case "windows":
		return getWindowsKiroInstallPath()
	case "darwin":
		return getDarwinKiroInstallPath()
	case "linux":
		return getLinuxKiroInstallPath()
	default:
		return "", ErrUnsupportedPlatform
	}
}

// GetKiroInstallPathAutoDetect 自動偵測 Kiro 安裝路徑（忽略自定義設定）
func GetKiroInstallPathAutoDetect() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsKiroInstallPath()
	case "darwin":
		return getDarwinKiroInstallPath()
	case "linux":
		return getLinuxKiroInstallPath()
	default:
		return "", ErrUnsupportedPlatform
	}
}

// IsKiroInstalled 檢查 Kiro 是否已安裝
func IsKiroInstalled() bool {
	path, err := GetKiroInstallPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// KiroHomeExists 檢查 Kiro 設定目錄是否存在
func KiroHomeExists() bool {
	path, err := GetKiroHomePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// KiroConfigExists 檢查 Kiro 應用程式設定目錄是否存在
func KiroConfigExists() bool {
	path, err := GetKiroConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// GetAWSConfigPath 取得 AWS CLI 的設定目錄 (~/.aws)
func GetAWSConfigPath() (string, error) {
	// 優先使用 AWS_CONFIG_FILE 環境變數的目錄
	if awsConfigFile := os.Getenv("AWS_CONFIG_FILE"); awsConfigFile != "" {
		return filepath.Dir(awsConfigFile), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".aws"), nil
}

// AWSConfigExists 檢查 AWS 設定目錄是否存在
func AWSConfigExists() bool {
	path, err := GetAWSConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func getWindowsKiroInstallPath() (string, error) {
	// 優先檢查使用者安裝路徑
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		userPath := filepath.Join(localAppData, "Programs", "Kiro", "Kiro.exe")
		if _, err := os.Stat(userPath); err == nil {
			return filepath.Dir(userPath), nil
		}
	}

	// 檢查系統安裝路徑
	programFiles := os.Getenv("PROGRAMFILES")
	if programFiles != "" {
		systemPath := filepath.Join(programFiles, "Kiro", "Kiro.exe")
		if _, err := os.Stat(systemPath); err == nil {
			return filepath.Dir(systemPath), nil
		}
	}

	// 檢查 x86 程式目錄
	programFilesX86 := os.Getenv("PROGRAMFILES(X86)")
	if programFilesX86 != "" {
		x86Path := filepath.Join(programFilesX86, "Kiro", "Kiro.exe")
		if _, err := os.Stat(x86Path); err == nil {
			return filepath.Dir(x86Path), nil
		}
	}

	return "", ErrKiroNotFound
}

func getDarwinKiroInstallPath() (string, error) {
	// 標準應用程式目錄
	appPath := "/Applications/Kiro.app"
	if _, err := os.Stat(appPath); err == nil {
		return appPath, nil
	}

	// 使用者應用程式目錄
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userAppPath := filepath.Join(homeDir, "Applications", "Kiro.app")
		if _, err := os.Stat(userAppPath); err == nil {
			return userAppPath, nil
		}
	}

	return "", ErrKiroNotFound
}

func getLinuxKiroInstallPath() (string, error) {
	// 常見的 Linux 安裝路徑
	paths := []string{
		"/usr/share/kiro",
		"/opt/kiro",
		"/usr/local/share/kiro",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// 檢查使用者本地安裝
	homeDir, err := os.UserHomeDir()
	if err == nil {
		localPath := filepath.Join(homeDir, ".local", "share", "kiro")
		if _, err := os.Stat(localPath); err == nil {
			return localPath, nil
		}
	}

	return "", ErrKiroNotFound
}
