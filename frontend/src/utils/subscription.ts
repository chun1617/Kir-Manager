/**
 * 訂閱類型工具函數
 * @description 提供訂閱類型相關的顏色映射和格式化函數
 * @requirements 6.2 - 訂閱類型顏色映射
 */

/**
 * 訂閱類型顏色映射
 * @description 將訂閱類型映射到對應的 Tailwind CSS 類別
 */
export const subscriptionColorMap: Record<string, string> = {
  'KIRO FREE': 'bg-zinc-500/20 text-zinc-400 border border-zinc-500/30',
  'KIRO PRO': 'bg-blue-500/20 text-blue-400 border border-blue-500/30',
  'KIRO PRO+': 'bg-violet-500/20 text-violet-400 border border-violet-500/30',
  'KIRO POWER': 'bg-amber-500/20 text-amber-400 border border-amber-500/30',
}

/**
 * 獲取訂閱類型的顏色類別
 * @param subscriptionTitle 訂閱類型標題
 * @returns Tailwind CSS 類別字串
 */
export function getSubscriptionColorClass(subscriptionTitle: string): string {
  const upperTitle = subscriptionTitle?.toUpperCase() || ''
  return subscriptionColorMap[upperTitle] || 'bg-zinc-500/20 text-zinc-400 border border-zinc-500/30'
}

/**
 * 獲取訂閱類型的簡稱
 * @param subscriptionTitle 訂閱類型標題
 * @returns 簡稱字串
 */
export function getSubscriptionShortName(subscriptionTitle: string): string {
  if (!subscriptionTitle) return ''
  const upperTitle = subscriptionTitle.toUpperCase()
  return upperTitle.replace(/^KIRO\s+/, '')
}
