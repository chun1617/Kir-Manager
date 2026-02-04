import { describe, it, expect } from 'vitest'
import fc from 'fast-check'
import { useRuleValidation } from '../useRuleValidation'
import type { RefreshRule } from '@/types/refreshInterval'

describe('Feature: refresh-interval-settings-ui, useRuleValidation', () => {
  const { validateMinBalance, validateInterval, validateMaxBalance, checkRangeOverlap } = useRuleValidation()

  describe('Property 5: 輸入值自動修正', () => {
    describe('NaN 防護', () => {
      it('validateMinBalance 應將 NaN 修正為 0', () => {
        expect(validateMinBalance(NaN)).toBe(0)
      })

      it('validateInterval 應將 NaN 修正為 1', () => {
        expect(validateInterval(NaN)).toBe(1)
      })
    })

    it('validateMinBalance 應將負數修正為 0', () => {
      fc.assert(
        fc.property(fc.integer({ min: -1000, max: -1 }), (negativeValue) => {
          const result = validateMinBalance(negativeValue)
          expect(result).toBe(0)
        }),
        { numRuns: 100 }
      )
    })

    it('validateMinBalance 應保留非負數', () => {
      fc.assert(
        fc.property(fc.integer({ min: 0, max: 10000 }), (nonNegativeValue) => {
          const result = validateMinBalance(nonNegativeValue)
          expect(result).toBe(nonNegativeValue)
        }),
        { numRuns: 100 }
      )
    })

    it('validateInterval 應將小於 1 的值修正為 1', () => {
      fc.assert(
        fc.property(fc.integer({ min: -100, max: 0 }), (smallValue) => {
          const result = validateInterval(smallValue)
          expect(result).toBe(1)
        }),
        { numRuns: 100 }
      )
    })

    it('validateInterval 應保留大於等於 1 的值', () => {
      fc.assert(
        fc.property(fc.integer({ min: 1, max: 1000 }), (validValue) => {
          const result = validateInterval(validValue)
          expect(result).toBe(validValue)
        }),
        { numRuns: 100 }
      )
    })
  })

  describe('Property 6: 餘額範圍驗證', () => {
    it('當 minBalance > maxBalance 且 maxBalance != -1 時應返回 invalid', () => {
      fc.assert(
        fc.property(
          fc.integer({ min: 1, max: 1000 }),
          fc.integer({ min: 0, max: 999 }),
          (minBalance, maxBalance) => {
            fc.pre(minBalance > maxBalance)
            const result = validateMaxBalance(maxBalance, minBalance)
            expect(result.valid).toBe(false)
            expect(result.error).toBe('refreshInterval.minGreaterThanMax')
          }
        ),
        { numRuns: 100 }
      )
    })

    it('當 maxBalance = -1 時應始終返回 valid', () => {
      fc.assert(
        fc.property(fc.integer({ min: 0, max: 10000 }), (minBalance) => {
          const result = validateMaxBalance(-1, minBalance)
          expect(result.valid).toBe(true)
        }),
        { numRuns: 100 }
      )
    })

    it('當 minBalance <= maxBalance 時應返回 valid', () => {
      fc.assert(
        fc.property(
          fc.integer({ min: 0, max: 500 }),
          fc.integer({ min: 500, max: 1000 }),
          (minBalance, maxBalance) => {
            const result = validateMaxBalance(maxBalance, minBalance)
            expect(result.valid).toBe(true)
          }
        ),
        { numRuns: 100 }
      )
    })
  })

  describe('Property 10: 區間重疊檢測', () => {
    const createRule = (id: string, min: number, max: number): RefreshRule => ({
      id,
      minBalance: min,
      maxBalance: max,
      interval: 1,
    })

    it('完全重疊的區間應返回 true', () => {
      const existingRules = [createRule('existing', 50, 100)]
      const newRule = createRule('new', 60, 80)
      expect(checkRangeOverlap(newRule, existingRules)).toBe(true)
    })

    it('部分重疊的區間應返回 true', () => {
      const existingRules = [createRule('existing', 50, 100)]
      const newRule = createRule('new', 80, 120)
      expect(checkRangeOverlap(newRule, existingRules)).toBe(true)
    })

    it('不重疊的區間應返回 false', () => {
      const existingRules = [createRule('existing', 50, 100)]
      const newRule = createRule('new', 100, 150)
      expect(checkRangeOverlap(newRule, existingRules)).toBe(false)
    })

    it('無上限區間與有限區間重疊應返回 true', () => {
      const existingRules = [createRule('existing', 100, -1)]
      const newRule = createRule('new', 150, 200)
      expect(checkRangeOverlap(newRule, existingRules)).toBe(true)
    })

    it('排除自身 ID 時不應檢測到重疊', () => {
      const existingRules = [createRule('self', 50, 100)]
      const newRule = createRule('self', 50, 100)
      expect(checkRangeOverlap(newRule, existingRules, 'self')).toBe(false)
    })

    it('屬性測試：相鄰但不重疊的區間', () => {
      fc.assert(
        fc.property(
          fc.integer({ min: 0, max: 100 }),
          fc.integer({ min: 1, max: 100 }),
          (start, width) => {
            const existingRules = [createRule('existing', start, start + width)]
            const newRule = createRule('new', start + width, start + width + 50)
            // 相鄰區間不應重疊
            expect(checkRangeOverlap(newRule, existingRules)).toBe(false)
          }
        ),
        { numRuns: 100 }
      )
    })
  })
})
