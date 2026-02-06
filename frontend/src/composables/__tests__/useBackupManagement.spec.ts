/**
 * useBackupManagement Composable 測試
 * @description Property-Based Testing for backup management composable
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fc from 'fast-check'
import { nextTick } from 'vue'
import { useBackupManagement } from '../useBackupManagement'
import {
  backupItemArbitrary,
  backupListArbitrary,
  snapshotNameArbitrary,
  resultArbitrary,
} from './arbitraries'
import type { BackupItem, Result } from '@/types/backup'

// ============================================================================
// Mock Setup
// ============================================================================

// Mock Wails API
const mockGetBackupList = vi.fn<() => Promise<BackupItem[]>>()
const mockCreateBackup = vi.fn<(name: string) => Promise<Result>>()
const mockSwitchToBackup = vi.fn<(name: string) => Promise<Result>>()
const mockDeleteBackup = vi.fn<(name: string) => Promise<Result>>()
const mockRegenerateMachineID = vi.fn<(name: string) => Promise<Result>>()
const mockRefreshBackupUsage = vi.fn<(name: string) => Promise<any>>()
const mockGetCurrentMachineID = vi.fn<() => Promise<string>>()
const mockGetCurrentEnvironmentName = vi.fn<() => Promise<string>>()

// Setup global window.go mock
beforeEach(() => {
  vi.clearAllMocks()
  
  // Default mock implementations
  mockGetBackupList.mockResolvedValue([])
  mockCreateBackup.mockResolvedValue({ success: true, message: '備份成功' })
  mockSwitchToBackup.mockResolvedValue({ success: true, message: '切換成功' })
  mockDeleteBackup.mockResolvedValue({ success: true, message: '刪除成功' })
  mockRegenerateMachineID.mockResolvedValue({ success: true, message: '重新生成成功' })
  mockRefreshBackupUsage.mockResolvedValue({ success: true, message: '刷新成功' })
  mockGetCurrentMachineID.mockResolvedValue('test-machine-id')
  mockGetCurrentEnvironmentName.mockResolvedValue('test-env')
  
  // @ts-ignore - Partial mock for testing
  globalThis.window = {
    go: {
      main: {
        App: {
          GetBackupList: mockGetBackupList,
          CreateBackup: mockCreateBackup,
          SwitchToBackup: mockSwitchToBackup,
          DeleteBackup: mockDeleteBackup,
          RegenerateMachineID: mockRegenerateMachineID,
          RefreshBackupUsage: mockRefreshBackupUsage,
          GetCurrentMachineID: mockGetCurrentMachineID,
          GetCurrentEnvironmentName: mockGetCurrentEnvironmentName,
        },
      },
    },
  } as any
})

afterEach(() => {
  vi.restoreAllMocks()
})


// ============================================================================
// Property 1: 篩選結果正確性
// ============================================================================

describe('Property 1: filteredBackups 篩選結果正確性', () => {
  it('應根據訂閱篩選條件正確過濾', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 20 }),
        fc.constantFrom('FREE', 'PRO', 'PRO+', 'POWER', ''),
        async (backups, filterValue) => {
          mockGetBackupList.mockResolvedValue(backups)
          
          const { filteredBackups, setFilterSubscription, loadBackups } = useBackupManagement()
          await loadBackups()
          
          setFilterSubscription(filterValue)
          await nextTick()
          
          if (!filterValue) {
            // 無篩選時應返回所有備份
            expect(filteredBackups.value.length).toBe(backups.length)
          } else {
            // 有篩選時應只返回匹配的備份
            const subscriptionMap: Record<string, string> = {
              'FREE': 'KIRO FREE',
              'PRO': 'KIRO PRO',
              'PRO+': 'KIRO PRO+',
              'POWER': 'KIRO POWER'
            }
            const target = subscriptionMap[filterValue]
            const expected = backups.filter(b => b.subscriptionTitle?.toUpperCase() === target)
            expect(filteredBackups.value.length).toBe(expected.length)
          }
        }
      ),
      { numRuns: 50 }
    )
  })

  it('應根據提供者篩選條件正確過濾', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 20 }),
        fc.constantFrom('AWS', 'GitHub', ''),
        async (backups, filterValue) => {
          mockGetBackupList.mockResolvedValue(backups)
          
          const { filteredBackups, setFilterProvider, loadBackups } = useBackupManagement()
          await loadBackups()
          
          setFilterProvider(filterValue)
          await nextTick()
          
          if (!filterValue) {
            expect(filteredBackups.value.length).toBe(backups.length)
          } else if (filterValue === 'AWS') {
            // AWS 篩選應包含 AWS 和 BuilderId
            const expected = backups.filter(b => b.provider === 'AWS' || b.provider === 'BuilderId')
            expect(filteredBackups.value.length).toBe(expected.length)
          } else {
            const expected = backups.filter(b => b.provider === filterValue)
            expect(filteredBackups.value.length).toBe(expected.length)
          }
        }
      ),
      { numRuns: 50 }
    )
  })

  it('應根據餘額篩選條件正確過濾', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 20 }),
        fc.constantFrom('LOW', 'NORMAL', 'NO_DATA', ''),
        async (backups, filterValue) => {
          mockGetBackupList.mockResolvedValue(backups)
          
          const { filteredBackups, setFilterBalance, loadBackups } = useBackupManagement()
          await loadBackups()
          
          setFilterBalance(filterValue)
          await nextTick()
          
          if (!filterValue) {
            expect(filteredBackups.value.length).toBe(backups.length)
          } else if (filterValue === 'LOW') {
            const expected = backups.filter(b => b.usageLimit > 0 && b.isLowBalance)
            expect(filteredBackups.value.length).toBe(expected.length)
          } else if (filterValue === 'NORMAL') {
            const expected = backups.filter(b => b.usageLimit > 0 && !b.isLowBalance)
            expect(filteredBackups.value.length).toBe(expected.length)
          } else if (filterValue === 'NO_DATA') {
            const expected = backups.filter(b => b.usageLimit === 0)
            expect(filteredBackups.value.length).toBe(expected.length)
          }
        }
      ),
      { numRuns: 50 }
    )
  })

  it('應根據搜尋關鍵字正確過濾', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 20 }),
        fc.string({ minLength: 0, maxLength: 10 }),
        async (backups, searchQuery) => {
          mockGetBackupList.mockResolvedValue(backups)
          
          const { filteredBackups, searchQuery: query, loadBackups } = useBackupManagement()
          await loadBackups()
          
          query.value = searchQuery
          await nextTick()
          
          if (!searchQuery.trim()) {
            expect(filteredBackups.value.length).toBe(backups.length)
          } else {
            const q = searchQuery.toLowerCase()
            const expected = backups.filter(b =>
              b.name.toLowerCase().includes(q) ||
              b.machineId?.toLowerCase().includes(q) ||
              b.provider?.toLowerCase().includes(q)
            )
            expect(filteredBackups.value.length).toBe(expected.length)
          }
        }
      ),
      { numRuns: 50 }
    )
  })
})


// ============================================================================
// Property 2: 備份 CRUD 列表長度不變量
// ============================================================================

describe('Property 2: 備份 CRUD 列表長度不變量', () => {
  it('創建備份後列表長度應增加 1', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 0, maxLength: 10 }),
        snapshotNameArbitrary,
        async (initialBackups, newName) => {
          // 確保新名稱不與現有名稱重複
          const existingNames = new Set(initialBackups.map(b => b.name))
          if (existingNames.has(newName)) return // Skip this case
          
          const initialLength = initialBackups.length
          
          // Mock: 創建成功後返回包含新備份的列表
          mockGetBackupList.mockResolvedValueOnce(initialBackups)
          mockCreateBackup.mockResolvedValue({ success: true, message: '備份成功' })
          
          const newBackup: BackupItem = {
            name: newName,
            backupTime: new Date().toISOString(),
            hasToken: true,
            hasMachineId: true,
            machineId: 'new-machine-id',
            provider: 'AWS',
            isCurrent: false,
            isOriginalMachine: false,
            isTokenExpired: false,
            subscriptionTitle: 'KIRO FREE',
            usageLimit: 1000,
            currentUsage: 0,
            balance: 1000,
            isLowBalance: false,
            cachedAt: new Date().toISOString(),
            folderId: '',
          }
          
          const { backups, createBackup, loadBackups } = useBackupManagement()
          await loadBackups()
          
          expect(backups.value.length).toBe(initialLength)
          
          // 模擬創建後重新載入
          mockGetBackupList.mockResolvedValueOnce([...initialBackups, newBackup])
          const result = await createBackup(newName)
          
          if (result.success) {
            await loadBackups()
            expect(backups.value.length).toBe(initialLength + 1)
          }
        }
      ),
      { numRuns: 30 }
    )
  })

  it('刪除備份後列表長度應減少 1', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 10 }),
        async (initialBackups) => {
          const initialLength = initialBackups.length
          const backupToDelete = initialBackups[0]
          
          mockGetBackupList.mockResolvedValueOnce(initialBackups)
          mockDeleteBackup.mockResolvedValue({ success: true, message: '刪除成功' })
          
          const { backups, deleteBackup, loadBackups } = useBackupManagement()
          await loadBackups()
          
          expect(backups.value.length).toBe(initialLength)
          
          // 模擬刪除後重新載入
          mockGetBackupList.mockResolvedValueOnce(initialBackups.slice(1))
          const result = await deleteBackup(backupToDelete.name)
          
          if (result.success) {
            await loadBackups()
            expect(backups.value.length).toBe(initialLength - 1)
          }
        }
      ),
      { numRuns: 30 }
    )
  })
})


// ============================================================================
// Property 3: 重複名稱創建失敗
// ============================================================================

describe('Property 3: 重複名稱創建失敗', () => {
  it('創建重複名稱的備份應失敗', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 10 }),
        async (initialBackups) => {
          const existingName = initialBackups[0].name
          
          mockGetBackupList.mockResolvedValue(initialBackups)
          mockCreateBackup.mockResolvedValue({ success: false, message: '備份名稱已存在' })
          
          const { createBackup, loadBackups } = useBackupManagement()
          await loadBackups()
          
          const result = await createBackup(existingName)
          
          expect(result.success).toBe(false)
        }
      ),
      { numRuns: 30 }
    )
  })
})

// ============================================================================
// Property 4: 備份切換狀態一致性
// ============================================================================

describe('Property 4: 備份切換狀態一致性', () => {
  it('切換後只有一個 isCurrent=true', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 2, maxLength: 10, withCurrentItem: true }),
        fc.nat(),
        async (initialBackups, targetIndex) => {
          const safeIndex = targetIndex % initialBackups.length
          const targetBackup = initialBackups[safeIndex]
          
          mockGetBackupList.mockResolvedValueOnce(initialBackups)
          mockSwitchToBackup.mockResolvedValue({ success: true, message: '切換成功' })
          
          const { backups, switchToBackup, loadBackups } = useBackupManagement()
          await loadBackups()
          
          // 模擬切換後的狀態
          const updatedBackups = initialBackups.map(b => ({
            ...b,
            isCurrent: b.name === targetBackup.name,
          }))
          mockGetBackupList.mockResolvedValueOnce(updatedBackups)
          
          const result = await switchToBackup(targetBackup.name)
          
          if (result.success) {
            await loadBackups()
            
            // 驗證只有一個 isCurrent=true
            const currentCount = backups.value.filter(b => b.isCurrent).length
            expect(currentCount).toBe(1)
            
            // 驗證是正確的備份
            const currentBackup = backups.value.find(b => b.isCurrent)
            expect(currentBackup?.name).toBe(targetBackup.name)
          }
        }
      ),
      { numRuns: 30 }
    )
  })
})


// ============================================================================
// Property 5: 批量刷新冷卻期過濾
// ============================================================================

describe('Property 5: 批量刷新冷卻期過濾', () => {
  it('批量刷新應跳過冷卻期中的備份', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 2, max: 10 }),
        async (count) => {
          // 生成唯一名稱的備份列表
          const initialBackups: BackupItem[] = Array.from({ length: count }, (_, i) => ({
            name: `backup-${i}`,
            backupTime: new Date().toISOString(),
            hasToken: true,
            hasMachineId: true,
            machineId: `machine-${i}`,
            provider: 'AWS',
            isCurrent: i === 0,
            isOriginalMachine: false,
            isTokenExpired: false,
            subscriptionTitle: 'KIRO FREE',
            usageLimit: 1000,
            currentUsage: 0,
            balance: 1000,
            isLowBalance: false,
            cachedAt: new Date().toISOString(),
            folderId: '',
          }))
          
          mockGetBackupList.mockResolvedValue(initialBackups)
          mockRefreshBackupUsage.mockClear()
          
          const { 
            selectedBackups, 
            toggleSelect, 
            batchRefreshUsage, 
            loadBackups 
          } = useBackupManagement()
          
          await loadBackups()
          
          // 選擇所有備份
          initialBackups.forEach(b => toggleSelect(b.name))
          
          // 模擬冷卻期函數：假設第一個備份在冷卻期中
          const cooldownSet = new Set([initialBackups[0].name])
          const isInCooldown = (name: string) => cooldownSet.has(name)
          
          // 執行批量刷新
          await batchRefreshUsage(isInCooldown)
          
          // 驗證：冷卻期中的備份不應被刷新
          const refreshCalls = mockRefreshBackupUsage.mock.calls
          const refreshedNames = refreshCalls.map(call => call[0])
          
          expect(refreshedNames).not.toContain(initialBackups[0].name)
          
          // 其他備份應該被刷新
          const expectedRefreshed = initialBackups.slice(1).map(b => b.name)
          expectedRefreshed.forEach(name => {
            expect(refreshedNames).toContain(name)
          })
        }
      ),
      { numRuns: 30 }
    )
  })
})

// ============================================================================
// Property 6: 批量重新生成機器碼
// ============================================================================

describe('Property 6: 批量重新生成機器碼', () => {
  it('批量操作應正確執行', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 1, max: 10 }),
        async (count) => {
          // 生成唯一名稱的備份列表
          const initialBackups: BackupItem[] = Array.from({ length: count }, (_, i) => ({
            name: `backup-${i}`,
            backupTime: new Date().toISOString(),
            hasToken: true,
            hasMachineId: true,
            machineId: `machine-${i}`,
            provider: 'AWS',
            isCurrent: i === 0,
            isOriginalMachine: false,
            isTokenExpired: false,
            subscriptionTitle: 'KIRO FREE',
            usageLimit: 1000,
            currentUsage: 0,
            balance: 1000,
            isLowBalance: false,
            cachedAt: new Date().toISOString(),
            folderId: '',
          }))
          
          mockGetBackupList.mockResolvedValue(initialBackups)
          mockRegenerateMachineID.mockClear()
          
          const { 
            selectedBackups, 
            toggleSelect, 
            batchRegenerateMachineID, 
            loadBackups 
          } = useBackupManagement()
          
          await loadBackups()
          
          // 選擇所有備份
          initialBackups.forEach(b => toggleSelect(b.name))
          
          const selectedCount = selectedBackups.value.size
          expect(selectedCount).toBe(initialBackups.length)
          
          // 執行批量重新生成
          await batchRegenerateMachineID()
          
          // 驗證：每個選中的備份都應該被調用
          expect(mockRegenerateMachineID).toHaveBeenCalledTimes(initialBackups.length)
          
          // 驗證：操作完成後選擇應被清空
          expect(selectedBackups.value.size).toBe(0)
        }
      ),
      { numRuns: 30 }
    )
  })
})


// ============================================================================
// Property 31: 並發 loadBackups 競態條件防護
// ============================================================================

describe('Property 31: 並發 loadBackups 競態條件防護', () => {
  it('多次並發調用應只執行一次', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 10 }),
        fc.integer({ min: 2, max: 5 }),
        async (initialBackups, concurrentCalls) => {
          // 使用延遲的 mock 來模擬異步操作
          let callCount = 0
          mockGetBackupList.mockImplementation(async () => {
            callCount++
            // 模擬網絡延遲
            await new Promise(resolve => setTimeout(resolve, 10))
            return initialBackups
          })
          
          const { loadBackups, isLoadingBackups } = useBackupManagement()
          
          // 同時發起多個 loadBackups 調用
          const promises = Array(concurrentCalls).fill(null).map(() => loadBackups())
          
          // 等待所有調用完成
          await Promise.all(promises)
          
          // 驗證：API 只應被調用一次（競態條件防護）
          expect(callCount).toBe(1)
          
          // 驗證：loading 狀態應該恢復為 false
          expect(isLoadingBackups.value).toBe(false)
        }
      ),
      { numRuns: 20 }
    )
  })

  it('第一次調用完成後可以再次調用', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 10 }),
        async (initialBackups) => {
          let callCount = 0
          mockGetBackupList.mockImplementation(async () => {
            callCount++
            await new Promise(resolve => setTimeout(resolve, 5))
            return initialBackups
          })
          
          const { loadBackups, isLoadingBackups } = useBackupManagement()
          
          // 第一次調用
          await loadBackups()
          expect(callCount).toBe(1)
          expect(isLoadingBackups.value).toBe(false)
          
          // 第二次調用（應該可以執行）
          await loadBackups()
          expect(callCount).toBe(2)
          expect(isLoadingBackups.value).toBe(false)
        }
      ),
      { numRuns: 20 }
    )
  })
})

// ============================================================================
// 額外測試：activeBackup 計算屬性
// ============================================================================

describe('activeBackup 計算屬性', () => {
  it('應返回 isCurrent=true 的備份', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 10, withCurrentItem: true }),
        async (initialBackups) => {
          mockGetBackupList.mockResolvedValue(initialBackups)
          
          const { activeBackup, loadBackups } = useBackupManagement()
          await loadBackups()
          
          const currentBackup = initialBackups.find(b => b.isCurrent)
          
          if (currentBackup) {
            expect(activeBackup.value).not.toBeNull()
            expect(activeBackup.value?.name).toBe(currentBackup.name)
          }
        }
      ),
      { numRuns: 30 }
    )
  })

  it('無 isCurrent=true 時應返回 null', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 10, withCurrentItem: false }),
        async (initialBackups) => {
          // 確保沒有 isCurrent=true
          const backupsWithoutCurrent = initialBackups.map(b => ({ ...b, isCurrent: false }))
          mockGetBackupList.mockResolvedValue(backupsWithoutCurrent)
          
          const { activeBackup, loadBackups } = useBackupManagement()
          await loadBackups()
          
          expect(activeBackup.value).toBeNull()
        }
      ),
      { numRuns: 30 }
    )
  })
})

// ============================================================================
// 額外測試：選擇操作
// ============================================================================

describe('選擇操作', () => {
  it('toggleSelect 應正確切換選擇狀態', async () => {
    await fc.assert(
      fc.asyncProperty(
        backupListArbitrary({ minLength: 1, maxLength: 10 }),
        async (initialBackups) => {
          mockGetBackupList.mockResolvedValue(initialBackups)
          
          const { selectedBackups, toggleSelect, loadBackups } = useBackupManagement()
          await loadBackups()
          
          const targetName = initialBackups[0].name
          
          // 初始狀態：未選中
          expect(selectedBackups.value.has(targetName)).toBe(false)
          
          // 第一次切換：選中
          toggleSelect(targetName)
          expect(selectedBackups.value.has(targetName)).toBe(true)
          
          // 第二次切換：取消選中
          toggleSelect(targetName)
          expect(selectedBackups.value.has(targetName)).toBe(false)
        }
      ),
      { numRuns: 30 }
    )
  })

  it('toggleSelectAll 應正確全選/取消全選', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 1, max: 10 }),
        async (count) => {
          // 生成唯一名稱的備份列表
          const initialBackups: BackupItem[] = Array.from({ length: count }, (_, i) => ({
            name: `backup-${i}`,
            backupTime: new Date().toISOString(),
            hasToken: true,
            hasMachineId: true,
            machineId: `machine-${i}`,
            provider: 'AWS',
            isCurrent: i === 0,
            isOriginalMachine: false,
            isTokenExpired: false,
            subscriptionTitle: 'KIRO FREE',
            usageLimit: 1000,
            currentUsage: 0,
            balance: 1000,
            isLowBalance: false,
            cachedAt: new Date().toISOString(),
            folderId: '',
          }))
          
          mockGetBackupList.mockResolvedValue(initialBackups)
          
          const { selectedBackups, toggleSelectAll, filteredBackups, loadBackups } = useBackupManagement()
          await loadBackups()
          
          // 初始狀態：無選中
          expect(selectedBackups.value.size).toBe(0)
          
          // 全選
          toggleSelectAll()
          expect(selectedBackups.value.size).toBe(filteredBackups.value.length)
          
          // 取消全選
          toggleSelectAll()
          expect(selectedBackups.value.size).toBe(0)
        }
      ),
      { numRuns: 30 }
    )
  })
})

// ============================================================================
// P0-FIX: 批量操作錯誤回報
// ============================================================================

describe('P0-FIX: 批量操作錯誤回報', () => {
  it('batchDelete 應返回 BatchResult 包含成功和失敗項目', async () => {
    // 生成 5 個備份
    const initialBackups: BackupItem[] = Array.from({ length: 5 }, (_, i) => ({
      name: `backup-${i}`,
      backupTime: new Date().toISOString(),
      hasToken: true,
      hasMachineId: true,
      machineId: `machine-${i}`,
      provider: 'AWS',
      isCurrent: false,
      isOriginalMachine: false,
      isTokenExpired: false,
      subscriptionTitle: 'KIRO FREE',
      usageLimit: 1000,
      currentUsage: 0,
      balance: 1000,
      isLowBalance: false,
      cachedAt: new Date().toISOString(),
      folderId: '',
    }))
    
    mockGetBackupList.mockResolvedValue(initialBackups)
    
    // 模擬第 3 個備份刪除失敗
    mockDeleteBackup.mockImplementation(async (name: string) => {
      if (name === 'backup-2') {
        throw new Error('刪除失敗')
      }
      return { success: true, message: '刪除成功' }
    })
    
    const { selectedBackups, toggleSelect, batchDelete, loadBackups } = useBackupManagement()
    await loadBackups()
    
    // 選擇所有備份
    initialBackups.forEach(b => toggleSelect(b.name))
    
    // 執行批量刪除
    const result = await batchDelete()
    
    // 驗證返回 BatchResult
    expect(result).toBeDefined()
    expect(result.successCount).toBe(4)
    expect(result.failedItems).toHaveLength(1)
    expect(result.failedItems[0].name).toBe('backup-2')
    expect(result.failedItems[0].error).toBe('刪除失敗')
  })

  it('batchRegenerateMachineID 應返回 BatchResult', async () => {
    const initialBackups: BackupItem[] = Array.from({ length: 3 }, (_, i) => ({
      name: `backup-${i}`,
      backupTime: new Date().toISOString(),
      hasToken: true,
      hasMachineId: true,
      machineId: `machine-${i}`,
      provider: 'AWS',
      isCurrent: false,
      isOriginalMachine: false,
      isTokenExpired: false,
      subscriptionTitle: 'KIRO FREE',
      usageLimit: 1000,
      currentUsage: 0,
      balance: 1000,
      isLowBalance: false,
      cachedAt: new Date().toISOString(),
      folderId: '',
    }))
    
    mockGetBackupList.mockResolvedValue(initialBackups)
    mockRegenerateMachineID.mockResolvedValue({ success: true, message: '' })
    
    const { selectedBackups, toggleSelect, batchRegenerateMachineID, loadBackups } = useBackupManagement()
    await loadBackups()
    
    initialBackups.forEach(b => toggleSelect(b.name))
    
    const result = await batchRegenerateMachineID()
    
    expect(result).toBeDefined()
    expect(result.success).toBe(true)
    expect(result.successCount).toBe(3)
    expect(result.failedItems).toHaveLength(0)
  })

  it('batchRefreshUsage 應返回 BatchResult 包含跳過的冷卻期項目', async () => {
    const initialBackups: BackupItem[] = Array.from({ length: 5 }, (_, i) => ({
      name: `backup-${i}`,
      backupTime: new Date().toISOString(),
      hasToken: true,
      hasMachineId: true,
      machineId: `machine-${i}`,
      provider: 'AWS',
      isCurrent: false,
      isOriginalMachine: false,
      isTokenExpired: false,
      subscriptionTitle: 'KIRO FREE',
      usageLimit: 1000,
      currentUsage: 0,
      balance: 1000,
      isLowBalance: false,
      cachedAt: new Date().toISOString(),
      folderId: '',
    }))
    
    mockGetBackupList.mockResolvedValue(initialBackups)
    mockRefreshBackupUsage.mockResolvedValue({ success: true, message: '' })
    
    const { selectedBackups, toggleSelect, batchRefreshUsage, loadBackups } = useBackupManagement()
    await loadBackups()
    
    initialBackups.forEach(b => toggleSelect(b.name))
    
    // 模擬 2 個備份在冷卻期
    const cooldownSet = new Set(['backup-0', 'backup-2'])
    const isInCooldown = (name: string) => cooldownSet.has(name)
    
    const result = await batchRefreshUsage(isInCooldown)
    
    expect(result).toBeDefined()
    expect(result.successCount).toBe(3) // 5 - 2 = 3
    expect(result.skippedCount).toBe(2) // 冷卻期中的 2 個
  })
})


// ============================================================================
// Phase 2 Task 1: 擴展功能測試
// ============================================================================

describe('Phase 2 Task 1.1: useBackupManagement 擴展功能', () => {
  describe('regenerateMachineID 單一備份重新生成機器碼', () => {
    it('應成功重新生成指定備份的機器碼', async () => {
      await fc.assert(
        fc.asyncProperty(
          snapshotNameArbitrary,
          async (backupName) => {
            mockRegenerateMachineID.mockResolvedValue({ success: true, message: '重新生成成功' })
            
            const { regenerateMachineID, regeneratingId } = useBackupManagement()
            
            // 驗證初始狀態
            expect(regeneratingId.value).toBe(null)
            
            const result = await regenerateMachineID(backupName)
            
            // 驗證 API 被正確調用
            expect(mockRegenerateMachineID).toHaveBeenCalledWith(backupName)
            expect(result.success).toBe(true)
            
            // 驗證狀態恢復
            expect(regeneratingId.value).toBe(null)
          }
        ),
        { numRuns: 30 }
      )
    })

    it('重新生成失敗時應返回錯誤結果', async () => {
      mockRegenerateMachineID.mockRejectedValue(new Error('重新生成失敗'))
      
      const { regenerateMachineID } = useBackupManagement()
      
      const result = await regenerateMachineID('test-backup')
      
      expect(result.success).toBe(false)
      expect(result.message).toBe('重新生成失敗')
    })

    it('重新生成過程中 regeneratingId 應設為備份名稱', async () => {
      let resolvePromise: (value: Result) => void
      const pendingPromise = new Promise<Result>((resolve) => {
        resolvePromise = resolve
      })
      mockRegenerateMachineID.mockReturnValue(pendingPromise)
      
      const { regenerateMachineID, regeneratingId } = useBackupManagement()
      
      const promise = regenerateMachineID('test-backup')
      
      // 驗證進行中狀態
      expect(regeneratingId.value).toBe('test-backup')
      
      // 完成操作
      resolvePromise!({ success: true, message: '' })
      await promise
      
      // 驗證狀態恢復
      expect(regeneratingId.value).toBe(null)
    })
  })

  describe('loadFullBackupData 整合載入', () => {
    it('應整合載入備份列表、設定和狀態', async () => {
      const mockBackups: BackupItem[] = [
        {
          name: 'backup-1',
          backupTime: new Date().toISOString(),
          hasToken: true,
          hasMachineId: true,
          machineId: 'machine-1',
          provider: 'AWS',
          isCurrent: true,
          isOriginalMachine: false,
          isTokenExpired: false,
          subscriptionTitle: 'KIRO FREE',
          usageLimit: 1000,
          currentUsage: 0,
          balance: 1000,
          isLowBalance: false,
          cachedAt: new Date().toISOString(),
          folderId: '',
        }
      ]
      
      mockGetBackupList.mockResolvedValue(mockBackups)
      mockGetCurrentMachineID.mockResolvedValue('current-machine-id')
      mockGetCurrentEnvironmentName.mockResolvedValue('current-env')
      
      const { loadFullBackupData, backups, currentMachineId, currentEnvironmentName } = useBackupManagement()
      
      await loadFullBackupData()
      
      // 驗證所有數據都被載入
      expect(backups.value).toEqual(mockBackups)
      expect(currentMachineId.value).toBe('current-machine-id')
      expect(currentEnvironmentName.value).toBe('current-env')
    })

    it('loadFullBackupData 應支援 onComplete 回調', async () => {
      const mockBackups: BackupItem[] = []
      mockGetBackupList.mockResolvedValue(mockBackups)
      mockGetCurrentMachineID.mockResolvedValue('machine-id')
      mockGetCurrentEnvironmentName.mockResolvedValue('env-name')
      
      const { loadFullBackupData } = useBackupManagement()
      
      const onComplete = vi.fn()
      await loadFullBackupData({ onComplete })
      
      expect(onComplete).toHaveBeenCalledTimes(1)
    })

    it('loadFullBackupData 失敗時應調用 onError 回調', async () => {
      mockGetBackupList.mockRejectedValue(new Error('載入失敗'))
      
      const { loadFullBackupData } = useBackupManagement()
      
      const onError = vi.fn()
      await loadFullBackupData({ onError })
      
      expect(onError).toHaveBeenCalledWith(expect.any(Error))
    })
  })
})


// ============================================================================
// Task 1.1: deleteBackup 超時保護測試
// ============================================================================

describe('Task 1.1: deleteBackup 超時保護', () => {
  it('deleteBackup 應該使用 withTimeout 包裝 API 調用', async () => {
    // Arrange: 模擬 API 超時（30秒以上）
    vi.useFakeTimers()
    
    let resolveDelete: (value: Result) => void
    mockDeleteBackup.mockImplementation(() => {
      return new Promise(resolve => {
        resolveDelete = resolve
      })
    })
    
    const { deleteBackup, deletingBackup } = useBackupManagement()
    
    // Act: 開始刪除操作
    const deletePromise = deleteBackup('test-backup')
    
    // 驗證 deletingBackup 狀態為 true
    expect(deletingBackup.value).toBe('test-backup')
    
    // 快進 30 秒（超時時間）
    await vi.advanceTimersByTimeAsync(30000)
    
    // 等待 Promise 完成
    const result = await deletePromise
    
    // Assert: 應該返回超時錯誤
    expect(result.success).toBe(false)
    expect(result.message).toContain('超時')
    
    // 驗證 deletingBackup 狀態已重置
    expect(deletingBackup.value).toBe(null)
    
    vi.useRealTimers()
  })

  it('deleteBackup 成功時應該正常返回結果', async () => {
    // Arrange
    mockDeleteBackup.mockResolvedValue({ success: true, message: '刪除成功' })
    
    const { deleteBackup, deletingBackup } = useBackupManagement()
    
    // Act
    const result = await deleteBackup('test-backup')
    
    // Assert
    expect(result.success).toBe(true)
    expect(deletingBackup.value).toBe(null)
  })

  it('deleteBackup 超時後 finally 區塊應該重置 deletingBackup 狀態', async () => {
    // Arrange
    vi.useFakeTimers()
    
    mockDeleteBackup.mockImplementation(() => {
      return new Promise(() => {
        // 永不 resolve，模擬超時
      })
    })
    
    const { deleteBackup, deletingBackup } = useBackupManagement()
    
    // Act
    const deletePromise = deleteBackup('test-backup')
    expect(deletingBackup.value).toBe('test-backup')
    
    // 快進超過超時時間
    await vi.advanceTimersByTimeAsync(31000)
    
    await deletePromise
    
    // Assert: 狀態應該被重置
    expect(deletingBackup.value).toBe(null)
    
    vi.useRealTimers()
  })
})

// ============================================================================
// Task 1.2: regenerateMachineID 超時保護測試
// ============================================================================

describe('Task 1.2: regenerateMachineID 超時保護', () => {
  it('regenerateMachineID 應該使用 withTimeout 包裝 API 調用', async () => {
    // Arrange
    vi.useFakeTimers()
    
    mockRegenerateMachineID.mockImplementation(() => {
      return new Promise(() => {
        // 永不 resolve，模擬超時
      })
    })
    
    const { regenerateMachineID, regeneratingId } = useBackupManagement()
    
    // Act
    const regeneratePromise = regenerateMachineID('test-backup')
    
    expect(regeneratingId.value).toBe('test-backup')
    
    // 快進 30 秒
    await vi.advanceTimersByTimeAsync(30000)
    
    const result = await regeneratePromise
    
    // Assert
    expect(result.success).toBe(false)
    expect(result.message).toContain('超時')
    expect(regeneratingId.value).toBe(null)
    
    vi.useRealTimers()
  })

  it('regenerateMachineID 成功時應該正常返回結果', async () => {
    // Arrange
    mockRegenerateMachineID.mockResolvedValue({ success: true, message: '重新生成成功' })
    
    const { regenerateMachineID, regeneratingId } = useBackupManagement()
    
    // Act
    const result = await regenerateMachineID('test-backup')
    
    // Assert
    expect(result.success).toBe(true)
    expect(regeneratingId.value).toBe(null)
  })

  it('regenerateMachineID 超時後 finally 區塊應該重置 regeneratingId 狀態', async () => {
    // Arrange
    vi.useFakeTimers()
    
    mockRegenerateMachineID.mockImplementation(() => {
      return new Promise(() => {})
    })
    
    const { regenerateMachineID, regeneratingId } = useBackupManagement()
    
    // Act
    const promise = regenerateMachineID('test-backup')
    expect(regeneratingId.value).toBe('test-backup')
    
    await vi.advanceTimersByTimeAsync(31000)
    await promise
    
    // Assert
    expect(regeneratingId.value).toBe(null)
    
    vi.useRealTimers()
  })
})

// ============================================================================
// Task 1.5: 批量刷新冷卻期檢查測試
// ============================================================================

describe('Task 1.5: 批量刷新全部冷卻期檢查', () => {
  it('當所有選中項目都在冷卻期時，應該返回 allInCooldown 標記', async () => {
    // Arrange
    const initialBackups: BackupItem[] = Array.from({ length: 3 }, (_, i) => ({
      name: `backup-${i}`,
      backupTime: new Date().toISOString(),
      hasToken: true,
      hasMachineId: true,
      machineId: `machine-${i}`,
      provider: 'AWS',
      isCurrent: false,
      isOriginalMachine: false,
      isTokenExpired: false,
      subscriptionTitle: 'KIRO FREE',
      usageLimit: 1000,
      currentUsage: 0,
      balance: 1000,
      isLowBalance: false,
      cachedAt: new Date().toISOString(),
      folderId: '',
    }))
    
    mockGetBackupList.mockResolvedValue(initialBackups)
    
    const { selectedBackups, toggleSelect, batchRefreshUsage, loadBackups } = useBackupManagement()
    await loadBackups()
    
    // 選擇所有備份
    initialBackups.forEach(b => toggleSelect(b.name))
    
    // 所有備份都在冷卻期
    const isInCooldown = () => true
    
    // Act
    const result = await batchRefreshUsage(isInCooldown)
    
    // Assert
    expect(result.success).toBe(false)
    expect(result.successCount).toBe(0)
    expect(result.skippedCount).toBe(3)
    expect(result.allInCooldown).toBe(true)
    
    // 選擇狀態不應該被清空
    expect(selectedBackups.value.size).toBe(3)
  })

  it('當部分項目在冷卻期時，應該正常處理非冷卻期項目', async () => {
    // Arrange
    const initialBackups: BackupItem[] = Array.from({ length: 3 }, (_, i) => ({
      name: `backup-${i}`,
      backupTime: new Date().toISOString(),
      hasToken: true,
      hasMachineId: true,
      machineId: `machine-${i}`,
      provider: 'AWS',
      isCurrent: false,
      isOriginalMachine: false,
      isTokenExpired: false,
      subscriptionTitle: 'KIRO FREE',
      usageLimit: 1000,
      currentUsage: 0,
      balance: 1000,
      isLowBalance: false,
      cachedAt: new Date().toISOString(),
      folderId: '',
    }))
    
    mockGetBackupList.mockResolvedValue(initialBackups)
    mockRefreshBackupUsage.mockResolvedValue({ success: true })
    
    const { selectedBackups, toggleSelect, batchRefreshUsage, loadBackups } = useBackupManagement()
    await loadBackups()
    
    initialBackups.forEach(b => toggleSelect(b.name))
    
    // 只有第一個在冷卻期
    const cooldownSet = new Set(['backup-0'])
    const isInCooldown = (name: string) => cooldownSet.has(name)
    
    // Act
    const result = await batchRefreshUsage(isInCooldown)
    
    // Assert
    expect(result.allInCooldown).toBeUndefined() // 不是全部在冷卻期
    expect(result.successCount).toBe(2)
    expect(result.skippedCount).toBe(1)
    
    // 選擇狀態應該被清空（因為有成功處理的項目）
    expect(selectedBackups.value.size).toBe(0)
  })
})
