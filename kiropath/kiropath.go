package kiropath

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"kiro-manager/settings"
)

func init() {
	// 註冊路徑快取失效回調，當設定變更時自動清除快取
	settings.SetPathCacheInvalidator(InvalidatePathCache)
}

var (
	ErrKiroNotFound        = errors.New("kiro installation not found")
	ErrUnsupportedPlatform = errors.New("unsupported platform: " + runtime.GOOS)
)

// DetectionFailedError 表示所有偵測策略都失敗
type DetectionFailedError struct {
	TriedStrategies []string
	FailureReasons  map[string]string
}

func (e *DetectionFailedError) Error() string {
	return "all detection strategies failed"
}

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
// 使用優先級偵測鏈：
// 1. 快取 - 如果已有快取的路徑，直接返回
// 2. 自定義路徑 - 用戶設定的路徑
// 3. Running Process - 從運行中的 Kiro 進程提取
// 4. 平台特定偵測 - Registry/Spotlight/which
// 5. 硬編碼路徑列表 - 常見安裝位置
// 6. PATH 環境變數 - 從 PATH 中搜索
func GetKiroInstallPath() (string, error) {
	// 1. 檢查快取
	if cached := getPathCache(); cached != "" {
		return cached, nil
	}

	// 追蹤嘗試過的策略和失敗原因
	triedStrategies := []string{}
	failureReasons := map[string]string{}

	// 2.1 自定義路徑
	triedStrategies = append(triedStrategies, "custom")
	customPath := settings.GetCustomKiroInstallPath()
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			setPathCache(customPath)
			return customPath, nil
		}
		failureReasons["custom"] = "path does not exist"
	} else {
		failureReasons["custom"] = "not configured"
	}

	// 2.2 Running Process 偵測
	triedStrategies = append(triedStrategies, "process")
	if processPath, err := getRunningProcessPath(); err == nil && processPath != "" {
		setPathCache(processPath)
		return processPath, nil
	} else if err != nil {
		failureReasons["process"] = err.Error()
	} else {
		failureReasons["process"] = "kiro not running"
	}

	// 2.3 平台特定偵測 (Registry/Spotlight/which)
	triedStrategies = append(triedStrategies, "platform")
	if platformPath, err := getPlatformSpecificPath(); err == nil && platformPath != "" {
		setPathCache(platformPath)
		return platformPath, nil
	} else if err != nil {
		failureReasons["platform"] = err.Error()
	} else {
		failureReasons["platform"] = "not found"
	}

	// 2.4 硬編碼路徑列表
	triedStrategies = append(triedStrategies, "hardcoded")
	if hardcodedPath, err := getHardcodedPath(); err == nil && hardcodedPath != "" {
		setPathCache(hardcodedPath)
		return hardcodedPath, nil
	} else if err != nil {
		failureReasons["hardcoded"] = err.Error()
	} else {
		failureReasons["hardcoded"] = "not found"
	}

	// 2.5 PATH 環境變數
	triedStrategies = append(triedStrategies, "path")
	if pathEnvPath, err := searchInPath(); err == nil && pathEnvPath != "" {
		setPathCache(pathEnvPath)
		return pathEnvPath, nil
	} else if err != nil {
		failureReasons["path"] = err.Error()
	} else {
		failureReasons["path"] = "not in PATH"
	}

	// 3. 所有策略失敗
	return "", &DetectionFailedError{
		TriedStrategies: triedStrategies,
		FailureReasons:  failureReasons,
	}
}

// getRunningProcessPath 從運行中的 Kiro 進程取得安裝路徑
func getRunningProcessPath() (string, error) {
	// 延遲導入以避免循環依賴，使用內部實作
	return getRunningProcessPathInternal()
}

// getPlatformSpecificPath 使用平台特定方式偵測 Kiro 安裝路徑
func getPlatformSpecificPath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsRegistryPath()
	case "darwin":
		return getDarwinSpotlightPath()
	case "linux":
		return getLinuxWhichPath()
	default:
		return "", ErrUnsupportedPlatform
	}
}

// getHardcodedPath 從硬編碼的路徑列表中搜索 Kiro
func getHardcodedPath() (string, error) {
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

// searchInPath 從 PATH 環境變數中搜索 Kiro 執行檔
// 遍歷 PATH 中的每個目錄，檢查是否存在 kiro 或 Kiro.exe
// 返回找到的執行檔所在目錄
func searchInPath() (string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return "", ErrKiroNotFound
	}

	// 根據平台決定執行檔名稱
	var execName string
	switch runtime.GOOS {
	case "windows":
		execName = "Kiro.exe"
	default:
		execName = "kiro"
	}

	// 使用 filepath.SplitList 分割 PATH（自動處理平台差異）
	dirs := filepath.SplitList(pathEnv)

	for _, dir := range dirs {
		if dir == "" {
			continue
		}

		// 檢查目錄是否存在
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		// 檢查執行檔是否存在
		execPath := filepath.Join(dir, execName)
		if _, err := os.Stat(execPath); err == nil {
			return dir, nil
		}
	}

	return "", ErrKiroNotFound
}
