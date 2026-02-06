import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fc from 'fast-check'
import { useUsageRefresh } from '../useUsageRefresh'

describe('Feature: app-vue-decoupling, useUsageRefresh composable', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('Property 15: 冷卻期狀態一致性', () => {
    it('isInCooldown 應根據 countdownTimers 正確返回', () => {
      // 使用合法的備份名稱（排除 JavaScript 保留屬性名稱）
      const validBackupName = fc.string({ minLength: 1, maxLength: 50 })
        .filter(name => !['__proto__', 'constructor', 'prototype'].includes(name))
      
      fc.assert(
        fc.property(
          validBackupName,
          fc.integer({ min: 0, max: 120 }),
          (backupName, seconds) => {
            const { countdownTimers, isInCooldown } = useUsageRefresh()
            
            // 設定倒計時值
            countdownTimers.value[backupName] = seconds
            
            // 驗證：isInCooldown 應與 countdownTimers > 0 一致
            const expected = seconds > 0
            return isInCooldown(backupName) === expected
          }
        ),
        { numRuns: 100 }
      )
    })

    it('未設定的備份名稱應返回 false', () => {
      fc.assert(
        fc.property(
          fc.string({ minLength: 1, maxLength: 50 }),
          (backupName) => {
            const { isInCooldown } = useUsageRefresh()
            
            // 未設定任何倒計時
            return isInCooldown(backupName) === false
          }
        ),
        { numRuns: 50 }
      )
    })

    it('isCurrentInCooldown 應根據 countdownCurrentAccount 正確返回', () => {
      fc.assert(
        fc.property(
          fc.integer({ min: 0, max: 120 }),
          (seconds) => {
            const { countdownCurrentAccount, isCurrentInCooldown } = useUsageRefresh()
            
            countdownCurrentAccount.value = seconds
            
            const expected = seconds > 0
            return isCurrentInCooldown() === expected
          }
        ),
        { numRuns: 100 }
      )
    })
  })

  describe('Property 16: Interval 累積防護', () => {
    it('多次調用 startCountdown 不應累積多個 interval', () => {
      fc.assert(
        fc.property(
          fc.string({ minLength: 1, maxLength: 50 }),
          fc.integer({ min: 1, max: 5 }),
          (backupName, callCount) => {
            const { countdownTimers, startCountdown } = useUsageRefresh()
            const initialSeconds = 60
            
            // 多次調用 startCountdown
            for (let i = 0; i < callCount; i++) {
              startCountdown(backupName, initialSeconds)
            }
            
            // 驗證初始值正確
            if (countdownTimers.value[backupName] !== initialSeconds) {
              return false
            }
            
            // 快進 1 秒
            vi.advanceTimersByTime(1000)
            
            // 驗證：只減少 1 秒（而非 callCount 秒）
            // 這證明只有一個 interval 在運行
            return countdownTimers.value[backupName] === initialSeconds - 1
          }
        ),
        { numRuns: 50 }
      )
    })

    it('多次調用 startCurrentCountdown 不應累積多個 interval', () => {
      fc.assert(
        fc.property(
          fc.integer({ min: 1, max: 5 }),
          (callCount) => {
            const { countdownCurrentAccount, startCurrentCountdown } = useUsageRefresh()
            const initialSeconds = 60
            
            // 多次調用 startCurrentCountdown
            for (let i = 0; i < callCount; i++) {
              startCurrentCountdown(initialSeconds)
            }
            
            // 驗證初始值正確
            if (countdownCurrentAccount.value !== initialSeconds) {
              return false
            }
            
            // 快進 1 秒
            vi.advanceTimersByTime(1000)
            
            // 驗證：只減少 1 秒
            return countdownCurrentAccount.value === initialSeconds - 1
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  describe('倒計時基本功能', () => {
    it('startCountdown 應正確遞減倒計時', () => {
      const { countdownTimers, startCountdown } = useUsageRefresh()
      const backupName = 'test-backup'
      const seconds = 5
      
      startCountdown(backupName, seconds)
      
      expect(countdownTimers.value[backupName]).toBe(5)
      
      vi.advanceTimersByTime(1000)
      expect(countdownTimers.value[backupName]).toBe(4)
      
      vi.advanceTimersByTime(1000)
      expect(countdownTimers.value[backupName]).toBe(3)
      
      vi.advanceTimersByTime(3000)
      expect(countdownTimers.value[backupName]).toBe(0)
      
      // 倒計時結束後不應繼續減少
      vi.advanceTimersByTime(1000)
      expect(countdownTimers.value[backupName]).toBe(0)
    })

    it('startCurrentCountdown 應正確遞減倒計時', () => {
      const { countdownCurrentAccount, startCurrentCountdown } = useUsageRefresh()
      const seconds = 5
      
      startCurrentCountdown(seconds)
      
      expect(countdownCurrentAccount.value).toBe(5)
      
      vi.advanceTimersByTime(1000)
      expect(countdownCurrentAccount.value).toBe(4)
      
      vi.advanceTimersByTime(4000)
      expect(countdownCurrentAccount.value).toBe(0)
    })

    it('clearAllCountdowns 應清除所有倒計時', () => {
      const { 
        countdownTimers, 
        countdownCurrentAccount, 
        startCountdown, 
        startCurrentCountdown,
        clearAllCountdowns 
      } = useUsageRefresh()
      
      // 啟動多個倒計時
      startCountdown('backup1', 60)
      startCountdown('backup2', 30)
      startCurrentCountdown(45)
      
      expect(countdownTimers.value['backup1']).toBe(60)
      expect(countdownTimers.value['backup2']).toBe(30)
      expect(countdownCurrentAccount.value).toBe(45)
      
      // 清除所有倒計時
      clearAllCountdowns()
      
      expect(countdownTimers.value['backup1']).toBe(0)
      expect(countdownTimers.value['backup2']).toBe(0)
      expect(countdownCurrentAccount.value).toBe(0)
      
      // 快進確認 interval 已被清除（不會繼續減少到負數）
      vi.advanceTimersByTime(5000)
      expect(countdownTimers.value['backup1']).toBe(0)
      expect(countdownTimers.value['backup2']).toBe(0)
      expect(countdownCurrentAccount.value).toBe(0)
    })
  })

  describe('刷新狀態管理', () => {
    it('refreshingBackup 初始值應為 null', () => {
      const { refreshingBackup } = useUsageRefresh()
      expect(refreshingBackup.value).toBe(null)
    })

    it('refreshingCurrent 初始值應為 false', () => {
      const { refreshingCurrent } = useUsageRefresh()
      expect(refreshingCurrent.value).toBe(false)
    })

    it('countdownTimers 初始值應為空物件', () => {
      const { countdownTimers } = useUsageRefresh()
      expect(countdownTimers.value).toEqual({})
    })

    it('countdownCurrentAccount 初始值應為 0', () => {
      const { countdownCurrentAccount } = useUsageRefresh()
      expect(countdownCurrentAccount.value).toBe(0)
    })
  })

  describe('多個備份的獨立倒計時', () => {
    it('不同備份的倒計時應獨立運行', () => {
      const { countdownTimers, startCountdown, isInCooldown } = useUsageRefresh()
      
      startCountdown('backup1', 10)
      startCountdown('backup2', 5)
      
      expect(countdownTimers.value['backup1']).toBe(10)
      expect(countdownTimers.value['backup2']).toBe(5)
      
      vi.advanceTimersByTime(5000)
      
      expect(countdownTimers.value['backup1']).toBe(5)
      expect(countdownTimers.value['backup2']).toBe(0)
      
      expect(isInCooldown('backup1')).toBe(true)
      expect(isInCooldown('backup2')).toBe(false)
    })
  })

  describe('P0-FIX: Memory Leak - cleanup 函數', () => {
    it('cleanup 函數應該導出並可用', () => {
      const result = useUsageRefresh()
      
      // 驗證 cleanup 函數存在
      expect(result.cleanup).toBeDefined()
      expect(typeof result.cleanup).toBe('function')
    })

    it('cleanup 應該清除所有 interval 並重置狀態', () => {
      const { 
        countdownTimers, 
        countdownCurrentAccount, 
        startCountdown, 
        startCurrentCountdown,
        cleanup 
      } = useUsageRefresh()
      
      // 啟動多個倒計時
      startCountdown('backup1', 60)
      startCountdown('backup2', 30)
      startCurrentCountdown(45)
      
      expect(countdownTimers.value['backup1']).toBe(60)
      expect(countdownTimers.value['backup2']).toBe(30)
      expect(countdownCurrentAccount.value).toBe(45)
      
      // 調用 cleanup
      cleanup()
      
      // 驗證狀態已重置
      expect(countdownTimers.value['backup1']).toBe(0)
      expect(countdownTimers.value['backup2']).toBe(0)
      expect(countdownCurrentAccount.value).toBe(0)
      
      // 快進確認 interval 已被清除（不會繼續減少到負數）
      vi.advanceTimersByTime(5000)
      expect(countdownTimers.value['backup1']).toBe(0)
      expect(countdownTimers.value['backup2']).toBe(0)
      expect(countdownCurrentAccount.value).toBe(0)
    })
  })
})


// ============================================================================
// Phase 2 Task 1.3: useUsageRefresh 擴展功能測試
// ============================================================================

// Mock Wails API for useUsageRefresh tests
const mockRefreshBackupUsage = vi.fn()
const mockGetCurrentUsageInfo = vi.fn()

describe('Phase 2 Task 1.3: useUsageRefresh 擴展功能', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockRefreshBackupUsage.mockResolvedValue({ success: true, message: '' })
    mockGetCurrentUsageInfo.mockResolvedValue({
      subscriptionTitle: 'KIRO PRO',
      usageLimit: 500,
      currentUsage: 100,
      balance: 400,
      isLowBalance: false,
    })
    
    // @ts-ignore - Partial mock for testing
    globalThis.window = {
      go: {
        main: {
          App: {
            RefreshBackupUsage: mockRefreshBackupUsage,
            GetCurrentUsageInfo: mockGetCurrentUsageInfo,
          },
        },
      },
    } as any
  })

  describe('refreshBackupUsageWithUpdate 含本地狀態更新', () => {
    it('應刷新備份用量並調用 onLocalUpdate 回調', async () => {
      const { refreshBackupUsageWithUpdate, refreshingBackup, startCountdown } = useUsageRefresh()
      
      const onLocalUpdate = vi.fn()
      const backupName = 'test-backup'
      
      const result = await refreshBackupUsageWithUpdate(backupName, { onLocalUpdate })
      
      expect(mockRefreshBackupUsage).toHaveBeenCalledWith(backupName)
      expect(result.success).toBe(true)
      expect(onLocalUpdate).toHaveBeenCalledTimes(1)
    })

    it('刷新過程中 refreshingBackup 應設為備份名稱', async () => {
      let resolvePromise: (value: any) => void
      const pendingPromise = new Promise((resolve) => {
        resolvePromise = resolve
      })
      mockRefreshBackupUsage.mockReturnValue(pendingPromise)
      
      const { refreshBackupUsageWithUpdate, refreshingBackup } = useUsageRefresh()
      
      const promise = refreshBackupUsageWithUpdate('test-backup', {})
      
      // 驗證進行中狀態
      expect(refreshingBackup.value).toBe('test-backup')
      
      // 完成操作
      resolvePromise!({ success: true, message: '' })
      await promise
      
      // 驗證狀態恢復
      expect(refreshingBackup.value).toBe(null)
    })

    it('刷新成功後應自動啟動冷卻期倒計時', async () => {
      const { refreshBackupUsageWithUpdate, isInCooldown } = useUsageRefresh()
      
      await refreshBackupUsageWithUpdate('test-backup', {})
      
      // 驗證冷卻期已啟動
      expect(isInCooldown('test-backup')).toBe(true)
    })

    it('刷新失敗時不應啟動冷卻期', async () => {
      mockRefreshBackupUsage.mockRejectedValue(new Error('刷新失敗'))
      
      const { refreshBackupUsageWithUpdate, isInCooldown } = useUsageRefresh()
      
      const result = await refreshBackupUsageWithUpdate('test-backup', {})
      
      expect(result.success).toBe(false)
      expect(isInCooldown('test-backup')).toBe(false)
    })

    it('冷卻期中不應執行刷新', async () => {
      const { refreshBackupUsageWithUpdate, startCountdown, isInCooldown } = useUsageRefresh()
      
      // 先啟動冷卻期
      startCountdown('test-backup', 60)
      expect(isInCooldown('test-backup')).toBe(true)
      
      // 嘗試刷新
      const result = await refreshBackupUsageWithUpdate('test-backup', {})
      
      // 應該被跳過
      expect(result.success).toBe(false)
      expect(result.message).toContain('冷卻期')
      expect(mockRefreshBackupUsage).not.toHaveBeenCalled()
    })
  })

  describe('refreshCurrentUsageWithUpdate 含本地狀態更新', () => {
    it('應刷新當前帳號用量並調用 onLocalUpdate 回調', async () => {
      const { refreshCurrentUsageWithUpdate, refreshingCurrent } = useUsageRefresh()
      
      const onLocalUpdate = vi.fn()
      
      const result = await refreshCurrentUsageWithUpdate({ onLocalUpdate })
      
      expect(mockGetCurrentUsageInfo).toHaveBeenCalled()
      expect(result.success).toBe(true)
      expect(onLocalUpdate).toHaveBeenCalledTimes(1)
    })

    it('刷新過程中 refreshingCurrent 應為 true', async () => {
      let resolvePromise: (value: any) => void
      const pendingPromise = new Promise((resolve) => {
        resolvePromise = resolve
      })
      mockGetCurrentUsageInfo.mockReturnValue(pendingPromise)
      
      const { refreshCurrentUsageWithUpdate, refreshingCurrent } = useUsageRefresh()
      
      const promise = refreshCurrentUsageWithUpdate({})
      
      // 驗證進行中狀態
      expect(refreshingCurrent.value).toBe(true)
      
      // 完成操作
      resolvePromise!({ success: true, message: '' })
      await promise
      
      // 驗證狀態恢復
      expect(refreshingCurrent.value).toBe(false)
    })

    it('刷新成功後應自動啟動當前帳號冷卻期', async () => {
      const { refreshCurrentUsageWithUpdate, isCurrentInCooldown } = useUsageRefresh()
      
      await refreshCurrentUsageWithUpdate({})
      
      // 驗證冷卻期已啟動
      expect(isCurrentInCooldown()).toBe(true)
    })

    it('刷新失敗時不應啟動冷卻期', async () => {
      mockGetCurrentUsageInfo.mockRejectedValue(new Error('刷新失敗'))
      
      const { refreshCurrentUsageWithUpdate, isCurrentInCooldown } = useUsageRefresh()
      
      const result = await refreshCurrentUsageWithUpdate({})
      
      expect(result.success).toBe(false)
      expect(isCurrentInCooldown()).toBe(false)
    })

    it('冷卻期中不應執行刷新', async () => {
      const { refreshCurrentUsageWithUpdate, startCurrentCountdown, isCurrentInCooldown } = useUsageRefresh()
      
      // 先啟動冷卻期
      startCurrentCountdown(60)
      expect(isCurrentInCooldown()).toBe(true)
      
      // 嘗試刷新
      const result = await refreshCurrentUsageWithUpdate({})
      
      // 應該被跳過
      expect(result.success).toBe(false)
      expect(result.message).toContain('冷卻期')
      expect(mockGetCurrentUsageInfo).not.toHaveBeenCalled()
    })
  })
})
