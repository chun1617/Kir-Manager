package autoswitch

import (
	"context"
	"sync"
	"time"
)

// RefreshFunc 刷新餘額回調函數類型
// 返回當前餘額和錯誤
type RefreshFunc func(ctx context.Context) (float64, error)

// SwitchFunc 切換快照回調函數類型
// 參數：目標快照名稱
// 返回：錯誤
type SwitchFunc func(ctx context.Context, targetName string) error

// GetCurrentNameFunc 取得當前快照名稱回調函數類型
type GetCurrentNameFunc func() string

// GetCandidatesFunc 取得候選快照回調函數類型
type GetCandidatesFunc func() []CandidateSnapshot

// ValidateCandidateFunc 驗證候選快照回調函數類型
// 在切換前刷新候選快照餘額以驗證可用性
// 參數：候選快照名稱
// 返回：刷新後的餘額和錯誤
type ValidateCandidateFunc func(ctx context.Context, candidateName string) (float64, error)

// ConfirmAfterSwitchFunc 切換後確認回調函數類型
// 在切換完成後刷新確認目標餘額狀態
// 參數：目標快照名稱
// 返回：確認後的餘額和錯誤
type ConfirmAfterSwitchFunc func(ctx context.Context, targetName string) (float64, error)

// MonitorStatus 監控狀態
type MonitorStatus string

const (
	StatusStopped  MonitorStatus = "stopped"
	StatusRunning  MonitorStatus = "running"
	StatusCooldown MonitorStatus = "cooldown"
)

// 重試相關常數
const (
	// ValidateRetryCount 驗證候選快照時的重試次數
	ValidateRetryCount = 3
	// ValidateRetryInterval 驗證重試間隔
	ValidateRetryInterval = 2 * time.Second
	// ConfirmAfterSwitchDelay 切換後確認延遲
	ConfirmAfterSwitchDelay = 1 * time.Second
)

// Monitor 自動切換監控器
type Monitor struct {
	ctx                context.Context
	cancel             context.CancelFunc
	config             *AutoSwitchSettings
	safety             *SafetyState
	switchMu           *sync.Mutex // 全域切換鎖（與手動切換共用）
	notifier           NotifyFunc
	refreshFunc        RefreshFunc
	switchFunc         SwitchFunc
	getCurrentName     GetCurrentNameFunc
	getCandidates      GetCandidatesFunc
	validateCandidate  ValidateCandidateFunc
	confirmAfterSwitch ConfirmAfterSwitchFunc
	mu                 sync.RWMutex
	status             MonitorStatus
	lastBalance        float64
	wg                 sync.WaitGroup
}

// MonitorConfig 監控器配置
type MonitorConfig struct {
	Config             *AutoSwitchSettings
	SwitchMu           *sync.Mutex
	Notifier           NotifyFunc
	RefreshFunc        RefreshFunc
	SwitchFunc         SwitchFunc
	GetCurrentName     GetCurrentNameFunc
	GetCandidates      GetCandidatesFunc
	ValidateCandidate  ValidateCandidateFunc  // 切換前驗證候選快照餘額
	ConfirmAfterSwitch ConfirmAfterSwitchFunc // 切換後確認目標餘額狀態
}

// NewMonitor 建立新的監控器
func NewMonitor(cfg MonitorConfig) *Monitor {
	return &Monitor{
		config:             cfg.Config,
		safety:             NewSafetyState(),
		switchMu:           cfg.SwitchMu,
		notifier:           cfg.Notifier,
		refreshFunc:        cfg.RefreshFunc,
		switchFunc:         cfg.SwitchFunc,
		getCurrentName:     cfg.GetCurrentName,
		getCandidates:      cfg.GetCandidates,
		validateCandidate:  cfg.ValidateCandidate,
		confirmAfterSwitch: cfg.ConfirmAfterSwitch,
		status:             StatusStopped,
	}
}

// Start 啟動監控
func (m *Monitor) Start() {
	m.mu.Lock()
	if m.status == StatusRunning {
		m.mu.Unlock()
		return
	}

	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.status = StatusRunning
	m.mu.Unlock()

	m.wg.Add(1)
	go m.monitorLoop()
}

// Stop 停止監控
func (m *Monitor) Stop() {
	m.mu.Lock()
	if m.status == StatusStopped {
		m.mu.Unlock()
		return
	}

	if m.cancel != nil {
		m.cancel()
	}
	m.status = StatusStopped
	m.mu.Unlock()

	m.wg.Wait()
}

// UpdateConfig 更新設定
func (m *Monitor) UpdateConfig(config *AutoSwitchSettings) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// GetStatus 取得監控狀態
func (m *Monitor) GetStatus() MonitorStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 檢查是否在冷卻期
	if m.status == StatusRunning && m.safety.GetCooldownRemaining() > 0 {
		return StatusCooldown
	}

	return m.status
}

// GetLastBalance 取得最後一次刷新的餘額
func (m *Monitor) GetLastBalance() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastBalance
}

// PanicRecoveryDelay panic 恢復後的等待時間
const PanicRecoveryDelay = 5 * time.Second

// monitorLoop 監控主循環
func (m *Monitor) monitorLoop() {
	defer m.wg.Done()

	for {
		m.mu.RLock()
		ctx := m.ctx
		m.mu.RUnlock()

		// 檢查是否已停止
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 執行單次迭代，帶 panic recovery
		recovered := m.runIterationWithRecovery()

		if recovered {
			// Panic 恢復後等待 5 秒再重試
			select {
			case <-ctx.Done():
				return
			case <-time.After(PanicRecoveryDelay):
				continue
			}
		}
	}
}

// runIterationWithRecovery 執行單次迭代，帶 panic recovery
// 返回 true 表示發生了 panic 並已恢復
func (m *Monitor) runIterationWithRecovery() (recovered bool) {
	defer func() {
		if r := recover(); r != nil {
			// 記錄錯誤日誌
			// log.Printf("Monitor panic recovered: %v", r)
			_ = r // 暫時忽略，避免 unused variable 警告
			recovered = true
		}
	}()

	m.monitorIteration()
	return false
}

// monitorIteration 單次監控迭代
func (m *Monitor) monitorIteration() {
	m.mu.RLock()
	config := m.config
	ctx := m.ctx
	m.mu.RUnlock()

	if config == nil || !config.Enabled {
		// 設定為空或未啟用，等待後重試
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			return
		}
	}

	// 刷新餘額
	balance, err := m.refreshFunc(ctx)
	if err != nil {
		// 刷新失敗，等待後重試
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
			return
		}
	}

	m.mu.Lock()
	m.lastBalance = balance
	m.mu.Unlock()

	// 檢查是否需要切換
	if balance <= config.BalanceThreshold {
		m.checkAndSwitch(ctx, balance)
	} else if balance <= config.BalanceThreshold*2 && config.NotifyOnLowBalance {
		// 餘額接近閾值，發送預警
		if m.notifier != nil {
			m.notifier(ctx, NewLowBalanceNotification(balance, config.BalanceThreshold))
		}
	}

	// 計算下一次刷新間隔
	interval := GetRefreshInterval(config.RefreshIntervals, balance)

	select {
	case <-ctx.Done():
		return
	case <-time.After(interval):
		return
	}
}

// checkAndSwitch 檢查並執行切換
func (m *Monitor) checkAndSwitch(ctx context.Context, currentBalance float64) {
	// 在切換開始時複製設定快照，確保整個切換過程使用一致的設定
	m.mu.RLock()
	configSnapshot := m.config.Clone()
	m.mu.RUnlock()

	// 檢查安全狀態
	canSwitch, reason := m.safety.CanSwitch()
	if !canSwitch {
		// 發送通知
		if m.notifier != nil {
			if m.safety.GetSwitchCount() >= MaxSwitchPerHour {
				m.notifier(ctx, NewMaxSwitchNotification())
			} else {
				remaining := int(m.safety.GetCooldownRemaining().Seconds())
				m.notifier(ctx, NewCooldownNotification(remaining))
			}
		}
		_ = reason // 已在通知中使用
		return
	}

	// 取得候選快照
	candidates := m.getCandidates()
	if len(candidates) == 0 {
		if m.notifier != nil {
			m.notifier(ctx, NewNoCandidatesNotification())
		}
		return
	}

	// 篩選候選 - 使用設定快照
	currentName := m.getCurrentName()
	filtered := FilterCandidates(configSnapshot, currentName, candidates)
	if len(filtered) == 0 {
		if m.notifier != nil {
			m.notifier(ctx, NewNoCandidatesNotification())
		}
		return
	}

	// 嘗試取得全域切換鎖
	if m.switchMu != nil {
		if !m.switchMu.TryLock() {
			// 正在切換中，跳過
			return
		}
		defer m.switchMu.Unlock()
	}

	// 按餘額排序候選（SelectBestCandidate 已經做了，但我們需要遍歷所有候選做 fallback）
	// 嘗試每個候選，直到成功或全部失敗
	for _, candidate := range filtered {
		// 驗證候選快照餘額（帶重試）
		if m.validateCandidate != nil {
			validatedBalance, err := m.validateCandidateWithRetry(ctx, candidate.Name)
			if err != nil {
				// 驗證失敗，嘗試下一個候選
				continue
			}
			// 檢查驗證後的餘額是否仍符合條件
			if validatedBalance < configSnapshot.MinTargetBalance {
				continue
			}
		}

		// 執行切換
		err := m.switchFunc(ctx, candidate.Name)
		if err != nil {
			if m.notifier != nil {
				m.notifier(ctx, NewSwitchFailNotification(err.Error()))
			}
			// 切換失敗，嘗試下一個候選
			continue
		}

		// 記錄切換
		m.safety.RecordSwitch()

		// 發送成功通知 - 使用設定快照
		if m.notifier != nil && configSnapshot.NotifyOnSwitch {
			m.notifier(ctx, NewSwitchNotification(currentName, candidate.Name))
		}

		// 切換後確認（異步執行，不阻塞）
		if m.confirmAfterSwitch != nil {
			m.wg.Add(1)
			go func() {
				defer m.wg.Done()
				m.confirmAfterSwitchAsync(ctx, candidate.Name, configSnapshot.BalanceThreshold)
			}()
		}

		// 切換成功，退出循環
		return
	}

	// 所有候選都失敗
	if m.notifier != nil {
		m.notifier(ctx, NewNoCandidatesNotification())
	}
}

// validateCandidateWithRetry 帶重試的候選驗證
func (m *Monitor) validateCandidateWithRetry(ctx context.Context, candidateName string) (float64, error) {
	var lastErr error
	for i := 0; i < ValidateRetryCount; i++ {
		balance, err := m.validateCandidate(ctx, candidateName)
		if err == nil {
			return balance, nil
		}
		lastErr = err

		// 如果不是最後一次重試，等待後再試
		if i < ValidateRetryCount-1 {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(ValidateRetryInterval):
			}
		}
	}
	return 0, lastErr
}

// confirmAfterSwitchAsync 異步執行切換後確認
func (m *Monitor) confirmAfterSwitchAsync(ctx context.Context, targetName string, threshold float64) {
	// 等待 1 秒後確認
	select {
	case <-ctx.Done():
		return
	case <-time.After(ConfirmAfterSwitchDelay):
	}

	balance, err := m.confirmAfterSwitch(ctx, targetName)
	if err != nil {
		// 確認失敗，記錄警告但不觸發切換
		return
	}

	// 若餘額仍 <= 閾值則記錄警告但不立即觸發下一次切換
	if balance <= threshold {
		// 記錄警告（可以通過 notifier 發送）
		// 但不觸發切換，因為已在冷卻期
	}
}
