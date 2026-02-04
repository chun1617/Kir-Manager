import { ref, computed, type Ref, type ComputedRef } from 'vue'
import type { SettingsTab } from '@/types/settings'

/**
 * 設定頁面狀態介面
 */
export interface SettingsPageState {
  /** 當前選中的 Tab */
  activeTab: Ref<SettingsTab>
  /** Tab 切換中狀態 */
  isTabSwitching: Ref<boolean>
  /** 開關切換中狀態 */
  isToggling: Ref<boolean>
  /** Tab 是否禁用（與 isToggling 同步） */
  isTabDisabled: ComputedRef<boolean>
  /** 處理 Tab 切換 */
  handleTabChange: (tab: SettingsTab) => void
  /** 重置狀態（頁面重入時調用） */
  resetState: () => void
}

/**
 * 設定頁面 Composable
 * @description 管理設定頁面的 Tab 狀態和切換邏輯
 * @requirements 1.2, 1.3, 1.4, 1.5, 4.1, 4.2, 4.3, 4.4
 */
export function useSettingsPage(): SettingsPageState {
  // Tab 狀態
  const activeTab = ref<SettingsTab>('basic')
  const isTabSwitching = ref(false)
  
  // 操作狀態
  const isToggling = ref(false)
  
  // Property 6: Tab 禁用狀態與操作狀態同步
  const isTabDisabled = computed(() => isToggling.value)
  
  /**
   * 處理 Tab 切換
   * @description 切換前觸發 blur 事件以保存變更
   * @requirements 4.1, 4.2 - Tab 切換前觸發 blur 事件
   */
  const handleTabChange = (tab: SettingsTab) => {
    // 如果 Tab 被禁用，不執行切換
    if (isTabDisabled.value) return
    
    // 觸發當前焦點元素的 blur 事件
    const activeElement = document.activeElement as HTMLElement
    if (activeElement && typeof activeElement.blur === 'function') {
      activeElement.blur()
    }
    
    // Property 1: Tab 切換更新狀態
    activeTab.value = tab
  }
  
  /**
   * 重置狀態
   * @description 頁面重入時重置為預設狀態
   * @requirements 1.5 - 頁面重入重置狀態
   */
  const resetState = () => {
    // Property 2: 頁面重入重置狀態
    activeTab.value = 'basic'
    isTabSwitching.value = false
    isToggling.value = false
  }
  
  return {
    activeTab,
    isTabSwitching,
    isToggling,
    isTabDisabled,
    handleTabChange,
    resetState,
  }
}
