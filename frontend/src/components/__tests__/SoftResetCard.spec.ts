import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SoftResetCard from '../SoftResetCard.vue'
import type { SoftResetStatus } from '@/types/backup'

// Mock vue-i18n
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        'status.patchStatus': 'PATCH 狀態',
        'status.patched': '已 Patch',
        'status.notPatched': '未 Patch',
        'status.patching': 'Patch 中...',
        'status.clickToPatch': '點擊執行 Patch',
        'status.hasCustomId': '使用自訂 ID',
        'status.noCustomId': '使用系統 ID',
        'status.openFolder': '打開文件夾',
        'status.softResetActive': '一鍵新機已啟用',
        'status.softResetInactive': '一鍵新機未啟用',
        'restore.reset': '一鍵新機',
        'restore.resetDesc': '產生新的機器指紋 ID',
        'restore.original': '還原出廠',
        'app.processing': '處理中...',
        'message.successChange': '切換成功',
      }
      return translations[key] || key
    },
  }),
}))

// Mock Icon component
const IconStub = {
  name: 'Icon',
  template: '<span class="icon-stub" :data-name="name"></span>',
  props: ['name'],
}

// 建立預設的 SoftResetStatus
const createDefaultStatus = (overrides: Partial<SoftResetStatus> = {}): SoftResetStatus => ({
  isPatched: false,
  hasCustomId: false,
  customMachineId: '',
  extensionPath: '/path/to/extension',
  isSupported: true,
  ...overrides,
})

// 建立預設的 props
const createDefaultProps = (overrides = {}) => ({
  softResetStatus: createDefaultStatus(),
  isResetting: false,
  isPatching: false,
  ...overrides,
})

describe('Feature: ui-component-extraction, SoftResetCard Component', () => {
  // ============================================================================
  // Task 10.1: 組件骨架測試
  // ============================================================================

  describe('Task 10.1: 組件骨架', () => {
    it('should render without errors', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.exists()).toBe(true)
    })

    it('should accept softResetStatus prop', () => {
      const status = createDefaultStatus({ isPatched: true })
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({ softResetStatus: status }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.exists()).toBe(true)
    })

    it('should accept isResetting prop', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({ isResetting: true }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.exists()).toBe(true)
    })

    it('should accept isPatching prop', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({ isPatching: true }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.exists()).toBe(true)
    })
  })

  // ============================================================================
  // Task 10.2: 狀態顯示測試
  // ============================================================================

  describe('Task 10.2: 狀態顯示', () => {
    it('should render "已 Patch" label when isPatched is true', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ isPatched: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.find('[data-testid="patched-label"]').exists()).toBe(true)
    })

    it('should render Machine ID status with custom ID', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ hasCustomId: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.find('[data-testid="machine-id-status"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('使用自訂 ID')
    })

    it('should render Machine ID status without custom ID', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ hasCustomId: false }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.find('[data-testid="machine-id-status"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('使用系統 ID')
    })

    it('should render green status indicator when both patched and hasCustomId', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ isPatched: true, hasCustomId: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      const indicator = wrapper.find('[data-testid="status-indicator"]')
      expect(indicator.exists()).toBe(true)
      expect(indicator.classes()).toContain('bg-app-success')
    })

    it('should render gray status indicator when not fully active', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ isPatched: false, hasCustomId: false }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      const indicator = wrapper.find('[data-testid="status-indicator"]')
      expect(indicator.exists()).toBe(true)
      expect(indicator.classes()).toContain('bg-zinc-500')
    })

    it('should render extension folder button when extensionPath exists', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ extensionPath: '/some/path' }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.find('[data-testid="open-extension-folder-btn"]').exists()).toBe(true)
    })

    it('should render machine id folder button', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.find('[data-testid="open-machine-id-folder-btn"]').exists()).toBe(true)
    })
  })

  // ============================================================================
  // Task 10.3: 一鍵新機按鈕測試
  // ============================================================================

  describe('Task 10.3: 一鍵新機按鈕', () => {
    it('should render reset button', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.find('[data-testid="reset-btn"]').exists()).toBe(true)
    })

    it('should render SVG icon in reset button', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      const resetBtn = wrapper.find('[data-testid="reset-btn"]')
      expect(resetBtn.find('svg').exists()).toBe(true)
    })

    it('should show loading animation when isResetting is true', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({ isResetting: true }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      const resetBtn = wrapper.find('[data-testid="reset-btn"]')
      expect(resetBtn.find('animateTransform').exists()).toBe(true)
    })

    it('should emit reset event when reset button is clicked', async () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      await wrapper.find('[data-testid="reset-btn"]').trigger('click')
      expect(wrapper.emitted('reset')).toBeTruthy()
    })

    it('should disable reset button when isResetting is true', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({ isResetting: true }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      const resetBtn = wrapper.find('[data-testid="reset-btn"]')
      expect(resetBtn.attributes('disabled')).toBeDefined()
    })
  })

  // ============================================================================
  // Task 10.4: Patch 按鈕測試
  // ============================================================================

  describe('Task 10.4: Patch 按鈕', () => {
    it('should render patch button when not patched', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ isPatched: false }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.find('[data-testid="patch-btn"]').exists()).toBe(true)
    })

    it('should not render patch button when already patched', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ isPatched: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.find('[data-testid="patch-btn"]').exists()).toBe(false)
    })

    it('should show loading state when isPatching is true', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ isPatched: false }),
          isPatching: true,
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      const patchBtn = wrapper.find('[data-testid="patch-btn"]')
      expect(patchBtn.text()).toContain('Patch 中...')
    })

    it('should emit patch event when patch button is clicked', async () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ isPatched: false }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      await wrapper.find('[data-testid="patch-btn"]').trigger('click')
      expect(wrapper.emitted('patch')).toBeTruthy()
    })

    it('should disable patch button when isPatching is true', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ isPatched: false }),
          isPatching: true,
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      const patchBtn = wrapper.find('[data-testid="patch-btn"]')
      expect(patchBtn.attributes('disabled')).toBeDefined()
    })
  })

  // ============================================================================
  // Task 10.5: 還原按鈕測試
  // ============================================================================

  describe('Task 10.5: 還原按鈕', () => {
    it('should render restore button', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      expect(wrapper.find('[data-testid="restore-btn"]').exists()).toBe(true)
    })

    it('should emit restore event when restore button is clicked', async () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ hasCustomId: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      await wrapper.find('[data-testid="restore-btn"]').trigger('click')
      expect(wrapper.emitted('restore')).toBeTruthy()
    })

    it('should disable restore button when hasCustomId is false', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ hasCustomId: false }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      const restoreBtn = wrapper.find('[data-testid="restore-btn"]')
      expect(restoreBtn.attributes('disabled')).toBeDefined()
    })

    it('should enable restore button when hasCustomId is true', () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ hasCustomId: true }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      const restoreBtn = wrapper.find('[data-testid="restore-btn"]')
      expect(restoreBtn.attributes('disabled')).toBeUndefined()
    })
  })

  // ============================================================================
  // 事件發射測試
  // ============================================================================

  describe('Events', () => {
    it('should emit open-extension-folder event when extension folder button is clicked', async () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps({
          softResetStatus: createDefaultStatus({ extensionPath: '/some/path' }),
        }),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      await wrapper.find('[data-testid="open-extension-folder-btn"]').trigger('click')
      expect(wrapper.emitted('open-extension-folder')).toBeTruthy()
    })

    it('should emit open-machine-id-folder event when machine id folder button is clicked', async () => {
      const wrapper = mount(SoftResetCard, {
        props: createDefaultProps(),
        global: {
          stubs: { Icon: IconStub },
        },
      })
      await wrapper.find('[data-testid="open-machine-id-folder-btn"]').trigger('click')
      expect(wrapper.emitted('open-machine-id-folder')).toBeTruthy()
    })
  })
})
