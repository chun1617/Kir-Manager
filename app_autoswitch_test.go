package main

import (
	"context"
	"testing"

	"kiro-manager/autoswitch"
)

// TestAutoSwitchSettingsDTO_DefaultValues 驗證 DTO 預設值
func TestAutoSwitchSettingsDTO_DefaultValues(t *testing.T) {
	dto := AutoSwitchSettingsDTO{}

	// 驗證預設值為零值
	if dto.Enabled != false {
		t.Errorf("Expected Enabled to be false, got %v", dto.Enabled)
	}
	if dto.BalanceThreshold != 0 {
		t.Errorf("Expected BalanceThreshold to be 0, got %v", dto.BalanceThreshold)
	}
	if dto.MinTargetBalance != 0 {
		t.Errorf("Expected MinTargetBalance to be 0, got %v", dto.MinTargetBalance)
	}
	if dto.FolderIds != nil {
		t.Errorf("Expected FolderIds to be nil, got %v", dto.FolderIds)
	}
	if dto.SubscriptionTypes != nil {
		t.Errorf("Expected SubscriptionTypes to be nil, got %v", dto.SubscriptionTypes)
	}
	if dto.NotifyOnSwitch != false {
		t.Errorf("Expected NotifyOnSwitch to be false, got %v", dto.NotifyOnSwitch)
	}
	if dto.NotifyOnLowBalance != false {
		t.Errorf("Expected NotifyOnLowBalance to be false, got %v", dto.NotifyOnLowBalance)
	}
}

// TestAutoSwitchSettingsDTO_WithValues 驗證 DTO 設定值
func TestAutoSwitchSettingsDTO_WithValues(t *testing.T) {
	dto := AutoSwitchSettingsDTO{
		Enabled:            true,
		BalanceThreshold:   10.0,
		MinTargetBalance:   50.0,
		FolderIds:          []string{"folder1", "folder2"},
		SubscriptionTypes:  []string{"pro", "enterprise"},
		NotifyOnSwitch:     true,
		NotifyOnLowBalance: false,
	}

	if dto.Enabled != true {
		t.Errorf("Expected Enabled true, got %v", dto.Enabled)
	}
	if dto.BalanceThreshold != 10.0 {
		t.Errorf("Expected BalanceThreshold 10.0, got %v", dto.BalanceThreshold)
	}
	if dto.MinTargetBalance != 50.0 {
		t.Errorf("Expected MinTargetBalance 50.0, got %v", dto.MinTargetBalance)
	}
	if len(dto.FolderIds) != 2 {
		t.Errorf("Expected FolderIds length 2, got %v", len(dto.FolderIds))
	}
	if len(dto.SubscriptionTypes) != 2 {
		t.Errorf("Expected SubscriptionTypes length 2, got %v", len(dto.SubscriptionTypes))
	}
	if dto.NotifyOnSwitch != true {
		t.Errorf("Expected NotifyOnSwitch true, got %v", dto.NotifyOnSwitch)
	}
	if dto.NotifyOnLowBalance != false {
		t.Errorf("Expected NotifyOnLowBalance false, got %v", dto.NotifyOnLowBalance)
	}
}

// TestAutoSwitchStatus_Fields 驗證狀態結構欄位
func TestAutoSwitchStatus_Fields(t *testing.T) {
	status := AutoSwitchStatus{
		Status:            "running",
		LastBalance:       75.5,
		CooldownRemaining: 120,
		SwitchCount:       2,
	}

	if status.Status != "running" {
		t.Errorf("Expected Status 'running', got '%s'", status.Status)
	}
	if status.LastBalance != 75.5 {
		t.Errorf("Expected LastBalance 75.5, got %v", status.LastBalance)
	}
	if status.CooldownRemaining != 120 {
		t.Errorf("Expected CooldownRemaining 120, got %v", status.CooldownRemaining)
	}
	if status.SwitchCount != 2 {
		t.Errorf("Expected SwitchCount 2, got %v", status.SwitchCount)
	}
}

// TestAutoSwitchStatus_StoppedState 驗證停止狀態
func TestAutoSwitchStatus_StoppedState(t *testing.T) {
	status := AutoSwitchStatus{
		Status:            "stopped",
		LastBalance:       0,
		CooldownRemaining: 0,
		SwitchCount:       0,
	}

	if status.Status != "stopped" {
		t.Errorf("Expected Status 'stopped', got '%s'", status.Status)
	}
}

// TestAutoSwitchStatus_CooldownState 驗證冷卻狀態
func TestAutoSwitchStatus_CooldownState(t *testing.T) {
	status := AutoSwitchStatus{
		Status:            "cooldown",
		LastBalance:       25.0,
		CooldownRemaining: 180,
		SwitchCount:       1,
	}

	if status.Status != "cooldown" {
		t.Errorf("Expected Status 'cooldown', got '%s'", status.Status)
	}
	if status.CooldownRemaining != 180 {
		t.Errorf("Expected CooldownRemaining 180, got %v", status.CooldownRemaining)
	}
}

// TestGetAutoSwitchSettings_ReturnsDefaultWhenNil 驗證當設定為 nil 時返回預設值
func TestGetAutoSwitchSettings_ReturnsDefaultWhenNil(t *testing.T) {
	app := &App{ctx: context.Background()}

	// 取得設定（應該返回預設值）
	dto := app.GetAutoSwitchSettings()

	// 驗證預設值
	defaults := autoswitch.DefaultAutoSwitchSettings()
	if dto.Enabled != defaults.Enabled {
		t.Errorf("Expected Enabled %v, got %v", defaults.Enabled, dto.Enabled)
	}
	if dto.BalanceThreshold != defaults.BalanceThreshold {
		t.Errorf("Expected BalanceThreshold %v, got %v", defaults.BalanceThreshold, dto.BalanceThreshold)
	}
	if dto.MinTargetBalance != defaults.MinTargetBalance {
		t.Errorf("Expected MinTargetBalance %v, got %v", defaults.MinTargetBalance, dto.MinTargetBalance)
	}
	if dto.NotifyOnSwitch != defaults.NotifyOnSwitch {
		t.Errorf("Expected NotifyOnSwitch %v, got %v", defaults.NotifyOnSwitch, dto.NotifyOnSwitch)
	}
	if dto.NotifyOnLowBalance != defaults.NotifyOnLowBalance {
		t.Errorf("Expected NotifyOnLowBalance %v, got %v", defaults.NotifyOnLowBalance, dto.NotifyOnLowBalance)
	}
}

// TestGetAutoSwitchStatus_InitialState 驗證初始狀態
func TestGetAutoSwitchStatus_InitialState(t *testing.T) {
	// 確保 autoSwitchMonitor 為 nil
	autoSwitchMonitor = nil

	app := &App{ctx: context.Background()}

	// 取得狀態
	status := app.GetAutoSwitchStatus()

	// 驗證初始狀態為 stopped
	if status.Status != "stopped" {
		t.Errorf("Expected initial status 'stopped', got '%s'", status.Status)
	}
	if status.LastBalance != 0 {
		t.Errorf("Expected LastBalance 0, got %v", status.LastBalance)
	}
	if status.CooldownRemaining != 0 {
		t.Errorf("Expected CooldownRemaining 0, got %v", status.CooldownRemaining)
	}
	if status.SwitchCount != 0 {
		t.Errorf("Expected SwitchCount 0, got %v", status.SwitchCount)
	}
}

// TestStopAutoSwitchMonitor_WhenNotStarted 驗證未啟動時停止
func TestStopAutoSwitchMonitor_WhenNotStarted(t *testing.T) {
	// 確保 autoSwitchMonitor 為 nil
	autoSwitchMonitor = nil

	app := &App{ctx: context.Background()}

	// 停止監控（應該成功，因為本來就沒啟動）
	result := app.StopAutoSwitchMonitor()

	if !result.Success {
		t.Errorf("Expected StopAutoSwitchMonitor to succeed, got: %s", result.Message)
	}
}

// TestSwitchToBackup_GlobalLock 驗證全域鎖
func TestSwitchToBackup_GlobalLock(t *testing.T) {
	app := &App{ctx: context.Background()}

	// 先取得全域鎖
	globalSwitchMu.Lock()

	// 嘗試切換應該失敗（因為鎖被佔用）
	result := app.SwitchToBackup("test-backup")

	// 釋放鎖
	globalSwitchMu.Unlock()

	// 驗證結果
	if result.Success {
		t.Error("Expected SwitchToBackup to fail when lock is held")
	}
	if result.Message != "正在切換中，請稍後再試" {
		t.Errorf("Expected lock message, got: %s", result.Message)
	}
}

// TestSwitchToBackup_EmptyName 驗證空名稱
func TestSwitchToBackup_EmptyName(t *testing.T) {
	app := &App{ctx: context.Background()}

	result := app.SwitchToBackup("")

	if result.Success {
		t.Error("Expected SwitchToBackup to fail with empty name")
	}
	if result.Message != "請選擇備份" {
		t.Errorf("Expected empty name message, got: %s", result.Message)
	}
}

// TestSwitchToBackup_NonExistentBackup 驗證不存在的備份
func TestSwitchToBackup_NonExistentBackup(t *testing.T) {
	app := &App{ctx: context.Background()}

	result := app.SwitchToBackup("non-existent-backup-12345")

	if result.Success {
		t.Error("Expected SwitchToBackup to fail with non-existent backup")
	}
	if result.Message != "備份不存在" {
		t.Errorf("Expected non-existent backup message, got: %s", result.Message)
	}
}
