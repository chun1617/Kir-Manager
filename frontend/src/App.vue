<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import Icon from './components/Icon.vue'
import OAuthLogin from './components/OAuthLogin.vue'
import TabBar from './components/settings/TabBar.vue'
import BasicSettingsTab from './components/settings/BasicSettingsTab.vue'
import AutoSwitchTab from './components/settings/AutoSwitchTab.vue'
import { useSettingsPage } from './composables/useSettingsPage'
import { SETTINGS_TABS } from './constants/settingsTabs'
import type { RefreshRule } from './types/refreshInterval'

const { t, locale } = useI18n()
const { activeTab, isTabDisabled, handleTabChange } = useSettingsPage()

// Debounce 工具函數
function debounce<T extends (...args: any[]) => any>(
  fn: T,
  delay: number
): (...args: Parameters<T>) => void {
  let timeoutId: ReturnType<typeof setTimeout> | null = null;
  return (...args: Parameters<T>) => {
    if (timeoutId) clearTimeout(timeoutId);
    timeoutId = setTimeout(() => fn(...args), delay);
  };
}

// 保存視窗大小（debounced 500ms）
const saveWindowSize = debounce(() => {
  const width = window.outerWidth;
  const height = window.outerHeight;
  // 調用後端 SaveWindowSize 函數（需要後端實作）
  if (window.go?.main?.App?.SaveWindowSize) {
    window.go.main.App.SaveWindowSize(width, height);
  }
}, 500);

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
  folderId: string           // 所屬文件夾 ID
}

interface FolderItem {
  id: string
  name: string
  createdAt: string
  order: number
  snapshotCount: number
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

interface RefreshIntervalRule {
  minBalance: number
  maxBalance: number
  interval: number
}

interface AutoSwitchSettings {
  enabled: boolean
  balanceThreshold: number
  minTargetBalance: number
  folderIds: string[]
  subscriptionTypes: string[]
  refreshIntervals: RefreshIntervalRule[]
  notifyOnSwitch: boolean
  notifyOnLowBalance: boolean
}

interface AutoSwitchStatus {
  status: string  // "stopped", "running", "cooldown"
  lastBalance: number
  cooldownRemaining: number
  switchCount: number
}

// PathDetectionResult 路徑偵測結果
interface PathDetectionResult {
  path: string
  success: boolean
  triedStrategies?: string[]
  failureReasons?: Record<string, string>
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
          GetKiroInstallPathWithStatus(): Promise<PathDetectionResult>
          OpenExtensionFolder(): Promise<Result>
          OpenMachineIDFolder(): Promise<Result>
          OpenSSOCacheFolder(): Promise<Result>
          RepatchExtension(): Promise<Result>
          SaveWindowSize(width: number, height: number): Promise<Result>
          // Auto Switch API
          GetAutoSwitchSettings(): Promise<AutoSwitchSettings>
          SaveAutoSwitchSettings(settings: AutoSwitchSettings): Promise<Result>
          StartAutoSwitchMonitor(): Promise<Result>
          StopAutoSwitchMonitor(): Promise<Result>
          GetAutoSwitchStatus(): Promise<AutoSwitchStatus>
          // Folder API
          GetFolderList(): Promise<FolderItem[]>
          CreateFolder(name: string): Promise<Result>
          RenameFolder(id: string, newName: string): Promise<Result>
          DeleteFolder(id: string, deleteSnapshots: boolean): Promise<Result>
          AssignSnapshotToFolder(snapshotName: string, folderId: string): Promise<Result>
          UnassignSnapshot(snapshotName: string): Promise<Result>
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
const filterSubscription = ref<string>('')
const filterProvider = ref<string>('')
const filterBalance = ref<string>('')
// 篩選下拉選單開關狀態
const openFilter = ref<string | null>(null)
// 主內容滾動區 ref（用於自動聚焦）
const mainScrollArea = ref<HTMLElement | null>(null)
const toast = ref<{ show: boolean; message: string; type: 'success' | 'error' | 'warning' }>({
  show: false,
  message: '',
  type: 'success'
})

// 一鍵新機模式相關
const resetMode = ref<'soft'>('soft')
const hasUsedReset = ref(false)

// 篩選下拉選單控制
const toggleFilter = (name: string) => {
  openFilter.value = openFilter.value === name ? null : name
}
const setFilterProvider = (value: string) => {
  filterProvider.value = value
  openFilter.value = null
}
const setFilterSubscription = (value: string) => {
  filterSubscription.value = value
  openFilter.value = null
}
const setFilterBalance = (value: string) => {
  filterBalance.value = value
  openFilter.value = null
}
const showFirstTimeResetModal = ref(false)
const showSettingsPanel = ref(false)
const activeMenu = ref<'dashboard' | 'settings' | 'oauth'>('dashboard')
const isMobileMenuOpen = ref(false) // 移動端菜單開關狀態
const resetting = ref(false) // 一鍵新機進行中狀態
const refreshingBackup = ref<string | null>(null) // 正在刷新餘額的備份名稱
const refreshingCurrent = ref(false) // 正在刷新當前帳號餘額
const patching = ref(false) // Extension Patch 進行中狀態

// 批量操作相關狀態
const selectedBackups = ref<Set<string>>(new Set())
const batchOperating = ref(false)
const showMoveToFolderDropdown = ref(false)

// 文件夾相關狀態
const folders = ref<FolderItem[]>([])
const expandedFolders = ref<Set<string>>(new Set())
const uncategorizedExpanded = ref(true)
const showCreateFolderModal = ref(false)
const newFolderName = ref('')
const creatingFolder = ref(false)
const renamingFolder = ref<string | null>(null)
const renameFolderName = ref('')
const deletingFolder = ref<string | null>(null)
const dragOverFolderId = ref<string | null>(null)
const dragOverUncategorized = ref(false)

// 刪除文件夾對話框狀態
const showDeleteFolderDialog = ref(false)
const folderToDelete = ref<FolderItem | null>(null)

// 獨立操作狀態（取代全屏 loading 覆蓋層）
const creatingBackup = ref(false)        // 建立備份進行中
const switchingBackup = ref<string | null>(null)  // 正在切換的備份名稱
const restoringOriginal = ref(false)     // 還原原始機器進行中
const deletingBackup = ref<string | null>(null)   // 正在刪除的備份名稱
const regeneratingId = ref<string | null>(null)   // 正在重新生成機器碼的備份名稱

// 刷新冷卻期（60 秒）
const REFRESH_COOLDOWN_SECONDS = 60
const copiedMachineId = ref<string | null>(null) // 剛複製的機器碼 ID（用於顯示提示）

// 倒計時狀態：key 為備份名稱，value 為剩餘秒數（0 表示無倒計時）
const countdownTimers = ref<Record<string, number>>({})
const countdownCurrentAccount = ref(0) // 當前帳號的倒計時秒數

// 重置狀態
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
  kiroVersion: '0.8.206',
  useAutoDetect: true,
  customKiroInstallPath: ''
})

// Kiro 版本號輸入值
const kiroVersionInput = ref('0.8.206')
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

// 自動切換設定
const autoSwitchSettings = ref<AutoSwitchSettings>({
  enabled: false,
  balanceThreshold: 5,
  minTargetBalance: 50,
  folderIds: [],
  subscriptionTypes: [],
  refreshIntervals: [],
  notifyOnSwitch: true,
  notifyOnLowBalance: true
})
const autoSwitchStatus = ref<AutoSwitchStatus>({
  status: 'stopped',
  lastBalance: 0,
  cooldownRemaining: 0,
  switchCount: 0
})
const savingAutoSwitch = ref(false)

// 刷新頻率規則轉換函數
// 後端 RefreshIntervalRule 無 id，前端 RefreshRule 需要 id
const toRefreshRules = (intervals: RefreshIntervalRule[]): RefreshRule[] => {
  return intervals.map((r, i) => ({
    id: `rule-${i}-${r.minBalance}-${r.maxBalance}`,
    minBalance: r.minBalance,
    maxBalance: r.maxBalance,
    interval: r.interval,
  }))
}

const toRefreshIntervals = (rules: RefreshRule[]): RefreshIntervalRule[] => {
  return rules.map(r => ({
    minBalance: r.minBalance,
    maxBalance: r.maxBalance,
    interval: r.interval,
  }))
}

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

  // 4. 文字搜尋（現有邏輯）
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

// 文件夾相關計算屬性
const uncategorizedBackups = computed(() => {
  return filteredBackups.value.filter(b => !b.folderId)
})

const getBackupsInFolder = (folderId: string) => {
  return filteredBackups.value.filter(b => b.folderId === folderId)
}

// 批量操作：切換單一選擇
const toggleSelect = (name: string) => {
  const newSet = new Set(selectedBackups.value)
  if (newSet.has(name)) {
    newSet.delete(name)
  } else {
    newSet.add(name)
  }
  selectedBackups.value = newSet
}

// 批量操作：全選/取消全選
const toggleSelectAll = () => {
  if (selectedBackups.value.size === filteredBackups.value.length) {
    selectedBackups.value = new Set()
  } else {
    selectedBackups.value = new Set(filteredBackups.value.map(b => b.name))
  }
}

// 計算屬性：是否全選
const isAllSelected = computed(() => 
  filteredBackups.value.length > 0 && 
  selectedBackups.value.size === filteredBackups.value.length
)

// 計算屬性：是否有選中項目
const hasSelection = computed(() => selectedBackups.value.size > 0)

// 批量刪除
const batchDelete = async () => {
  if (!hasSelection.value || batchOperating.value) return
  batchOperating.value = true
  try {
    for (const name of selectedBackups.value) {
      await window.go.main.App.DeleteBackup(name)
    }
    selectedBackups.value = new Set()
    await loadBackups(false)
    showToast(t('message.success'), 'success')
  } finally {
    batchOperating.value = false
  }
}

// 批量刷新機器碼
const batchRegenerateMachineID = async () => {
  if (!hasSelection.value || batchOperating.value) return
  batchOperating.value = true
  try {
    for (const name of selectedBackups.value) {
      await window.go.main.App.RegenerateMachineID(name)
    }
    selectedBackups.value = new Set()
    await loadBackups(false)
    showToast(t('message.success'), 'success')
  } finally {
    batchOperating.value = false
  }
}

// 批量刷新餘額（序列執行）
const batchRefreshUsage = async () => {
  if (!hasSelection.value || batchOperating.value) return
  batchOperating.value = true
  try {
    const toRefresh = [...selectedBackups.value].filter(name => !isInCooldown(name))
    for (const name of toRefresh) {
      await refreshBackupUsage(name)
    }
    selectedBackups.value = new Set()
    showToast(t('message.success'), 'success')
  } finally {
    batchOperating.value = false
  }
}

// 批量移動到文件夾
const batchMoveToFolder = async (folderId: string | null) => {
  if (selectedBackups.value.size === 0) return
  
  batchOperating.value = true
  showMoveToFolderDropdown.value = false
  
  try {
    const backupNames = Array.from(selectedBackups.value)
    for (const name of backupNames) {
      if (folderId) {
        await window.go.main.App.AssignSnapshotToFolder(name, folderId)
      } else {
        await window.go.main.App.UnassignSnapshot(name)
      }
    }
    
    // 清除選擇
    selectedBackups.value = new Set()
    
    // 刷新列表
    await loadBackups(false)
    await loadFolders()
    
    // 顯示成功訊息
    showToast(t('batch.moveSuccess', { count: backupNames.length }), 'success')
  } catch (e: any) {
    showToast(e.message || t('message.error'), 'error')
  } finally {
    batchOperating.value = false
  }
}

const switchLanguage = (lang: string) => {
  locale.value = lang
  localStorage.setItem('kiro-manager-lang', lang)
}

const showToast = (message: string, type: 'success' | 'error' | 'warning') => {
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

const loadBackups = async (showOverlay: boolean = true) => {
  if (showOverlay) {
    loading.value = true
  }
  try {
    backups.value = await window.go.main.App.GetBackupList() || []
    currentMachineId.value = await window.go.main.App.GetCurrentMachineID()
    currentEnvironmentName.value = await window.go.main.App.GetCurrentEnvironmentName()
    softResetStatus.value = await window.go.main.App.GetSoftResetStatus()
    currentProvider.value = await window.go.main.App.GetCurrentProvider()
    currentUsageInfo.value = await window.go.main.App.GetCurrentUsageInfo()
    appSettings.value = await window.go.main.App.GetSettings()
    thresholdPreview.value = Math.round(appSettings.value.lowBalanceThreshold * 100)
    kiroVersionInput.value = appSettings.value.kiroVersion || '0.8.206'
    kiroVersionModified.value = false // 重置修改狀態
    kiroInstallPathInput.value = appSettings.value.customKiroInstallPath || ''
    kiroInstallPathModified.value = false // 重置修改狀態
    await checkKiroStatus()
    await loadFolders() // 載入文件夾列表
    await loadAutoSwitchSettings() // 載入自動切換設定
  } catch (e) {
    console.error(e)
  } finally {
    if (showOverlay) {
      loading.value = false
    }
  }
}

// 自動切換相關方法
const loadAutoSwitchSettings = async () => {
  try {
    autoSwitchSettings.value = await window.go.main.App.GetAutoSwitchSettings()
    autoSwitchStatus.value = await window.go.main.App.GetAutoSwitchStatus()
  } catch (e) {
    console.error('Failed to load auto switch settings:', e)
  }
}

const saveAutoSwitchSettings = async () => {
  savingAutoSwitch.value = true
  try {
    const result = await window.go.main.App.SaveAutoSwitchSettings(autoSwitchSettings.value)
    if (result.success) {
      showToast(t('message.success'), 'success')
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    savingAutoSwitch.value = false
  }
}

const toggleAutoSwitch = async () => {
  // 先保存設定，確保後端有最新的 Enabled 狀態
  await saveAutoSwitchSettings()
  
  if (autoSwitchSettings.value.enabled) {
    const result = await window.go.main.App.StartAutoSwitchMonitor()
    if (!result.success) {
      showToast(result.message, 'error')
      autoSwitchSettings.value.enabled = false
      // 回滾設定
      await saveAutoSwitchSettings()
    }
  } else {
    await window.go.main.App.StopAutoSwitchMonitor()
  }
  
  // 刷新狀態顯示
  autoSwitchStatus.value = await window.go.main.App.GetAutoSwitchStatus()
}

// 處理 AutoSwitchTab 組件的 toggle 事件
const handleAutoSwitchToggle = async (enabled: boolean) => {
  autoSwitchSettings.value.enabled = enabled
  await toggleAutoSwitch()
}

// 自動切換篩選相關方法
const availableSubscriptionTypes = ['Free', 'Pro', 'Pro+', 'Enterprise']

const addAutoSwitchFolder = async (event: Event) => {
  const select = event.target as HTMLSelectElement
  const folderId = select.value
  if (folderId && !autoSwitchSettings.value.folderIds.includes(folderId)) {
    autoSwitchSettings.value.folderIds.push(folderId)
    await saveAutoSwitchSettings()
  }
  select.value = ''
}

const removeAutoSwitchFolder = async (folderId: string) => {
  autoSwitchSettings.value.folderIds = autoSwitchSettings.value.folderIds.filter(id => id !== folderId)
  await saveAutoSwitchSettings()
}

const addAutoSwitchSubscription = async (event: Event) => {
  const select = event.target as HTMLSelectElement
  const subType = select.value
  if (subType && !autoSwitchSettings.value.subscriptionTypes.includes(subType)) {
    autoSwitchSettings.value.subscriptionTypes.push(subType)
    await saveAutoSwitchSettings()
  }
  select.value = ''
}

const removeAutoSwitchSubscription = async (subType: string) => {
  autoSwitchSettings.value.subscriptionTypes = autoSwitchSettings.value.subscriptionTypes.filter(s => s !== subType)
  await saveAutoSwitchSettings()
}

// 刷新頻率規則相關方法
const addRefreshRule = () => {
  autoSwitchSettings.value.refreshIntervals.push({
    minBalance: 0,
    maxBalance: -1,
    interval: 60
  })
}

const removeRefreshRule = async (index: number) => {
  autoSwitchSettings.value.refreshIntervals.splice(index, 1)
  await saveAutoSwitchSettings()
}

// 文件夾相關方法
const loadFolders = async () => {
  try {
    folders.value = await window.go.main.App.GetFolderList() || []
  } catch (e) {
    console.error('Failed to load folders:', e)
  }
}

const toggleFolder = (folderId: string) => {
  const newSet = new Set(expandedFolders.value)
  if (newSet.has(folderId)) {
    newSet.delete(folderId)
  } else {
    newSet.add(folderId)
  }
  expandedFolders.value = newSet
}

const toggleUncategorized = () => {
  uncategorizedExpanded.value = !uncategorizedExpanded.value
}

const createFolder = async () => {
  if (!newFolderName.value.trim()) return
  creatingFolder.value = true
  try {
    const result = await window.go.main.App.CreateFolder(newFolderName.value.trim())
    if (result.success) {
      showToast(t('message.success'), 'success')
      showCreateFolderModal.value = false
      newFolderName.value = ''
      await loadFolders()
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    creatingFolder.value = false
  }
}

const startRenameFolder = (folder: FolderItem) => {
  renamingFolder.value = folder.id
  renameFolderName.value = folder.name
}

const confirmRenameFolder = async () => {
  if (!renamingFolder.value || !renameFolderName.value.trim()) return
  try {
    const result = await window.go.main.App.RenameFolder(renamingFolder.value, renameFolderName.value.trim())
    if (result.success) {
      showToast(t('message.success'), 'success')
      await loadFolders()
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    renamingFolder.value = null
    renameFolderName.value = ''
  }
}

const cancelRenameFolder = () => {
  renamingFolder.value = null
  renameFolderName.value = ''
}

const deleteFolder = async (folder: FolderItem) => {
  const backupsInFolder = getBackupsInFolder(folder.id)
  
  // 檢查是否包含當前使用中的快照
  if (backupsInFolder.some(b => b.isCurrent)) {
    showToast(t('folder.cannotDeleteActive'), 'error')
    return
  }
  
  if (backupsInFolder.length > 0) {
    // 非空文件夾，顯示專用對話框（提供「移到未分類」和「一併刪除」兩個選項）
    folderToDelete.value = folder
    showDeleteFolderDialog.value = true
  } else {
    // 空文件夾，直接刪除
    deletingFolder.value = folder.id
    try {
      const result = await window.go.main.App.DeleteFolder(folder.id, false)
      if (result.success) {
        showToast(t('message.success'), 'success')
        await loadFolders()
      } else {
        showToast(result.message, 'error')
      }
    } finally {
      deletingFolder.value = null
    }
  }
}

// 確認刪除文件夾（處理用戶選擇）
const confirmDeleteFolder = async (deleteSnapshots: boolean) => {
  if (!folderToDelete.value) return
  
  const folder = folderToDelete.value
  showDeleteFolderDialog.value = false
  deletingFolder.value = folder.id
  
  try {
    const result = await window.go.main.App.DeleteFolder(folder.id, deleteSnapshots)
    if (result.success) {
      showToast(t('message.success'), 'success')
      await Promise.all([loadFolders(), loadBackups(false)])
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    deletingFolder.value = null
    folderToDelete.value = null
  }
}

// 取消刪除文件夾對話框
const cancelDeleteFolder = () => {
  showDeleteFolderDialog.value = false
  folderToDelete.value = null
}

// 拖放功能
const onDragStart = (event: DragEvent, backupName: string) => {
  if (!event.dataTransfer) return
  
  // 如果拖放的項目在選中列表中，移動所有選中項目
  // 否則只移動當前項目
  let itemsToMove: string[]
  if (selectedBackups.value.has(backupName)) {
    itemsToMove = Array.from(selectedBackups.value)
  } else {
    itemsToMove = [backupName]
  }
  
  event.dataTransfer.setData('application/json', JSON.stringify(itemsToMove))
  event.dataTransfer.effectAllowed = 'move'
}

const onDragOver = (e: DragEvent) => {
  e.preventDefault()
  e.dataTransfer!.dropEffect = 'move'
}

const onDragEnterFolder = (folderId: string) => {
  dragOverFolderId.value = folderId
  dragOverUncategorized.value = false
}

const onDragLeaveFolder = () => {
  dragOverFolderId.value = null
}

const onDragEnterUncategorized = () => {
  dragOverUncategorized.value = true
  dragOverFolderId.value = null
}

const onDragLeaveUncategorized = () => {
  dragOverUncategorized.value = false
}

const onDropToFolder = async (event: DragEvent, folderId: string) => {
  event.preventDefault()
  dragOverFolderId.value = ''
  
  if (!event.dataTransfer) return
  
  const data = event.dataTransfer.getData('application/json')
  if (!data) return
  
  try {
    const itemsToMove: string[] = JSON.parse(data)
    
    for (const name of itemsToMove) {
      await window.go.main.App.AssignSnapshotToFolder(name, folderId)
    }
    
    // 清除批量選擇
    selectedBackups.value = new Set()
    
    await loadBackups(false)
    await loadFolders()
  } catch (e: any) {
    showToast(e.message || t('message.error'), 'error')
  }
}

const onDropToUncategorized = async (event: DragEvent) => {
  event.preventDefault()
  dragOverUncategorized.value = false
  
  if (!event.dataTransfer) return
  
  const data = event.dataTransfer.getData('application/json')
  if (!data) return
  
  try {
    const itemsToMove: string[] = JSON.parse(data)
    
    for (const name of itemsToMove) {
      await window.go.main.App.UnassignSnapshot(name)
    }
    
    // 清除批量選擇
    selectedBackups.value = new Set()
    
    await loadBackups(false)
    await loadFolders()
  } catch (e: any) {
    showToast(e.message || t('message.error'), 'error')
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
  
  creatingBackup.value = true
  try {
    const result = await window.go.main.App.CreateBackup(newBackupName.value.trim())
    if (result.success) {
      showToast(t('message.success'), 'success')
      showCreateModal.value = false
      newBackupName.value = ''
      await loadBackups(false)
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    creatingBackup.value = false
  }
}

const switchToBackup = async (name: string) => {
  switchingBackup.value = name
  try {
    const result = await window.go.main.App.SwitchToBackup(name)
    if (result.success) {
      showToast(t('message.successChange'), 'success')
      await loadBackups(false)
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    switchingBackup.value = null
  }
}

const restoreOriginal = async () => {
  const confirmed = await showConfirmDialog({
    title: t('dialog.warningTitle'),
    message: t('message.confirmRestore'),
    type: 'warning'
  })
  if (!confirmed) return
  
  restoringOriginal.value = true
  try {
    const result = await window.go.main.App.RestoreSoftReset()
    if (result.success) {
      showToast(t('message.successChange'), 'success')
      await loadBackups(false)
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    restoringOriginal.value = false
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
      await loadBackups(false)
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
  
  deletingBackup.value = name
  try {
    const result = await window.go.main.App.DeleteBackup(name)
    if (result.success) {
      showToast(t('message.success'), 'success')
      await loadBackups(false)
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    deletingBackup.value = null
  }
}

const regenerateMachineID = async (name: string) => {
  regeneratingId.value = name
  const startTime = Date.now()
  const MIN_ANIMATION_DURATION = 600 // 最少 600ms（3 次閃爍，每次 200ms）
  
  try {
    const result = await window.go.main.App.RegenerateMachineID(name)
    
    // 確保動畫至少播放 3 次
    const elapsed = Date.now() - startTime
    if (elapsed < MIN_ANIMATION_DURATION) {
      await new Promise(resolve => setTimeout(resolve, MIN_ANIMATION_DURATION - elapsed))
    }
    
    if (result.success) {
      showToast(t('message.regenerateIdSuccess'), 'success')
      await loadBackups(false)
    } else {
      showToast(result.message, 'error')
    }
  } finally {
    regeneratingId.value = null
  }
}

// 訂閱方案顏色映射（使用大寫 key 以支援 API 返回的全大寫格式）
const subscriptionColorMap: Record<string, string> = {
  'KIRO FREE': 'bg-zinc-500/20 text-zinc-400 border border-zinc-500/30',
  'KIRO PRO': 'bg-blue-500/20 text-blue-400 border border-blue-500/30',
  'KIRO PRO+': 'bg-violet-500/20 text-violet-400 border border-violet-500/30',
  'KIRO POWER': 'bg-amber-500/20 text-amber-400 border border-amber-500/30',
}

// 根據訂閱方案名稱取得對應的顏色 class（大小寫不敏感）
const getSubscriptionColorClass = (subscriptionTitle: string): string => {
  const upperTitle = subscriptionTitle?.toUpperCase() || ''
  return subscriptionColorMap[upperTitle] || 'bg-zinc-500/20 text-zinc-400 border border-zinc-500/30'
}

// 簡化訂閱方案名稱顯示（移除 "KIRO " 前綴）
const getSubscriptionShortName = (subscriptionTitle: string): string => {
  if (!subscriptionTitle) return ''
  const upperTitle = subscriptionTitle.toUpperCase()
  return upperTitle.replace(/^KIRO\s+/, '')
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
      // 更新重置狀態
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

// 檢查 Kiro 安裝路徑偵測狀態，失敗時自動跳轉到設定頁面
const checkPathDetectionStatus = async () => {
  try {
    const result = await window.go.main.App.GetKiroInstallPathWithStatus()
    if (!result.success) {
      // 偵測失敗，自動切換到設定頁面
      activeMenu.value = 'settings'
      showSettingsPanel.value = true
      // 顯示提示訊息
      showToast(t('message.pathDetectionFailed'), 'error')
      // 聚焦到安裝路徑輸入欄位（延遲執行以確保 DOM 已更新）
      setTimeout(() => {
        const pathInput = document.querySelector('input[placeholder*="Kiro"]') as HTMLInputElement
        if (pathInput) {
          pathInput.focus()
        }
      }, 100)
    }
  } catch (e) {
    console.error('Failed to check path detection status:', e)
  }
}

onMounted(() => {
  // 語言已在 i18n/index.ts 中根據系統語言初始化
  // 這裡只需同步 locale 到當前組件（如果 localStorage 有值）
  const savedLang = localStorage.getItem('kiro-manager-lang')
  if (savedLang && ['zh-TW', 'zh-CN'].includes(savedLang)) {
    locale.value = savedLang
  }
  
  // 載入一鍵新機模式設定（硬一鍵新機暫時停用，強制使用一鍵新機）
  resetMode.value = 'soft'
  localStorage.setItem('kiro-manager-reset-mode', 'soft')
  
  // 載入是否已使用過一鍵新機
  hasUsedReset.value = localStorage.getItem('kiro-manager-has-used-reset') === 'true'
  
  loadBackups()
  
  // 檢查 Kiro 安裝路徑偵測狀態（偵測失敗時自動跳轉到設定頁面）
  checkPathDetectionStatus()
  
  // 每 5 秒檢查一次 Kiro 運行狀態
  setInterval(checkKiroStatus, 5000)
  
  // 監聽視窗大小變化
  window.addEventListener('resize', saveWindowSize)
  
  // 自動聚焦主內容滾動區，解決滾輪無法滾動的問題
  setTimeout(() => {
    mainScrollArea.value?.focus()
  }, 100)
  
  // 監聽自動切換事件
  EventsOn('auto-switch', (data: any) => {
    switch (data.Type) {
      case 'switch':
        showToast(t('autoSwitch.toast.switched', { name: data.Data?.to }), 'success')
        loadBackups(false)
        break
      case 'switch_fail':
        showToast(data.Message || t('autoSwitch.toast.switchFailed'), 'error')
        break
      case 'low_balance':
        showToast(t('autoSwitch.toast.lowBalance'), 'warning')
        break
      case 'cooldown':
        // 更新狀態
        loadAutoSwitchSettings()
        break
      case 'max_switch':
        showToast(t('autoSwitch.toast.maxSwitchReached'), 'warning')
        break
      case 'no_candidates':
        showToast(t('autoSwitch.toast.noCandidates'), 'warning')
        break
    }
  })
})

onUnmounted(() => {
  window.removeEventListener('resize', saveWindowSize)
  EventsOff('auto-switch')
})
</script>

<template>
  <div class="flex flex-col h-screen bg-app-bg font-sans text-sm text-zinc-300">
    
    <!-- 頂部導航欄 -->
    <header class="h-16 flex-shrink-0 border-b border-app-border flex items-center justify-between px-6 bg-[#0c0c0e] sticky top-0 z-20">
      <!-- 左側：Logo + 導航項 -->
      <div class="flex items-center gap-6">
        <!-- Logo (只保留 SVG，移除文字) -->
        <svg width="28" height="28" viewBox="0 0 200 200" xmlns="http://www.w3.org/2000/svg" class="flex-shrink-0">
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
        
        <!-- 桌面版水平導航項 (md:flex) -->
        <nav class="hidden md:flex items-center gap-1">
          <div 
            @click="activeMenu = 'dashboard'; showSettingsPanel = false"
            :class="[
              'px-3 py-2 rounded-lg flex items-center cursor-pointer transition-colors',
              activeMenu === 'dashboard' 
                ? 'text-zinc-100 bg-zinc-800/50 border border-zinc-700/50' 
                : 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900'
            ]"
          >
            <Icon name="Home" :class="['w-4 h-4 mr-2', activeMenu === 'dashboard' ? 'text-app-accent' : '']" />
            {{ t('menu.dashboard') }}
          </div>
          <div 
            @click="activeMenu = 'oauth'; showSettingsPanel = false"
            :class="[
              'px-3 py-2 rounded-lg flex items-center cursor-pointer transition-colors',
              activeMenu === 'oauth' 
                ? 'text-zinc-100 bg-zinc-800/50 border border-zinc-700/50' 
                : 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900'
            ]"
          >
            <Icon name="Key" :class="['w-4 h-4 mr-2', activeMenu === 'oauth' ? 'text-app-accent' : '']" />
            {{ t('menu.oauthLogin') }}
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
            <Icon name="Settings" :class="['w-4 h-4 mr-2', activeMenu === 'settings' ? 'text-app-accent' : '']" />
            {{ t('menu.settings') }}
          </div>
        </nav>
        
        <!-- 移動版漢堡菜單按鈕 (md:hidden) -->
        <button 
          @click="isMobileMenuOpen = !isMobileMenuOpen"
          class="md:hidden p-2 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors"
        >
          <Icon :name="isMobileMenuOpen ? 'X' : 'Menu'" class="w-5 h-5" />
        </button>
      </div>
      
      <!-- 右側：狀態指示器 + 版本信息 -->
      <div class="flex flex-col items-end">
        <div class="flex items-center gap-2">
          <div :class="['w-2 h-2 rounded-full', loading ? 'bg-yellow-500 animate-pulse' : kiroRunning ? 'bg-green-500' : 'bg-zinc-500']"></div>
          <span class="text-xs text-zinc-400 font-mono">{{ loading ? t('app.processing') : kiroRunning ? t('app.kiroRunning') : t('app.kiroStopped') }}</span>
        </div>
        <p class="text-zinc-500 text-xs">{{ t('app.version') }}</p>
      </div>
    </header>
    
    <!-- 移動端下拉菜單 -->
    <div v-if="isMobileMenuOpen" class="md:hidden relative z-10">
      <!-- Backdrop 遮罩 -->
      <div 
        class="fixed inset-0 bg-black/50" 
        @click="isMobileMenuOpen = false"
      ></div>
      <!-- 下拉菜單內容 -->
      <nav class="relative bg-[#0c0c0e] border-b border-app-border p-4 space-y-1">
        <div 
          @click="activeMenu = 'dashboard'; showSettingsPanel = false; isMobileMenuOpen = false"
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
          @click="activeMenu = 'oauth'; showSettingsPanel = false; isMobileMenuOpen = false"
          :class="[
            'px-3 py-2 rounded-lg flex items-center cursor-pointer transition-colors',
            activeMenu === 'oauth' 
              ? 'text-zinc-100 bg-zinc-800/50 border border-zinc-700/50' 
              : 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900'
          ]"
        >
          <Icon name="Key" :class="['w-4 h-4 mr-3', activeMenu === 'oauth' ? 'text-app-accent' : '']" />
          {{ t('menu.oauthLogin') }}
        </div>
        <div 
          @click="activeMenu = 'settings'; showSettingsPanel = true; isMobileMenuOpen = false"
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
    </div>

    <!-- 主內容區 -->
    <main class="flex-1 flex flex-col min-w-0 overflow-hidden bg-app-bg relative">

      <!-- 內容滾動區 -->
      <div ref="mainScrollArea" tabindex="0" class="flex-1 overflow-y-auto p-8 space-y-8 focus:outline-none">
        
        <!-- 設定面板 -->
        <div v-if="showSettingsPanel" class="space-y-6">
          <!-- Tab 導航 -->
          <TabBar
            :tabs="SETTINGS_TABS"
            :active-tab="activeTab"
            :disabled="isTabDisabled"
            @update:active-tab="handleTabChange"
          />
          
          <!-- 基礎設定分頁 -->
          <BasicSettingsTab
            v-if="activeTab === 'basic'"
            :kiro-install-path="kiroInstallPathInput"
            :kiro-version="kiroVersionInput"
            :language="locale"
            :low-balance-threshold="appSettings.lowBalanceThreshold"
            :detecting-version="detectingVersion"
            :detecting-path="detectingPath"
            @update:kiro-install-path="kiroInstallPathInput = $event; onKiroInstallPathInput()"
            @update:kiro-version="kiroVersionInput = $event; onKiroVersionInput()"
            @update:language="switchLanguage"
            @update:low-balance-threshold="saveLowBalanceThreshold"
            @detect-version="detectKiroVersion"
            @detect-path="detectKiroInstallPath"
            @save-version="saveKiroVersion"
            @save-path="saveKiroInstallPath"
          />
          
          <!-- 自動切換分頁 -->
          <AutoSwitchTab
            v-if="activeTab === 'autoSwitch'"
            :auto-switch-enabled="autoSwitchSettings.enabled"
            :balance-threshold="autoSwitchSettings.balanceThreshold"
            :min-target-balance="autoSwitchSettings.minTargetBalance"
            :monitor-status="autoSwitchStatus.status as 'stopped' | 'running' | 'cooldown'"
            :folders="folders"
            :selected-folder-ids="autoSwitchSettings.folderIds"
            :selected-subscription-types="autoSwitchSettings.subscriptionTypes"
            :notify-on-switch="autoSwitchSettings.notifyOnSwitch"
            :notify-on-low-balance="autoSwitchSettings.notifyOnLowBalance"
            :refresh-rules="toRefreshRules(autoSwitchSettings.refreshIntervals)"
            @toggle="handleAutoSwitchToggle"
            @update:balance-threshold="autoSwitchSettings.balanceThreshold = $event; saveAutoSwitchSettings()"
            @update:min-target-balance="autoSwitchSettings.minTargetBalance = $event; saveAutoSwitchSettings()"
            @update:selected-folder-ids="autoSwitchSettings.folderIds = $event; saveAutoSwitchSettings()"
            @update:selected-subscription-types="autoSwitchSettings.subscriptionTypes = $event; saveAutoSwitchSettings()"
            @update:notify-on-switch="autoSwitchSettings.notifyOnSwitch = $event; saveAutoSwitchSettings()"
            @update:notify-on-low-balance="autoSwitchSettings.notifyOnLowBalance = $event; saveAutoSwitchSettings()"
            @update:refresh-rules="autoSwitchSettings.refreshIntervals = toRefreshIntervals($event); saveAutoSwitchSettings()"
          />
        </div>
        
        <!-- OAuth 登入頁面 -->
        <div v-else-if="activeMenu === 'oauth'" class="space-y-6">
          <OAuthLogin @snapshot-created="loadBackups(false)" />
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
                <Icon v-else-if="activeBackup.provider === 'Enterprise'" name="AWS" class="w-32 h-32 text-white pointer-events-none" />
                <Icon v-else-if="activeBackup.provider === 'Google'" name="Google" class="w-32 h-32 text-white pointer-events-none" />
                <Icon v-else name="Cpu" class="w-32 h-32 text-white pointer-events-none" />
              </template>
              <!-- 沒有 activeBackup（原始機器）時顯示當前登入的 provider 圖標 -->
              <template v-else>
                <Icon v-if="currentProvider === 'Github'" name="Github" class="w-32 h-32 text-white pointer-events-none" />
                <Icon v-else-if="currentProvider === 'AWS' || currentProvider === 'BuilderId'" name="AWS" class="w-32 h-32 text-white pointer-events-none" />
                <Icon v-else-if="currentProvider === 'Enterprise'" name="AWS" class="w-32 h-32 text-white pointer-events-none" />
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
                  <span :class="['px-2 py-0.5 rounded text-[10px] font-medium', getSubscriptionColorClass(currentUsageInfo.subscriptionTitle)]">
                    {{ getSubscriptionShortName(currentUsageInfo.subscriptionTitle) }}
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
                  :disabled="restoringOriginal"
                  :class="[
                    'flex items-center px-4 py-2 border rounded-lg text-sm transition-all',
                    restoringOriginal
                      ? 'bg-zinc-800/50 border-zinc-700/50 text-zinc-500 cursor-wait'
                      : 'bg-zinc-800/50 hover:bg-red-900/30 border-zinc-700/50 hover:border-red-800/50 text-zinc-400 hover:text-red-400'
                  ]"
                >
                  <Icon 
                    name="Rotate" 
                    :class="['w-4 h-4 mr-2', restoringOriginal ? 'animate-spin' : '']" 
                  />
                  {{ restoringOriginal ? t('app.processing') : t('restore.original') }}
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
                  {{ resetting ? t('message.successChange') : t('restore.resetDesc') }}
                </span>
              </div>
            </button>
          </div>
        </div>

        <!-- 文件夾區域 -->
        <div class="space-y-4">
          <!-- 標題列 -->
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <h3 class="text-zinc-400 text-sm font-semibold flex items-center">
                <Icon name="Folder" class="w-4 h-4 mr-2" />
                {{ t('folder.title') }}
              </h3>
              <!-- Provider 篩選 -->
              <div class="relative">
                <button 
                  @click="toggleFilter('provider')"
                  :class="[
                    'flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-xs transition-all',
                    filterProvider 
                      ? 'border-app-accent bg-app-accent/10 text-app-accent' 
                      : 'border-zinc-700 bg-zinc-900 text-zinc-400 hover:text-zinc-200 hover:border-zinc-600'
                  ]"
                >
                  <span>{{ filterProvider || t('backup.provider') }}</span>
                  <Icon name="ChevronDown" :class="['w-3 h-3 transition-transform', openFilter === 'provider' ? 'rotate-180' : '']" />
                </button>
                <div 
                  v-if="openFilter === 'provider'"
                  class="absolute top-full left-0 mt-1 min-w-[120px] py-1 bg-zinc-900 border border-zinc-700 rounded-lg shadow-xl z-50"
                >
                  <button @click="setFilterProvider('')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', !filterProvider ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">{{ t('backup.provider') }}</button>
                  <button @click="setFilterProvider('Github')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterProvider === 'Github' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">Github</button>
                  <button @click="setFilterProvider('AWS')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterProvider === 'AWS' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">AWS</button>
                  <button @click="setFilterProvider('Google')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterProvider === 'Google' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">Google</button>
                  <button @click="setFilterProvider('Enterprise')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterProvider === 'Enterprise' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">Enterprise</button>
                </div>
              </div>
              <!-- Subscription 篩選 -->
              <div class="relative">
                <button 
                  @click="toggleFilter('subscription')"
                  :class="[
                    'flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-xs transition-all',
                    filterSubscription 
                      ? 'border-app-accent bg-app-accent/10 text-app-accent' 
                      : 'border-zinc-700 bg-zinc-900 text-zinc-400 hover:text-zinc-200 hover:border-zinc-600'
                  ]"
                >
                  <span>{{ filterSubscription || t('backup.subscription') }}</span>
                  <Icon name="ChevronDown" :class="['w-3 h-3 transition-transform', openFilter === 'subscription' ? 'rotate-180' : '']" />
                </button>
                <div 
                  v-if="openFilter === 'subscription'"
                  class="absolute top-full left-0 mt-1 min-w-[120px] py-1 bg-zinc-900 border border-zinc-700 rounded-lg shadow-xl z-50"
                >
                  <button @click="setFilterSubscription('')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', !filterSubscription ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">{{ t('backup.subscription') }}</button>
                  <button @click="setFilterSubscription('FREE')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterSubscription === 'FREE' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">FREE</button>
                  <button @click="setFilterSubscription('PRO')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterSubscription === 'PRO' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">PRO</button>
                  <button @click="setFilterSubscription('PRO+')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterSubscription === 'PRO+' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">PRO+</button>
                  <button @click="setFilterSubscription('POWER')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterSubscription === 'POWER' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">POWER</button>
                </div>
              </div>
              <!-- Balance 篩選 -->
              <div class="relative">
                <button 
                  @click="toggleFilter('balance')"
                  :class="[
                    'flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-xs transition-all',
                    filterBalance 
                      ? 'border-app-accent bg-app-accent/10 text-app-accent' 
                      : 'border-zinc-700 bg-zinc-900 text-zinc-400 hover:text-zinc-200 hover:border-zinc-600'
                  ]"
                >
                  <span>{{ filterBalance === 'LOW' ? t('filter.lowBalance') : filterBalance === 'NORMAL' ? t('filter.normal') : filterBalance === 'NO_DATA' ? t('filter.noData') : t('backup.balance') }}</span>
                  <Icon name="ChevronDown" :class="['w-3 h-3 transition-transform', openFilter === 'balance' ? 'rotate-180' : '']" />
                </button>
                <div 
                  v-if="openFilter === 'balance'"
                  class="absolute top-full left-0 mt-1 min-w-[120px] py-1 bg-zinc-900 border border-zinc-700 rounded-lg shadow-xl z-50"
                >
                  <button @click="setFilterBalance('')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', !filterBalance ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">{{ t('backup.balance') }}</button>
                  <button @click="setFilterBalance('LOW')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterBalance === 'LOW' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">{{ t('filter.lowBalance') }}</button>
                  <button @click="setFilterBalance('NORMAL')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterBalance === 'NORMAL' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">{{ t('filter.normal') }}</button>
                  <button @click="setFilterBalance('NO_DATA')" :class="['w-full px-3 py-2 text-left text-xs transition-colors', filterBalance === 'NO_DATA' ? 'text-app-accent bg-app-accent/10' : 'text-zinc-400 hover:text-white hover:bg-zinc-800']">{{ t('filter.noData') }}</button>
                </div>
              </div>
            </div>
            <div class="flex items-center gap-3">
              <!-- 批量操作按鈕（僅在有選中項目時顯示） -->
              <div v-if="hasSelection" class="flex items-center gap-1 mr-2 pr-3 border-r border-zinc-700">
                <span class="text-xs text-zinc-400 mr-1">
                  {{ t('batch.selected', { count: selectedBackups.size }) }}
                </span>
                <button 
                  @click="batchRefreshUsage"
                  :disabled="batchOperating"
                  :class="[
                    'p-1.5 rounded transition-all',
                    !batchOperating
                      ? 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/50'
                      : 'text-zinc-600 cursor-not-allowed'
                  ]"
                  :title="t('batch.refreshAll')"
                >
                  <Icon name="RefreshCw" :class="['w-4 h-4', batchOperating ? 'animate-spin' : '']" />
                </button>
                <button 
                  @click="batchRegenerateMachineID"
                  :disabled="batchOperating"
                  :class="[
                    'p-1.5 rounded transition-all',
                    !batchOperating
                      ? 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/50'
                      : 'text-zinc-600 cursor-not-allowed'
                  ]"
                  :title="t('batch.regenerateAll')"
                >
                  <Icon name="Key" class="w-4 h-4" />
                </button>
                <!-- 移動到文件夾下拉 -->
                <div class="relative">
                  <button 
                    @click="showMoveToFolderDropdown = !showMoveToFolderDropdown"
                    :disabled="batchOperating"
                    :class="[
                      'p-1.5 rounded transition-all',
                      !batchOperating
                        ? 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/50'
                        : 'text-zinc-600 cursor-not-allowed'
                    ]"
                    :title="t('batch.moveToFolder')"
                  >
                    <Icon name="FolderInput" class="w-4 h-4" />
                  </button>
                  <div 
                    v-if="showMoveToFolderDropdown"
                    class="absolute top-full right-0 mt-1 min-w-[150px] py-1 bg-zinc-900 border border-zinc-700 rounded-lg shadow-xl z-50"
                  >
                    <button 
                      v-for="folder in folders" 
                      :key="folder.id"
                      @click="batchMoveToFolder(folder.id)"
                      class="w-full px-3 py-2 text-left text-xs text-zinc-400 hover:text-white hover:bg-zinc-800 flex items-center gap-2"
                    >
                      <Icon name="Folder" class="w-3.5 h-3.5 text-app-accent" />
                      {{ folder.name }}
                    </button>
                    <div class="border-t border-zinc-700 my-1"></div>
                    <button 
                      @click="batchMoveToFolder(null)"
                      class="w-full px-3 py-2 text-left text-xs text-zinc-400 hover:text-white hover:bg-zinc-800 flex items-center gap-2"
                    >
                      <Icon name="Inbox" class="w-3.5 h-3.5 text-zinc-500" />
                      {{ t('folder.uncategorized') }}
                    </button>
                  </div>
                </div>
                <button 
                  @click="batchDelete"
                  :disabled="batchOperating"
                  :class="[
                    'p-1.5 rounded transition-all',
                    !batchOperating
                      ? 'text-zinc-400 hover:text-red-400 hover:bg-zinc-700/50'
                      : 'text-zinc-600 cursor-not-allowed'
                  ]"
                  :title="t('batch.deleteAll')"
                >
                  <Icon name="Trash" class="w-4 h-4" />
                </button>
              </div>
              <button 
                @click="showCreateFolderModal = true"
                class="flex items-center px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-300 rounded-lg text-xs transition-all"
              >
                <Icon name="FolderPlus" class="w-3.5 h-3.5 mr-1.5" />
                {{ t('folder.create') }}
              </button>
              <!-- 搜索框 -->
              <div class="relative">
                <Icon name="Search" class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
                <input 
                  v-model="searchQuery"
                  :placeholder="t('backup.search')"
                  class="pl-9 pr-4 py-1.5 bg-zinc-900 border border-zinc-700 rounded-lg text-zinc-200 text-sm focus:outline-none focus:border-app-accent transition-colors w-48"
                />
              </div>
            </div>
          </div>

          <!-- 文件夾列表 -->
          <div v-if="folders.length > 0" class="space-y-2">
            <div 
              v-for="folder in folders" 
              :key="folder.id"
              class="bg-app-surface border border-app-border rounded-xl overflow-hidden"
              @dragover="onDragOver"
              @dragenter="onDragEnterFolder(folder.id)"
              @dragleave="onDragLeaveFolder"
              @drop="onDropToFolder($event, folder.id)"
              :class="{ 'ring-2 ring-app-accent': dragOverFolderId === folder.id }"
            >
              <!-- 文件夾標題 -->
              <div 
                class="flex items-center justify-between px-4 py-3 bg-zinc-900/50 cursor-pointer hover:bg-zinc-800/50 transition-colors"
                @click="toggleFolder(folder.id)"
              >
                <div class="flex items-center gap-3">
                  <Icon 
                    :name="expandedFolders.has(folder.id) ? 'ChevronDown' : 'ChevronRight'" 
                    class="w-4 h-4 text-zinc-500 transition-transform" 
                  />
                  <Icon name="Folder" class="w-4 h-4 text-app-accent" />
                  <!-- 重新命名模式 -->
                  <template v-if="renamingFolder === folder.id">
                    <input 
                      v-model="renameFolderName"
                      @click.stop
                      @keyup.enter="confirmRenameFolder"
                      @keyup.escape="cancelRenameFolder"
                      class="px-2 py-1 bg-zinc-800 border border-zinc-600 rounded text-sm text-zinc-200 focus:outline-none focus:border-app-accent"
                      autofocus
                    />
                    <button @click.stop="confirmRenameFolder" class="p-1 text-app-success hover:bg-zinc-700 rounded">
                      <Icon name="Check" class="w-4 h-4" />
                    </button>
                    <button @click.stop="cancelRenameFolder" class="p-1 text-zinc-400 hover:bg-zinc-700 rounded">
                      <Icon name="X" class="w-4 h-4" />
                    </button>
                  </template>
                  <template v-else>
                    <span class="text-zinc-200 font-medium">{{ folder.name }}</span>
                    <span class="text-zinc-500 text-xs">({{ folder.snapshotCount }})</span>
                  </template>
                </div>
                <div class="flex items-center gap-1" @click.stop>
                  <button 
                    @click="startRenameFolder(folder)"
                    class="p-1.5 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50 rounded transition-colors"
                    :title="t('folder.rename')"
                  >
                    <Icon name="Edit" class="w-3.5 h-3.5" />
                  </button>
                  <button 
                    @click="deleteFolder(folder)"
                    :disabled="deletingFolder === folder.id"
                    class="p-1.5 text-zinc-500 hover:text-red-400 hover:bg-zinc-700/50 rounded transition-colors"
                    :title="t('folder.delete')"
                  >
                    <Icon name="Trash" class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
              
              <!-- 文件夾內容（展開時顯示） -->
              <div v-if="expandedFolders.has(folder.id)" class="border-t border-zinc-800">
                <div v-if="getBackupsInFolder(folder.id).length === 0" class="px-4 py-6 text-center text-zinc-500 text-sm">
                  {{ t('folder.emptyFolder') }}
                </div>
                <div v-else class="divide-y divide-zinc-800/50">
                  <!-- 文件夾內的快照 -->
                  <div 
                    v-for="backup in getBackupsInFolder(folder.id)" 
                    :key="backup.name"
                    draggable="true"
                    @dragstart="onDragStart($event, backup.name)"
                    :class="['flex items-center px-4 py-3 group transition-colors cursor-move', backup.isCurrent ? 'bg-app-accent/5' : 'hover:bg-zinc-800/30']"
                  >
                    <!-- Checkbox -->
                    <div class="w-8 flex-shrink-0">
                      <input 
                        type="checkbox"
                        :checked="selectedBackups.has(backup.name)"
                        @change="toggleSelect(backup.name)"
                        class="custom-checkbox"
                      />
                    </div>
                    <!-- 快照名稱 -->
                    <div class="flex-1 min-w-0">
                      <div class="flex items-center">
                        <Icon name="GripVertical" class="w-4 h-4 text-zinc-600 mr-2 opacity-0 group-hover:opacity-100 transition-opacity" />
                        <div v-if="backup.isCurrent" class="w-1.5 h-1.5 rounded-full bg-app-warning mr-2 shadow-[0_0_8px_rgba(245,158,11,0.8)]"></div>
                        <span :class="['font-medium truncate', backup.isCurrent ? 'text-white' : 'text-zinc-400']">
                          {{ backup.name }}
                        </span>
                        <span v-if="backup.isOriginalMachine" class="ml-2 px-1.5 py-0.5 rounded text-[10px] bg-app-accent/20 text-app-accent border border-app-accent/30">
                          {{ t('backup.original') }}
                        </span>
                      </div>
                    </div>
                    <!-- Provider -->
                    <div class="w-24 flex-shrink-0 px-2 flex justify-center">
                      <span class="px-2 py-1 rounded text-[10px] bg-zinc-800 text-zinc-400 border border-zinc-700 inline-flex items-center gap-1">
                        <Icon v-if="backup.provider === 'Github'" name="Github" class="w-3 h-3" />
                        <Icon v-else-if="backup.provider === 'AWS' || backup.provider === 'BuilderId'" name="AWS" class="w-3 h-3" />
                        <Icon v-else-if="backup.provider === 'Enterprise'" name="AWS" class="w-3 h-3" />
                        <Icon v-else-if="backup.provider === 'Google'" name="Google" class="w-3 h-3" />
                        {{ backup.provider }}
                      </span>
                    </div>
                    <!-- 訂閱類型 -->
                    <div class="w-16 flex-shrink-0 px-2">
                      <span 
                        v-if="backup.subscriptionTitle"
                        :class="['px-2 py-0.5 rounded text-[10px] font-medium', getSubscriptionColorClass(backup.subscriptionTitle)]"
                      >
                        {{ getSubscriptionShortName(backup.subscriptionTitle) }}
                      </span>
                      <span v-else class="text-zinc-500 text-xs">-</span>
                    </div>
                    <!-- 餘額 -->
                    <div class="w-28 flex-shrink-0 px-2 flex items-center justify-end gap-2">
                      <span v-if="backup.usageLimit > 0" :class="['font-mono text-xs', backup.isLowBalance ? 'text-app-warning' : 'text-zinc-400']">
                        {{ Math.round(backup.balance) }}/{{ Math.round(backup.usageLimit) }}
                      </span>
                      <span v-else class="text-zinc-500 text-xs">-</span>
                      <!-- 刷新按鈕：顯示倒計時或刷新圖標 -->
                      <button
                        @click.stop="refreshBackupUsage(backup.name)"
                        :disabled="isInCooldown(backup.name)"
                        class="p-0.5 text-zinc-500 hover:text-zinc-300 transition-colors disabled:cursor-not-allowed"
                        :title="t('backup.refreshUsage')"
                      >
                        <span v-if="isInCooldown(backup.name)" class="text-xs font-mono text-zinc-500 w-4 inline-block text-center">
                          {{ countdownTimers[backup.name] }}
                        </span>
                        <Icon v-else name="RefreshCw" :class="['w-3 h-3', refreshingBackup === backup.name ? 'animate-spin' : '']" />
                      </button>
                    </div>
                    <!-- Machine ID -->
                    <div class="w-32 flex-shrink-0 px-2">
                      <button
                        v-if="backup.machineId"
                        @click="copyMachineId(backup.machineId)"
                        class="font-mono text-[10px] text-zinc-500 hover:text-zinc-300 cursor-pointer transition-colors inline-flex items-center gap-1 group"
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
                      <span v-else class="font-mono text-[10px] text-zinc-500">-</span>
                    </div>
                    <!-- 操作按鈕 -->
                    <div class="w-28 flex-shrink-0 flex justify-end gap-1">
                      <button 
                        v-if="!backup.isCurrent"
                        @click="switchToBackup(backup.name)"
                        :disabled="switchingBackup === backup.name"
                        class="p-1.5 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50 rounded transition-colors"
                        :title="t('backup.switchTo')"
                      >
                        <Icon name="Download" :class="['w-3.5 h-3.5', switchingBackup === backup.name ? 'animate-bounce' : '']" />
                      </button>
                      <button 
                        v-if="!backup.isCurrent"
                        @click="regenerateMachineID(backup.name)"
                        :disabled="regeneratingId === backup.name"
                        class="p-1.5 text-zinc-500 hover:text-app-accent hover:bg-zinc-700/50 rounded transition-colors"
                        :title="t('backup.regenerateId')"
                      >
                        <Icon name="Key" :class="['w-3.5 h-3.5', regeneratingId === backup.name ? 'animate-pulse-fast' : '']" />
                      </button>
                      <button 
                        v-if="!backup.isCurrent"
                        @click="deleteBackup(backup.name)"
                        :disabled="deletingBackup === backup.name"
                        class="p-1.5 text-zinc-500 hover:text-red-400 hover:bg-zinc-700/50 rounded transition-colors"
                        :title="t('backup.delete')"
                      >
                        <Icon name="Trash" :class="['w-3.5 h-3.5', deletingBackup === backup.name ? 'animate-pulse' : '']" />
                      </button>
                      <span v-if="backup.isCurrent" class="text-app-warning text-xs font-bold px-2">{{ t('status.active') }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 未分類區域 -->
          <div 
            class="bg-app-surface border border-app-border rounded-xl overflow-hidden"
            @dragover="onDragOver"
            @dragenter="onDragEnterUncategorized"
            @dragleave="onDragLeaveUncategorized"
            @drop="onDropToUncategorized"
            :class="{ 'ring-2 ring-app-accent': dragOverUncategorized }"
          >
            <!-- 未分類標題（可點擊展開） -->
            <div 
              class="flex items-center justify-between px-4 py-3 bg-zinc-900/50 cursor-pointer hover:bg-zinc-800/50 transition-colors"
              @click="toggleUncategorized"
            >
              <div class="flex items-center gap-3">
                <Icon 
                  :name="uncategorizedExpanded ? 'ChevronDown' : 'ChevronRight'" 
                  class="w-4 h-4 text-zinc-500 transition-transform" 
                />
                <Icon name="Inbox" class="w-4 h-4 text-zinc-500" />
                <span class="text-zinc-400 font-medium">{{ t('folder.uncategorized') }}</span>
                <span class="text-zinc-500 text-xs">({{ uncategorizedBackups.length }})</span>
              </div>
              <!-- 全選 checkbox -->
              <div @click.stop>
                <input 
                  type="checkbox"
                  :checked="isAllSelected"
                  @change="toggleSelectAll"
                  class="custom-checkbox"
                />
              </div>
            </div>
            <!-- 未分類內容（展開時顯示） -->
            <div v-if="uncategorizedExpanded" class="border-t border-zinc-800">
              <div v-if="uncategorizedBackups.length === 0" class="px-4 py-6 text-center text-zinc-500 text-sm">
                {{ t('backup.noBackups') }}
              </div>
              <div v-else class="divide-y divide-zinc-800/50">
                <!-- 未分類的快照 -->
                <div 
                  v-for="backup in uncategorizedBackups" 
                  :key="backup.name"
                  draggable="true"
                  @dragstart="onDragStart($event, backup.name)"
                  :class="['flex items-center px-4 py-3 group transition-colors cursor-move', backup.isCurrent ? 'bg-app-accent/5' : 'hover:bg-zinc-800/30']"
                >
                  <!-- Checkbox -->
                  <div class="w-8 flex-shrink-0">
                    <input 
                      type="checkbox"
                      :checked="selectedBackups.has(backup.name)"
                      @change="toggleSelect(backup.name)"
                      class="custom-checkbox"
                    />
                  </div>
                  <!-- 快照名稱 -->
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center">
                      <Icon name="GripVertical" class="w-4 h-4 text-zinc-600 mr-2 opacity-0 group-hover:opacity-100 transition-opacity" />
                      <div v-if="backup.isCurrent" class="w-1.5 h-1.5 rounded-full bg-app-warning mr-2 shadow-[0_0_8px_rgba(245,158,11,0.8)]"></div>
                      <span :class="['font-medium truncate', backup.isCurrent ? 'text-white' : 'text-zinc-400']">
                        {{ backup.name }}
                      </span>
                      <span v-if="backup.isOriginalMachine" class="ml-2 px-1.5 py-0.5 rounded text-[10px] bg-app-accent/20 text-app-accent border border-app-accent/30">
                        {{ t('backup.original') }}
                      </span>
                    </div>
                  </div>
                  <!-- Provider -->
                  <div class="w-24 flex-shrink-0 px-2 flex justify-center">
                    <span class="px-2 py-1 rounded text-[10px] bg-zinc-800 text-zinc-400 border border-zinc-700 inline-flex items-center gap-1">
                      <Icon v-if="backup.provider === 'Github'" name="Github" class="w-3 h-3" />
                      <Icon v-else-if="backup.provider === 'AWS' || backup.provider === 'BuilderId'" name="AWS" class="w-3 h-3" />
                      <Icon v-else-if="backup.provider === 'Enterprise'" name="AWS" class="w-3 h-3" />
                      <Icon v-else-if="backup.provider === 'Google'" name="Google" class="w-3 h-3" />
                      {{ backup.provider }}
                    </span>
                  </div>
                  <!-- 訂閱類型 -->
                  <div class="w-16 flex-shrink-0 px-2">
                    <span 
                      v-if="backup.subscriptionTitle"
                      :class="['px-2 py-0.5 rounded text-[10px] font-medium', getSubscriptionColorClass(backup.subscriptionTitle)]"
                    >
                      {{ getSubscriptionShortName(backup.subscriptionTitle) }}
                    </span>
                    <span v-else class="text-zinc-500 text-xs">-</span>
                  </div>
                  <!-- 餘額 -->
                  <div class="w-28 flex-shrink-0 px-2 flex items-center justify-end gap-2">
                    <span v-if="backup.usageLimit > 0" :class="['font-mono text-xs', backup.isLowBalance ? 'text-app-warning' : 'text-zinc-400']">
                      {{ Math.round(backup.balance) }}/{{ Math.round(backup.usageLimit) }}
                    </span>
                    <span v-else class="text-zinc-500 text-xs">-</span>
                    <!-- 刷新按鈕：顯示倒計時或刷新圖標 -->
                    <button
                      @click.stop="refreshBackupUsage(backup.name)"
                      :disabled="isInCooldown(backup.name)"
                      class="p-0.5 text-zinc-500 hover:text-zinc-300 transition-colors disabled:cursor-not-allowed"
                      :title="t('backup.refreshUsage')"
                    >
                      <span v-if="isInCooldown(backup.name)" class="text-xs font-mono text-zinc-500 w-4 inline-block text-center">
                        {{ countdownTimers[backup.name] }}
                      </span>
                      <Icon v-else name="RefreshCw" :class="['w-3 h-3', refreshingBackup === backup.name ? 'animate-spin' : '']" />
                    </button>
                  </div>
                  <!-- Machine ID -->
                  <div class="w-32 flex-shrink-0 px-2">
                    <button
                      v-if="backup.machineId"
                      @click="copyMachineId(backup.machineId)"
                      class="font-mono text-[10px] text-zinc-500 hover:text-zinc-300 cursor-pointer transition-colors inline-flex items-center gap-1 group"
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
                    <span v-else class="font-mono text-[10px] text-zinc-500">-</span>
                  </div>
                  <!-- 操作按鈕 -->
                  <div class="w-28 flex-shrink-0 flex justify-end gap-1">
                    <button 
                      v-if="!backup.isCurrent"
                      @click="switchToBackup(backup.name)"
                      :disabled="switchingBackup === backup.name"
                      class="p-1.5 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50 rounded transition-colors"
                      :title="t('backup.switchTo')"
                    >
                      <Icon name="Download" :class="['w-3.5 h-3.5', switchingBackup === backup.name ? 'animate-bounce' : '']" />
                    </button>
                    <button 
                      v-if="!backup.isCurrent"
                      @click="regenerateMachineID(backup.name)"
                      :disabled="regeneratingId === backup.name"
                      class="p-1.5 text-zinc-500 hover:text-app-accent hover:bg-zinc-700/50 rounded transition-colors"
                      :title="t('backup.regenerateId')"
                    >
                      <Icon name="Key" :class="['w-3.5 h-3.5', regeneratingId === backup.name ? 'animate-pulse-fast' : '']" />
                    </button>
                    <button 
                      v-if="!backup.isCurrent"
                      @click="deleteBackup(backup.name)"
                      :disabled="deletingBackup === backup.name"
                      class="p-1.5 text-zinc-500 hover:text-red-400 hover:bg-zinc-700/50 rounded transition-colors"
                      :title="t('backup.delete')"
                    >
                      <Icon name="Trash" :class="['w-3.5 h-3.5', deletingBackup === backup.name ? 'animate-pulse' : '']" />
                    </button>
                    <span v-if="backup.isCurrent" class="text-app-warning text-xs font-bold px-2">{{ t('status.active') }}</span>
                  </div>
                </div>
              </div>
            </div>
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
            :disabled="creatingBackup"
            class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ t('backup.cancel') }}
          </button>
          <button 
            @click="createBackup"
            :disabled="creatingBackup || !newBackupName.trim()"
            :class="[
              'px-4 py-2 rounded-lg text-sm transition-colors flex items-center gap-2',
              creatingBackup || !newBackupName.trim()
                ? 'bg-app-accent/50 text-white/70 cursor-wait'
                : 'bg-app-accent hover:bg-app-accent/80 text-white'
            ]"
          >
            <Icon v-if="creatingBackup" name="Loader" class="w-4 h-4 animate-spin" />
            {{ creatingBackup ? t('app.processing') : t('backup.confirm') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 建立文件夾 Modal -->
    <div v-if="showCreateFolderModal" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50" @click.self="showCreateFolderModal = false; newFolderName = ''">
      <div class="bg-zinc-900 border border-zinc-700 rounded-xl p-6 w-96 shadow-2xl">
        <h3 class="text-lg font-semibold text-zinc-200 mb-4">{{ t('folder.create') }}</h3>
        <input 
          v-model="newFolderName"
          :placeholder="t('backup.namePlaceholder')"
          @keyup.enter="createFolder"
          class="w-full px-4 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-200 focus:outline-none focus:border-app-accent mb-4"
          autofocus
        />
        <div class="flex justify-end gap-3">
          <button 
            @click="showCreateFolderModal = false; newFolderName = ''"
            class="px-4 py-2 text-zinc-400 hover:text-zinc-200 transition-colors"
          >
            {{ t('backup.cancel') }}
          </button>
          <button 
            @click="createFolder"
            :disabled="!newFolderName.trim() || creatingFolder"
            class="px-4 py-2 bg-app-accent hover:bg-app-accent/80 disabled:opacity-50 text-white rounded-lg transition-colors"
          >
            {{ creatingFolder ? t('app.processing') : t('backup.confirm') }}
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

    <!-- Delete Folder Dialog (非空文件夾刪除確認) -->
    <div v-if="showDeleteFolderDialog && folderToDelete" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50" @click.self="cancelDeleteFolder">
      <div class="bg-app-surface border border-app-border rounded-xl p-6 max-w-md shadow-2xl">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 rounded-full flex items-center justify-center bg-app-danger/20">
            <Icon name="Trash" class="w-5 h-5 text-app-danger" />
          </div>
          <h3 class="text-white font-semibold text-lg">{{ t('dialog.deleteTitle') }}</h3>
        </div>
        
        <p class="text-zinc-300 text-sm leading-relaxed mb-6">
          {{ t('folder.deleteConfirm', { name: folderToDelete.name, count: getBackupsInFolder(folderToDelete.id).length }) }}
        </p>
        
        <div class="flex justify-end gap-3">
          <button 
            @click="cancelDeleteFolder"
            class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-sm transition-colors"
          >
            {{ t('backup.cancel') }}
          </button>
          <button 
            @click="confirmDeleteFolder(false)"
            class="px-4 py-2 bg-app-accent hover:bg-app-accent/80 text-white rounded-lg text-sm transition-colors"
          >
            {{ t('folder.moveToUncategorized') }}
          </button>
          <button 
            @click="confirmDeleteFolder(true)"
            class="px-4 py-2 bg-app-danger hover:bg-app-danger/80 text-white rounded-lg text-sm transition-colors"
          >
            {{ t('folder.deleteWithSnapshots') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Toast -->
    <Transition name="slide">
      <div 
        v-if="toast.show" 
        :class="[
          'fixed bottom-5 right-5 px-4 py-3 rounded-xl text-white text-sm z-50',
          'shadow-lg border border-app-border backdrop-blur-xl',
          'flex items-center gap-3',
          'bg-zinc-900/90'
        ]"
      >
        <Icon 
          :name="toast.type === 'success' ? 'CheckCircle' : 'XCircle'" 
          :class="[
            'w-5 h-5 flex-shrink-0',
            toast.type === 'success' ? 'text-app-success' : 'text-app-danger'
          ]" 
        />
        <span>{{ toast.message }}</span>
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

/* 刷新頻率設定樣式 */
.refresh-intervals-list {
  margin: 8px 0;
}

.refresh-interval-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  padding: 8px;
  background: rgb(39 39 42);
  border-radius: 4px;
}

.interval-inputs {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.interval-input {
  width: 60px;
  padding: 4px 8px;
  border: 1px solid rgb(63 63 70);
  border-radius: 4px;
  background: rgb(24 24 27);
  color: rgb(212 212 216);
  font-size: 12px;
}

.interval-separator {
  color: rgb(161 161 170);
  font-size: 12px;
}

.interval-unit {
  color: rgb(161 161 170);
  font-size: 12px;
}

.btn-remove {
  padding: 4px 8px;
  background: transparent;
  border: none;
  color: rgb(161 161 170);
  cursor: pointer;
  font-size: 14px;
}

.btn-remove:hover {
  color: #ef4444;
}

.btn-add-rule {
  padding: 6px 12px;
  background: rgb(39 39 42);
  border: 1px dashed rgb(63 63 70);
  border-radius: 4px;
  color: rgb(161 161 170);
  cursor: pointer;
  font-size: 12px;
}

.btn-add-rule:hover {
  border-color: var(--app-accent, #3b82f6);
  color: var(--app-accent, #3b82f6);
}
</style>
