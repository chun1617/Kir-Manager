package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"kiro-manager/backup"
	"kiro-manager/kiroprocess"
	"kiro-manager/machineid"
	"kiro-manager/reset"
	"kiro-manager/softreset"
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
}

// BackupItem 備份項目（前端用）
type BackupItem struct {
	Name              string `json:"name"`
	BackupTime        string `json:"backupTime"`
	HasToken          bool   `json:"hasToken"`
	HasMachineID      bool   `json:"hasMachineId"`
	MachineID         string `json:"machineId"`
	Provider          string `json:"provider"`
	IsCurrent         bool   `json:"isCurrent"`
	IsOriginalMachine bool   `json:"isOriginalMachine"` // Machine ID 與原始機器相同
}

// Result 通用回傳結果
type Result struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetBackupList 取得備份列表
func (a *App) GetBackupList() ([]BackupItem, error) {
	backups, err := backup.ListBackups()
	if err != nil {
		return nil, err
	}

	// 取得當前 Machine ID（優先使用軟重置的自訂 ID）
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

		// 讀取 token 中的 provider
		if b.HasToken {
			token, err := backup.ReadBackupToken(b.Name)
			if err == nil && token.Provider != "" {
				item.Provider = token.Provider
			}
		}

		items = append(items, item)
	}

	return items, nil
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

// SwitchToBackup 切換至指定備份帳號
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

	// 讀取備份的 Machine ID
	mid, err := backup.ReadBackupMachineID(name)
	if err != nil {
		return Result{Success: false, Message: fmt.Sprintf("讀取備份失敗: %v", err)}
	}

	// 檢查平台
	if runtime.GOOS != "windows" {
		return Result{Success: false, Message: "僅支援 Windows 平台"}
	}

	// 設定 Machine ID
	if err := reset.SetWindowsMachineID(mid.MachineID); err != nil {
		if err == reset.ErrRequiresAdmin {
			return Result{Success: false, Message: "需要管理員權限"}
		}
		return Result{Success: false, Message: err.Error()}
	}

	// 恢復 token
	if err := backup.RestoreBackup(name); err != nil {
		return Result{Success: false, Message: fmt.Sprintf("恢復 Token 失敗: %v", err)}
	}

	return Result{Success: true, Message: "切換成功"}
}

// RestoreOriginal 還原原始機器（僅還原 Machine ID，不涉及 token）
func (a *App) RestoreOriginal() Result {
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

	// 讀取原始備份的 Machine ID
	mid, err := backup.ReadBackupMachineID(backup.OriginalBackupName)
	if err != nil {
		return Result{Success: false, Message: fmt.Sprintf("讀取原始備份失敗: %v", err)}
	}

	// 檢查平台
	if runtime.GOOS != "windows" {
		return Result{Success: false, Message: "僅支援 Windows 平台"}
	}

	// 僅還原 Machine ID，不還原 token
	if err := reset.SetWindowsMachineID(mid.MachineID); err != nil {
		if err == reset.ErrRequiresAdmin {
			return Result{Success: false, Message: "需要管理員權限"}
		}
		return Result{Success: false, Message: err.Error()}
	}

	return Result{Success: true, Message: "已還原出廠設定"}
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

// GetCurrentMachineID 取得當前 Machine ID
// 如果軟重置已啟用（有自訂 ID 且已 Patch），返回自訂 ID
// 否則返回系統原始 Machine ID
func (a *App) GetCurrentMachineID() string {
	// 優先檢查軟重置的自訂 Machine ID
	status, err := softreset.GetSoftResetStatus()
	if err == nil && status.IsPatched && status.HasCustomID {
		return status.CustomMachineID
	}

	// 否則返回系統 Machine ID
	id, _ := machineid.GetRawMachineId()
	return id
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

// ResetToNewMachine 一鍵新機
func (a *App) ResetToNewMachine() Result {
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

	// 先確保原始備份存在
	if _, err := backup.EnsureOriginalBackup(); err != nil {
		return Result{Success: false, Message: fmt.Sprintf("備份失敗: %v", err)}
	}

	result, err := reset.ResetEnvironment(true)
	if err != nil {
		if err == reset.ErrRequiresAdmin {
			return Result{Success: false, Message: "需要管理員權限"}
		}
		return Result{Success: false, Message: err.Error()}
	}

	return Result{
		Success: true,
		Message: fmt.Sprintf("重置成功！新 Machine ID: %s", result.NewMachineID[:8]+"..."),
	}
}

// GetAppInfo 取得應用資訊
func (a *App) GetAppInfo() map[string]string {
	return map[string]string{
		"version":   "0.1.1",
		"platform":  runtime.GOOS,
		"buildTime": time.Now().Format("2006-01-02"),
	}
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
// 軟一鍵新機功能（跨平台）
// ============================================================================

// SoftResetStatus 軟重置狀態（前端用）
type SoftResetStatus struct {
	IsPatched       bool   `json:"isPatched"`
	HasCustomID     bool   `json:"hasCustomId"`
	CustomMachineID string `json:"customMachineId"`
	ExtensionPath   string `json:"extensionPath"`
	IsSupported     bool   `json:"isSupported"`
}

// SoftResetToNewMachine 軟一鍵新機（跨平台，不需要管理員權限）
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
		Message: fmt.Sprintf("軟重置成功！新 Machine ID: %s", result.NewMachineID[:8]+"..."),
	}
}

// GetSoftResetStatus 取得軟重置狀態
func (a *App) GetSoftResetStatus() SoftResetStatus {
	status := SoftResetStatus{
		IsSupported: true,
	}

	// 取得軟重置狀態
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

// RestoreSoftReset 還原軟重置（恢復系統原始 Machine ID）
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

	if err := softreset.RestoreOriginalMachineID(); err != nil {
		return Result{Success: false, Message: err.Error()}
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
