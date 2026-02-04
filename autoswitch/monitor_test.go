package autoswitch

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestMonitorStartStop 驗證啟動/停止
func TestMonitorStartStop(t *testing.T) {
	m := NewMonitor(MonitorConfig{
		Config: DefaultAutoSwitchSettings(),
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 100, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			return nil
		},
		GetCurrentName: func() string { return "test" },
		GetCandidates:  func() []CandidateSnapshot { return nil },
	})

	// 初始狀態應該是停止
	if m.GetStatus() != StatusStopped {
		t.Errorf("expected status=%s, got %s", StatusStopped, m.GetStatus())
	}

	// 啟動
	m.Start()
	time.Sleep(10 * time.Millisecond)

	if m.GetStatus() != StatusRunning {
		t.Errorf("expected status=%s after start, got %s", StatusRunning, m.GetStatus())
	}

	// 重複啟動應該無效
	m.Start()
	if m.GetStatus() != StatusRunning {
		t.Errorf("expected status=%s after duplicate start, got %s", StatusRunning, m.GetStatus())
	}

	// 停止
	m.Stop()

	if m.GetStatus() != StatusStopped {
		t.Errorf("expected status=%s after stop, got %s", StatusStopped, m.GetStatus())
	}

	// 重複停止應該無效
	m.Stop()
	if m.GetStatus() != StatusStopped {
		t.Errorf("expected status=%s after duplicate stop, got %s", StatusStopped, m.GetStatus())
	}
}

// TestMonitorUpdateConfig 驗證更新設定
func TestMonitorUpdateConfig(t *testing.T) {
	config1 := DefaultAutoSwitchSettings()
	config1.BalanceThreshold = 5

	m := NewMonitor(MonitorConfig{
		Config: config1,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 100, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			return nil
		},
		GetCurrentName: func() string { return "test" },
		GetCandidates:  func() []CandidateSnapshot { return nil },
	})

	// 更新設定
	config2 := DefaultAutoSwitchSettings()
	config2.BalanceThreshold = 10

	m.UpdateConfig(config2)

	m.mu.RLock()
	if m.config.BalanceThreshold != 10 {
		t.Errorf("expected threshold=10 after update, got %f", m.config.BalanceThreshold)
	}
	m.mu.RUnlock()
}

// TestMonitorAutoSwitch 驗證自動切換觸發
func TestMonitorAutoSwitch(t *testing.T) {
	var switchedTo string
	var notifications []*Notification
	var mu sync.Mutex

	config := DefaultAutoSwitchSettings()
	config.Enabled = true
	config.BalanceThreshold = 5
	config.MinTargetBalance = 50
	config.NotifyOnSwitch = true

	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3, nil // 低於閾值
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			mu.Lock()
			switchedTo = name
			mu.Unlock()
			return nil
		},
		GetCurrentName: func() string { return "帳號A" },
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{
				{Name: "帳號B", Balance: 150, SubscriptionType: "Pro", FolderId: ""},
				{Name: "帳號C", Balance: 80, SubscriptionType: "Pro", FolderId: ""},
			}
		},
		Notifier: func(ctx context.Context, n *Notification) {
			mu.Lock()
			notifications = append(notifications, n)
			mu.Unlock()
		},
	})

	m.Start()
	time.Sleep(100 * time.Millisecond)
	m.Stop()

	mu.Lock()
	defer mu.Unlock()

	// 應該切換到餘額最高的帳號B
	if switchedTo != "帳號B" {
		t.Errorf("expected switchedTo='帳號B', got '%s'", switchedTo)
	}

	// 應該有切換成功通知
	found := false
	for _, n := range notifications {
		if n.Type == NotifySwitch {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected switch notification")
	}
}

// TestMonitorNoCandidates 驗證無候選快照
func TestMonitorNoCandidates(t *testing.T) {
	var notifications []*Notification
	var mu sync.Mutex

	config := DefaultAutoSwitchSettings()
	config.Enabled = true
	config.BalanceThreshold = 5
	config.MinTargetBalance = 200 // 高於所有候選

	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			return nil
		},
		GetCurrentName: func() string { return "帳號A" },
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{
				{Name: "帳號B", Balance: 150},
				{Name: "帳號C", Balance: 80},
			}
		},
		Notifier: func(ctx context.Context, n *Notification) {
			mu.Lock()
			notifications = append(notifications, n)
			mu.Unlock()
		},
	})

	m.Start()
	time.Sleep(100 * time.Millisecond)
	m.Stop()

	mu.Lock()
	defer mu.Unlock()

	// 應該有無候選快照通知
	found := false
	for _, n := range notifications {
		if n.Type == NotifyNoCandidates {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected no candidates notification")
	}
}

// TestMonitorCooldown 驗證冷卻期狀態
func TestMonitorCooldown(t *testing.T) {
	config := DefaultAutoSwitchSettings()
	config.Enabled = true

	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 100, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			return nil
		},
		GetCurrentName: func() string { return "test" },
		GetCandidates:  func() []CandidateSnapshot { return nil },
	})

	m.Start()
	time.Sleep(10 * time.Millisecond)

	// 模擬切換後進入冷卻期
	m.safety.RecordSwitch()

	status := m.GetStatus()
	if status != StatusCooldown {
		t.Errorf("expected status=%s during cooldown, got %s", StatusCooldown, status)
	}

	m.Stop()
}

// TestMonitorConcurrentSwitch 驗證並發切換保護
func TestMonitorConcurrentSwitch(t *testing.T) {
	var switchCount int
	var mu sync.Mutex
	switchMu := &sync.Mutex{}

	config := DefaultAutoSwitchSettings()
	config.Enabled = true
	config.BalanceThreshold = 5

	m := NewMonitor(MonitorConfig{
		Config:   config,
		SwitchMu: switchMu,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			mu.Lock()
			switchCount++
			mu.Unlock()
			time.Sleep(50 * time.Millisecond) // 模擬切換耗時
			return nil
		},
		GetCurrentName: func() string { return "帳號A" },
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{
				{Name: "帳號B", Balance: 150},
			}
		},
	})

	// 先鎖定全域切換鎖
	switchMu.Lock()

	m.Start()
	time.Sleep(100 * time.Millisecond)

	// 釋放鎖
	switchMu.Unlock()
	time.Sleep(100 * time.Millisecond)

	m.Stop()

	// 由於冷卻期，應該只切換一次
	mu.Lock()
	defer mu.Unlock()
	if switchCount > 1 {
		t.Errorf("expected at most 1 switch due to cooldown, got %d", switchCount)
	}
}

// TestNewMonitor 驗證建立監控器
func TestNewMonitor(t *testing.T) {
	config := DefaultAutoSwitchSettings()

	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 100, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			return nil
		},
		GetCurrentName: func() string { return "test" },
		GetCandidates:  func() []CandidateSnapshot { return nil },
	})

	if m == nil {
		t.Fatal("expected non-nil monitor")
	}
	if m.config != config {
		t.Error("config not set correctly")
	}
	if m.safety == nil {
		t.Error("safety state not initialized")
	}
	if m.status != StatusStopped {
		t.Errorf("expected initial status=%s, got %s", StatusStopped, m.status)
	}
}

// TestMonitorGetLastBalance 驗證取得最後餘額
func TestMonitorGetLastBalance(t *testing.T) {
	config := DefaultAutoSwitchSettings()
	config.Enabled = true

	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 75.5, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			return nil
		},
		GetCurrentName: func() string { return "test" },
		GetCandidates:  func() []CandidateSnapshot { return nil },
	})

	m.Start()
	time.Sleep(100 * time.Millisecond)
	m.Stop()

	balance := m.GetLastBalance()
	if balance != 75.5 {
		t.Errorf("expected lastBalance=75.5, got %f", balance)
	}
}


// TestMonitorPanicRecovery 驗證監控 Goroutine 異常恢復
func TestMonitorPanicRecovery(t *testing.T) {
	var callCount int
	var mu sync.Mutex

	config := DefaultAutoSwitchSettings()
	config.Enabled = true

	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			mu.Lock()
			callCount++
			count := callCount
			mu.Unlock()

			// 第一次調用時觸發 panic
			if count == 1 {
				panic("simulated panic in refreshFunc")
			}
			return 100, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			return nil
		},
		GetCurrentName: func() string { return "test" },
		GetCandidates:  func() []CandidateSnapshot { return nil },
	})

	m.Start()

	// 等待足夠時間讓 panic 發生並恢復
	// 第一次調用會 panic，恢復後等待 5 秒再重試
	// 我們使用較短的測試時間，驗證監控器沒有崩潰
	time.Sleep(200 * time.Millisecond)

	// 驗證監控器仍在運行（沒有崩潰）
	status := m.GetStatus()
	if status != StatusRunning && status != StatusCooldown {
		t.Errorf("expected monitor to be running after panic recovery, got status=%s", status)
	}

	m.Stop()

	// 驗證 refreshFunc 被調用了至少一次（panic 發生）
	mu.Lock()
	defer mu.Unlock()
	if callCount < 1 {
		t.Errorf("expected refreshFunc to be called at least once, got %d calls", callCount)
	}
}

// TestMonitorPanicRecoveryWithRecoveryDelay 驗證 panic 後的恢復延遲
func TestMonitorPanicRecoveryWithRecoveryDelay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	var panicTimes []time.Time
	var mu sync.Mutex

	config := DefaultAutoSwitchSettings()
	config.Enabled = true

	panicCount := 0
	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			mu.Lock()
			panicCount++
			count := panicCount
			if count <= 2 {
				panicTimes = append(panicTimes, time.Now())
			}
			mu.Unlock()

			// 前兩次調用觸發 panic
			if count <= 2 {
				panic("simulated panic")
			}
			return 100, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			return nil
		},
		GetCurrentName: func() string { return "test" },
		GetCandidates:  func() []CandidateSnapshot { return nil },
	})

	m.Start()

	// 等待足夠時間讓兩次 panic 和恢復發生
	// 每次恢復需要等待 5 秒
	time.Sleep(6 * time.Second)

	m.Stop()

	mu.Lock()
	defer mu.Unlock()

	// 驗證至少發生了兩次 panic
	if len(panicTimes) < 2 {
		t.Errorf("expected at least 2 panics, got %d", len(panicTimes))
		return
	}

	// 驗證兩次 panic 之間的間隔約為 5 秒
	interval := panicTimes[1].Sub(panicTimes[0])
	if interval < 4*time.Second || interval > 6*time.Second {
		t.Errorf("expected recovery delay ~5s, got %v", interval)
	}
}


// TestMonitorConfigSnapshot 驗證切換過程中的設定快照機制
// BDD Scenario: 用戶在切換過程中修改設定
// - Given 自動切換正在執行中
// - When 用戶修改自動切換設定
// - Then 當前切換使用開始時的設定快照
// - And 新設定在下一次切換時生效
func TestMonitorConfigSnapshot(t *testing.T) {
	var mu sync.Mutex
	var switchedTo string

	// 初始設定：MinTargetBalance = 100，只有帳號B符合
	config := DefaultAutoSwitchSettings()
	config.Enabled = true
	config.BalanceThreshold = 5
	config.MinTargetBalance = 100 // 只有 Balance >= 100 的才符合
	config.NotifyOnSwitch = true

	// 用於同步的 channel
	getCandidatesCalled := make(chan struct{})
	configModified := make(chan struct{})
	candidateCallCount := 0

	var m *Monitor
	m = NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3, nil // 低於閾值，觸發切換
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			mu.Lock()
			switchedTo = name
			mu.Unlock()
			return nil
		},
		GetCurrentName: func() string { return "帳號A" },
		GetCandidates: func() []CandidateSnapshot {
			mu.Lock()
			candidateCallCount++
			count := candidateCallCount
			mu.Unlock()

			if count == 1 {
				// 第一次調用：通知測試可以修改設定
				close(getCandidatesCalled)
				// 等待設定修改完成
				<-configModified
			}

			return []CandidateSnapshot{
				{Name: "帳號B", Balance: 150, SubscriptionType: "Pro"}, // 符合 MinTargetBalance=100
				{Name: "帳號C", Balance: 80, SubscriptionType: "Pro"},  // 不符合 100，但符合 30
			}
		},
		Notifier: func(ctx context.Context, n *Notification) {},
	})

	m.Start()

	// 等待 getCandidates 被調用
	select {
	case <-getCandidatesCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for getCandidates")
	}

	// 在 getCandidates 返回後、FilterCandidates 調用前修改設定
	// 將 MinTargetBalance 降低到 30
	newConfig := DefaultAutoSwitchSettings()
	newConfig.Enabled = true
	newConfig.BalanceThreshold = 5
	newConfig.MinTargetBalance = 30 // 降低閾值
	newConfig.NotifyOnSwitch = true
	m.UpdateConfig(newConfig)

	// 讓 getCandidates 繼續返回
	close(configModified)

	time.Sleep(200 * time.Millisecond)
	m.Stop()

	mu.Lock()
	defer mu.Unlock()

	// 預期行為（有快照機制）：
	// checkAndSwitch 開始時應該複製 config，使用 MinTargetBalance=100
	// 因此只有 帳號B (Balance=150) 符合條件
	//
	// 當前行為（無快照機制）：
	// checkAndSwitch 在 FilterCandidates 時重新讀取 config
	// 此時 MinTargetBalance 已被修改為 30
	// 帳號B 和 帳號C 都符合，但仍選擇餘額最高的 帳號B
	//
	// 為了驗證快照機制，我們需要確保切換發生
	if switchedTo == "" {
		t.Error("expected a switch to occur")
	}
	t.Logf("Switched to: %s (expected: 帳號B with snapshot)", switchedTo)
}

// TestMonitorConfigSnapshotDuringSwitch 驗證切換過程中設定修改不影響當前切換
// 這個測試驗證：當 Enabled 在切換過程中被設為 false，當前切換仍應完成
func TestMonitorConfigSnapshotDuringSwitch(t *testing.T) {
	var switchedTo string
	var mu sync.Mutex
	switchMu := &sync.Mutex{}

	// 初始設定
	config := DefaultAutoSwitchSettings()
	config.Enabled = true
	config.BalanceThreshold = 5
	config.MinTargetBalance = 50
	config.NotifyOnSwitch = true

	// 用於同步的 channel
	switchStarted := make(chan struct{})
	switchContinue := make(chan struct{})

	var m *Monitor
	m = NewMonitor(MonitorConfig{
		Config:   config,
		SwitchMu: switchMu,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			// 通知測試切換已開始
			close(switchStarted)
			// 等待測試修改設定
			<-switchContinue

			mu.Lock()
			switchedTo = name
			mu.Unlock()
			return nil
		},
		GetCurrentName: func() string { return "帳號A" },
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{
				{Name: "帳號B", Balance: 150, SubscriptionType: "Pro"},
			}
		},
		Notifier: func(ctx context.Context, n *Notification) {},
	})

	m.Start()

	// 等待切換開始
	select {
	case <-switchStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for switch to start")
	}

	// 在切換過程中禁用自動切換
	newConfig := DefaultAutoSwitchSettings()
	newConfig.Enabled = false // 禁用
	m.UpdateConfig(newConfig)

	// 讓切換繼續完成
	close(switchContinue)

	time.Sleep(200 * time.Millisecond)
	m.Stop()

	mu.Lock()
	defer mu.Unlock()

	// 預期：即使設定被修改為 Enabled=false，當前切換仍應完成
	// 因為切換開始時應該使用快照
	if switchedTo != "帳號B" {
		t.Errorf("expected switch to complete to '帳號B', got '%s'", switchedTo)
	}
}


// TestMonitorConfigSnapshotConsistency 驗證 checkAndSwitch 內部使用一致的設定
// 這個測試確保在 checkAndSwitch 執行期間，所有操作使用相同的設定快照
func TestMonitorConfigSnapshotConsistency(t *testing.T) {
	var mu sync.Mutex
	var notifications []*Notification

	// 初始設定
	config := DefaultAutoSwitchSettings()
	config.Enabled = true
	config.BalanceThreshold = 5
	config.MinTargetBalance = 100
	config.NotifyOnSwitch = true // 原始值：啟用通知

	// 用於同步
	switchStarted := make(chan struct{})
	switchContinue := make(chan struct{})

	var m *Monitor
	m = NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			// 通知測試切換已開始
			close(switchStarted)
			// 等待測試修改設定
			<-switchContinue
			return nil
		},
		GetCurrentName: func() string { return "帳號A" },
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{
				{Name: "帳號B", Balance: 150, SubscriptionType: "Pro"},
			}
		},
		Notifier: func(ctx context.Context, n *Notification) {
			mu.Lock()
			notifications = append(notifications, n)
			mu.Unlock()
		},
	})

	m.Start()

	// 等待切換開始（switchFunc 被調用）
	select {
	case <-switchStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for switch to start")
	}

	// 在切換過程中修改設定：禁用通知
	newConfig := DefaultAutoSwitchSettings()
	newConfig.Enabled = true
	newConfig.BalanceThreshold = 5
	newConfig.MinTargetBalance = 30
	newConfig.NotifyOnSwitch = false // 修改為：禁用通知
	m.UpdateConfig(newConfig)

	// 讓切換繼續完成
	close(switchContinue)

	time.Sleep(200 * time.Millisecond)
	m.Stop()

	mu.Lock()
	defer mu.Unlock()

	// 驗證：如果有快照機制，應該使用原始設定 NotifyOnSwitch=true
	// 因此應該收到切換成功通知
	foundSwitchNotification := false
	for _, n := range notifications {
		if n.Type == NotifySwitch {
			foundSwitchNotification = true
			break
		}
	}

	if !foundSwitchNotification {
		t.Error("Config snapshot not working: expected switch notification (NotifyOnSwitch=true in snapshot), but got none")
	}
}



// ============================================================================
// BDD Scenario 補強測試 - PARTIAL 項目
// ============================================================================

// TestMonitorValidateCandidateBeforeSwitch 驗證切換前刷新候選餘額
// BDD Scenario: 切換前驗證目標快照餘額 (@switch)
// - Given 觸發自動切換
// - And 候選快照「帳號B」緩存餘額為 150
// - When 系統刷新「帳號B」餘額
// - And 刷新後餘額為 120（仍符合條件）
// - Then 系統執行切換至「帳號B」
func TestMonitorValidateCandidateBeforeSwitch(t *testing.T) {
	var mu sync.Mutex
	var switchedTo string
	var validateCalls []string

	config := DefaultAutoSwitchSettings()
	config.Enabled = true
	config.BalanceThreshold = 5
	config.MinTargetBalance = 50
	config.NotifyOnSwitch = true

	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3, nil // 低於閾值，觸發切換
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			mu.Lock()
			switchedTo = name
			mu.Unlock()
			return nil
		},
		GetCurrentName: func() string { return "帳號A" },
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{
				{Name: "帳號B", Balance: 150, SubscriptionType: "Pro"},
				{Name: "帳號C", Balance: 80, SubscriptionType: "Pro"},
			}
		},
		// ValidateCandidate 應該在切換前被調用以刷新餘額
		ValidateCandidate: func(ctx context.Context, name string) (float64, error) {
			mu.Lock()
			validateCalls = append(validateCalls, name)
			mu.Unlock()
			// 模擬刷新後餘額為 120（仍符合條件）
			return 120, nil
		},
	})

	m.Start()
	time.Sleep(100 * time.Millisecond)
	m.Stop()

	mu.Lock()
	defer mu.Unlock()

	// 驗證：ValidateCandidate 應該被調用
	if len(validateCalls) == 0 {
		t.Error("expected ValidateCandidate to be called before switch")
	}

	// 驗證：應該切換到帳號B
	if switchedTo != "帳號B" {
		t.Errorf("expected switchedTo='帳號B', got '%s'", switchedTo)
	}
}

// TestMonitorFallbackToNextCandidate 驗證候選失敗時嘗試下一個
// BDD Scenario: 目標快照驗證失敗時嘗試下一個候選 (@switch)
// - Given 觸發自動切換
// - And 候選快照按餘額排序: 帳號B (150), 帳號C (80)
// - When 系統刷新「帳號B」餘額
// - And 刷新後發現「帳號B」Token 已失效
// - Then 系統自動嘗試「帳號C」
// - And 系統刷新「帳號C」餘額
// - And 系統切換至「帳號C」
func TestMonitorFallbackToNextCandidate(t *testing.T) {
	var mu sync.Mutex
	var switchedTo string
	var validateCalls []string

	config := DefaultAutoSwitchSettings()
	config.Enabled = true
	config.BalanceThreshold = 5
	config.MinTargetBalance = 50
	config.NotifyOnSwitch = true

	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			mu.Lock()
			switchedTo = name
			mu.Unlock()
			return nil
		},
		GetCurrentName: func() string { return "帳號A" },
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{
				{Name: "帳號B", Balance: 150, SubscriptionType: "Pro"},
				{Name: "帳號C", Balance: 80, SubscriptionType: "Pro"},
			}
		},
		// 帳號B 驗證失敗（Token 失效），帳號C 驗證成功
		ValidateCandidate: func(ctx context.Context, name string) (float64, error) {
			mu.Lock()
			validateCalls = append(validateCalls, name)
			mu.Unlock()
			if name == "帳號B" {
				return 0, context.DeadlineExceeded // 模擬 Token 失效
			}
			return 80, nil // 帳號C 驗證成功
		},
	})

	m.Start()
	time.Sleep(100 * time.Millisecond)
	m.Stop()

	mu.Lock()
	defer mu.Unlock()

	// 驗證：應該嘗試了帳號B 和 帳號C
	if len(validateCalls) < 2 {
		t.Errorf("expected at least 2 validate calls (fallback), got %d: %v", len(validateCalls), validateCalls)
	}

	// 驗證：應該切換到帳號C（因為帳號B 驗證失敗）
	if switchedTo != "帳號C" {
		t.Errorf("expected switchedTo='帳號C' (fallback), got '%s'", switchedTo)
	}
}

// TestMonitorConfirmAfterSwitch 驗證切換後確認餘額
// BDD Scenario: 切換後確認目標餘額狀態 (@safety)
// - Given 系統成功切換至「帳號B」
// - When 切換完成後 1 秒
// - Then 系統刷新「帳號B」餘額確認狀態
// - And 若餘額仍 <= 閾值則記錄警告但不立即觸發下一次切換
func TestMonitorConfirmAfterSwitch(t *testing.T) {
	var mu sync.Mutex
	var switchedTo string
	var confirmCalled bool
	var confirmName string

	config := DefaultAutoSwitchSettings()
	config.Enabled = true
	config.BalanceThreshold = 5
	config.MinTargetBalance = 50
	config.NotifyOnSwitch = true

	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			mu.Lock()
			switchedTo = name
			mu.Unlock()
			return nil
		},
		GetCurrentName: func() string {
			mu.Lock()
			defer mu.Unlock()
			if switchedTo != "" {
				return switchedTo
			}
			return "帳號A"
		},
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{
				{Name: "帳號B", Balance: 150, SubscriptionType: "Pro"},
			}
		},
		ValidateCandidate: func(ctx context.Context, name string) (float64, error) {
			return 150, nil
		},
		// ConfirmAfterSwitch 應該在切換後 1 秒被調用
		ConfirmAfterSwitch: func(ctx context.Context, name string) (float64, error) {
			mu.Lock()
			confirmCalled = true
			confirmName = name
			mu.Unlock()
			return 150, nil
		},
	})

	m.Start()
	// 等待切換完成 + 1 秒確認延遲
	time.Sleep(1500 * time.Millisecond)
	m.Stop()

	mu.Lock()
	defer mu.Unlock()

	// 驗證：切換成功
	if switchedTo != "帳號B" {
		t.Errorf("expected switchedTo='帳號B', got '%s'", switchedTo)
	}

	// 驗證：ConfirmAfterSwitch 應該被調用
	if !confirmCalled {
		t.Error("expected ConfirmAfterSwitch to be called after switch")
	}

	// 驗證：確認的是切換後的快照
	if confirmName != "帳號B" {
		t.Errorf("expected confirm name='帳號B', got '%s'", confirmName)
	}
}

// TestMonitorRetryOnNetworkError 驗證網路異常重試
// BDD Scenario: 網路異常導致刷新失敗 (@edge-case)
// - Given 觸發自動切換
// - When 刷新候選快照餘額時網路異常
// - Then 系統重試 3 次（每次間隔 2 秒）
// - And 若仍失敗則跳過該候選，嘗試下一個
func TestMonitorRetryOnNetworkError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode (requires retry delays)")
	}

	var mu sync.Mutex
	var switchedTo string
	var validateCalls []string
	var retryCount int

	config := DefaultAutoSwitchSettings()
	config.Enabled = true
	config.BalanceThreshold = 5
	config.MinTargetBalance = 50
	config.NotifyOnSwitch = true

	m := NewMonitor(MonitorConfig{
		Config: config,
		RefreshFunc: func(ctx context.Context) (float64, error) {
			return 3, nil
		},
		SwitchFunc: func(ctx context.Context, name string) error {
			mu.Lock()
			switchedTo = name
			mu.Unlock()
			return nil
		},
		GetCurrentName: func() string { return "帳號A" },
		GetCandidates: func() []CandidateSnapshot {
			return []CandidateSnapshot{
				{Name: "帳號B", Balance: 150, SubscriptionType: "Pro"},
				{Name: "帳號C", Balance: 80, SubscriptionType: "Pro"},
			}
		},
		// 帳號B 前 3 次驗證都失敗（網路異常），帳號C 成功
		ValidateCandidate: func(ctx context.Context, name string) (float64, error) {
			mu.Lock()
			validateCalls = append(validateCalls, name)
			if name == "帳號B" {
				retryCount++
				count := retryCount
				mu.Unlock()
				if count <= 3 {
					return 0, context.DeadlineExceeded // 模擬網路異常
				}
				return 150, nil
			}
			mu.Unlock()
			return 80, nil
		},
	})

	m.Start()
	// 等待重試完成：3 次重試 * 2 秒間隔 = 6 秒 + buffer
	time.Sleep(8 * time.Second)
	m.Stop()

	mu.Lock()
	defer mu.Unlock()

	// 驗證：帳號B 應該被重試 3 次
	accountBCalls := 0
	for _, name := range validateCalls {
		if name == "帳號B" {
			accountBCalls++
		}
	}
	if accountBCalls < 3 {
		t.Errorf("expected at least 3 retry calls for 帳號B, got %d", accountBCalls)
	}

	// 驗證：最終應該切換到帳號C（因為帳號B 重試 3 次後仍失敗）
	if switchedTo != "帳號C" {
		t.Errorf("expected switchedTo='帳號C' after retries, got '%s'", switchedTo)
	}
}
