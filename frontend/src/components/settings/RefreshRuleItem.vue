<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RefreshRule } from '@/types/refreshInterval'
import { REFRESH_INTERVAL_STYLES } from '@/constants/refreshIntervalStyles'

const props = withDefaults(defineProps<{
  rule: RefreshRule
  canDelete?: boolean
  hasError?: boolean
}>(), {
  canDelete: true,
  hasError: false,
})

const emit = defineEmits<{
  'update:minBalance': [value: number]
  'update:maxBalance': [value: number | boolean]
  'update:interval': [value: number]
  'delete': []
}>()

const { t } = useI18n()

/**
 * 是否為無上限模式
 * @description maxBalance === -1 表示無上限
 */
const isUnlimited = computed(() => props.rule.maxBalance === -1)

/**
 * 刪除按鈕的 tooltip
 */
const deleteTooltip = computed(() => 
  props.canDelete ? '' : t('refreshInterval.cannotDeleteLastRule')
)

/**
 * 刪除按鈕樣式
 */
const deleteButtonClass = computed(() => 
  props.canDelete 
    ? REFRESH_INTERVAL_STYLES.deleteButton.enabled 
    : REFRESH_INTERVAL_STYLES.deleteButton.disabled
)

/**
 * 容器樣式（包含錯誤狀態）
 */
const containerClass = computed(() => {
  const baseClass = REFRESH_INTERVAL_STYLES.ruleRow
  return props.hasError 
    ? `${baseClass} ${REFRESH_INTERVAL_STYLES.input.error}` 
    : baseClass
})


/**
 * 處理 minBalance 輸入變更
 */
function handleMinBalanceChange(event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  emit('update:minBalance', value)
}

/**
 * 處理 maxBalance 輸入變更
 */
function handleMaxBalanceChange(event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  emit('update:maxBalance', value)
}

/**
 * 處理 interval 輸入變更
 */
function handleIntervalChange(event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  emit('update:interval', value)
}

/**
 * 處理無上限 checkbox 變更
 */
function handleUnlimitedChange(event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  emit('update:maxBalance', checked)
}

/**
 * 處理刪除按鈕點擊
 */
function handleDelete() {
  if (props.canDelete) {
    emit('delete')
  }
}
</script>

<template>
  <div 
    :class="containerClass"
    data-testid="rule-row"
  >
    <!-- 餘額下限輸入 -->
    <input
      type="number"
      :value="rule.minBalance"
      :class="REFRESH_INTERVAL_STYLES.input.balance"
      data-testid="min-balance-input"
      min="0"
      @input="handleMinBalanceChange"
    />
    
    <span :class="REFRESH_INTERVAL_STYLES.arrow">~</span>
    
    <!-- 餘額上限輸入 -->
    <input
      type="number"
      :value="rule.maxBalance"
      :class="REFRESH_INTERVAL_STYLES.input.balance"
      :disabled="isUnlimited"
      data-testid="max-balance-input"
      min="0"
      @input="handleMaxBalanceChange"
    />
    
    <!-- 無上限 checkbox -->
    <label class="flex items-center gap-1 text-sm text-zinc-400 cursor-pointer">
      <input
        type="checkbox"
        :checked="isUnlimited"
        data-testid="unlimited-checkbox"
        class="custom-checkbox"
        @change="handleUnlimitedChange"
      />
      {{ t('refreshInterval.unlimited') }}
    </label>
    
    <span :class="REFRESH_INTERVAL_STYLES.arrow">→</span>
    
    <!-- 間隔輸入 -->
    <input
      type="number"
      :value="rule.interval"
      :class="REFRESH_INTERVAL_STYLES.input.interval"
      data-testid="interval-input"
      min="1"
      @input="handleIntervalChange"
    />
    
    <span class="text-sm text-zinc-400">{{ t('refreshInterval.minutes') }}</span>
    
    <!-- 刪除按鈕 -->
    <button
      type="button"
      :class="deleteButtonClass"
      :disabled="!canDelete"
      :title="deleteTooltip"
      data-testid="delete-button"
      @click="handleDelete"
    >
      ✕
    </button>
  </div>
</template>
