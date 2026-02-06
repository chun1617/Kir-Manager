import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import CurrentStatusCard from '../CurrentStatusCard.vue'
import type { CurrentUsageInfo, BackupItem } from '../../types/backup'

// Mock Icon component
const IconStub = {
  name: 'Icon',
  template: '<span class="icon-stub" :data-name="name"></span>',
  props: ['name', 'class'],
}

// 建立 i18n 實例
const i18n = createI18n({
  legacy: false,
  locale: 'zh-TW',
  messages: {
    'zh-TW': {
      status: {
        current: '當前',
        originalMachine: '原始機器',
        openSSOCache: '開啟 SSO Cache 資料夾',
      },
      backup: {
        create: '建立備份',
        refresh: '刷新',
      },
      restore: {
        original: '還原原始機器',
      },
      app: {
        processing: '處理中...',
      },
      message: {
        tokenExpiredTip: 'Token 已過期',
      },
    },
  },
})

// 建立預設的 usageInfo
const createDefaultUsageInfo = (overrides: Partial<CurrentUsageInfo> = {}): CurrentUsageInfo => ({
  subscriptionTitle: 'KIRO PRO',
  usageLimit: 1000,
  currentUsage: 500,
  balance: 500,
  isLowBalance: false,
  ...overrides,
})

// 建立預設的 activeBackup
const createDefaultBackup = (overrides: Partial<BackupItem> = {}): BackupItem => ({
  name: 'test-backup',
  backupTime: '2024-01-01T00:00:00Z',
  hasToken: true,
  hasMachineId: true,
  machineId: 'abc123def456',
  provider: 'Github',
  isCurrent: true,
  isOriginalMachine: false,
  isTokenExpired: false,
  subscriptionTitle: 'KIRO PRO',
  usageLimit: 1000,
  currentUsage: 500,
  balance: 500,
  isLowBalance: false,
  cachedAt: '2024-01-01T00:00:00Z',
  folderId: '',
  ...overrides,
})

// 建立預設 Props
const createDefaultProps = (overrides = {}) => ({
  currentEnvironmentName: 'Test Environment',
  currentMachineId: 'abc123def456',
  currentProvider: 'Github',
  usageInfo: createDefaultUsageInfo(),
  activeBackup: createDefaultBackup(),
  isRefreshing: false,
  isRestoring: false,
  cooldownSeconds: 0,
  ...overrides,
})

// 掛載組件的輔助函數
const mountComponent = (props = {}) => {
  return mount(CurrentStatusCard, {
    props: createDefaultProps(props),
    global: {
      plugins: [i18n],
      stubs: { Icon: IconStub },
    },
  })
}

describe('CurrentStatusCard', () => {
  // ============================================================================
  // Task 8.1: Props 和 Events 測試
  // ============================================================================

  describe('Task 8.1: Props 和 Events', () => {
    it('should accept all required props', () => {
      const wrapper = mountComponent()
      expect(wrapper.exists()).toBe(true)
    })

    it('should emit refresh event when refresh button is clicked', async () => {
      const wrapper = mountComponent()
      const refreshBtn = wrapper.find('[data-testid="refresh-btn"]')
      await refreshBtn.trigger('click')
      expect(wrapper.emitted('refresh')).toBeTruthy()
    })

    it('should emit create-backup event when create button is clicked', async () => {
      const wrapper = mountComponent()
      const createBtn = wrapper.find('[data-testid="create-backup-btn"]')
      await createBtn.trigger('click')
      expect(wrapper.emitted('create-backup')).toBeTruthy()
    })

    it('should emit restore-original event when restore button is clicked', async () => {
      const wrapper = mountComponent()
      const restoreBtn = wrapper.find('[data-testid="restore-original-btn"]')
      await restoreBtn.trigger('click')
      expect(wrapper.emitted('restore-original')).toBeTruthy()
    })

    it('should emit open-sso-cache event when provider icon is clicked', async () => {
      const wrapper = mountComponent()
      const providerIcon = wrapper.find('[data-testid="provider-icon"]')
      await providerIcon.trigger('click')
      expect(wrapper.emitted('open-sso-cache')).toBeTruthy()
    })
  })

  // ============================================================================
  // Task 8.2: 狀態顯示測試
  // ============================================================================

  describe('Task 8.2: 狀態顯示', () => {
    it('should render current environment name', () => {
      const wrapper = mountComponent({ currentEnvironmentName: 'My Environment' })
      expect(wrapper.text()).toContain('My Environment')
    })

    it('should render "原始機器" when currentEnvironmentName is empty', () => {
      const wrapper = mountComponent({ currentEnvironmentName: '' })
      expect(wrapper.text()).toContain('原始機器')
    })

    it('should render current machine ID', () => {
      const wrapper = mountComponent({ currentMachineId: 'xyz789' })
      expect(wrapper.text()).toContain('xyz789')
    })

    it('should render subscription type badge', () => {
      const wrapper = mountComponent({
        usageInfo: createDefaultUsageInfo({ subscriptionTitle: 'KIRO PRO+' }),
      })
      expect(wrapper.text()).toContain('PRO+')
    })

    it('should render balance information', () => {
      const wrapper = mountComponent({
        usageInfo: createDefaultUsageInfo({ balance: 750, usageLimit: 1000 }),
      })
      expect(wrapper.text()).toContain('750')
      expect(wrapper.text()).toContain('1,000')
    })

    it('should show low balance warning when isLowBalance is true', () => {
      const wrapper = mountComponent({
        usageInfo: createDefaultUsageInfo({ isLowBalance: true }),
      })
      // 應該顯示警告圖標
      const alertIcon = wrapper.find('[data-name="AlertTriangle"]')
      expect(alertIcon.exists()).toBe(true)
    })

    it('should render Github provider icon when provider is Github', () => {
      const wrapper = mountComponent({
        activeBackup: createDefaultBackup({ provider: 'Github' }),
      })
      const providerIcon = wrapper.find('[data-testid="provider-icon"] [data-name="Github"]')
      expect(providerIcon.exists()).toBe(true)
    })

    it('should render AWS provider icon when provider is AWS', () => {
      const wrapper = mountComponent({
        activeBackup: createDefaultBackup({ provider: 'AWS' }),
      })
      const providerIcon = wrapper.find('[data-testid="provider-icon"] [data-name="AWS"]')
      expect(providerIcon.exists()).toBe(true)
    })

    it('should render Google provider icon when provider is Google', () => {
      const wrapper = mountComponent({
        activeBackup: createDefaultBackup({ provider: 'Google' }),
      })
      const providerIcon = wrapper.find('[data-testid="provider-icon"] [data-name="Google"]')
      expect(providerIcon.exists()).toBe(true)
    })

    it('should use currentProvider when activeBackup is null', () => {
      const wrapper = mountComponent({
        activeBackup: null,
        currentProvider: 'AWS',
      })
      const providerIcon = wrapper.find('[data-testid="provider-icon"] [data-name="AWS"]')
      expect(providerIcon.exists()).toBe(true)
    })
  })

  // ============================================================================
  // Task 8.3: 刷新按鈕測試
  // ============================================================================

  describe('Task 8.3: 刷新按鈕', () => {
    it('should show spinning icon when isRefreshing is true', () => {
      const wrapper = mountComponent({ isRefreshing: true })
      // 刷新按鈕內應該有 RefreshCw 圖標
      const refreshBtn = wrapper.find('[data-testid="refresh-btn"]')
      expect(refreshBtn.find('[data-name="RefreshCw"]').exists()).toBe(true)
      // 按鈕應該有 cursor-wait 類別
      expect(refreshBtn.classes()).toContain('cursor-wait')
    })

    it('should show countdown when cooldownSeconds > 0', () => {
      const wrapper = mountComponent({ cooldownSeconds: 30 })
      expect(wrapper.text()).toContain('30')
    })

    it('should disable refresh button when cooldownSeconds > 0', () => {
      const wrapper = mountComponent({ cooldownSeconds: 30 })
      const refreshBtn = wrapper.find('[data-testid="refresh-btn"]')
      expect(refreshBtn.attributes('disabled')).toBeDefined()
    })

    it('should not emit refresh event when button is disabled', async () => {
      const wrapper = mountComponent({ cooldownSeconds: 30 })
      const refreshBtn = wrapper.find('[data-testid="refresh-btn"]')
      await refreshBtn.trigger('click')
      expect(wrapper.emitted('refresh')).toBeFalsy()
    })
  })

  // ============================================================================
  // Task 8.4: 操作按鈕測試
  // ============================================================================

  describe('Task 8.4: 操作按鈕', () => {
    it('should render create backup button', () => {
      const wrapper = mountComponent()
      const createBtn = wrapper.find('[data-testid="create-backup-btn"]')
      expect(createBtn.exists()).toBe(true)
      expect(wrapper.text()).toContain('建立備份')
    })

    it('should render restore original button', () => {
      const wrapper = mountComponent()
      const restoreBtn = wrapper.find('[data-testid="restore-original-btn"]')
      expect(restoreBtn.exists()).toBe(true)
      expect(wrapper.text()).toContain('還原原始機器')
    })

    it('should show loading state when isRestoring is true', () => {
      const wrapper = mountComponent({ isRestoring: true })
      const restoreBtn = wrapper.find('[data-testid="restore-original-btn"]')
      // 應該顯示 Rotate 圖標
      expect(restoreBtn.find('[data-name="Rotate"]').exists()).toBe(true)
      // 按鈕應該有 cursor-wait 類別
      expect(restoreBtn.classes()).toContain('cursor-wait')
      // 應該顯示處理中文字
      expect(wrapper.text()).toContain('處理中')
    })

    it('should disable restore button when isRestoring is true', () => {
      const wrapper = mountComponent({ isRestoring: true })
      const restoreBtn = wrapper.find('[data-testid="restore-original-btn"]')
      expect(restoreBtn.attributes('disabled')).toBeDefined()
    })
  })

  // ============================================================================
  // 邊界情況測試
  // ============================================================================

  describe('Edge Cases', () => {
    it('should handle null usageInfo gracefully', () => {
      const wrapper = mountComponent({ usageInfo: null })
      expect(wrapper.exists()).toBe(true)
      // 不應該顯示訂閱類型和餘額
      expect(wrapper.find('[data-testid="subscription-badge"]').exists()).toBe(false)
    })

    it('should handle empty currentMachineId', () => {
      const wrapper = mountComponent({ currentMachineId: '' })
      expect(wrapper.text()).toContain('-')
    })
  })
})
