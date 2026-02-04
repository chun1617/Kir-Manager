import type { TabItem } from '@/types/settings'

/**
 * 設定頁面 Tab 配置
 * @description 定義設定頁面的分頁列表
 * @requirements 1.1 - 顯示包含「基礎設定」和「自動切換」兩個 Tab
 */
export const SETTINGS_TABS: TabItem[] = [
  { id: 'basic', labelKey: 'settings.tabs.basic', icon: 'Globe' },
  { id: 'autoSwitch', labelKey: 'settings.tabs.autoSwitch', icon: 'Sparkles' },
]
