<script setup lang="ts">
/**
 * FolderTree 組件
 * @description 顯示文件夾樹狀結構，包含文件夾列表和未分類區域
 * @requirements Task 6.1-6.7
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import BackupCard from './BackupCard.vue'
import type { FolderItem, BackupItem } from '@/types/backup'

// ============================================================================
// Props 定義 (Task 6.1)
// ============================================================================

interface Props {
  folders: FolderItem[]
  backups: BackupItem[]
  expandedFolders: Set<string>
  uncategorizedExpanded: boolean
  dragOverFolderId: string | null
  dragOverUncategorized: boolean
  renamingFolder: string | null
  renameFolderName: string
  deletingFolder: string | null
  selectedBackups: Set<string>
  switchingBackup: string | null
  deletingBackup: string | null
  refreshingBackup: string | null
  regeneratingId: string | null
  countdownTimers: Record<string, number>
  copiedMachineId: string | null
}

const props = defineProps<Props>()

// ============================================================================
// Events 定義 (Task 6.1)
// ============================================================================

const emit = defineEmits<{
  // 文件夾事件
  (e: 'toggle-folder', folderId: string): void
  (e: 'toggle-uncategorized'): void
  (e: 'start-rename-folder', folder: FolderItem): void
  (e: 'confirm-rename-folder'): void
  (e: 'cancel-rename-folder'): void
  (e: 'delete-folder', folder: FolderItem): void
  (e: 'create-folder'): void
  // 拖放事件
  (e: 'drag-enter-folder', folderId: string): void
  (e: 'drag-leave-folder'): void
  (e: 'drag-enter-uncategorized'): void
  (e: 'drag-leave-uncategorized'): void
  (e: 'drop-to-folder', event: DragEvent, folderId: string): void
  (e: 'drop-to-uncategorized', event: DragEvent): void
  // 備份操作事件
  (e: 'toggle-select', name: string): void
  (e: 'toggle-select-all'): void
  (e: 'switch-backup', name: string): void
  (e: 'delete-backup', name: string): void
  (e: 'refresh-backup', name: string): void
  (e: 'regenerate-id', name: string): void
  (e: 'copy-machine-id', machineId: string): void
  (e: 'drag-start', event: DragEvent, name: string): void
  (e: 'drag-end', event: DragEvent): void
}>()

// ============================================================================
// i18n
// ============================================================================

const { t } = useI18n()

// ============================================================================
// 計算屬性 (Task 6.2)
// ============================================================================

/**
 * 取得指定文件夾內的備份列表
 */
const getBackupsInFolder = (folderId: string): BackupItem[] => {
  return props.backups.filter(backup => backup.folderId === folderId)
}

/**
 * 未分類的備份列表
 */
const uncategorizedBackups = computed(() => {
  return props.backups.filter(backup => !backup.folderId)
})

/**
 * 是否全選未分類備份
 */
const isAllSelected = computed(() => {
  const uncategorized = uncategorizedBackups.value
  if (uncategorized.length === 0) return false
  return uncategorized.every(backup => props.selectedBackups.has(backup.name))
})

// ============================================================================
// 事件處理
// ============================================================================

const onDragOver = (event: DragEvent) => {
  event.preventDefault()
}

const handleToggleFolder = (folderId: string) => {
  emit('toggle-folder', folderId)
}

const handleToggleUncategorized = () => {
  emit('toggle-uncategorized')
}

const handleStartRename = (folder: FolderItem) => {
  emit('start-rename-folder', folder)
}

const handleConfirmRename = () => {
  emit('confirm-rename-folder')
}

const handleCancelRename = () => {
  emit('cancel-rename-folder')
}

const handleDeleteFolder = (folder: FolderItem) => {
  emit('delete-folder', folder)
}

const handleDragEnterFolder = (folderId: string) => {
  emit('drag-enter-folder', folderId)
}

const handleDragLeaveFolder = () => {
  emit('drag-leave-folder')
}

const handleDropToFolder = (event: DragEvent, folderId: string) => {
  emit('drop-to-folder', event, folderId)
}

const handleDragEnterUncategorized = () => {
  emit('drag-enter-uncategorized')
}

const handleDragLeaveUncategorized = () => {
  emit('drag-leave-uncategorized')
}

const handleDropToUncategorized = (event: DragEvent) => {
  emit('drop-to-uncategorized', event)
}

const handleToggleSelectAll = () => {
  emit('toggle-select-all')
}

// BackupCard 事件轉發
const handleSelect = (name: string) => {
  emit('toggle-select', name)
}

const handleSwitch = (name: string) => {
  emit('switch-backup', name)
}

const handleDelete = (name: string) => {
  emit('delete-backup', name)
}

const handleRefresh = (name: string) => {
  emit('refresh-backup', name)
}

const handleRegenerateId = (name: string) => {
  emit('regenerate-id', name)
}

const handleCopyMachineId = (machineId: string) => {
  emit('copy-machine-id', machineId)
}

const handleDragStart = (event: DragEvent, name: string) => {
  emit('drag-start', event, name)
}

const handleDragEnd = (event: DragEvent) => {
  emit('drag-end', event)
}

/**
 * 檢查備份是否在冷卻中
 */
const isInCooldown = (name: string): boolean => {
  return (props.countdownTimers[name] || 0) > 0
}
</script>

<template>
  <div class="space-y-4">
    <!-- 文件夾列表 (Task 6.3-6.5) -->
    <div class="space-y-3">
      <div 
        v-for="folder in folders" 
        :key="folder.id"
        :data-testid="`folder-${folder.id}`"
        class="bg-app-surface border border-app-border rounded-xl overflow-hidden"
        :class="{ 'ring-2 ring-app-accent': dragOverFolderId === folder.id }"
        @dragover="onDragOver"
        @dragenter="handleDragEnterFolder(folder.id)"
        @dragleave="handleDragLeaveFolder"
        @drop="handleDropToFolder($event, folder.id)"
      >
        <!-- 文件夾標題 -->
        <div 
          :data-testid="`folder-header-${folder.id}`"
          class="flex items-center justify-between px-4 py-3 bg-zinc-900/50 cursor-pointer hover:bg-zinc-800/50 transition-colors"
          @click="handleToggleFolder(folder.id)"
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
                v-model="props.renameFolderName"
                :data-testid="`rename-input-${folder.id}`"
                @click.stop
                @keyup.enter="handleConfirmRename"
                @keyup.escape="handleCancelRename"
                class="px-2 py-1 bg-zinc-800 border border-zinc-600 rounded text-sm text-zinc-200 focus:outline-none focus:border-app-accent"
                autofocus
              />
              <button 
                :data-testid="`confirm-rename-btn-${folder.id}`"
                @click.stop="handleConfirmRename" 
                class="p-1 text-app-success hover:bg-zinc-700 rounded"
              >
                <Icon name="Check" class="w-4 h-4" />
              </button>
              <button 
                :data-testid="`cancel-rename-btn-${folder.id}`"
                @click.stop="handleCancelRename" 
                class="p-1 text-zinc-400 hover:bg-zinc-700 rounded"
              >
                <Icon name="X" class="w-4 h-4" />
              </button>
            </template>
            <template v-else>
              <span class="text-zinc-200 font-medium">{{ folder.name }}</span>
              <span class="text-zinc-500 text-xs">({{ folder.snapshotCount }})</span>
            </template>
          </div>
          
          <!-- 文件夾操作按鈕 -->
          <div class="flex items-center gap-1" @click.stop>
            <button 
              :data-testid="`rename-btn-${folder.id}`"
              @click="handleStartRename(folder)"
              class="p-1.5 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50 rounded transition-colors"
              :title="t('folder.rename')"
            >
              <Icon name="Edit" class="w-3.5 h-3.5" />
            </button>
            <button 
              :data-testid="`delete-btn-${folder.id}`"
              @click="handleDeleteFolder(folder)"
              :disabled="deletingFolder === folder.id"
              class="p-1.5 text-zinc-500 hover:text-red-400 hover:bg-zinc-700/50 rounded transition-colors"
              :title="t('folder.delete')"
            >
              <Icon name="Trash" class="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
        
        <!-- 文件夾內容（展開時顯示）(Task 6.4) -->
        <div 
          v-if="expandedFolders.has(folder.id)" 
          :data-testid="`folder-content-${folder.id}`"
          class="border-t border-zinc-800"
        >
          <div v-if="getBackupsInFolder(folder.id).length === 0" class="px-4 py-6 text-center text-zinc-500 text-sm">
            {{ t('folder.emptyFolder') }}
          </div>
          <div v-else class="divide-y divide-zinc-800/50">
            <BackupCard
              v-for="backup in getBackupsInFolder(folder.id)"
              :key="backup.name"
              :backup="backup"
              :is-selected="selectedBackups.has(backup.name)"
              :is-switching="switchingBackup === backup.name"
              :is-deleting="deletingBackup === backup.name"
              :is-refreshing="refreshingBackup === backup.name"
              :is-regenerating="regeneratingId === backup.name"
              :cooldown-seconds="countdownTimers[backup.name] || 0"
              :copied-machine-id="copiedMachineId"
              @select="handleSelect"
              @switch="handleSwitch"
              @delete="handleDelete"
              @refresh="handleRefresh"
              @regenerate-id="handleRegenerateId"
              @copy-machine-id="handleCopyMachineId"
              @drag-start="handleDragStart"
              @drag-end="handleDragEnd"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 未分類區域 (Task 6.6-6.7) -->
    <div 
      data-testid="uncategorized-section"
      class="bg-app-surface border border-app-border rounded-xl overflow-hidden"
      :class="{ 'ring-2 ring-app-accent': dragOverUncategorized }"
      @dragover="onDragOver"
      @dragenter="handleDragEnterUncategorized"
      @dragleave="handleDragLeaveUncategorized"
      @drop="handleDropToUncategorized"
    >
      <!-- 未分類標題 -->
      <div 
        data-testid="uncategorized-header"
        class="flex items-center justify-between px-4 py-3 bg-zinc-900/50 cursor-pointer hover:bg-zinc-800/50 transition-colors"
        @click="handleToggleUncategorized"
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
            data-testid="select-all-checkbox"
            :checked="isAllSelected"
            @change="handleToggleSelectAll"
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
          <BackupCard
            v-for="backup in uncategorizedBackups"
            :key="backup.name"
            :backup="backup"
            :is-selected="selectedBackups.has(backup.name)"
            :is-switching="switchingBackup === backup.name"
            :is-deleting="deletingBackup === backup.name"
            :is-refreshing="refreshingBackup === backup.name"
            :is-regenerating="regeneratingId === backup.name"
            :cooldown-seconds="countdownTimers[backup.name] || 0"
            :copied-machine-id="copiedMachineId"
            @select="handleSelect"
            @switch="handleSwitch"
            @delete="handleDelete"
            @refresh="handleRefresh"
            @regenerate-id="handleRegenerateId"
            @copy-machine-id="handleCopyMachineId"
            @drag-start="handleDragStart"
            @drag-end="handleDragEnd"
          />
        </div>
      </div>
    </div>
  </div>
</template>
