<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from './components/Icon.vue'

const { t, locale } = useI18n()

interface BackupItem {
  name: string
  backupTime: string
  hasToken: boolean
  hasMachineId: boolean
  machineId: string
  provider: string
  isCurrent: boolean
  isOriginalMachine: boolean // Machine ID 與原始機器相同
  isTokenExpired: boolean    // Token 是否已過期
  // Usage 相關欄位 (Requirements: 1.1, 1.2)
  subscriptionTitle: string  // 訂閱類型名稱
  usageLimit: number         // 總額度
  currentUsage: number       // 已使用
  balance: number            // 餘額
  isLowBalance: boolean      // 餘額低於 20%
  cachedAt: string           // 緩存時間（用於判斷冷卻期）
}

interface Result {
  success: boolean
  message: string
}

interface CurrentUsageInfo {
  subscriptionTitle: string
  usageLimit: number
  currentUsage: number
  balance: number
  isLowBalance: boolean
}

interface AppSettings {
  lowBalanceThreshold: number
  kiroVersion: string
  useAutoDetect: boolean
  customKiroInstallPath: string
}

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetBackupList(): Promise<BackupItem[]>
          CreateBackup(name: string): Promise<Result>
          SwitchToBackup(name: string): Promise<Result>
          RestoreSoftReset(): Promise<Result>
          DeleteBackup(name: string): Promise<Result>
          RegenerateMachineID(name: string): Promise<Result>
          GetCurrentMachineID(): Promise<string>
          GetCurrentEnvironmentName(): Promise<string>
          EnsureOriginalBackup(): Promise<Result>
          SoftResetToNewMachine(): Promise<Result>
          IsKiroRunning(): Promise<boolean>
          GetSoftResetStatus(): Promise<{
            isPatched: boolean
            hasCustomId: boolean
            customMachineId: string
            extensionPath: string
            isSupported: boolean
          }>
          GetCurrentProvider(): Promise<string>
          GetCurrentUsageInfo(): Promise<CurrentUsageInfo | null>
          RefreshBackupUsage(name: string): Promise<{
            success: boolean
            message: string
            subscriptionTitle: string
            usageLimit: number
            currentUsage: number
            balance: number
            isLowBalance: boolean
            isTokenExpired: boolean
            cachedAt: string
          }>
          GetSettings(): Promise<AppSettings>
          SaveSettings(settings: AppSettings): Promise<Result>
          GetDetectedKiroVersion(): Promise<Result>
          GetDetectedKiroInstallPath(): Promise<Result>
          OpenExtensionFolder(): Promise<Result>
          OpenMachineIDFolder(): Promise<Result>
          OpenSSOCacheFolder(): Promise<Result>
          RepatchExtension(): Promise<Result>
        }
      }
    }
  }
}

const backups = ref<BackupItem[]>([])
const currentMachineId = ref('')
const currentEnvironmentName = ref('') // 當前運行環境的名稱（對應的環境快照名稱）
const currentProvider = ref('') // 當前 Kiro 登入的帳號來源
const currentUsageInfo = ref<CurrentUsageInfo | null>(null) // 當前帳號用量資訊
const loading = ref(false)
const kiroRunning = ref(false)
const showCreateModal = ref(false)
const newBackupName = ref('')
const searchQuery = ref('')
const toast = ref<{ show: boolean; message: string; type: 'success' | 'error' }>({
  show: false,
  message: '',
  type: 'success'
})

// 一鍵新機模式相關
const resetMode = ref<'soft'>('soft')
const hasUsedReset = ref(false)
const showFirstTimeResetModal = ref(false)
const showSettingsPanel = ref(false)
const activeMenu = ref<'dashboard' | 'settings'>('dashboard')
const resetting = ref(false) // 一鍵新機進行中狀態
const refreshingBackup = ref<string | null>(null) // 正在刷新餘額的備份名稱
const refreshingCurrent = ref(false) // 正在刷新當前帳號餘額
const patching = ref(false) // Extension Patch 進行中狀態

// 刷新冷卻期（60 秒）
const REFRESH_COOLDOWN_SECONDS = 60
const copiedMachineId = ref<string | null>(null) // 剛複製的機器碼 ID（用於顯示提示）

// 倒計時狀態：key 為備份名稱，value 為剩餘秒數（0 表示無倒計時）
const countdownTimers = ref<Record<string, number>>({})
const countdownCurrentAccount = ref(0) // 當前帳號的倒計時秒數

// 軟重置狀態
const softResetStatus = ref<{
  isPatched: boolean
  hasCustomId: boolean
  customMachineId: string
  extensionPath: string
  isSupported: boolean
}>({
  isPatched: false,
  hasCustomId: false,
  customMachineId: '',
  extensionPath: '',
  isSupported: false
})

// 全域設定
const appSettings = ref<AppSettings>({
  lowBalanceThreshold: 0.2,
  kiroVersion: '0.7.5',
  useAutoDetect: true,
  customKiroInstallPath: ''
})

// Kiro 版本號輸入值
const kiroVersionInput = ref('0.7.5')
// 追蹤版本號是否被用戶手動修改（用於控制確認按鍵狀態）
const kiroVersionModified = ref(false)

// Kiro 安裝路徑輸入值
const kiroInstallPathInput = ref('')
// 追蹤路徑是否被用戶手動修改
const kiroInstallPathModified = ref(false)
// 偵測路徑中狀態
const detectingPath = ref(false)

// 低餘額閾值預覽值（拖動滑桿時實時更新）
const thresholdPreview = ref(20)

// 確認對話框狀態
const confirmDialog = ref<{
  show: boolean
  title: string
  message: string
  type: 'warning' | 'danger' | 'info'
  confirmText: string
  cancelText: string
  onConfirm: () => void
  onCancel: () => void
}>({
  show: false,
  title: '',
  message: '',
  type: 'warning',
  confirmText: '',
  cancelText: '',
  onConfirm: () => {},
  onCancel: () => {}
})

// 顯示確認對話框並返回 Promise
const showConfirmDialog = (options: {
  title: string
  message: string
  type?: 'warning' | 'danger' | 'info'
  confirmText?: string
  cancelText?: string
}): Promise<boolean> => {
  return new Promise((resolve) => {
    confirmDialog.value = {
      show: true,
      title: options.title,
      message: options.message,
      type: options.type || 'warning',
      confirmText: options.confirmText || t('backup.confirm'),
      cancelText: options.cancelText || t('backup.cancel'),
      onConfirm: () => {
        confirmDialog.value.show = false
        resolve(true)
      },
      onCancel: () => {
        confirmDialog.value.show = false
        resolve(false)
      }
    }
  })
}

const activeBackup = computed(() => {
  return backups.value.find(b => b.isCurrent) || null
})

const filteredBackups = computed(() => {
  if (!searchQuery.value.trim()) return backups.value
  const query = searchQuery.value.toLowerCase()
  return backups.value.filter(b => 
    b.name.toLowerCase().includes(query) ||
    b.machineId?.toLowerCase().includes(query) ||
    b.provider?.toLowerCase().includes(query)
  )
})

const switchLanguage = (lang: string) => {
  locale.value = lang
  localStorage.setItem('kiro-manager-lang', lang)
}

const showToast = (message: string, type: 'success' | 'error') => {
  toast.value = { show: true, message, type }
  setTimeout(() => {
    toast.value.show = false
  }, 3000)
}

const checkKiroStatus = async () => {
  try {
    kiroRunning.value = await window.go.main.App.IsKiroRunning()
  } catch (e) {
    console.error(e)
  }
}

const loadBackups = async () => {
  loading.value = true
  try {
    backups.value = await window.go.main.App.GetBackupList() || []
    currentMachineId.value = await window.go.main.App.GetCurrentMachineID()
    currentEnvironmentName.value = await window.go.main.App.GetCurrentEnvironmentName()
    softResetStatus.value = await window.go.main.App.GetSoftResetStatus()
    currentProvider.value = await window.go.main.App.GetCurrentProvider()
    currentUsageInfo.value = await window.go.main.App.GetCurrentUsageInfo()
    appSettings.value = await window.go.main.App.GetSettings()
    thresholdPreview.value = Math.round(appSettings.value.lowBalanceThreshold * 100)
    kiroVersionInput.value = appSettings.value.kiroVersion || '0.7.5'
    kiroVersionModified.value = false // 重置修改狀態
    kiroInstallPathInput.value = appSettings.value.customKiroInstallPath || ''
    kiroInstallPathModified.value = false // 重置修改狀態
    await checkKiroStatus()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

const saveLowBalanceThreshold = async (value: number) => {
  try {
    const result = await window.go.main.App.SaveSettings({
      lowBalanceThreshold: value,
      kiroVersion: appSettings.value.kiroVersion,
      useAutoDetect: appSettings.value.useAutoDetect,
      customKiroInstallPath: appSettings.value.customKiroInstallPath
    })
    if (result.success) {
      appSettings.value.lowBalanceThreshold = value
      // 本地更新 isLowBalance 狀態，避免觸發全域 loading
      backups.value.forEach(backup => {
        if (backup.usageLimit > 0) {
          backup.isLowBalance = (backup.balance / backup.usageLimit) < value
        }
      })
      // 更新當前帳號的 isLowBalance
      if (currentUsageInfo.value && currentUsageInfo.value.usageLimit > 0) {
        currentUsageInfo.value.isLowBalance = 
          (currentUsageInfo.value.balance / currentUsageInfo.value.usageLimit) < value
      }
    } else {
      showToast(result.message, 'error')
    }
  } catch (e) {
    console.error(e)
  }
}

const saveKiroVersion = async () => {
  const version = kiroVersionInput.value.trim()
  if (!version) return
  
  try {
    // 儲存自定義版本時，關閉自動偵測模式
    const result = await window.go.main.App.SaveSettings({
      lowBalanceThreshold: appSettings.value.lowBalanceThreshold,
      kiroVersion: version,
      useAutoDetect: false,
      customKiroInstallPath: appSettings.value.customKiroInstallPath
    })
    if (result.success) {
      appSettings.value.kiroVersion = version
      appSettings.value.useAutoDetect = false
      kiroVersionModified.value = false // 儲存後重置修改狀態
      showToast(t('message.success'), 'success')
    } else {
      showToast(result.message, 'error')
    }
  } catch (e) {
    console.error(e)
  }
}

// 處理版本號輸入變更
const onKiroVersionInput = () => {
  kiroVersionModified.value = true
}

// 偵測版本中狀態
const detectingVersion = ref(false)

// 自動偵測 Kiro 版本並啟用自動偵測模式
const detectKiroVersion = async () => {
  detectingVersion.value = true
  try {
    const result = await window.go.main.App.GetDetectedKiroVersion()
    if (result.success) {
      kiroVersionInput.value = result.message
      // 啟用自動偵測模式並儲存設定
      const saveResult = await window.go.main.App.SaveSettings({
        lowBalanceThreshold: appSettings.value.lowBalanceThreshold,
        kiroVersion: result.message,
        useAutoDetect: true,
        customKiroInstallPath: appSettings.value.customKiroInstallPath
      })
      if (saveResult.success) {
        appSettings.value.kiroVersion = result.message
        appSettings.value.useAutoDetect = true
        kiroVersionModified.value = false // 自動偵測後重置修改狀態
        showToast(t('message.success'), 'success')
      } else {
        showToast(saveResult.message, 'error')
      }
    } else {
      showToast(t('settings.detectVersionFailed'), 'error')
    }
  } catch (e) {
    console.error(e)
    showToast(t('settings.detectVersionFailed'), 'error')
  } finally {
    detectingVersion.value = false
  }
}

// 處理安裝路徑輸入變更
const onKiroInstallPathInput = () => {
  kiroInstallPathModified.value = true
}

// 儲存自定義安裝路徑
const saveKiroInstallPath = async () => {
  const path = kiroInstallPathInput.value.trim()
  
  try {
    const result = await window.go.main.App.SaveSettings({
      lowBalanceThreshold: appSettings.value.lowBalanceThreshold,
      kiroVersion: appSettings.value.kiroVersion,
      useAutoDetect: appSettings.value.useAutoDetect,
      customKiroInstallPath: path
    })
    if (result.success) {
      appSettings.value.customKiroInstallPath = path
      kiroInstallPathModified.value = false
      showToast(t('message.success'), 'success')
    } else {
      showToast(result.message, 'error')
    }
  } catch (e) {
    console.error(e)
  }
}

// 自動偵測 Kiro 安裝路徑
const detectKiroInstallPath = async () => {
  detectingPath.value = true
  try {
    const result = await window.go.main.App.GetDetectedKiroInstallPath()
    if (result.success) {
      kiroInstallPathInput.value = result.message
      // 儲存偵測到的路徑
      const saveResult = await window.go.main.App.SaveSettings({
        lowBalanceThreshold: appSettings.value.lowBalanceThreshold,
        kiroVersion: appSettings.value.kiroVersion,
        useAutoDetect: appSettings.value.useAutoDetect,
        customKiroInstallPath: result.message
      })
      if (saveResult.success) {
        appSettings.value.customKiroInstallPath = result.message
        kiroInstallPathModified.value = false
        showToast(t('message.success'), 'success')
      } else {
        showToast(saveResult.message, 'error')
      }
    } else {
      showToast(t('settings.detectPathFailed'), 'error')
    }
  } catch (e) {
    console.error(e)
    showToast(t('settings.detectPathFailed'), 'error')
  } finally {
    detectingPath.value = false
  }
}

// 清除自定義安裝路徑（恢復自動偵測）
const clearKiroInstallPath = async () => {
  try {
    const result = await window.go.main.App.SaveSettings({
      lowBalanceThreshold: appSettings.value.lowBalanceThreshold,
      kiroVersion: appSettings.value.kiroVersion,
      useAutoDetect: appSettings.value.useAutoDetect,
      customKiroInstallPath: ''
    })
    if (result.success) {
      appSettings.value.customKiroInstallPath = ''
      kiroInstallPathInput.value = ''
      kiroInstallPathModified.value = false
      showToast(t('message.success'), 'success')
    } else {
      showToast(result.message, 'error')
    }
  } catch (e) {
    console.error(e)
  }
}

const createBackup = async () => {
  if (!newBackupName.value.trim()) return
  
  loading.value = true
  try {
    const result = await window.go.main.App.CreateBackup(newBackupName.value.trim())
    if (result.success) {
      showToast(t('message.success'), 'success')
      showCreateModal.value = false
      newBackupName.value = ''
      await loadBackups()
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    loading.value = false
  }
}

const switchToBackup = async (name: string) => {
  const confirmed = await showConfirmDialog({
    title: t('dialog.confirmTitle'),
    message: t('message.confirmSwitch', { name }),
    type: 'warning'
  })
  if (!confirmed) return
  
  loading.value = true
  try {
    const result = await window.go.main.App.SwitchToBackup(name)
    if (result.success) {
      showToast(t('message.restartKiro'), 'success')
      await loadBackups()
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    loading.value = false
  }
}

const restoreOriginal = async () => {
  const confirmed = await showConfirmDialog({
    title: t('dialog.warningTitle'),
    message: t('message.confirmRestore'),
    type: 'warning'
  })
  if (!confirmed) return
  
  loading.value = true
  try {
    const result = await window.go.main.App.RestoreSoftReset()
    if (result.success) {
      showToast(t('message.restartKiro'), 'success')
      await loadBackups()
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    loading.value = false
  }
}

const resetToNew = async () => {
  // 首次使用時顯示提示 Modal
  if (!hasUsedReset.value) {
    showFirstTimeResetModal.value = true
    return
  }
  
  const confirmed = await showConfirmDialog({
    title: t('dialog.warningTitle'),
    message: t('message.confirmReset'),
    type: 'warning'
  })
  if (!confirmed) return
  
  await executeReset()
}

const executeReset = async () => {
  resetting.value = true
  try {
    const result = await window.go.main.App.SoftResetToNewMachine()
    
    if (result.success) {
      showToast(result.message, 'success')
      // 標記已使用過一鍵新機
      hasUsedReset.value = true
      localStorage.setItem('kiro-manager-has-used-reset', 'true')
      await loadBackups()
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    resetting.value = false
  }
}

const confirmFirstTimeReset = async () => {
  showFirstTimeResetModal.value = false
  const confirmed = await showConfirmDialog({
    title: t('dialog.warningTitle'),
    message: t('message.confirmReset'),
    type: 'warning'
  })
  if (!confirmed) return
  await executeReset()
}

const deleteBackup = async (name: string) => {
  const confirmed = await showConfirmDialog({
    title: t('dialog.deleteTitle'),
    message: t('message.confirmDelete', { name }),
    type: 'danger'
  })
  if (!confirmed) return
  
  loading.value = true
  try {
    const result = await window.go.main.App.DeleteBackup(name)
    if (result.success) {
      showToast(t('message.success'), 'success')
      await loadBackups()
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    loading.value = false
  }
}

const regenerateMachineID = async (name: string) => {
  const confirmed = await showConfirmDialog({
    title: t('dialog.confirmTitle'),
    message: t('message.confirmRegenerateId', { name }),
    type: 'warning'
  })
  if (!confirmed) return
  
  loading.value = true
  try {
    const result = await window.go.main.App.RegenerateMachineID(name)
    if (result.success) {
      showToast(t('message.regenerateIdSuccess'), 'success')
      await loadBackups()
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    loading.value = false
  }
}

// 截取機器碼 ID 的首兩節（例如 4fa2ec40-7c9e-... → 4fa2ec40-7c9e...）
const truncateMachineId = (machineId: string): string => {
  if (!machineId) return '-'
  const parts = machineId.split('-')
  if (parts.length >= 2) {
    return `${parts[0]}-${parts[1]}...`
  }
  // 如果不是 UUID 格式，顯示前 13 個字元
  return machineId.length > 13 ? `${machineId.substring(0, 13)}...` : machineId
}

// 複製機器碼 ID 到剪貼簿
const copyMachineId = async (machineId: string) => {
  if (!machineId) return
  try {
    await navigator.clipboard.writeText(machineId)
    copiedMachineId.value = machineId
    setTimeout(() => {
      copiedMachineId.value = null
    }, 2000)
  } catch (e) {
    console.error('Failed to copy machine ID:', e)
  }
}

// 判斷備份是否在冷卻期內（使用倒計時狀態）
const isInCooldown = (backupName: string): boolean => {
  return (countdownTimers.value[backupName] || 0) > 0
}

// 判斷當前帳號是否在冷卻期內
const isCurrentInCooldown = (): boolean => {
  return countdownCurrentAccount.value > 0
}

// 啟動倒計時
const startCountdown = (backupName: string) => {
  countdownTimers.value[backupName] = REFRESH_COOLDOWN_SECONDS
  const interval = setInterval(() => {
    if (countdownTimers.value[backupName] > 0) {
      countdownTimers.value[backupName]--
    } else {
      clearInterval(interval)
    }
  }, 1000)
}

// 啟動當前帳號的倒計時
const startCurrentCountdown = () => {
  countdownCurrentAccount.value = REFRESH_COOLDOWN_SECONDS
  const interval = setInterval(() => {
    if (countdownCurrentAccount.value > 0) {
      countdownCurrentAccount.value--
    } else {
      clearInterval(interval)
    }
  }, 1000)
}

const refreshBackupUsage = async (name: string) => {
  // 檢查冷卻期
  if (isInCooldown(name)) {
    return
  }

  const backup = backups.value.find(b => b.name === name)
  
  // 如果是當前帳號，也檢查當前帳號的冷卻狀態
  if (backup?.isCurrent && isCurrentInCooldown()) {
    return
  }
  refreshingBackup.value = name
  try {
    const result = await window.go.main.App.RefreshBackupUsage(name)
    if (result.success) {
      // 更新本地備份列表中的餘額資訊
      if (backup) {
        backup.subscriptionTitle = result.subscriptionTitle
        backup.usageLimit = result.usageLimit
        backup.currentUsage = result.currentUsage
        backup.balance = result.balance
        backup.isLowBalance = result.isLowBalance
        backup.isTokenExpired = result.isTokenExpired // 更新 token 過期狀態
        backup.cachedAt = result.cachedAt // 更新緩存時間
      }
      // 如果是當前帳號，也更新 currentUsageInfo 並同步倒計時
      if (backup?.isCurrent) {
        currentUsageInfo.value = {
          subscriptionTitle: result.subscriptionTitle,
          usageLimit: result.usageLimit,
          currentUsage: result.currentUsage,
          balance: result.balance,
          isLowBalance: result.isLowBalance
        }
        // 同時啟動當前帳號的倒計時
        startCurrentCountdown()
      }
      // 啟動備份的倒計時
      startCountdown(name)
    } else {
      showToast(result.message, 'error')
    }
  } catch (e) {
    showToast(t('message.refreshFailed'), 'error')
  } finally {
    refreshingBackup.value = null
  }
}

const refreshCurrentUsage = async () => {
  // 找到當前帳號對應的備份
  const currentBackup = backups.value.find(b => b.isCurrent)
  
  // 檢查冷卻期（當前帳號或對應備份任一在冷卻中都不允許刷新）
  if (isCurrentInCooldown() || (currentBackup && isInCooldown(currentBackup.name))) {
    return
  }

  if (currentBackup) {
    // 使用現有的 refreshBackupUsage 函數
    refreshingCurrent.value = true
    try {
      const result = await window.go.main.App.RefreshBackupUsage(currentBackup.name)
      if (result.success) {
        currentBackup.subscriptionTitle = result.subscriptionTitle
        currentBackup.usageLimit = result.usageLimit
        currentBackup.currentUsage = result.currentUsage
        currentBackup.balance = result.balance
        currentBackup.isLowBalance = result.isLowBalance
        currentBackup.isTokenExpired = result.isTokenExpired // 更新 token 過期狀態
        currentBackup.cachedAt = result.cachedAt // 更新緩存時間
        currentUsageInfo.value = {
          subscriptionTitle: result.subscriptionTitle,
          usageLimit: result.usageLimit,
          currentUsage: result.currentUsage,
          balance: result.balance,
          isLowBalance: result.isLowBalance
        }
        // 同時啟動當前帳號和對應備份的倒計時
        startCurrentCountdown()
        startCountdown(currentBackup.name)
      } else {
        showToast(result.message, 'error')
      }
    } catch (e) {
      showToast(t('message.refreshFailed'), 'error')
    } finally {
      refreshingCurrent.value = false
    }
  }
}

const openExtensionFolder = async () => {
  try {
    const result = await window.go.main.App.OpenExtensionFolder()
    if (!result.success) {
      showToast(result.message, 'error')
    }
  } catch (e) {
    console.error('Failed to open extension folder:', e)
  }
}

const openMachineIDFolder = async () => {
  try {
    const result = await window.go.main.App.OpenMachineIDFolder()
    if (!result.success) {
      showToast(result.message, 'error')
    }
  } catch (e) {
    console.error('Failed to open machine ID folder:', e)
  }
}

const openSSOCacheFolder = async () => {
  try {
    const result = await window.go.main.App.OpenSSOCacheFolder()
    if (!result.success) {
      showToast(result.message, 'error')
    }
  } catch (e) {
    console.error('Failed to open SSO cache folder:', e)
  }
}

const patchExtension = async () => {
  patching.value = true
  try {
    const result = await window.go.main.App.RepatchExtension()
    if (result.success) {
      showToast(result.message, 'success')
      // 更新軟重置狀態
      softResetStatus.value = await window.go.main.App.GetSoftResetStatus()
    } else {
      showToast(result.message, 'error')
    }
  } catch (e) {
    console.error('Failed to patch extension:', e)
    showToast(t('message.error'), 'error')
  } finally {
    patching.value = false
  }
}

onMounted(() => {
  // 語言已在 i18n/index.ts 中根據系統語言初始化
  // 這裡只需同步 locale 到當前組件（如果 localStorage 有值）
  const savedLang = localStorage.getItem('kiro-manager-lang')
  if (savedLang && ['zh-TW', 'zh-CN'].includes(savedLang)) {
    locale.value = savedLang
  }
  
  // 載入一鍵新機模式設定（硬一鍵新機暫時停用，強制使用軟一鍵新機）
  resetMode.value = 'soft'
  localStorage.setItem('kiro-manager-reset-mode', 'soft')
  
  // 載入是否已使用過一鍵新機
  hasUsedReset.value = localStorage.getItem('kiro-manager-has-used-reset') === 'true'
  
  loadBackups()
  
  // 每 5 秒檢查一次 Kiro 運行狀態
  setInterval(checkKiroStatus, 5000)
})
</script>

<template>
  <div class="flex h-screen bg-app-bg font-sans text-sm text-zinc-300">
    
    <!-- 左側邊欄 -->
    <aside class="w-[220px] flex-shrink-0 border-r border-app-border flex flex-col bg-[#0c0c0e]">
      <div class="h-16 flex items-center px-6 border-b border-app-border">
        <!-- Kiro Logo SVG -->
        <svg width="28" height="28" viewBox="0 0 200 200" xmlns="http://www.w3.org/2000/svg" class="mr-3 flex-shrink-0">
          <defs>
            <linearGradient id="bgGradient" x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" style="stop-color:#2b3245;stop-opacity:1" />
              <stop offset="100%" style="stop-color:#1e222e;stop-opacity:1" />
            </linearGradient>
            <linearGradient id="kGradient" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" style="stop-color:#61afef;stop-opacity:1" />
              <stop offset="100%" style="stop-color:#c678dd;stop-opacity:1" />
            </linearGradient>
            <filter id="dropShadow" x="-20%" y="-20%" width="140%" height="140%">
              <feGaussianBlur in="SourceAlpha" stdDeviation="3" />
              <feOffset dx="2" dy="4" result="offsetblur" />
              <feComponentTransfer>
                <feFuncA type="linear" slope="0.3" />
              </feComponentTransfer>
              <feMerge>
                <feMergeNode />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>
          <rect x="10" y="10" width="180" height="180" rx="40" ry="40" fill="url(#bgGradient)" stroke="#3e4451" stroke-width="2" />
          <circle cx="40" cy="40" r="6" fill="#ff5f56" />
          <circle cx="60" cy="40" r="6" fill="#ffbd2e" />
          <circle cx="80" cy="40" r="6" fill="#27c93f" />
          <g transform="translate(50, 70)" filter="url(#dropShadow)">
            <path d="M30 0 L0 40 L30 80" fill="none" stroke="url(#kGradient)" stroke-width="16" stroke-linecap="round" stroke-linejoin="round" />
            <line x1="35" y1="40" x2="75" y2="0" stroke="url(#kGradient)" stroke-width="16" stroke-linecap="round" />
            <line x1="35" y1="40" x2="65" y2="80" stroke="url(#kGradient)" stroke-width="16" stroke-linecap="round" />
            <rect x="85" y="70" width="20" height="10" fill="#98c379">
              <animate attributeName="opacity" values="1;0;1" dur="1s" repeatCount="indefinite" />
            </rect>
          </g>
        </svg>
        <span class="font-bold text-lg tracking-tight text-white">{{ t('app.name') }}</span>
      </div>
      
      <nav class="flex-1 p-4 space-y-1">
        <div 
          @click="activeMenu = 'dashboard'; showSettingsPanel = false"
          :class="[
            'px-3 py-2 rounded-lg flex items-center cursor-pointer transition-colors',
            activeMenu === 'dashboard' 
              ? 'text-zinc-100 bg-zinc-800/50 border border-zinc-700/50' 
              : 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900'
          ]"
        >
          <Icon name="Home" :class="['w-4 h-4 mr-3', activeMenu === 'dashboard' ? 'text-app-accent' : '']" />
          {{ t('menu.dashboard') }}
        </div>
        <div 
          @click="activeMenu = 'settings'; showSettingsPanel = true"
          :class="[
            'px-3 py-2 rounded-lg flex items-center cursor-pointer transition-colors',
            activeMenu === 'settings' 
              ? 'text-zinc-100 bg-zinc-800/50 border border-zinc-700/50' 
              : 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900'
          ]"
        >
          <Icon name="Settings" :class="['w-4 h-4 mr-3', activeMenu === 'settings' ? 'text-app-accent' : '']" />
          {{ t('menu.settings') }}
        </div>
      </nav>

    </aside>

    <!-- 右側主內容 -->
    <main class="flex-1 flex flex-col min-w-0 overflow-hidden bg-app-bg relative">
      <!-- 頂部標題列 -->
      <header class="h-16 border-b border-app-border flex items-center justify-between px-8 glass sticky top-0 z-10">
        <div>
          <h2 class="text-white font-semibold text-lg">{{ showSettingsPanel ? t('settings.title') : t('menu.dashboard') }}</h2>
          <p class="text-zinc-500 text-xs">{{ t('app.systemReady') }} • {{ t('app.version') }}</p>
        </div>
        <div class="flex items-center gap-2">
          <div :class="['w-2 h-2 rounded-full', loading ? 'bg-yellow-500 animate-pulse' : kiroRunning ? 'bg-green-500' : 'bg-zinc-500']"></div>
          <span class="text-xs text-zinc-400 font-mono">{{ loading ? t('app.processing') : kiroRunning ? t('app.kiroRunning') : t('app.kiroStopped') }}</span>
        </div>
      </header>

      <!-- 內容滾動區 -->
      <div class="flex-1 overflow-y-auto p-8 space-y-8">
        
        <!-- 設定面板 -->
        <div v-if="showSettingsPanel" class="space-y-6">
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-stretch">
            <!-- 左欄：Kiro 安裝路徑 + Kiro 版本號 -->
            <div class="flex flex-col gap-6">
              <!-- Kiro 安裝路徑設定 -->
              <div class="bg-zinc-900 border border-app-border rounded-xl p-6 flex-1 flex flex-col">
                <h4 class="text-zinc-300 font-medium mb-4 flex items-center">
                  <Icon name="FolderOpen" class="w-5 h-5 mr-2 text-zinc-400" />
                  {{ t('settings.kiroInstallPath') }}
                </h4>
                
                <p class="text-zinc-500 text-sm mb-4">{{ t('settings.kiroInstallPathDesc') }}</p>
                
                <div class="flex-1"></div>
                
                <!-- 狀態指示 -->
                <div class="flex items-center gap-2 mb-3">
                  <div :class="[
                    'w-2 h-2 rounded-full',
                    appSettings.customKiroInstallPath ? 'bg-app-accent' : 'bg-green-500'
                  ]"></div>
                  <span class="text-xs text-zinc-400">
                    {{ appSettings.customKiroInstallPath ? t('settings.usingCustomPath') : t('settings.usingAutoDetect') }}
                  </span>
                </div>
                
                <div class="flex gap-2">
                  <input 
                    type="text"
                    v-model="kiroInstallPathInput"
                    @input="onKiroInstallPathInput"
                    :placeholder="t('settings.kiroInstallPathPlaceholder')"
                    class="flex-1 px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-300 text-sm focus:outline-none focus:border-zinc-500 placeholder-zinc-600"
                  />
                  <button
                    v-if="kiroInstallPathModified"
                    @click="saveKiroInstallPath"
                    class="px-3 py-2 bg-app-accent hover:bg-app-accent/80 text-white rounded-lg text-sm transition-colors"
                  >
                    {{ t('backup.confirm') }}
                  </button>
                  <button
                    v-else
                    @click="detectKiroInstallPath"
                    :disabled="detectingPath"
                    class="px-3 py-2 bg-zinc-700 hover:bg-zinc-600 text-zinc-300 rounded-lg text-sm transition-colors disabled:opacity-50"
                  >
                    {{ detectingPath ? '...' : t('settings.detectPath') }}
                  </button>
                  <button
                    v-if="appSettings.customKiroInstallPath && !kiroInstallPathModified"
                    @click="clearKiroInstallPath"
                    class="px-3 py-2 bg-zinc-800 hover:bg-red-900/30 border border-zinc-700 hover:border-red-800/50 text-zinc-400 hover:text-red-400 rounded-lg text-sm transition-colors"
                  >
                    {{ t('settings.clearPath') }}
                  </button>
                </div>
              </div>
              
              <!-- Kiro 版本號設定 -->
              <div class="bg-zinc-900 border border-app-border rounded-xl p-6 flex-1 flex flex-col">
                <h4 class="text-zinc-300 font-medium mb-4 flex items-center">
                  <Icon name="Tag" class="w-5 h-5 mr-2 text-zinc-400" />
                  {{ t('settings.kiroVersion') }}
                  <!-- 自動偵測狀態指示 -->
                  <span 
                    v-if="appSettings.useAutoDetect" 
                    class="ml-3 px-2 py-0.5 rounded text-[10px] bg-app-success/20 text-app-success border border-app-success/30"
                  >
                    {{ t('settings.autoDetectActive') }}
                  </span>
                </h4>
                
                <p class="text-zinc-500 text-sm mb-4">{{ t('settings.kiroVersionDesc') }}</p>
                
                <div class="flex-1"></div>
                
                <div class="flex items-center gap-2">
                  <input 
                    v-model="kiroVersionInput"
                    @input="onKiroVersionInput"
                    type="text"
                    :placeholder="t('settings.kiroVersionPlaceholder')"
                    class="flex-1 px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-200 text-sm font-mono focus:outline-none focus:border-app-accent transition-colors"
                  />
                  <button 
                    @click="detectKiroVersion"
                    :disabled="detectingVersion || appSettings.useAutoDetect"
                    class="px-3 py-2 bg-zinc-700 hover:bg-zinc-600 disabled:opacity-50 disabled:cursor-not-allowed text-zinc-200 rounded-lg text-sm transition-colors flex items-center gap-2"
                  >
                    <Icon v-if="detectingVersion" name="RefreshCw" class="w-4 h-4 animate-spin" />
                    <Icon v-else name="Search" class="w-4 h-4" />
                    {{ t('settings.detectVersion') }}
                  </button>
                  <button 
                    @click="saveKiroVersion"
                    :disabled="!kiroVersionModified"
                    class="px-3 py-2 bg-app-accent hover:bg-app-accent/80 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg text-sm transition-colors"
                  >
                    {{ t('backup.confirm') }}
                  </button>
                </div>
              </div>
            </div>
          
            <!-- 右欄：介面語言 + 低餘額設置 -->
            <div class="flex flex-col gap-6">
              <!-- 語言設定 -->
              <div class="bg-zinc-900 border border-app-border rounded-xl p-6 flex-1 flex flex-col">
                <h4 class="text-zinc-300 font-medium mb-4 flex items-center">
                  <Icon name="Globe" class="w-5 h-5 mr-2 text-zinc-400" />
                  {{ t('settings.language') }}
                </h4>
                
                <div class="flex-1"></div>
                
                <div class="flex gap-3">
                  <button 
                    v-for="lang in ['zh-TW', 'zh-CN']" 
                    :key="lang"
                    @click="switchLanguage(lang)"
                    :class="[
                      'flex-1 py-3 px-4 rounded-lg border transition-all text-sm',
                      locale === lang 
                        ? 'bg-zinc-800 border-zinc-600 text-zinc-200' 
                        : 'border-zinc-700 hover:border-zinc-600 text-zinc-400 hover:text-zinc-300'
                    ]"
                  >
                    {{ lang === 'zh-TW' ? t('language.zhTW') : t('language.zhCN') }}
                  </button>
                </div>
              </div>
              
              <!-- 低餘額閾值設定 -->
              <div class="bg-zinc-900 border border-app-border rounded-xl p-6 flex-1 flex flex-col">
                <h4 class="text-zinc-300 font-medium mb-4 flex items-center">
                  <Icon name="AlertTriangle" class="w-5 h-5 mr-2 text-zinc-400" />
                  {{ t('settings.lowBalanceThreshold') }}
                </h4>
                
                <p class="text-zinc-500 text-sm mb-4">{{ t('settings.lowBalanceThresholdDesc') }}</p>
                
                <div class="flex-1"></div>
                
                <div class="flex items-center gap-4">
                  <input 
                    type="range" 
                    min="0" 
                    max="100" 
                    step="5"
                    :value="thresholdPreview"
                    @input="(e) => thresholdPreview = Number((e.target as HTMLInputElement).value)"
                    @change="(e) => saveLowBalanceThreshold(Number((e.target as HTMLInputElement).value) / 100)"
                    class="flex-1 h-2 bg-zinc-700 rounded-lg appearance-none cursor-pointer accent-app-accent"
                  />
                  <span class="text-zinc-200 font-mono text-sm w-12 text-right">
                    {{ thresholdPreview }}%
                  </span>
                </div>
                
                <div class="flex justify-between text-xs text-zinc-500 mt-2">
                  <span>0%</span>
                  <span>50%</span>
                  <span>100%</span>
                </div>
              </div>
            </div>
          </div>
        </div>
        
        <!-- Dashboard 內容 -->
        <div v-else class="space-y-8">
        
        <!-- 當前狀態 + 操作按鈕 -->
        <div class="grid grid-cols-1 lg:grid-cols-5 gap-6">
          
          <!-- 當前狀態卡片 -->
          <div class="lg:col-span-3 bg-gradient-to-br from-zinc-900 to-zinc-900/50 border border-app-border rounded-xl p-6 relative overflow-hidden group">
            <!-- 背景圖標：根據 Provider 動態顯示，點擊打開 SSO Cache 文件夾 -->
            <div 
              @click="openSSOCacheFolder"
              class="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 hover:!opacity-40 transition-opacity cursor-pointer z-20"
              :title="t('status.openSSOCache')"
            >
              <!-- 有 activeBackup 時顯示備份的 provider 圖標 -->
              <template v-if="activeBackup">
                <Icon v-if="activeBackup.provider === 'Github'" name="Github" class="w-32 h-32 text-white pointer-events-none" />
                <Icon v-else-if="activeBackup.provider === 'AWS' || activeBackup.provider === 'BuilderId'" name="AWS" class="w-32 h-32 text-white pointer-events-none" />
                <Icon v-else-if="activeBackup.provider === 'Google'" name="Google" class="w-32 h-32 text-white pointer-events-none" />
                <Icon v-else name="Cpu" class="w-32 h-32 text-white pointer-events-none" />
              </template>
              <!-- 沒有 activeBackup（原始機器）時顯示當前登入的 provider 圖標 -->
              <template v-else>
                <Icon v-if="currentProvider === 'Github'" name="Github" class="w-32 h-32 text-white pointer-events-none" />
                <Icon v-else-if="currentProvider === 'AWS' || currentProvider === 'BuilderId'" name="AWS" class="w-32 h-32 text-white pointer-events-none" />
                <Icon v-else-if="currentProvider === 'Google'" name="Google" class="w-32 h-32 text-white pointer-events-none" />
                <Icon v-else name="Cpu" class="w-32 h-32 text-white pointer-events-none" />
              </template>
            </div>
            
            <div class="relative z-10">
              <div class="flex items-center gap-2 mb-4">
                <span class="px-2 py-0.5 rounded text-[10px] font-bold bg-app-warning text-black uppercase tracking-wider">
                  {{ t('status.current') }}
                </span>
                <!-- 顯示當前帳號訂閱和餘額 -->
                <template v-if="currentUsageInfo">
                  <span class="px-2 py-0.5 rounded text-[10px] bg-app-accent/20 text-app-accent border border-app-accent/30 font-medium">
                    {{ currentUsageInfo.subscriptionTitle }}
                  </span>
                  <span 
                    :class="[
                      'text-xs font-mono',
                      currentUsageInfo.isLowBalance ? 'text-app-warning' : 'text-zinc-400'
                    ]"
                  >
                    <span v-if="currentUsageInfo.isLowBalance" class="inline-flex items-center gap-1">
                      <Icon name="AlertTriangle" class="w-3 h-3" />
                      {{ Math.round(currentUsageInfo.balance) }} / {{ Math.round(currentUsageInfo.usageLimit) }}
                    </span>
                    <span v-else>
                      {{ Math.round(currentUsageInfo.balance) }} / {{ Math.round(currentUsageInfo.usageLimit) }}
                    </span>
                  </span>
                  <!-- 刷新按鈕 / 倒計時 -->
                  <button
                    @click="refreshCurrentUsage"
                    :disabled="refreshingCurrent || isCurrentInCooldown()"
                    :class="[
                      'w-[22px] h-[22px] rounded transition-all inline-flex items-center justify-center',
                      refreshingCurrent
                        ? 'text-app-accent cursor-wait'
                        : isCurrentInCooldown()
                          ? 'text-zinc-500 cursor-not-allowed'
                          : backups.find(b => b.isCurrent)?.isTokenExpired
                            ? 'text-app-warning hover:text-amber-400'
                            : 'text-zinc-500 hover:text-zinc-300'
                    ]"
                    :title="backups.find(b => b.isCurrent)?.isTokenExpired
                      ? t('message.tokenExpiredTip')
                      : t('backup.refresh')"
                  >
                    <!-- 刷新中：旋轉圖標 -->
                    <Icon 
                      v-if="refreshingCurrent"
                      name="RefreshCw" 
                      class="w-3.5 h-3.5 animate-spin" 
                    />
                    <!-- 倒計時數字 -->
                    <span 
                      v-else-if="isCurrentInCooldown()" 
                      class="text-xs font-mono font-medium leading-none"
                    >
                      {{ countdownCurrentAccount }}
                    </span>
                    <!-- 正常狀態：靜態圖標 -->
                    <Icon 
                      v-else
                      name="RefreshCw" 
                      class="w-3.5 h-3.5" 
                    />
                  </button>
                </template>
              </div>
              
              <h3 class="text-3xl font-bold text-white mb-1 glow-text">
                {{ currentEnvironmentName || t('status.originalMachine') }}
              </h3>
              <div class="flex items-center gap-2 text-app-accent font-mono text-sm mb-6">
                <Icon name="Check" class="w-4 h-4" />
                {{ currentMachineId || '-' }}
              </div>

              <div class="flex gap-3">
                <button 
                  @click="showCreateModal = true"
                  class="flex items-center px-4 py-2 bg-zinc-800 hover:bg-zinc-700 border border-zinc-600 text-zinc-200 rounded-lg text-sm transition-all active:scale-95"
                >
                  <Icon name="Save" class="w-4 h-4 mr-2" />
                  {{ t('backup.create') }}
                </button>
                <button 
                  @click="restoreOriginal"
                  class="flex items-center px-4 py-2 bg-zinc-800/50 hover:bg-red-900/30 border border-zinc-700/50 hover:border-red-800/50 text-zinc-400 hover:text-red-400 rounded-lg text-sm transition-all"
                >
                  <Icon name="Rotate" class="w-4 h-4 mr-2" />
                  {{ t('restore.original') }}
                </button>
              </div>
            </div>
          </div>

          <!-- PATCH 狀態 + 一鍵新機合併卡片 -->
          <div class="lg:col-span-2 bg-zinc-900 border border-app-border rounded-xl p-4 flex flex-col">
            <!-- 上方：PATCH 狀態 -->
            <div class="flex items-center gap-2 mb-3">
              <Icon name="Cpu" class="w-4 h-4 text-zinc-400" />
              <span class="text-zinc-400 text-xs font-semibold uppercase tracking-wider">{{ t('status.patchStatus') }}</span>
            </div>
            
            <div class="space-y-2 mb-4">
              <!-- Patch 狀態 -->
              <div class="flex items-center justify-between">
                <span class="text-zinc-500 text-sm">Extension Patch</span>
                <div class="flex items-center gap-2">
                  <!-- 已 Patch：顯示靜態標籤 -->
                  <span 
                    v-if="softResetStatus.isPatched"
                    class="px-2 py-0.5 rounded text-xs font-medium bg-app-success/20 text-app-success border border-app-success/30"
                  >
                    {{ t('status.patched') }}
                  </span>
                  <!-- 未 Patch：顯示可點擊按鍵 -->
                  <button
                    v-else
                    @click="patchExtension"
                    :disabled="patching"
                    :class="[
                      'px-2 py-0.5 rounded text-xs font-medium transition-all',
                      patching
                        ? 'bg-zinc-700/50 text-zinc-500 border border-zinc-600/30 cursor-wait'
                        : 'bg-app-warning/20 text-app-warning border border-app-warning/30 hover:bg-app-warning/30 cursor-pointer'
                    ]"
                    :title="t('status.clickToPatch')"
                  >
                    <span v-if="patching" class="flex items-center gap-1">
                      <Icon name="Loader" class="w-3 h-3 animate-spin" />
                      {{ t('status.patching') }}
                    </span>
                    <span v-else>{{ t('status.notPatched') }}</span>
                  </button>
                  <button
                    v-if="softResetStatus.extensionPath"
                    @click="openExtensionFolder"
                    class="p-1 rounded text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50 transition-colors"
                    :title="t('status.openFolder')"
                  >
                    <Icon name="FolderOpen" class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
              
              <!-- 自訂 ID 狀態 -->
              <div class="flex items-center justify-between">
                <span class="text-zinc-500 text-sm">Machine ID</span>
                <div class="flex items-center gap-2">
                  <span :class="[
                    'px-2 py-0.5 rounded text-xs font-medium',
                    softResetStatus.hasCustomId 
                      ? 'bg-app-accent/20 text-app-accent border border-app-accent/30' 
                      : 'bg-zinc-700/50 text-zinc-400 border border-zinc-600/30'
                  ]">
                    {{ softResetStatus.hasCustomId ? t('status.hasCustomId') : t('status.noCustomId') }}
                  </span>
                  <button
                    @click="openMachineIDFolder"
                    class="p-1 rounded text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50 transition-colors"
                    :title="t('status.openFolder')"
                  >
                    <Icon name="FolderOpen" class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
              
              <!-- 總體狀態指示 -->
              <div class="flex items-center gap-2 pt-1">
                <div :class="[
                  'w-2 h-2 rounded-full',
                  softResetStatus.isPatched && softResetStatus.hasCustomId 
                    ? 'bg-app-success shadow-[0_0_6px_rgba(34,197,94,0.6)]' 
                    : 'bg-zinc-500'
                ]"></div>
                <span :class="[
                  'text-xs font-medium',
                  softResetStatus.isPatched && softResetStatus.hasCustomId 
                    ? 'text-app-success' 
                    : 'text-zinc-500'
                ]">
                  {{ softResetStatus.isPatched && softResetStatus.hasCustomId ? t('status.softResetActive') : t('status.softResetInactive') }}
                </span>
              </div>
            </div>
            
            <!-- 下方：一鍵新機按鈕 -->
            <button 
              @click="resetToNew"
              :disabled="resetting"
              :class="[
                'mt-auto relative group flex items-center justify-center gap-3 px-4 py-3 border rounded-lg transition-all',
                resetting 
                  ? 'bg-app-accent border-app-accent cursor-wait' 
                  : 'bg-zinc-800 hover:bg-app-accent border-zinc-700 hover:border-app-accent active:scale-95'
              ]"
            >
              <!-- 一鍵新機 SVG Icon -->
              <svg width="32" height="32" viewBox="0 0 100 100" fill="none" xmlns="http://www.w3.org/2000/svg" class="flex-shrink-0">
                <!-- 手機主體 (靜態) -->
                <rect x="25" y="15" width="50" height="80" rx="6" stroke="currentColor" stroke-width="4" fill="none"/>
                <line x1="42" y1="22" x2="58" y2="22" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
                <circle cx="50" cy="85" r="3" fill="currentColor"/>
                
                <!-- 進行中：只顯示持續旋轉的箭頭 -->
                <g v-if="resetting">
                  <path d="M50 40 A 15 15 0 1 1 38 63" stroke="currentColor" stroke-width="3" stroke-linecap="round" fill="none" />
                  <path d="M38 63 L34 58 M38 63 L43 59" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>
                  <animateTransform attributeName="transform" type="rotate" from="0 50 55" to="360 50 55" dur="0.6s" repeatCount="indefinite" />
                </g>
                
                <!-- 靜態：不旋轉的箭頭 -->
                <g v-else>
                  <path d="M50 40 A 15 15 0 1 1 38 63" stroke="currentColor" stroke-width="3" stroke-linecap="round" fill="none" />
                  <path d="M38 63 L34 58 M38 63 L43 59" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>
                </g>
              </svg>
              <div class="text-left">
                <span :class="['text-sm font-bold block', resetting ? 'text-white' : 'text-zinc-200 group-hover:text-white']">
                  {{ resetting ? t('app.processing') : t('restore.reset') }}
                </span>
                <span :class="['text-[10px]', resetting ? 'text-zinc-200' : 'text-zinc-500 group-hover:text-zinc-300']">
                  {{ resetting ? t('message.restartKiro') : t('restore.resetDesc') }}
                </span>
              </div>
            </button>
          </div>
        </div>

        <!-- 表格區域 -->
        <div>
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-zinc-400 text-sm font-semibold flex items-center">
              <Icon name="Database" class="w-4 h-4 mr-2" />
              {{ t('backup.list') }}
            </h3>
            <div class="relative">
              <Icon name="Search" class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
              <input 
                v-model="searchQuery"
                :placeholder="t('backup.search')"
                class="pl-9 pr-4 py-1.5 bg-zinc-900 border border-zinc-700 rounded-lg text-zinc-200 text-sm focus:outline-none focus:border-app-accent transition-colors w-48"
              />
            </div>
          </div>
          
          <div class="bg-app-surface border border-app-border rounded-xl overflow-hidden shadow-xl">
            <table class="w-full text-left border-collapse">
              <thead>
                <tr class="border-b border-zinc-800 bg-zinc-900/50 text-zinc-500 text-xs uppercase tracking-wider">
                  <th class="px-6 py-4 font-medium">{{ t('backup.name') }}</th>
                  <th class="px-6 py-4 font-medium">{{ t('backup.provider') }}</th>
                  <th class="px-6 py-4 font-medium">{{ t('backup.subscription') }}</th>
                  <th class="px-6 py-4 font-medium">{{ t('backup.balance') }}</th>
                  <th class="px-6 py-4 font-medium">{{ t('backup.machineId') }}</th>
                  <th class="px-6 py-4 font-medium text-right">{{ t('backup.actions') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-zinc-800/50">
                <tr v-if="filteredBackups.length === 0">
                  <td colspan="6" class="px-6 py-12 text-center text-zinc-500">{{ t('backup.noBackups') }}</td>
                </tr>
                <tr 
                  v-for="backup in filteredBackups" 
                  :key="backup.name"
                  :class="['group transition-colors', backup.isCurrent ? 'bg-app-accent/5' : 'hover:bg-zinc-800/30']"
                >
                  <td class="px-6 py-4">
                    <div class="flex items-center">
                      <div v-if="backup.isCurrent" class="w-1.5 h-1.5 rounded-full bg-app-warning mr-3 shadow-[0_0_8px_rgba(245,158,11,0.8)]"></div>
                      <span :class="['font-medium', backup.isCurrent ? 'text-white' : 'text-zinc-400 group-hover:text-zinc-300']">
                        {{ backup.name }}
                      </span>
                      <span v-if="backup.isOriginalMachine" class="ml-2 px-1.5 py-0.5 rounded text-[10px] bg-app-accent/20 text-app-accent border border-app-accent/30">
                        {{ t('backup.original') }}
                      </span>
                    </div>
                  </td>
                  <td class="px-6 py-4">
                    <span class="px-2 py-1 rounded text-[10px] bg-zinc-800 text-zinc-400 border border-zinc-700 inline-flex items-center gap-1.5">
                      <Icon v-if="backup.provider === 'Github'" name="Github" class="w-3.5 h-3.5" />
                      <Icon v-else-if="backup.provider === 'AWS' || backup.provider === 'BuilderId'" name="AWS" class="w-3.5 h-3.5" />
                      <Icon v-else-if="backup.provider === 'Google'" name="Google" class="w-3.5 h-3.5" />
                      {{ backup.provider || '-' }}
                    </span>
                  </td>
                  <!-- 訂閱類型 (Requirements: 3.3) -->
                  <td class="px-6 py-4">
                    <span 
                      v-if="backup.subscriptionTitle"
                      class="px-2 py-1 rounded text-[10px] bg-app-accent/20 text-app-accent border border-app-accent/30 font-medium"
                    >
                      {{ backup.subscriptionTitle }}
                    </span>
                    <span v-else class="text-zinc-500">-</span>
                  </td>
                  <!-- 餘額 (Requirements: 3.1, 3.2) -->
                  <td class="px-6 py-4">
                    <div class="flex items-center gap-2">
                      <span 
                        v-if="backup.usageLimit > 0"
                        :class="[
                          'font-mono text-xs',
                          backup.isLowBalance 
                            ? 'text-app-warning' 
                            : 'text-zinc-400'
                        ]"
                      >
                        <span v-if="backup.isLowBalance" class="inline-flex items-center gap-1">
                          <Icon name="AlertTriangle" class="w-3 h-3" />
                          {{ Math.round(backup.balance) }} / {{ Math.round(backup.usageLimit) }}
                        </span>
                        <span v-else>
                          {{ Math.round(backup.balance) }} / {{ Math.round(backup.usageLimit) }}
                        </span>
                      </span>
                      <span v-else class="text-zinc-500">-</span>
                      <!-- 刷新按鈕 / 倒計時 -->
                      <template v-if="backup.hasToken">
                        <button
                          @click="refreshBackupUsage(backup.name)"
                          :disabled="refreshingBackup === backup.name || isInCooldown(backup.name)"
                          :class="[
                            'w-[26px] h-[26px] rounded transition-all inline-flex items-center justify-center',
                            refreshingBackup === backup.name
                              ? 'text-app-accent cursor-wait'
                              : isInCooldown(backup.name)
                                ? 'text-zinc-500 cursor-not-allowed'
                                : backup.isTokenExpired
                                  ? 'text-app-warning hover:text-amber-400 hover:bg-zinc-700/50'
                                  : 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50'
                          ]"
                          :title="backup.isTokenExpired
                            ? t('message.tokenExpiredTip')
                            : t('backup.refresh')"
                        >
                          <!-- 倒計時數字 -->
                          <span 
                            v-if="isInCooldown(backup.name) && refreshingBackup !== backup.name" 
                            class="text-xs font-mono font-medium leading-none"
                          >
                            {{ countdownTimers[backup.name] }}
                          </span>
                          <!-- 刷新圖標 -->
                          <Icon 
                            v-else
                            name="RefreshCw" 
                            :class="['w-3.5 h-3.5', refreshingBackup === backup.name ? 'animate-spin' : '']" 
                          />
                        </button>
                      </template>
                    </div>
                  </td>
                  <td class="px-6 py-4">
                    <button
                      v-if="backup.machineId"
                      @click="copyMachineId(backup.machineId)"
                      class="font-mono text-xs text-zinc-500 hover:text-zinc-300 cursor-pointer transition-colors inline-flex items-center gap-1.5 group"
                      :title="backup.machineId"
                    >
                      <span>{{ truncateMachineId(backup.machineId) }}</span>
                      <Icon 
                        :name="copiedMachineId === backup.machineId ? 'Check' : 'Copy'" 
                        :class="[
                          'w-3 h-3 transition-all',
                          copiedMachineId === backup.machineId 
                            ? 'text-app-success' 
                            : 'opacity-0 group-hover:opacity-100 text-zinc-400'
                        ]" 
                      />
                    </button>
                    <span v-else class="font-mono text-xs text-zinc-500">-</span>
                  </td>
                  <td class="px-6 py-4 text-right">
                    <div v-if="backup.isCurrent" class="text-app-warning text-xs font-bold flex items-center justify-end gap-1">
                      <div class="w-1 h-1 bg-app-warning rounded-full animate-ping"></div>
                      {{ t('status.active') }}
                    </div>
                    <div v-else class="flex items-center justify-end gap-2">
                      <button 
                        @click="switchToBackup(backup.name)"
                        class="text-xs bg-transparent border border-zinc-700 hover:border-zinc-500 text-zinc-400 hover:text-white px-3 py-1.5 rounded transition-all"
                      >
                        {{ t('backup.switchTo') }}
                      </button>
                      <button 
                        @click="regenerateMachineID(backup.name)"
                        class="text-xs bg-transparent border border-zinc-700 hover:border-app-accent text-zinc-400 hover:text-app-accent px-2 py-1.5 rounded transition-all"
                        :title="t('backup.regenerateId')"
                      >
                        <Icon name="RefreshCw" class="w-3 h-3" />
                      </button>
                      <button 
                        @click="deleteBackup(backup.name)"
                        class="text-xs bg-transparent border border-zinc-700 hover:border-red-700 text-zinc-400 hover:text-red-400 px-2 py-1.5 rounded transition-all"
                      >
                        <Icon name="Trash" class="w-3 h-3" />
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        
        </div>
      </div>

      <!-- Loading 遮罩 -->
      <div v-if="loading" class="absolute inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center">
        <div class="flex flex-col items-center">
          <div class="w-10 h-10 border-4 border-app-accent border-t-transparent rounded-full animate-spin mb-4"></div>
          <span class="text-white text-sm font-medium tracking-widest">PROCESSING</span>
        </div>
      </div>
    </main>

    <!-- Create Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showCreateModal = false">
      <div class="bg-app-surface border border-app-border rounded-xl p-6 min-w-[400px] shadow-2xl">
        <h3 class="text-white font-semibold text-lg mb-4">{{ t('backup.createTitle') }}</h3>
        <div class="mb-4">
          <label class="block text-zinc-400 text-sm mb-2">{{ t('backup.nameLabel') }}</label>
          <input 
            v-model="newBackupName" 
            :placeholder="t('backup.namePlaceholder')"
            @keyup.enter="createBackup"
            class="w-full px-4 py-2 bg-zinc-900 border border-zinc-700 rounded-lg text-zinc-200 text-sm focus:outline-none focus:border-app-accent transition-colors"
          />
        </div>
        <div class="flex justify-end gap-3">
          <button 
            @click="showCreateModal = false"
            class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-sm transition-colors"
          >
            {{ t('backup.cancel') }}
          </button>
          <button 
            @click="createBackup"
            class="px-4 py-2 bg-app-accent hover:bg-app-accent/80 text-white rounded-lg text-sm transition-colors"
          >
            {{ t('backup.confirm') }}
          </button>
        </div>
      </div>
    </div>

    <!-- First Time Reset Modal -->
    <div v-if="showFirstTimeResetModal" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showFirstTimeResetModal = false">
      <div class="bg-app-surface border border-app-border rounded-xl p-6 max-w-md shadow-2xl">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 rounded-full bg-app-accent/20 flex items-center justify-center">
            <Icon name="Sparkles" class="w-5 h-5 text-app-accent" />
          </div>
          <h3 class="text-white font-semibold text-lg">{{ t('message.firstTimeResetTitle') }}</h3>
        </div>
        
        <div class="space-y-4 mb-6">
          <p class="text-zinc-300 text-sm leading-relaxed">
            {{ t('message.firstTimeResetInfo') }}
          </p>
          <div class="bg-zinc-800/50 border border-zinc-700 rounded-lg p-3">
            <p class="text-zinc-400 text-xs leading-relaxed">
              💡 {{ t('message.firstTimeResetTip') }}
            </p>
          </div>
        </div>
        
        <div class="flex justify-end gap-3">
          <button 
            @click="showFirstTimeResetModal = false"
            class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-sm transition-colors"
          >
            {{ t('backup.cancel') }}
          </button>
          <button 
            @click="confirmFirstTimeReset"
            class="px-4 py-2 bg-app-accent hover:bg-app-accent/80 text-white rounded-lg text-sm transition-colors"
          >
            {{ t('message.continueReset') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Confirm Dialog -->
    <div v-if="confirmDialog.show" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50" @click.self="confirmDialog.onCancel">
      <div class="bg-app-surface border border-app-border rounded-xl p-6 max-w-md shadow-2xl">
        <div class="flex items-center gap-3 mb-4">
          <div :class="[
            'w-10 h-10 rounded-full flex items-center justify-center',
            confirmDialog.type === 'danger' ? 'bg-app-danger/20' : confirmDialog.type === 'info' ? 'bg-app-accent/20' : 'bg-app-warning/20'
          ]">
            <Icon 
              :name="confirmDialog.type === 'danger' ? 'Trash' : confirmDialog.type === 'info' ? 'Info' : 'AlertTriangle'" 
              :class="[
                'w-5 h-5',
                confirmDialog.type === 'danger' ? 'text-app-danger' : confirmDialog.type === 'info' ? 'text-app-accent' : 'text-app-warning'
              ]" 
            />
          </div>
          <h3 class="text-white font-semibold text-lg">{{ confirmDialog.title }}</h3>
        </div>
        
        <p class="text-zinc-300 text-sm leading-relaxed mb-6">
          {{ confirmDialog.message }}
        </p>
        
        <div class="flex justify-end gap-3">
          <button 
            @click="confirmDialog.onCancel"
            class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-sm transition-colors"
          >
            {{ confirmDialog.cancelText }}
          </button>
          <button 
            @click="confirmDialog.onConfirm"
            :class="[
              'px-4 py-2 rounded-lg text-sm transition-colors',
              confirmDialog.type === 'danger' 
                ? 'bg-app-danger hover:bg-app-danger/80 text-white' 
                : 'bg-app-accent hover:bg-app-accent/80 text-white'
            ]"
          >
            {{ confirmDialog.confirmText }}
          </button>
        </div>
      </div>
    </div>

    <!-- Toast -->
    <Transition name="slide">
      <div 
        v-if="toast.show" 
        :class="[
          'fixed bottom-5 right-5 px-5 py-3 rounded-lg text-white text-sm z-50 shadow-lg',
          toast.type === 'success' ? 'bg-app-success' : 'bg-app-danger'
        ]"
      >
        {{ toast.message }}
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}
.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
  opacity: 0;
}
</style>
