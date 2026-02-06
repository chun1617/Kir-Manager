<script setup lang="ts">
/**
 * CurrentStatusCard 組件
 * @description 顯示當前環境狀態、訂閱資訊和操作按鈕
 * @requirements 8.1-8.4
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import NumberFlow from './NumberFlow.vue'
import { getSubscriptionColorClass, getSubscriptionShortName } from '../utils/subscription'
import type { CurrentUsageInfo, BackupItem } from '../types/backup'

// ============================================================================
// Props 定義
// ============================================================================

interface Props {
  /** 當前環境名稱 */
  currentEnvironmentName: string
  /** 當前機器碼 */
  currentMachineId: string
  /** 當前 Provider */
  currentProvider: string
  /** 用量資訊 */
  usageInfo: CurrentUsageInfo | null
  /** 當前活躍的備份 */
  activeBackup: BackupItem | null
  /** 是否正在刷新 */
  isRefreshing: boolean
  /** 是否正在還原 */
  isRestoring: boolean
  /** 冷卻倒計時秒數 */
  cooldownSeconds: number
}

const props = defineProps<Props>()

// ============================================================================
// Events 定義
// ============================================================================

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'create-backup'): void
  (e: 'restore-original'): void
  (e: 'open-sso-cache'): void
}>()

// ============================================================================
// i18n
// ============================================================================

const { t } = useI18n()

// ============================================================================
// 計算屬性
// ============================================================================

/** 是否在冷卻期 */
const isInCooldown = computed(() => props.cooldownSeconds > 0)

/** 顯示的環境名稱 */
const displayEnvironmentName = computed(() => 
  props.currentEnvironmentName || t('status.originalMachine')
)

/** 顯示的機器碼 */
const displayMachineId = computed(() => props.currentMachineId || '-')

/** 訂閱類型顏色類別 */
const subscriptionClass = computed(() => 
  props.usageInfo ? getSubscriptionColorClass(props.usageInfo.subscriptionTitle) : ''
)

/** 訂閱類型簡稱 */
const subscriptionShortName = computed(() => 
  props.usageInfo ? getSubscriptionShortName(props.usageInfo.subscriptionTitle) : ''
)

/** 當前 Provider（優先使用 activeBackup 的 provider） */
const currentDisplayProvider = computed(() => 
  props.activeBackup?.provider || props.currentProvider
)

// ============================================================================
// 事件處理
// ============================================================================

const handleRefresh = () => {
  if (!isInCooldown.value && !props.isRefreshing) {
    emit('refresh')
  }
}

const handleCreateBackup = () => {
  emit('create-backup')
}

const handleRestoreOriginal = () => {
  if (!props.isRestoring) {
    emit('restore-original')
  }
}

const handleOpenSSOCache = () => {
  emit('open-sso-cache')
}
</script>

<template>
  <div class="lg:col-span-3 bg-gradient-to-br from-zinc-900 to-zinc-900/50 border border-app-border rounded-xl p-6 relative overflow-hidden group">
    <!-- 背景圖標：根據 Provider 動態顯示，點擊打開 SSO Cache 文件夾 -->
    <div 
      data-testid="provider-icon"
      @click="handleOpenSSOCache"
      class="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 hover:!opacity-40 transition-opacity cursor-pointer z-20"
      :title="t('status.openSSOCache')"
    >
      <Icon v-if="currentDisplayProvider === 'Github'" name="Github" class="w-32 h-32 text-white pointer-events-none" />
      <Icon v-else-if="currentDisplayProvider === 'AWS' || currentDisplayProvider === 'BuilderId'" name="AWS" class="w-32 h-32 text-white pointer-events-none" />
      <Icon v-else-if="currentDisplayProvider === 'Enterprise'" name="AWS" class="w-32 h-32 text-white pointer-events-none" />
      <Icon v-else-if="currentDisplayProvider === 'Google'" name="Google" class="w-32 h-32 text-white pointer-events-none" />
      <Icon v-else name="Cpu" class="w-32 h-32 text-white pointer-events-none" />
    </div>
    
    <div class="relative z-10">
      <!-- 標籤列：當前狀態 + 訂閱類型 + 餘額 + 刷新按鈕 -->
      <div class="flex items-center gap-2 mb-4">
        <span class="px-2 py-0.5 rounded text-[10px] font-bold bg-app-warning text-black uppercase tracking-wider">
          {{ t('status.current') }}
        </span>
        
        <!-- 顯示當前帳號訂閱和餘額 -->
        <template v-if="usageInfo">
          <!-- 訂閱類型標籤 -->
          <span 
            data-testid="subscription-badge"
            :class="['px-2 py-0.5 rounded text-[10px] font-medium', subscriptionClass]"
          >
            {{ subscriptionShortName }}
          </span>
          
          <!-- 餘額資訊 -->
          <span 
            data-testid="balance-info"
            :class="[
              'text-xs font-mono',
              usageInfo.isLowBalance ? 'text-app-warning' : 'text-zinc-400'
            ]"
          >
            <span v-if="usageInfo.isLowBalance" class="inline-flex items-center gap-1">
              <Icon name="AlertTriangle" class="w-3 h-3" />
              <NumberFlow :value="Math.round(usageInfo.balance)" /> / <NumberFlow :value="Math.round(usageInfo.usageLimit)" />
            </span>
            <span v-else>
              <NumberFlow :value="Math.round(usageInfo.balance)" /> / <NumberFlow :value="Math.round(usageInfo.usageLimit)" />
            </span>
          </span>
          
          <!-- 刷新按鈕 / 倒計時 -->
          <button
            data-testid="refresh-btn"
            @click="handleRefresh"
            :disabled="isRefreshing || isInCooldown"
            :class="[
              'w-[22px] h-[22px] rounded transition-all inline-flex items-center justify-center',
              isRefreshing
                ? 'text-app-accent cursor-wait'
                : isInCooldown
                  ? 'text-zinc-500 cursor-not-allowed'
                  : 'text-zinc-500 hover:text-zinc-300'
            ]"
            :title="t('backup.refresh')"
          >
            <!-- 刷新中：旋轉圖標 -->
            <Icon 
              v-if="isRefreshing"
              name="RefreshCw" 
              class="w-3.5 h-3.5 animate-spin" 
            />
            <!-- 倒計時數字 -->
            <span 
              v-else-if="isInCooldown" 
              class="text-xs font-mono font-medium leading-none"
            >
              <NumberFlow :value="cooldownSeconds" />
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
      
      <!-- 環境名稱 -->
      <h3 class="text-3xl font-bold text-white mb-1 glow-text">
        {{ displayEnvironmentName }}
      </h3>
      
      <!-- 機器碼 -->
      <div class="flex items-center gap-2 text-app-accent font-mono text-sm mb-6">
        <Icon name="Check" class="w-4 h-4" />
        {{ displayMachineId }}
      </div>

      <!-- 操作按鈕 -->
      <div class="flex gap-3">
        <!-- 建立備份按鈕 -->
        <button 
          data-testid="create-backup-btn"
          @click="handleCreateBackup"
          class="flex items-center px-4 py-2 bg-zinc-800 hover:bg-zinc-700 border border-zinc-600 text-zinc-200 rounded-lg text-sm transition-all active:scale-95"
        >
          <Icon name="Save" class="w-4 h-4 mr-2" />
          {{ t('backup.create') }}
        </button>
        
        <!-- 還原原始機器按鈕 -->
        <button 
          data-testid="restore-original-btn"
          @click="handleRestoreOriginal"
          :disabled="isRestoring"
          :class="[
            'flex items-center px-4 py-2 border rounded-lg text-sm transition-all',
            isRestoring
              ? 'bg-zinc-800/50 border-zinc-700/50 text-zinc-500 cursor-wait'
              : 'bg-zinc-800/50 hover:bg-red-900/30 border-zinc-700/50 hover:border-red-800/50 text-zinc-400 hover:text-red-400'
          ]"
        >
          <Icon 
            name="Rotate" 
            :class="['w-4 h-4 mr-2', isRestoring ? 'animate-spin' : '']" 
          />
          {{ isRestoring ? t('app.processing') : t('restore.original') }}
        </button>
      </div>
    </div>
  </div>
</template>
