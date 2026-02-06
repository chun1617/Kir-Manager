/**
 * 機器碼工具函數
 * @description 提供機器碼相關的格式化函數
 * @requirements 6.1 - 機器碼顯示
 */

/**
 * 截斷機器碼顯示
 * @param machineId 完整機器碼
 * @param length 顯示長度（預設 8）
 * @returns 截斷後的機器碼
 */
export function truncateMachineId(machineId: string, length: number = 8): string {
  if (!machineId) return ''
  if (machineId.length <= length) return machineId
  return machineId.substring(0, length) + '...'
}
