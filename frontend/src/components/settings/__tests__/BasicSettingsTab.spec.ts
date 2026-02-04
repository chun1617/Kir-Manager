import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BasicSettingsTab from '../BasicSettingsTab.vue'

// Mock vue-i18n
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('Feature: settings-tabbed-layout, BasicSettingsTab component', () => {
  const defaultProps = {
    kiroInstallPath: '/path/to/kiro',
    kiroVersion: '0.8.206',
    language: 'zh-TW',
    lowBalanceThreshold: 0.2,
  }

  describe('設定項目渲染', () => {
    it('should render Kiro install path setting', () => {
      const wrapper = mount(BasicSettingsTab, { props: defaultProps })
      
      expect(wrapper.text()).toContain('settings.kiroInstallPath')
      const input = wrapper.find('input[type="text"]')
      expect(input.exists()).toBe(true)
    })

    it('should render Kiro version setting', () => {
      const wrapper = mount(BasicSettingsTab, { props: defaultProps })
      
      expect(wrapper.text()).toContain('settings.kiroVersion')
    })

    it('should render language setting', () => {
      const wrapper = mount(BasicSettingsTab, { props: defaultProps })
      
      expect(wrapper.text()).toContain('settings.language')
      expect(wrapper.text()).toContain('繁體中文')
      expect(wrapper.text()).toContain('简体中文')
    })

    it('should render low balance threshold setting', () => {
      const wrapper = mount(BasicSettingsTab, { props: defaultProps })
      
      expect(wrapper.text()).toContain('settings.lowBalanceThreshold')
      const rangeInput = wrapper.find('input[type="range"]')
      expect(rangeInput.exists()).toBe(true)
    })
  })

  describe('設定值顯示', () => {
    it('should display current kiro install path', () => {
      const wrapper = mount(BasicSettingsTab, { props: defaultProps })
      
      const inputs = wrapper.findAll('input[type="text"]')
      expect((inputs[0].element as HTMLInputElement).value).toBe('/path/to/kiro')
    })

    it('should display current kiro version', () => {
      const wrapper = mount(BasicSettingsTab, { props: defaultProps })
      
      const inputs = wrapper.findAll('input[type="text"]')
      expect((inputs[1].element as HTMLInputElement).value).toBe('0.8.206')
    })

    it('should highlight current language', () => {
      const wrapper = mount(BasicSettingsTab, { props: defaultProps })
      
      const langButtons = wrapper.findAll('button').filter(b => 
        b.text() === '繁體中文' || b.text() === '简体中文'
      )
      const twButton = langButtons.find(b => b.text() === '繁體中文')
      expect(twButton?.classes().join(' ')).toContain('bg-zinc-700')
    })

    it('should display current threshold percentage', () => {
      const wrapper = mount(BasicSettingsTab, { props: defaultProps })
      
      expect(wrapper.text()).toContain('20%')
    })
  })

  describe('事件發射', () => {
    it('should emit update:language when language button clicked', async () => {
      const wrapper = mount(BasicSettingsTab, { props: defaultProps })
      
      const cnButton = wrapper.findAll('button').find(b => b.text() === '简体中文')
      await cnButton?.trigger('click')
      
      expect(wrapper.emitted('update:language')).toBeTruthy()
      expect(wrapper.emitted('update:language')![0]).toEqual(['zh-CN'])
    })

    it('should emit detectVersion when detect button clicked', async () => {
      const wrapper = mount(BasicSettingsTab, { props: defaultProps })
      
      const detectButtons = wrapper.findAll('button').filter(b => 
        b.text() === 'settings.detectVersion'
      )
      await detectButtons[0]?.trigger('click')
      
      expect(wrapper.emitted('detectVersion')).toBeTruthy()
    })
  })
})
