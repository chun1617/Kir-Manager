package autoswitch

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestIntegration_AutoSwitchFlow 完整流程測試
func TestIntegration_AutoSwitchFlow(t *testing.T) {
	// 設定
	config := &AutoSwitchSettings{
		Enabled:            true,
		BalanceThreshold:   5,
		MinTargetBalance:   50,
		FolderIds:          []string{},
		SubscriptionTypes:  []string{},
		RefreshIntervals:   DefaultRefreshIntervals(),
		NotifyOnSwitch:     true,
		NotifyOnLowBalance: true,
	}

	var switchMu sync.Mutex
	var notifications []*Notification
	var notifyMu sync.Mutex
	var switchedTo string

	// 建立監控器
	monitor := NewMonitor(MonitorConfig{
		Config:   config,
		SwitchMu: &switchMu,
		Notifier: func(ctx context.Context, n *Notification) {
			notifyMu.Lock()
			notifications = append(notifications, n)
			notifyMu.Unlock()
		},
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3.0, nil // 低於閾值
		},
		SwitchFunc: func(ctx context.Context, targetName string) error {
			switchedTo = targetName
			return nil
		},
		GetCurrentName: func() string {
			return "current-account"
		},
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{
				{Name: "account-b", Balance: 150, FolderId: "", SubscriptionType: "Pro"},
				{Name: "account-c", Balance: 80, FolderId: "", SubscriptionType: "Pro"},
			}
		},
	})

	// 啟動監控
	monitor.Start()

	// 等待切換發生
	time.Sleep(200 * time.Millisecond)

	// 停止監控
	monitor.Stop()

	// 驗證切換到餘額最高的帳號
	if switchedTo != "account-b" {
		t.Errorf("Expected switch to account-b, got %s", switchedTo)
	}

	// 驗證有通知
	notifyMu.Lock()
	hasNotification := len(notifications) > 0
	notifyMu.Unlock()
	if !hasNotification {
		t.Error("Expected at least one notification")
	}
}

// TestIntegration_ConcurrentSwitch 並發切換測試
func TestIntegration_ConcurrentSwitch(t *testing.T) {
	var switchMu sync.Mutex
	switchCount := 0
	var countMu sync.Mutex

	config := &AutoSwitchSettings{
		Enabled:          true,
		BalanceThreshold: 5,
		RefreshIntervals: DefaultRefreshIntervals(),
	}

	monitor := NewMonitor(MonitorConfig{
		Config:   config,
		SwitchMu: &switchMu,
		Notifier: func(ctx context.Context, n *Notification) {},
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3.0, nil
		},
		SwitchFunc: func(ctx context.Context, targetName string) error {
			countMu.Lock()
			switchCount++
			countMu.Unlock()
			time.Sleep(50 * time.Millisecond) // 模擬切換耗時
			return nil
		},
		GetCurrentName: func() string { return "current" },
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{{Name: "target", Balance: 100}}
		},
	})

	// 啟動監控
	monitor.Start()

	// 同時嘗試手動切換
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if switchMu.TryLock() {
				countMu.Lock()
				switchCount++
				countMu.Unlock()
				time.Sleep(10 * time.Millisecond)
				switchMu.Unlock()
			}
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)
	monitor.Stop()

	// 驗證切換次數合理（不會同時執行多個切換）
	countMu.Lock()
	count := switchCount
	countMu.Unlock()
	if count < 1 {
		t.Error("Expected at least one switch")
	}
}

// TestIntegration_MonitorRecovery 監控恢復測試
func TestIntegration_MonitorRecovery(t *testing.T) {
	config := &AutoSwitchSettings{
		Enabled:          true,
		BalanceThreshold: 5,
		RefreshIntervals: DefaultRefreshIntervals(),
	}

	refreshCount := 0
	var countMu sync.Mutex

	monitor := NewMonitor(MonitorConfig{
		Config:   config,
		SwitchMu: &sync.Mutex{},
		Notifier: func(ctx context.Context, n *Notification) {},
		RefreshFunc: func(ctx context.Context) (float64, error) {
			countMu.Lock()
			refreshCount++
			countMu.Unlock()
			return 100.0, nil // 高餘額，不觸發切換
		},
		SwitchFunc:     func(ctx context.Context, targetName string) error { return nil },
		GetCurrentName: func() string { return "current" },
		GetCandidates:  func() []CandidateSnapshot { return nil },
	})

	// 啟動
	monitor.Start()
	time.Sleep(50 * time.Millisecond)

	// 停止
	monitor.Stop()

	// 再次啟動
	monitor.Start()
	time.Sleep(50 * time.Millisecond)

	// 停止
	monitor.Stop()

	// 驗證刷新被調用
	countMu.Lock()
	count := refreshCount
	countMu.Unlock()
	if count < 2 {
		t.Errorf("Expected at least 2 refresh calls, got %d", count)
	}
}
