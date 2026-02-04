<script setup lang="ts">
import { watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RefreshRule } from '@/types/refreshInterval'
import SettingsCard from './SettingsCard.vue'
import RefreshRuleItem from './RefreshRuleItem.vue'
import AddRuleButton from './AddRuleButton.vue'

const props = withDefaults(defineProps<{
  rules: RefreshRule[]
  isAddingDisabled?: boolean
  addDisabledReason?: string | null
}>(), {
  isAddingDisabled: false,
  addDisabledReason: null,
})

const emit = defineEmits<{
  'add': []
  'update': [id: string, field: string, value: number | boolean]
  'delete': [id: string]
  'save': []
}>()

const { t } = useI18n()

/** 追蹤最後新增的規則 ID，用於自動聚焦 */
let lastAddedRuleId: string | null = null

/**
 * 監聽規則列表變化，當新增規則時自動聚焦
 */
watch(() => props.rules, (newRules, oldRules) => {
  // 檢查是否有新增規則（新列表比舊列表多一個）
  if (oldRules && newRules.length > oldRules.length) {
    // 找出新增的規則（在新列表中但不在舊列表中的）
    const oldIds = new Set(oldRules.map(r => r.id))
    const newRule = newRules.find(r => !oldIds.has(r.id))
    if (newRule) {
      lastAddedRuleId = newRule.id
      // 使用 nextTick 確保 DOM 已更新
      nextTick(() => {
        focusNewRuleInput(newRule.id)
      })
    }
  }
}, { deep: true })

/**
 * 聚焦到指定規則的餘額下限輸入框
 * @param ruleId 規則 ID
 */
function focusNewRuleInput(ruleId: string) {
  const ruleElement = document.querySelector(`[data-rule-id="${ruleId}"]`)
  if (ruleElement) {
    const input = ruleElement.querySelector('[data-testid="min-balance-input"]') as HTMLInputElement
    if (input) {
      input.focus()
    }
  }
}

/**
 * 處理新增規則
 */
function handleAdd() {
  emit('add')
}

/**
 * 處理規則更新
 * @param id 規則 ID
 * @param field 欄位名稱
 * @param value 新值
 */
function handleUpdate(id: string, field: string, value: number | boolean) {
  emit('update', id, field, value)
}

/**
 * 處理規則刪除
 * @param id 規則 ID
 */
function handleDelete(id: string) {
  emit('delete', id)
}

/**
 * 處理輸入框失焦，觸發儲存
 */
function handleBlur() {
  emit('save')
}
</script>

<template>
  <SettingsCard :title="t('autoSwitch.refreshIntervals.title')">
    <p class="text-sm text-zinc-400 mb-4">{{ t('autoSwitch.refreshIntervals.desc') }}</p>
    
    <div class="space-y-2" data-testid="rules-list">
      <RefreshRuleItem
        v-for="rule in rules"
        :key="rule.id"
        :rule="rule"
        :can-delete="rules.length > 1"
        :data-rule-id="rule.id"
        @update:min-balance="(v) => handleUpdate(rule.id, 'minBalance', v)"
        @update:max-balance="(v) => handleUpdate(rule.id, 'maxBalance', v)"
        @update:interval="(v) => handleUpdate(rule.id, 'interval', v)"
        @delete="handleDelete(rule.id)"
        @blur="handleBlur"
      />
    </div>
    
    <div class="mt-4">
      <AddRuleButton
        :disabled="isAddingDisabled"
        :disabled-reason="addDisabledReason"
        @add="handleAdd"
      />
    </div>
  </SettingsCard>
</template>
