package autoswitch

import (
	"testing"
	"time"
)

// TestCooldownPeriod 驗證冷卻期
func TestCooldownPeriod(t *testing.T) {
	state := NewSafetyState()

	// 初始狀態應該可以切換
	canSwitch, reason := state.CanSwitch()
	if !canSwitch {
		t.Errorf("expected canSwitch=true initially, got false: %s", reason)
	}

	// 記錄切換
	state.RecordSwitch()

	// 切換後應該在冷卻期內
	canSwitch, reason = state.CanSwitch()
	if canSwitch {
		t.Error("expected canSwitch=false during cooldown")
	}
	if reason == "" {
		t.Error("expected reason to be non-empty during cooldown")
	}

	// 驗證冷卻期剩餘時間
	remaining := state.GetCooldownRemaining()
	if remaining <= 0 || remaining > CooldownPeriod {
		t.Errorf("expected remaining in (0, %v], got %v", CooldownPeriod, remaining)
	}
}

// TestCooldownPeriod_Expired 驗證冷卻期過期
func TestCooldownPeriod_Expired(t *testing.T) {
	state := NewSafetyState()

	// 模擬過去的切換時間（超過冷卻期）
	state.mu.Lock()
	state.LastSwitchTime = time.Now().Add(-CooldownPeriod - time.Second)
	state.mu.Unlock()

	// 冷卻期過期後應該可以切換
	canSwitch, _ := state.CanSwitch()
	if !canSwitch {
		t.Error("expected canSwitch=true after cooldown expired")
	}

	// 冷卻期剩餘時間應該為 0
	remaining := state.GetCooldownRemaining()
	if remaining != 0 {
		t.Errorf("expected remaining=0 after cooldown expired, got %v", remaining)
	}
}

// TestMaxSwitchLimit 驗證切換次數上限
func TestMaxSwitchLimit(t *testing.T) {
	state := NewSafetyState()

	// 模擬已達切換上限（不觸發冷卻期）
	state.mu.Lock()
	state.SwitchCount = MaxSwitchPerHour
	state.LastSwitchTime = time.Now().Add(-CooldownPeriod - time.Second) // 冷卻期已過
	state.mu.Unlock()

	// 應該因為達到上限而無法切換
	canSwitch, reason := state.CanSwitch()
	if canSwitch {
		t.Error("expected canSwitch=false when max switch limit reached")
	}
	if reason != "已達切換上限，暫停自動切換 1 小時" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

// TestCountReset 驗證計數重置
func TestCountReset(t *testing.T) {
	state := NewSafetyState()

	// 模擬過去的計數（超過重置週期）
	state.mu.Lock()
	state.SwitchCount = MaxSwitchPerHour
	state.CountResetTime = time.Now().Add(-CountResetPeriod - time.Second)
	state.LastSwitchTime = time.Now().Add(-CooldownPeriod - time.Second) // 冷卻期已過
	state.mu.Unlock()

	// 計數應該被重置，可以切換
	canSwitch, _ := state.CanSwitch()
	if !canSwitch {
		t.Error("expected canSwitch=true after count reset")
	}

	// 驗證計數已重置
	count := state.GetSwitchCount()
	if count != 0 {
		t.Errorf("expected count=0 after reset, got %d", count)
	}
}

// TestRecordSwitch 驗證記錄切換
func TestRecordSwitch(t *testing.T) {
	state := NewSafetyState()

	// 初始計數應該為 0
	if state.GetSwitchCount() != 0 {
		t.Error("expected initial count=0")
	}

	// 記錄切換
	state.RecordSwitch()

	// 計數應該增加
	if state.GetSwitchCount() != 1 {
		t.Errorf("expected count=1 after first switch, got %d", state.GetSwitchCount())
	}

	// 模擬冷卻期過期後再次切換
	state.mu.Lock()
	state.LastSwitchTime = time.Now().Add(-CooldownPeriod - time.Second)
	state.mu.Unlock()

	state.RecordSwitch()

	// 計數應該再增加
	if state.GetSwitchCount() != 2 {
		t.Errorf("expected count=2 after second switch, got %d", state.GetSwitchCount())
	}
}

// TestSafetyState_ConcurrentAccess 驗證並發安全
func TestSafetyState_ConcurrentAccess(t *testing.T) {
	state := NewSafetyState()

	done := make(chan bool, 10)

	// 並發讀取
	for i := 0; i < 5; i++ {
		go func() {
			state.CanSwitch()
			state.GetCooldownRemaining()
			state.GetSwitchCount()
			done <- true
		}()
	}

	// 並發寫入
	for i := 0; i < 5; i++ {
		go func() {
			state.RecordSwitch()
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 如果沒有 panic 或 race condition，測試通過
}

// TestNewSafetyState 驗證初始狀態
func TestNewSafetyState(t *testing.T) {
	state := NewSafetyState()

	if state == nil {
		t.Fatal("expected non-nil state")
	}

	// 初始狀態應該可以切換
	canSwitch, _ := state.CanSwitch()
	if !canSwitch {
		t.Error("expected canSwitch=true for new state")
	}

	// 初始計數應該為 0
	if state.GetSwitchCount() != 0 {
		t.Error("expected count=0 for new state")
	}

	// 初始冷卻期剩餘應該為 0
	if state.GetCooldownRemaining() != 0 {
		t.Error("expected cooldown remaining=0 for new state")
	}
}

// TestResetForTesting 驗證測試重置功能
func TestResetForTesting(t *testing.T) {
	state := NewSafetyState()

	// 記錄一些切換
	state.RecordSwitch()
	state.RecordSwitch()

	// 重置
	state.ResetForTesting()

	// 驗證狀態已重置
	if state.GetSwitchCount() != 0 {
		t.Error("expected count=0 after reset")
	}
	if state.GetCooldownRemaining() != 0 {
		t.Error("expected cooldown remaining=0 after reset")
	}
	canSwitch, _ := state.CanSwitch()
	if !canSwitch {
		t.Error("expected canSwitch=true after reset")
	}
}

// TestCooldownMessage 驗證冷卻期訊息格式
func TestCooldownMessage(t *testing.T) {
	state := NewSafetyState()
	state.RecordSwitch()

	_, reason := state.CanSwitch()

	// 訊息應該包含「冷卻期內」和「後恢復」
	if reason == "" {
		t.Error("expected non-empty cooldown message")
	}
}

// TestConstants 驗證常數值
func TestConstants(t *testing.T) {
	if CooldownPeriod != 5*time.Minute {
		t.Errorf("expected CooldownPeriod=5m, got %v", CooldownPeriod)
	}
	if MaxSwitchPerHour != 3 {
		t.Errorf("expected MaxSwitchPerHour=3, got %d", MaxSwitchPerHour)
	}
	if CountResetPeriod != 1*time.Hour {
		t.Errorf("expected CountResetPeriod=1h, got %v", CountResetPeriod)
	}
}

// TestFormatDuration 驗證時間格式化函數
func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		name     string
		value    int
		unit     string
		expected string
	}{
		{"zero value", 0, "秒", ""},
		{"negative value", -5, "秒", ""},
		{"single digit", 5, "秒", "5 秒"},
		{"double digit", 30, "秒", "30 秒"},
		{"minutes single digit", 3, "分鐘", "3 分鐘"},
		{"minutes double digit", 45, "分鐘", "45 分鐘"},
		{"boundary 59", 59, "秒", "59 秒"},
		{"value 1", 1, "秒", "1 秒"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatDuration(tc.value, tc.unit)
			if result != tc.expected {
				t.Errorf("formatDuration(%d, %q) = %q, expected %q", tc.value, tc.unit, result, tc.expected)
			}
		})
	}
}
