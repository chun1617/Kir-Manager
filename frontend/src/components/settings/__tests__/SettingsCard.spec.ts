import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SettingsCard from '../SettingsCard.vue'

describe('Feature: settings-tabbed-layout, SettingsCard component', () => {
  describe('卡片樣式', () => {
    it('should apply card styles from constants', () => {
      const wrapper = mount(SettingsCard)
      
      const card = wrapper.find('div')
      expect(card.classes().join(' ')).toContain('bg-zinc-800/50')
      expect(card.classes().join(' ')).toContain('rounded-xl')
      expect(card.classes().join(' ')).toContain('p-6')
      expect(card.classes().join(' ')).toContain('border')
    })
  })

  describe('標題渲染', () => {
    it('should render title when provided', () => {
      const wrapper = mount(SettingsCard, {
        props: {
          title: 'Test Title',
        },
      })
      
      const title = wrapper.find('h3')
      expect(title.exists()).toBe(true)
      expect(title.text()).toBe('Test Title')
    })

    it('should not render title element when not provided', () => {
      const wrapper = mount(SettingsCard)
      
      const title = wrapper.find('h3')
      expect(title.exists()).toBe(false)
    })
  })

  describe('插槽渲染', () => {
    it('should render default slot content', () => {
      const wrapper = mount(SettingsCard, {
        slots: {
          default: '<p>Slot Content</p>',
        },
      })
      
      expect(wrapper.find('p').text()).toBe('Slot Content')
    })

    it('should render slot with title', () => {
      const wrapper = mount(SettingsCard, {
        props: {
          title: 'Card Title',
        },
        slots: {
          default: '<input type="text" />',
        },
      })
      
      expect(wrapper.find('h3').text()).toBe('Card Title')
      expect(wrapper.find('input').exists()).toBe(true)
    })
  })
})
