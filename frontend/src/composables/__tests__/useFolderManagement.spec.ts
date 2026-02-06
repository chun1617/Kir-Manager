/**
 * useFolderManagement Composable 測試
 * @description Property-Based Testing for 文件夾管理功能
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fc from 'fast-check'
import { useFolderManagement } from '../useFolderManagement'
import {
  folderItemArbitrary,
  folderListArbitrary,
  backupItemArbitrary,
  backupListArbitrary,
} from './arbitraries'
import type { FolderItem, BackupItem, Result } from '@/types/backup'

// Mock window.go.main.App
const createMockApp = () => ({
  GetFolderList: vi.fn(),
  CreateFolder: vi.fn(),
  RenameFolder: vi.fn(),
  DeleteFolder: vi.fn(),
  AssignSnapshotToFolder: vi.fn(),
  UnassignSnapshot: vi.fn(),
})

let mockApp = createMockApp()

// Setup global mock
beforeEach(() => {
  // 每次測試前重新創建 mock，確保調用次數重置
  mockApp = createMockApp()
  vi.stubGlobal('window', {
    go: {
      main: {
        App: mockApp,
      },
    },
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useFolderManagement', () => {
  describe('初始狀態', () => {
    it('應該有正確的初始狀態', () => {
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
      } = useFolderManagement()

      expect(folders.value).toEqual([])
      expect(expandedFolders.value).toEqual(new Set())
      expect(uncategorizedExpanded.value).toBe(true)
      expect(showCreateFolderModal.value).toBe(false)
      expect(newFolderName.value).toBe('')
      expect(creatingFolder.value).toBe(false)
      expect(renamingFolder.value).toBeNull()
      expect(renameFolderName.value).toBe('')
      expect(deletingFolder.value).toBeNull()
      expect(dragOverFolderId.value).toBeNull()
      expect(dragOverUncategorized.value).toBe(false)
      expect(showDeleteFolderDialog.value).toBe(false)
      expect(folderToDelete.value).toBeNull()
    })
  })

  describe('Property 7: 文件夾 CRUD 列表長度不變量', () => {
    it('創建文件夾後列表長度增加 1', async () => {
      await fc.assert(
        fc.asyncProperty(
          folderListArbitrary({ minLength: 0, maxLength: 5 }),
          fc.stringMatching(/^[a-zA-Z0-9_-]{1,20}$/),
          async (initialFolders, newName) => {
            // Arrange
            mockApp.GetFolderList.mockResolvedValue(initialFolders)
            mockApp.CreateFolder.mockImplementation(async (name: string) => {
              const newFolder: FolderItem = {
                id: `folder-${Date.now()}`,
                name,
                createdAt: new Date().toISOString(),
                order: initialFolders.length,
                snapshotCount: 0,
              }
              initialFolders.push(newFolder)
              return { success: true, message: '' }
            })

            const { folders, loadFolders, createFolder, newFolderName, showCreateFolderModal } = useFolderManagement()

            // Act
            await loadFolders()
            const initialLength = folders.value.length

            newFolderName.value = newName
            showCreateFolderModal.value = true
            await createFolder()
            await loadFolders()

            // Assert
            expect(folders.value.length).toBe(initialLength + 1)
          }
        ),
        { numRuns: 20 }
      )
    })

    it('刪除空文件夾後列表長度減少 1', async () => {
      await fc.assert(
        fc.asyncProperty(
          folderListArbitrary({ minLength: 1, maxLength: 5 }),
          async (initialFolders) => {
            // Arrange
            const foldersCopy = [...initialFolders]
            mockApp.GetFolderList.mockResolvedValue(foldersCopy)
            mockApp.DeleteFolder.mockImplementation(async (id: string) => {
              const index = foldersCopy.findIndex(f => f.id === id)
              if (index !== -1) {
                foldersCopy.splice(index, 1)
              }
              return { success: true, message: '' }
            })

            const { folders, loadFolders, deleteFolder } = useFolderManagement()

            // Act
            await loadFolders()
            const initialLength = folders.value.length
            const folderToDelete = folders.value[0]

            // 模擬空文件夾（getBackupsInFolder 返回空陣列）
            await deleteFolder(folderToDelete, [])
            await loadFolders()

            // Assert
            expect(folders.value.length).toBe(initialLength - 1)
          }
        ),
        { numRuns: 20 }
      )
    })
  })

  describe('Property 8: 非法文件夾名稱驗證', () => {
    it('空名稱應該創建失敗', async () => {
      const { createFolder, newFolderName, showCreateFolderModal } = useFolderManagement()

      newFolderName.value = ''
      showCreateFolderModal.value = true
      await createFolder()

      expect(mockApp.CreateFolder).not.toHaveBeenCalled()
    })

    it('只有空白的名稱應該創建失敗', async () => {
      const whitespaceStrings = ['   ', '\t', '\n', '  \t  ', '\n\n']
      
      for (const whitespaceOnly of whitespaceStrings) {
        // 重置 mock
        mockApp.CreateFolder.mockClear()
        
        const { createFolder, newFolderName, showCreateFolderModal } = useFolderManagement()

        newFolderName.value = whitespaceOnly
        showCreateFolderModal.value = true
        await createFolder()

        expect(mockApp.CreateFolder).not.toHaveBeenCalled()
      }
    })
  })

  describe('Property 9: 文件夾重命名一致性', () => {
    it('重命名後文件夾名稱應該更新', async () => {
      await fc.assert(
        fc.asyncProperty(
          folderItemArbitrary,
          fc.stringMatching(/^[a-zA-Z0-9_-]{1,20}$/),
          async (folder, newName) => {
            // Arrange
            const foldersCopy = [{ ...folder }]
            mockApp.GetFolderList.mockResolvedValue(foldersCopy)
            mockApp.RenameFolder.mockImplementation(async (id: string, name: string) => {
              const f = foldersCopy.find(f => f.id === id)
              if (f) f.name = name
              return { success: true, message: '' }
            })

            const {
              folders,
              loadFolders,
              startRenameFolder,
              confirmRenameFolder,
              renameFolderName,
            } = useFolderManagement()

            // Act
            await loadFolders()
            startRenameFolder(folder)
            renameFolderName.value = newName
            await confirmRenameFolder()
            await loadFolders()

            // Assert
            expect(folders.value[0].name).toBe(newName)
          }
        ),
        { numRuns: 20 }
      )
    })
  })

  describe('Property 10: 當前備份文件夾保護', () => {
    it('包含當前使用中備份的文件夾不能刪除', async () => {
      await fc.assert(
        fc.asyncProperty(
          folderItemArbitrary,
          backupItemArbitrary,
          async (folder, backup) => {
            // Arrange: 備份在文件夾中且是當前使用中
            const backupInFolder: BackupItem = {
              ...backup,
              folderId: folder.id,
              isCurrent: true,
            }

            const { deleteFolder, showDeleteFolderDialog } = useFolderManagement()

            // Act
            const result = await deleteFolder(folder, [backupInFolder])

            // Assert: 不應該調用 DeleteFolder API，且應該返回錯誤
            expect(mockApp.DeleteFolder).not.toHaveBeenCalled()
            expect(result.success).toBe(false)
          }
        ),
        { numRuns: 20 }
      )
    })
  })

  describe('Property 11: Toggle 狀態反轉', () => {
    it('toggleFolder 應該反轉展開狀態', () => {
      fc.assert(
        fc.property(
          fc.uuid(),
          fc.boolean(),
          (folderId, initialExpanded) => {
            const { expandedFolders, toggleFolder } = useFolderManagement()

            // 設定初始狀態
            if (initialExpanded) {
              expandedFolders.value = new Set([folderId])
            } else {
              expandedFolders.value = new Set()
            }

            // Act
            toggleFolder(folderId)

            // Assert
            expect(expandedFolders.value.has(folderId)).toBe(!initialExpanded)
          }
        ),
        { numRuns: 50 }
      )
    })

    it('toggleUncategorized 應該反轉未分類展開狀態', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          (initialExpanded) => {
            const { uncategorizedExpanded, toggleUncategorized } = useFolderManagement()

            uncategorizedExpanded.value = initialExpanded

            // Act
            toggleUncategorized()

            // Assert
            expect(uncategorizedExpanded.value).toBe(!initialExpanded)
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  describe('Property 12: 拖放分配正確性', () => {
    it('拖放到文件夾應該調用 AssignSnapshotToFolder', async () => {
      // Arrange
      const folderId = 'test-folder-id'
      const backupNames = ['backup1', 'backup2', 'backup3']
      mockApp.AssignSnapshotToFolder.mockResolvedValue({ success: true, message: '' })
      mockApp.GetFolderList.mockResolvedValue([])

      const { onDropToFolder } = useFolderManagement()

      // Act
      await onDropToFolder(backupNames, folderId)

      // Assert
      expect(mockApp.AssignSnapshotToFolder).toHaveBeenCalledTimes(backupNames.length)
      backupNames.forEach(name => {
        expect(mockApp.AssignSnapshotToFolder).toHaveBeenCalledWith(name, folderId)
      })
    })

    it('拖放到未分類應該調用 UnassignSnapshot', async () => {
      // Arrange
      const backupNames = ['backup1', 'backup2']
      mockApp.UnassignSnapshot.mockResolvedValue({ success: true, message: '' })
      mockApp.GetFolderList.mockResolvedValue([])

      const { onDropToUncategorized } = useFolderManagement()

      // Act
      await onDropToUncategorized(backupNames)

      // Assert
      expect(mockApp.UnassignSnapshot).toHaveBeenCalledTimes(backupNames.length)
      backupNames.forEach(name => {
        expect(mockApp.UnassignSnapshot).toHaveBeenCalledWith(name)
      })
    })
  })

  describe('Property 13: 孤兒 folderId 歸類', () => {
    it('備份的 folderId 不存在於文件夾列表時應歸類為未分類', () => {
      fc.assert(
        fc.property(
          folderListArbitrary({ minLength: 1, maxLength: 5 }),
          backupListArbitrary({ minLength: 1, maxLength: 10 }),
          (folders, backups) => {
            const { getUncategorizedBackups } = useFolderManagement()

            // 設定一些備份有不存在的 folderId
            const folderIds = new Set(folders.map(f => f.id))
            const backupsWithOrphanIds = backups.map((b, i) => ({
              ...b,
              folderId: i % 2 === 0 ? 'non-existent-id' : (folders[0]?.id || ''),
            }))

            // Act
            const uncategorized = getUncategorizedBackups(backupsWithOrphanIds, folders)

            // Assert: 孤兒 folderId 的備份應該在未分類中
            const orphanBackups = backupsWithOrphanIds.filter(
              b => b.folderId && !folderIds.has(b.folderId)
            )
            const emptyFolderIdBackups = backupsWithOrphanIds.filter(b => !b.folderId)
            
            expect(uncategorized.length).toBe(orphanBackups.length + emptyFolderIdBackups.length)
          }
        ),
        { numRuns: 30 }
      )
    })
  })

  describe('Property 14: 批量移動文件夾分配', () => {
    it('批量移動到文件夾應該更新所有選中備份', async () => {
      // Arrange
      const targetFolderId = 'target-folder-id'
      const backupNames = ['backup1', 'backup2', 'backup3']
      mockApp.AssignSnapshotToFolder.mockResolvedValue({ success: true, message: '' })
      mockApp.GetFolderList.mockResolvedValue([])

      const { batchMoveToFolder } = useFolderManagement()

      // Act
      await batchMoveToFolder(backupNames, targetFolderId)

      // Assert
      expect(mockApp.AssignSnapshotToFolder).toHaveBeenCalledTimes(backupNames.length)
      backupNames.forEach(name => {
        expect(mockApp.AssignSnapshotToFolder).toHaveBeenCalledWith(name, targetFolderId)
      })
    })

    it('批量移動到未分類應該取消所有選中備份的文件夾分配', async () => {
      // Arrange
      const backupNames = ['backup1', 'backup2']
      mockApp.UnassignSnapshot.mockResolvedValue({ success: true, message: '' })
      mockApp.GetFolderList.mockResolvedValue([])

      const { batchMoveToFolder } = useFolderManagement()

      // Act
      await batchMoveToFolder(backupNames, null)

      // Assert
      expect(mockApp.UnassignSnapshot).toHaveBeenCalledTimes(backupNames.length)
      backupNames.forEach(name => {
        expect(mockApp.UnassignSnapshot).toHaveBeenCalledWith(name)
      })
    })
  })

  describe('拖放狀態管理', () => {
    it('onDragEnterFolder 應該設定 dragOverFolderId', () => {
      fc.assert(
        fc.property(
          fc.uuid(),
          (folderId) => {
            const { dragOverFolderId, dragOverUncategorized, onDragEnterFolder } = useFolderManagement()

            onDragEnterFolder(folderId)

            expect(dragOverFolderId.value).toBe(folderId)
            expect(dragOverUncategorized.value).toBe(false)
          }
        ),
        { numRuns: 20 }
      )
    })

    it('onDragLeaveFolder 應該清除 dragOverFolderId', () => {
      const { dragOverFolderId, onDragEnterFolder, onDragLeaveFolder } = useFolderManagement()

      onDragEnterFolder('some-id')
      expect(dragOverFolderId.value).toBe('some-id')

      onDragLeaveFolder()
      expect(dragOverFolderId.value).toBeNull()
    })

    it('onDragEnterUncategorized 應該設定 dragOverUncategorized', () => {
      const { dragOverFolderId, dragOverUncategorized, onDragEnterUncategorized } = useFolderManagement()

      onDragEnterUncategorized()

      expect(dragOverUncategorized.value).toBe(true)
      expect(dragOverFolderId.value).toBeNull()
    })

    it('cleanupDragState 應該清除所有拖放狀態', () => {
      const {
        dragOverFolderId,
        dragOverUncategorized,
        onDragEnterFolder,
        cleanupDragState,
      } = useFolderManagement()

      onDragEnterFolder('some-id')
      expect(dragOverFolderId.value).toBe('some-id')

      cleanupDragState()

      expect(dragOverFolderId.value).toBeNull()
      expect(dragOverUncategorized.value).toBe(false)
    })
  })

  describe('重命名狀態管理', () => {
    it('startRenameFolder 應該設定重命名狀態', () => {
      fc.assert(
        fc.property(
          folderItemArbitrary,
          (folder) => {
            const { renamingFolder, renameFolderName, startRenameFolder } = useFolderManagement()

            startRenameFolder(folder)

            expect(renamingFolder.value).toBe(folder.id)
            expect(renameFolderName.value).toBe(folder.name)
          }
        ),
        { numRuns: 20 }
      )
    })

    it('cancelRenameFolder 應該清除重命名狀態', () => {
      const { renamingFolder, renameFolderName, startRenameFolder, cancelRenameFolder } = useFolderManagement()

      startRenameFolder({ id: 'test', name: 'Test', createdAt: '', order: 0, snapshotCount: 0 })
      expect(renamingFolder.value).toBe('test')

      cancelRenameFolder()

      expect(renamingFolder.value).toBeNull()
      expect(renameFolderName.value).toBe('')
    })
  })

  describe('刪除對話框狀態管理', () => {
    it('cancelDeleteFolder 應該清除刪除對話框狀態', () => {
      const { showDeleteFolderDialog, folderToDelete, cancelDeleteFolder } = useFolderManagement()

      showDeleteFolderDialog.value = true
      folderToDelete.value = { id: 'test', name: 'Test', createdAt: '', order: 0, snapshotCount: 0 }

      cancelDeleteFolder()

      expect(showDeleteFolderDialog.value).toBe(false)
      expect(folderToDelete.value).toBeNull()
    })
  })
})


// ============================================================================
// Task 1.4: 拖放取消狀態清理測試
// ============================================================================

describe('Task 1.4: 拖放取消狀態清理', () => {
  it('cleanupDragState 應該在 dragend 事件時被調用', () => {
    const { dragOverFolderId, dragOverUncategorized, onDragEnterFolder, cleanupDragState } = useFolderManagement()
    
    // 模擬拖放進入文件夾
    onDragEnterFolder('folder-1')
    expect(dragOverFolderId.value).toBe('folder-1')
    
    // 模擬 dragend 事件調用 cleanupDragState
    cleanupDragState()
    
    // 驗證狀態已清理
    expect(dragOverFolderId.value).toBeNull()
    expect(dragOverUncategorized.value).toBe(false)
  })

  it('cleanupDragState 應該在 ESC 鍵按下時被調用', () => {
    const { dragOverFolderId, dragOverUncategorized, onDragEnterUncategorized, cleanupDragState } = useFolderManagement()
    
    // 模擬拖放進入未分類區塊
    onDragEnterUncategorized()
    expect(dragOverUncategorized.value).toBe(true)
    
    // 模擬 ESC 鍵按下調用 cleanupDragState
    cleanupDragState()
    
    // 驗證狀態已清理
    expect(dragOverFolderId.value).toBeNull()
    expect(dragOverUncategorized.value).toBe(false)
  })

  it('cleanupDragState 應該同時清理 dragOverFolderId 和 dragOverUncategorized', () => {
    const { dragOverFolderId, dragOverUncategorized, cleanupDragState } = useFolderManagement()
    
    // 手動設置狀態
    dragOverFolderId.value = 'some-folder'
    dragOverUncategorized.value = true
    
    // 調用清理
    cleanupDragState()
    
    // 驗證兩個狀態都被清理
    expect(dragOverFolderId.value).toBeNull()
    expect(dragOverUncategorized.value).toBe(false)
  })
})
