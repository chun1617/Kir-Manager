import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import * as fc from 'fast-check'
import AutoSwitchTab from '../AutoSwitchTab.vue'
import RefreshIntervalCard from '../RefreshIntervalCard.vue'
import { getVisibleCards } from '@/constants/autoSwitchCards'
import type { RefreshRule } from '@/types/refreshInterval'

// Mock vue-i18n
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('Feature: settings-tabbed-layout, AutoSwitchTab component', () => {
  const defaultProps = {
    autoSwitchEnabled: false,
    balanceThreshold: 5,
    minTargetBalance: 50,
    monitorStatus: 'stopped' as const,
  }

  describe('Property 3: 卡片可見性與啟用狀態關係', () => {
    it('should show only 1 card when disabled, 5 cards when enabled', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          (enabled) => {
            const visibleCards = getVisibleCards(enabled)
            
            if (enabled) {
              return visibleCards.length === 5
            } else {
              return visibleCards.length === 1 && visibleCards[0].id === 'switchStatus'
            }
          }
        ),
        { numRuns: 100 }
      )
    })

    it('should render only switch status card when disabled', () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { ...defaultProps, autoSwitchEnabled: false },
      })
      
      // 只應該有開關卡片
      expect(wrapper.text()).toContain('autoSwitch.enabled')
      expect(wrapper.text()).not.toContain('autoSwitch.balanceThreshold')
    })

    it('should render all 5 cards when enabled', () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { ...defaultProps, autoSwitchEnabled: true },
      })
      
      // 應該有所有卡片
      expect(wrapper.text()).toContain('autoSwitch.enabled')
      expect(wrapper.text()).toContain('autoSwitch.balanceThreshold')
      expect(wrapper.text()).toContain('autoSwitch.folderFilter')
      expect(wrapper.text()).toContain('autoSwitch.refreshIntervals.title')
      expect(wrapper.text()).toContain('autoSwitch.notifyOnSwitch')
    })
  })

  describe('Property 4: 開關切換更新卡片和監控狀態', () => {
    it('should emit toggle event when switch is clicked', async () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { ...defaultProps, autoSwitchEnabled: false },
      })
      
      const toggle = wrapper.find('input[type="checkbox"]')
      await toggle.setValue(true)
      
      expect(wrapper.emitted('toggle')).toBeTruthy()
      expect(wrapper.emitted('toggle')![0]).toEqual([true])
    })
  })

  describe('監控狀態顯示', () => {
    it('should display stopped status', () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { ...defaultProps, monitorStatus: 'stopped' },
      })
      
      expect(wrapper.text()).toContain('autoSwitch.status.stopped')
    })

    it('should display running status when enabled', () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { ...defaultProps, autoSwitchEnabled: true, monitorStatus: 'running' },
      })
      
      expect(wrapper.text()).toContain('autoSwitch.status.running')
    })

    it('should display cooldown status', () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { ...defaultProps, autoSwitchEnabled: true, monitorStatus: 'cooldown' },
      })
      
      expect(wrapper.text()).toContain('autoSwitch.status.cooldown')
    })
  })

  describe('設定值更新', () => {
    it('should emit update:balanceThreshold when threshold changes', async () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { ...defaultProps, autoSwitchEnabled: true },
      })
      
      const inputs = wrapper.findAll('input[type="number"]')
      await inputs[0].setValue(10)
      
      expect(wrapper.emitted('update:balanceThreshold')).toBeTruthy()
    })
  })

  describe('Property 11: RefreshIntervalCard 可見性與啟用狀態關係', () => {
    const mockRefreshRules: RefreshRule[] = [
      { id: 'rule-1', minBalance: 0, maxBalance: 50, interval: 5 },
      { id: 'rule-2', minBalance: 50, maxBalance: -1, interval: 10 },
    ]

    it('should NOT render RefreshIntervalCard when autoSwitchEnabled is false', () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { 
          ...defaultProps, 
          autoSwitchEnabled: false,
          refreshRules: mockRefreshRules,
        },
      })
      
      expect(wrapper.findComponent(RefreshIntervalCard).exists()).toBe(false)
    })

    it('should render RefreshIntervalCard when autoSwitchEnabled is true', () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { 
          ...defaultProps, 
          autoSwitchEnabled: true,
          refreshRules: mockRefreshRules,
        },
      })
      
      expect(wrapper.findComponent(RefreshIntervalCard).exists()).toBe(true)
    })

    it('should pass refreshRules prop to RefreshIntervalCard', () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { 
          ...defaultProps, 
          autoSwitchEnabled: true,
          refreshRules: mockRefreshRules,
        },
      })
      
      const refreshCard = wrapper.findComponent(RefreshIntervalCard)
      expect(refreshCard.props('rules')).toEqual(mockRefreshRules)
    })

    it('should emit update:refreshRules when rules are updated via save event', async () => {
      const wrapper = mount(AutoSwitchTab, {
        props: { 
          ...defaultProps, 
          autoSwitchEnabled: true,
          refreshRules: mockRefreshRules,
        },
      })
      
      const refreshCard = wrapper.findComponent(RefreshIntervalCard)
      await refreshCard.vm.$emit('save')
      
      expect(wrapper.emitted('update:refreshRules')).toBeTruthy()
    })

    it('Property: RefreshIntervalCard visibility follows autoSwitchEnabled state', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          fc.array(
            fc.record({
              id: fc.string(),
              minBalance: fc.integer({ min: 0, max: 100 }),
              maxBalance: fc.oneof(fc.constant(-1), fc.integer({ min: 0, max: 1000 })),
              interval: fc.integer({ min: 1, max: 60 }),
            }),
            { minLength: 0, maxLength: 5 }
          ),
          (enabled, rules) => {
            const wrapper = mount(AutoSwitchTab, {
              props: { 
                ...defaultProps, 
                autoSwitchEnabled: enabled,
                refreshRules: rules,
              },
            })
            
            const cardExists = wrapper.findComponent(RefreshIntervalCard).exists()
            
            // 當 enabled 為 true 時，卡片應該存在；否則不應該存在
            return cardExists === enabled
          }
        ),
        { numRuns: 50 }
      )
    })
  })
})
