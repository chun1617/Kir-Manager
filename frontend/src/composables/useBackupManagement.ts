/**
 * useBackupManagement Composable
 * @description 備份管理核心 Composable，負責管理備份列表、篩選、批量操作等功能
 * @see App.vue - 原始實作參考
 */
import { ref, computed, type Ref, type ComputedRef } from 'vue'
import type { BackupItem, Result, BatchResult } from '@/types/backup'
import { withTimeout, TimeoutError } from '@/utils/withTimeout'

/** 操作超時時間（毫秒） */
const OPERATION_TIMEOUT_MS = 30000

/**
 * loadFullBackupData 選項
 */
export interface LoadFullBackupDataOptions {
  onComplete?: () => void
  onError?: (error: Error) => void
}

/**
 * 備份管理 Composable 返回類型
 */
export interface UseBackupManagementReturn {
  // 狀態
  backups: Ref<BackupItem[]>
  currentMachineId: Ref<string>
  currentEnvironmentName: Ref<string>
  isLoadingBackups: Ref<boolean>
  searchQuery: Ref<string>
  filterSubscription: Ref<string>
  filterProvider: Ref<string>
  filterBalance: Ref<string>
  selectedBackups: Ref<Set<string>>
  batchOperating: Ref<boolean>
  creatingBackup: Ref<boolean>
  switchingBackup: Ref<string | null>
  deletingBackup: Ref<string | null>
  regeneratingId: Ref<string | null>

  // 計算屬性
  activeBackup: ComputedRef<BackupItem | null>
  filteredBackups: ComputedRef<BackupItem[]>

  // 方法
  loadBackups: (showOverlay?: boolean) => Promise<void>
  loadFullBackupData: (options?: LoadFullBackupDataOptions) => Promise<void>
  createBackup: (name: string) => Promise<Result>
  switchToBackup: (name: string) => Promise<Result>
  deleteBackup: (name: string) => Promise<Result>
  regenerateMachineID: (name: string) => Promise<Result>
  setFilterSubscription: (value: string) => void
  setFilterProvider: (value: string) => void
  setFilterBalance: (value: string) => void
  toggleSelect: (name: string) => void
  toggleSelectAll: () => void
  batchDelete: () => Promise<BatchResult>
  batchRegenerateMachineID: () => Promise<BatchResult>
  batchRefreshUsage: (isInCooldown: (name: string) => boolean) => Promise<BatchResult>
}

/**
 * 備份管理 Composable
 * @returns 備份管理相關的狀態和方法
 */
export function useBackupManagement(): UseBackupManagementReturn {
  // ============================================================================
  // 狀態定義
  // ============================================================================
  
  /** 備份列表 */
  const backups = ref<BackupItem[]>([])
  
  /** 當前機器 ID */
  const currentMachineId = ref<string>('')
  
  /** 當前環境名稱 */
  const currentEnvironmentName = ref<string>('')
  
  /** 是否正在載入備份（P0: 防止並發） */
  const isLoadingBackups = ref<boolean>(false)
  
  /** 搜尋關鍵字 */
  const searchQuery = ref<string>('')
  
  /** 訂閱篩選 */
  const filterSubscription = ref<string>('')
  
  /** 提供者篩選 */
  const filterProvider = ref<string>('')
  
  /** 餘額篩選 */
  const filterBalance = ref<string>('')
  
  /** 已選擇的備份 */
  const selectedBackups = ref<Set<string>>(new Set())
  
  /** 批量操作進行中 */
  const batchOperating = ref<boolean>(false)
  
  /** 創建備份進行中 */
  const creatingBackup = ref<boolean>(false)
  
  /** 正在切換的備份 */
  const switchingBackup = ref<string | null>(null)
  
  /** 正在刪除的備份 */
  const deletingBackup = ref<string | null>(null)
  
  /** 正在重新生成機器碼的備份 */
  const regeneratingId = ref<string | null>(null)


  // ============================================================================
  // 計算屬性
  // ============================================================================
  
  /**
   * 當前使用中的備份
   */
  const activeBackup = computed<BackupItem | null>(() => {
    return backups.value.find(b => b.isCurrent) || null
  })

  /**
   * 篩選後的備份列表
   * @description 根據訂閱、提供者、餘額和搜尋關鍵字進行篩選
   */
  const filteredBackups = computed<BackupItem[]>(() => {
    let result = backups.value

    // 1. 訂閱方案篩選（精確匹配）
    if (filterSubscription.value) {
      const subscriptionMap: Record<string, string> = {
        'FREE': 'KIRO FREE',
        'PRO': 'KIRO PRO',
        'PRO+': 'KIRO PRO+',
        'POWER': 'KIRO POWER'
      }
      const target = subscriptionMap[filterSubscription.value]
      result = result.filter(b => b.subscriptionTitle?.toUpperCase() === target)
    }

    // 2. 來源篩選（AWS 含 BuilderId）
    if (filterProvider.value) {
      if (filterProvider.value === 'AWS') {
        result = result.filter(b => b.provider === 'AWS' || b.provider === 'BuilderId')
      } else {
        result = result.filter(b => b.provider === filterProvider.value)
      }
    }

    // 3. 餘額篩選
    if (filterBalance.value) {
      if (filterBalance.value === 'LOW') {
        result = result.filter(b => b.usageLimit > 0 && b.isLowBalance)
      } else if (filterBalance.value === 'NORMAL') {
        result = result.filter(b => b.usageLimit > 0 && !b.isLowBalance)
      } else if (filterBalance.value === 'NO_DATA') {
        result = result.filter(b => b.usageLimit === 0)
      }
    }

    // 4. 文字搜尋
    if (searchQuery.value.trim()) {
      const query = searchQuery.value.toLowerCase()
      result = result.filter(b =>
        b.name.toLowerCase().includes(query) ||
        b.machineId?.toLowerCase().includes(query) ||
        b.provider?.toLowerCase().includes(query)
      )
    }

    return result
  })


  // ============================================================================
  // 方法
  // ============================================================================

  /**
   * 載入備份列表
   * @param showOverlay 是否顯示載入覆蓋層（預設 true）
   * @description 包含 P0 競態條件防護：防止並發調用
   */
  const loadBackups = async (showOverlay: boolean = true): Promise<void> => {
    // P0: 防止並發調用
    if (isLoadingBackups.value) return
    isLoadingBackups.value = true

    try {
      backups.value = await window.go.main.App.GetBackupList() || []
      currentMachineId.value = await window.go.main.App.GetCurrentMachineID()
      currentEnvironmentName.value = await window.go.main.App.GetCurrentEnvironmentName()
    } catch (e) {
      console.error('Failed to load backups:', e)
    } finally {
      isLoadingBackups.value = false
    }
  }

  /**
   * 整合載入備份數據（備份列表 + 設定 + 狀態）
   * @param options 選項，包含 onComplete 和 onError 回調
   * @description Phase 2 Task 1.1: 整合載入方法
   */
  const loadFullBackupData = async (options: LoadFullBackupDataOptions = {}): Promise<void> => {
    const { onComplete, onError } = options
    
    // P0: 防止並發調用
    if (isLoadingBackups.value) return
    isLoadingBackups.value = true

    try {
      backups.value = await window.go.main.App.GetBackupList() || []
      currentMachineId.value = await window.go.main.App.GetCurrentMachineID()
      currentEnvironmentName.value = await window.go.main.App.GetCurrentEnvironmentName()
      
      onComplete?.()
    } catch (e) {
      console.error('Failed to load full backup data:', e)
      onError?.(e as Error)
    } finally {
      isLoadingBackups.value = false
    }
  }

  /**
   * 創建備份
   * @param name 備份名稱
   * @returns 操作結果
   */
  const createBackup = async (name: string): Promise<Result> => {
    if (!name.trim()) {
      return { success: false, message: '備份名稱不能為空' }
    }

    // 檢查是否已存在同名備份
    const exists = backups.value.some(b => b.name === name)
    if (exists) {
      return { success: false, message: '備份名稱已存在' }
    }

    creatingBackup.value = true
    try {
      const result = await window.go.main.App.CreateBackup(name)
      return result
    } catch (e: any) {
      return { success: false, message: e.message || '創建備份失敗' }
    } finally {
      creatingBackup.value = false
    }
  }

  /**
   * 切換到指定備份
   * @param name 備份名稱
   * @returns 操作結果
   */
  const switchToBackup = async (name: string): Promise<Result> => {
    if (!name) {
      return { success: false, message: '請選擇備份' }
    }

    switchingBackup.value = name
    try {
      const result = await window.go.main.App.SwitchToBackup(name)
      return result
    } catch (e: any) {
      return { success: false, message: e.message || '切換備份失敗' }
    } finally {
      switchingBackup.value = null
    }
  }

  /**
   * 刪除備份
   * @param name 備份名稱
   * @returns 操作結果
   * @description 包含 30 秒超時保護
   */
  const deleteBackup = async (name: string): Promise<Result> => {
    if (!name) {
      return { success: false, message: '請選擇備份' }
    }

    deletingBackup.value = name
    try {
      const result = await withTimeout(
        window.go.main.App.DeleteBackup(name),
        OPERATION_TIMEOUT_MS,
        '刪除備份操作超時'
      )
      return result
    } catch (e: any) {
      if (e instanceof TimeoutError) {
        return { success: false, message: e.message }
      }
      return { success: false, message: e.message || '刪除備份失敗' }
    } finally {
      deletingBackup.value = null
    }
  }

  /**
   * 重新生成指定備份的機器碼
   * @param name 備份名稱
   * @returns 操作結果
   * @description Phase 2 Task 1.1: 單一備份機器碼重新生成，包含 30 秒超時保護
   */
  const regenerateMachineID = async (name: string): Promise<Result> => {
    if (!name) {
      return { success: false, message: '請選擇備份' }
    }

    regeneratingId.value = name
    try {
      const result = await withTimeout(
        window.go.main.App.RegenerateMachineID(name),
        OPERATION_TIMEOUT_MS,
        '重新生成機器碼操作超時'
      )
      return result
    } catch (e: any) {
      if (e instanceof TimeoutError) {
        return { success: false, message: e.message }
      }
      return { success: false, message: e.message || '重新生成機器碼失敗' }
    } finally {
      regeneratingId.value = null
    }
  }


  // ============================================================================
  // 篩選方法
  // ============================================================================

  /**
   * 設定訂閱篩選
   * @param value 篩選值
   */
  const setFilterSubscription = (value: string): void => {
    filterSubscription.value = value
  }

  /**
   * 設定提供者篩選
   * @param value 篩選值
   */
  const setFilterProvider = (value: string): void => {
    filterProvider.value = value
  }

  /**
   * 設定餘額篩選
   * @param value 篩選值
   */
  const setFilterBalance = (value: string): void => {
    filterBalance.value = value
  }

  // ============================================================================
  // 選擇操作
  // ============================================================================

  /**
   * 切換選擇
   * @param name 備份名稱
   */
  const toggleSelect = (name: string): void => {
    const newSet = new Set(selectedBackups.value)
    if (newSet.has(name)) {
      newSet.delete(name)
    } else {
      newSet.add(name)
    }
    selectedBackups.value = newSet
  }

  /**
   * 全選/取消全選
   */
  const toggleSelectAll = (): void => {
    if (selectedBackups.value.size === filteredBackups.value.length) {
      selectedBackups.value = new Set()
    } else {
      selectedBackups.value = new Set(filteredBackups.value.map(b => b.name))
    }
  }

  // ============================================================================
  // 批量操作
  // ============================================================================

  /**
   * 批量刪除
   * @returns BatchResult 包含成功數量和失敗項目
   */
  const batchDelete = async (): Promise<BatchResult> => {
    const result: BatchResult = {
      success: true,
      successCount: 0,
      failedItems: [],
    }
    
    if (selectedBackups.value.size === 0 || batchOperating.value) {
      return result
    }
    
    batchOperating.value = true
    try {
      for (const name of selectedBackups.value) {
        try {
          await window.go.main.App.DeleteBackup(name)
          result.successCount++
        } catch (e: any) {
          result.failedItems.push({
            name,
            error: e.message || '刪除失敗',
          })
        }
      }
      result.success = result.failedItems.length === 0
      selectedBackups.value = new Set()
      await loadBackups(false)
    } finally {
      batchOperating.value = false
    }
    return result
  }

  /**
   * 批量重新生成機器碼
   * @returns BatchResult 包含成功數量和失敗項目
   */
  const batchRegenerateMachineID = async (): Promise<BatchResult> => {
    const result: BatchResult = {
      success: true,
      successCount: 0,
      failedItems: [],
    }
    
    if (selectedBackups.value.size === 0 || batchOperating.value) {
      return result
    }
    
    batchOperating.value = true
    try {
      for (const name of selectedBackups.value) {
        try {
          await window.go.main.App.RegenerateMachineID(name)
          result.successCount++
        } catch (e: any) {
          result.failedItems.push({
            name,
            error: e.message || '重新生成失敗',
          })
        }
      }
      result.success = result.failedItems.length === 0
      selectedBackups.value = new Set()
      await loadBackups(false)
    } finally {
      batchOperating.value = false
    }
    return result
  }

  /**
   * 批量刷新用量
   * @param isInCooldown 判斷備份是否在冷卻期的函數
   * @returns BatchResult 包含成功數量、失敗項目和跳過數量
   * @description 包含全部冷卻期檢查：若所有選中項目都在冷卻期，返回特殊標記且不清空選擇
   */
  const batchRefreshUsage = async (isInCooldown: (name: string) => boolean): Promise<BatchResult> => {
    const result: BatchResult = {
      success: true,
      successCount: 0,
      failedItems: [],
      skippedCount: 0,
    }
    
    if (selectedBackups.value.size === 0 || batchOperating.value) {
      return result
    }
    
    // Task 1.5: 檢查是否所有選中項目都在冷卻期
    const selectedNames = Array.from(selectedBackups.value)
    const allInCooldown = selectedNames.every(name => isInCooldown(name))
    
    if (allInCooldown) {
      // 返回特殊結果，不清空選擇
      return {
        success: false,
        successCount: 0,
        failedItems: [],
        skippedCount: selectedNames.length,
        allInCooldown: true,
      }
    }
    
    batchOperating.value = true
    try {
      for (const name of selectedBackups.value) {
        // 跳過冷卻期中的備份
        if (isInCooldown(name)) {
          result.skippedCount!++
          continue
        }
        
        try {
          await window.go.main.App.RefreshBackupUsage(name)
          result.successCount++
        } catch (e: any) {
          result.failedItems.push({
            name,
            error: e.message || '刷新失敗',
          })
        }
      }
      result.success = result.failedItems.length === 0
      selectedBackups.value = new Set()
    } finally {
      batchOperating.value = false
    }
    return result
  }


  // ============================================================================
  // 返回
  // ============================================================================

  return {
    // 狀態
    backups,
    currentMachineId,
    currentEnvironmentName,
    isLoadingBackups,
    searchQuery,
    filterSubscription,
    filterProvider,
    filterBalance,
    selectedBackups,
    batchOperating,
    creatingBackup,
    switchingBackup,
    deletingBackup,
    regeneratingId,

    // 計算屬性
    activeBackup,
    filteredBackups,

    // 方法
    loadBackups,
    loadFullBackupData,
    createBackup,
    switchToBackup,
    deleteBackup,
    regenerateMachineID,
    setFilterSubscription,
    setFilterProvider,
    setFilterBalance,
    toggleSelect,
    toggleSelectAll,
    batchDelete,
    batchRegenerateMachineID,
    batchRefreshUsage,
  }
}
