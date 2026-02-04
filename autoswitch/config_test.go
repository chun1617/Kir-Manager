package autoswitch

import (
	"testing"
	"time"
)

// TestDefaultRefreshIntervals 驗證預設分級規則
func TestDefaultRefreshIntervals(t *testing.T) {
	intervals := DefaultRefreshIntervals()

	// 應該有 3 個分級
	if len(intervals) != 3 {
		t.Errorf("expected 3 intervals, got %d", len(intervals))
	}

	// 驗證第一個分級：餘額 >= 100, 5 分鐘
	if intervals[0].MinBalance != 100 {
		t.Errorf("expected first interval MinBalance 100, got %f", intervals[0].MinBalance)
	}
	if intervals[0].MaxBalance != -1 {
		t.Errorf("expected first interval MaxBalance -1, got %f", intervals[0].MaxBalance)
	}
	if intervals[0].Interval != 5*time.Minute {
		t.Errorf("expected first interval 5m, got %v", intervals[0].Interval)
	}

	// 驗證第二個分級：50 <= 餘額 < 100, 2 分鐘
	if intervals[1].MinBalance != 50 {
		t.Errorf("expected second interval MinBalance 50, got %f", intervals[1].MinBalance)
	}
	if intervals[1].MaxBalance != 100 {
		t.Errorf("expected second interval MaxBalance 100, got %f", intervals[1].MaxBalance)
	}
	if intervals[1].Interval != 2*time.Minute {
		t.Errorf("expected second interval 2m, got %v", intervals[1].Interval)
	}

	// 驗證第三個分級：餘額 < 50, 1 分鐘
	if intervals[2].MinBalance != 0 {
		t.Errorf("expected third interval MinBalance 0, got %f", intervals[2].MinBalance)
	}
	if intervals[2].MaxBalance != 50 {
		t.Errorf("expected third interval MaxBalance 50, got %f", intervals[2].MaxBalance)
	}
	if intervals[2].Interval != 1*time.Minute {
		t.Errorf("expected third interval 1m, got %v", intervals[2].Interval)
	}
}

// TestRefreshIntervalSelection 驗證邊界值選擇
func TestRefreshIntervalSelection(t *testing.T) {
	intervals := DefaultRefreshIntervals()

	testCases := []struct {
		name     string
		balance  float64
		expected time.Duration
	}{
		// 餘額 >= 100 應使用 5 分鐘
		{"balance 120", 120, 5 * time.Minute},
		{"balance 100 (boundary)", 100, 5 * time.Minute},
		{"balance 150", 150, 5 * time.Minute},

		// 50 <= 餘額 < 100 應使用 2 分鐘
		{"balance 99 (just below 100)", 99, 2 * time.Minute},
		{"balance 80", 80, 2 * time.Minute},
		{"balance 50 (boundary)", 50, 2 * time.Minute},

		// 餘額 < 50 應使用 1 分鐘
		{"balance 49 (just below 50)", 49, 1 * time.Minute},
		{"balance 30", 30, 1 * time.Minute},
		{"balance 10", 10, 1 * time.Minute},
		{"balance 0", 0, 1 * time.Minute},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetRefreshInterval(intervals, tc.balance)
			if result != tc.expected {
				t.Errorf("balance %f: expected %v, got %v", tc.balance, tc.expected, result)
			}
		})
	}
}

// TestDefaultAutoSwitchSettings 驗證預設設定
func TestDefaultAutoSwitchSettings(t *testing.T) {
	settings := DefaultAutoSwitchSettings()

	// 預設應該是停用的
	if settings.Enabled {
		t.Error("expected Enabled to be false by default")
	}

	// 預設觸發閾值應該是 5
	if settings.BalanceThreshold != 5 {
		t.Errorf("expected BalanceThreshold 5, got %f", settings.BalanceThreshold)
	}

	// 預設目標最低餘額應該是 50
	if settings.MinTargetBalance != 50 {
		t.Errorf("expected MinTargetBalance 50, got %f", settings.MinTargetBalance)
	}

	// 預設應該啟用通知
	if !settings.NotifyOnSwitch {
		t.Error("expected NotifyOnSwitch to be true by default")
	}
	if !settings.NotifyOnLowBalance {
		t.Error("expected NotifyOnLowBalance to be true by default")
	}

	// 預設應該有刷新間隔規則
	if len(settings.RefreshIntervals) == 0 {
		t.Error("expected RefreshIntervals to have default values")
	}
}

// TestAutoSwitchSettingsStructure 驗證設定結構欄位
func TestAutoSwitchSettingsStructure(t *testing.T) {
	settings := AutoSwitchSettings{
		Enabled:            true,
		BalanceThreshold:   10,
		MinTargetBalance:   100,
		FolderIds:          []string{"folder-1", "folder-2"},
		SubscriptionTypes:  []string{"Pro", "Enterprise"},
		RefreshIntervals:   DefaultRefreshIntervals(),
		NotifyOnSwitch:     true,
		NotifyOnLowBalance: false,
	}

	if !settings.Enabled {
		t.Error("expected Enabled to be true")
	}
	if settings.BalanceThreshold != 10 {
		t.Errorf("expected BalanceThreshold 10, got %f", settings.BalanceThreshold)
	}
	if len(settings.FolderIds) != 2 {
		t.Errorf("expected 2 FolderIds, got %d", len(settings.FolderIds))
	}
	if len(settings.SubscriptionTypes) != 2 {
		t.Errorf("expected 2 SubscriptionTypes, got %d", len(settings.SubscriptionTypes))
	}
}


// TestAutoSwitchSettingsClone 驗證設定深拷貝
func TestAutoSwitchSettingsClone(t *testing.T) {
	original := &AutoSwitchSettings{
		Enabled:            true,
		BalanceThreshold:   10,
		MinTargetBalance:   100,
		FolderIds:          []string{"folder1", "folder2"},
		SubscriptionTypes:  []string{"Pro", "Team"},
		RefreshIntervals:   DefaultRefreshIntervals(),
		NotifyOnSwitch:     true,
		NotifyOnLowBalance: false,
	}

	clone := original.Clone()

	// 驗證值相等
	if clone.Enabled != original.Enabled {
		t.Errorf("Enabled mismatch: got %v, want %v", clone.Enabled, original.Enabled)
	}
	if clone.BalanceThreshold != original.BalanceThreshold {
		t.Errorf("BalanceThreshold mismatch: got %v, want %v", clone.BalanceThreshold, original.BalanceThreshold)
	}
	if clone.MinTargetBalance != original.MinTargetBalance {
		t.Errorf("MinTargetBalance mismatch: got %v, want %v", clone.MinTargetBalance, original.MinTargetBalance)
	}
	if clone.NotifyOnSwitch != original.NotifyOnSwitch {
		t.Errorf("NotifyOnSwitch mismatch: got %v, want %v", clone.NotifyOnSwitch, original.NotifyOnSwitch)
	}
	if clone.NotifyOnLowBalance != original.NotifyOnLowBalance {
		t.Errorf("NotifyOnLowBalance mismatch: got %v, want %v", clone.NotifyOnLowBalance, original.NotifyOnLowBalance)
	}

	// 驗證切片是深拷貝（修改原始不影響克隆）
	original.FolderIds[0] = "modified"
	if clone.FolderIds[0] == "modified" {
		t.Error("FolderIds is not a deep copy")
	}

	original.SubscriptionTypes[0] = "modified"
	if clone.SubscriptionTypes[0] == "modified" {
		t.Error("SubscriptionTypes is not a deep copy")
	}

	// 驗證 nil 處理
	var nilSettings *AutoSwitchSettings
	nilClone := nilSettings.Clone()
	if nilClone != nil {
		t.Error("Clone of nil should return nil")
	}
}

// TestAutoSwitchSettingsClone_EmptySlices 驗證空切片的深拷貝
func TestAutoSwitchSettingsClone_EmptySlices(t *testing.T) {
	original := &AutoSwitchSettings{
		Enabled:           true,
		FolderIds:         []string{},
		SubscriptionTypes: []string{},
		RefreshIntervals:  []RefreshInterval{},
	}

	clone := original.Clone()

	if clone.FolderIds == nil {
		t.Error("FolderIds should not be nil")
	}
	if len(clone.FolderIds) != 0 {
		t.Error("FolderIds should be empty")
	}
}

// TestAutoSwitchSettingsClone_NilSlices 驗證 nil 切片的處理
func TestAutoSwitchSettingsClone_NilSlices(t *testing.T) {
	original := &AutoSwitchSettings{
		Enabled:           true,
		FolderIds:         nil,
		SubscriptionTypes: nil,
		RefreshIntervals:  nil,
	}

	clone := original.Clone()

	if clone.FolderIds != nil {
		t.Error("FolderIds should be nil")
	}
	if clone.SubscriptionTypes != nil {
		t.Error("SubscriptionTypes should be nil")
	}
	if clone.RefreshIntervals != nil {
		t.Error("RefreshIntervals should be nil")
	}
}
