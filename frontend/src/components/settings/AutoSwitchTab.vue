<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import SettingsCard from './SettingsCard.vue'
import RefreshIntervalCard from './RefreshIntervalCard.vue'
import NumberInput from '@/components/NumberInput.vue'
import { getVisibleCards, type AutoSwitchCardConfig } from '@/constants/autoSwitchCards'
import { useRefreshIntervals } from '@/composables/useRefreshIntervals'
import type { RefreshRule } from '@/types/refreshInterval'

/**
 * AutoSwitchTab 組件 Props
 * @requirements 3.1-3.7 - 自動切換分頁內容
 */
interface Props {
  /** 自動切換是否啟用 */
  autoSwitchEnabled: boolean
  /** 觸發閾值 */
  balanceThreshold: number
  /** 目標最低餘額 */
  minTargetBalance: number
  /** 監控狀態 */
  monitorStatus: 'stopped' | 'running' | 'cooldown'
  /** 文件夾列表 */
  folders?: Array<{ id: string; name: string }>
  /** 已選文件夾 ID */
  selectedFolderIds?: string[]
  /** 已選訂閱類型 */
  selectedSubscriptionTypes?: string[]
  /** 切換時通知 */
  notifyOnSwitch?: boolean
  /** 低餘額預警 */
  notifyOnLowBalance?: boolean
  /** 刷新頻率規則 */
  refreshRules?: RefreshRule[]
}

const props = withDefaults(defineProps<Props>(), {
  folders: () => [],
  selectedFolderIds: () => [],
  selectedSubscriptionTypes: () => [],
  notifyOnSwitch: true,
  notifyOnLowBalance: true,
  refreshRules: () => [],
})

const emit = defineEmits<{
  (e: 'toggle', enabled: boolean): void
  (e: 'update:balanceThreshold', value: number): void
  (e: 'update:minTargetBalance', value: number): void
  (e: 'update:selectedFolderIds', value: string[]): void
  (e: 'update:selectedSubscriptionTypes', value: string[]): void
  (e: 'update:notifyOnSwitch', value: boolean): void
  (e: 'update:notifyOnLowBalance', value: boolean): void
  (e: 'update:refreshRules', value: RefreshRule[]): void
}>()

const { t } = useI18n()

// 整合 useRefreshIntervals composable
const {
  rules,
  isAddingDisabled,
  addDisabledReason,
  addRule,
  updateRule,
  deleteRule,
} = useRefreshIntervals(props.refreshRules, (newRules) => {
  emit('update:refreshRules', newRules)
})

// 處理規則操作
function handleAddRule() {
  addRule()
}

function handleUpdateRule(id: string, field: string, value: number | boolean) {
  updateRule(id, field as keyof RefreshRule, value)
}

function handleDeleteRule(id: string) {
  deleteRule(id)
}

function handleSaveRules() {
  emit('update:refreshRules', [...rules.value])
}

const visibleCards = computed(() => getVisibleCards(props.autoSwitchEnabled))

const showThresholdCard = computed(() => 
  visibleCards.value.some((c: AutoSwitchCardConfig) => c.id === 'threshold')
)
const showFilterCard = computed(() => 
  visibleCards.value.some((c: AutoSwitchCardConfig) => c.id === 'filter')
)
const showRefreshCard = computed(() => 
  visibleCards.value.some((c: AutoSwitchCardConfig) => c.id === 'refreshRate')
)
const showNotificationCard = computed(() => 
  visibleCards.value.some((c: AutoSwitchCardConfig) => c.id === 'notification')
)

const statusText = computed(() => {
  switch (props.monitorStatus) {
    case 'running': return t('autoSwitch.status.running')
    case 'cooldown': return t('autoSwitch.status.cooldown')
    default: return t('autoSwitch.status.stopped')
  }
})

const statusColor = computed(() => {
  switch (props.monitorStatus) {
    case 'running': return 'text-green-400'
    case 'cooldown': return 'text-yellow-400'
    default: return 'text-zinc-400'
  }
})

const subscriptionTypes = ['Free', 'Pro', 'Pro+', 'Enterprise']
</script>

<template>
  <div class="space-y-4">
    <!-- 開關與狀態卡片 (始終顯示) -->
    <SettingsCard>
      <div class="flex items-center justify-between">
        <div>
          <h4 class="text-zinc-100 font-medium">{{ t('autoSwitch.enabled') }}</h4>
          <p class="text-sm text-zinc-400">{{ t('autoSwitch.enabledDesc') }}</p>
        </div>
        <label class="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            :checked="autoSwitchEnabled"
            class="sr-only peer"
            @change="emit('toggle', ($event.target as HTMLInputElement).checked)"
          />
          <div class="w-11 h-6 bg-zinc-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-violet-500"></div>
        </label>
      </div>
      <div class="mt-4 pt-4 border-t border-zinc-700">
        <div class="flex items-center gap-2">
          <span class="text-sm text-zinc-400">{{ t('autoSwitch.status.label') }}:</span>
          <span :class="['text-sm font-medium', statusColor]">{{ statusText }}</span>
        </div>
      </div>
    </SettingsCard>

    <!-- 閾值設定卡片 -->
    <SettingsCard v-if="showThresholdCard" :title="t('autoSwitch.balanceThreshold')">
      <div class="space-y-6">
        <div>
          <label class="text-sm text-zinc-400 block mb-3">{{ t('autoSwitch.balanceThresholdDesc') }}</label>
          <NumberInput
            :model-value="balanceThreshold"
            :min="0"
            :max="100"
            :step="5"
            min-width="min-w-[4rem]"
            @update:model-value="emit('update:balanceThreshold', $event)"
          />
        </div>
        <div>
          <label class="text-sm text-zinc-400 block mb-3">{{ t('autoSwitch.minTargetBalanceDesc') }}</label>
          <NumberInput
            :model-value="minTargetBalance"
            :min="0"
            :max="1000"
            :step="10"
            min-width="min-w-[4rem]"
            @update:model-value="emit('update:minTargetBalance', $event)"
          />
        </div>
      </div>
    </SettingsCard>

    <!-- 篩選條件卡片 -->
    <SettingsCard v-if="showFilterCard" :title="t('autoSwitch.folderFilter')">
      <p class="text-sm text-zinc-400 mb-3">{{ t('autoSwitch.folderFilterDesc') }}</p>
      <div class="space-y-2">
        <div v-for="folder in folders" :key="folder.id" class="flex items-center gap-2">
          <input
            type="checkbox"
            :checked="selectedFolderIds.includes(folder.id)"
            class="custom-checkbox"
            @change="emit('update:selectedFolderIds', 
              ($event.target as HTMLInputElement).checked 
                ? [...selectedFolderIds, folder.id]
                : selectedFolderIds.filter(id => id !== folder.id)
            )"
          />
          <span class="text-sm text-zinc-300">{{ folder.name }}</span>
        </div>
      </div>
      
      <div class="mt-4 pt-4 border-t border-zinc-700">
        <p class="text-sm text-zinc-400 mb-3">{{ t('autoSwitch.subscriptionFilterDesc') }}</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="subType in subscriptionTypes"
            :key="subType"
            :class="[
              'px-3 py-1 rounded-lg text-sm transition-colors',
              selectedSubscriptionTypes.includes(subType)
                ? 'bg-violet-500 text-white'
                : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'
            ]"
            @click="emit('update:selectedSubscriptionTypes',
              selectedSubscriptionTypes.includes(subType)
                ? selectedSubscriptionTypes.filter(s => s !== subType)
                : [...selectedSubscriptionTypes, subType]
            )"
          >
            {{ subType }}
          </button>
        </div>
      </div>
    </SettingsCard>

    <!-- 刷新頻率卡片 - 整合 RefreshIntervalCard -->
    <RefreshIntervalCard
      v-if="showRefreshCard"
      :rules="rules"
      :is-adding-disabled="isAddingDisabled"
      :add-disabled-reason="addDisabledReason"
      @add="handleAddRule"
      @update="handleUpdateRule"
      @delete="handleDeleteRule"
      @save="handleSaveRules"
    />

    <!-- 通知選項卡片 -->
    <SettingsCard v-if="showNotificationCard">
      <div class="space-y-4">
        <label class="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            :checked="notifyOnSwitch"
            class="custom-checkbox"
            @change="emit('update:notifyOnSwitch', ($event.target as HTMLInputElement).checked)"
          />
          <span class="text-sm text-zinc-300">{{ t('autoSwitch.notifyOnSwitch') }}</span>
        </label>
        <label class="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            :checked="notifyOnLowBalance"
            class="custom-checkbox"
            @change="emit('update:notifyOnLowBalance', ($event.target as HTMLInputElement).checked)"
          />
          <span class="text-sm text-zinc-300">{{ t('autoSwitch.notifyOnLowBalance') }}</span>
        </label>
      </div>
    </SettingsCard>
  </div>
</template>
