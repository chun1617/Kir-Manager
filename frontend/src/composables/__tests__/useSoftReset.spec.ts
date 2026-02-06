/**
 * useSoftReset Composable 測試
 * @description Property-Based Testing for 軟重置功能
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fc from 'fast-check'
import { useSoftReset } from '../useSoftReset'
import {
  softResetStatusArbitrary,
  hexStringArbitrary,
  resultArbitrary,
} from './arbitraries'
import type { SoftResetStatus, Result } from '@/types/backup'

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key]
    }),
    clear: vi.fn(() => {
      store = {}
    }),
  }
})()

// Mock window.go.main.App
const createMockApp = () => ({
  GetSoftResetStatus: vi.fn(),
  SoftResetToNewMachine: vi.fn(),
  RestoreSoftReset: vi.fn(),
  RegenerateMachineID: vi.fn(),
  RepatchExtension: vi.fn(),
  OpenExtensionFolder: vi.fn(),
  OpenMachineIDFolder: vi.fn(),
  OpenSSOCacheFolder: vi.fn(),
})

let mockApp = createMockApp()

// Setup global mock
beforeEach(() => {
  mockApp = createMockApp()
  localStorageMock.clear()
  vi.stubGlobal('window', {
    go: {
      main: {
        App: mockApp,
      },
    },
  })
  vi.stubGlobal('localStorage', localStorageMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useSoftReset', () => {
  describe('初始狀態', () => {
    it('應該有正確的初始狀態', () => {
      const {
        softResetStatus,
        resetting,
        restoringOriginal,
        patching,
        hasUsedReset,
        showFirstTimeResetModal,
      } = useSoftReset()

      expect(softResetStatus.value).toEqual({
        isPatched: false,
        hasCustomId: false,
        customMachineId: '',
        extensionPath: '',
        isSupported: true,
      })
      expect(resetting.value).toBe(false)
      expect(restoringOriginal.value).toBe(false)
      expect(patching.value).toBe(false)
      expect(hasUsedReset.value).toBe(false)
      expect(showFirstTimeResetModal.value).toBe(false)
    })
  })

  describe('Property 20: 軟重置機器碼變更', () => {
    it('執行軟重置後應該調用 SoftResetToNewMachine API', async () => {
      // Arrange
      mockApp.SoftResetToNewMachine.mockResolvedValue({ success: true, message: '重置成功' })
      localStorageMock.setItem('kiro-manager-has-used-reset', 'true')

      const { executeReset, hasUsedReset } = useSoftReset()
      hasUsedReset.value = true

      // Act
      const result = await executeReset()

      // Assert
      expect(mockApp.SoftResetToNewMachine).toHaveBeenCalled()
      expect(result.success).toBe(true)
    })

    it('軟重置成功後應該設置 hasUsedReset 為 true 並保存到 localStorage', async () => {
      // Arrange
      mockApp.SoftResetToNewMachine.mockResolvedValue({ success: true, message: '重置成功' })

      const { executeReset, hasUsedReset } = useSoftReset()

      // Act
      await executeReset()

      // Assert
      expect(hasUsedReset.value).toBe(true)
      expect(localStorageMock.setItem).toHaveBeenCalledWith('kiro-manager-has-used-reset', 'true')
    })

    it('軟重置失敗時不應該更新 hasUsedReset', async () => {
      // Arrange
      mockApp.SoftResetToNewMachine.mockResolvedValue({ success: false, message: '重置失敗' })

      const { executeReset, hasUsedReset } = useSoftReset()
      hasUsedReset.value = false

      // Act
      await executeReset()

      // Assert
      expect(hasUsedReset.value).toBe(false)
    })

    it('首次使用軟重置時應該顯示首次使用對話框', async () => {
      // Arrange
      const { resetToNew, hasUsedReset, showFirstTimeResetModal } = useSoftReset()
      hasUsedReset.value = false

      // Act
      await resetToNew()

      // Assert
      expect(showFirstTimeResetModal.value).toBe(true)
      expect(mockApp.SoftResetToNewMachine).not.toHaveBeenCalled()
    })

    it('resetting 狀態應該在執行期間為 true', async () => {
      // Arrange
      let resolveSoftReset: (value: Result) => void
      mockApp.SoftResetToNewMachine.mockImplementation(() => {
        return new Promise(resolve => {
          resolveSoftReset = resolve
        })
      })

      const { executeReset, resetting } = useSoftReset()

      // Act
      const resetPromise = executeReset()

      // Assert - 執行期間
      expect(resetting.value).toBe(true)

      // 完成重置
      resolveSoftReset!({ success: true, message: '' })
      await resetPromise

      // Assert - 完成後
      expect(resetting.value).toBe(false)
    })
  })

  describe('Property 21: 還原原始機器狀態', () => {
    it('執行還原後應該調用 RestoreSoftReset API', async () => {
      // Arrange
      mockApp.RestoreSoftReset.mockResolvedValue({ success: true, message: '還原成功' })

      const { restoreOriginal } = useSoftReset()

      // Act
      const result = await restoreOriginal()

      // Assert
      expect(mockApp.RestoreSoftReset).toHaveBeenCalled()
      expect(result.success).toBe(true)
    })

    it('restoringOriginal 狀態應該在執行期間為 true', async () => {
      // Arrange
      let resolveRestore: (value: Result) => void
      mockApp.RestoreSoftReset.mockImplementation(() => {
        return new Promise(resolve => {
          resolveRestore = resolve
        })
      })

      const { restoreOriginal, restoringOriginal } = useSoftReset()

      // Act
      const restorePromise = restoreOriginal()

      // Assert - 執行期間
      expect(restoringOriginal.value).toBe(true)

      // 完成還原
      resolveRestore!({ success: true, message: '' })
      await restorePromise

      // Assert - 完成後
      expect(restoringOriginal.value).toBe(false)
    })

    it('還原失敗時應該返回失敗結果', async () => {
      // Arrange
      mockApp.RestoreSoftReset.mockResolvedValue({ success: false, message: '還原失敗' })

      const { restoreOriginal } = useSoftReset()

      // Act
      const result = await restoreOriginal()

      // Assert
      expect(result.success).toBe(false)
      expect(result.message).toBe('還原失敗')
    })
  })

  describe('Property 22: 備份機器碼重新生成', () => {
    it('重新生成機器碼應該調用 RegenerateMachineID API', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.string({ minLength: 1, maxLength: 50 }),
          async (backupName) => {
            // Arrange
            mockApp.RegenerateMachineID.mockResolvedValue({ success: true, message: '重新生成成功' })

            const { regenerateMachineID } = useSoftReset()

            // Act
            const result = await regenerateMachineID(backupName)

            // Assert
            expect(mockApp.RegenerateMachineID).toHaveBeenCalledWith(backupName)
            expect(result.success).toBe(true)
          }
        ),
        { numRuns: 10 }
      )
    })

    it('重新生成機器碼失敗時應該返回失敗結果', async () => {
      // Arrange
      mockApp.RegenerateMachineID.mockResolvedValue({ success: false, message: '重新生成失敗' })

      const { regenerateMachineID } = useSoftReset()

      // Act
      const result = await regenerateMachineID('test-backup')

      // Assert
      expect(result.success).toBe(false)
      expect(result.message).toBe('重新生成失敗')
    })
  })

  describe('Property 23: Extension Patch 狀態', () => {
    it('執行 Patch 後應該調用 RepatchExtension API', async () => {
      // Arrange
      const newStatus: SoftResetStatus = {
        isPatched: true,
        hasCustomId: true,
        customMachineId: 'abc123',
        extensionPath: '/path/to/extension',
        isSupported: true,
      }
      mockApp.RepatchExtension.mockResolvedValue({ success: true, message: 'Patch 成功' })
      mockApp.GetSoftResetStatus.mockResolvedValue(newStatus)

      const { patchExtension, softResetStatus } = useSoftReset()

      // Act
      const result = await patchExtension()

      // Assert
      expect(mockApp.RepatchExtension).toHaveBeenCalled()
      expect(result.success).toBe(true)
      expect(softResetStatus.value).toEqual(newStatus)
    })

    it('patching 狀態應該在執行期間為 true', async () => {
      // Arrange
      let resolvePatch: (value: Result) => void
      mockApp.RepatchExtension.mockImplementation(() => {
        return new Promise(resolve => {
          resolvePatch = resolve
        })
      })
      mockApp.GetSoftResetStatus.mockResolvedValue({
        isPatched: true,
        hasCustomId: false,
        customMachineId: '',
        extensionPath: '',
        isSupported: true,
      })

      const { patchExtension, patching } = useSoftReset()

      // Act
      const patchPromise = patchExtension()

      // Assert - 執行期間
      expect(patching.value).toBe(true)

      // 完成 Patch
      resolvePatch!({ success: true, message: '' })
      await patchPromise

      // Assert - 完成後
      expect(patching.value).toBe(false)
    })

    it('Patch 成功後應該更新 softResetStatus', async () => {
      await fc.assert(
        fc.asyncProperty(
          softResetStatusArbitrary,
          async (newStatus) => {
            // Arrange
            mockApp.RepatchExtension.mockResolvedValue({ success: true, message: '' })
            mockApp.GetSoftResetStatus.mockResolvedValue(newStatus)

            const { patchExtension, softResetStatus } = useSoftReset()

            // Act
            await patchExtension()

            // Assert
            expect(softResetStatus.value).toEqual(newStatus)
          }
        ),
        { numRuns: 10 }
      )
    })

    it('Patch 失敗時不應該更新 softResetStatus', async () => {
      // Arrange
      mockApp.RepatchExtension.mockResolvedValue({ success: false, message: 'Patch 失敗' })

      const { patchExtension, softResetStatus } = useSoftReset()
      const initialStatus = { ...softResetStatus.value }

      // Act
      await patchExtension()

      // Assert
      expect(softResetStatus.value).toEqual(initialStatus)
      expect(mockApp.GetSoftResetStatus).not.toHaveBeenCalled()
    })
  })

  describe('getSoftResetStatus', () => {
    it('應該載入並更新 softResetStatus', async () => {
      await fc.assert(
        fc.asyncProperty(
          softResetStatusArbitrary,
          async (status) => {
            // Arrange
            mockApp.GetSoftResetStatus.mockResolvedValue(status)

            const { getSoftResetStatus, softResetStatus } = useSoftReset()

            // Act
            await getSoftResetStatus()

            // Assert
            expect(softResetStatus.value).toEqual(status)
          }
        ),
        { numRuns: 10 }
      )
    })
  })

  describe('loadHasUsedReset', () => {
    it('應該從 localStorage 載入 hasUsedReset 狀態', () => {
      // Arrange
      localStorageMock.getItem.mockReturnValue('true')

      const { loadHasUsedReset, hasUsedReset } = useSoftReset()

      // Act
      loadHasUsedReset()

      // Assert
      expect(hasUsedReset.value).toBe(true)
      expect(localStorageMock.getItem).toHaveBeenCalledWith('kiro-manager-has-used-reset')
    })

    it('localStorage 沒有值時 hasUsedReset 應該為 false', () => {
      // Arrange
      localStorageMock.getItem.mockReturnValue(null)

      const { loadHasUsedReset, hasUsedReset } = useSoftReset()

      // Act
      loadHasUsedReset()

      // Assert
      expect(hasUsedReset.value).toBe(false)
    })
  })

  describe('confirmFirstTimeReset', () => {
    it('應該關閉首次使用對話框', async () => {
      // Arrange
      mockApp.SoftResetToNewMachine.mockResolvedValue({ success: true, message: '' })

      const { confirmFirstTimeReset, showFirstTimeResetModal } = useSoftReset()
      showFirstTimeResetModal.value = true

      // Act
      await confirmFirstTimeReset()

      // Assert
      expect(showFirstTimeResetModal.value).toBe(false)
    })

    it('確認後應該執行重置', async () => {
      // Arrange
      mockApp.SoftResetToNewMachine.mockResolvedValue({ success: true, message: '' })

      const { confirmFirstTimeReset, hasUsedReset } = useSoftReset()

      // Act
      await confirmFirstTimeReset()

      // Assert
      expect(mockApp.SoftResetToNewMachine).toHaveBeenCalled()
      expect(hasUsedReset.value).toBe(true)
    })
  })

  describe('工具方法', () => {
    it('openExtensionFolder 應該調用 OpenExtensionFolder API', async () => {
      // Arrange
      mockApp.OpenExtensionFolder.mockResolvedValue(undefined)

      const { openExtensionFolder } = useSoftReset()

      // Act
      await openExtensionFolder()

      // Assert
      expect(mockApp.OpenExtensionFolder).toHaveBeenCalled()
    })

    it('openMachineIDFolder 應該調用 OpenMachineIDFolder API', async () => {
      // Arrange
      mockApp.OpenMachineIDFolder.mockResolvedValue(undefined)

      const { openMachineIDFolder } = useSoftReset()

      // Act
      await openMachineIDFolder()

      // Assert
      expect(mockApp.OpenMachineIDFolder).toHaveBeenCalled()
    })

    it('openSSOCacheFolder 應該調用 OpenSSOCacheFolder API', async () => {
      // Arrange
      mockApp.OpenSSOCacheFolder.mockResolvedValue(undefined)

      const { openSSOCacheFolder } = useSoftReset()

      // Act
      await openSSOCacheFolder()

      // Assert
      expect(mockApp.OpenSSOCacheFolder).toHaveBeenCalled()
    })
  })

  describe('錯誤處理', () => {
    it('executeReset 發生異常時應該正確處理', async () => {
      // Arrange
      mockApp.SoftResetToNewMachine.mockRejectedValue(new Error('網路錯誤'))

      const { executeReset, resetting } = useSoftReset()

      // Act
      const result = await executeReset()

      // Assert
      expect(result.success).toBe(false)
      expect(resetting.value).toBe(false)
    })

    it('restoreOriginal 發生異常時應該正確處理', async () => {
      // Arrange
      mockApp.RestoreSoftReset.mockRejectedValue(new Error('網路錯誤'))

      const { restoreOriginal, restoringOriginal } = useSoftReset()

      // Act
      const result = await restoreOriginal()

      // Assert
      expect(result.success).toBe(false)
      expect(restoringOriginal.value).toBe(false)
    })

    it('patchExtension 發生異常時應該正確處理', async () => {
      // Arrange
      mockApp.RepatchExtension.mockRejectedValue(new Error('網路錯誤'))

      const { patchExtension, patching } = useSoftReset()

      // Act
      const result = await patchExtension()

      // Assert
      expect(result.success).toBe(false)
      expect(patching.value).toBe(false)
    })
  })
})


// ============================================================================
// Task 1.3: patchExtension 超時保護測試
// ============================================================================

describe('Task 1.3: patchExtension 超時保護', () => {
  it('patchExtension 應該使用 withTimeout 包裝 API 調用', async () => {
    // Arrange
    vi.useFakeTimers()
    
    mockApp.RepatchExtension.mockImplementation(() => {
      return new Promise(() => {
        // 永不 resolve，模擬超時
      })
    })
    
    const { patchExtension, patching } = useSoftReset()
    
    // Act
    const patchPromise = patchExtension()
    
    expect(patching.value).toBe(true)
    
    // 快進 30 秒（超時時間）
    await vi.advanceTimersByTimeAsync(30000)
    
    const result = await patchPromise
    
    // Assert
    expect(result.success).toBe(false)
    expect(result.message).toContain('超時')
    expect(patching.value).toBe(false)
    
    vi.useRealTimers()
  })

  it('patchExtension 成功時應該正常返回結果', async () => {
    // Arrange
    mockApp.RepatchExtension.mockResolvedValue({ success: true, message: 'Patch 成功' })
    mockApp.GetSoftResetStatus.mockResolvedValue({
      isPatched: true,
      hasCustomId: false,
      customMachineId: '',
      extensionPath: '',
      isSupported: true,
    })
    
    const { patchExtension, patching } = useSoftReset()
    
    // Act
    const result = await patchExtension()
    
    // Assert
    expect(result.success).toBe(true)
    expect(patching.value).toBe(false)
  })

  it('patchExtension 超時後 finally 區塊應該重置 patching 狀態', async () => {
    // Arrange
    vi.useFakeTimers()
    
    mockApp.RepatchExtension.mockImplementation(() => {
      return new Promise(() => {})
    })
    
    const { patchExtension, patching } = useSoftReset()
    
    // Act
    const promise = patchExtension()
    expect(patching.value).toBe(true)
    
    await vi.advanceTimersByTimeAsync(31000)
    await promise
    
    // Assert
    expect(patching.value).toBe(false)
    
    vi.useRealTimers()
  })
})
