package autoswitch

import (
	"context"
)

// NotifyType 通知類型
type NotifyType string

const (
	NotifySwitch       NotifyType = "switch"        // 切換成功
	NotifySwitchFail   NotifyType = "switch_fail"   // 切換失敗
	NotifyLowBalance   NotifyType = "low_balance"   // 低餘額預警
	NotifyCooldown     NotifyType = "cooldown"      // 冷卻期
	NotifyMaxSwitch    NotifyType = "max_switch"    // 達到切換上限
	NotifyCooldownEnd  NotifyType = "cooldown_end"  // 冷卻期結束
	NotifyNoCandidates NotifyType = "no_candidates" // 無候選快照
)

// Notification 通知結構
type Notification struct {
	Type    NotifyType             `json:"type"`
	Title   string                 `json:"title"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// NotifyFunc 通知回調函數類型
// 用於將通知發送到前端 (Toast) 和系統托盤
type NotifyFunc func(ctx context.Context, n *Notification)

// NewSwitchNotification 建立切換成功通知
func NewSwitchNotification(fromName, toName string) *Notification {
	return &Notification{
		Type:    NotifySwitch,
		Title:   "Kiro Manager",
		Message: "已自動切換至 " + toName,
		Data: map[string]interface{}{
			"from": fromName,
			"to":   toName,
		},
	}
}

// NewSwitchFailNotification 建立切換失敗通知
func NewSwitchFailNotification(reason string) *Notification {
	return &Notification{
		Type:    NotifySwitchFail,
		Title:   "Kiro Manager",
		Message: "自動切換失敗：" + reason,
		Data: map[string]interface{}{
			"reason": reason,
		},
	}
}

// NewLowBalanceNotification 建立低餘額預警通知
func NewLowBalanceNotification(currentBalance float64, threshold float64) *Notification {
	return &Notification{
		Type:    NotifyLowBalance,
		Title:   "Kiro Manager",
		Message: "餘額即將不足，將自動切換",
		Data: map[string]interface{}{
			"currentBalance": currentBalance,
			"threshold":      threshold,
		},
	}
}

// NewCooldownNotification 建立冷卻期通知
func NewCooldownNotification(remainingSeconds int) *Notification {
	return &Notification{
		Type:    NotifyCooldown,
		Title:   "Kiro Manager",
		Message: "冷卻期內，稍後恢復自動切換",
		Data: map[string]interface{}{
			"remainingSeconds": remainingSeconds,
		},
	}
}

// NewMaxSwitchNotification 建立達到切換上限通知
func NewMaxSwitchNotification() *Notification {
	return &Notification{
		Type:    NotifyMaxSwitch,
		Title:   "Kiro Manager",
		Message: "已達切換上限，暫停自動切換 1 小時",
	}
}

// NewCooldownEndNotification 建立冷卻期結束通知
func NewCooldownEndNotification() *Notification {
	return &Notification{
		Type:    NotifyCooldownEnd,
		Title:   "Kiro Manager",
		Message: "自動切換已恢復監控",
	}
}

// NewNoCandidatesNotification 建立無候選快照通知
func NewNoCandidatesNotification() *Notification {
	return &Notification{
		Type:    NotifyNoCandidates,
		Title:   "Kiro Manager",
		Message: "無符合條件的候選快照",
	}
}
