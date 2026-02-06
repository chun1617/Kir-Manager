import { ref, type Ref } from 'vue'
import type { Result } from '@/types/backup'

/**
 * 刷新選項
 */
export interface RefreshOptions {
  onLocalUpdate?: () => void
}

/**
 * 用量刷新狀態返回類型
 */
export interface UsageRefreshReturn {
  // 狀態
  refreshingBackup: Ref<string | null>
  refreshingCurrent: Ref<boolean>
  countdownTimers: Ref<Record<string, number>>
  countdownCurrentAccount: Ref<number>
  
  // 方法
  isInCooldown: (name: string) => boolean
  isCurrentInCooldown: () => boolean
  startCountdown: (name: string, seconds: number) => void
  startCurrentCountdown: (seconds: number) => void
  clearAllCountdowns: () => void
  cleanup: () => void  // P0-FIX: Memory Leak 修復
  
  // Phase 2 Task 1.3: 擴展方法
  refreshBackupUsageWithUpdate: (name: string, options: RefreshOptions) => Promise<Result>
  refreshCurrentUsageWithUpdate: (options: RefreshOptions) => Promise<Result>
}

/** 預設刷新冷卻期（秒） */
export const REFRESH_COOLDOWN_SECONDS = 60

/**
 * 用量刷新管理 Composable
 * @description 管理備份用量刷新狀態和冷卻期倒計時
 * @requirements 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7
 */
export function useUsageRefresh(): UsageRefreshReturn {
  // ============================================
  // 狀態定義
  // ============================================
  
  /** 正在刷新的備份名稱 */
  const refreshingBackup = ref<string | null>(null)
  
  /** 是否正在刷新當前帳號 */
  const refreshingCurrent = ref(false)
  
  /** 各備份的倒計時秒數 */
  const countdownTimers = ref<Record<string, number>>({})
  
  /** 當前帳號的倒計時秒數 */
  const countdownCurrentAccount = ref(0)
  
  // ============================================
  // Interval 追蹤（Property 16: Interval 累積防護）
  // ============================================
  
  /** 各備份的 interval ID */
  const countdownIntervals = ref<Record<string, ReturnType<typeof setInterval> | null>>({})
  
  /** 當前帳號的 interval ID */
  let currentAccountInterval: ReturnType<typeof setInterval> | null = null

  // ============================================
  // 冷卻期檢查方法
  // ============================================
  
  /**
   * 檢查備份是否在冷卻期
   * @description Property 15: 冷卻期狀態一致性
   * @param name 備份名稱
   * @returns 是否在冷卻期
   */
  const isInCooldown = (name: string): boolean => {
    return (countdownTimers.value[name] ?? 0) > 0
  }
  
  /**
   * 檢查當前帳號是否在冷卻期
   * @description Property 15: 冷卻期狀態一致性
   * @returns 是否在冷卻期
   */
  const isCurrentInCooldown = (): boolean => {
    return countdownCurrentAccount.value > 0
  }

  // ============================================
  // 倒計時控制方法
  // ============================================
  
  /**
   * 開始備份倒計時
   * @description Property 16: Interval 累積防護 - 清除現有 interval 後再建立新的
   * @param name 備份名稱
   * @param seconds 倒計時秒數
   */
  const startCountdown = (name: string, seconds: number = REFRESH_COOLDOWN_SECONDS): void => {
    // Property 16: 清除現有 interval 防止累積
    if (countdownIntervals.value[name]) {
      clearInterval(countdownIntervals.value[name]!)
      countdownIntervals.value[name] = null
    }
    
    // 設定倒計時初始值
    countdownTimers.value[name] = seconds
    
    // 建立新的 interval
    countdownIntervals.value[name] = setInterval(() => {
      if (countdownTimers.value[name] > 0) {
        countdownTimers.value[name]--
      } else {
        // 倒計時結束，清除 interval
        if (countdownIntervals.value[name]) {
          clearInterval(countdownIntervals.value[name]!)
          countdownIntervals.value[name] = null
        }
      }
    }, 1000)
  }
  
  /**
   * 開始當前帳號倒計時
   * @description Property 16: Interval 累積防護 - 清除現有 interval 後再建立新的
   * @param seconds 倒計時秒數
   */
  const startCurrentCountdown = (seconds: number = REFRESH_COOLDOWN_SECONDS): void => {
    // Property 16: 清除現有 interval 防止累積
    if (currentAccountInterval) {
      clearInterval(currentAccountInterval)
      currentAccountInterval = null
    }
    
    // 設定倒計時初始值
    countdownCurrentAccount.value = seconds
    
    // 建立新的 interval
    currentAccountInterval = setInterval(() => {
      if (countdownCurrentAccount.value > 0) {
        countdownCurrentAccount.value--
      } else {
        // 倒計時結束，清除 interval
        if (currentAccountInterval) {
          clearInterval(currentAccountInterval)
          currentAccountInterval = null
        }
      }
    }, 1000)
  }
  
  /**
   * 清除所有倒計時
   * @description 清除所有備份和當前帳號的倒計時及其 interval
   */
  const clearAllCountdowns = (): void => {
    // 清除所有備份的 interval
    for (const name in countdownIntervals.value) {
      if (countdownIntervals.value[name]) {
        clearInterval(countdownIntervals.value[name]!)
        countdownIntervals.value[name] = null
      }
    }
    
    // 清除當前帳號的 interval
    if (currentAccountInterval) {
      clearInterval(currentAccountInterval)
      currentAccountInterval = null
    }
    
    // 重置所有倒計時值
    for (const name in countdownTimers.value) {
      countdownTimers.value[name] = 0
    }
    countdownCurrentAccount.value = 0
  }

  /**
   * 清理函數（供組件卸載時調用）
   * @description P0-FIX: Memory Leak 修復 - 導出 cleanup 函數供外部在 onUnmounted 中調用
   */
  const cleanup = (): void => {
    clearAllCountdowns()
  }

  // ============================================
  // Phase 2 Task 1.3: 擴展方法
  // ============================================

  /**
   * 刷新備份用量並更新本地狀態
   * @param name 備份名稱
   * @param options 選項，包含 onLocalUpdate 回調
   * @returns 操作結果
   * @description Phase 2 Task 1.3: 含本地狀態更新回調的刷新方法
   */
  const refreshBackupUsageWithUpdate = async (name: string, options: RefreshOptions): Promise<Result> => {
    const { onLocalUpdate } = options
    
    // 檢查冷卻期
    if (isInCooldown(name)) {
      return { success: false, message: '備份正在冷卻期中，請稍後再試' }
    }
    
    refreshingBackup.value = name
    try {
      await window.go.main.App.RefreshBackupUsage(name)
      
      // 成功後啟動冷卻期
      startCountdown(name, REFRESH_COOLDOWN_SECONDS)
      
      // 調用本地更新回調
      onLocalUpdate?.()
      
      return { success: true, message: '刷新成功' }
    } catch (e: any) {
      return { success: false, message: e.message || '刷新失敗' }
    } finally {
      refreshingBackup.value = null
    }
  }

  /**
   * 刷新當前帳號用量並更新本地狀態
   * @param options 選項，包含 onLocalUpdate 回調
   * @returns 操作結果
   * @description Phase 2 Task 1.3: 含本地狀態更新回調的刷新方法
   */
  const refreshCurrentUsageWithUpdate = async (options: RefreshOptions): Promise<Result> => {
    const { onLocalUpdate } = options
    
    // 檢查冷卻期
    if (isCurrentInCooldown()) {
      return { success: false, message: '當前帳號正在冷卻期中，請稍後再試' }
    }
    
    refreshingCurrent.value = true
    try {
      // 使用 GetCurrentUsageInfo 獲取最新用量資訊
      await window.go.main.App.GetCurrentUsageInfo()
      
      // 成功後啟動冷卻期
      startCurrentCountdown(REFRESH_COOLDOWN_SECONDS)
      
      // 調用本地更新回調
      onLocalUpdate?.()
      
      return { success: true, message: '刷新成功' }
    } catch (e: any) {
      return { success: false, message: e.message || '刷新失敗' }
    } finally {
      refreshingCurrent.value = false
    }
  }

  // ============================================
  // 返回公開 API
  // ============================================
  
  return {
    // 狀態
    refreshingBackup,
    refreshingCurrent,
    countdownTimers,
    countdownCurrentAccount,
    
    // 方法
    isInCooldown,
    isCurrentInCooldown,
    startCountdown,
    startCurrentCountdown,
    clearAllCountdowns,
    cleanup,  // P0-FIX: 導出 cleanup 函數
    
    // Phase 2 Task 1.3: 擴展方法
    refreshBackupUsageWithUpdate,
    refreshCurrentUsageWithUpdate,
  }
}
