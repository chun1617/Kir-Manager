/**
 * Tab 樣式常量
 * @description 定義 Tab 導航的樣式類別
 * @requirements 5.1, 5.2 - 響應式佈局規格
 * @requirements 6.1, 6.2, 6.3, 6.4 - Tab 樣式規格
 */
export const TAB_STYLES = {
  /** 選中狀態樣式 */
  active: 'bg-zinc-800/50 border border-zinc-700/50 text-zinc-100',
  /** 未選中狀態樣式 */
  inactive: 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900',
  /** 禁用狀態樣式 */
  disabled: 'opacity-50 cursor-not-allowed',
  /** 容器樣式 (含響應式：單行顯示 + 水平滾動) */
  container: 'flex gap-1 flex-nowrap overflow-x-auto',
  /** Tab 按鈕基礎樣式 (含 whitespace-nowrap 防止換行) */
  button: 'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors cursor-pointer whitespace-nowrap',
} as const

/**
 * 卡片樣式常量
 * @description 定義設定卡片的樣式類別
 * @requirements 7.1, 7.2 - 卡片佈局規格
 */
export const CARD_STYLES = {
  /** 卡片容器間距 */
  container: 'space-y-4',
  /** 單一卡片樣式 */
  card: 'bg-zinc-800/50 rounded-xl p-6 border border-zinc-700/50',
} as const
