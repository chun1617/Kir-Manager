/**
 * FolderTree 組件測試
 * @description 測試文件夾樹狀結構組件的渲染和事件處理
 */
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import FolderTree from '../FolderTree.vue'
import type { FolderItem, BackupItem } from '@/types/backup'

// 建立測試用 i18n 實例
const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      folder: {
        rename: 'Rename',
        delete: 'Delete',
        emptyFolder: 'Empty folder',
        uncategorized: 'Uncategorized',
      },
      backup: {
        noBackups: 'No backups',
      },
    },
  },
})

// Mock Icon component
const IconStub = {
  name: 'Icon',
  template: '<span class="icon-stub" :data-name="name"></span>',
  props: ['name'],
}

// Mock BackupCard component
const BackupCardStub = {
  name: 'BackupCard',
  template: '<div class="backup-card-stub" :data-name="backup.name"></div>',
  props: ['backup', 'isSelected', 'isSwitching', 'isDeleting', 'isRefreshing', 'isRegenerating', 'cooldownSeconds', 'copiedMachineId'],
  emits: ['select', 'switch', 'delete', 'refresh', 'regenerate-id', 'copy-machine-id', 'drag-start', 'drag-end'],
}

// 建立預設的 folder 物件
const createDefaultFolder = (overrides: Partial<FolderItem> = {}): FolderItem => ({
  id: 'folder-1',
  name: 'Test Folder',
  createdAt: '2024-01-01T00:00:00Z',
  order: 0,
  snapshotCount: 2,
  ...overrides,
})

// 建立預設的 backup 物件
const createDefaultBackup = (overrides: Partial<BackupItem> = {}): BackupItem => ({
  name: 'test-backup',
  backupTime: '2024-01-01T00:00:00Z',
  hasToken: true,
  hasMachineId: true,
  machineId: 'abc123def456',
  provider: 'Github',
  isCurrent: false,
  isOriginalMachine: false,
  isTokenExpired: false,
  subscriptionTitle: 'KIRO PRO',
  usageLimit: 1000,
  currentUsage: 500,
  balance: 500,
  isLowBalance: false,
  cachedAt: '2024-01-01T00:00:00Z',
  folderId: '',
  ...overrides,
})

// 建立預設的 props
const createDefaultProps = (overrides = {}) => ({
  folders: [] as FolderItem[],
  backups: [] as BackupItem[],
  expandedFolders: new Set<string>(),
  uncategorizedExpanded: true,
  dragOverFolderId: null as string | null,
  dragOverUncategorized: false,
  renamingFolder: null as string | null,
  renameFolderName: '',
  deletingFolder: null as string | null,
  selectedBackups: new Set<string>(),
  switchingBackup: null as string | null,
  deletingBackup: null as string | null,
  refreshingBackup: null as string | null,
  regeneratingId: null as string | null,
  countdownTimers: {} as Record<string, number>,
  copiedMachineId: null as string | null,
  ...overrides,
})

// 建立 wrapper 的輔助函數
const createWrapper = (props = {}) => {
  return mount(FolderTree, {
    props: createDefaultProps(props),
    global: {
      plugins: [i18n],
      stubs: { Icon: IconStub, BackupCard: BackupCardStub },
    },
  })
}

describe('FolderTree', () => {
  // ============================================================================
  // Task 6.1: Props 渲染測試
  // ============================================================================
  describe('Task 6.1: Props 渲染', () => {
    it('should render empty state when no folders and backups', () => {
      const wrapper = createWrapper()
      
      // 應該渲染未分類區域
      expect(wrapper.find('[data-testid="uncategorized-section"]').exists()).toBe(true)
    })

    it('should render folders list', () => {
      const folders = [
        createDefaultFolder({ id: 'folder-1', name: 'Folder 1' }),
        createDefaultFolder({ id: 'folder-2', name: 'Folder 2' }),
      ]
      
      const wrapper = createWrapper({ folders })
      
      expect(wrapper.text()).toContain('Folder 1')
      expect(wrapper.text()).toContain('Folder 2')
    })

    it('should show expanded folder content', () => {
      const folders = [createDefaultFolder({ id: 'folder-1', name: 'Test Folder' })]
      const backups = [createDefaultBackup({ name: 'backup-1', folderId: 'folder-1' })]
      const expandedFolders = new Set(['folder-1'])
      
      const wrapper = createWrapper({ folders, backups, expandedFolders })
      
      // 展開的文件夾應該顯示內容
      expect(wrapper.find('[data-testid="folder-content-folder-1"]').exists()).toBe(true)
    })

    it('should show rename input when renamingFolder is set', () => {
      const folders = [createDefaultFolder({ id: 'folder-1', name: 'Test Folder' })]
      
      const wrapper = createWrapper({ 
        folders, 
        renamingFolder: 'folder-1',
        renameFolderName: 'New Name',
      })
      
      const input = wrapper.find('[data-testid="rename-input-folder-1"]')
      expect(input.exists()).toBe(true)
    })

    it('should highlight folder when dragOverFolderId matches', () => {
      const folders = [createDefaultFolder({ id: 'folder-1' })]
      
      const wrapper = createWrapper({ folders, dragOverFolderId: 'folder-1' })
      
      const folderEl = wrapper.find('[data-testid="folder-folder-1"]')
      expect(folderEl.classes()).toContain('ring-2')
    })
  })

  // ============================================================================
  // Task 6.2: 計算屬性測試
  // ============================================================================
  describe('Task 6.2: 計算屬性', () => {
    it('should compute uncategorizedBackups correctly', () => {
      const backups = [
        createDefaultBackup({ name: 'backup-1', folderId: 'folder-1' }),
        createDefaultBackup({ name: 'backup-2', folderId: '' }),
        createDefaultBackup({ name: 'backup-3', folderId: '' }),
      ]
      
      const wrapper = createWrapper({ backups, uncategorizedExpanded: true })
      
      // 應該顯示 2 個未分類的備份
      const uncategorizedCards = wrapper.findAll('[data-testid="uncategorized-section"] .backup-card-stub')
      expect(uncategorizedCards.length).toBe(2)
    })

    it('should compute isAllSelected correctly when all uncategorized are selected', () => {
      const backups = [
        createDefaultBackup({ name: 'backup-1', folderId: '' }),
        createDefaultBackup({ name: 'backup-2', folderId: '' }),
      ]
      const selectedBackups = new Set(['backup-1', 'backup-2'])
      
      const wrapper = createWrapper({ backups, selectedBackups })
      
      const selectAllCheckbox = wrapper.find('[data-testid="select-all-checkbox"]')
      expect((selectAllCheckbox.element as HTMLInputElement).checked).toBe(true)
    })

    it('should compute isAllSelected as false when not all selected', () => {
      const backups = [
        createDefaultBackup({ name: 'backup-1', folderId: '' }),
        createDefaultBackup({ name: 'backup-2', folderId: '' }),
      ]
      const selectedBackups = new Set(['backup-1'])
      
      const wrapper = createWrapper({ backups, selectedBackups })
      
      const selectAllCheckbox = wrapper.find('[data-testid="select-all-checkbox"]')
      expect((selectAllCheckbox.element as HTMLInputElement).checked).toBe(false)
    })
  })

  // ============================================================================
  // Task 6.3-6.7: 事件觸發測試
  // ============================================================================
  describe('Task 6.3-6.7: 事件觸發', () => {
    it('should emit toggle-folder when folder header is clicked', async () => {
      const folders = [createDefaultFolder({ id: 'folder-1' })]
      
      const wrapper = createWrapper({ folders })
      
      await wrapper.find('[data-testid="folder-header-folder-1"]').trigger('click')
      
      expect(wrapper.emitted('toggle-folder')).toBeTruthy()
      expect(wrapper.emitted('toggle-folder')![0]).toEqual(['folder-1'])
    })

    it('should emit toggle-uncategorized when uncategorized header is clicked', async () => {
      const wrapper = createWrapper()
      
      await wrapper.find('[data-testid="uncategorized-header"]').trigger('click')
      
      expect(wrapper.emitted('toggle-uncategorized')).toBeTruthy()
    })

    it('should emit start-rename-folder when rename button is clicked', async () => {
      const folders = [createDefaultFolder({ id: 'folder-1', name: 'Test' })]
      
      const wrapper = createWrapper({ folders })
      
      await wrapper.find('[data-testid="rename-btn-folder-1"]').trigger('click')
      
      expect(wrapper.emitted('start-rename-folder')).toBeTruthy()
      expect(wrapper.emitted('start-rename-folder')![0]).toEqual([folders[0]])
    })

    it('should emit confirm-rename-folder when confirm button is clicked', async () => {
      const folders = [createDefaultFolder({ id: 'folder-1' })]
      
      const wrapper = createWrapper({ 
        folders, 
        renamingFolder: 'folder-1',
        renameFolderName: 'New Name',
      })
      
      await wrapper.find('[data-testid="confirm-rename-btn-folder-1"]').trigger('click')
      
      expect(wrapper.emitted('confirm-rename-folder')).toBeTruthy()
    })

    it('should emit cancel-rename-folder when cancel button is clicked', async () => {
      const folders = [createDefaultFolder({ id: 'folder-1' })]
      
      const wrapper = createWrapper({ 
        folders, 
        renamingFolder: 'folder-1',
        renameFolderName: 'New Name',
      })
      
      await wrapper.find('[data-testid="cancel-rename-btn-folder-1"]').trigger('click')
      
      expect(wrapper.emitted('cancel-rename-folder')).toBeTruthy()
    })

    it('should emit delete-folder when delete button is clicked', async () => {
      const folders = [createDefaultFolder({ id: 'folder-1' })]
      
      const wrapper = createWrapper({ folders })
      
      await wrapper.find('[data-testid="delete-btn-folder-1"]').trigger('click')
      
      expect(wrapper.emitted('delete-folder')).toBeTruthy()
      expect(wrapper.emitted('delete-folder')![0]).toEqual([folders[0]])
    })

    it('should emit toggle-select-all when select all checkbox is changed', async () => {
      const backups = [createDefaultBackup({ name: 'backup-1', folderId: '' })]
      
      const wrapper = createWrapper({ backups })
      
      await wrapper.find('[data-testid="select-all-checkbox"]').trigger('change')
      
      expect(wrapper.emitted('toggle-select-all')).toBeTruthy()
    })

    it('should emit drag-enter-folder on dragenter', async () => {
      const folders = [createDefaultFolder({ id: 'folder-1' })]
      
      const wrapper = createWrapper({ folders })
      
      await wrapper.find('[data-testid="folder-folder-1"]').trigger('dragenter')
      
      expect(wrapper.emitted('drag-enter-folder')).toBeTruthy()
      expect(wrapper.emitted('drag-enter-folder')![0]).toEqual(['folder-1'])
    })

    it('should emit drag-leave-folder on dragleave', async () => {
      const folders = [createDefaultFolder({ id: 'folder-1' })]
      
      const wrapper = createWrapper({ folders })
      
      await wrapper.find('[data-testid="folder-folder-1"]').trigger('dragleave')
      
      expect(wrapper.emitted('drag-leave-folder')).toBeTruthy()
    })

    it('should emit drop-to-folder on drop', async () => {
      const folders = [createDefaultFolder({ id: 'folder-1' })]
      
      const wrapper = createWrapper({ folders })
      
      const mockEvent = new Event('drop') as DragEvent
      await wrapper.find('[data-testid="folder-folder-1"]').trigger('drop', mockEvent)
      
      expect(wrapper.emitted('drop-to-folder')).toBeTruthy()
    })

    it('should emit drag-enter-uncategorized on dragenter', async () => {
      const wrapper = createWrapper()
      
      await wrapper.find('[data-testid="uncategorized-section"]').trigger('dragenter')
      
      expect(wrapper.emitted('drag-enter-uncategorized')).toBeTruthy()
    })

    it('should emit drop-to-uncategorized on drop', async () => {
      const wrapper = createWrapper()
      
      const mockEvent = new Event('drop') as DragEvent
      await wrapper.find('[data-testid="uncategorized-section"]').trigger('drop', mockEvent)
      
      expect(wrapper.emitted('drop-to-uncategorized')).toBeTruthy()
    })
  })
})
