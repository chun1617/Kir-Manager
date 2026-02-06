/**
 * 測試數據生成器驗證
 * @description 確保所有 arbitrary 生成器能正確產生符合類型的數據
 */
import { describe, it, expect } from 'vitest'
import * as fc from 'fast-check'
import {
  backupItemArbitrary,
  folderItemArbitrary,
  currentUsageInfoArbitrary,
  appSettingsArbitrary,
  autoSwitchSettingsArbitrary,
  autoSwitchStatusArbitrary,
  pathDetectionResultArbitrary,
  resultArbitrary,
  consistentBackupItemArbitrary,
  backupListArbitrary,
  folderListArbitrary,
} from './arbitraries'

describe('Arbitraries - 測試數據生成器', () => {
  describe('backupItemArbitrary', () => {
    it('應生成有效的 BackupItem', () => {
      fc.assert(
        fc.property(backupItemArbitrary, (item) => {
          expect(item).toHaveProperty('name')
          expect(item).toHaveProperty('backupTime')
          expect(item).toHaveProperty('hasToken')
          expect(item).toHaveProperty('balance')
          expect(typeof item.name).toBe('string')
          expect(typeof item.hasToken).toBe('boolean')
          expect(typeof item.balance).toBe('number')
          return true
        }),
        { numRuns: 20 }
      )
    })
  })

  describe('folderItemArbitrary', () => {
    it('應生成有效的 FolderItem', () => {
      fc.assert(
        fc.property(folderItemArbitrary, (item) => {
          expect(item).toHaveProperty('id')
          expect(item).toHaveProperty('name')
          expect(item).toHaveProperty('order')
          expect(typeof item.id).toBe('string')
          expect(typeof item.order).toBe('number')
          expect(item.order).toBeGreaterThanOrEqual(0)
          return true
        }),
        { numRuns: 20 }
      )
    })
  })

  describe('currentUsageInfoArbitrary', () => {
    it('應生成有效的 CurrentUsageInfo', () => {
      fc.assert(
        fc.property(currentUsageInfoArbitrary, (info) => {
          expect(info).toHaveProperty('subscriptionTitle')
          expect(info).toHaveProperty('usageLimit')
          expect(info).toHaveProperty('currentUsage')
          expect(info).toHaveProperty('balance')
          expect(info).toHaveProperty('isLowBalance')
          expect(typeof info.usageLimit).toBe('number')
          expect(info.usageLimit).toBeGreaterThanOrEqual(0)
          return true
        }),
        { numRuns: 20 }
      )
    })
  })

  describe('consistentBackupItemArbitrary', () => {
    it('應生成用量數據一致的 BackupItem (balance = usageLimit - currentUsage)', () => {
      fc.assert(
        fc.property(consistentBackupItemArbitrary, (item) => {
          const expectedBalance = item.usageLimit - item.currentUsage
          // 允許浮點數誤差
          expect(Math.abs(item.balance - expectedBalance)).toBeLessThan(0.01)
          expect(item.currentUsage).toBeLessThanOrEqual(item.usageLimit)
          return true
        }),
        { numRuns: 20 }
      )
    })
  })

  describe('backupListArbitrary', () => {
    it('withCurrentItem=true 時應確保只有一個 isCurrent=true', () => {
      fc.assert(
        fc.property(backupListArbitrary({ minLength: 2, maxLength: 5, withCurrentItem: true }), (list) => {
          const currentItems = list.filter(item => item.isCurrent)
          expect(currentItems.length).toBeLessThanOrEqual(1)
          return true
        }),
        { numRuns: 20 }
      )
    })
  })

  describe('autoSwitchStatusArbitrary', () => {
    it('status 應為有效值', () => {
      fc.assert(
        fc.property(autoSwitchStatusArbitrary, (status) => {
          expect(['stopped', 'running', 'cooldown']).toContain(status.status)
          expect(status.cooldownRemaining).toBeGreaterThanOrEqual(0)
          return true
        }),
        { numRuns: 20 }
      )
    })
  })
})
