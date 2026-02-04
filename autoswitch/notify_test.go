package autoswitch

import (
	"testing"
)

// TestNotificationTypes 驗證通知類型常數
func TestNotificationTypes(t *testing.T) {
	types := []NotifyType{
		NotifySwitch,
		NotifySwitchFail,
		NotifyLowBalance,
		NotifyCooldown,
		NotifyMaxSwitch,
		NotifyCooldownEnd,
		NotifyNoCandidates,
	}

	// 驗證所有類型都是非空字串
	for _, nt := range types {
		if nt == "" {
			t.Error("notification type should not be empty")
		}
	}

	// 驗證類型值
	if NotifySwitch != "switch" {
		t.Errorf("expected NotifySwitch='switch', got '%s'", NotifySwitch)
	}
	if NotifySwitchFail != "switch_fail" {
		t.Errorf("expected NotifySwitchFail='switch_fail', got '%s'", NotifySwitchFail)
	}
	if NotifyLowBalance != "low_balance" {
		t.Errorf("expected NotifyLowBalance='low_balance', got '%s'", NotifyLowBalance)
	}
}

// TestNewSwitchNotification 驗證切換成功通知
func TestNewSwitchNotification(t *testing.T) {
	n := NewSwitchNotification("帳號A", "帳號B")

	if n.Type != NotifySwitch {
		t.Errorf("expected type=%s, got %s", NotifySwitch, n.Type)
	}
	if n.Title != "Kiro Manager" {
		t.Errorf("expected title='Kiro Manager', got '%s'", n.Title)
	}
	if n.Message != "已自動切換至 帳號B" {
		t.Errorf("unexpected message: %s", n.Message)
	}
	if n.Data["from"] != "帳號A" {
		t.Errorf("expected data.from='帳號A', got '%v'", n.Data["from"])
	}
	if n.Data["to"] != "帳號B" {
		t.Errorf("expected data.to='帳號B', got '%v'", n.Data["to"])
	}
}

// TestNewSwitchFailNotification 驗證切換失敗通知
func TestNewSwitchFailNotification(t *testing.T) {
	n := NewSwitchFailNotification("Token 已失效")

	if n.Type != NotifySwitchFail {
		t.Errorf("expected type=%s, got %s", NotifySwitchFail, n.Type)
	}
	if n.Message != "自動切換失敗：Token 已失效" {
		t.Errorf("unexpected message: %s", n.Message)
	}
	if n.Data["reason"] != "Token 已失效" {
		t.Errorf("expected data.reason='Token 已失效', got '%v'", n.Data["reason"])
	}
}

// TestNewLowBalanceNotification 驗證低餘額預警通知
func TestNewLowBalanceNotification(t *testing.T) {
	n := NewLowBalanceNotification(8.5, 5.0)

	if n.Type != NotifyLowBalance {
		t.Errorf("expected type=%s, got %s", NotifyLowBalance, n.Type)
	}
	if n.Message != "餘額即將不足，將自動切換" {
		t.Errorf("unexpected message: %s", n.Message)
	}
	if n.Data["currentBalance"] != 8.5 {
		t.Errorf("expected data.currentBalance=8.5, got %v", n.Data["currentBalance"])
	}
	if n.Data["threshold"] != 5.0 {
		t.Errorf("expected data.threshold=5.0, got %v", n.Data["threshold"])
	}
}

// TestNewCooldownNotification 驗證冷卻期通知
func TestNewCooldownNotification(t *testing.T) {
	n := NewCooldownNotification(180)

	if n.Type != NotifyCooldown {
		t.Errorf("expected type=%s, got %s", NotifyCooldown, n.Type)
	}
	if n.Data["remainingSeconds"] != 180 {
		t.Errorf("expected data.remainingSeconds=180, got %v", n.Data["remainingSeconds"])
	}
}

// TestNewMaxSwitchNotification 驗證達到切換上限通知
func TestNewMaxSwitchNotification(t *testing.T) {
	n := NewMaxSwitchNotification()

	if n.Type != NotifyMaxSwitch {
		t.Errorf("expected type=%s, got %s", NotifyMaxSwitch, n.Type)
	}
	if n.Message != "已達切換上限，暫停自動切換 1 小時" {
		t.Errorf("unexpected message: %s", n.Message)
	}
}

// TestNewCooldownEndNotification 驗證冷卻期結束通知
func TestNewCooldownEndNotification(t *testing.T) {
	n := NewCooldownEndNotification()

	if n.Type != NotifyCooldownEnd {
		t.Errorf("expected type=%s, got %s", NotifyCooldownEnd, n.Type)
	}
	if n.Message != "自動切換已恢復監控" {
		t.Errorf("unexpected message: %s", n.Message)
	}
}

// TestNewNoCandidatesNotification 驗證無候選快照通知
func TestNewNoCandidatesNotification(t *testing.T) {
	n := NewNoCandidatesNotification()

	if n.Type != NotifyNoCandidates {
		t.Errorf("expected type=%s, got %s", NotifyNoCandidates, n.Type)
	}
	if n.Message != "無符合條件的候選快照" {
		t.Errorf("unexpected message: %s", n.Message)
	}
}

// TestNotificationStructure 驗證通知結構
func TestNotificationStructure(t *testing.T) {
	n := &Notification{
		Type:    NotifySwitch,
		Title:   "Test Title",
		Message: "Test Message",
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	if n.Type != NotifySwitch {
		t.Error("Type field not set correctly")
	}
	if n.Title != "Test Title" {
		t.Error("Title field not set correctly")
	}
	if n.Message != "Test Message" {
		t.Error("Message field not set correctly")
	}
	if n.Data["key"] != "value" {
		t.Error("Data field not set correctly")
	}
}
