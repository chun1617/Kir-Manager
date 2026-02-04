import { ref, computed, type Ref } from 'vue'
import type { RefreshRule, ValidationResult } from '@/types/refreshInterval'
import { useRuleValidation } from './useRuleValidation'
import { NEW_RULE_DEFAULTS, MAX_RULES, ADD_DEBOUNCE_MS } from '@/constants/refreshIntervalDefaults'

/**
 * useRefreshIntervals composable 返回類型
 */
export interface UseRefreshIntervalsReturn {
  /** 規則列表 */
  rules: Ref<RefreshRule[]>
  /** 是否禁用新增按鈕 */
  isAddingDisabled: Ref<boolean>
  /** 禁用原因 i18n key */
  addDisabledReason: Ref<string | null>
  /** 新增規則 */
  addRule: () => RefreshRule | null
  /** 更新規則 */
  updateRule: (id: string, field: keyof RefreshRule, value: number | boolean) => ValidationResult
  /** 刪除規則 */
  deleteRule: (id: string) => boolean
  /** 格式化規則顯示 */
  formatRule: (rule: RefreshRule) => string
  /** 排序規則（按餘額下限降序） */
  sortRules: () => void
  /** 檢查規則是否可刪除 */
  canDeleteRule: (id: string) => boolean
}

/**
 * 生成唯一 ID
 */
function generateId(): string {
  return `rule-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
}

/**
 * 刷新頻率規則管理 composable
 * @param initialRules 初始規則列表
 * @param onSave 保存回調函數
 * @requirements 1.2, 1.3, 2.1, 2.3, 3.1, 3.2, 3.3, 3.6, 4.1, 6.1, 7.1, 7.2
 */
export function useRefreshIntervals(
  initialRules: RefreshRule[],
  onSave?: (rules: RefreshRule[]) => void
): UseRefreshIntervalsReturn {
  const rules = ref<RefreshRule[]>([...initialRules])
  const lastAddTime = ref<number>(0)
  const { validateMinBalance, validateInterval, validateMaxBalance, checkRangeOverlap } = useRuleValidation()

  /**
   * 是否禁用新增按鈕
   * @requirements 7.1, 7.2
   */
  const isAddingDisabled = computed(() => {
    const now = Date.now()
    const isDebouncing = now - lastAddTime.value < ADD_DEBOUNCE_MS
    const isMaxReached = rules.value.length >= MAX_RULES
    return isDebouncing || isMaxReached
  })

  /**
   * 禁用原因
   * @requirements 7.2
   */
  const addDisabledReason = computed(() => {
    if (rules.value.length >= MAX_RULES) {
      return 'refreshInterval.maxRulesReached'
    }
    return null
  })

  /**
   * 排序規則（按餘額下限降序）
   * @requirements 6.1
   */
  const sortRules = (): void => {
    rules.value.sort((a, b) => b.minBalance - a.minBalance)
  }

  /**
   * 觸發保存
   */
  const triggerSave = (): void => {
    if (onSave) {
      onSave([...rules.value])
    }
  }

  /**
   * 新增規則
   * @returns 新增的規則或 null（如果被防抖動阻止）
   * @requirements 2.1, 7.1, 7.2
   */
  const addRule = (): RefreshRule | null => {
    const now = Date.now()
    
    // 防抖動檢查
    if (now - lastAddTime.value < ADD_DEBOUNCE_MS) {
      return null
    }
    
    // 數量上限檢查
    if (rules.value.length >= MAX_RULES) {
      return null
    }

    const newRule: RefreshRule = {
      id: generateId(),
      ...NEW_RULE_DEFAULTS,
    }

    rules.value.push(newRule)
    lastAddTime.value = now
    sortRules()
    triggerSave()
    
    return newRule
  }

  /**
   * 更新規則
   * @param id 規則 ID
   * @param field 欄位名稱
   * @param value 新值
   * @returns 驗證結果
   * @requirements 3.1, 3.2, 3.3, 3.4, 3.5
   */
  const updateRule = (
    id: string,
    field: keyof RefreshRule,
    value: number | boolean
  ): ValidationResult => {
    const ruleIndex = rules.value.findIndex(r => r.id === id)
    if (ruleIndex === -1) {
      return { valid: false, error: 'refreshInterval.ruleNotFound' }
    }

    const rule = rules.value[ruleIndex]
    let newValue: number

    switch (field) {
      case 'minBalance':
        newValue = validateMinBalance(value as number)
        rule.minBalance = newValue
        // 檢查是否導致下限大於上限
        if (rule.maxBalance !== -1) {
          const validation = validateMaxBalance(rule.maxBalance, newValue)
          if (!validation.valid) {
            return validation
          }
        }
        break

      case 'maxBalance':
        // 處理無上限 checkbox
        if (typeof value === 'boolean') {
          rule.maxBalance = value ? -1 : 100 // 取消勾選時恢復預設值 100
        } else {
          newValue = value as number
          const validation = validateMaxBalance(newValue, rule.minBalance)
          if (!validation.valid) {
            return validation
          }
          rule.maxBalance = newValue
        }
        break

      case 'interval':
        newValue = validateInterval(value as number)
        rule.interval = newValue
        break

      default:
        return { valid: false, error: 'refreshInterval.invalidField' }
    }

    // 檢查區間重疊
    if (field === 'minBalance' || field === 'maxBalance') {
      if (checkRangeOverlap(rule, rules.value, id)) {
        return { valid: false, error: 'refreshInterval.rangeOverlap' }
      }
    }

    sortRules()
    triggerSave()
    return { valid: true }
  }

  /**
   * 檢查規則是否可刪除
   * @param id 規則 ID
   * @returns 是否可刪除
   * @requirements 4.2
   */
  const canDeleteRule = (id: string): boolean => {
    return rules.value.length > 1
  }

  /**
   * 刪除規則
   * @param id 規則 ID
   * @returns 是否成功刪除
   * @requirements 4.1, 4.2
   */
  const deleteRule = (id: string): boolean => {
    // 至少保留一條規則
    if (rules.value.length <= 1) {
      return false
    }

    const index = rules.value.findIndex(r => r.id === id)
    if (index === -1) {
      return false
    }

    rules.value.splice(index, 1)
    triggerSave()
    return true
  }

  /**
   * 格式化規則顯示
   * @param rule 規則
   * @returns 格式化字串
   * @requirements 1.2, 1.3
   */
  const formatRule = (rule: RefreshRule): string => {
    const maxDisplay = rule.maxBalance === -1 ? '無上限' : rule.maxBalance.toString()
    return `${rule.minBalance} - ${maxDisplay} → ${rule.interval} 分鐘`
  }

  return {
    rules,
    isAddingDisabled,
    addDisabledReason,
    addRule,
    updateRule,
    deleteRule,
    formatRule,
    sortRules,
    canDeleteRule,
  }
}
