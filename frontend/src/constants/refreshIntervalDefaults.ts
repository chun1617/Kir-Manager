import type { RefreshRule } from '@/types/refreshInterval'

/**
 * 預設刷新頻率規則
 * @description 與後端 autoswitch.DefaultRefreshIntervals() 對應
 * @requirements 1.1
 */
export const DEFAULT_REFRESH_RULES: RefreshRule[] = [
  { id: 'default-1', minBalance: 100, maxBalance: -1, interval: 5 },
  { id: 'default-2', minBalance: 50, maxBalance: 100, interval: 2 },
  { id: 'default-3', minBalance: 0, maxBalance: 50, interval: 1 },
]

/**
 * 新規則預設值
 * @requirements 2.1
 */
export const NEW_RULE_DEFAULTS = {
  minBalance: 0,
  maxBalance: 0,
  interval: 1,
} as const

/**
 * 規則數量上限
 * @requirements 7.2
 */
export const MAX_RULES = 10

/**
 * 新增規則防抖動時間 (毫秒)
 * @requirements 7.1
 */
export const ADD_DEBOUNCE_MS = 500
