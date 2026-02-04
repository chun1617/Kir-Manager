import type { SettingsCardConfig } from '@/types/settings'

/**
 * 自動切換卡片配置類型別名
 */
export type AutoSwitchCardConfig = SettingsCardConfig

/**
 * 自動切換分頁卡片配置
 * @description 定義自動切換分頁中的卡片列表
 * @requirements 3.1, 3.2, 3.3, 3.4, 3.5 - 自動切換分頁內容
 */
export const AUTO_SWITCH_CARDS: SettingsCardConfig[] = [
  { id: 'switchStatus', titleKey: 'autoSwitch.cards.switchStatus', visible: true },
  { id: 'threshold', titleKey: 'autoSwitch.cards.threshold', visible: false },
  { id: 'filter', titleKey: 'autoSwitch.cards.filter', visible: false },
  { id: 'refreshRate', titleKey: 'autoSwitch.cards.refreshRate', visible: false },
  { id: 'notification', titleKey: 'autoSwitch.cards.notification', visible: false },
]

/**
 * 取得可見卡片列表
 * @param enabled 自動切換是否啟用
 * @returns 可見的卡片配置列表
 */
export function getVisibleCards(enabled: boolean): SettingsCardConfig[] {
  if (!enabled) {
    // 未啟用時只顯示開關卡片
    return AUTO_SWITCH_CARDS.filter(card => card.id === 'switchStatus')
  }
  // 啟用時顯示所有卡片
  return AUTO_SWITCH_CARDS.map(card => ({ ...card, visible: true }))
}
