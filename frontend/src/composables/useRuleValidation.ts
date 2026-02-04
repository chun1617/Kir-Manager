import type { RefreshRule, ValidationResult } from '@/types/refreshInterval'

/**
 * useRuleValidation composable 返回類型
 */
export interface UseRuleValidationReturn {
  /** 驗證餘額下限（負數修正為 0） */
  validateMinBalance: (value: number) => number
  /** 驗證刷新間隔（小於 1 修正為 1） */
  validateInterval: (value: number) => number
  /** 驗證餘額上限（檢查下限大於上限） */
  validateMaxBalance: (value: number, minBalance: number) => ValidationResult
  /** 檢查區間重疊 */
  checkRangeOverlap: (rule: RefreshRule, existingRules: RefreshRule[], excludeId?: string) => boolean
}

/**
 * 規則驗證 composable
 * @description 提供刷新頻率規則的驗證邏輯
 * @requirements 5.1, 5.2, 5.3, 8.1
 */
export function useRuleValidation(): UseRuleValidationReturn {
  /**
   * 驗證餘額下限
   * @param value 輸入值
   * @returns 修正後的值（NaN 或負數修正為 0）
   * @requirements 5.1
   */
  const validateMinBalance = (value: number): number => {
    if (Number.isNaN(value)) return 0
    return Math.max(0, value)
  }

  /**
   * 驗證刷新間隔
   * @param value 輸入值
   * @returns 修正後的值（NaN 或小於 1 修正為 1）
   * @requirements 5.2
   */
  const validateInterval = (value: number): number => {
    if (Number.isNaN(value)) return 1
    return Math.max(1, value)
  }

  /**
   * 驗證餘額上限
   * @param value 餘額上限值
   * @param minBalance 餘額下限值
   * @returns 驗證結果
   * @requirements 5.3
   */
  const validateMaxBalance = (value: number, minBalance: number): ValidationResult => {
    if (value !== -1 && minBalance > value) {
      return { valid: false, error: 'refreshInterval.minGreaterThanMax' }
    }
    return { valid: true }
  }

  /**
   * 檢查區間重疊
   * @param rule 要檢查的規則
   * @param existingRules 現有規則列表
   * @param excludeId 排除的規則 ID（用於更新時排除自身）
   * @returns 是否有重疊
   * @requirements 8.1
   */
  const checkRangeOverlap = (
    rule: RefreshRule,
    existingRules: RefreshRule[],
    excludeId?: string
  ): boolean => {
    const otherRules = existingRules.filter(r => r.id !== excludeId)
    return otherRules.some(other => {
      const ruleMax = rule.maxBalance === -1 ? Infinity : rule.maxBalance
      const otherMax = other.maxBalance === -1 ? Infinity : other.maxBalance
      // 區間 [a, b] 和 [c, d] 重疊條件：a < d && b > c
      return rule.minBalance < otherMax && ruleMax > other.minBalance
    })
  }

  return {
    validateMinBalance,
    validateInterval,
    validateMaxBalance,
    checkRangeOverlap,
  }
}
