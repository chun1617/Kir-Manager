package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kiro-manager/awssso"
	"kiro-manager/machineid"
	"kiro-manager/softreset"
)

const (
	BackupDirName       = "backups"
	MachineIDFileName   = "machine-id.json"
	KiroAuthTokenFile   = "kiro-auth-token.json"
	UsageCacheFileName  = "usage-cache.json"
)

var (
	ErrBackupNotFound    = errors.New("backup not found")
	ErrBackupExists      = errors.New("backup already exists")
	ErrInvalidBackupName = errors.New("invalid backup name")
	ErrNoTokenToBackup   = errors.New("no kiro auth token to backup")
)

// MachineIDBackup 代表備份的 Machine ID 結構
type MachineIDBackup struct {
	MachineID  string `json:"machineId"`
	BackupTime string `json:"backupTime"`
}

// BackupInfo 代表備份的基本資訊
type BackupInfo struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	BackupTime time.Time `json:"backupTime"`
	HasToken   bool      `json:"hasToken"`
	HasMachineID bool    `json:"hasMachineId"`
}

// UsageCache 餘額緩存結構
type UsageCache struct {
	SubscriptionTitle string    `json:"subscriptionTitle"`
	UsageLimit        float64   `json:"usageLimit"`
	CurrentUsage      float64   `json:"currentUsage"`
	Balance           float64   `json:"balance"`
	IsLowBalance      bool      `json:"isLowBalance"`
	CachedAt          time.Time `json:"cachedAt"`
}

// GetBackupRootPath 取得備份根目錄（執行檔同層的 backups 資料夾）
func GetBackupRootPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	execDir := filepath.Dir(execPath)
	return filepath.Join(execDir, BackupDirName), nil
}


// ensureBackupRoot 確保備份根目錄存在
func ensureBackupRoot() (string, error) {
	rootPath, err := GetBackupRootPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		return "", err
	}
	return rootPath, nil
}

// GetBackupPath 取得指定備份的完整路徑
func GetBackupPath(name string) (string, error) {
	if name == "" {
		return "", ErrInvalidBackupName
	}
	rootPath, err := GetBackupRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootPath, name), nil
}

// BackupExists 檢查指定名稱的備份是否存在
func BackupExists(name string) bool {
	backupPath, err := GetBackupPath(name)
	if err != nil {
		return false
	}
	info, err := os.Stat(backupPath)
	return err == nil && info.IsDir()
}

// ListBackups 列出所有備份
func ListBackups() ([]BackupInfo, error) {
	rootPath, err := GetBackupRootPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		backupPath := filepath.Join(rootPath, entry.Name())
		info := BackupInfo{
			Name: entry.Name(),
			Path: backupPath,
		}

		// 檢查是否有 token 檔案
		tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
		if _, err := os.Stat(tokenPath); err == nil {
			info.HasToken = true
		}

		// 檢查是否有 machine-id 檔案並讀取備份時間
		machineIDPath := filepath.Join(backupPath, MachineIDFileName)
		if data, err := os.ReadFile(machineIDPath); err == nil {
			info.HasMachineID = true
			var mid MachineIDBackup
			if json.Unmarshal(data, &mid) == nil && mid.BackupTime != "" {
				if t, err := time.Parse(time.RFC3339, mid.BackupTime); err == nil {
					info.BackupTime = t
				}
			}
		}

		backups = append(backups, info)
	}

	return backups, nil
}


// CreateBackup 創建一個新的備份
func CreateBackup(name string) error {
	if name == "" {
		return ErrInvalidBackupName
	}

	if BackupExists(name) {
		return ErrBackupExists
	}

	// 確保備份根目錄存在
	_, err := ensureBackupRoot()
	if err != nil {
		return fmt.Errorf("failed to create backup root: %w", err)
	}

	// 創建備份資料夾
	backupPath, err := GetBackupPath(name)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 備份 kiro-auth-token.json
	tokenSrcPath, err := awssso.GetKiroAuthTokenPath()
	if err != nil {
		// 清理已創建的資料夾
		os.RemoveAll(backupPath)
		return fmt.Errorf("failed to get token path: %w", err)
	}

	if _, err := os.Stat(tokenSrcPath); os.IsNotExist(err) {
		os.RemoveAll(backupPath)
		return ErrNoTokenToBackup
	}

	tokenDstPath := filepath.Join(backupPath, KiroAuthTokenFile)
	if err := copyFile(tokenSrcPath, tokenDstPath); err != nil {
		os.RemoveAll(backupPath)
		return fmt.Errorf("failed to backup token: %w", err)
	}

	// 讀取 token 以檢查是否需要備份 IdC 的 clientIdHash 文件
	token, err := awssso.ReadKiroAuthToken()
	if err == nil && token != nil {
		// 如果是 IdC 認證且有 clientIdHash，備份對應的 clientId/clientSecret 文件
		if isIdCAuth(token.AuthMethod) && token.ClientIdHash != "" {
			clientIdHashFile := token.ClientIdHash + ".json"
			ssoCachePath, err := awssso.GetSSOCachePath()
			if err == nil {
				clientIdHashSrcPath := filepath.Join(ssoCachePath, clientIdHashFile)
				if _, err := os.Stat(clientIdHashSrcPath); err == nil {
					clientIdHashDstPath := filepath.Join(backupPath, clientIdHashFile)
					if err := copyFile(clientIdHashSrcPath, clientIdHashDstPath); err != nil {
						// 備份 clientIdHash 文件失敗不應該阻止整個備份流程，只記錄警告
						fmt.Printf("Warning: failed to backup clientIdHash file: %v\n", err)
					}
				}
			}
		}
	}

	// 備份 Machine ID
	rawMachineID, err := machineid.GetRawMachineId()
	if err != nil {
		os.RemoveAll(backupPath)
		return fmt.Errorf("failed to get machine id: %w", err)
	}

	machineIDBackup := MachineIDBackup{
		MachineID:  rawMachineID,
		BackupTime: time.Now().Format(time.RFC3339),
	}

	machineIDData, err := json.MarshalIndent(machineIDBackup, "", "  ")
	if err != nil {
		os.RemoveAll(backupPath)
		return fmt.Errorf("failed to marshal machine id: %w", err)
	}

	machineIDPath := filepath.Join(backupPath, MachineIDFileName)
	if err := os.WriteFile(machineIDPath, machineIDData, 0644); err != nil {
		os.RemoveAll(backupPath)
		return fmt.Errorf("failed to write machine id: %w", err)
	}

	return nil
}

// isIdCAuth 判斷是否為 IdC 認證類型
func isIdCAuth(authMethod string) bool {
	if authMethod == "" {
		return false
	}
	lower := strings.ToLower(authMethod)
	return lower == "idc" || lower == "identitycenter"
}

// copyFile 複製檔案
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}


// RestoreBackup 恢復指定的備份
func RestoreBackup(name string) error {
	if name == "" {
		return ErrInvalidBackupName
	}

	if !BackupExists(name) {
		return ErrBackupNotFound
	}

	backupPath, err := GetBackupPath(name)
	if err != nil {
		return err
	}

	// 恢復 kiro-auth-token.json
	tokenSrcPath := filepath.Join(backupPath, KiroAuthTokenFile)
	if _, err := os.Stat(tokenSrcPath); os.IsNotExist(err) {
		return fmt.Errorf("backup token file not found")
	}

	tokenDstPath, err := awssso.GetKiroAuthTokenPath()
	if err != nil {
		return fmt.Errorf("failed to get token destination path: %w", err)
	}

	// 確保目標目錄存在
	tokenDstDir := filepath.Dir(tokenDstPath)
	if err := os.MkdirAll(tokenDstDir, 0755); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	if err := copyFile(tokenSrcPath, tokenDstPath); err != nil {
		return fmt.Errorf("failed to restore token: %w", err)
	}

	// 讀取備份的 token 以檢查是否需要恢復 IdC 的 clientIdHash 文件
	token, err := ReadBackupToken(name)
	if err == nil && token != nil {
		// 如果是 IdC 認證且有 clientIdHash，恢復對應的 clientId/clientSecret 文件
		if isIdCAuth(token.AuthMethod) && token.ClientIdHash != "" {
			clientIdHashFile := token.ClientIdHash + ".json"
			clientIdHashSrcPath := filepath.Join(backupPath, clientIdHashFile)
			if _, err := os.Stat(clientIdHashSrcPath); err == nil {
				ssoCachePath, err := awssso.GetSSOCachePath()
				if err == nil {
					clientIdHashDstPath := filepath.Join(ssoCachePath, clientIdHashFile)
					if err := copyFile(clientIdHashSrcPath, clientIdHashDstPath); err != nil {
						// 恢復 clientIdHash 文件失敗不應該阻止整個恢復流程，只記錄警告
						fmt.Printf("Warning: failed to restore clientIdHash file: %v\n", err)
					}
				}
			}
		}
	}

	// 恢復 Machine ID（寫入 custom-machine-id 和 custom-machine-id-raw）
	machineIDBackup, err := ReadBackupMachineID(name)
	if err == nil && machineIDBackup != nil && machineIDBackup.MachineID != "" {
		rawMachineID := machineIDBackup.MachineID

		// 寫入原始 UUID（給 UI 顯示）
		if err := softreset.WriteCustomMachineIDRaw(rawMachineID); err != nil {
			return fmt.Errorf("failed to restore custom machine id raw: %w", err)
		}

		// 寫入 SHA256 雜湊值（給 Kiro 使用）
		hashedMachineID := machineid.HashMachineID(rawMachineID)
		if err := softreset.WriteCustomMachineID(hashedMachineID); err != nil {
			return fmt.Errorf("failed to restore custom machine id: %w", err)
		}
	}

	return nil
}

// DeleteBackup 刪除指定的備份
func DeleteBackup(name string) error {
	if name == "" {
		return ErrInvalidBackupName
	}

	if !BackupExists(name) {
		return ErrBackupNotFound
	}

	backupPath, err := GetBackupPath(name)
	if err != nil {
		return err
	}

	return os.RemoveAll(backupPath)
}

// GetBackupInfo 取得指定備份的詳細資訊
func GetBackupInfo(name string) (*BackupInfo, error) {
	if name == "" {
		return nil, ErrInvalidBackupName
	}

	if !BackupExists(name) {
		return nil, ErrBackupNotFound
	}

	backupPath, err := GetBackupPath(name)
	if err != nil {
		return nil, err
	}

	info := &BackupInfo{
		Name: name,
		Path: backupPath,
	}

	// 檢查 token 檔案
	tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
	if _, err := os.Stat(tokenPath); err == nil {
		info.HasToken = true
	}

	// 檢查 machine-id 檔案
	machineIDPath := filepath.Join(backupPath, MachineIDFileName)
	if data, err := os.ReadFile(machineIDPath); err == nil {
		info.HasMachineID = true
		var mid MachineIDBackup
		if json.Unmarshal(data, &mid) == nil && mid.BackupTime != "" {
			if t, err := time.Parse(time.RFC3339, mid.BackupTime); err == nil {
				info.BackupTime = t
			}
		}
	}

	return info, nil
}

// ReadBackupMachineID 讀取備份中的 Machine ID
func ReadBackupMachineID(name string) (*MachineIDBackup, error) {
	if name == "" {
		return nil, ErrInvalidBackupName
	}

	if !BackupExists(name) {
		return nil, ErrBackupNotFound
	}

	backupPath, err := GetBackupPath(name)
	if err != nil {
		return nil, err
	}

	machineIDPath := filepath.Join(backupPath, MachineIDFileName)
	data, err := os.ReadFile(machineIDPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read machine id file: %w", err)
	}

	var mid MachineIDBackup
	if err := json.Unmarshal(data, &mid); err != nil {
		return nil, fmt.Errorf("failed to parse machine id file: %w", err)
	}

	return &mid, nil
}

// OriginalBackupName 原始備份的固定名稱
const OriginalBackupName = "original"

// CreateMachineIDOnlyBackup 僅備份 Machine ID（不備份 token）
// 用於軟體啟動時確保原始 Machine ID 被保存
func CreateMachineIDOnlyBackup(name string) error {
	if name == "" {
		return ErrInvalidBackupName
	}

	if BackupExists(name) {
		return ErrBackupExists
	}

	// 確保備份根目錄存在
	_, err := ensureBackupRoot()
	if err != nil {
		return fmt.Errorf("failed to create backup root: %w", err)
	}

	// 創建備份資料夾
	backupPath, err := GetBackupPath(name)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 僅備份 Machine ID
	rawMachineID, err := machineid.GetRawMachineId()
	if err != nil {
		os.RemoveAll(backupPath)
		return fmt.Errorf("failed to get machine id: %w", err)
	}

	machineIDBackup := MachineIDBackup{
		MachineID:  rawMachineID,
		BackupTime: time.Now().Format(time.RFC3339),
	}

	machineIDData, err := json.MarshalIndent(machineIDBackup, "", "  ")
	if err != nil {
		os.RemoveAll(backupPath)
		return fmt.Errorf("failed to marshal machine id: %w", err)
	}

	machineIDPath := filepath.Join(backupPath, MachineIDFileName)
	if err := os.WriteFile(machineIDPath, machineIDData, 0644); err != nil {
		os.RemoveAll(backupPath)
		return fmt.Errorf("failed to write machine id: %w", err)
	}

	return nil
}

// EnsureOriginalBackup 確保原始 Machine ID 已備份
// 如果名為 "original" 的備份不存在，則自動創建
// 回傳 (true, nil) 表示新建了備份
// 回傳 (false, nil) 表示備份已存在，無需操作
func EnsureOriginalBackup() (bool, error) {
	if BackupExists(OriginalBackupName) {
		return false, nil
	}

	// 使用僅備份 Machine ID 的方式，不強制要求 token
	if err := CreateMachineIDOnlyBackup(OriginalBackupName); err != nil {
		return false, fmt.Errorf("failed to create original backup: %w", err)
	}

	return true, nil
}

// ReadBackupToken 讀取備份中的 kiro-auth-token.json
func ReadBackupToken(name string) (*awssso.KiroAuthToken, error) {
	if name == "" {
		return nil, ErrInvalidBackupName
	}

	if !BackupExists(name) {
		return nil, ErrBackupNotFound
	}

	backupPath, err := GetBackupPath(name)
	if err != nil {
		return nil, err
	}

	tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var token awssso.KiroAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &token, nil
}

// ReadBackupIdCCredentials 從備份目錄讀取 IdC 的 clientId 和 clientSecret
// 根據 token 中的 clientIdHash 查找對應的 JSON 文件
func ReadBackupIdCCredentials(name string, clientIdHash string) (clientID, clientSecret string, err error) {
	if name == "" {
		return "", "", ErrInvalidBackupName
	}

	if clientIdHash == "" {
		return "", "", fmt.Errorf("clientIdHash is empty")
	}

	if !BackupExists(name) {
		return "", "", ErrBackupNotFound
	}

	backupPath, err := GetBackupPath(name)
	if err != nil {
		return "", "", err
	}

	// 讀取 clientIdHash 對應的 JSON 文件
	clientIdHashFile := clientIdHash + ".json"
	clientIdHashPath := filepath.Join(backupPath, clientIdHashFile)

	data, err := os.ReadFile(clientIdHashPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read clientIdHash file: %w", err)
	}

	// 解析 JSON 文件
	var cacheFile struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	if err := json.Unmarshal(data, &cacheFile); err != nil {
		return "", "", fmt.Errorf("failed to parse clientIdHash file: %w", err)
	}

	if cacheFile.ClientID == "" || cacheFile.ClientSecret == "" {
		return "", "", fmt.Errorf("clientId or clientSecret not found in file")
	}

	return cacheFile.ClientID, cacheFile.ClientSecret, nil
}

// ReadUsageCache 讀取備份的餘額緩存
func ReadUsageCache(name string) (*UsageCache, error) {
	if name == "" {
		return nil, ErrInvalidBackupName
	}

	if !BackupExists(name) {
		return nil, ErrBackupNotFound
	}

	backupPath, err := GetBackupPath(name)
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(backupPath, UsageCacheFileName)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read usage cache file: %w", err)
	}

	var cache UsageCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse usage cache file: %w", err)
	}

	return &cache, nil
}

// WriteUsageCache 寫入備份的餘額緩存
func WriteUsageCache(name string, cache *UsageCache) error {
	if name == "" {
		return ErrInvalidBackupName
	}

	if cache == nil {
		return fmt.Errorf("cache cannot be nil")
	}

	if !BackupExists(name) {
		return ErrBackupNotFound
	}

	backupPath, err := GetBackupPath(name)
	if err != nil {
		return err
	}

	// 設定緩存時間
	cache.CachedAt = time.Now()

	cacheData, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal usage cache: %w", err)
	}

	cachePath := filepath.Join(backupPath, UsageCacheFileName)
	if err := os.WriteFile(cachePath, cacheData, 0644); err != nil {
		return fmt.Errorf("failed to write usage cache: %w", err)
	}

	return nil
}


// orderedKiroAuthToken 用於確保 JSON 輸出時 key 的順序
// 順序: accessToken, refreshToken, profileArn, expiresAt, authMethod, provider
type orderedKiroAuthToken struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ProfileArn   string `json:"profileArn"`
	ExpiresAt    string `json:"expiresAt"`
	AuthMethod   string `json:"authMethod"`
	Provider     string `json:"provider"`
}

// WriteBackupToken 將刷新後的 Token 寫入備份檔案
// 保留原有欄位，僅更新 accessToken、expiresAt
// 確保 JSON key 順序: accessToken, refreshToken, profileArn, expiresAt, authMethod, provider
// 需求: 3.1, 3.2, 3.3
func WriteBackupToken(name string, accessToken string, expiresAt string) error {
	if name == "" {
		return ErrInvalidBackupName
	}

	if !BackupExists(name) {
		return ErrBackupNotFound
	}

	backupPath, err := GetBackupPath(name)
	if err != nil {
		return err
	}

	tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)

	// 讀取現有 token 檔案以保留原始欄位
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return fmt.Errorf("failed to read existing token file: %w", err)
	}

	// 先解析到 map 以讀取原始值
	var tokenMap map[string]interface{}
	if err := json.Unmarshal(data, &tokenMap); err != nil {
		return fmt.Errorf("failed to parse existing token file: %w", err)
	}

	// 使用有序結構體來確保 key 順序
	orderedToken := orderedKiroAuthToken{
		AccessToken:  accessToken,
		RefreshToken: getStringFromMap(tokenMap, "refreshToken"),
		ProfileArn:   getStringFromMap(tokenMap, "profileArn"),
		ExpiresAt:    expiresAt,
		AuthMethod:   getStringFromMap(tokenMap, "authMethod"),
		Provider:     getStringFromMap(tokenMap, "provider"),
	}

	// 將更新後的 token 寫回檔案
	updatedData, err := json.MarshalIndent(orderedToken, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated token: %w", err)
	}

	if err := os.WriteFile(tokenPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// getStringFromMap 從 map 中安全地取得字串值
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// UpdateBackupMachineID 更新備份中的 Machine ID
// 用於為指定備份生成新的機器碼
func UpdateBackupMachineID(name string, newMachineID string) error {
	if name == "" {
		return ErrInvalidBackupName
	}

	if newMachineID == "" {
		return fmt.Errorf("new machine id cannot be empty")
	}

	if !BackupExists(name) {
		return ErrBackupNotFound
	}

	backupPath, err := GetBackupPath(name)
	if err != nil {
		return err
	}

	machineIDBackup := MachineIDBackup{
		MachineID:  newMachineID,
		BackupTime: time.Now().Format(time.RFC3339),
	}

	machineIDData, err := json.MarshalIndent(machineIDBackup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal machine id: %w", err)
	}

	machineIDPath := filepath.Join(backupPath, MachineIDFileName)
	if err := os.WriteFile(machineIDPath, machineIDData, 0644); err != nil {
		return fmt.Errorf("failed to write machine id: %w", err)
	}

	return nil
}
