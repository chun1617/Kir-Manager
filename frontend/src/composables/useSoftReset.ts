/**
 * useSoftReset Composable
 * @description 軟重置核心 Composable，負責管理軟重置、還原原始機器、Extension Patch 等功能
 * @see App.vue - 原始實作參考
 */
import { ref, type Ref } from 'vue'
import type { SoftResetStatus, Result } from '@/types/backup'
import { withTimeout, TimeoutError } from '@/utils/withTimeout'

/** 操作超時時間（毫秒） */
const OPERATION_TIMEOUT_MS = 30000

/**
 * 軟重置 Composable 返回類型
 */
export interface UseSoftResetReturn {
  // 狀態
  softResetStatus: Ref<SoftResetStatus>
  resetting: Ref<boolean>
  restoringOriginal: Ref<boolean>
  patching: Ref<boolean>
  hasUsedReset: Ref<boolean>
  showFirstTimeResetModal: Ref<boolean>

  // 方法
  getSoftResetStatus: () => Promise<void>
  loadHasUsedReset: () => void
  resetToNew: () => Promise<void>
  executeReset: () => Promise<Result>
  confirmFirstTimeReset: () => Promise<void>
  restoreOriginal: () => Promise<Result>
  regenerateMachineID: (backupName: string) => Promise<Result>
  patchExtension: () => Promise<Result>

  // 工具方法
  openExtensionFolder: () => Promise<void>
  openMachineIDFolder: () => Promise<void>
  openSSOCacheFolder: () => Promise<void>
}

/** localStorage key for hasUsedReset */
const HAS_USED_RESET_KEY = 'kiro-manager-has-used-reset'

/**
 * 軟重置 Composable
 * @returns 軟重置相關的狀態和方法
 */
export function useSoftReset(): UseSoftResetReturn {
  // ============================================================================
  // 狀態定義
  // ============================================================================

  /** 軟重置狀態 */
  const softResetStatus = ref<SoftResetStatus>({
    isPatched: false,
    hasCustomId: false,
    customMachineId: '',
    extensionPath: '',
    isSupported: true,
  })

  /** 一鍵新機進行中狀態 */
  const resetting = ref<boolean>(false)

  /** 還原原始機器進行中狀態 */
  const restoringOriginal = ref<boolean>(false)

  /** Extension Patch 進行中狀態 */
  const patching = ref<boolean>(false)

  /** 是否已使用過一鍵新機 */
  const hasUsedReset = ref<boolean>(false)

  /** 是否顯示首次使用對話框 */
  const showFirstTimeResetModal = ref<boolean>(false)

  // ============================================================================
  // 方法
  // ============================================================================

  /**
   * 載入軟重置狀態
   */
  const getSoftResetStatus = async (): Promise<void> => {
    try {
      softResetStatus.value = await window.go.main.App.GetSoftResetStatus()
    } catch (e) {
      console.error('Failed to get soft reset status:', e)
    }
  }

  /**
   * 從 localStorage 載入是否已使用過一鍵新機
   */
  const loadHasUsedReset = (): void => {
    hasUsedReset.value = localStorage.getItem(HAS_USED_RESET_KEY) === 'true'
  }

  /**
   * 一鍵新機（檢查是否首次使用）
   */
  const resetToNew = async (): Promise<void> => {
    if (!hasUsedReset.value) {
      showFirstTimeResetModal.value = true
      return
    }
    // 如果已經使用過，直接執行（實際使用時會有確認對話框）
    // 這裡不直接執行，讓調用方處理確認邏輯
  }

  /**
   * 執行軟重置
   * @returns 操作結果
   */
  const executeReset = async (): Promise<Result> => {
    resetting.value = true
    try {
      const result = await window.go.main.App.SoftResetToNewMachine()
      if (result.success) {
        hasUsedReset.value = true
        localStorage.setItem(HAS_USED_RESET_KEY, 'true')
      }
      return result
    } catch (e: any) {
      return { success: false, message: e.message || '軟重置失敗' }
    } finally {
      resetting.value = false
    }
  }

  /**
   * 確認首次使用軟重置
   */
  const confirmFirstTimeReset = async (): Promise<void> => {
    showFirstTimeResetModal.value = false
    await executeReset()
  }

  /**
   * 還原原始機器
   * @returns 操作結果
   */
  const restoreOriginal = async (): Promise<Result> => {
    restoringOriginal.value = true
    try {
      const result = await window.go.main.App.RestoreSoftReset()
      return result
    } catch (e: any) {
      return { success: false, message: e.message || '還原失敗' }
    } finally {
      restoringOriginal.value = false
    }
  }

  /**
   * 重新生成備份的機器碼
   * @param backupName 備份名稱
   * @returns 操作結果
   */
  const regenerateMachineID = async (backupName: string): Promise<Result> => {
    try {
      const result = await window.go.main.App.RegenerateMachineID(backupName)
      return result
    } catch (e: any) {
      return { success: false, message: e.message || '重新生成機器碼失敗' }
    }
  }

  /**
   * 重新 Patch Extension
   * @returns 操作結果
   * @description 包含 30 秒超時保護
   */
  const patchExtension = async (): Promise<Result> => {
    patching.value = true
    try {
      const result = await withTimeout(
        window.go.main.App.RepatchExtension(),
        OPERATION_TIMEOUT_MS,
        'Patch Extension 操作超時'
      )
      if (result.success) {
        softResetStatus.value = await window.go.main.App.GetSoftResetStatus()
      }
      return result
    } catch (e: any) {
      if (e instanceof TimeoutError) {
        return { success: false, message: e.message }
      }
      return { success: false, message: e.message || 'Patch 失敗' }
    } finally {
      patching.value = false
    }
  }

  // ============================================================================
  // 工具方法
  // ============================================================================

  /**
   * 開啟 Extension 資料夾
   */
  const openExtensionFolder = async (): Promise<void> => {
    await window.go.main.App.OpenExtensionFolder()
  }

  /**
   * 開啟 Machine ID 資料夾
   */
  const openMachineIDFolder = async (): Promise<void> => {
    await window.go.main.App.OpenMachineIDFolder()
  }

  /**
   * 開啟 SSO Cache 資料夾
   */
  const openSSOCacheFolder = async (): Promise<void> => {
    await window.go.main.App.OpenSSOCacheFolder()
  }

  // ============================================================================
  // 返回
  // ============================================================================

  return {
    // 狀態
    softResetStatus,
    resetting,
    restoringOriginal,
    patching,
    hasUsedReset,
    showFirstTimeResetModal,

    // 方法
    getSoftResetStatus,
    loadHasUsedReset,
    resetToNew,
    executeReset,
    confirmFirstTimeReset,
    restoreOriginal,
    regenerateMachineID,
    patchExtension,

    // 工具方法
    openExtensionFolder,
    openMachineIDFolder,
    openSSOCacheFolder,
  }
}
