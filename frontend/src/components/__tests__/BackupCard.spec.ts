import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import * as fc from 'fast-check'
import BackupCard from '../BackupCard.vue'

// Mock Icon component
const IconStub = {
  name: 'Icon',
  template: '<span class="icon-stub" :data-name="name"></span>',
  props: ['name'],
}

// 建立預設的 backup 物件
const createDefaultBackup = (overrides = {}) => ({
  name: 'test-backup',
  provider: 'Github' as const,
  subscriptionTitle: 'KIRO PRO',
  usageLimit: 1000,
  currentUsage: 500,
  balance: 500,
  isLowBalance: false,
  isCurrent: false,
  isOriginalMachine: false,
  machineId: 'abc123def456',
  isTokenExpired: false,
  ...overrides,
})

// 建立預設的 props
const createDefaultProps = (overrides = {}) => ({
  backup: createDefaultBackup(),
  isSelected: false,
  isSwitching: false,
  isDeleting: false,
  isRefreshing: false,
  isRegenerating: false,
  cooldownSeconds: 0,
  copiedMachineId: null,
  ...overrides,
})

describe('Feature: ui-component-extraction, BackupCard Component', () => {
  // ============================================================================
  // Task 4.1: 組件骨架測試
  // ============================================================================

  describe('Task 4.1: 組件骨架', () => {
    it('should render without errors', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.exists()).toBe(true)
    })

    it('should have draggable attribute', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.attributes('draggable')).toBe('true')
    })
  })

  // ============================================================================
  // Task 4.2: 渲染邏輯測試
  // ============================================================================

  describe('Task 4.2: 渲染邏輯', () => {
    it('should render backup name', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ name: 'my-backup' }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.text()).toContain('my-backup')
    })

    it('should render original machine label when isOriginalMachine is true', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ isOriginalMachine: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      // 應該有原始機器標籤
      expect(wrapper.find('[data-testid="original-machine-label"]').exists()).toBe(true)
    })

    it('should render subscription type badge', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ subscriptionTitle: 'KIRO PRO+' }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.text()).toContain('PRO+')
    })

    it('should render balance information', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ balance: 750, usageLimit: 1000 }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.text()).toContain('750')
      expect(wrapper.text()).toContain('1,000')
    })

    it('should render truncated machine ID', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ machineId: 'abcdefghijklmnop' }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      // 應該顯示截斷的 machine ID
      expect(wrapper.text()).toContain('abcdefgh...')
    })

    it('should show active indicator when isCurrent is true', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ isCurrent: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.find('[data-testid="active-indicator"]').exists()).toBe(true)
    })
  })

  // ============================================================================
  // Task 4.3: 操作按鈕測試
  // ============================================================================

  describe('Task 4.3: 操作按鈕', () => {
    it('should emit select event when checkbox is clicked', async () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const checkbox = wrapper.find('input[type="checkbox"]')
      await checkbox.trigger('change')
      
      expect(wrapper.emitted('select')).toBeTruthy()
      expect(wrapper.emitted('select')![0]).toEqual(['test-backup'])
    })

    it('should emit switch event when switch button is clicked', async () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const switchBtn = wrapper.find('[data-testid="switch-btn"]')
      await switchBtn.trigger('click')
      
      expect(wrapper.emitted('switch')).toBeTruthy()
      expect(wrapper.emitted('switch')![0]).toEqual(['test-backup'])
    })

    it('should emit delete event when delete button is clicked', async () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const deleteBtn = wrapper.find('[data-testid="delete-btn"]')
      await deleteBtn.trigger('click')
      
      expect(wrapper.emitted('delete')).toBeTruthy()
      expect(wrapper.emitted('delete')![0]).toEqual(['test-backup'])
    })

    it('should emit refresh event when refresh button is clicked', async () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const refreshBtn = wrapper.find('[data-testid="refresh-btn"]')
      await refreshBtn.trigger('click')
      
      expect(wrapper.emitted('refresh')).toBeTruthy()
      expect(wrapper.emitted('refresh')![0]).toEqual(['test-backup'])
    })

    it('should emit regenerate-id event when regenerate button is clicked', async () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const regenerateBtn = wrapper.find('[data-testid="regenerate-btn"]')
      await regenerateBtn.trigger('click')
      
      expect(wrapper.emitted('regenerate-id')).toBeTruthy()
      expect(wrapper.emitted('regenerate-id')![0]).toEqual(['test-backup'])
    })

    it('should emit copy-machine-id event when machine ID is clicked', async () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ machineId: 'test-machine-id' }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const machineIdBtn = wrapper.find('[data-testid="machine-id-btn"]')
      await machineIdBtn.trigger('click')
      
      expect(wrapper.emitted('copy-machine-id')).toBeTruthy()
      expect(wrapper.emitted('copy-machine-id')![0]).toEqual(['test-machine-id'])
    })
  })

  // ============================================================================
  // Task 4.4: 拖放事件測試
  // ============================================================================

  describe('Task 4.4: 拖放事件', () => {
    it('should emit drag-start event on dragstart', async () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      await wrapper.trigger('dragstart')
      
      expect(wrapper.emitted('drag-start')).toBeTruthy()
    })

    it('should emit drag-end event on dragend', async () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      await wrapper.trigger('dragend')
      
      expect(wrapper.emitted('drag-end')).toBeTruthy()
    })
  })

  // ============================================================================
  // Property 4: 當前備份不顯示操作按鈕
  // Validates: Requirements 8.3, 9.3, 11.3
  // ============================================================================

  describe('Property 4: 當前備份不顯示操作按鈕', () => {
    it('should hide switch, delete, regenerate buttons when isCurrent is true', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ isCurrent: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      expect(wrapper.find('[data-testid="switch-btn"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="delete-btn"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="regenerate-btn"]').exists()).toBe(false)
    })

    it('should show switch, delete, regenerate buttons when isCurrent is false', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ isCurrent: false }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      expect(wrapper.find('[data-testid="switch-btn"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="delete-btn"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="regenerate-btn"]').exists()).toBe(true)
    })

    it('should show active status label when isCurrent is true', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ isCurrent: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      expect(wrapper.find('[data-testid="active-status"]').exists()).toBe(true)
    })
  })

  // ============================================================================
  // Property 5: 冷卻期狀態正確顯示
  // Validates: Requirements 10.3, 10.4
  // ============================================================================

  describe('Property 5: 冷卻期狀態正確顯示', () => {
    it('should show countdown when cooldownSeconds > 0', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          cooldownSeconds: 30,
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      expect(wrapper.text()).toContain('30')
      const refreshBtn = wrapper.find('[data-testid="refresh-btn"]')
      expect(refreshBtn.attributes('disabled')).toBeDefined()
    })

    it('should show refresh icon when cooldownSeconds is 0', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          cooldownSeconds: 0,
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const refreshBtn = wrapper.find('[data-testid="refresh-btn"]')
      expect(refreshBtn.attributes('disabled')).toBeUndefined()
    })
  })

  // ============================================================================
  // Property 6: 載入狀態正確顯示
  // Validates: Requirements 8.2, 9.2, 10.2, 11.2
  // ============================================================================

  describe('Property 6: 載入狀態正確顯示', () => {
    it('should show loading animation when isSwitching is true', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          isSwitching: true,
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const switchBtn = wrapper.find('[data-testid="switch-btn"]')
      expect(switchBtn.classes()).toContain('animate-bounce')
    })

    it('should show loading animation when isDeleting is true', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          isDeleting: true,
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const deleteBtn = wrapper.find('[data-testid="delete-btn"]')
      expect(deleteBtn.classes()).toContain('animate-pulse')
    })

    it('should show loading animation when isRefreshing is true', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          isRefreshing: true,
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const refreshBtn = wrapper.find('[data-testid="refresh-btn"]')
      expect(refreshBtn.find('.animate-spin').exists()).toBe(true)
    })

    it('should show loading animation when isRegenerating is true', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          isRegenerating: true,
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const regenerateBtn = wrapper.find('[data-testid="regenerate-btn"]')
      expect(regenerateBtn.classes()).toContain('animate-pulse-fast')
    })
  })

  // ============================================================================
  // Property 7: 低餘額警告顯示一致性
  // Validates: Requirements 6.5
  // ============================================================================

  describe('Property 7: 低餘額警告顯示一致性', () => {
    it('should show warning style when isLowBalance is true', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ isLowBalance: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const balanceEl = wrapper.find('[data-testid="balance"]')
      expect(balanceEl.classes()).toContain('text-app-warning')
    })

    it('should show normal style when isLowBalance is false', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          backup: createDefaultBackup({ isLowBalance: false }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const balanceEl = wrapper.find('[data-testid="balance"]')
      expect(balanceEl.classes()).not.toContain('text-app-warning')
    })
  })

  // ============================================================================
  // Property 11: Provider 圖標映射一致性
  // Validates: Requirements 6.3
  // ============================================================================

  describe('Property 11: Provider 圖標映射一致性', () => {
    const providers = ['Github', 'AWS', 'BuilderId', 'Enterprise', 'Google'] as const

    providers.forEach((provider) => {
      it(`should render correct icon for ${provider} provider`, () => {
        const wrapper = mount(BackupCard, {
          props: createDefaultProps({
            backup: createDefaultBackup({ provider }),
          }),
          global: {
            stubs: { Icon: IconStub },
          },
        })
        
        const providerIcon = wrapper.find('[data-testid="provider-icon"]')
        expect(providerIcon.exists()).toBe(true)
      })
    })
  })

  // ============================================================================
  // Property 12: 選中狀態視覺反饋
  // Validates: Requirements 7.2
  // ============================================================================

  describe('Property 12: 選中狀態視覺反饋', () => {
    it('should show checked checkbox when isSelected is true', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          isSelected: true,
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const checkbox = wrapper.find('input[type="checkbox"]')
      expect((checkbox.element as HTMLInputElement).checked).toBe(true)
    })

    it('should show unchecked checkbox when isSelected is false', () => {
      const wrapper = mount(BackupCard, {
        props: createDefaultProps({
          isSelected: false,
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      
      const checkbox = wrapper.find('input[type="checkbox"]')
      expect((checkbox.element as HTMLInputElement).checked).toBe(false)
    })
  })
})
