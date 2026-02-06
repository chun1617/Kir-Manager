import { describe, it, expect } from 'vitest'
import * as fc from 'fast-check'
import {
  subscriptionColorMap,
  getSubscriptionColorClass,
  getSubscriptionShortName,
} from '../subscription'

describe('Feature: ui-component-extraction, utils/subscription', () => {
  // ============================================================================
  // Property 2: 訂閱類型顏色映射一致性
  // Validates: Requirements 6.2
  // ============================================================================

  describe('Property 2: 訂閱類型顏色映射一致性', () => {
    it('should return consistent color class for same subscription type', () => {
      fc.assert(
        fc.property(
          fc.constantFrom('KIRO FREE', 'KIRO PRO', 'KIRO PRO+', 'KIRO POWER'),
          (subscriptionType) => {
            const result1 = getSubscriptionColorClass(subscriptionType)
            const result2 = getSubscriptionColorClass(subscriptionType)
            return result1 === result2 && typeof result1 === 'string'
          }
        ),
        { numRuns: 100 }
      )
    })

    it('should return default color for unknown subscription types', () => {
      fc.assert(
        fc.property(
          fc.string().filter(s => !['KIRO FREE', 'KIRO PRO', 'KIRO PRO+', 'KIRO POWER'].includes(s.toUpperCase())),
          (unknownType) => {
            const result = getSubscriptionColorClass(unknownType)
            // 應返回預設顏色類別
            return typeof result === 'string' && result.includes('bg-zinc-500')
          }
        ),
        { numRuns: 50 }
      )
    })

    it('should handle case-insensitive input', () => {
      fc.assert(
        fc.property(
          fc.constantFrom('kiro free', 'Kiro Pro', 'KIRO PRO+', 'kiro power'),
          (subscriptionType) => {
            const result = getSubscriptionColorClass(subscriptionType)
            return typeof result === 'string' && result.length > 0
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  // ============================================================================
  // Unit Tests: subscriptionColorMap
  // ============================================================================

  describe('subscriptionColorMap', () => {
    it('should have correct color for KIRO FREE', () => {
      expect(subscriptionColorMap['KIRO FREE']).toContain('bg-zinc-500')
    })

    it('should have correct color for KIRO PRO', () => {
      expect(subscriptionColorMap['KIRO PRO']).toContain('bg-blue-500')
    })

    it('should have correct color for KIRO PRO+', () => {
      expect(subscriptionColorMap['KIRO PRO+']).toContain('bg-violet-500')
    })

    it('should have correct color for KIRO POWER', () => {
      expect(subscriptionColorMap['KIRO POWER']).toContain('bg-amber-500')
    })
  })

  // ============================================================================
  // Unit Tests: getSubscriptionColorClass
  // ============================================================================

  describe('getSubscriptionColorClass', () => {
    it('should return correct class for each subscription type', () => {
      expect(getSubscriptionColorClass('KIRO FREE')).toContain('bg-zinc-500')
      expect(getSubscriptionColorClass('KIRO PRO')).toContain('bg-blue-500')
      expect(getSubscriptionColorClass('KIRO PRO+')).toContain('bg-violet-500')
      expect(getSubscriptionColorClass('KIRO POWER')).toContain('bg-amber-500')
    })

    it('should return default class for empty string', () => {
      expect(getSubscriptionColorClass('')).toContain('bg-zinc-500')
    })

    it('should return default class for null/undefined', () => {
      expect(getSubscriptionColorClass(null as any)).toContain('bg-zinc-500')
      expect(getSubscriptionColorClass(undefined as any)).toContain('bg-zinc-500')
    })

    it('should handle lowercase input', () => {
      expect(getSubscriptionColorClass('kiro free')).toContain('bg-zinc-500')
      expect(getSubscriptionColorClass('kiro pro')).toContain('bg-blue-500')
    })
  })

  // ============================================================================
  // Unit Tests: getSubscriptionShortName
  // ============================================================================

  describe('getSubscriptionShortName', () => {
    it('should return correct short name for each subscription type', () => {
      expect(getSubscriptionShortName('KIRO FREE')).toBe('FREE')
      expect(getSubscriptionShortName('KIRO PRO')).toBe('PRO')
      expect(getSubscriptionShortName('KIRO PRO+')).toBe('PRO+')
      expect(getSubscriptionShortName('KIRO POWER')).toBe('POWER')
    })

    it('should return empty string for empty input', () => {
      expect(getSubscriptionShortName('')).toBe('')
    })

    it('should return empty string for null/undefined', () => {
      expect(getSubscriptionShortName(null as any)).toBe('')
      expect(getSubscriptionShortName(undefined as any)).toBe('')
    })

    it('should handle lowercase input', () => {
      expect(getSubscriptionShortName('kiro free')).toBe('FREE')
      expect(getSubscriptionShortName('kiro pro')).toBe('PRO')
    })

    it('should return original string if not a known type', () => {
      expect(getSubscriptionShortName('UNKNOWN')).toBe('UNKNOWN')
    })
  })
})
