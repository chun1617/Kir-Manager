import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import AddRuleButton from '../AddRuleButton.vue'
import { REFRESH_INTERVAL_STYLES } from '@/constants/refreshIntervalStyles'

// 建立測試用 i18n 實例
const i18n = createI18n({
  legacy: false,
  locale: 'zh-TW',
  messages: {
    'zh-TW': {
      refreshInterval: {
        addRule: '新增規則',
        maxRulesReached: '已達規則上限',
      },
    },
  },
})

// 測試輔助函數
function createWrapper(props = {}) {
  return mount(AddRuleButton, {
    props,
    global: {
      plugins: [i18n],
    },
  })
}

describe('AddRuleButton', () => {
  describe('基本渲染測試', () => {
    it('應該渲染新增按鈕', () => {
      const wrapper = createWrapper()
      
      expect(wrapper.find('[data-testid="add-rule-button"]').exists()).toBe(true)
    })

    it('按鈕應該顯示 i18n 文字', () => {
      const wrapper = createWrapper()
      
      expect(wrapper.text()).toContain('新增規則')
    })
  })

  describe('禁用狀態測試', () => {
    it('當 disabled 為 false 時，按鈕應該啟用', () => {
      const wrapper = createWrapper({ disabled: false })
      const button = wrapper.find('[data-testid="add-rule-button"]')
      
      expect(button.attributes('disabled')).toBeUndefined()
    })

    it('當 disabled 為 true 時，按鈕應該禁用', () => {
      const wrapper = createWrapper({ disabled: true })
      const button = wrapper.find('[data-testid="add-rule-button"]')
      
      expect(button.attributes('disabled')).toBeDefined()
    })

    it('禁用時應該顯示 disabledReason tooltip', () => {
      const wrapper = createWrapper({
        disabled: true,
        disabledReason: 'refreshInterval.maxRulesReached',
      })
      const button = wrapper.find('[data-testid="add-rule-button"]')
      
      expect(button.attributes('title')).toBe('已達規則上限')
    })
  })

  describe('點擊事件測試', () => {
    it('點擊啟用的按鈕應該 emit add 事件', async () => {
      const wrapper = createWrapper({ disabled: false })
      const button = wrapper.find('[data-testid="add-rule-button"]')
      
      await button.trigger('click')
      
      expect(wrapper.emitted('add')).toHaveLength(1)
    })

    it('點擊禁用的按鈕不應該 emit 事件', async () => {
      const wrapper = createWrapper({ disabled: true })
      const button = wrapper.find('[data-testid="add-rule-button"]')
      
      await button.trigger('click')
      
      expect(wrapper.emitted('add')).toBeUndefined()
    })
  })

  describe('樣式測試', () => {
    it('啟用時應該應用 enabled 樣式', () => {
      const wrapper = createWrapper({ disabled: false })
      const button = wrapper.find('[data-testid="add-rule-button"]')
      
      const expectedClasses = REFRESH_INTERVAL_STYLES.addButton.enabled.split(' ')
      expectedClasses.forEach(cls => {
        expect(button.classes()).toContain(cls)
      })
    })

    it('禁用時應該應用 disabled 樣式', () => {
      const wrapper = createWrapper({ disabled: true })
      const button = wrapper.find('[data-testid="add-rule-button"]')
      
      const expectedClasses = REFRESH_INTERVAL_STYLES.addButton.disabled.split(' ')
      expectedClasses.forEach(cls => {
        expect(button.classes()).toContain(cls)
      })
    })
  })
})
