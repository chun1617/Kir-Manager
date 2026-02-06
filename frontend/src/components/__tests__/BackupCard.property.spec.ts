import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import * as fc from 'fast-check'
import BackupCard from '../BackupCard.vue'

// Mock Icon component
const IconStub = {
  name: 'Icon',
  template: '<span class="icon-stub" :data-name="name"></span>',
  props: ['name'],
}

// Arbitrary for backup data
const backupArbitrary = fc.record({
  name: fc.string({ minLength: 1, maxLength: 50 }),
  provider: fc.constantFrom('Github', 'AWS', 'BuilderId', 'Enterprise', 'Google') as fc.Arbitrary<'Github' | 'AWS' | 'BuilderId' | 'Enterprise' | 'Google'>,
  subscriptionTitle: fc.constantFrom('KIRO FREE', 'KIRO PRO', 'KIRO PRO+', 'KIRO POWER', ''),
  usageLimit: fc.integer({ min: 0, max: 10000 }),
  currentUsage: fc.integer({ min: 0, max: 10000 }),
  balance: fc.integer({ min: 0, max: 10000 }),
  isLowBalance: fc.boolean(),
  isCurrent: fc.boolean(),
  isOriginalMachine: fc.boolean(),
  machineId: fc.string({ minLength: 0, maxLength: 64 }),
  isTokenExpired: fc.boolean(),
})

describe('Feature: ui-component-extraction, BackupCard Property Tests', () => {
  // ============================================================================
  // Property 4: 當前備份不顯示操作按鈕
  // Validates: Requirements 8.3, 9.3, 11.3
  // ============================================================================

  describe('Property 4: 當前備份不顯示操作按鈕', () => {
    it('should consistently hide action buttons when isCurrent is true', () => {
      fc.assert(
        fc.property(
          backupArbitrary.map(b => ({ ...b, isCurrent: true })),
          (backup) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected: false,
                isSwitching: false,
                isDeleting: false,
                isRefreshing: false,
                isRegenerating: false,
                cooldownSeconds: 0,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const switchBtn = wrapper.find('[data-testid="switch-btn"]')
            const deleteBtn = wrapper.find('[data-testid="delete-btn"]')
            const regenerateBtn = wrapper.find('[data-testid="regenerate-btn"]')
            const activeStatus = wrapper.find('[data-testid="active-status"]')

            return (
              !switchBtn.exists() &&
              !deleteBtn.exists() &&
              !regenerateBtn.exists() &&
              activeStatus.exists()
            )
          }
        ),
        { numRuns: 50 }
      )
    })

    it('should consistently show action buttons when isCurrent is false', () => {
      fc.assert(
        fc.property(
          backupArbitrary.map(b => ({ ...b, isCurrent: false })),
          (backup) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected: false,
                isSwitching: false,
                isDeleting: false,
                isRefreshing: false,
                isRegenerating: false,
                cooldownSeconds: 0,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const switchBtn = wrapper.find('[data-testid="switch-btn"]')
            const deleteBtn = wrapper.find('[data-testid="delete-btn"]')
            const regenerateBtn = wrapper.find('[data-testid="regenerate-btn"]')
            const activeStatus = wrapper.find('[data-testid="active-status"]')

            return (
              switchBtn.exists() &&
              deleteBtn.exists() &&
              regenerateBtn.exists() &&
              !activeStatus.exists()
            )
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  // ============================================================================
  // Property 5: 冷卻期狀態正確顯示
  // Validates: Requirements 10.3, 10.4
  // ============================================================================

  describe('Property 5: 冷卻期狀態正確顯示', () => {
    it('should disable refresh button when cooldownSeconds > 0', () => {
      fc.assert(
        fc.property(
          backupArbitrary,
          fc.integer({ min: 1, max: 60 }),
          (backup, cooldownSeconds) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected: false,
                isSwitching: false,
                isDeleting: false,
                isRefreshing: false,
                isRegenerating: false,
                cooldownSeconds,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const refreshBtn = wrapper.find('[data-testid="refresh-btn"]')
            const hasDisabled = refreshBtn.attributes('disabled') !== undefined
            const showsCountdown = wrapper.text().includes(String(cooldownSeconds))

            return hasDisabled && showsCountdown
          }
        ),
        { numRuns: 50 }
      )
    })

    it('should enable refresh button when cooldownSeconds is 0', () => {
      fc.assert(
        fc.property(
          backupArbitrary,
          (backup) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected: false,
                isSwitching: false,
                isDeleting: false,
                isRefreshing: false,
                isRegenerating: false,
                cooldownSeconds: 0,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const refreshBtn = wrapper.find('[data-testid="refresh-btn"]')
            return refreshBtn.attributes('disabled') === undefined
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  // ============================================================================
  // Property 7: 低餘額警告顯示一致性
  // Validates: Requirements 6.5
  // ============================================================================

  describe('Property 7: 低餘額警告顯示一致性', () => {
    it('should show warning style when isLowBalance is true and has usage data', () => {
      fc.assert(
        fc.property(
          backupArbitrary.map(b => ({ ...b, isLowBalance: true, usageLimit: 1000, balance: 100 })),
          (backup) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected: false,
                isSwitching: false,
                isDeleting: false,
                isRefreshing: false,
                isRegenerating: false,
                cooldownSeconds: 0,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const balanceEl = wrapper.find('[data-testid="balance"]')
            return balanceEl.exists() && balanceEl.classes().includes('text-app-warning')
          }
        ),
        { numRuns: 50 }
      )
    })

    it('should show normal style when isLowBalance is false', () => {
      fc.assert(
        fc.property(
          backupArbitrary.map(b => ({ ...b, isLowBalance: false, usageLimit: 1000, balance: 500 })),
          (backup) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected: false,
                isSwitching: false,
                isDeleting: false,
                isRefreshing: false,
                isRegenerating: false,
                cooldownSeconds: 0,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const balanceEl = wrapper.find('[data-testid="balance"]')
            return balanceEl.exists() && !balanceEl.classes().includes('text-app-warning')
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  // ============================================================================
  // Property 11: Provider 圖標映射一致性
  // Validates: Requirements 6.3
  // ============================================================================

  describe('Property 11: Provider 圖標映射一致性', () => {
    it('should always render provider icon for any valid provider', () => {
      fc.assert(
        fc.property(
          backupArbitrary,
          (backup) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected: false,
                isSwitching: false,
                isDeleting: false,
                isRefreshing: false,
                isRegenerating: false,
                cooldownSeconds: 0,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const providerIcon = wrapper.find('[data-testid="provider-icon"]')
            return providerIcon.exists()
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  // ============================================================================
  // Property 12: 選中狀態視覺反饋
  // Validates: Requirements 7.2
  // ============================================================================

  describe('Property 12: 選中狀態視覺反饋', () => {
    it('should reflect isSelected state in checkbox', () => {
      fc.assert(
        fc.property(
          backupArbitrary,
          fc.boolean(),
          (backup, isSelected) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected,
                isSwitching: false,
                isDeleting: false,
                isRefreshing: false,
                isRegenerating: false,
                cooldownSeconds: 0,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const checkbox = wrapper.find('input[type="checkbox"]')
            const checkboxElement = checkbox.element as HTMLInputElement
            return checkboxElement.checked === isSelected
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  // ============================================================================
  // Property 6: 載入狀態正確顯示
  // Validates: Requirements 8.2, 9.2, 10.2, 11.2
  // ============================================================================

  describe('Property 6: 載入狀態正確顯示', () => {
    it('should show loading animation for switching state', () => {
      fc.assert(
        fc.property(
          backupArbitrary.map(b => ({ ...b, isCurrent: false })),
          (backup) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected: false,
                isSwitching: true,
                isDeleting: false,
                isRefreshing: false,
                isRegenerating: false,
                cooldownSeconds: 0,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const switchBtn = wrapper.find('[data-testid="switch-btn"]')
            return switchBtn.exists() && switchBtn.classes().includes('animate-bounce')
          }
        ),
        { numRuns: 30 }
      )
    })

    it('should show loading animation for deleting state', () => {
      fc.assert(
        fc.property(
          backupArbitrary.map(b => ({ ...b, isCurrent: false })),
          (backup) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected: false,
                isSwitching: false,
                isDeleting: true,
                isRefreshing: false,
                isRegenerating: false,
                cooldownSeconds: 0,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const deleteBtn = wrapper.find('[data-testid="delete-btn"]')
            return deleteBtn.exists() && deleteBtn.classes().includes('animate-pulse')
          }
        ),
        { numRuns: 30 }
      )
    })

    it('should show loading animation for regenerating state', () => {
      fc.assert(
        fc.property(
          backupArbitrary.map(b => ({ ...b, isCurrent: false })),
          (backup) => {
            const wrapper = mount(BackupCard, {
              props: {
                backup,
                isSelected: false,
                isSwitching: false,
                isDeleting: false,
                isRefreshing: false,
                isRegenerating: true,
                cooldownSeconds: 0,
                copiedMachineId: null,
              },
              global: {
                stubs: { Icon: IconStub },
              },
            })

            const regenerateBtn = wrapper.find('[data-testid="regenerate-btn"]')
            return regenerateBtn.exists() && regenerateBtn.classes().includes('animate-pulse-fast')
          }
        ),
        { numRuns: 30 }
      )
    })
  })
})
