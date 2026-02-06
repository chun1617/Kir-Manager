<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import Icon from './components/Icon.vue'
import OAuthLogin from './components/OAuthLogin.vue'
import TabBar from './components/settings/TabBar.vue'
import BasicSettingsTab from './components/settings/BasicSettingsTab.vue'
import AutoSwitchTab from './components/settings/AutoSwitchTab.vue'
import CurrentStatusCard from './components/CurrentStatusCard.vue'
import SoftResetCard from './components/SoftResetCard.vue'
import FolderTree from './components/FolderTree.vue'
import { useSettingsPage } from './composables/useSettingsPage'
import { SETTINGS_TABS } from './constants/settingsTabs'
import type { RefreshRule } from './types/refreshInterval'
import type { AutoSwitchEvent } from './types/ui'
import { withTimeout, TimeoutError } from './utils/withTimeout'

// 引入所有 Composables
import { useUIState } from './composables/useUIState'
import { useUsageRefresh } from './composables/useUsageRefresh'
import { useBackupManagement } from './composables/useBackupManagement'
import { useFolderManagement } from './composables/useFolderManagement'
import { useAutoSwitch } from './composables/useAutoSwitch'
import { useSoftReset } from './composables/useSoftReset'
import { useAppSettings } from './composables/useAppSettings'

const { t, locale } = useI18n()
const { activeTab, isTabDisabled, handleTabChange } = useSettingsPage()

// ============================================================================
// 初始化 Composables
// ============================================================================

// UI 狀態管理
const {
  activeMenu,
  isMobileMenuOpen,
  openFilter,
  toast,
  confirmDialog,
  toggleMobileMenu,
  setActiveMenu,
  toggleFilter,
  closeAllFilters,
  showToast,
  showConfirmDialog,
  handleError,
} = useUIState()

// 用量刷新
const {
  refreshingBackup,
  refreshingCurrent,
  countdownTimers,
  countdownCurrentAccount,
  isInCooldown,
  isCurrentInCooldown,
  cleanup: cleanupUsageRefresh,  // P0-FIX: 解構 cleanup 函數用於 onUnmounted 清理
  // Phase 2 Task 7: 從 Composable 解構用量刷新方法
  refreshBackupUsageWithUpdate,
  refreshCurrentUsageWithUpdate,
} = useUsageRefresh()

// P0-FIX: Kiro 狀態檢查 interval ID（用於 onUnmounted 清理）
let kiroStatusInterval: ReturnType<typeof setInterval> | null = null

// 備份管理
const {
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
  activeBackup,
  filteredBackups,
  setFilterSubscription,
  setFilterProvider,
  setFilterBalance,
  toggleSelect,
  toggleSelectAll,
  // Phase 2 Task 4: 從 Composable 解構備份操作方法
  createBackup: createBackupCore,
  switchToBackup: switchToBackupCore,
  deleteBackup: deleteBackupCore,
  regenerateMachineID: regenerateMachineIDCore,
  batchDelete: batchDeleteCore,
  batchRegenerateMachineID: batchRegenerateMachineIDCore,
  batchRefreshUsage: batchRefreshUsageCore,
  loadBackups: loadBackupsCore,
} = useBackupManagement()

// 文件夾管理
const {
  folders,
  expandedFolders,
  uncategorizedExpanded,
  showCreateFolderModal,
  newFolderName,
  creatingFolder,
  renamingFolder,
  renameFolderName,
  deletingFolder,
  dragOverFolderId,
  dragOverUncategorized,
  showDeleteFolderDialog,
  folderToDelete,
  loadFolders,
  startRenameFolder,
  confirmRenameFolder,
  cancelRenameFolder,
  cancelDeleteFolder,
  toggleFolder,
  toggleUncategorized,
  onDragEnterFolder,
  onDragLeaveFolder,
  onDragEnterUncategorized,
  onDragLeaveUncategorized,
} = useFolderManagement()

// 自動切換
const {
  autoSwitchSettings,
  autoSwitchStatus,
  savingAutoSwitch,
  loadAutoSwitchSettings,
  saveAutoSwitchSettings,
  addAutoSwitchFolder,
  removeAutoSwitchFolder,
  addAutoSwitchSubscription,
  removeAutoSwitchSubscription,
  addRefreshRule,
  removeRefreshRule,
} = useAutoSwitch()

// 軟重置
const {
  softResetStatus,
  resetting,
  restoringOriginal,
  patching,
  hasUsedReset,
  showFirstTimeResetModal,
  getSoftResetStatus,
  openExtensionFolder,
  openMachineIDFolder,
  openSSOCacheFolder,
  // Phase 2 Task 6: 從 Composable 解構軟重置操作方法
  resetToNew: resetToNewCore,
  executeReset: executeResetCore,
  confirmFirstTimeReset: confirmFirstTimeResetCore,
  restoreOriginal: restoreOriginalCore,
  regenerateMachineID: regenerateMachineIDSoftResetCore,
  patchExtension: patchExtensionCore,
} = useSoftReset()

// 應用設定
const {
  appSettings,
  kiroVersionInput,
  kiroVersionModified,
  kiroInstallPathInput,
  kiroInstallPathModified,
  thresholdPreview,
  detectingVersion,
  detectingPath,
  loadSettings,
  onKiroVersionInput,
  onKiroInstallPathInput,
  // Phase 2 Task 2: 從 Composable 解構設定相關方法
  saveKiroVersion: saveKiroVersionCore,
  detectKiroVersion: detectKiroVersionCore,
  saveKiroInstallPath: saveKiroInstallPathCore,
  detectKiroInstallPath: detectKiroInstallPathCore,
  clearKiroInstallPath: clearKiroInstallPathCore,
  switchLanguage,
  // Phase 2 Task 7.3: 從 Composable 解構閾值儲存方法
  saveLowBalanceThreshold: saveLowBalanceThresholdCore,
} = useAppSettings()

// ============================================================================
// App.vue 特有的狀態和邏輯
// ============================================================================

// 主內容滾動區 ref（用於自動聚焦）
const mainScrollArea = ref<HTMLElement | null>(null)

// 設定面板顯示狀態
const showSettingsPanel = ref(false)

// 建立備份 Modal 狀態
const showCreateModal = ref(false)
const newBackupName = ref('')

// 批量移動到文件夾下拉選單
const showMoveToFolderDropdown = ref(false)

// 複製機器碼 ID 狀態
const copiedMachineId = ref<string | null>(null)

// 一鍵新機模式（硬一鍵新機暫時停用）
const resetMode = ref<'soft'>('soft')

// 當前帳號用量資訊
const currentUsageInfo = ref<{
  subscriptionTitle: string
  usageLimit: number
  currentUsage: number
  balance: number
  isLowBalance: boolean
} | null>(null)

// 當前 Provider
const currentProvider = ref('')

// Kiro 運行狀態
const kiroRunning = ref(false)

// 全域 loading 狀態
const loading = ref(false)

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
  if (window.go?.main?.App?.SaveWindowSize) {
    window.go.main.App.SaveWindowSize(width, height);
  }
}, 500);

// ============================================================================
// 類型定義（保留給模板使用）
// ============================================================================

interface RefreshIntervalRule {
  minBalance: number
  maxBalance: number
  interval: number
}

// 刷新頻率規則轉換函數
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

// ============================================================================
// 計算屬性
// ============================================================================

// 未分類備份（使用 filteredBackups）
const uncategorizedBackups = computed(() => {
  return filteredBackups.value.filter(b => !b.folderId)
})

// 獲取文件夾內的備份
const getBackupsInFolder = (folderId: string) => {
  return filteredBackups.value.filter(b => b.folderId === folderId)
}

// 是否全選
const isAllSelected = computed(() => 
  filteredBackups.value.length > 0 && 
  selectedBackups.value.size === filteredBackups.value.length
)

// 是否有選中項目
const hasSelection = computed(() => selectedBackups.value.size > 0)

// ============================================================================
// 輔助函數
// ============================================================================

// 拖放事件處理
const onDragStart = (event: DragEvent, backupName: string) => {
  if (event.dataTransfer) {
    event.dataTransfer.setData('text/plain', backupName)
    event.dataTransfer.effectAllowed = 'move'
  }
}

const onDragOver = (event: DragEvent) => {
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
}

// 訂閱方案和機器碼工具函數已移至 utils/subscription.ts 和 utils/machineId.ts
// Phase 4: 移除重複定義，組件直接從 utils 導入

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

// ============================================================================
// 包裝函數（整合 Composables 和 App.vue 特有邏輯）
// ============================================================================

const checkKiroStatus = async () => {
  try {
    kiroRunning.value = await window.go.main.App.IsKiroRunning()
  } catch (e) {
    console.error(e)
  }
}

// 載入備份（整合多個 Composables 的載入邏輯）
const loadBackups = async (showOverlay: boolean = true) => {
  // P0-1 FIX: 並發保護 - 防止多個 loadBackups 同時執行
  if (isLoadingBackups.value) return
  isLoadingBackups.value = true
  
  if (showOverlay) {
    loading.value = true
  }
  try {
    // 載入備份列表
    backups.value = await window.go.main.App.GetBackupList() || []
    currentMachineId.value = await window.go.main.App.GetCurrentMachineID()
    currentEnvironmentName.value = await window.go.main.App.GetCurrentEnvironmentName()
    
    // 載入軟重置狀態
    softResetStatus.value = await window.go.main.App.GetSoftResetStatus()
    
    // 載入當前帳號資訊
    currentProvider.value = await window.go.main.App.GetCurrentProvider()
    currentUsageInfo.value = await window.go.main.App.GetCurrentUsageInfo()
    
    // 載入應用設定
    const settings = await window.go.main.App.GetSettings()
    appSettings.value = settings
    thresholdPreview.value = Math.round(settings.lowBalanceThreshold * 100)
    kiroVersionInput.value = settings.kiroVersion || '0.8.206'
    kiroVersionModified.value = false
    kiroInstallPathInput.value = settings.customKiroInstallPath || ''
    kiroInstallPathModified.value = false
    
    await checkKiroStatus()
    await loadFolders()
    await loadAutoSwitchSettings()
  } catch (e) {
    console.error(e)
  } finally {
    isLoadingBackups.value = false  // P0-1 FIX: 重置並發保護
    if (showOverlay) {
      loading.value = false
    }
  }
}

// 建立備份
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

// 切換備份
const switchToBackup = async (name: string) => {
  if (switchingBackup.value) return
  switchingBackup.value = name
  try {
    // P1-3 FIX: 加入超時保護（30 秒）
    const result = await withTimeout(
      window.go.main.App.SwitchToBackup(name),
      30000,
      t('message.operationTimeout')
    )
    if (result.success) {
      showToast(t('message.successChange'), 'success')
      await loadBackups(false)
    } else {
      showToast(result.message, 'error')
    }
  } catch (e: any) {
    if (e instanceof TimeoutError) {
      showToast(e.message, 'error')
    } else {
      showToast(e.message || t('message.error'), 'error')
    }
  } finally {
    switchingBackup.value = null
  }
}

// 刪除備份
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

// Phase 2 Task 6.4: 簡化重新生成機器碼 - 調用 Composable 方法 + 動畫延遲 + Toast
const regenerateMachineID = async (name: string) => {
  regeneratingId.value = name
  const startTime = Date.now()
  const MIN_ANIMATION_DURATION = 600
  
  try {
    const result = await regenerateMachineIDSoftResetCore(name)
    
    // 保留動畫延遲邏輯
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

// Phase 2 Task 6.2: 簡化還原原始機器 - 確認對話框 + 調用 Composable 方法 + Toast
const restoreOriginal = async () => {
  const confirmed = await showConfirmDialog({
    title: t('dialog.warningTitle'),
    message: t('message.confirmRestore'),
    type: 'warning'
  })
  if (!confirmed) return
  
  const result = await restoreOriginalCore()
  if (result.success) {
    showToast(t('message.successChange'), 'success')
    await loadBackups(false)
  } else {
    showToast(result.message, 'error')
  }
}

// Phase 2 Task 6.1: 簡化一鍵新機 - 調用 Composable 方法 + 確認對話框 + Toast
const resetToNew = async () => {
  // 首次使用檢查由 Composable 處理（會顯示 modal）
  await resetToNewCore()
  
  // 如果不是首次使用，顯示確認對話框
  if (hasUsedReset.value) {
    const confirmed = await showConfirmDialog({
      title: t('dialog.warningTitle'),
      message: t('message.confirmReset'),
      type: 'warning'
    })
    if (!confirmed) return
    
    await executeReset()
  }
}

const executeReset = async () => {
  const result = await executeResetCore()
  
  if (result.success) {
    showToast(result.message, 'success')
    await loadBackups(false)
  } else {
    showToast(result.message, 'error')
  }
}

const confirmFirstTimeReset = async () => {
  // 關閉 modal 由 Composable 處理
  showFirstTimeResetModal.value = false
  
  const confirmed = await showConfirmDialog({
    title: t('dialog.warningTitle'),
    message: t('message.confirmReset'),
    type: 'warning'
  })
  if (!confirmed) return
  
  await executeReset()
}

// Phase 2 Task 6.3: 簡化 Patch Extension - 調用 Composable 方法 + Toast
const patchExtension = async () => {
  const result = await patchExtensionCore()
  if (result.success) {
    showToast(result.message, 'success')
  } else {
    showToast(result.message, 'error')
  }
}

// Phase 2 Task 7.1: 簡化刷新備份餘額 - 調用 Composable 方法 + 本地狀態更新回調
const refreshBackupUsage = async (name: string) => {
  const backup = backups.value.find(b => b.name === name)
  
  // 額外檢查：如果是當前帳號且當前帳號在冷卻期，也不刷新
  if (backup?.isCurrent && isCurrentInCooldown()) return
  
  const result = await refreshBackupUsageWithUpdate(name, {
    onLocalUpdate: async () => {
      // 重新載入備份以獲取最新數據
      await loadBackups(false)
      
      // 如果是當前帳號，同步更新 currentUsageInfo
      if (backup?.isCurrent) {
        const updatedBackup = backups.value.find(b => b.name === name)
        if (updatedBackup) {
          currentUsageInfo.value = {
            subscriptionTitle: updatedBackup.subscriptionTitle,
            usageLimit: updatedBackup.usageLimit,
            currentUsage: updatedBackup.currentUsage,
            balance: updatedBackup.balance,
            isLowBalance: updatedBackup.isLowBalance
          }
        }
      }
    }
  })
  
  if (!result.success) {
    showToast(result.message, 'error')
  }
}

// Phase 2 Task 7.2: 簡化刷新當前帳號餘額 - 調用 Composable 方法 + 本地狀態更新回調
const refreshCurrentUsage = async () => {
  const currentBackup = backups.value.find(b => b.isCurrent)
  if (!currentBackup) return
  
  // 如果備份也在冷卻期，不刷新
  if (isInCooldown(currentBackup.name)) return

  const result = await refreshCurrentUsageWithUpdate({
    onLocalUpdate: async () => {
      // 重新載入備份以獲取最新數據
      await loadBackups(false)
      
      // 同步更新 currentUsageInfo
      const updatedBackup = backups.value.find(b => b.isCurrent)
      if (updatedBackup) {
        currentUsageInfo.value = {
          subscriptionTitle: updatedBackup.subscriptionTitle,
          usageLimit: updatedBackup.usageLimit,
          currentUsage: updatedBackup.currentUsage,
          balance: updatedBackup.balance,
          isLowBalance: updatedBackup.isLowBalance
        }
      }
    }
  })
  
  if (!result.success) {
    showToast(result.message, 'error')
  }
}

// Phase 2 Task 4: 簡化批量操作 - 調用 Composable 方法 + Toast 通知
const batchDelete = async () => {
  if (!hasSelection.value || batchOperating.value) return
  
  const result = await batchDeleteCore()
  await loadBackups(false)
  
  // 根據 BatchResult 顯示適當訊息
  if (result.failedItems.length === 0) {
    showToast(t('message.success'), 'success')
  } else if (result.successCount > 0) {
    showToast(t('batch.partialSuccess', { failed: result.failedItems.length }), 'warning')
  } else {
    showToast(t('batch.allFailed'), 'error')
  }
}

const batchRegenerateMachineID = async () => {
  if (!hasSelection.value || batchOperating.value) return
  
  const result = await batchRegenerateMachineIDCore()
  await loadBackups(false)
  
  // 根據 BatchResult 顯示適當訊息
  if (result.failedItems.length === 0) {
    showToast(t('message.success'), 'success')
  } else if (result.successCount > 0) {
    showToast(t('batch.partialSuccess', { failed: result.failedItems.length }), 'warning')
  } else {
    showToast(t('batch.allFailed'), 'error')
  }
}

const batchRefreshUsage = async () => {
  if (!hasSelection.value || batchOperating.value) return
  
  const result = await batchRefreshUsageCore(isInCooldown)
  
  // 根據 BatchResult 顯示適當訊息
  if (result.failedItems.length === 0) {
    showToast(t('message.success'), 'success')
  } else if (result.successCount > 0) {
    showToast(t('batch.partialSuccess', { failed: result.failedItems.length }), 'warning')
  } else {
    showToast(t('batch.allFailed'), 'error')
  }
}

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
    
    selectedBackups.value = new Set()
    await loadBackups(false)
    await loadFolders()
    showToast(t('batch.moveSuccess', { count: backupNames.length }), 'success')
  } catch (e: any) {
    showToast(e.message || t('message.error'), 'error')
  } finally {
    batchOperating.value = false
  }
}

// 文件夾操作
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

const deleteFolder = async (folder: any) => {
  const backupsInFolder = getBackupsInFolder(folder.id)
  
  if (backupsInFolder.some((b: any) => b.isCurrent)) {
    showToast(t('folder.cannotDeleteActive'), 'error')
    return
  }
  
  if (backupsInFolder.length > 0) {
    folderToDelete.value = folder
    showDeleteFolderDialog.value = true
  } else {
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

// 拖放操作
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
    
    selectedBackups.value = new Set()
    await loadBackups(false)
    await loadFolders()
  } catch (e: any) {
    showToast(e.message || t('message.error'), 'error')
  }
}

// Phase 2 Task 7.3: 簡化設定操作 - 調用 Composable 方法 + 本地狀態更新回調
const saveLowBalanceThreshold = async (value: number) => {
  await saveLowBalanceThresholdCore(value, (threshold: number) => {
    // 更新所有備份的 isLowBalance 狀態
    backups.value.forEach(backup => {
      if (backup.usageLimit > 0) {
        backup.isLowBalance = (backup.balance / backup.usageLimit) < threshold
      }
    })
    // 更新當前帳號的 isLowBalance 狀態
    if (currentUsageInfo.value && currentUsageInfo.value.usageLimit > 0) {
      currentUsageInfo.value.isLowBalance = 
        (currentUsageInfo.value.balance / currentUsageInfo.value.usageLimit) < threshold
    }
  })
}

// Phase 2 Task 2: 包裝函數 - 調用 Composable 方法 + Toast 通知
const saveKiroVersion = async () => {
  const version = kiroVersionInput.value.trim()
  if (!version) return
  
  await saveKiroVersionCore()
  // 檢查是否保存成功（通過檢查 modified 狀態）
  if (!kiroVersionModified.value) {
    showToast(t('message.success'), 'success')
  }
}

const detectKiroVersion = async () => {
  await detectKiroVersionCore()
  // 檢查是否偵測成功（通過檢查 modified 狀態和 useAutoDetect）
  if (appSettings.value.useAutoDetect && !kiroVersionModified.value) {
    showToast(t('message.success'), 'success')
  } else if (!appSettings.value.useAutoDetect) {
    showToast(t('settings.detectVersionFailed'), 'error')
  }
}

const saveKiroInstallPath = async () => {
  await saveKiroInstallPathCore()
  // 檢查是否保存成功
  if (!kiroInstallPathModified.value) {
    showToast(t('message.success'), 'success')
  }
}

const detectKiroInstallPath = async () => {
  const previousPath = appSettings.value.customKiroInstallPath
  await detectKiroInstallPathCore()
  // 檢查是否偵測成功（路徑有變化且 modified 為 false）
  if (appSettings.value.customKiroInstallPath && !kiroInstallPathModified.value) {
    showToast(t('message.success'), 'success')
  } else if (!appSettings.value.customKiroInstallPath && previousPath === appSettings.value.customKiroInstallPath) {
    showToast(t('settings.detectPathFailed'), 'error')
  }
}

const clearKiroInstallPath = async () => {
  await clearKiroInstallPathCore()
  // 檢查是否清除成功
  if (appSettings.value.customKiroInstallPath === '' && !kiroInstallPathModified.value) {
    showToast(t('message.success'), 'success')
  }
}

// switchLanguage 已從 useAppSettings 解構，不需要重複定義

// 自動切換
const toggleAutoSwitch = async () => {
  // P1-2 FIX: 並發保護 - 防止快速連續點擊
  if (savingAutoSwitch.value) return
  savingAutoSwitch.value = true
  
  try {
    await saveAutoSwitchSettings()
    
    if (autoSwitchSettings.value.enabled) {
      const result = await window.go.main.App.StartAutoSwitchMonitor()
      if (!result.success) {
        showToast(result.message, 'error')
        autoSwitchSettings.value.enabled = false
        await saveAutoSwitchSettings()
      }
    } else {
      await window.go.main.App.StopAutoSwitchMonitor()
    }
    
    const status = await window.go.main.App.GetAutoSwitchStatus()
    autoSwitchStatus.value = {
      status: status.status as 'stopped' | 'running' | 'cooldown',
      lastBalance: status.lastBalance,
      cooldownRemaining: status.cooldownRemaining,
      switchCount: status.switchCount,
    }
  } finally {
    savingAutoSwitch.value = false
  }
}

const handleAutoSwitchToggle = async (enabled: boolean) => {
  autoSwitchSettings.value.enabled = enabled
  await toggleAutoSwitch()
}

// 路徑偵測狀態檢查
const checkPathDetectionStatus = async () => {
  try {
    const result = await window.go.main.App.GetKiroInstallPathWithStatus()
    if (!result.success) {
      activeMenu.value = 'settings'
      showSettingsPanel.value = true
      showToast(t('message.pathDetectionFailed'), 'error')
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
  // P0-FIX: 保存 interval ID 用於 onUnmounted 清理
  kiroStatusInterval = setInterval(checkKiroStatus, 5000)
  
  // 監聽視窗大小變化
  window.addEventListener('resize', saveWindowSize)
  
  // 自動聚焦主內容滾動區，解決滾輪無法滾動的問題
  setTimeout(() => {
    mainScrollArea.value?.focus()
  }, 100)
  
  // 監聽自動切換事件
  EventsOn('auto-switch', (data: AutoSwitchEvent) => {
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
  
  // P0-FIX: 清理 Kiro 狀態檢查 interval
  if (kiroStatusInterval) {
    clearInterval(kiroStatusInterval)
    kiroStatusInterval = null
  }
  
  // P0-FIX: 清理 useUsageRefresh 的所有 intervals
  cleanupUsageRefresh()
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
          <CurrentStatusCard
            :current-environment-name="currentEnvironmentName"
            :current-machine-id="currentMachineId"
            :current-provider="currentProvider"
            :usage-info="currentUsageInfo"
            :active-backup="activeBackup"
            :is-refreshing="refreshingCurrent"
            :is-restoring="restoringOriginal"
            :cooldown-seconds="countdownCurrentAccount"
            @refresh="refreshCurrentUsage"
            @create-backup="showCreateModal = true"
            @restore-original="restoreOriginal"
            @open-sso-cache="openSSOCacheFolder"
          />

          <!-- PATCH 狀態 + 一鍵新機合併卡片 -->
          <SoftResetCard
            :soft-reset-status="softResetStatus"
            :is-resetting="resetting"
            :is-patching="patching"
            @reset="resetToNew"
            @patch="patchExtension"
            @restore="restoreOriginal"
            @open-extension-folder="openExtensionFolder"
            @open-machine-id-folder="openMachineIDFolder"
          />
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

          <!-- 文件夾列表和未分類區域 -->
          <FolderTree
            :folders="folders"
            :backups="filteredBackups"
            :expanded-folders="expandedFolders"
            :uncategorized-expanded="uncategorizedExpanded"
            :drag-over-folder-id="dragOverFolderId"
            :drag-over-uncategorized="dragOverUncategorized"
            :renaming-folder="renamingFolder"
            :rename-folder-name="renameFolderName"
            :deleting-folder="deletingFolder"
            :selected-backups="selectedBackups"
            :switching-backup="switchingBackup"
            :deleting-backup="deletingBackup"
            :refreshing-backup="refreshingBackup"
            :regenerating-id="regeneratingId"
            :countdown-timers="countdownTimers"
            :copied-machine-id="copiedMachineId"
            @toggle-folder="toggleFolder"
            @toggle-uncategorized="toggleUncategorized"
            @start-rename-folder="startRenameFolder"
            @confirm-rename-folder="confirmRenameFolder"
            @cancel-rename-folder="cancelRenameFolder"
            @delete-folder="deleteFolder"
            @create-folder="showCreateFolderModal = true"
            @drag-enter-folder="onDragEnterFolder"
            @drag-leave-folder="onDragLeaveFolder"
            @drag-enter-uncategorized="onDragEnterUncategorized"
            @drag-leave-uncategorized="onDragLeaveUncategorized"
            @drop-to-folder="onDropToFolder"
            @drop-to-uncategorized="onDropToUncategorized"
            @toggle-select="toggleSelect"
            @toggle-select-all="toggleSelectAll"
            @switch-backup="switchToBackup"
            @delete-backup="deleteBackup"
            @refresh-backup="refreshBackupUsage"
            @regenerate-id="regenerateMachineID"
            @copy-machine-id="copyMachineId"
            @drag-start="onDragStart"
            @drag-end="() => { dragOverFolderId = null; dragOverUncategorized = false }"
          />
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
