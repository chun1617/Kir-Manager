<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { TabItem, SettingsTab } from '@/types/settings'
import { TAB_STYLES } from '@/constants/settingsStyles'
import Icon from '@/components/Icon.vue'

/**
 * TabBar 組件 Props
 * @requirements 1.1, 1.3, 1.4, 4.3, 4.4, 6.1, 6.2, 6.3, 6.4
 */
interface Props {
  /** Tab 項目列表 */
  tabs: TabItem[]
  /** 當前選中的 Tab */
  activeTab: SettingsTab
  /** 是否禁用 Tab 切換 */
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
})

const emit = defineEmits<{
  /** 更新選中的 Tab */
  (e: 'update:activeTab', tab: SettingsTab): void
  /** Tab 切換前事件 */
  (e: 'beforeChange', tab: SettingsTab): void
}>()

const { t } = useI18n()

/**
 * 取得 Tab 樣式類別
 * @requirements 5.1, 5.2 - 響應式佈局（whitespace-nowrap 防止換行）
 */
const getTabClass = (tab: TabItem) => {
  if (props.disabled) {
    return `${TAB_STYLES.button} ${TAB_STYLES.disabled} ${TAB_STYLES.inactive}`
  }
  
  if (tab.id === props.activeTab) {
    return `${TAB_STYLES.button} ${TAB_STYLES.active}`
  }
  
  return `${TAB_STYLES.button} ${TAB_STYLES.inactive}`
}

/**
 * 處理 Tab 點擊
 */
const handleTabClick = (tab: TabItem) => {
  if (props.disabled) return
  if (tab.id === props.activeTab) return
  
  emit('beforeChange', tab.id)
  emit('update:activeTab', tab.id)
}
</script>

<template>
  <div :class="TAB_STYLES.container" role="tablist">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      :class="getTabClass(tab)"
      :aria-selected="tab.id === activeTab"
      :aria-disabled="disabled"
      :disabled="disabled"
      role="tab"
      @click="handleTabClick(tab)"
    >
      <Icon v-if="tab.icon" :name="tab.icon as any" class="w-4 h-4" />
      {{ t(tab.labelKey) }}
    </button>
  </div>
</template>
