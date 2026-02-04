/**
 * 設定分頁類型
 * @description 定義設定頁面可用的分頁標識
 */
export type SettingsTab = 'basic' | 'autoSwitch'

/**
 * Tab 項目定義
 * @description 定義單一分頁的配置結構
 */
export interface TabItem {
  /** 分頁唯一標識 */
  id: SettingsTab
  /** i18n 翻譯鍵值 */
  labelKey: string
  /** 圖標名稱 */
  icon?: string
}

/**
 * 設定卡片定義
 * @description 定義自動切換分頁中的卡片配置
 */
export interface SettingsCardConfig {
  /** 卡片唯一標識 */
  id: string
  /** i18n 翻譯鍵值 */
  titleKey: string
  /** 是否可見（根據 autoSwitchEnabled 狀態決定） */
  visible: boolean
}
