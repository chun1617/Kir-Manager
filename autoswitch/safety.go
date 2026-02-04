package autoswitch

import (
	"strconv"
	"sync"
	"time"
)

const (
	// CooldownPeriod 切換後冷卻期
	CooldownPeriod = 5 * time.Minute
	// MaxSwitchPerHour 每小時最多切換次數
	MaxSwitchPerHour = 3
	// CountResetPeriod 計數重置週期
	CountResetPeriod = 1 * time.Hour
)

// SafetyState 安全狀態
type SafetyState struct {
	LastSwitchTime time.Time
	SwitchCount    int
	CountResetTime time.Time
	mu             sync.Mutex
}

// NewSafetyState 建立新的安全狀態
func NewSafetyState() *SafetyState {
	return &SafetyState{
		CountResetTime: time.Now(),
	}
}

// CanSwitch 檢查是否可以執行切換
// 返回：(是否可切換, 不可切換的原因)
func (s *SafetyState) CanSwitch() (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// 檢查計數是否需要重置
	if now.Sub(s.CountResetTime) >= CountResetPeriod {
		s.SwitchCount = 0
		s.CountResetTime = now
	}

	// 檢查冷卻期
	if !s.LastSwitchTime.IsZero() {
		elapsed := now.Sub(s.LastSwitchTime)
		if elapsed < CooldownPeriod {
			remaining := CooldownPeriod - elapsed
			return false, formatCooldownMessage(remaining)
		}
	}

	// 檢查切換次數上限
	if s.SwitchCount >= MaxSwitchPerHour {
		return false, "已達切換上限，暫停自動切換 1 小時"
	}

	return true, ""
}

// RecordSwitch 記錄切換
func (s *SafetyState) RecordSwitch() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// 檢查計數是否需要重置
	if now.Sub(s.CountResetTime) >= CountResetPeriod {
		s.SwitchCount = 0
		s.CountResetTime = now
	}

	s.LastSwitchTime = now
	s.SwitchCount++
}

// GetCooldownRemaining 取得冷卻期剩餘時間
// 返回 0 表示不在冷卻期
func (s *SafetyState) GetCooldownRemaining() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.LastSwitchTime.IsZero() {
		return 0
	}

	elapsed := time.Since(s.LastSwitchTime)
	if elapsed >= CooldownPeriod {
		return 0
	}

	return CooldownPeriod - elapsed
}

// GetSwitchCount 取得當前切換次數
func (s *SafetyState) GetSwitchCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 檢查計數是否需要重置
	if time.Since(s.CountResetTime) >= CountResetPeriod {
		return 0
	}

	return s.SwitchCount
}

// ResetForTesting 重置狀態（僅供測試使用）
func (s *SafetyState) ResetForTesting() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastSwitchTime = time.Time{}
	s.SwitchCount = 0
	s.CountResetTime = time.Now()
}

// formatCooldownMessage 格式化冷卻期訊息
func formatCooldownMessage(remaining time.Duration) string {
	minutes := int(remaining.Minutes())
	seconds := int(remaining.Seconds()) % 60

	if minutes > 0 {
		return "冷卻期內，" + formatDuration(minutes, "分鐘") + formatDuration(seconds, "秒") + "後恢復"
	}
	return "冷卻期內，" + formatDuration(seconds, "秒") + "後恢復"
}

// formatDuration 格式化時間單位
func formatDuration(value int, unit string) string {
	if value <= 0 {
		return ""
	}
	// 使用 strconv 避免前導零問題
	return strconv.Itoa(value) + " " + unit
}
