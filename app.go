package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"kiro-manager/awssso"
	"kiro-manager/backup"
	"kiro-manager/deeplink"
	"kiro-manager/kiropath"
	"kiro-manager/kiroprocess"
	"kiro-manager/kiroversion"
	"kiro-manager/machineid"
	"kiro-manager/oauthlogin"
	"kiro-manager/settings"
	"kiro-manager/softreset"
	"kiro-manager/tokenrefresh"
	"kiro-manager/usage"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/wailsapp/wails/v2/pkg/options"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 不再於啟動時自動備份，避免觸發防毒軟體誤報
	// 改為在用戶首次執行需要備份的操作時才觸發

	// 註冊 URL Scheme (Windows only)
	if err := deeplink.EnsureURLSchemeRegistered(); err != nil {
		// 記錄錯誤但不阻止啟動
		println("Warning: Failed to register URL scheme:", err.Error())
	}

	// 檢查啟動時的命令行參數是否包含 deep link URL
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "kiro://") {
			result, err := deeplink.HandleDeepLinkCallback(arg)
			if err == nil {
				deeplink.SendCallback(result)
			}
			break
		}
	}
}

// BackupItem 備份項目（前端用）
type BackupItem struct {
	Name              string  `json:"name"`
	BackupTime        string  `json:"backupTime"`
	HasToken          bool    `json:"hasToken"`
	HasMachineID      bool    `json:"hasMachineId"`
	MachineID         string  `json:"machineId"`
	Provider          string  `json:"provider"`
	IsCurrent         bool    `json:"isCurrent"`
	IsOriginalMachine bool    `json:"isOriginalMachine"` // Machine ID 與原始機器相同
	IsTokenExpired    bool    `json:"isTokenExpired"`    // Token 是否已過期
	// Usage 相關欄位 (Requirements: 1.1, 1.2)
	SubscriptionTitle string  `json:"subscriptionTitle"` // 訂閱類型名稱
	UsageLimit        float64 `json:"usageLimit"`        // 總額度
	CurrentUsage      float64 `json:"currentUsage"`      // 已使用
	Balance           float64 `json:"balance"`           // 餘額
	IsLowBalance      bool    `json:"isLowBalance"`      // 餘額低於 20%
	CachedAt          string  `json:"cachedAt"`          // 緩存時間（用於前端判斷冷卻期）
	// 文件夾相關欄位
	FolderId          string  `json:"folderId"`          // 所屬文件夾 ID，空字串表示未分類
}

// Result 通用回傳結果
type Result struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// PathDetectionResult 路徑偵測結果（前端用）
// 用於 GetKiroInstallPathWithStatus() 返回偵測狀態和詳細資訊
type PathDetectionResult struct {
	Path            string            `json:"path"`
	Success         bool              `json:"success"`
	TriedStrategies []string          `json:"triedStrategies,omitempty"`
	FailureReasons  map[string]string `json:"failureReasons,omitempty"`
}

// GetBackupList 取得備份列表
func (a *App) GetBackupList() ([]BackupItem, error) {
	backups, err := backup.ListBackups()
	if err != nil {
		return nil, err
	}

	// 取得當前 Machine ID（優先使用重置的自訂 ID）
	currentMachineID := a.GetCurrentMachineID()

	// 讀取原始 Machine ID
	var originalMachineID string
	if originalBackup, err := backup.ReadBackupMachineID(backup.OriginalBackupName); err == nil {
		originalMachineID = originalBackup.MachineID
	}

	var items []BackupItem
	for _, b := range backups {
		// 過濾掉 "original" 備份，不顯示在列表中
		if b.Name == backup.OriginalBackupName {
			continue
		}

		item := BackupItem{
			Name:         b.Name,
			HasToken:     b.HasToken,
			HasMachineID: b.HasMachineID,
		}

		if !b.BackupTime.IsZero() {
			item.BackupTime = b.BackupTime.Format("2006-01-02 15:04:05")
		}

		if b.HasMachineID {
			mid, err := backup.ReadBackupMachineID(b.Name)
			if err == nil {
				item.MachineID = mid.MachineID
				item.IsCurrent = mid.MachineID == currentMachineID
				item.IsOriginalMachine = mid.MachineID == originalMachineID
			}
		}

		// 讀取 token 中的 provider 和過期狀態
		if b.HasToken {
			token, err := backup.ReadBackupToken(b.Name)
			if err == nil && token != nil {
				if token.Provider != "" {
					item.Provider = token.Provider
				}
				// 檢查 token 是否已過期
				item.IsTokenExpired = awssso.IsTokenExpired(token)
			}
		}

		// 從緩存讀取用量資訊（不再自動呼叫 API）
		if usageCache, err := backup.ReadUsageCache(b.Name); err == nil && usageCache != nil {
			item.SubscriptionTitle = usageCache.SubscriptionTitle
			item.UsageLimit = usageCache.UsageLimit
			item.CurrentUsage = usageCache.CurrentUsage
			item.Balance = usageCache.Balance
			// 使用設定的閾值重新計算 IsLowBalance
			threshold := settings.GetLowBalanceThreshold()
			if usageCache.UsageLimit > 0 {
				item.IsLowBalance = (usageCache.Balance / usageCache.UsageLimit) < threshold
			}
			// 傳遞緩存時間供前端判斷冷卻期
			if !usageCache.CachedAt.IsZero() {
				item.CachedAt = usageCache.CachedAt.Format(time.RFC3339)
			}
		}
		// 沒有緩存的備份，用量欄位保持零值（前端顯示 "-"）

		// 取得快照所屬文件夾
		folderId, _ := backup.GetSnapshotFolderId(b.Name)
		item.FolderId = folderId

		items = append(items, item)
	}

	return items, nil
}

// UsageCacheResult 餘額刷新結果
type UsageCacheResult struct {
	Success           bool    `json:"success"`
	Message           string  `json:"message"`
	SubscriptionTitle string  `json:"subscriptionTitle"`
	UsageLimit        float64 `json:"usageLimit"`
	CurrentUsage      float64 `json:"currentUsage"`
	Balance           float64 `json:"balance"`
	IsLowBalance      bool    `json:"isLowBalance"`
	IsTokenExpired    bool    `json:"isTokenExpired"` // Token 是否已過期（刷新成功後為 false）
	CachedAt          string  `json:"cachedAt"`       // 緩存時間（用於前端判斷冷卻期）
}

// RefreshBackupUsage 刷新指定備份的餘額資訊
// 需求: 1.1, 1.2, 1.3, 1.4, 1.5
func (a *App) RefreshBackupUsage(name string) UsageCacheResult {
	if name == "" {
		return UsageCacheResult{Success: false, Message: "備份名稱不能為空"}
	}

	if !backup.BackupExists(name) {
		return UsageCacheResult{Success: false, Message: "備份不存在"}
	}

	// 先讀取備份的 Machine ID（用於 Token 刷新和 API 呼叫）
	mid, err := backup.ReadBackupMachineID(name)
	if err != nil {
		return UsageCacheResult{Success: false, Message: "無法讀取備份的 Machine ID"}
	}
	hashedMachineID := machineid.HashMachineID(mid.MachineID)

	// 讀取備份的 token
	token, err := backup.ReadBackupToken(name)
	if err != nil {
		return UsageCacheResult{Success: false, Message: "無法讀取備份的 token"}
	}

	// 檢查 token 是否已過期（需求 1.1）
	if awssso.IsTokenExpired(token) {
		// 嘗試刷新 Token（需求 1.1, 1.2, 1.3）
		// 使用對應環境快照的 Machine ID 的 SHA256 雜湊值
		var newTokenInfo *tokenrefresh.TokenInfo
		var err error

		// 檢查是否為 IdC 認證，如果是則從備份目錄讀取 clientId/clientSecret
		authType := tokenrefresh.DetectAuthType(token)
		if authType == "idc" && token.ClientIdHash != "" {
			// 從備份目錄讀取 IdC credentials
			clientID, clientSecret, credErr := backup.ReadBackupIdCCredentials(name, token.ClientIdHash)
			if credErr != nil {
				return UsageCacheResult{Success: false, Message: "無法讀取 IdC 認證資訊: " + credErr.Error()}
			}
			newTokenInfo, err = tokenrefresh.RefreshAccessTokenFromBackup(token, hashedMachineID, clientID, clientSecret)
		} else {
			// Social 認證或其他情況，使用原有邏輯
			newTokenInfo, err = tokenrefresh.RefreshAccessToken(token, hashedMachineID)
		}

		if err != nil {
			// 刷新失敗，返回錯誤（需求 1.5）
			return UsageCacheResult{Success: false, Message: err.Error()}
		}

		// 更新 token 結構的新值（需求 1.2, 1.3）
		token.AccessToken = newTokenInfo.AccessToken
		token.ExpiresAt = newTokenInfo.ExpiresAt.UTC().Format("2006-01-02T15:04:05.000Z")

		// 呼叫 WriteBackupToken() 持久化刷新後的 token（需求 3.1, 3.2）
		if err := backup.WriteBackupToken(name, token.AccessToken, token.ExpiresAt); err != nil {
			return UsageCacheResult{Success: false, Message: "Token 刷新成功但寫入失敗: " + err.Error()}
		}
	}

	// 呼叫 API 取得用量資訊（需求 1.4）
	// hashedMachineID 已在上方計算
	usageInfo, err := usage.GetUsageLimitsWithMachineID(token, hashedMachineID)
	if err != nil {
		return UsageCacheResult{Success: false, Message: fmt.Sprintf("API 呼叫失敗: %v", err)}
	}

	if usageInfo == nil || usageInfo.SubscriptionTitle == "" {
		return UsageCacheResult{Success: false, Message: "無法取得用量資訊"}
	}

	// 使用設定的閾值重新計算 IsLowBalance
	threshold := settings.GetLowBalanceThreshold()
	isLowBalance := false
	if usageInfo.UsageLimit > 0 {
		isLowBalance = (usageInfo.Balance / usageInfo.UsageLimit) < threshold
	}

	// 寫入緩存
	cache := &backup.UsageCache{
		SubscriptionTitle: usageInfo.SubscriptionTitle,
		UsageLimit:        usageInfo.UsageLimit,
		CurrentUsage:      usageInfo.CurrentUsage,
		Balance:           usageInfo.Balance,
		IsLowBalance:      isLowBalance,
	}
	if err := backup.WriteUsageCache(name, cache); err != nil {
		return UsageCacheResult{Success: false, Message: fmt.Sprintf("緩存寫入失敗: %v", err)}
	}

	// 緩存時間為當前時間（WriteUsageCache 會設定 CachedAt）
	cachedAt := time.Now().Format(time.RFC3339)

	return UsageCacheResult{
		Success:           true,
		Message:           "刷新成功",
		SubscriptionTitle: usageInfo.SubscriptionTitle,
		UsageLimit:        usageInfo.UsageLimit,
		CurrentUsage:      usageInfo.CurrentUsage,
		Balance:           usageInfo.Balance,
		IsLowBalance:      isLowBalance,
		IsTokenExpired:    false, // 刷新成功代表 token 有效
		CachedAt:          cachedAt,
	}
}

// CreateBackup 建立新備份
func (a *App) CreateBackup(name string) Result {
	if name == "" {
		return Result{Success: false, Message: "備份名稱不能為空"}
	}

	if err := backup.CreateBackup(name); err != nil {
		return Result{Success: false, Message: err.Error()}
	}

	return Result{Success: true, Message: "備份成功"}
}

// SwitchToBackup 切換至指定備份帳號（恢復 token）
func (a *App) SwitchToBackup(name string) Result {
	if name == "" {
		return Result{Success: false, Message: "請選擇備份"}
	}

	// 檢測並強制關閉 Kiro
	if kiroprocess.IsKiroRunning() {
		killed, err := kiroprocess.KillKiroProcesses()
		if err != nil {
			return Result{Success: false, Message: fmt.Sprintf("關閉 Kiro 失敗: %v", err)}
		}
		if killed == 0 && kiroprocess.IsKiroRunning() {
			return Result{Success: false, Message: "無法關閉 Kiro，請手動關閉後重試"}
		}
	}

	if err := backup.RestoreBackup(name); err != nil {
		return Result{Success: false, Message: fmt.Sprintf("恢復 Token 失敗: %v", err)}
	}

	return Result{Success: true, Message: "切換成功"}
}



// DeleteBackup 刪除備份
func (a *App) DeleteBackup(name string) Result {
	if name == backup.OriginalBackupName {
		return Result{Success: false, Message: "不能刪除原始備份"}
	}

	if err := backup.DeleteBackup(name); err != nil {
		return Result{Success: false, Message: err.Error()}
	}

	return Result{Success: true, Message: "刪除成功"}
}

// RegenerateMachineID 為指定備份生成新的機器碼
func (a *App) RegenerateMachineID(name string) Result {
	if name == "" {
		return Result{Success: false, Message: "備份名稱不能為空"}
	}

	if name == backup.OriginalBackupName {
		return Result{Success: false, Message: "不能修改原始備份的機器碼"}
	}

	if !backup.BackupExists(name) {
		return Result{Success: false, Message: "備份不存在"}
	}

	// 生成新的 Machine ID（UUID v4）
	newMachineID := softreset.GenerateNewMachineID()

	// 檢查該備份是否為當前使用中的環境（在更新前檢查）
	currentEnvName := a.GetCurrentEnvironmentName()
	isCurrent := currentEnvName == name

	// 更新備份中的 Machine ID
	if err := backup.UpdateBackupMachineID(name, newMachineID); err != nil {
		return Result{Success: false, Message: fmt.Sprintf("更新機器碼失敗: %v", err)}
	}

	// 如果當前環境使用的是這個備份，則同步更新 custom-machine-id
	if isCurrent {
		// 寫入原始 UUID（給 UI 顯示）
		if err := softreset.WriteCustomMachineIDRaw(newMachineID); err != nil {
			return Result{Success: false, Message: fmt.Sprintf("更新自訂機器碼失敗: %v", err)}
		}

		// 寫入 SHA256 雜湊值（給 Kiro 使用）
		hashedMachineID := machineid.HashMachineID(newMachineID)
		if err := softreset.WriteCustomMachineID(hashedMachineID); err != nil {
			return Result{Success: false, Message: fmt.Sprintf("更新自訂機器碼雜湊失敗: %v", err)}
		}

		return Result{
			Success: true,
			Message: fmt.Sprintf("已生成新機器碼並同步更新當前環境: %s", newMachineID[:8]+"..."),
		}
	}

	return Result{
		Success: true,
		Message: fmt.Sprintf("已生成新機器碼: %s", newMachineID[:8]+"..."),
	}
}

// GetCurrentMachineID 取得當前 Machine ID
// 優先順序：
// 1. custom-machine-id-raw（原始 UUID，用於 UI 顯示）
// 2. 系統原始 Machine ID
// 注意：不使用 custom-machine-id（SHA256 雜湊值），因為那是給 Kiro 內部使用的
func (a *App) GetCurrentMachineID() string {
	// 優先讀取 custom-machine-id-raw（原始 UUID）
	rawID, err := softreset.ReadCustomMachineIDRaw()
	if err == nil && rawID != "" {
		return rawID
	}

	// 否則返回系統原始 Machine ID
	id, _ := machineid.GetRawMachineId()
	return id
}

// GetCurrentEnvironmentName 取得當前運行環境的名稱
// 根據當前 Machine ID 查找對應的環境快照名稱
// 如果找不到對應的環境快照，返回空字串（前端顯示「原始機器」）
func (a *App) GetCurrentEnvironmentName() string {
	currentMachineID := a.GetCurrentMachineID()
	if currentMachineID == "" {
		return ""
	}

	// 遍歷所有備份，找到 Machine ID 匹配的備份
	backups, err := backup.ListBackups()
	if err != nil {
		return ""
	}

	for _, b := range backups {
		// 跳過 "original" 備份
		if b.Name == backup.OriginalBackupName {
			continue
		}

		if b.HasMachineID {
			mid, err := backup.ReadBackupMachineID(b.Name)
			if err == nil && mid.MachineID == currentMachineID {
				return b.Name
			}
		}
	}

	return ""
}

// EnsureOriginalBackup 確保原始備份存在
func (a *App) EnsureOriginalBackup() Result {
	created, err := backup.EnsureOriginalBackup()
	if err != nil {
		return Result{Success: false, Message: err.Error()}
	}

	if created {
		return Result{Success: true, Message: "已建立原始備份"}
	}
	return Result{Success: true, Message: "原始備份已存在"}
}



// onSecondInstanceLaunch 處理第二個實例啟動 (deep link 回調)
func (a *App) onSecondInstanceLaunch(data options.SecondInstanceData) {
	// 檢查是否有 deep link URL
	for _, arg := range data.Args {
		if strings.HasPrefix(arg, "kiro://") {
			// 解析並處理 deep link
			result, err := deeplink.HandleDeepLinkCallback(arg)
			if err == nil {
				deeplink.SendCallback(result)
			}
			break
		}
	}

	// 聚焦視窗
	wailsRuntime.WindowUnminimise(a.ctx)
	wailsRuntime.Show(a.ctx)
}

// IsDeepLinkSupported 檢查當前平台是否支援 Deep Link
func (a *App) IsDeepLinkSupported() bool {
	return deeplink.IsDeepLinkSupported()
}

// GetAppInfo 取得應用資訊
func (a *App) GetAppInfo() map[string]string {
	return map[string]string{
		"version":   "0.5.0",
		"platform":  runtime.GOOS,
		"buildTime": time.Now().Format("2025-12-07"),
	}
}

// GetCurrentProvider 取得當前 Kiro 登入的帳號來源（Provider）
// 讀取 ~/.aws/sso/cache/kiro-auth-token.json 中的 provider 欄位
func (a *App) GetCurrentProvider() string {
	token, err := awssso.ReadKiroAuthToken()
	if err != nil {
		return ""
	}
	return token.Provider
}

// CurrentUsageInfo 當前帳號用量資訊（前端用）
type CurrentUsageInfo struct {
	SubscriptionTitle string  `json:"subscriptionTitle"` // 訂閱類型名稱
	UsageLimit        float64 `json:"usageLimit"`        // 總額度
	CurrentUsage      float64 `json:"currentUsage"`      // 已使用
	Balance           float64 `json:"balance"`           // 餘額
	IsLowBalance      bool    `json:"isLowBalance"`      // 餘額低於 20%
}

// GetCurrentUsageInfo 取得當前帳號的用量資訊
// 讀取當前 Kiro 登入的 token，優先從緩存讀取，緩存不存在時呼叫 API
func (a *App) GetCurrentUsageInfo() *CurrentUsageInfo {
	// 取得當前 Machine ID（優先使用重置的自訂 ID）
	currentMachineID := a.GetCurrentMachineID()
	threshold := settings.GetLowBalanceThreshold()

	// 查找當前 Machine ID 對應的備份
	backupName := a.findBackupByMachineID(currentMachineID)
	if backupName != "" {
		// 優先從緩存讀取
		if usageCache, err := backup.ReadUsageCache(backupName); err == nil && usageCache != nil {
			// 使用設定的閾值重新計算 IsLowBalance
			isLowBalance := false
			if usageCache.UsageLimit > 0 {
				isLowBalance = (usageCache.Balance / usageCache.UsageLimit) < threshold
			}
			return &CurrentUsageInfo{
				SubscriptionTitle: usageCache.SubscriptionTitle,
				UsageLimit:        usageCache.UsageLimit,
				CurrentUsage:      usageCache.CurrentUsage,
				Balance:           usageCache.Balance,
				IsLowBalance:      isLowBalance,
			}
		}
	}

	// 緩存不存在，呼叫 API
	token, err := awssso.ReadKiroAuthToken()
	if err != nil {
		return nil
	}

	hashedMachineID := machineid.HashMachineID(currentMachineID)
	usageInfo := usage.GetUsageLimitsSafeWithMachineID(token, hashedMachineID)
	if usageInfo == nil || usageInfo.SubscriptionTitle == "" {
		return nil
	}

	// 使用設定的閾值重新計算 IsLowBalance
	isLowBalance := false
	if usageInfo.UsageLimit > 0 {
		isLowBalance = (usageInfo.Balance / usageInfo.UsageLimit) < threshold
	}

	// 如果找到對應的備份，將結果寫入緩存
	if backupName != "" {
		cache := &backup.UsageCache{
			SubscriptionTitle: usageInfo.SubscriptionTitle,
			UsageLimit:        usageInfo.UsageLimit,
			CurrentUsage:      usageInfo.CurrentUsage,
			Balance:           usageInfo.Balance,
			IsLowBalance:      isLowBalance,
		}
		backup.WriteUsageCache(backupName, cache)
	}

	return &CurrentUsageInfo{
		SubscriptionTitle: usageInfo.SubscriptionTitle,
		UsageLimit:        usageInfo.UsageLimit,
		CurrentUsage:      usageInfo.CurrentUsage,
		Balance:           usageInfo.Balance,
		IsLowBalance:      isLowBalance,
	}
}

// findBackupByMachineID 根據 Machine ID 查找對應的備份名稱
func (a *App) findBackupByMachineID(machineID string) string {
	backups, err := backup.ListBackups()
	if err != nil {
		return ""
	}

	for _, b := range backups {
		if b.Name == backup.OriginalBackupName {
			continue
		}
		if b.HasMachineID {
			mid, err := backup.ReadBackupMachineID(b.Name)
			if err == nil && mid.MachineID == machineID {
				return b.Name
			}
		}
	}
	return ""
}

// IsKiroRunning 檢查 Kiro 是否正在運行
func (a *App) IsKiroRunning() bool {
	return kiroprocess.IsKiroRunning()
}

// GetKiroProcesses 取得所有 Kiro 進程資訊
func (a *App) GetKiroProcesses() []kiroprocess.ProcessInfo {
	processes, err := kiroprocess.GetKiroProcesses()
	if err != nil {
		return []kiroprocess.ProcessInfo{}
	}
	return processes
}


// ============================================================================
// 一鍵新機功能（跨平台）
// ============================================================================

// SoftResetStatus 重置狀態（前端用）
type SoftResetStatus struct {
	IsPatched       bool   `json:"isPatched"`
	HasCustomID     bool   `json:"hasCustomId"`
	CustomMachineID string `json:"customMachineId"`
	ExtensionPath   string `json:"extensionPath"`
	IsSupported     bool   `json:"isSupported"`
}

// SoftResetToNewMachine 一鍵新機（跨平台，不需要管理員權限）
func (a *App) SoftResetToNewMachine() Result {
	// 檢測並強制關閉 Kiro
	if kiroprocess.IsKiroRunning() {
		killed, err := kiroprocess.KillKiroProcesses()
		if err != nil {
			return Result{Success: false, Message: fmt.Sprintf("關閉 Kiro 失敗: %v", err)}
		}
		if killed == 0 && kiroprocess.IsKiroRunning() {
			return Result{Success: false, Message: "無法關閉 Kiro，請手動關閉後重試"}
		}
	}

	result, err := softreset.SoftResetEnvironment()
	if err != nil {
		return Result{Success: false, Message: err.Error()}
	}

	return Result{
		Success: true,
		Message: fmt.Sprintf("重置成功！新 Machine ID: %s", result.NewMachineID[:8]+"..."),
	}
}

// GetSoftResetStatus 取得重置狀態
func (a *App) GetSoftResetStatus() SoftResetStatus {
	status := SoftResetStatus{
		IsSupported: true,
	}

	// 取得重置狀態
	softStatus, err := softreset.GetSoftResetStatus()
	if err != nil {
		status.IsSupported = false
		return status
	}

	status.IsPatched = softStatus.IsPatched
	status.HasCustomID = softStatus.HasCustomID
	status.CustomMachineID = softStatus.CustomMachineID
	status.ExtensionPath = softStatus.ExtensionPath

	return status
}

// RestoreSoftReset 還原重置（恢復系統原始 Machine ID）
func (a *App) RestoreSoftReset() Result {
	// 檢測並強制關閉 Kiro
	if kiroprocess.IsKiroRunning() {
		killed, err := kiroprocess.KillKiroProcesses()
		if err != nil {
			return Result{Success: false, Message: fmt.Sprintf("關閉 Kiro 失敗: %v", err)}
		}
		if killed == 0 && kiroprocess.IsKiroRunning() {
			return Result{Success: false, Message: "無法關閉 Kiro，請手動關閉後重試"}
		}
	}

	// 執行還原（刪除自訂 Machine ID、還原 extension.js）
	if err := softreset.RestoreOriginalMachineID(); err != nil {
		return Result{Success: false, Message: err.Error()}
	}

	// 取得系統原始 Machine ID（原始 UUID，用於比對備份）
	originalMachineID, err := machineid.GetRawMachineId()
	if err != nil {
		return Result{Success: true, Message: "已還原為系統原始 Machine ID（無法讀取機器碼）"}
	}

	// 比對備份，找到使用相同機器碼的備份並恢復
	backups, err := backup.ListBackups()
	if err == nil {
		for _, b := range backups {
			backupMID, err := backup.ReadBackupMachineID(b.Name)
			if err == nil && backupMID.MachineID == originalMachineID {
				// 找到匹配的備份，恢復 SSO cache（token）
				if err := backup.RestoreBackup(b.Name); err == nil {
					return Result{
						Success: true,
						Message: fmt.Sprintf("已還原為系統原始 Machine ID，並恢復帳號「%s」", b.Name),
					}
				}
				break
			}
		}
	}

	return Result{Success: true, Message: "已還原為系統原始 Machine ID"}
}

// RepatchExtension 重新 Patch extension.js（Kiro 更新後使用）
func (a *App) RepatchExtension() Result {
	// 檢測並強制關閉 Kiro
	if kiroprocess.IsKiroRunning() {
		killed, err := kiroprocess.KillKiroProcesses()
		if err != nil {
			return Result{Success: false, Message: fmt.Sprintf("關閉 Kiro 失敗: %v", err)}
		}
		if killed == 0 && kiroprocess.IsKiroRunning() {
			return Result{Success: false, Message: "無法關閉 Kiro，請手動關閉後重試"}
		}
	}

	if err := softreset.PatchExtensionJS(); err != nil {
		return Result{Success: false, Message: err.Error()}
	}

	return Result{Success: true, Message: "Patch 成功"}
}

// UnpatchExtension 移除 Patch（還原 extension.js）
func (a *App) UnpatchExtension() Result {
	// 檢測並強制關閉 Kiro
	if kiroprocess.IsKiroRunning() {
		killed, err := kiroprocess.KillKiroProcesses()
		if err != nil {
			return Result{Success: false, Message: fmt.Sprintf("關閉 Kiro 失敗: %v", err)}
		}
		if killed == 0 && kiroprocess.IsKiroRunning() {
			return Result{Success: false, Message: "無法關閉 Kiro，請手動關閉後重試"}
		}
	}

	if err := softreset.UnpatchExtensionJS(); err != nil {
		return Result{Success: false, Message: err.Error()}
	}

	return Result{Success: true, Message: "已移除 Patch"}
}

// ============================================================================
// 全域設定功能
// ============================================================================

// AppSettings 應用設定（前端用）
type AppSettings struct {
	LowBalanceThreshold   float64 `json:"lowBalanceThreshold"`   // 低餘額閾值（0.0 ~ 1.0）
	KiroVersion           string  `json:"kiroVersion"`           // Kiro IDE 版本號
	UseAutoDetect         bool    `json:"useAutoDetect"`         // 是否使用自動偵測版本號
	CustomKiroInstallPath string  `json:"customKiroInstallPath"` // 自定義 Kiro 安裝路徑
}

// WindowSize 視窗尺寸結構
type WindowSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// GetSettings 取得全域設定
func (a *App) GetSettings() AppSettings {
	s := settings.GetCurrentSettings()
	return AppSettings{
		LowBalanceThreshold:   s.LowBalanceThreshold,
		KiroVersion:           s.KiroVersion,
		UseAutoDetect:         s.UseAutoDetect,
		CustomKiroInstallPath: s.CustomKiroInstallPath,
	}
}

// SaveSettings 儲存全域設定
func (a *App) SaveSettings(appSettings AppSettings) Result {
	s := &settings.Settings{
		LowBalanceThreshold:   appSettings.LowBalanceThreshold,
		KiroVersion:           appSettings.KiroVersion,
		UseAutoDetect:         appSettings.UseAutoDetect,
		CustomKiroInstallPath: appSettings.CustomKiroInstallPath,
	}
	if err := settings.SaveSettings(s); err != nil {
		return Result{Success: false, Message: fmt.Sprintf("儲存設定失敗: %v", err)}
	}
	return Result{Success: true, Message: "設定已儲存"}
}

// GetWindowSize 取得已保存的視窗尺寸
func (a *App) GetWindowSize() WindowSize {
	s := settings.GetCurrentSettings()
	return WindowSize{
		Width:  s.WindowWidth,
		Height: s.WindowHeight,
	}
}

// SaveWindowSize 保存視窗尺寸
func (a *App) SaveWindowSize(width, height int) Result {
	s := settings.GetCurrentSettings()
	newSettings := &settings.Settings{
		LowBalanceThreshold:   s.LowBalanceThreshold,
		KiroVersion:           s.KiroVersion,
		UseAutoDetect:         s.UseAutoDetect,
		CustomKiroInstallPath: s.CustomKiroInstallPath,
		WindowWidth:           width,
		WindowHeight:          height,
	}
	if err := settings.SaveSettings(newSettings); err != nil {
		return Result{Success: false, Message: fmt.Sprintf("保存視窗尺寸失敗: %v", err)}
	}
	return Result{Success: true, Message: "視窗尺寸已保存"}
}

// GetDetectedKiroInstallPath 自動偵測 Kiro 安裝路徑
func (a *App) GetDetectedKiroInstallPath() Result {
	path, err := kiropath.GetKiroInstallPathAutoDetect()
	if err != nil {
		return Result{Success: false, Message: fmt.Sprintf("偵測失敗: %v", err)}
	}
	return Result{Success: true, Message: path}
}

// GetKiroInstallPathWithStatus 取得 Kiro 安裝路徑及偵測狀態
// 返回結構包含路徑、是否成功、嘗試過的策略、失敗原因等資訊
// 用於前端判斷是否需要引導用戶手動設定路徑
func (a *App) GetKiroInstallPathWithStatus() PathDetectionResult {
	path, err := kiropath.GetKiroInstallPath()
	if err != nil {
		// 檢查是否為 DetectionFailedError，提取詳細資訊
		if detectionErr, ok := err.(*kiropath.DetectionFailedError); ok {
			return PathDetectionResult{
				Path:            "",
				Success:         false,
				TriedStrategies: detectionErr.TriedStrategies,
				FailureReasons:  detectionErr.FailureReasons,
			}
		}
		// 其他錯誤
		return PathDetectionResult{
			Path:            "",
			Success:         false,
			TriedStrategies: []string{},
			FailureReasons:  map[string]string{"error": err.Error()},
		}
	}
	return PathDetectionResult{
		Path:    path,
		Success: true,
	}
}

// GetDetectedKiroVersion 自動偵測 Kiro IDE 執行檔的版本號
func (a *App) GetDetectedKiroVersion() Result {
	version, err := kiroversion.GetKiroVersion()
	if err != nil {
		return Result{Success: false, Message: fmt.Sprintf("偵測版本失敗: %v", err)}
	}
	return Result{Success: true, Message: version}
}

// OpenExtensionFolder 打開 extension.js 所在的文件夾
func (a *App) OpenExtensionFolder() Result {
	extPath, err := softreset.GetExtensionJSPath()
	if err != nil {
		return Result{Success: false, Message: fmt.Sprintf("無法取得 extension.js 路徑: %v", err)}
	}

	// 取得文件夾路徑
	folderPath := filepath.Dir(extPath)

	return openFolder(folderPath)
}

// OpenMachineIDFolder 打開自訂 Machine ID 所在的文件夾 (~/.kiro)
func (a *App) OpenMachineIDFolder() Result {
	idPath, err := softreset.GetCustomMachineIDPath()
	if err != nil {
		return Result{Success: false, Message: fmt.Sprintf("無法取得 Machine ID 路徑: %v", err)}
	}

	// 取得文件夾路徑 (~/.kiro)
	folderPath := filepath.Dir(idPath)

	return openFolder(folderPath)
}

// OpenSSOCacheFolder 打開 AWS SSO Cache 所在的文件夾 (~/.aws/sso/cache)
func (a *App) OpenSSOCacheFolder() Result {
	cachePath, err := awssso.GetSSOCachePath()
	if err != nil {
		return Result{Success: false, Message: fmt.Sprintf("無法取得 SSO Cache 路徑: %v", err)}
	}

	return openFolder(cachePath)
}

// openFolder 使用系統檔案管理器打開指定文件夾
func openFolder(folderPath string) Result {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", folderPath)
	case "darwin":
		cmd = exec.Command("open", folderPath)
	case "linux":
		cmd = exec.Command("xdg-open", folderPath)
	default:
		return Result{Success: false, Message: "不支援的平台"}
	}

	if err := cmd.Start(); err != nil {
		return Result{Success: false, Message: fmt.Sprintf("無法打開文件夾: %v", err)}
	}

	return Result{Success: true, Message: "已打開文件夾"}
}


// ============================================================================
// OAuth 登入功能
// ============================================================================

// OAuthLoginResult OAuth 登入結果（前端用）
type OAuthLoginResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
	Provider     string `json:"provider,omitempty"`
	AuthMethod   string `json:"authMethod,omitempty"`
	// IdC 專用
	ClientId     string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	ClientIdHash string `json:"clientIdHash,omitempty"`
	// IdC 設備授權專用
	UserCode        string `json:"userCode,omitempty"`
	VerificationUri string `json:"verificationUri,omitempty"`
}

// StartSocialLogin 啟動 Social 登入流程
// 參數: provider 為 "Github" 或 "Google"
// 設定 5 分鐘超時，自動開啟瀏覽器
// Windows 平台使用 Deep Link 模式，其他平台使用本地 Callback Server 模式
func (a *App) StartSocialLogin(provider string) OAuthLoginResult {
	// 驗證 provider
	if provider != oauthlogin.ProviderGithub && provider != oauthlogin.ProviderGoogle {
		return OAuthLoginResult{
			Success: false,
			Message: fmt.Sprintf("不支援的登入提供者: %s，請使用 Github 或 Google", provider),
		}
	}

	// 建立帶超時的 context
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()

	// 配置 Social 登入
	config := oauthlogin.SocialLoginCoordinatorConfig{
		Provider:    provider,
		Timeout:     5 * time.Minute,
		OpenBrowser: true,
	}

	var result *oauthlogin.LoginResult
	var err error

	// Windows 平台使用 Deep Link 模式
	if deeplink.IsDeepLinkSupported() {
		result, err = oauthlogin.SocialLoginWithDeepLink(ctx, config)
	} else {
		// 非 Windows 平台使用本地 Callback Server 模式
		result, err = oauthlogin.SocialLogin(ctx, config)
	}

	if err != nil {
		// 處理 OAuth 錯誤
		if oauthErr, ok := err.(*oauthlogin.OAuthError); ok {
			switch oauthErr.Code {
			case oauthlogin.ErrCodeTimeout:
				return OAuthLoginResult{Success: false, Message: "登入超時，請重試"}
			case oauthlogin.ErrCodeCancelled:
				return OAuthLoginResult{Success: false, Message: "登入已取消"}
			case oauthlogin.ErrCodeStateMismatch:
				return OAuthLoginResult{Success: false, Message: "安全驗證失敗，請重試"}
			default:
				return OAuthLoginResult{Success: false, Message: fmt.Sprintf("登入失敗: %s", oauthErr.Message)}
			}
		}
		return OAuthLoginResult{Success: false, Message: fmt.Sprintf("登入失敗: %v", err)}
	}

	// 返回成功結果
	return OAuthLoginResult{
		Success:      true,
		Message:      "登入成功",
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt.Format(time.RFC3339),
		Provider:     result.Provider,
		AuthMethod:   result.AuthMethod,
	}
}

// IdCStartURL Kiro IdC 登入起始 URL
const IdCStartURL = "https://view.awsapps.com/start"

// StartIdCLogin 啟動 IdC 登入流程
// 設定 5 分鐘超時，自動開啟瀏覽器
// 返回結果包含 userCode 和 verificationUri 供前端顯示
func (a *App) StartIdCLogin() OAuthLoginResult {
	// 建立帶超時的 context
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()

	// 配置 IdC 登入
	config := oauthlogin.IdCLoginCoordinatorConfig{
		StartURL:    IdCStartURL,
		ClientName:  "Kiro Manager",
		Timeout:     5 * time.Minute,
		OpenBrowser: true,
	}

	// 執行登入
	result, err := oauthlogin.IdCLogin(ctx, config)
	if err != nil {
		// 處理 OAuth 錯誤
		if oauthErr, ok := err.(*oauthlogin.OAuthError); ok {
			switch oauthErr.Code {
			case oauthlogin.ErrCodeTimeout:
				return OAuthLoginResult{Success: false, Message: "登入超時，請重試"}
			case oauthlogin.ErrCodeCancelled:
				return OAuthLoginResult{Success: false, Message: "登入已取消"}
			default:
				return OAuthLoginResult{Success: false, Message: fmt.Sprintf("登入失敗: %s", oauthErr.Message)}
			}
		}
		return OAuthLoginResult{Success: false, Message: fmt.Sprintf("登入失敗: %v", err)}
	}

	// 返回成功結果（包含 IdC 專用欄位）
	return OAuthLoginResult{
		Success:      true,
		Message:      "登入成功",
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt.Format(time.RFC3339),
		Provider:     result.Provider,
		AuthMethod:   result.AuthMethod,
		ClientId:     result.ClientId,
		ClientSecret: result.ClientSecret,
		ClientIdHash: result.ClientIdHash,
	}
}

// CreateSnapshotFromOAuth 從 OAuth 登入結果建立環境快照
// 將 OAuthLoginResult 轉換為 backup.OAuthBackupData 並建立快照
func (a *App) CreateSnapshotFromOAuth(name string, data OAuthLoginResult) Result {
	// 驗證名稱
	if name == "" {
		return Result{Success: false, Message: "快照名稱不能為空"}
	}

	// 驗證登入結果
	if !data.Success {
		return Result{Success: false, Message: "無法從失敗的登入結果建立快照"}
	}

	// 解析過期時間
	expiresAt, err := time.Parse(time.RFC3339, data.ExpiresAt)
	if err != nil {
		return Result{Success: false, Message: fmt.Sprintf("無效的過期時間格式: %v", err)}
	}

	// 轉換為 backup.OAuthBackupData
	backupData := &backup.OAuthBackupData{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		ExpiresAt:    expiresAt,
		Provider:     data.Provider,
		AuthMethod:   data.AuthMethod,
		ClientId:     data.ClientId,
		ClientSecret: data.ClientSecret,
		ClientIdHash: data.ClientIdHash,
	}

	// 建立快照
	if err := backup.CreateBackupFromOAuth(name, backupData); err != nil {
		return Result{Success: false, Message: fmt.Sprintf("建立快照失敗: %v", err)}
	}

	return Result{Success: true, Message: fmt.Sprintf("已建立快照: %s", name)}
}

// ValidateSnapshotName 驗證快照名稱是否有效
// 規則：不可為空、不可包含非法字元、不可與現有快照重複
func (a *App) ValidateSnapshotName(name string) Result {
	if err := backup.ValidateSnapshotName(name); err != nil {
		return Result{Success: false, Message: err.Error()}
	}
	return Result{Success: true, Message: "名稱有效"}
}


// ============================================================================
// 文件夾管理功能
// ============================================================================

// FolderItem 文件夾項目（前端用）
type FolderItem struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	CreatedAt     string `json:"createdAt"`
	Order         int    `json:"order"`
	SnapshotCount int    `json:"snapshotCount"`
}

// GetFolderList 取得文件夾列表
func (a *App) GetFolderList() ([]FolderItem, error) {
	folders, err := backup.ListFolders()
	if err != nil {
		return nil, err
	}

	items := make([]FolderItem, len(folders))
	for i, f := range folders {
		items[i] = FolderItem{
			ID:            f.ID,
			Name:          f.Name,
			CreatedAt:     f.CreatedAt,
			Order:         f.Order,
			SnapshotCount: f.SnapshotCount,
		}
	}

	return items, nil
}

// CreateFolder 建立新文件夾
func (a *App) CreateFolder(name string) Result {
	folder, err := backup.CreateFolder(name)
	if err != nil {
		return Result{Success: false, Message: err.Error()}
	}
	return Result{Success: true, Message: folder.ID}
}

// RenameFolder 重新命名文件夾
func (a *App) RenameFolder(id, newName string) Result {
	if err := backup.RenameFolder(id, newName); err != nil {
		return Result{Success: false, Message: err.Error()}
	}
	return Result{Success: true, Message: "文件夾已重新命名"}
}

// DeleteFolder 刪除文件夾
// deleteSnapshots: true 表示一併刪除快照，false 表示移到未分類
func (a *App) DeleteFolder(id string, deleteSnapshots bool) Result {
	// 無論選擇哪個選項，都需要先檢查是否包含當前使用中的快照
	// 根據規格：「無法刪除包含當前使用中環境的文件夾」
	currentMachineID := a.GetCurrentMachineID()

	// 取得文件夾中的快照
	data, err := backup.LoadFolders()
	if err != nil {
		return Result{Success: false, Message: err.Error()}
	}

	for snapshotName, folderId := range data.Assignments {
		if folderId == id {
			// 檢查這個快照是否是當前使用中的
			mid, err := backup.ReadBackupMachineID(snapshotName)
			if err == nil && mid.MachineID == currentMachineID {
				return Result{Success: false, Message: "無法刪除包含當前使用中環境的文件夾"}
			}
		}
	}

	snapshotsToDelete, err := backup.DeleteFolder(id, deleteSnapshots)
	if err != nil {
		return Result{Success: false, Message: err.Error()}
	}

	// 如果選擇刪除快照，實際刪除它們
	if deleteSnapshots {
		for _, name := range snapshotsToDelete {
			backup.DeleteBackup(name)
		}
	}

	return Result{Success: true, Message: "文件夾已刪除"}
}

// AssignSnapshotToFolder 將快照分配到文件夾
func (a *App) AssignSnapshotToFolder(snapshotName, folderId string) Result {
	if err := backup.AssignSnapshotToFolder(snapshotName, folderId); err != nil {
		return Result{Success: false, Message: err.Error()}
	}
	return Result{Success: true, Message: "快照已移入文件夾"}
}

// UnassignSnapshot 將快照移至未分類
func (a *App) UnassignSnapshot(snapshotName string) Result {
	if err := backup.UnassignSnapshot(snapshotName); err != nil {
		return Result{Success: false, Message: err.Error()}
	}
	return Result{Success: true, Message: "快照已移至未分類"}
}
