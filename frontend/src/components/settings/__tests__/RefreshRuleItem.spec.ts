import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import RefreshRuleItem from '../RefreshRuleItem.vue'
import type { RefreshRule } from '@/types/refreshInterval'
import { REFRESH_INTERVAL_STYLES } from '@/constants/refreshIntervalStyles'

// Mock vue-i18n
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const createMockRule = (overrides: Partial<RefreshRule> = {}): RefreshRule => ({
  id: 'rule-1',
  minBalance: 0,
  maxBalance: 100,
  interval: 5,
  ...overrides,
})

describe('Feature: refresh-interval-settings-ui, RefreshRuleItem component', () => {
  describe('基本渲染測試', () => {
    it('should render minBalance input', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const minBalanceInput = wrapper.find('input[data-testid="min-balance-input"]')
      expect(minBalanceInput.exists()).toBe(true)
      expect((minBalanceInput.element as HTMLInputElement).value).toBe('0')
    })

    it('should render maxBalance input', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const maxBalanceInput = wrapper.find('input[data-testid="max-balance-input"]')
      expect(maxBalanceInput.exists()).toBe(true)
      expect((maxBalanceInput.element as HTMLInputElement).value).toBe('100')
    })

    it('should render interval input', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const intervalInput = wrapper.find('input[data-testid="interval-input"]')
      expect(intervalInput.exists()).toBe(true)
      expect((intervalInput.element as HTMLInputElement).value).toBe('5')
    })


    it('should render unlimited checkbox', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const checkbox = wrapper.find('input[data-testid="unlimited-checkbox"]')
      expect(checkbox.exists()).toBe(true)
    })

    it('should render delete button', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const deleteButton = wrapper.find('button[data-testid="delete-button"]')
      expect(deleteButton.exists()).toBe(true)
    })
  })

  describe('Property 4: 無上限 checkbox 狀態同步', () => {
    it('should check checkbox when maxBalance === -1', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule({ maxBalance: -1 }) },
      })
      
      const checkbox = wrapper.find('input[data-testid="unlimited-checkbox"]')
      expect((checkbox.element as HTMLInputElement).checked).toBe(true)
    })

    it('should uncheck checkbox when maxBalance !== -1', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule({ maxBalance: 100 }) },
      })
      
      const checkbox = wrapper.find('input[data-testid="unlimited-checkbox"]')
      expect((checkbox.element as HTMLInputElement).checked).toBe(false)
    })

    it('should emit update:maxBalance with true when checkbox is checked', async () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule({ maxBalance: 100 }) },
      })
      
      const checkbox = wrapper.find('input[data-testid="unlimited-checkbox"]')
      await checkbox.setValue(true)
      
      expect(wrapper.emitted('update:maxBalance')).toHaveLength(1)
      expect(wrapper.emitted('update:maxBalance')![0]).toEqual([true])
    })

    it('should emit update:maxBalance with false when checkbox is unchecked', async () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule({ maxBalance: -1 }) },
      })
      
      const checkbox = wrapper.find('input[data-testid="unlimited-checkbox"]')
      await checkbox.setValue(false)
      
      expect(wrapper.emitted('update:maxBalance')).toHaveLength(1)
      expect(wrapper.emitted('update:maxBalance')![0]).toEqual([false])
    })

    it('should disable maxBalance input when checkbox is checked', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule({ maxBalance: -1 }) },
      })
      
      const maxBalanceInput = wrapper.find('input[data-testid="max-balance-input"]')
      expect((maxBalanceInput.element as HTMLInputElement).disabled).toBe(true)
    })

    it('should enable maxBalance input when checkbox is unchecked', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule({ maxBalance: 100 }) },
      })
      
      const maxBalanceInput = wrapper.find('input[data-testid="max-balance-input"]')
      expect((maxBalanceInput.element as HTMLInputElement).disabled).toBe(false)
    })
  })

  describe('Property 12: 刪除按鈕禁用狀態', () => {
    it('should enable delete button when canDelete is true', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule(), canDelete: true },
      })
      
      const deleteButton = wrapper.find('button[data-testid="delete-button"]')
      expect((deleteButton.element as HTMLButtonElement).disabled).toBe(false)
    })

    it('should disable delete button when canDelete is false', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule(), canDelete: false },
      })
      
      const deleteButton = wrapper.find('button[data-testid="delete-button"]')
      expect((deleteButton.element as HTMLButtonElement).disabled).toBe(true)
    })

    it('should show tooltip when delete button is disabled', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule(), canDelete: false },
      })
      
      const deleteButton = wrapper.find('button[data-testid="delete-button"]')
      expect(deleteButton.attributes('title')).toBe('refreshInterval.cannotDeleteLastRule')
    })

    it('should apply disabled style when canDelete is false', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule(), canDelete: false },
      })
      
      const deleteButton = wrapper.find('button[data-testid="delete-button"]')
      expect(deleteButton.classes().join(' ')).toContain('cursor-not-allowed')
    })

    it('should apply enabled style when canDelete is true', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule(), canDelete: true },
      })
      
      const deleteButton = wrapper.find('button[data-testid="delete-button"]')
      expect(deleteButton.classes().join(' ')).toContain('hover:text-red-500')
    })
  })


  describe('輸入事件測試', () => {
    it('should emit update:minBalance when minBalance input changes', async () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const minBalanceInput = wrapper.find('input[data-testid="min-balance-input"]')
      await minBalanceInput.setValue(50)
      
      expect(wrapper.emitted('update:minBalance')).toHaveLength(1)
      expect(wrapper.emitted('update:minBalance')![0]).toEqual([50])
    })

    it('should emit update:interval when interval input changes', async () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const intervalInput = wrapper.find('input[data-testid="interval-input"]')
      await intervalInput.setValue(10)
      
      expect(wrapper.emitted('update:interval')).toHaveLength(1)
      expect(wrapper.emitted('update:interval')![0]).toEqual([10])
    })

    it('should emit update:maxBalance when maxBalance input changes', async () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const maxBalanceInput = wrapper.find('input[data-testid="max-balance-input"]')
      await maxBalanceInput.setValue(200)
      
      expect(wrapper.emitted('update:maxBalance')).toHaveLength(1)
      expect(wrapper.emitted('update:maxBalance')![0]).toEqual([200])
    })

    it('should emit delete event when delete button is clicked', async () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule(), canDelete: true },
      })
      
      const deleteButton = wrapper.find('button[data-testid="delete-button"]')
      await deleteButton.trigger('click')
      
      expect(wrapper.emitted('delete')).toHaveLength(1)
    })

    it('should not emit delete event when delete button is disabled', async () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule(), canDelete: false },
      })
      
      const deleteButton = wrapper.find('button[data-testid="delete-button"]')
      await deleteButton.trigger('click')
      
      expect(wrapper.emitted('delete')).toBeUndefined()
    })
  })

  describe('錯誤樣式測試', () => {
    it('should apply error border style when hasError is true', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule(), hasError: true },
      })
      
      const container = wrapper.find('[data-testid="rule-row"]')
      expect(container.classes().join(' ')).toContain('border-red-500')
    })

    it('should not apply error border style when hasError is false', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule(), hasError: false },
      })
      
      const container = wrapper.find('[data-testid="rule-row"]')
      expect(container.classes().join(' ')).not.toContain('border-red-500')
    })
  })

  describe('樣式應用測試', () => {
    it('should apply ruleRow style to container', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const container = wrapper.find('[data-testid="rule-row"]')
      // REFRESH_INTERVAL_STYLES.ruleRow = 'flex items-center gap-2 py-2'
      expect(container.classes().join(' ')).toContain('flex')
      expect(container.classes().join(' ')).toContain('items-center')
    })

    it('should apply balance input style', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const minBalanceInput = wrapper.find('input[data-testid="min-balance-input"]')
      // REFRESH_INTERVAL_STYLES.input.balance = 'w-20 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-zinc-100 text-sm'
      expect(minBalanceInput.classes().join(' ')).toContain('w-20')
      expect(minBalanceInput.classes().join(' ')).toContain('bg-zinc-900')
    })

    it('should apply interval input style', () => {
      const wrapper = mount(RefreshRuleItem, {
        props: { rule: createMockRule() },
      })
      
      const intervalInput = wrapper.find('input[data-testid="interval-input"]')
      // REFRESH_INTERVAL_STYLES.input.interval = 'w-16 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-zinc-100 text-sm'
      expect(intervalInput.classes().join(' ')).toContain('w-16')
      expect(intervalInput.classes().join(' ')).toContain('bg-zinc-900')
    })
  })
})
