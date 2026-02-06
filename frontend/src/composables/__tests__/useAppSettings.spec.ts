import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fc from 'fast-check'
import { useAppSettings } from '../useAppSettings'
import type { AppSettings, Result } from '@/types/backup'

// Mock window.go.main.App
const mockApp = {
  GetSettings: vi.fn(),
  SaveSettings: vi.fn(),
  GetDetectedKiroVersion: vi.fn(),
  GetDetectedKiroInstallPath: vi.fn(),
}

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}

// Mock useI18n
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'zh-TW' },
  }),
}))

describe('Feature: app-vue-decoupling, useAppSettings composable', () => {
  beforeEach(() => {
    // Setup window.go mock
    ;(window as any).go = {
      main: {
        App: mockApp,
      },
    }
    // Setup localStorage mock
    Object.defineProperty(window, 'localStorage', {
      value: localStorageMock,
      writable: true,
    })
    // Reset all mocks
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Property 24: 版本號保存與自動偵測互斥', () => {
    it('saveKiroVersion 應關閉 useAutoDetect', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.string({ minLength: 1, maxLength: 20 }).filter(s => s.trim().length > 0),
          async (version) => {
            // 每次迭代前重置 mock
            mockApp.GetSettings.mockReset()
            mockApp.SaveSettings.mockReset()
            
            // Arrange
            const defaultSettings: AppSettings = {
              lowBalanceThreshold: 0.2,
              kiroVersion: '0.8.206',
              useAutoDetect: true,
              customKiroInstallPath: '',
            }
            mockApp.GetSettings.mockResolvedValue(defaultSettings)
            mockApp.SaveSettings.mockResolvedValue({ success: true, message: '' })

            const {
              appSettings,
              kiroVersionInput,
              loadSettings,
              saveKiroVersion,
            } = useAppSettings()

            await loadSettings()

            // Act
            kiroVersionInput.value = version
            await saveKiroVersion()

            // Assert: SaveSettings 應被調用，且 useAutoDetect 為 false
            // 注意：實作中會對版本號進行 trim()
            const expectedVersion = version.trim()
            expect(mockApp.SaveSettings).toHaveBeenCalledWith(
              expect.objectContaining({
                kiroVersion: expectedVersion,
                useAutoDetect: false,
              })
            )
            return true
          }
        ),
        { numRuns: 50 }
      )
    })

    it('detectKiroVersion 應啟用 useAutoDetect', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.string({ minLength: 1, maxLength: 20 }).filter(s => s.trim().length > 0),
          async (detectedVersion) => {
            // 每次迭代前重置 mock
            mockApp.GetSettings.mockReset()
            mockApp.GetDetectedKiroVersion.mockReset()
            mockApp.SaveSettings.mockReset()
            
            // Arrange
            const defaultSettings: AppSettings = {
              lowBalanceThreshold: 0.2,
              kiroVersion: '0.8.206',
              useAutoDetect: false,
              customKiroInstallPath: '',
            }
            mockApp.GetSettings.mockResolvedValue(defaultSettings)
            mockApp.GetDetectedKiroVersion.mockResolvedValue({
              success: true,
              message: detectedVersion,
            })
            mockApp.SaveSettings.mockResolvedValue({ success: true, message: '' })

            const {
              appSettings,
              loadSettings,
              detectKiroVersion,
            } = useAppSettings()

            await loadSettings()

            // Act
            await detectKiroVersion()

            // Assert: SaveSettings 應被調用，且 useAutoDetect 為 true
            expect(mockApp.SaveSettings).toHaveBeenCalledWith(
              expect.objectContaining({
                kiroVersion: detectedVersion,
                useAutoDetect: true,
              })
            )
            return true
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  describe('Property 25: 安裝路徑清除', () => {
    it('clearKiroInstallPath 應將 customKiroInstallPath 設為空字串', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.string({ minLength: 1, maxLength: 100 }).filter(s => s.trim().length > 0),
          async (initialPath) => {
            // 每次迭代前重置 mock
            mockApp.GetSettings.mockReset()
            mockApp.SaveSettings.mockReset()
            
            // Arrange
            const defaultSettings: AppSettings = {
              lowBalanceThreshold: 0.2,
              kiroVersion: '0.8.206',
              useAutoDetect: true,
              customKiroInstallPath: initialPath,
            }
            mockApp.GetSettings.mockResolvedValue(defaultSettings)
            mockApp.SaveSettings.mockResolvedValue({ success: true, message: '' })

            const {
              appSettings,
              kiroInstallPathInput,
              loadSettings,
              clearKiroInstallPath,
            } = useAppSettings()

            await loadSettings()
            expect(appSettings.value.customKiroInstallPath).toBe(initialPath)

            // Act
            await clearKiroInstallPath()

            // Assert
            expect(mockApp.SaveSettings).toHaveBeenCalledWith(
              expect.objectContaining({
                customKiroInstallPath: '',
              })
            )
            expect(appSettings.value.customKiroInstallPath).toBe('')
            expect(kiroInstallPathInput.value).toBe('')
            return true
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  describe('Property 26: 低餘額閾值本地更新', () => {
    it('saveLowBalanceThreshold 應更新本地 isLowBalance 狀態', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.float({ min: Math.fround(0.01), max: Math.fround(0.99), noNaN: true }),
          fc.float({ min: Math.fround(0.01), max: Math.fround(0.99), noNaN: true }),
          fc.integer({ min: 100, max: 1000 }),
          async (oldThreshold, newThreshold, usageLimit) => {
            // 每次迭代前重置 mock
            mockApp.GetSettings.mockReset()
            mockApp.SaveSettings.mockReset()
            
            // Arrange
            const balance = usageLimit * 0.5 // 50% 餘額
            const defaultSettings: AppSettings = {
              lowBalanceThreshold: oldThreshold,
              kiroVersion: '0.8.206',
              useAutoDetect: true,
              customKiroInstallPath: '',
            }
            mockApp.GetSettings.mockResolvedValue(defaultSettings)
            mockApp.SaveSettings.mockResolvedValue({ success: true, message: '' })

            const {
              appSettings,
              loadSettings,
              saveLowBalanceThreshold,
            } = useAppSettings()

            await loadSettings()

            // 模擬 backups 和 currentUsageInfo（這些會由外部傳入的回調處理）
            let localIsLowBalanceUpdated = false
            const mockUpdateCallback = (threshold: number) => {
              localIsLowBalanceUpdated = true
            }

            // Act
            await saveLowBalanceThreshold(newThreshold, mockUpdateCallback)

            // Assert
            expect(mockApp.SaveSettings).toHaveBeenCalledWith(
              expect.objectContaining({
                lowBalanceThreshold: newThreshold,
              })
            )
            expect(appSettings.value.lowBalanceThreshold).toBe(newThreshold)
            expect(localIsLowBalanceUpdated).toBe(true)
            return true
          }
        ),
        { numRuns: 50 }
      )
    })
  })

  describe('Property 27: 語言切換持久化', () => {
    it('switchLanguage 應將語言設定存入 localStorage', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.constantFrom('zh-TW', 'zh-CN', 'en'),
          async (lang) => {
            // Arrange
            const { switchLanguage } = useAppSettings()

            // Act
            switchLanguage(lang)

            // Assert
            expect(localStorageMock.setItem).toHaveBeenCalledWith(
              'kiro-manager-lang',
              lang
            )
            return true
          }
        ),
        { numRuns: 10 }
      )
    })
  })

  describe('Unit Tests', () => {
    it('should initialize with default values', async () => {
      const defaultSettings: AppSettings = {
        lowBalanceThreshold: 0.2,
        kiroVersion: '0.8.206',
        useAutoDetect: true,
        customKiroInstallPath: '',
      }
      mockApp.GetSettings.mockResolvedValue(defaultSettings)

      const {
        appSettings,
        kiroVersionInput,
        kiroVersionModified,
        kiroInstallPathInput,
        kiroInstallPathModified,
        thresholdPreview,
        detectingVersion,
        detectingPath,
        loadSettings,
      } = useAppSettings()

      await loadSettings()

      expect(appSettings.value).toEqual(defaultSettings)
      expect(kiroVersionInput.value).toBe('0.8.206')
      expect(kiroVersionModified.value).toBe(false)
      expect(kiroInstallPathInput.value).toBe('')
      expect(kiroInstallPathModified.value).toBe(false)
      expect(thresholdPreview.value).toBe(20)
      expect(detectingVersion.value).toBe(false)
      expect(detectingPath.value).toBe(false)
    })

    it('onKiroVersionInput should set kiroVersionModified to true', () => {
      const { kiroVersionModified, onKiroVersionInput } = useAppSettings()

      expect(kiroVersionModified.value).toBe(false)
      onKiroVersionInput()
      expect(kiroVersionModified.value).toBe(true)
    })

    it('onKiroInstallPathInput should set kiroInstallPathModified to true', () => {
      const { kiroInstallPathModified, onKiroInstallPathInput } = useAppSettings()

      expect(kiroInstallPathModified.value).toBe(false)
      onKiroInstallPathInput()
      expect(kiroInstallPathModified.value).toBe(true)
    })

    it('detectKiroVersion should set detectingVersion during detection', async () => {
      const defaultSettings: AppSettings = {
        lowBalanceThreshold: 0.2,
        kiroVersion: '0.8.206',
        useAutoDetect: false,
        customKiroInstallPath: '',
      }
      mockApp.GetSettings.mockResolvedValue(defaultSettings)
      
      // 使用延遲來測試 loading 狀態
      let resolveDetection: (value: Result) => void
      mockApp.GetDetectedKiroVersion.mockReturnValue(
        new Promise<Result>((resolve) => {
          resolveDetection = resolve
        })
      )
      mockApp.SaveSettings.mockResolvedValue({ success: true, message: '' })

      const { detectingVersion, loadSettings, detectKiroVersion } = useAppSettings()

      await loadSettings()
      expect(detectingVersion.value).toBe(false)

      const detectPromise = detectKiroVersion()
      expect(detectingVersion.value).toBe(true)

      resolveDetection!({ success: true, message: '0.9.0' })
      await detectPromise

      expect(detectingVersion.value).toBe(false)
    })

    it('detectKiroInstallPath should set detectingPath during detection', async () => {
      const defaultSettings: AppSettings = {
        lowBalanceThreshold: 0.2,
        kiroVersion: '0.8.206',
        useAutoDetect: true,
        customKiroInstallPath: '',
      }
      mockApp.GetSettings.mockResolvedValue(defaultSettings)
      
      let resolveDetection: (value: Result) => void
      mockApp.GetDetectedKiroInstallPath.mockReturnValue(
        new Promise<Result>((resolve) => {
          resolveDetection = resolve
        })
      )
      mockApp.SaveSettings.mockResolvedValue({ success: true, message: '' })

      const { detectingPath, loadSettings, detectKiroInstallPath } = useAppSettings()

      await loadSettings()
      expect(detectingPath.value).toBe(false)

      const detectPromise = detectKiroInstallPath()
      expect(detectingPath.value).toBe(true)

      resolveDetection!({ success: true, message: 'C:\\Program Files\\Kiro' })
      await detectPromise

      expect(detectingPath.value).toBe(false)
    })

    it('saveKiroInstallPath should save custom path', async () => {
      const defaultSettings: AppSettings = {
        lowBalanceThreshold: 0.2,
        kiroVersion: '0.8.206',
        useAutoDetect: true,
        customKiroInstallPath: '',
      }
      mockApp.GetSettings.mockResolvedValue(defaultSettings)
      mockApp.SaveSettings.mockResolvedValue({ success: true, message: '' })

      const {
        appSettings,
        kiroInstallPathInput,
        kiroInstallPathModified,
        loadSettings,
        saveKiroInstallPath,
      } = useAppSettings()

      await loadSettings()

      kiroInstallPathInput.value = 'D:\\Custom\\Kiro'
      kiroInstallPathModified.value = true

      await saveKiroInstallPath()

      expect(mockApp.SaveSettings).toHaveBeenCalledWith(
        expect.objectContaining({
          customKiroInstallPath: 'D:\\Custom\\Kiro',
        })
      )
      expect(appSettings.value.customKiroInstallPath).toBe('D:\\Custom\\Kiro')
      expect(kiroInstallPathModified.value).toBe(false)
    })

    it('checkPathDetectionStatus should return correct status', async () => {
      const { checkPathDetectionStatus } = useAppSettings()

      // 有自定義路徑
      expect(checkPathDetectionStatus('C:\\Kiro', true)).toBe('custom')
      
      // 無自定義路徑，使用自動偵測
      expect(checkPathDetectionStatus('', true)).toBe('auto')
      
      // 無自定義路徑，未使用自動偵測
      expect(checkPathDetectionStatus('', false)).toBe('none')
    })
  })
})
