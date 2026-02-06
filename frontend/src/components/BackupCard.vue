<script setup lang="ts">
/**
 * BackupCard 組件
 * @description 顯示單一備份的資訊和操作按鈕
 * @requirements 6.1-12.3
 */
import { computed } from 'vue'
import Icon from './Icon.vue'
import NumberFlow from './NumberFlow.vue'
import { getSubscriptionColorClass, getSubscriptionShortName } from '../utils/subscription'
import { truncateMachineId } from '../utils/machineId'

// ============================================================================
// Props 定義
// ============================================================================

interface BackupData {
  name: string
  provider: string
  subscriptionTitle: string
  usageLimit: number
  currentUsage: number
  balance: number
  isLowBalance: boolean
  isCurrent: boolean
  isOriginalMachine: boolean
  machineId: string
  isTokenExpired?: boolean
}

interface Props {
  backup: BackupData
  isSelected: boolean
  isSwitching: boolean
  isDeleting: boolean
  isRefreshing: boolean
  isRegenerating: boolean
  cooldownSeconds: number
  copiedMachineId: string | null
}

const props = defineProps<Props>()

// ============================================================================
// Events 定義
// ============================================================================

const emit = defineEmits<{
  (e: 'select', name: string): void
  (e: 'switch', name: string): void
  (e: 'delete', name: string): void
  (e: 'refresh', name: string): void
  (e: 'regenerate-id', name: string): void
  (e: 'copy-machine-id', machineId: string): void
  (e: 'drag-start', event: DragEvent, name: string): void
  (e: 'drag-end', event: DragEvent): void
}>()

// ============================================================================
// 計算屬性
// ============================================================================

const isInCooldown = computed(() => props.cooldownSeconds > 0)

const isCopied = computed(() => props.copiedMachineId === props.backup.machineId)

const displayMachineId = computed(() => truncateMachineId(props.backup.machineId))

const subscriptionClass = computed(() => getSubscriptionColorClass(props.backup.subscriptionTitle))

const subscriptionShortName = computed(() => getSubscriptionShortName(props.backup.subscriptionTitle))

// ============================================================================
// 事件處理
// ============================================================================

const handleSelect = () => {
  emit('select', props.backup.name)
}

const handleSwitch = () => {
  emit('switch', props.backup.name)
}

const handleDelete = () => {
  emit('delete', props.backup.name)
}

const handleRefresh = () => {
  if (!isInCooldown.value) {
    emit('refresh', props.backup.name)
  }
}

const handleRegenerate = () => {
  emit('regenerate-id', props.backup.name)
}

const handleCopyMachineId = () => {
  if (props.backup.machineId) {
    emit('copy-machine-id', props.backup.machineId)
  }
}

const handleDragStart = (event: DragEvent) => {
  emit('drag-start', event, props.backup.name)
}

const handleDragEnd = (event: DragEvent) => {
  emit('drag-end', event)
}
</script>

<template>
  <div
    draggable="true"
    @dragstart="handleDragStart"
    @dragend="handleDragEnd"
    :class="[
      'flex items-center px-4 py-3 group transition-colors cursor-move',
      backup.isCurrent ? 'bg-app-accent/5' : 'hover:bg-zinc-800/30'
    ]"
  >
    <!-- Checkbox -->
    <div class="w-8 flex-shrink-0">
      <input
        type="checkbox"
        :checked="isSelected"
        @change="handleSelect"
        class="custom-checkbox"
      />
    </div>

    <!-- 快照名稱 -->
    <div class="flex-1 min-w-0">
      <div class="flex items-center">
        <Icon name="GripVertical" class="w-4 h-4 text-zinc-600 mr-2 opacity-0 group-hover:opacity-100 transition-opacity" />
        
        <!-- 活躍狀態指示器 -->
        <div
          v-if="backup.isCurrent"
          data-testid="active-indicator"
          class="w-1.5 h-1.5 rounded-full bg-app-warning mr-2 shadow-[0_0_8px_rgba(245,158,11,0.8)]"
        ></div>
        
        <span :class="['font-medium truncate', backup.isCurrent ? 'text-white' : 'text-zinc-400']">
          {{ backup.name }}
        </span>
        
        <!-- 原始機器標籤 -->
        <span
          v-if="backup.isOriginalMachine"
          data-testid="original-machine-label"
          class="ml-2 px-1.5 py-0.5 rounded text-[10px] bg-app-accent/20 text-app-accent border border-app-accent/30"
        >
          Original
        </span>
      </div>
    </div>

    <!-- Provider -->
    <div class="w-24 flex-shrink-0 px-2 flex justify-center">
      <span class="px-2 py-1 rounded text-[10px] bg-zinc-800 text-zinc-400 border border-zinc-700 inline-flex items-center gap-1">
        <span data-testid="provider-icon">
          <Icon v-if="backup.provider === 'Github'" name="Github" class="w-3 h-3" />
          <Icon v-else-if="backup.provider === 'AWS' || backup.provider === 'BuilderId'" name="AWS" class="w-3 h-3" />
          <Icon v-else-if="backup.provider === 'Enterprise'" name="AWS" class="w-3 h-3" />
          <Icon v-else-if="backup.provider === 'Google'" name="Google" class="w-3 h-3" />
        </span>
        {{ backup.provider }}
      </span>
    </div>

    <!-- 訂閱類型 -->
    <div class="w-16 flex-shrink-0 px-2">
      <span
        v-if="backup.subscriptionTitle"
        :class="['px-2 py-0.5 rounded text-[10px] font-medium', subscriptionClass]"
      >
        {{ subscriptionShortName }}
      </span>
      <span v-else class="text-zinc-500 text-xs">-</span>
    </div>

    <!-- 餘額 -->
    <div class="w-36 flex-shrink-0 px-2 flex items-center justify-end gap-2">
      <span
        v-if="backup.usageLimit > 0"
        data-testid="balance"
        :class="['font-mono text-xs whitespace-nowrap', backup.isLowBalance ? 'text-app-warning' : 'text-zinc-400']"
      >
        <NumberFlow :value="Math.round(backup.balance)" />/<NumberFlow :value="Math.round(backup.usageLimit)" />
      </span>
      <span v-else class="text-zinc-500 text-xs">-</span>
      
      <!-- 刷新按鈕 -->
      <button
        data-testid="refresh-btn"
        @click.stop="handleRefresh"
        :disabled="isInCooldown"
        class="p-0.5 text-zinc-500 hover:text-zinc-300 transition-colors disabled:cursor-not-allowed"
        :title="backup.isTokenExpired ? 'Token expired' : 'Refresh usage'"
      >
        <!-- 倒計時數字 -->
        <span v-if="isInCooldown" class="text-xs font-mono text-zinc-500 w-4 inline-block text-center">
          <NumberFlow :value="cooldownSeconds" />
        </span>
        <!-- 刷新圖標 -->
        <Icon
          v-else
          name="RefreshCw"
          :class="['w-3 h-3', isRefreshing ? 'animate-spin' : '']"
        />
      </button>
    </div>

    <!-- Machine ID -->
    <div class="w-32 flex-shrink-0 px-2">
      <button
        v-if="backup.machineId"
        data-testid="machine-id-btn"
        @click="handleCopyMachineId"
        class="font-mono text-[10px] text-zinc-500 hover:text-zinc-300 cursor-pointer transition-colors inline-flex items-center gap-1 group"
        :title="backup.machineId"
      >
        <span>{{ displayMachineId }}</span>
        <Icon
          :name="isCopied ? 'Check' : 'Copy'"
          :class="[
            'w-3 h-3 transition-all',
            isCopied
              ? 'text-app-success'
              : 'opacity-0 group-hover:opacity-100 text-zinc-400'
          ]"
        />
      </button>
      <span v-else class="font-mono text-[10px] text-zinc-500">-</span>
    </div>

    <!-- 操作按鈕 -->
    <div class="w-28 flex-shrink-0 flex justify-end gap-1">
      <!-- 切換按鈕 (非當前備份時顯示) -->
      <button
        v-if="!backup.isCurrent"
        data-testid="switch-btn"
        @click="handleSwitch"
        :disabled="isSwitching"
        :class="[
          'p-1.5 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50 rounded transition-colors',
          isSwitching ? 'animate-bounce' : ''
        ]"
        title="Switch to this backup"
      >
        <Icon name="Download" class="w-3.5 h-3.5" />
      </button>
      
      <!-- 重新生成機器碼按鈕 (非當前備份時顯示) -->
      <button
        v-if="!backup.isCurrent"
        data-testid="regenerate-btn"
        @click="handleRegenerate"
        :disabled="isRegenerating"
        :class="[
          'p-1.5 text-zinc-500 hover:text-app-accent hover:bg-zinc-700/50 rounded transition-colors',
          isRegenerating ? 'animate-pulse-fast' : ''
        ]"
        title="Regenerate machine ID"
      >
        <Icon name="Key" class="w-3.5 h-3.5" />
      </button>
      
      <!-- 刪除按鈕 (非當前備份時顯示) -->
      <button
        v-if="!backup.isCurrent"
        data-testid="delete-btn"
        @click="handleDelete"
        :disabled="isDeleting"
        :class="[
          'p-1.5 text-zinc-500 hover:text-red-400 hover:bg-zinc-700/50 rounded transition-colors',
          isDeleting ? 'animate-pulse' : ''
        ]"
        title="Delete backup"
      >
        <Icon name="Trash" class="w-3.5 h-3.5" />
      </button>
      
      <!-- 活躍狀態標籤 (當前備份時顯示) -->
      <span
        v-if="backup.isCurrent"
        data-testid="active-status"
        class="text-app-warning text-xs font-bold px-2"
      >
        ACTIVE
      </span>
    </div>
  </div>
</template>
