/**
 * useAutoSwitch Composable
 * @description 自動切換核心 Composable，負責管理自動切換設定、監控狀態等功能
 * @see App.vue - 原始實作參考
 */
import { ref, type Ref } from 'vue'
import type { AutoSwitchSettings, AutoSwitchStatus, RefreshIntervalRule, Result } from '@/types/backup'

/**
 * 自動切換 Composable 返回類型
 */
export interface UseAutoSwitchReturn {
  // 狀態
  autoSwitchSettings: Ref<AutoSwitchSettings>
  autoSwitchStatus: Ref<AutoSwitchStatus>
  savingAutoSwitch: Ref<boolean>
  isToggling: Ref<boolean>  // P1-FIX: 並發保護狀態

  // 方法
  loadAutoSwitchSettings: () => Promise<void>
  saveAutoSwitchSettings: () => Promise<Result>
  toggleAutoSwitch: () => Promise<Result>
  handleAutoSwitchToggle: (enabled: boolean) => Promise<void>

  // 文件夾篩選方法
  addAutoSwitchFolder: (folderId: string) => Promise<void>
  removeAutoSwitchFolder: (folderId: string) => Promise<void>

  // 訂閱類型篩選方法
  addAutoSwitchSubscription: (subType: string) => Promise<void>
  removeAutoSwitchSubscription: (subType: string) => Promise<void>

  // 刷新規則方法
  addRefreshRule: () => void
  removeRefreshRule: (index: number) => Promise<void>
}

/**
 * 自動切換 Composable
 * @returns 自動切換相關的狀態和方法
 */
export function useAutoSwitch(): UseAutoSwitchReturn {
  // ============================================================================
  // 狀態定義
  // ============================================================================

  /** 自動切換設定 */
  const autoSwitchSettings = ref<AutoSwitchSettings>({
    enabled: false,
    balanceThreshold: 5,
    minTargetBalance: 50,
    folderIds: [],
    subscriptionTypes: [],
    refreshIntervals: [],
    notifyOnSwitch: true,
    notifyOnLowBalance: true,
  })

  /** 自動切換狀態 */
  const autoSwitchStatus = ref<AutoSwitchStatus>({
    status: 'stopped',
    lastBalance: 0,
    cooldownRemaining: 0,
    switchCount: 0,
  })

  /** 是否正在保存設定 */
  const savingAutoSwitch = ref<boolean>(false)

  /** P1-FIX: 是否正在切換（並發保護） */
  const isToggling = ref<boolean>(false)

  // ============================================================================
  // 方法
  // ============================================================================

  /**
   * 載入自動切換設定
   */
  const loadAutoSwitchSettings = async (): Promise<void> => {
    try {
      const settings = await window.go.main.App.GetAutoSwitchSettings()
      autoSwitchSettings.value = {
        enabled: settings.enabled,
        balanceThreshold: settings.balanceThreshold,
        minTargetBalance: settings.minTargetBalance,
        folderIds: settings.folderIds,
        subscriptionTypes: settings.subscriptionTypes,
        refreshIntervals: settings.refreshIntervals,
        notifyOnSwitch: settings.notifyOnSwitch,
        notifyOnLowBalance: settings.notifyOnLowBalance,
      }
      const status = await window.go.main.App.GetAutoSwitchStatus()
      autoSwitchStatus.value = {
        status: status.status as 'stopped' | 'running' | 'cooldown',
        lastBalance: status.lastBalance,
        cooldownRemaining: status.cooldownRemaining,
        switchCount: status.switchCount,
      }
    } catch (e) {
      console.error('Failed to load auto switch settings:', e)
    }
  }

  /**
   * 保存自動切換設定
   * @returns 操作結果
   */
  const saveAutoSwitchSettings = async (): Promise<Result> => {
    savingAutoSwitch.value = true
    try {
      // 使用 as any 繞過 Wails 生成的 DTO 類型中的 convertValues 方法要求
      const result = await window.go.main.App.SaveAutoSwitchSettings(autoSwitchSettings.value as any)
      return result
    } catch (e: any) {
      return { success: false, message: e.message || '保存設定失敗' }
    } finally {
      savingAutoSwitch.value = false
    }
  }

  /**
   * 切換自動切換狀態
   * @description P1-FIX: 新增並發保護，防止快速連續點擊導致狀態錯亂
   * @returns 操作結果
   */
  const toggleAutoSwitch = async (): Promise<Result> => {
    // P1-FIX: 並發保護
    if (isToggling.value) {
      return { success: false, message: '操作進行中' }
    }
    isToggling.value = true

    try {
      // 先保存設定，確保後端有最新的 Enabled 狀態
      await saveAutoSwitchSettings()

      if (autoSwitchSettings.value.enabled) {
        const result = await window.go.main.App.StartAutoSwitchMonitor()
        if (!result.success) {
          // 啟動失敗，回滾 enabled 狀態
          autoSwitchSettings.value.enabled = false
          await saveAutoSwitchSettings()
          // 刷新狀態顯示
          const status = await window.go.main.App.GetAutoSwitchStatus()
          autoSwitchStatus.value = {
            status: status.status as 'stopped' | 'running' | 'cooldown',
            lastBalance: status.lastBalance,
            cooldownRemaining: status.cooldownRemaining,
            switchCount: status.switchCount,
          }
          return result
        }
      } else {
        await window.go.main.App.StopAutoSwitchMonitor()
      }

      // 刷新狀態顯示
      const status = await window.go.main.App.GetAutoSwitchStatus()
      autoSwitchStatus.value = {
        status: status.status as 'stopped' | 'running' | 'cooldown',
        lastBalance: status.lastBalance,
        cooldownRemaining: status.cooldownRemaining,
        switchCount: status.switchCount,
      }
      return { success: true, message: '' }
    } finally {
      isToggling.value = false
    }
  }

  /**
   * 處理自動切換開關事件
   * @param enabled 是否啟用
   */
  const handleAutoSwitchToggle = async (enabled: boolean): Promise<void> => {
    autoSwitchSettings.value.enabled = enabled
    await toggleAutoSwitch()
  }

  // ============================================================================
  // 文件夾篩選方法
  // ============================================================================

  /**
   * 添加自動切換文件夾
   * @param folderId 文件夾 ID
   */
  const addAutoSwitchFolder = async (folderId: string): Promise<void> => {
    if (!folderId || autoSwitchSettings.value.folderIds.includes(folderId)) {
      return
    }
    autoSwitchSettings.value.folderIds.push(folderId)
    await saveAutoSwitchSettings()
  }

  /**
   * 移除自動切換文件夾
   * @param folderId 文件夾 ID
   */
  const removeAutoSwitchFolder = async (folderId: string): Promise<void> => {
    autoSwitchSettings.value.folderIds = autoSwitchSettings.value.folderIds.filter(id => id !== folderId)
    await saveAutoSwitchSettings()
  }

  // ============================================================================
  // 訂閱類型篩選方法
  // ============================================================================

  /**
   * 添加自動切換訂閱類型
   * @param subType 訂閱類型
   */
  const addAutoSwitchSubscription = async (subType: string): Promise<void> => {
    if (!subType || autoSwitchSettings.value.subscriptionTypes.includes(subType)) {
      return
    }
    autoSwitchSettings.value.subscriptionTypes.push(subType)
    await saveAutoSwitchSettings()
  }

  /**
   * 移除自動切換訂閱類型
   * @param subType 訂閱類型
   */
  const removeAutoSwitchSubscription = async (subType: string): Promise<void> => {
    autoSwitchSettings.value.subscriptionTypes = autoSwitchSettings.value.subscriptionTypes.filter(s => s !== subType)
    await saveAutoSwitchSettings()
  }

  // ============================================================================
  // 刷新規則方法
  // ============================================================================

  /**
   * 添加刷新規則
   */
  const addRefreshRule = (): void => {
    autoSwitchSettings.value.refreshIntervals.push({
      minBalance: 0,
      maxBalance: -1,
      interval: 60,
    })
  }

  /**
   * 移除刷新規則
   * @param index 規則索引
   */
  const removeRefreshRule = async (index: number): Promise<void> => {
    autoSwitchSettings.value.refreshIntervals.splice(index, 1)
    await saveAutoSwitchSettings()
  }

  // ============================================================================
  // 返回
  // ============================================================================

  return {
    // 狀態
    autoSwitchSettings,
    autoSwitchStatus,
    savingAutoSwitch,
    isToggling,  // P1-FIX: 導出並發保護狀態

    // 方法
    loadAutoSwitchSettings,
    saveAutoSwitchSettings,
    toggleAutoSwitch,
    handleAutoSwitchToggle,

    // 文件夾篩選方法
    addAutoSwitchFolder,
    removeAutoSwitchFolder,

    // 訂閱類型篩選方法
    addAutoSwitchSubscription,
    removeAutoSwitchSubscription,

    // 刷新規則方法
    addRefreshRule,
    removeRefreshRule,
  }
}
