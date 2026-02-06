/**
 * useAutoSwitch Composable 測試
 * @description Property-Based Testing for 自動切換功能
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fc from 'fast-check'
import { useAutoSwitch } from '../useAutoSwitch'
import {
  autoSwitchSettingsArbitrary,
  autoSwitchStatusArbitrary,
  refreshIntervalRuleArbitrary,
} from './arbitraries'
import type { AutoSwitchSettings, AutoSwitchStatus, RefreshIntervalRule } from '@/types/backup'

// Mock window.go.main.App
const createMockApp = () => ({
  GetAutoSwitchSettings: vi.fn(),
  SaveAutoSwitchSettings: vi.fn(),
  StartAutoSwitchMonitor: vi.fn(),
  StopAutoSwitchMonitor: vi.fn(),
  GetAutoSwitchStatus: vi.fn(),
})

let mockApp = createMockApp()

// Setup global mock
beforeEach(() => {
  mockApp = createMockApp()
  vi.stubGlobal('window', {
    go: {
      main: {
        App: mockApp,
      },
    },
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useAutoSwitch', () => {
  describe('初始狀態', () => {
    it('應該有正確的初始狀態', () => {
      const {
        autoSwitchSettings,
        autoSwitchStatus,
        savingAutoSwitch,
      } = useAutoSwitch()

      expect(autoSwitchSettings.value).toEqual({
        enabled: false,
        balanceThreshold: 5,
        minTargetBalance: 50,
        folderIds: [],
        subscriptionTypes: [],
        refreshIntervals: [],
        notifyOnSwitch: true,
        notifyOnLowBalance: true,
      })
      expect(autoSwitchStatus.value).toEqual({
        status: 'stopped',
        lastBalance: 0,
        cooldownRemaining: 0,
        switchCount: 0,
      })
      expect(savingAutoSwitch.value).toBe(false)
    })
  })

  describe('Property 17: 自動切換設定持久化', () => {
    it('保存設定後應該調用 SaveAutoSwitchSettings API', async () => {
      await fc.assert(
        fc.asyncProperty(
          autoSwitchSettingsArbitrary,
          async (settings) => {
            // Arrange
            mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })

            const { autoSwitchSettings, saveAutoSwitchSettings } = useAutoSwitch()
            autoSwitchSettings.value = settings

            // Act
            await saveAutoSwitchSettings()

            // Assert
            expect(mockApp.SaveAutoSwitchSettings).toHaveBeenCalledWith(settings)
          }
        ),
        { numRuns: 20 }
      )
    })

    it('載入設定後應該更新本地狀態', async () => {
      await fc.assert(
        fc.asyncProperty(
          autoSwitchSettingsArbitrary,
          autoSwitchStatusArbitrary,
          async (settings, status) => {
            // Arrange
            mockApp.GetAutoSwitchSettings.mockResolvedValue(settings)
            mockApp.GetAutoSwitchStatus.mockResolvedValue(status)

            const { autoSwitchSettings, autoSwitchStatus, loadAutoSwitchSettings } = useAutoSwitch()

            // Act
            await loadAutoSwitchSettings()

            // Assert
            expect(autoSwitchSettings.value).toEqual(settings)
            expect(autoSwitchStatus.value).toEqual(status)
          }
        ),
        { numRuns: 20 }
      )
    })
  })

  describe('Property 18: 自動切換狀態一致性', () => {
    it('啟用自動切換應該調用 StartAutoSwitchMonitor', async () => {
      // Arrange
      mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })
      mockApp.StartAutoSwitchMonitor.mockResolvedValue({ success: true, message: '' })
      mockApp.GetAutoSwitchStatus.mockResolvedValue({
        status: 'running',
        lastBalance: 0,
        cooldownRemaining: 0,
        switchCount: 0,
      })

      const { autoSwitchSettings, toggleAutoSwitch } = useAutoSwitch()
      autoSwitchSettings.value.enabled = true

      // Act
      await toggleAutoSwitch()

      // Assert
      expect(mockApp.StartAutoSwitchMonitor).toHaveBeenCalled()
    })

    it('停用自動切換應該調用 StopAutoSwitchMonitor', async () => {
      // Arrange
      mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })
      mockApp.StopAutoSwitchMonitor.mockResolvedValue({ success: true, message: '' })
      mockApp.GetAutoSwitchStatus.mockResolvedValue({
        status: 'stopped',
        lastBalance: 0,
        cooldownRemaining: 0,
        switchCount: 0,
      })

      const { autoSwitchSettings, toggleAutoSwitch } = useAutoSwitch()
      autoSwitchSettings.value.enabled = false

      // Act
      await toggleAutoSwitch()

      // Assert
      expect(mockApp.StopAutoSwitchMonitor).toHaveBeenCalled()
    })

    it('啟動監控失敗時應該回滾 enabled 狀態', async () => {
      // Arrange
      mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })
      mockApp.StartAutoSwitchMonitor.mockResolvedValue({ success: false, message: '啟動失敗' })
      mockApp.GetAutoSwitchStatus.mockResolvedValue({
        status: 'stopped',
        lastBalance: 0,
        cooldownRemaining: 0,
        switchCount: 0,
      })

      const { autoSwitchSettings, toggleAutoSwitch } = useAutoSwitch()
      autoSwitchSettings.value.enabled = true

      // Act
      const result = await toggleAutoSwitch()

      // Assert
      expect(autoSwitchSettings.value.enabled).toBe(false)
      expect(result.success).toBe(false)
    })
  })

  describe('Property 19: 刷新規則列表長度不變量', () => {
    it('添加規則後列表長度增加 1', () => {
      fc.assert(
        fc.property(
          fc.array(refreshIntervalRuleArbitrary, { minLength: 0, maxLength: 5 }),
          (initialRules) => {
            const { autoSwitchSettings, addRefreshRule } = useAutoSwitch()
            autoSwitchSettings.value.refreshIntervals = [...initialRules]

            const initialLength = autoSwitchSettings.value.refreshIntervals.length

            // Act
            addRefreshRule()

            // Assert
            expect(autoSwitchSettings.value.refreshIntervals.length).toBe(initialLength + 1)
          }
        ),
        { numRuns: 30 }
      )
    })

    it('移除規則後列表長度減少 1', async () => {
      // Arrange
      mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })

      const { autoSwitchSettings, removeRefreshRule } = useAutoSwitch()
      autoSwitchSettings.value.refreshIntervals = [
        { minBalance: 0, maxBalance: 100, interval: 30 },
        { minBalance: 100, maxBalance: 500, interval: 60 },
        { minBalance: 500, maxBalance: -1, interval: 120 },
      ]

      const initialLength = autoSwitchSettings.value.refreshIntervals.length

      // Act
      await removeRefreshRule(1)

      // Assert
      expect(autoSwitchSettings.value.refreshIntervals.length).toBe(initialLength - 1)
    })
  })

  describe('文件夾篩選方法', () => {
    it('addAutoSwitchFolder 應該添加文件夾 ID', async () => {
      // Arrange
      mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })

      const { autoSwitchSettings, addAutoSwitchFolder } = useAutoSwitch()
      autoSwitchSettings.value.folderIds = ['folder-1']

      // Act
      await addAutoSwitchFolder('folder-2')

      // Assert
      expect(autoSwitchSettings.value.folderIds).toContain('folder-2')
      expect(mockApp.SaveAutoSwitchSettings).toHaveBeenCalled()
    })

    it('addAutoSwitchFolder 不應該添加重複的文件夾 ID', async () => {
      // Arrange
      const { autoSwitchSettings, addAutoSwitchFolder } = useAutoSwitch()
      autoSwitchSettings.value.folderIds = ['folder-1']

      // Act
      await addAutoSwitchFolder('folder-1')

      // Assert
      expect(autoSwitchSettings.value.folderIds).toEqual(['folder-1'])
      expect(mockApp.SaveAutoSwitchSettings).not.toHaveBeenCalled()
    })

    it('removeAutoSwitchFolder 應該移除文件夾 ID', async () => {
      // Arrange
      mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })

      const { autoSwitchSettings, removeAutoSwitchFolder } = useAutoSwitch()
      autoSwitchSettings.value.folderIds = ['folder-1', 'folder-2']

      // Act
      await removeAutoSwitchFolder('folder-1')

      // Assert
      expect(autoSwitchSettings.value.folderIds).toEqual(['folder-2'])
      expect(mockApp.SaveAutoSwitchSettings).toHaveBeenCalled()
    })
  })

  describe('訂閱類型篩選方法', () => {
    it('addAutoSwitchSubscription 應該添加訂閱類型', async () => {
      // Arrange
      mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })

      const { autoSwitchSettings, addAutoSwitchSubscription } = useAutoSwitch()
      autoSwitchSettings.value.subscriptionTypes = ['Free']

      // Act
      await addAutoSwitchSubscription('Pro')

      // Assert
      expect(autoSwitchSettings.value.subscriptionTypes).toContain('Pro')
      expect(mockApp.SaveAutoSwitchSettings).toHaveBeenCalled()
    })

    it('addAutoSwitchSubscription 不應該添加重複的訂閱類型', async () => {
      // Arrange
      const { autoSwitchSettings, addAutoSwitchSubscription } = useAutoSwitch()
      autoSwitchSettings.value.subscriptionTypes = ['Free']

      // Act
      await addAutoSwitchSubscription('Free')

      // Assert
      expect(autoSwitchSettings.value.subscriptionTypes).toEqual(['Free'])
      expect(mockApp.SaveAutoSwitchSettings).not.toHaveBeenCalled()
    })

    it('removeAutoSwitchSubscription 應該移除訂閱類型', async () => {
      // Arrange
      mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })

      const { autoSwitchSettings, removeAutoSwitchSubscription } = useAutoSwitch()
      autoSwitchSettings.value.subscriptionTypes = ['Free', 'Pro']

      // Act
      await removeAutoSwitchSubscription('Free')

      // Assert
      expect(autoSwitchSettings.value.subscriptionTypes).toEqual(['Pro'])
      expect(mockApp.SaveAutoSwitchSettings).toHaveBeenCalled()
    })
  })

  describe('handleAutoSwitchToggle', () => {
    it('應該更新 enabled 狀態並調用 toggleAutoSwitch', async () => {
      // Arrange
      mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })
      mockApp.StartAutoSwitchMonitor.mockResolvedValue({ success: true, message: '' })
      mockApp.GetAutoSwitchStatus.mockResolvedValue({
        status: 'running',
        lastBalance: 0,
        cooldownRemaining: 0,
        switchCount: 0,
      })

      const { autoSwitchSettings, handleAutoSwitchToggle } = useAutoSwitch()
      expect(autoSwitchSettings.value.enabled).toBe(false)

      // Act
      await handleAutoSwitchToggle(true)

      // Assert
      expect(autoSwitchSettings.value.enabled).toBe(true)
      expect(mockApp.StartAutoSwitchMonitor).toHaveBeenCalled()
    })
  })

  describe('savingAutoSwitch 狀態', () => {
    it('保存期間 savingAutoSwitch 應該為 true', async () => {
      // Arrange
      let resolveSave: (value: any) => void
      mockApp.SaveAutoSwitchSettings.mockImplementation(() => {
        return new Promise(resolve => {
          resolveSave = resolve
        })
      })

      const { savingAutoSwitch, saveAutoSwitchSettings } = useAutoSwitch()

      // Act
      const savePromise = saveAutoSwitchSettings()
      
      // Assert - 保存期間
      expect(savingAutoSwitch.value).toBe(true)

      // 完成保存
      resolveSave!({ success: true, message: '' })
      await savePromise

      // Assert - 保存完成後
      expect(savingAutoSwitch.value).toBe(false)
    })
  })

  describe('P1-FIX: toggleAutoSwitch 並發保護', () => {
    it('快速連續調用 toggleAutoSwitch 應該只執行一次', async () => {
      // Arrange
      let resolveToggle: (value: any) => void
      let callCount = 0
      
      mockApp.SaveAutoSwitchSettings.mockImplementation(() => {
        callCount++
        return new Promise(resolve => {
          resolveToggle = resolve
        })
      })
      mockApp.StartAutoSwitchMonitor.mockResolvedValue({ success: true, message: '' })
      mockApp.GetAutoSwitchStatus.mockResolvedValue({
        status: 'running',
        lastBalance: 0,
        cooldownRemaining: 0,
        switchCount: 0,
      })

      const { autoSwitchSettings, toggleAutoSwitch, isToggling } = useAutoSwitch()
      autoSwitchSettings.value.enabled = true

      // Act - 快速連續調用
      const promise1 = toggleAutoSwitch()
      const promise2 = toggleAutoSwitch()
      const promise3 = toggleAutoSwitch()

      // Assert - 第一次調用應該正在進行
      expect(isToggling.value).toBe(true)

      // 完成第一次調用
      resolveToggle!({ success: true, message: '' })
      await Promise.all([promise1, promise2, promise3])

      // Assert - 只應該調用一次 SaveAutoSwitchSettings
      expect(callCount).toBe(1)
      expect(isToggling.value).toBe(false)
    })

    it('第一次 toggleAutoSwitch 完成後可以再次調用', async () => {
      // Arrange
      mockApp.SaveAutoSwitchSettings.mockResolvedValue({ success: true, message: '' })
      mockApp.StartAutoSwitchMonitor.mockResolvedValue({ success: true, message: '' })
      mockApp.StopAutoSwitchMonitor.mockResolvedValue({ success: true, message: '' })
      mockApp.GetAutoSwitchStatus.mockResolvedValue({
        status: 'running',
        lastBalance: 0,
        cooldownRemaining: 0,
        switchCount: 0,
      })

      const { autoSwitchSettings, toggleAutoSwitch, isToggling } = useAutoSwitch()

      // Act - 第一次調用
      autoSwitchSettings.value.enabled = true
      await toggleAutoSwitch()
      expect(isToggling.value).toBe(false)

      // Act - 第二次調用（應該可以執行）
      mockApp.SaveAutoSwitchSettings.mockClear()
      autoSwitchSettings.value.enabled = false
      await toggleAutoSwitch()

      // Assert - 第二次調用應該成功
      expect(mockApp.SaveAutoSwitchSettings).toHaveBeenCalled()
    })
  })
})
