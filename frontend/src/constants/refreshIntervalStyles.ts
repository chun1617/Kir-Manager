/**
 * 刷新頻率卡片樣式常量
 * @description 定義刷新頻率相關組件的 Tailwind CSS 樣式
 * @requirements 1.2, 1.4
 */
export const REFRESH_INTERVAL_STYLES = {
  /** 卡片容器樣式 */
  card: 'bg-zinc-800/50 rounded-xl p-6 border border-zinc-700/50',
  
  /** 規則行樣式 */
  ruleRow: 'flex items-center gap-2 py-2',
  
  /** 輸入框樣式 */
  input: {
    /** 餘額輸入框 */
    balance: 'w-20 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-zinc-100 text-sm',
    /** 間隔輸入框 */
    interval: 'w-16 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-zinc-100 text-sm',
    /** 錯誤狀態 */
    error: 'border-red-500',
  },
  
  /** 刪除按鈕樣式 */
  deleteButton: {
    /** 啟用狀態 */
    enabled: 'text-zinc-500 hover:text-red-500 transition-colors',
    /** 禁用狀態 */
    disabled: 'text-zinc-700 cursor-not-allowed',
  },
  
  /** 新增按鈕樣式 */
  addButton: {
    /** 啟用狀態 */
    enabled: 'text-zinc-400 hover:text-zinc-200 transition-colors flex items-center gap-1',
    /** 禁用狀態 */
    disabled: 'text-zinc-600 cursor-not-allowed flex items-center gap-1',
  },
  
  /** 標籤樣式 */
  label: 'text-sm text-zinc-400',
  
  /** 箭頭樣式 */
  arrow: 'text-zinc-500 mx-1',
} as const
