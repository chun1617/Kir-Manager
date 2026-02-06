import { describe, it, expect } from 'vitest'
import * as fc from 'fast-check'
import { truncateMachineId } from '../machineId'

describe('Feature: ui-component-extraction, utils/machineId', () => {
  // ============================================================================
  // Property Tests: truncateMachineId
  // Validates: Requirements 6.1
  // ============================================================================

  describe('Property: truncateMachineId 一致性', () => {
    it('should return consistent result for same input', () => {
      fc.assert(
        fc.property(
          fc.string({ minLength: 0, maxLength: 100 }),
          fc.integer({ min: 1, max: 20 }),
          (machineId, length) => {
            const result1 = truncateMachineId(machineId, length)
            const result2 = truncateMachineId(machineId, length)
            return result1 === result2
          }
        ),
        { numRuns: 100 }
      )
    })

    it('should never return string longer than length + 3 (for "...")', () => {
      fc.assert(
        fc.property(
          fc.string({ minLength: 1, maxLength: 100 }),
          fc.integer({ min: 1, max: 20 }),
          (machineId, length) => {
            const result = truncateMachineId(machineId, length)
            // 結果長度應該 <= length + 3 (加上 "...")
            return result.length <= length + 3
          }
        ),
        { numRuns: 100 }
      )
    })

    it('should return original string if shorter than or equal to length', () => {
      fc.assert(
        fc.property(
          fc.string({ minLength: 1, maxLength: 10 }),
          (machineId) => {
            const length = machineId.length + 5 // 確保 length > machineId.length
            const result = truncateMachineId(machineId, length)
            return result === machineId
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  // ============================================================================
  // Unit Tests: truncateMachineId
  // ============================================================================

  describe('truncateMachineId', () => {
    it('should return empty string for empty input', () => {
      expect(truncateMachineId('')).toBe('')
    })

    it('should return empty string for null/undefined', () => {
      expect(truncateMachineId(null as any)).toBe('')
      expect(truncateMachineId(undefined as any)).toBe('')
    })

    it('should return original string if shorter than default length (8)', () => {
      expect(truncateMachineId('abc')).toBe('abc')
      expect(truncateMachineId('12345678')).toBe('12345678')
    })

    it('should truncate and add ellipsis if longer than default length', () => {
      expect(truncateMachineId('123456789')).toBe('12345678...')
      expect(truncateMachineId('abcdefghijklmnop')).toBe('abcdefgh...')
    })

    it('should respect custom length parameter', () => {
      expect(truncateMachineId('abcdefghij', 5)).toBe('abcde...')
      expect(truncateMachineId('abcdefghij', 10)).toBe('abcdefghij')
      expect(truncateMachineId('abcdefghij', 15)).toBe('abcdefghij')
    })

    it('should handle typical machine ID format', () => {
      const machineId = 'fbc2127b1dfba39ceea01d5d988149a3f105220213636d9f1dfc226c40aa8c12'
      const result = truncateMachineId(machineId)
      expect(result).toBe('fbc2127b...')
      expect(result.length).toBe(11) // 8 + 3 for "..."
    })

    it('should handle UUID-like format', () => {
      const uuid = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'
      const result = truncateMachineId(uuid, 13)
      expect(result).toBe('a1b2c3d4-e5f6...')
    })
  })
})
