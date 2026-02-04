import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import TabBar from '../TabBar.vue'
import type { TabItem } from '@/types/settings'

// Mock vue-i18n
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const mockTabs: TabItem[] = [
  { id: 'basic', labelKey: 'settings.tabs.basic' },
  { id: 'autoSwitch', labelKey: 'settings.tabs.autoSwitch' },
]

describe('Feature: settings-tabbed-layout, TabBar component', () => {
  describe('Tab 渲染', () => {
    it('should render correct number of tabs', () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
        },
      })
      
      const buttons = wrapper.findAll('button[role="tab"]')
      expect(buttons).toHaveLength(2)
    })

    it('should render tab labels using i18n keys', () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
        },
      })
      
      const buttons = wrapper.findAll('button')
      expect(buttons[0].text()).toBe('settings.tabs.basic')
      expect(buttons[1].text()).toBe('settings.tabs.autoSwitch')
    })
  })

  describe('選中狀態樣式', () => {
    it('should apply active style to selected tab', () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
        },
      })
      
      const buttons = wrapper.findAll('button')
      expect(buttons[0].classes().join(' ')).toContain('bg-zinc-800/50')
      expect(buttons[0].attributes('aria-selected')).toBe('true')
    })

    it('should apply inactive style to unselected tab', () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
        },
      })
      
      const buttons = wrapper.findAll('button')
      expect(buttons[1].classes().join(' ')).toContain('text-zinc-500')
      expect(buttons[1].attributes('aria-selected')).toBe('false')
    })
  })

  describe('禁用狀態行為', () => {
    it('should apply disabled style when disabled is true', () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
          disabled: true,
        },
      })
      
      const buttons = wrapper.findAll('button')
      buttons.forEach(button => {
        expect(button.classes().join(' ')).toContain('opacity-50')
        expect(button.classes().join(' ')).toContain('cursor-not-allowed')
        expect(button.attributes('aria-disabled')).toBe('true')
      })
    })

    it('should not emit events when disabled', async () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
          disabled: true,
        },
      })
      
      await wrapper.findAll('button')[1].trigger('click')
      
      expect(wrapper.emitted('update:activeTab')).toBeUndefined()
      expect(wrapper.emitted('beforeChange')).toBeUndefined()
    })
  })

  describe('Tab 切換事件', () => {
    it('should emit beforeChange and update:activeTab when clicking different tab', async () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
        },
      })
      
      await wrapper.findAll('button')[1].trigger('click')
      
      expect(wrapper.emitted('beforeChange')).toHaveLength(1)
      expect(wrapper.emitted('beforeChange')![0]).toEqual(['autoSwitch'])
      expect(wrapper.emitted('update:activeTab')).toHaveLength(1)
      expect(wrapper.emitted('update:activeTab')![0]).toEqual(['autoSwitch'])
    })

    it('should not emit events when clicking already active tab', async () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
        },
      })
      
      await wrapper.findAll('button')[0].trigger('click')
      
      expect(wrapper.emitted('update:activeTab')).toBeUndefined()
      expect(wrapper.emitted('beforeChange')).toBeUndefined()
    })
  })

  describe('Accessibility', () => {
    it('should have correct ARIA attributes', () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
        },
      })
      
      const tablist = wrapper.find('[role="tablist"]')
      expect(tablist.exists()).toBe(true)
      
      const buttons = wrapper.findAll('button[role="tab"]')
      expect(buttons).toHaveLength(2)
    })
  })

  describe('響應式佈局 (Requirements 5.1, 5.2)', () => {
    it('should apply flex-nowrap to keep tabs in single row', () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
        },
      })
      
      const tablist = wrapper.find('[role="tablist"]')
      const classes = tablist.classes().join(' ')
      expect(classes).toContain('flex-nowrap')
    })

    it('should apply overflow-x-auto for horizontal scrolling support', () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
        },
      })
      
      const tablist = wrapper.find('[role="tablist"]')
      const classes = tablist.classes().join(' ')
      expect(classes).toContain('overflow-x-auto')
    })

    it('should apply whitespace-nowrap to tab buttons to prevent wrapping', () => {
      const wrapper = mount(TabBar, {
        props: {
          tabs: mockTabs,
          activeTab: 'basic',
        },
      })
      
      const buttons = wrapper.findAll('button')
      buttons.forEach(button => {
        expect(button.classes().join(' ')).toContain('whitespace-nowrap')
      })
    })
  })
})
