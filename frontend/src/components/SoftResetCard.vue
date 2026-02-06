<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import type { SoftResetStatus } from '@/types/backup'

// ============================================================================
// Props & Emits
// ============================================================================

interface Props {
  softResetStatus: SoftResetStatus
  isResetting: boolean
  isPatching: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'reset'): void
  (e: 'patch'): void
  (e: 'restore'): void
  (e: 'open-extension-folder'): void
  (e: 'open-machine-id-folder'): void
}>()

// ============================================================================
// i18n
// ============================================================================

const { t } = useI18n()

// ============================================================================
// Computed
// ============================================================================

/** 是否完全啟用（已 Patch 且有自訂 ID） */
const isFullyActive = computed(() => 
  props.softResetStatus.isPatched && props.softResetStatus.hasCustomId
)
</script>

<template>
  <div class="lg:col-span-2 bg-zinc-900 border border-app-border rounded-xl p-4 flex flex-col">
    <!-- 上方：PATCH 狀態 -->
    <div class="flex items-center gap-2 mb-3">
      <Icon name="Cpu" class="w-4 h-4 text-zinc-400" />
      <span class="text-zinc-400 text-xs font-semibold uppercase tracking-wider">{{ t('status.patchStatus') }}</span>
    </div>
    
    <div class="space-y-2 mb-4">
      <!-- Extension Patch 狀態 -->
      <div class="flex items-center justify-between">
        <span class="text-zinc-500 text-sm">Extension Patch</span>
        <div class="flex items-center gap-2">
          <!-- 已 Patch：顯示靜態標籤 -->
          <span 
            v-if="softResetStatus.isPatched"
            data-testid="patched-label"
            class="px-2 py-0.5 rounded text-xs font-medium bg-app-success/20 text-app-success border border-app-success/30"
          >
            {{ t('status.patched') }}
          </span>
          <!-- 未 Patch：顯示可點擊按鍵 -->
          <button
            v-else
            data-testid="patch-btn"
            @click="emit('patch')"
            :disabled="isPatching"
            :class="[
              'px-2 py-0.5 rounded text-xs font-medium transition-all',
              isPatching
                ? 'bg-zinc-700/50 text-zinc-500 border border-zinc-600/30 cursor-wait'
                : 'bg-app-warning/20 text-app-warning border border-app-warning/30 hover:bg-app-warning/30 cursor-pointer'
            ]"
            :title="t('status.clickToPatch')"
          >
            <span v-if="isPatching" class="flex items-center gap-1">
              <Icon name="Loader" class="w-3 h-3 animate-spin" />
              {{ t('status.patching') }}
            </span>
            <span v-else>{{ t('status.notPatched') }}</span>
          </button>
          <button
            v-if="softResetStatus.extensionPath"
            data-testid="open-extension-folder-btn"
            @click="emit('open-extension-folder')"
            class="p-1 rounded text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50 transition-colors"
            :title="t('status.openFolder')"
          >
            <Icon name="FolderOpen" class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
      
      <!-- Machine ID 狀態 -->
      <div class="flex items-center justify-between">
        <span class="text-zinc-500 text-sm">Machine ID</span>
        <div class="flex items-center gap-2">
          <span 
            data-testid="machine-id-status"
            :class="[
              'px-2 py-0.5 rounded text-xs font-medium',
              softResetStatus.hasCustomId 
                ? 'bg-app-accent/20 text-app-accent border border-app-accent/30' 
                : 'bg-zinc-700/50 text-zinc-400 border border-zinc-600/30'
            ]"
          >
            {{ softResetStatus.hasCustomId ? t('status.hasCustomId') : t('status.noCustomId') }}
          </span>
          <button
            data-testid="open-machine-id-folder-btn"
            @click="emit('open-machine-id-folder')"
            class="p-1 rounded text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700/50 transition-colors"
            :title="t('status.openFolder')"
          >
            <Icon name="FolderOpen" class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
      
      <!-- 總體狀態指示 -->
      <div class="flex items-center gap-2 pt-1">
        <div 
          data-testid="status-indicator"
          :class="[
            'w-2 h-2 rounded-full',
            isFullyActive 
              ? 'bg-app-success shadow-[0_0_6px_rgba(34,197,94,0.6)]' 
              : 'bg-zinc-500'
          ]"
        ></div>
        <span :class="[
          'text-xs font-medium',
          isFullyActive 
            ? 'text-app-success' 
            : 'text-zinc-500'
        ]">
          {{ isFullyActive ? t('status.softResetActive') : t('status.softResetInactive') }}
        </span>
      </div>
    </div>
    
    <!-- 下方：按鈕區域 -->
    <div class="mt-auto space-y-2">
      <!-- 一鍵新機按鈕 -->
      <button 
        data-testid="reset-btn"
        @click="emit('reset')"
        :disabled="isResetting"
        :class="[
          'w-full relative group flex items-center justify-center gap-3 px-4 py-3 border rounded-lg transition-all',
          isResetting 
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
          <g v-if="isResetting">
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
          <span :class="['text-sm font-bold block', isResetting ? 'text-white' : 'text-zinc-200 group-hover:text-white']">
            {{ isResetting ? t('app.processing') : t('restore.reset') }}
          </span>
          <span :class="['text-[10px]', isResetting ? 'text-zinc-200' : 'text-zinc-500 group-hover:text-zinc-300']">
            {{ isResetting ? t('message.successChange') : t('restore.resetDesc') }}
          </span>
        </div>
      </button>
      
      <!-- 還原原始機器按鈕 -->
      <button
        data-testid="restore-btn"
        @click="emit('restore')"
        :disabled="!softResetStatus.hasCustomId"
        :class="[
          'w-full px-3 py-2 rounded-lg text-sm font-medium transition-all',
          softResetStatus.hasCustomId
            ? 'bg-zinc-800 hover:bg-zinc-700 text-zinc-300 hover:text-white border border-zinc-700'
            : 'bg-zinc-800/50 text-zinc-600 border border-zinc-700/50 cursor-not-allowed'
        ]"
      >
        {{ t('restore.original') }}
      </button>
    </div>
  </div>
</template>
