package autoswitch

import (
	"time"
)

// AutoSwitchSettings 自動切換設定結構
type AutoSwitchSettings struct {
	// Enabled 是否啟用自動切換
	Enabled bool `json:"enabled"`
	// BalanceThreshold 觸發閾值（絕對值）
	// 當餘額 <= 此值時觸發自動切換
	BalanceThreshold float64 `json:"balanceThreshold"`
	// MinTargetBalance 目標最低餘額
	// 只切換至餘額 >= 此值的快照
	MinTargetBalance float64 `json:"minTargetBalance"`
	// FolderIds 限定文件夾 ID 列表
	// 空列表表示不限制
	FolderIds []string `json:"folderIds"`
	// SubscriptionTypes 限定訂閱類型列表
	// 空列表表示不限制
	SubscriptionTypes []string `json:"subscriptionTypes"`
	// RefreshIntervals 刷新頻率分級規則
	RefreshIntervals []RefreshInterval `json:"refreshIntervals"`
	// NotifyOnSwitch 切換時是否通知
	NotifyOnSwitch bool `json:"notifyOnSwitch"`
	// NotifyOnLowBalance 低餘額時是否預警
	NotifyOnLowBalance bool `json:"notifyOnLowBalance"`
}

// RefreshInterval 刷新頻率分級規則
// 使用左閉右開區間：MinBalance <= 餘額 < MaxBalance
type RefreshInterval struct {
	// MinBalance 餘額下限（含）
	MinBalance float64 `json:"minBalance"`
	// MaxBalance 餘額上限（不含），-1 表示無上限
	MaxBalance float64 `json:"maxBalance"`
	// Interval 刷新間隔
	Interval time.Duration `json:"interval"`
}

// DefaultRefreshIntervals 預設刷新頻率分級規則
// 根據 BDD 規格：
// - 餘額 >= 100: 5 分鐘
// - 50 <= 餘額 < 100: 2 分鐘
// - 餘額 < 50: 1 分鐘
func DefaultRefreshIntervals() []RefreshInterval {
	return []RefreshInterval{
		{MinBalance: 100, MaxBalance: -1, Interval: 5 * time.Minute},
		{MinBalance: 50, MaxBalance: 100, Interval: 2 * time.Minute},
		{MinBalance: 0, MaxBalance: 50, Interval: 1 * time.Minute},
	}
}

// GetRefreshInterval 根據餘額取得對應的刷新間隔
// 使用左閉右開區間匹配
func GetRefreshInterval(intervals []RefreshInterval, balance float64) time.Duration {
	for _, interval := range intervals {
		// 檢查是否在區間內
		if balance >= interval.MinBalance {
			// MaxBalance == -1 表示無上限
			if interval.MaxBalance == -1 || balance < interval.MaxBalance {
				return interval.Interval
			}
		}
	}
	// 預設返回 1 分鐘
	return 1 * time.Minute
}

// DefaultAutoSwitchSettings 取得預設自動切換設定
func DefaultAutoSwitchSettings() *AutoSwitchSettings {
	return &AutoSwitchSettings{
		Enabled:            false,
		BalanceThreshold:   5,
		MinTargetBalance:   50,
		FolderIds:          []string{},
		SubscriptionTypes:  []string{},
		RefreshIntervals:   DefaultRefreshIntervals(),
		NotifyOnSwitch:     true,
		NotifyOnLowBalance: true,
	}
}

// Clone 建立設定的深拷貝
// 用於在切換過程中保持設定一致性
func (s *AutoSwitchSettings) Clone() *AutoSwitchSettings {
	if s == nil {
		return nil
	}

	clone := &AutoSwitchSettings{
		Enabled:            s.Enabled,
		BalanceThreshold:   s.BalanceThreshold,
		MinTargetBalance:   s.MinTargetBalance,
		NotifyOnSwitch:     s.NotifyOnSwitch,
		NotifyOnLowBalance: s.NotifyOnLowBalance,
	}

	// 深拷貝 FolderIds
	if s.FolderIds != nil {
		clone.FolderIds = make([]string, len(s.FolderIds))
		copy(clone.FolderIds, s.FolderIds)
	}

	// 深拷貝 SubscriptionTypes
	if s.SubscriptionTypes != nil {
		clone.SubscriptionTypes = make([]string, len(s.SubscriptionTypes))
		copy(clone.SubscriptionTypes, s.SubscriptionTypes)
	}

	// 深拷貝 RefreshIntervals
	if s.RefreshIntervals != nil {
		clone.RefreshIntervals = make([]RefreshInterval, len(s.RefreshIntervals))
		copy(clone.RefreshIntervals, s.RefreshIntervals)
	}

	return clone
}
