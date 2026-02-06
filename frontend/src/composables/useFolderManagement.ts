/**
 * useFolderManagement Composable
 * @description 文件夾管理核心 Composable，負責管理文件夾列表、拖放、重命名等功能
 * @see App.vue - 原始實作參考
 */
import { ref, type Ref } from 'vue'
import type { FolderItem, BackupItem, Result } from '@/types/backup'

/**
 * 文件夾管理 Composable 返回類型
 */
export interface UseFolderManagementReturn {
  // 狀態
  folders: Ref<FolderItem[]>
  expandedFolders: Ref<Set<string>>
  uncategorizedExpanded: Ref<boolean>
  showCreateFolderModal: Ref<boolean>
  newFolderName: Ref<string>
  creatingFolder: Ref<boolean>
  renamingFolder: Ref<string | null>
  renameFolderName: Ref<string>
  deletingFolder: Ref<string | null>
  dragOverFolderId: Ref<string | null>
  dragOverUncategorized: Ref<boolean>
  showDeleteFolderDialog: Ref<boolean>
  folderToDelete: Ref<FolderItem | null>
  showMoveToFolderDropdown: Ref<boolean>

  // 方法
  loadFolders: () => Promise<void>
  createFolder: () => Promise<Result>
  startRenameFolder: (folder: FolderItem) => void
  confirmRenameFolder: () => Promise<Result>
  cancelRenameFolder: () => void
  deleteFolder: (folder: FolderItem, backupsInFolder: BackupItem[]) => Promise<Result>
  confirmDeleteFolder: (deleteSnapshots: boolean) => Promise<Result>
  cancelDeleteFolder: () => void
  toggleFolder: (folderId: string) => void
  toggleUncategorized: () => void
  getBackupsInFolder: (backups: BackupItem[], folderId: string) => BackupItem[]
  getUncategorizedBackups: (backups: BackupItem[], folders: FolderItem[]) => BackupItem[]

  // 拖放方法
  onDragEnterFolder: (folderId: string) => void
  onDragLeaveFolder: () => void
  onDragEnterUncategorized: () => void
  onDragLeaveUncategorized: () => void
  onDropToFolder: (backupNames: string[], folderId: string) => Promise<void>
  onDropToUncategorized: (backupNames: string[]) => Promise<void>
  cleanupDragState: () => void

  // 批量操作
  batchMoveToFolder: (backupNames: string[], folderId: string | null) => Promise<void>
}

/**
 * 文件夾管理 Composable
 * @returns 文件夾管理相關的狀態和方法
 */
export function useFolderManagement(): UseFolderManagementReturn {
  // ============================================================================
  // 狀態定義
  // ============================================================================

  /** 文件夾列表 */
  const folders = ref<FolderItem[]>([])

  /** 展開的文件夾 ID 集合 */
  const expandedFolders = ref<Set<string>>(new Set())

  /** 未分類區塊是否展開 */
  const uncategorizedExpanded = ref<boolean>(true)

  /** 是否顯示創建文件夾對話框 */
  const showCreateFolderModal = ref<boolean>(false)

  /** 新文件夾名稱 */
  const newFolderName = ref<string>('')

  /** 是否正在創建文件夾 */
  const creatingFolder = ref<boolean>(false)

  /** 正在重命名的文件夾 ID */
  const renamingFolder = ref<string | null>(null)

  /** 重命名文件夾的新名稱 */
  const renameFolderName = ref<string>('')

  /** 正在刪除的文件夾 ID */
  const deletingFolder = ref<string | null>(null)

  /** 拖放懸停的文件夾 ID */
  const dragOverFolderId = ref<string | null>(null)

  /** 是否拖放懸停在未分類區塊 */
  const dragOverUncategorized = ref<boolean>(false)

  /** 是否顯示刪除文件夾對話框 */
  const showDeleteFolderDialog = ref<boolean>(false)

  /** 待刪除的文件夾 */
  const folderToDelete = ref<FolderItem | null>(null)

  /** 是否顯示移動到文件夾下拉選單 */
  const showMoveToFolderDropdown = ref<boolean>(false)

  // ============================================================================
  // 方法
  // ============================================================================

  /**
   * 載入文件夾列表
   */
  const loadFolders = async (): Promise<void> => {
    try {
      folders.value = await window.go.main.App.GetFolderList() || []
    } catch (e) {
      console.error('Failed to load folders:', e)
    }
  }

  /**
   * 創建文件夾
   * @returns 操作結果
   */
  const createFolder = async (): Promise<Result> => {
    const name = newFolderName.value.trim()
    if (!name) {
      return { success: false, message: '文件夾名稱不能為空' }
    }

    creatingFolder.value = true
    try {
      const result = await window.go.main.App.CreateFolder(name)
      if (result.success) {
        showCreateFolderModal.value = false
        newFolderName.value = ''
        await loadFolders()
      }
      return result
    } catch (e: any) {
      return { success: false, message: e.message || '創建文件夾失敗' }
    } finally {
      creatingFolder.value = false
    }
  }

  /**
   * 開始重命名文件夾
   * @param folder 要重命名的文件夾
   */
  const startRenameFolder = (folder: FolderItem): void => {
    renamingFolder.value = folder.id
    renameFolderName.value = folder.name
  }

  /**
   * 確認重命名文件夾
   * @returns 操作結果
   */
  const confirmRenameFolder = async (): Promise<Result> => {
    if (!renamingFolder.value) {
      return { success: false, message: '沒有正在重命名的文件夾' }
    }

    const name = renameFolderName.value.trim()
    if (!name) {
      return { success: false, message: '文件夾名稱不能為空' }
    }

    try {
      const result = await window.go.main.App.RenameFolder(renamingFolder.value, name)
      if (result.success) {
        await loadFolders()
      }
      return result
    } catch (e: any) {
      return { success: false, message: e.message || '重命名文件夾失敗' }
    } finally {
      renamingFolder.value = null
      renameFolderName.value = ''
    }
  }

  /**
   * 取消重命名文件夾
   */
  const cancelRenameFolder = (): void => {
    renamingFolder.value = null
    renameFolderName.value = ''
  }

  /**
   * 刪除文件夾
   * @param folder 要刪除的文件夾
   * @param backupsInFolder 文件夾中的備份列表
   * @returns 操作結果
   */
  const deleteFolder = async (folder: FolderItem, backupsInFolder: BackupItem[]): Promise<Result> => {
    // Property 10: 檢查是否包含當前使用中的快照
    if (backupsInFolder.some(b => b.isCurrent)) {
      return { success: false, message: '無法刪除包含當前使用中快照的文件夾' }
    }

    if (backupsInFolder.length > 0) {
      // 非空文件夾，顯示專用對話框
      folderToDelete.value = folder
      showDeleteFolderDialog.value = true
      return { success: true, message: '顯示刪除確認對話框' }
    } else {
      // 空文件夾，直接刪除
      deletingFolder.value = folder.id
      try {
        const result = await window.go.main.App.DeleteFolder(folder.id, false)
        if (result.success) {
          await loadFolders()
        }
        return result
      } catch (e: any) {
        return { success: false, message: e.message || '刪除文件夾失敗' }
      } finally {
        deletingFolder.value = null
      }
    }
  }

  /**
   * 確認刪除文件夾（處理用戶選擇）
   * @param deleteSnapshots 是否一併刪除快照
   * @returns 操作結果
   */
  const confirmDeleteFolder = async (deleteSnapshots: boolean): Promise<Result> => {
    if (!folderToDelete.value) {
      return { success: false, message: '沒有待刪除的文件夾' }
    }

    const folder = folderToDelete.value
    showDeleteFolderDialog.value = false
    deletingFolder.value = folder.id

    try {
      const result = await window.go.main.App.DeleteFolder(folder.id, deleteSnapshots)
      if (result.success) {
        await loadFolders()
      }
      return result
    } catch (e: any) {
      return { success: false, message: e.message || '刪除文件夾失敗' }
    } finally {
      deletingFolder.value = null
      folderToDelete.value = null
    }
  }

  /**
   * 取消刪除文件夾對話框
   */
  const cancelDeleteFolder = (): void => {
    showDeleteFolderDialog.value = false
    folderToDelete.value = null
  }

  /**
   * 切換文件夾展開狀態
   * @param folderId 文件夾 ID
   */
  const toggleFolder = (folderId: string): void => {
    const newSet = new Set(expandedFolders.value)
    if (newSet.has(folderId)) {
      newSet.delete(folderId)
    } else {
      newSet.add(folderId)
    }
    expandedFolders.value = newSet
  }

  /**
   * 切換未分類區塊展開狀態
   */
  const toggleUncategorized = (): void => {
    uncategorizedExpanded.value = !uncategorizedExpanded.value
  }

  /**
   * 獲取指定文件夾中的備份
   * @param backups 備份列表
   * @param folderId 文件夾 ID
   * @returns 文件夾中的備份列表
   */
  const getBackupsInFolder = (backups: BackupItem[], folderId: string): BackupItem[] => {
    return backups.filter(b => b.folderId === folderId)
  }

  /**
   * 獲取未分類的備份（包含孤兒 folderId）
   * @param backups 備份列表
   * @param folderList 文件夾列表
   * @returns 未分類的備份列表
   */
  const getUncategorizedBackups = (backups: BackupItem[], folderList: FolderItem[]): BackupItem[] => {
    const folderIds = new Set(folderList.map(f => f.id))
    return backups.filter(b => !b.folderId || !folderIds.has(b.folderId))
  }

  // ============================================================================
  // 拖放方法
  // ============================================================================

  /**
   * 拖放進入文件夾
   * @param folderId 文件夾 ID
   */
  const onDragEnterFolder = (folderId: string): void => {
    dragOverFolderId.value = folderId
    dragOverUncategorized.value = false
  }

  /**
   * 拖放離開文件夾
   */
  const onDragLeaveFolder = (): void => {
    dragOverFolderId.value = null
  }

  /**
   * 拖放進入未分類區塊
   */
  const onDragEnterUncategorized = (): void => {
    dragOverUncategorized.value = true
    dragOverFolderId.value = null
  }

  /**
   * 拖放離開未分類區塊
   */
  const onDragLeaveUncategorized = (): void => {
    dragOverUncategorized.value = false
  }

  /**
   * 拖放到文件夾
   * @param backupNames 備份名稱列表
   * @param folderId 目標文件夾 ID
   */
  const onDropToFolder = async (backupNames: string[], folderId: string): Promise<void> => {
    dragOverFolderId.value = null

    try {
      for (const name of backupNames) {
        await window.go.main.App.AssignSnapshotToFolder(name, folderId)
      }
      await loadFolders()
    } catch (e: any) {
      console.error('Failed to assign snapshots to folder:', e)
    }
  }

  /**
   * 拖放到未分類區塊
   * @param backupNames 備份名稱列表
   */
  const onDropToUncategorized = async (backupNames: string[]): Promise<void> => {
    dragOverUncategorized.value = false

    try {
      for (const name of backupNames) {
        await window.go.main.App.UnassignSnapshot(name)
      }
      await loadFolders()
    } catch (e: any) {
      console.error('Failed to unassign snapshots:', e)
    }
  }

  /**
   * 清理拖放狀態（P1: 拖放取消時清理）
   */
  const cleanupDragState = (): void => {
    dragOverFolderId.value = null
    dragOverUncategorized.value = false
  }

  // ============================================================================
  // 批量操作
  // ============================================================================

  /**
   * 批量移動到文件夾
   * @param backupNames 備份名稱列表
   * @param folderId 目標文件夾 ID（null 表示移到未分類）
   */
  const batchMoveToFolder = async (backupNames: string[], folderId: string | null): Promise<void> => {
    if (backupNames.length === 0) return

    showMoveToFolderDropdown.value = false

    try {
      for (const name of backupNames) {
        if (folderId) {
          await window.go.main.App.AssignSnapshotToFolder(name, folderId)
        } else {
          await window.go.main.App.UnassignSnapshot(name)
        }
      }
      await loadFolders()
    } catch (e: any) {
      console.error('Failed to batch move to folder:', e)
    }
  }

  // ============================================================================
  // 返回
  // ============================================================================

  return {
    // 狀態
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
    showMoveToFolderDropdown,

    // 方法
    loadFolders,
    createFolder,
    startRenameFolder,
    confirmRenameFolder,
    cancelRenameFolder,
    deleteFolder,
    confirmDeleteFolder,
    cancelDeleteFolder,
    toggleFolder,
    toggleUncategorized,
    getBackupsInFolder,
    getUncategorizedBackups,

    // 拖放方法
    onDragEnterFolder,
    onDragLeaveFolder,
    onDragEnterUncategorized,
    onDragLeaveUncategorized,
    onDropToFolder,
    onDropToUncategorized,
    cleanupDragState,

    // 批量操作
    batchMoveToFolder,
  }
}
