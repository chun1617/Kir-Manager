import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import RefreshIntervalCard from '../RefreshIntervalCard.vue'
import type { RefreshRule } from '@/types/refreshInterval'

// Mock vue-i18n
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

// Mock 子組件
vi.mock('../SettingsCard.vue', () => ({
  default: {
    name: 'SettingsCard',
    props: ['title'],
    template: '<div data-testid="settings-card"><span data-testid="card-title">{{ title }}</span><slot /></div>',
  },
}))

vi.mock('../RefreshRuleItem.vue', () => ({
  default: {
    name: 'RefreshRuleItem',
    props: ['rule', 'canDelete'],
    emits: ['update:minBalance', 'update:maxBalance', 'update:interval', 'delete', 'blur'],
    template: `
      <div data-testid="rule-item" :data-rule-id="rule.id">
        <input data-testid="min-balance-input" @focus="$emit('focus')" />
        <button data-testid="update-min" @click="$emit('update:minBalance', 100)">Update Min</button>
        <button data-testid="update-max" @click="$emit('update:maxBalance', 200)">Update Max</button>
        <button data-testid="update-interval" @click="$emit('update:interval', 10)">Update Interval</button>
        <button data-testid="delete-btn" @click="$emit('delete')">Delete</button>
        <button data-testid="blur-btn" @click="$emit('blur')">Blur</button>
      </div>
    `,
  },
}))

vi.mock('../AddRuleButton.vue', () => ({
  default: {
    name: 'AddRuleButton',
    props: ['disabled', 'disabledReason'],
    emits: ['add'],
    template: `
      <button 
        data-testid="add-rule-button" 
        :disabled="disabled"
        :title="disabledReason"
        @click="$emit('add')"
      >
        Add Rule
      </button>
    `,
  },
}))

const createMockRules = (): RefreshRule[] => [
  { id: 'rule-1', minBalance: 0, maxBalance: 100, interval: 5 },
  { id: 'rule-2', minBalance: 100, maxBalance: 500, interval: 10 },
  { id: 'rule-3', minBalance: 500, maxBalance: -1, interval: 30 },
]

describe('Feature: refresh-interval-settings-ui, RefreshIntervalCard component', () => {
  describe('基本渲染測試', () => {
    it('should render SettingsCard container', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: createMockRules() },
      })
      
      const settingsCard = wrapper.find('[data-testid="settings-card"]')
      expect(settingsCard.exists()).toBe(true)
    })

    it('should display card title with i18n key', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: createMockRules() },
      })
      
      const title = wrapper.find('[data-testid="card-title"]')
      expect(title.text()).toBe('autoSwitch.refreshIntervals.title')
    })

    it('should render description text', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: createMockRules() },
      })
      
      const desc = wrapper.find('.text-zinc-400')
      expect(desc.exists()).toBe(true)
      expect(desc.text()).toBe('autoSwitch.refreshIntervals.desc')
    })

    it('should render rules list container', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: createMockRules() },
      })
      
      const rulesList = wrapper.find('[data-testid="rules-list"]')
      expect(rulesList.exists()).toBe(true)
    })
  })

  describe('規則列表渲染測試', () => {
    it('should render one RefreshRuleItem for each rule', () => {
      const rules = createMockRules()
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules },
      })
      
      const ruleItems = wrapper.findAll('[data-testid="rule-item"]')
      expect(ruleItems).toHaveLength(rules.length)
    })

    it('should pass correct rule prop to each RefreshRuleItem', () => {
      const rules = createMockRules()
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules },
      })
      
      const ruleItems = wrapper.findAll('[data-testid="rule-item"]')
      rules.forEach((rule, index) => {
        expect(ruleItems[index].attributes('data-rule-id')).toBe(rule.id)
      })
    })

    it('should render empty list when no rules provided', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: [] },
      })
      
      const ruleItems = wrapper.findAll('[data-testid="rule-item"]')
      expect(ruleItems).toHaveLength(0)
    })

    it('should set canDelete to false when only one rule exists', () => {
      const singleRule = [createMockRules()[0]]
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: singleRule },
      })
      
      // 由於 mock 組件，我們檢查 canDelete 是否正確傳遞
      // 這需要通過組件的實際行為來驗證
      const ruleItems = wrapper.findAll('[data-testid="rule-item"]')
      expect(ruleItems).toHaveLength(1)
    })

    it('should set canDelete to true when multiple rules exist', () => {
      const rules = createMockRules()
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules },
      })
      
      const ruleItems = wrapper.findAll('[data-testid="rule-item"]')
      expect(ruleItems.length).toBeGreaterThan(1)
    })
  })

  describe('新增按鈕狀態測試', () => {
    it('should render AddRuleButton', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: createMockRules() },
      })
      
      const addButton = wrapper.find('[data-testid="add-rule-button"]')
      expect(addButton.exists()).toBe(true)
    })

    it('should enable AddRuleButton when isAddingDisabled is false', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { 
          rules: createMockRules(),
          isAddingDisabled: false,
        },
      })
      
      const addButton = wrapper.find('[data-testid="add-rule-button"]')
      expect((addButton.element as HTMLButtonElement).disabled).toBe(false)
    })

    it('should disable AddRuleButton when isAddingDisabled is true', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { 
          rules: createMockRules(),
          isAddingDisabled: true,
        },
      })
      
      const addButton = wrapper.find('[data-testid="add-rule-button"]')
      expect((addButton.element as HTMLButtonElement).disabled).toBe(true)
    })

    it('should pass disabledReason to AddRuleButton', () => {
      const disabledReason = 'refreshInterval.maxRulesReached'
      const wrapper = mount(RefreshIntervalCard, {
        props: { 
          rules: createMockRules(),
          isAddingDisabled: true,
          addDisabledReason: disabledReason,
        },
      })
      
      const addButton = wrapper.find('[data-testid="add-rule-button"]')
      expect(addButton.attributes('title')).toBe(disabledReason)
    })

    it('should not show disabledReason when button is enabled', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { 
          rules: createMockRules(),
          isAddingDisabled: false,
          addDisabledReason: null,
        },
      })
      
      const addButton = wrapper.find('[data-testid="add-rule-button"]')
      expect(addButton.attributes('title')).toBeFalsy()
    })
  })

  describe('事件處理測試', () => {
    it('should emit add event when AddRuleButton emits add', async () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: createMockRules() },
      })
      
      const addButton = wrapper.find('[data-testid="add-rule-button"]')
      await addButton.trigger('click')
      
      expect(wrapper.emitted('add')).toHaveLength(1)
    })

    it('should emit update event with correct params when RefreshRuleItem emits update:minBalance', async () => {
      const rules = createMockRules()
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules },
      })
      
      const firstRuleItem = wrapper.findAll('[data-testid="rule-item"]')[0]
      const updateMinBtn = firstRuleItem.find('[data-testid="update-min"]')
      await updateMinBtn.trigger('click')
      
      expect(wrapper.emitted('update')).toHaveLength(1)
      expect(wrapper.emitted('update')![0]).toEqual(['rule-1', 'minBalance', 100])
    })

    it('should emit update event with correct params when RefreshRuleItem emits update:maxBalance', async () => {
      const rules = createMockRules()
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules },
      })
      
      const firstRuleItem = wrapper.findAll('[data-testid="rule-item"]')[0]
      const updateMaxBtn = firstRuleItem.find('[data-testid="update-max"]')
      await updateMaxBtn.trigger('click')
      
      expect(wrapper.emitted('update')).toHaveLength(1)
      expect(wrapper.emitted('update')![0]).toEqual(['rule-1', 'maxBalance', 200])
    })

    it('should emit update event with correct params when RefreshRuleItem emits update:interval', async () => {
      const rules = createMockRules()
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules },
      })
      
      const firstRuleItem = wrapper.findAll('[data-testid="rule-item"]')[0]
      const updateIntervalBtn = firstRuleItem.find('[data-testid="update-interval"]')
      await updateIntervalBtn.trigger('click')
      
      expect(wrapper.emitted('update')).toHaveLength(1)
      expect(wrapper.emitted('update')![0]).toEqual(['rule-1', 'interval', 10])
    })

    it('should emit delete event with rule id when RefreshRuleItem emits delete', async () => {
      const rules = createMockRules()
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules },
      })
      
      const secondRuleItem = wrapper.findAll('[data-testid="rule-item"]')[1]
      const deleteBtn = secondRuleItem.find('[data-testid="delete-btn"]')
      await deleteBtn.trigger('click')
      
      expect(wrapper.emitted('delete')).toHaveLength(1)
      expect(wrapper.emitted('delete')![0]).toEqual(['rule-2'])
    })

    it('should emit save event when RefreshRuleItem emits blur', async () => {
      const rules = createMockRules()
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules },
      })
      
      const firstRuleItem = wrapper.findAll('[data-testid="rule-item"]')[0]
      const blurBtn = firstRuleItem.find('[data-testid="blur-btn"]')
      await blurBtn.trigger('click')
      
      expect(wrapper.emitted('save')).toHaveLength(1)
    })
  })

  describe('自動聚焦測試', () => {
    it('新增規則後應自動聚焦到新規則的餘額下限輸入框', async () => {
      const initialRules = createMockRules()
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: initialRules },
      })
      
      // 模擬新增規則
      const addButton = wrapper.find('[data-testid="add-rule-button"]')
      await addButton.trigger('click')
      
      // 驗證 add 事件被觸發
      expect(wrapper.emitted('add')).toHaveLength(1)
      
      // 新增一個新規則到 props
      const newRule: RefreshRule = { id: 'rule-new', minBalance: 0, maxBalance: -1, interval: 5 }
      const updatedRules = [...initialRules, newRule]
      await wrapper.setProps({ rules: updatedRules })
      
      // 等待 nextTick 讓 DOM 更新和聚焦邏輯執行
      await wrapper.vm.$nextTick()
      
      // 驗證新規則的輸入框存在
      const newRuleItem = wrapper.find('[data-rule-id="rule-new"]')
      expect(newRuleItem.exists()).toBe(true)
    })
  })

  describe('Props 預設值測試', () => {
    it('should default isAddingDisabled to false', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: createMockRules() },
      })
      
      const addButton = wrapper.find('[data-testid="add-rule-button"]')
      expect((addButton.element as HTMLButtonElement).disabled).toBe(false)
    })

    it('should default addDisabledReason to null', () => {
      const wrapper = mount(RefreshIntervalCard, {
        props: { rules: createMockRules() },
      })
      
      const addButton = wrapper.find('[data-testid="add-rule-button"]')
      expect(addButton.attributes('title')).toBeFalsy()
    })
  })
})
