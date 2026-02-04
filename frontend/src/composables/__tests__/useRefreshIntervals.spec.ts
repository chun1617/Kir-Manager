import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import fc from 'fast-check'
import { useRefreshIntervals } from '../useRefreshIntervals'
import { DEFAULT_REFRESH_RULES, MAX_RULES, ADD_DEBOUNCE_MS } from '@/constants/refreshIntervalDefaults'
import type { RefreshRule } from '@/types/refreshInterval'

describe('Feature: refresh-interval-settings-ui, useRefreshIntervals', () => {
  const createRule = (id: string, min: number, max: number, interval: number): RefreshRule => ({
    id,
    minBalance: min,
    maxBalance: max,
    interval,
  })

  describe('Property 1: 規則格式化輸出', () => {
    it('formatRule 應產生正確格式的字串', () => {
      const { formatRule } = useRefreshIntervals([])
      
      fc.assert(
        fc.property(
          fc.integer({ min: 0, max: 1000 }),
          fc.integer({ min: 1, max: 100 }),
          (minBalance, interval) => {
            const rule = createRule('test', minBalance, 500, interval)
            const result = formatRule(rule)
            expect(result).toBe(`${minBalance} - 500 → ${interval} 分鐘`)
          }
        ),
        { numRuns: 100 }
      )
    })

    it('formatRule 應將 maxBalance=-1 顯示為「無上限」', () => {
      const { formatRule } = useRefreshIntervals([])
      
      fc.assert(
        fc.property(
          fc.integer({ min: 0, max: 1000 }),
          fc.integer({ min: 1, max: 100 }),
          (minBalance, interval) => {
            const rule = createRule('test', minBalance, -1, interval)
            const result = formatRule(rule)
            expect(result).toBe(`${minBalance} - 無上限 → ${interval} 分鐘`)
          }
        ),
        { numRuns: 100 }
      )
    })
  })

  describe('Property 2: 規則列表長度不變量', () => {
    it('addRule 應使列表長度增加 1', () => {
      const { rules, addRule } = useRefreshIntervals([createRule('1', 0, 100, 1)])
      const initialLength = rules.value.length
      
      const result = addRule()
      
      expect(result).not.toBeNull()
      expect(rules.value.length).toBe(initialLength + 1)
    })

    it('deleteRule 應使列表長度減少 1（當有多條規則時）', () => {
      const initialRules = [
        createRule('1', 0, 50, 1),
        createRule('2', 50, 100, 2),
      ]
      const { rules, deleteRule } = useRefreshIntervals(initialRules)
      const initialLength = rules.value.length
      
      const result = deleteRule('1')
      
      expect(result).toBe(true)
      expect(rules.value.length).toBe(initialLength - 1)
    })

    it('deleteRule 應在只剩一條規則時返回 false', () => {
      const { rules, deleteRule } = useRefreshIntervals([createRule('1', 0, 100, 1)])
      
      const result = deleteRule('1')
      
      expect(result).toBe(false)
      expect(rules.value.length).toBe(1)
    })
  })

  describe('Property 3: 規則欄位更新', () => {
    it('updateRule 應正確更新 minBalance', () => {
      const { rules, updateRule } = useRefreshIntervals([createRule('1', 0, 100, 1)])
      
      fc.assert(
        fc.property(fc.integer({ min: 0, max: 99 }), (newMin) => {
          const result = updateRule('1', 'minBalance', newMin)
          expect(result.valid).toBe(true)
          expect(rules.value[0].minBalance).toBe(newMin)
        }),
        { numRuns: 50 }
      )
    })

    it('updateRule 應正確更新 interval', () => {
      const { rules, updateRule } = useRefreshIntervals([createRule('1', 0, 100, 1)])
      
      fc.assert(
        fc.property(fc.integer({ min: 1, max: 60 }), (newInterval) => {
          const result = updateRule('1', 'interval', newInterval)
          expect(result.valid).toBe(true)
          expect(rules.value[0].interval).toBe(newInterval)
        }),
        { numRuns: 50 }
      )
    })

    it('updateRule 應將負數 minBalance 修正為 0', () => {
      const { rules, updateRule } = useRefreshIntervals([createRule('1', 50, 100, 1)])
      
      updateRule('1', 'minBalance', -10)
      
      expect(rules.value[0].minBalance).toBe(0)
    })

    it('updateRule 應將小於 1 的 interval 修正為 1', () => {
      const { rules, updateRule } = useRefreshIntervals([createRule('1', 0, 100, 5)])
      
      updateRule('1', 'interval', 0)
      
      expect(rules.value[0].interval).toBe(1)
    })
  })

  describe('Property 7: 規則排序不變量', () => {
    it('sortRules 後應滿足降序不變量', () => {
      const initialRules = [
        createRule('1', 0, 50, 1),
        createRule('2', 100, -1, 5),
        createRule('3', 50, 100, 2),
      ]
      const { rules, sortRules } = useRefreshIntervals(initialRules)
      
      sortRules()
      
      for (let i = 0; i < rules.value.length - 1; i++) {
        expect(rules.value[i].minBalance).toBeGreaterThanOrEqual(rules.value[i + 1].minBalance)
      }
    })

    it('addRule 後規則應自動排序', () => {
      const { rules, addRule } = useRefreshIntervals([createRule('1', 100, -1, 5)])
      
      addRule() // 新規則 minBalance=0
      
      // 新規則 (minBalance=0) 應排在最後
      expect(rules.value[rules.value.length - 1].minBalance).toBe(0)
    })
  })

  describe('Property 8: 防抖動機制', () => {
    beforeEach(() => {
      vi.useFakeTimers()
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it('連續快速新增應被防抖動阻止', () => {
      const { rules, addRule } = useRefreshIntervals([createRule('1', 0, 100, 1)])
      
      const first = addRule()
      const second = addRule() // 應被阻止
      
      expect(first).not.toBeNull()
      expect(second).toBeNull()
      expect(rules.value.length).toBe(2)
    })

    it('超過防抖動時間後應可再次新增', () => {
      const { rules, addRule } = useRefreshIntervals([createRule('1', 0, 100, 1)])
      
      addRule()
      vi.advanceTimersByTime(ADD_DEBOUNCE_MS + 10)
      const result = addRule()
      
      expect(result).not.toBeNull()
      expect(rules.value.length).toBe(3)
    })
  })

  describe('Property 9: 規則數量上限', () => {
    it('達到上限時 isAddingDisabled 應為 true', () => {
      const maxRules = Array.from({ length: MAX_RULES }, (_, i) => 
        createRule(`rule-${i}`, i * 10, (i + 1) * 10, 1)
      )
      const { isAddingDisabled, addDisabledReason } = useRefreshIntervals(maxRules)
      
      expect(isAddingDisabled.value).toBe(true)
      expect(addDisabledReason.value).toBe('refreshInterval.maxRulesReached')
    })

    it('達到上限時 addRule 應返回 null', () => {
      const maxRules = Array.from({ length: MAX_RULES }, (_, i) => 
        createRule(`rule-${i}`, i * 10, (i + 1) * 10, 1)
      )
      const { addRule } = useRefreshIntervals(maxRules)
      
      const result = addRule()
      
      expect(result).toBeNull()
    })
  })

  describe('Property 12: 刪除按鈕禁用狀態', () => {
    it('只有一條規則時 canDeleteRule 應返回 false', () => {
      const { canDeleteRule } = useRefreshIntervals([createRule('1', 0, 100, 1)])
      
      expect(canDeleteRule('1')).toBe(false)
    })

    it('有多條規則時 canDeleteRule 應返回 true', () => {
      const { canDeleteRule } = useRefreshIntervals([
        createRule('1', 0, 50, 1),
        createRule('2', 50, 100, 2),
      ])
      
      expect(canDeleteRule('1')).toBe(true)
      expect(canDeleteRule('2')).toBe(true)
    })
  })

  describe('onSave 回調', () => {
    it('addRule 應觸發 onSave', () => {
      const onSave = vi.fn()
      const { addRule } = useRefreshIntervals([createRule('1', 0, 100, 1)], onSave)
      
      addRule()
      
      expect(onSave).toHaveBeenCalledTimes(1)
    })

    it('updateRule 應觸發 onSave', () => {
      const onSave = vi.fn()
      const { updateRule } = useRefreshIntervals([createRule('1', 0, 100, 1)], onSave)
      
      updateRule('1', 'interval', 5)
      
      expect(onSave).toHaveBeenCalledTimes(1)
    })

    it('deleteRule 應觸發 onSave', () => {
      const onSave = vi.fn()
      const { deleteRule } = useRefreshIntervals([
        createRule('1', 0, 50, 1),
        createRule('2', 50, 100, 2),
      ], onSave)
      
      deleteRule('1')
      
      expect(onSave).toHaveBeenCalledTimes(1)
    })
  })
})
