package softreset

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"kiro-manager/awssso"
	"kiro-manager/kiropath"
	"kiro-manager/machineid"
)

const (
	CustomMachineIDFileName    = "custom-machine-id"     // SHA256 雜湊後的值（給 Kiro 使用）
	CustomMachineIDRawFileName = "custom-machine-id-raw" // 原始 UUID（給 UI 顯示）
)

var (
	ErrCustomIDNotFound = errors.New("custom machine ID not found")
	ErrKiroHomeNotFound = errors.New("kiro home directory not found")
)

// SoftResetResult 重置結果
type SoftResetResult struct {
	OldMachineID string `json:"oldMachineId"`
	NewMachineID string `json:"newMachineId"`
	Patched      bool   `json:"patched"`
	CacheCleared bool   `json:"cacheCleared"`
}

// SoftResetStatus 重置狀態
type SoftResetStatus struct {
	IsPatched       bool   `json:"isPatched"`
	HasCustomID     bool   `json:"hasCustomId"`
	CustomMachineID string `json:"customMachineId"`
	ExtensionPath   string `json:"extensionPath"`
}

// GetCustomMachineIDPath 取得自訂 Machine ID 檔案路徑 (~/.kiro/custom-machine-id)
func GetCustomMachineIDPath() (string, error) {
	kiroHome, err := kiropath.GetKiroHomePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(kiroHome, CustomMachineIDFileName), nil
}

// GetCustomMachineIDRawPath 取得原始 Machine ID 檔案路徑 (~/.kiro/custom-machine-id-raw)
func GetCustomMachineIDRawPath() (string, error) {
	kiroHome, err := kiropath.GetKiroHomePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(kiroHome, CustomMachineIDRawFileName), nil
}

// ReadCustomMachineID 讀取自訂 Machine ID（如果存在）
func ReadCustomMachineID() (string, error) {
	idPath, err := GetCustomMachineIDPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(idPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrCustomIDNotFound
		}
		return "", err
	}

	id := strings.TrimSpace(string(data))
	if id == "" {
		return "", ErrCustomIDNotFound
	}

	return id, nil
}

// WriteCustomMachineID 寫入自訂 Machine ID（SHA256 雜湊後的值）
func WriteCustomMachineID(machineID string) error {
	idPath, err := GetCustomMachineIDPath()
	if err != nil {
		return err
	}

	// 確保 ~/.kiro 目錄存在
	dir := filepath.Dir(idPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(idPath, []byte(machineID), 0644)
}

// ReadCustomMachineIDRaw 讀取原始 Machine ID（UUID 格式，用於 UI 顯示）
func ReadCustomMachineIDRaw() (string, error) {
	idPath, err := GetCustomMachineIDRawPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(idPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrCustomIDNotFound
		}
		return "", err
	}

	id := strings.TrimSpace(string(data))
	if id == "" {
		return "", ErrCustomIDNotFound
	}

	return id, nil
}

// WriteCustomMachineIDRaw 寫入原始 Machine ID（UUID 格式）
func WriteCustomMachineIDRaw(machineID string) error {
	idPath, err := GetCustomMachineIDRawPath()
	if err != nil {
		return err
	}

	// 確保 ~/.kiro 目錄存在
	dir := filepath.Dir(idPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(idPath, []byte(machineID), 0644)
}

// GenerateNewMachineID 生成新的 UUID v4
func GenerateNewMachineID() string {
	return strings.ToLower(uuid.New().String())
}

// ClearCustomMachineID 刪除自訂 Machine ID 檔案（還原為系統原始值）
func ClearCustomMachineID() error {
	// 刪除 SHA256 雜湊檔案
	idPath, err := GetCustomMachineIDPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(idPath); err == nil {
		if err := os.Remove(idPath); err != nil {
			return err
		}
	}

	// 刪除原始 UUID 檔案
	rawPath, err := GetCustomMachineIDRawPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(rawPath); err == nil {
		if err := os.Remove(rawPath); err != nil {
			return err
		}
	}

	return nil
}

// ClearSSOCache 刪除 SSO cache（複用 reset 模組的邏輯）
func ClearSSOCache() error {
	cachePath, err := awssso.GetSSOCachePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil
	}

	return os.RemoveAll(cachePath)
}

// SoftResetEnvironment 執行一鍵新機
func SoftResetEnvironment() (*SoftResetResult, error) {
	result := &SoftResetResult{}

	// 1. 讀取舊的原始 Machine ID（如果有，用於 UI 顯示）
	oldID, _ := ReadCustomMachineIDRaw()
	result.OldMachineID = oldID

	// 2. 生成新的 Machine ID（UUID v4）
	rawID := GenerateNewMachineID()

	// 3. 將 UUID 經過 SHA256 雜湊（Kiro 使用雜湊後的值）
	hashedID := machineid.HashMachineID(rawID)

	// 4. 返回原始 UUID（用於 UI 顯示）
	result.NewMachineID = rawID

	// 5. 寫入自訂 Machine ID 檔案（雜湊後的值，給 Kiro 使用）
	if err := WriteCustomMachineID(hashedID); err != nil {
		return result, err
	}

	// 6. 寫入原始 Machine ID 檔案（UUID 格式，給 UI 顯示）
	if err := WriteCustomMachineIDRaw(rawID); err != nil {
		return result, err
	}

	// 7. Patch extension.js（如果尚未 patch）
	patched, err := IsPatched()
	if err != nil {
		return result, err
	}

	if !patched {
		if err := PatchExtensionJS(); err != nil {
			return result, err
		}
		result.Patched = true
	} else {
		result.Patched = true // 已經 patch 過
	}

	// 5. 清除 SSO cache
	if err := ClearSSOCache(); err != nil {
		return result, err
	}
	result.CacheCleared = true

	return result, nil
}

// RestoreOriginalMachineID 還原為系統原始 Machine ID
func RestoreOriginalMachineID() error {
	// 1. 刪除自訂 Machine ID 檔案
	if err := ClearCustomMachineID(); err != nil {
		return err
	}

	// 2. 從備份還原 extension.js
	if err := RestoreExtensionJS(); err != nil {
		// 如果備份不存在，嘗試移除 patch
		if err == ErrBackupNotFound {
			_ = UnpatchExtensionJS() // 忽略錯誤
		} else if err != ErrExtensionNotFound {
			return err
		}
	}

	// 注意：SSO cache 的恢復邏輯由呼叫端（app.go）處理
	// 因為需要比對備份的 Machine ID，這是 backup 模組的職責

	return nil
}

// GetSoftResetStatus 取得重置狀態
func GetSoftResetStatus() (*SoftResetStatus, error) {
	status := &SoftResetStatus{}

	// 檢查是否已 patch
	patched, err := IsPatched()
	if err == nil {
		status.IsPatched = patched
	}

	// 檢查自訂 Machine ID（優先讀取原始 UUID，用於 UI 顯示）
	rawID, err := ReadCustomMachineIDRaw()
	if err == nil {
		status.HasCustomID = true
		status.CustomMachineID = rawID
	} else {
		// 向後兼容：如果沒有 raw 檔案，嘗試讀取雜湊檔案
		hashedID, err := ReadCustomMachineID()
		if err == nil {
			status.HasCustomID = true
			status.CustomMachineID = hashedID
		}
	}

	// 取得 extension.js 路徑
	extPath, err := GetExtensionJSPath()
	if err == nil {
		status.ExtensionPath = extPath
	}

	return status, nil
}
