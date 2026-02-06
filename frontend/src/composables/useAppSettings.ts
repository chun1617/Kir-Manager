/**
 * useAppSettings Composable
 * @description 應用設定核心 Composable，負責管理 Kiro 版本號、安裝路徑、低餘額閾值、語言切換等功能
 * @see App.vue - 原始實作參考
 * @requirements 6.1-6.8
 */
import { ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AppSettings, Result } from '@/types/backup'

/**
 * 路徑偵測狀態類型
 */
export type PathDetectionStatus = 'auto' | 'custom' | 'none'

/**
 * 應用設定 Composable 返回類型
 */
export interface UseAppSettingsReturn {
  // 狀態
  appSettings: Ref<AppSettings>
  kiroVersionInput: Ref<string>
  kiroVersionModified: Ref<boolean>
  kiroInstallPathInput: Ref<string>
  kiroInstallPathModified: Ref<boolean>
  thresholdPreview: Ref<number>
  detectingVersion: Ref<boolean>
  detectingPath: Ref<boolean>

  // 方法
  loadSettings: () => Promise<void>
  saveKiroVersion: () => Promise<void>
  detectKiroVersion: () => Promise<void>
  saveKiroInstallPath: () => Promise<void>
  detectKiroInstallPath: () => Promise<void>
  clearKiroInstallPath: () => Promise<void>
  saveLowBalanceThreshold: (
    value: number,
    onLocalUpdate?: (threshold: number) => void
  ) => Promise<void>
  switchLanguage: (lang: string) => void
  onKiroVersionInput: () => void
  onKiroInstallPathInput: () => void
  checkPathDetectionStatus: (
    customPath: string,
    useAutoDetect: boolean
  ) => PathDetectionStatus
}

/** localStorage key for language */
const LANG_STORAGE_KEY = 'kiro-manager-lang'

/**
 * 應用設定 Composable
 * @returns 應用設定相關的狀態和方法
 */
export function useAppSettings(): UseAppSettingsReturn {
  const { locale } = useI18n()

  // ============================================================================
  // 狀態定義
  // ============================================================================

  /** 應用設定 */
  const appSettings = ref<AppSettings>({
    lowBalanceThreshold: 0.2,
    kiroVersion: '0.8.206',
    useAutoDetect: true,
    customKiroInstallPath: '',
  })

  /** Kiro 版本號輸入值 */
  const kiroVersionInput = ref<string>('0.8.206')

  /** 追蹤版本號是否被用戶手動修改 */
  const kiroVersionModified = ref<boolean>(false)

  /** Kiro 安裝路徑輸入值 */
  const kiroInstallPathInput = ref<string>('')

  /** 追蹤路徑是否被用戶手動修改 */
  const kiroInstallPathModified = ref<boolean>(false)

  /** 低餘額閾值預覽值（拖動滑桿時實時更新） */
  const thresholdPreview = ref<number>(20)

  /** 偵測版本中狀態 */
  const detectingVersion = ref<boolean>(false)

  /** 偵測路徑中狀態 */
  const detectingPath = ref<boolean>(false)

  // ============================================================================
  // 方法
  // ============================================================================

  /**
   * 載入應用設定
   */
  const loadSettings = async (): Promise<void> => {
    try {
      appSettings.value = await window.go.main.App.GetSettings()
      thresholdPreview.value = Math.round(appSettings.value.lowBalanceThreshold * 100)
      kiroVersionInput.value = appSettings.value.kiroVersion || '0.8.206'
      kiroVersionModified.value = false
      kiroInstallPathInput.value = appSettings.value.customKiroInstallPath || ''
      kiroInstallPathModified.value = false
    } catch (e) {
      console.error('Failed to load settings:', e)
    }
  }

  /**
   * 儲存 Kiro 版本號
   * @description 儲存自定義版本時，關閉自動偵測模式
   * @requirements 6.1 - 版本號保存與自動偵測互斥
   */
  const saveKiroVersion = async (): Promise<void> => {
    const version = kiroVersionInput.value.trim()
    if (!version) return

    try {
      const result = await window.go.main.App.SaveSettings({
        lowBalanceThreshold: appSettings.value.lowBalanceThreshold,
        kiroVersion: version,
        useAutoDetect: false, // Property 24: 保存版本號時關閉自動偵測
        customKiroInstallPath: appSettings.value.customKiroInstallPath,
      })
      if (result.success) {
        appSettings.value.kiroVersion = version
        appSettings.value.useAutoDetect = false
        kiroVersionModified.value = false
      }
    } catch (e) {
      console.error('Failed to save Kiro version:', e)
    }
  }

  /**
   * 自動偵測 Kiro 版本並啟用自動偵測模式
   * @description 偵測成功後啟用自動偵測模式
   * @requirements 6.2 - 版本號保存與自動偵測互斥
   */
  const detectKiroVersion = async (): Promise<void> => {
    detectingVersion.value = true
    try {
      const result = await window.go.main.App.GetDetectedKiroVersion()
      if (result.success) {
        kiroVersionInput.value = result.message
        // Property 24: 自動偵測時啟用 useAutoDetect
        const saveResult = await window.go.main.App.SaveSettings({
          lowBalanceThreshold: appSettings.value.lowBalanceThreshold,
          kiroVersion: result.message,
          useAutoDetect: true,
          customKiroInstallPath: appSettings.value.customKiroInstallPath,
        })
        if (saveResult.success) {
          appSettings.value.kiroVersion = result.message
          appSettings.value.useAutoDetect = true
          kiroVersionModified.value = false
        }
      }
    } catch (e) {
      console.error('Failed to detect Kiro version:', e)
    } finally {
      detectingVersion.value = false
    }
  }

  /**
   * 儲存自定義安裝路徑
   * @requirements 6.3
   */
  const saveKiroInstallPath = async (): Promise<void> => {
    const path = kiroInstallPathInput.value.trim()

    try {
      const result = await window.go.main.App.SaveSettings({
        lowBalanceThreshold: appSettings.value.lowBalanceThreshold,
        kiroVersion: appSettings.value.kiroVersion,
        useAutoDetect: appSettings.value.useAutoDetect,
        customKiroInstallPath: path,
      })
      if (result.success) {
        appSettings.value.customKiroInstallPath = path
        kiroInstallPathModified.value = false
      }
    } catch (e) {
      console.error('Failed to save Kiro install path:', e)
    }
  }

  /**
   * 自動偵測 Kiro 安裝路徑
   * @requirements 6.4
   */
  const detectKiroInstallPath = async (): Promise<void> => {
    detectingPath.value = true
    try {
      const result = await window.go.main.App.GetDetectedKiroInstallPath()
      if (result.success) {
        kiroInstallPathInput.value = result.message
        const saveResult = await window.go.main.App.SaveSettings({
          lowBalanceThreshold: appSettings.value.lowBalanceThreshold,
          kiroVersion: appSettings.value.kiroVersion,
          useAutoDetect: appSettings.value.useAutoDetect,
          customKiroInstallPath: result.message,
        })
        if (saveResult.success) {
          appSettings.value.customKiroInstallPath = result.message
          kiroInstallPathModified.value = false
        }
      }
    } catch (e) {
      console.error('Failed to detect Kiro install path:', e)
    } finally {
      detectingPath.value = false
    }
  }

  /**
   * 清除自定義安裝路徑（恢復自動偵測）
   * @description Property 25: 清除路徑時將 customKiroInstallPath 設為空字串
   * @requirements 6.5
   */
  const clearKiroInstallPath = async (): Promise<void> => {
    try {
      const result = await window.go.main.App.SaveSettings({
        lowBalanceThreshold: appSettings.value.lowBalanceThreshold,
        kiroVersion: appSettings.value.kiroVersion,
        useAutoDetect: appSettings.value.useAutoDetect,
        customKiroInstallPath: '',
      })
      if (result.success) {
        appSettings.value.customKiroInstallPath = ''
        kiroInstallPathInput.value = ''
        kiroInstallPathModified.value = false
      }
    } catch (e) {
      console.error('Failed to clear Kiro install path:', e)
    }
  }

  /**
   * 儲存低餘額閾值
   * @description Property 26: 儲存後調用回調函數更新本地 isLowBalance 狀態
   * @param value 閾值 (0.0 ~ 1.0)
   * @param onLocalUpdate 本地更新回調函數
   * @requirements 6.6
   */
  const saveLowBalanceThreshold = async (
    value: number,
    onLocalUpdate?: (threshold: number) => void
  ): Promise<void> => {
    try {
      const result = await window.go.main.App.SaveSettings({
        lowBalanceThreshold: value,
        kiroVersion: appSettings.value.kiroVersion,
        useAutoDetect: appSettings.value.useAutoDetect,
        customKiroInstallPath: appSettings.value.customKiroInstallPath,
      })
      if (result.success) {
        appSettings.value.lowBalanceThreshold = value
        // Property 26: 調用回調函數更新本地 isLowBalance 狀態
        if (onLocalUpdate) {
          onLocalUpdate(value)
        }
      }
    } catch (e) {
      console.error('Failed to save low balance threshold:', e)
    }
  }

  /**
   * 切換語言
   * @description Property 27: 將語言設定存入 localStorage
   * @param lang 語言代碼
   * @requirements 6.7
   */
  const switchLanguage = (lang: string): void => {
    locale.value = lang
    localStorage.setItem(LANG_STORAGE_KEY, lang)
  }

  /**
   * 處理版本號輸入變更
   * @requirements 6.8
   */
  const onKiroVersionInput = (): void => {
    kiroVersionModified.value = true
  }

  /**
   * 處理安裝路徑輸入變更
   * @requirements 6.8
   */
  const onKiroInstallPathInput = (): void => {
    kiroInstallPathModified.value = true
  }

  /**
   * 檢查路徑偵測狀態
   * @param customPath 自定義路徑
   * @param useAutoDetect 是否使用自動偵測
   * @returns 路徑偵測狀態
   */
  const checkPathDetectionStatus = (
    customPath: string,
    useAutoDetect: boolean
  ): PathDetectionStatus => {
    if (customPath) {
      return 'custom'
    }
    if (useAutoDetect) {
      return 'auto'
    }
    return 'none'
  }

  // ============================================================================
  // 返回
  // ============================================================================

  return {
    // 狀態
    appSettings,
    kiroVersionInput,
    kiroVersionModified,
    kiroInstallPathInput,
    kiroInstallPathModified,
    thresholdPreview,
    detectingVersion,
    detectingPath,

    // 方法
    loadSettings,
    saveKiroVersion,
    detectKiroVersion,
    saveKiroInstallPath,
    detectKiroInstallPath,
    clearKiroInstallPath,
    saveLowBalanceThreshold,
    switchLanguage,
    onKiroVersionInput,
    onKiroInstallPathInput,
    checkPathDetectionStatus,
  }
}
