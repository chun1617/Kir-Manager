import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fc from 'fast-check'
import { useUIState } from '../useUIState'
import type { MenuType, ToastType } from '@/types/ui'

describe('Feature: app-vue-decoupling, useUIState composable', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('Property 11: Toggle 狀態反轉', () => {
    it('toggleMobileMenu 應反轉 isMobileMenuOpen 狀態', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          (initialState) => {
            const { isMobileMenuOpen, toggleMobileMenu } = useUIState()
            
            // 設定初始狀態
            isMobileMenuOpen.value = initialState
            
            // 執行 toggle
            toggleMobileMenu()
            
            // 驗證：狀態應反轉
            return isMobileMenuOpen.value === !initialState
          }
        ),
        { numRuns: 100 }
      )
    })

    it('連續 toggle 兩次應回到原始狀態', () => {
      fc.assert(
        fc.property(
          fc.boolean(),
          (initialState) => {
            const { isMobileMenuOpen, toggleMobileMenu } = useUIState()
            
            isMobileMenuOpen.value = initialState
            
            toggleMobileMenu()
            toggleMobileMenu()
            
            return isMobileMenuOpen.value === initialState
          }
        ),
        { numRuns: 100 }
      )
    })
  })

  describe('Property 28: 菜單切換與移動端菜單關閉', () => {
    it('setActiveMenu 應關閉移動端菜單', () => {
      fc.assert(
        fc.property(
          fc.constantFrom<MenuType>('dashboard', 'settings', 'oauth'),
          fc.boolean(),
          (targetMenu, mobileMenuState) => {
            const { activeMenu, isMobileMenuOpen, setActiveMenu } = useUIState()
            
            // 設定初始狀態
            isMobileMenuOpen.value = mobileMenuState
            
            // 執行菜單切換
            setActiveMenu(targetMenu)
            
            // 驗證：
            // 1. activeMenu 應更新為目標菜單
            // 2. isMobileMenuOpen 應為 false
            return activeMenu.value === targetMenu && isMobileMenuOpen.value === false
          }
        ),
        { numRuns: 100 }
      )
    })

    it('setActiveMenu 應正確更新 activeMenu 狀態', () => {
      fc.assert(
        fc.property(
          fc.constantFrom<MenuType>('dashboard', 'settings', 'oauth'),
          fc.constantFrom<MenuType>('dashboard', 'settings', 'oauth'),
          (initialMenu, targetMenu) => {
            const { activeMenu, setActiveMenu } = useUIState()
            
            activeMenu.value = initialMenu
            setActiveMenu(targetMenu)
            
            return activeMenu.value === targetMenu
          }
        ),
        { numRuns: 100 }
      )
    })
  })

  describe('Property 29: 確認對話框 Promise 解析', () => {
    it('showConfirmDialog 應返回 Promise，onConfirm 解析為 true', async () => {
      const { confirmDialog, showConfirmDialog } = useUIState()
      
      const promise = showConfirmDialog({
        title: 'Test Title',
        message: 'Test Message'
      })
      
      // 驗證對話框已顯示
      expect(confirmDialog.value.show).toBe(true)
      expect(confirmDialog.value.title).toBe('Test Title')
      expect(confirmDialog.value.message).toBe('Test Message')
      
      // 模擬用戶點擊確認
      confirmDialog.value.onConfirm()
      
      // 驗證 Promise 解析為 true
      const result = await promise
      expect(result).toBe(true)
      expect(confirmDialog.value.show).toBe(false)
    })

    it('showConfirmDialog 應返回 Promise，onCancel 解析為 false', async () => {
      const { confirmDialog, showConfirmDialog } = useUIState()
      
      const promise = showConfirmDialog({
        title: 'Test Title',
        message: 'Test Message'
      })
      
      // 驗證對話框已顯示
      expect(confirmDialog.value.show).toBe(true)
      
      // 模擬用戶點擊取消
      confirmDialog.value.onCancel()
      
      // 驗證 Promise 解析為 false
      const result = await promise
      expect(result).toBe(false)
      expect(confirmDialog.value.show).toBe(false)
    })

    it('showConfirmDialog 應使用預設值', async () => {
      const { confirmDialog, showConfirmDialog } = useUIState()
      
      showConfirmDialog({
        title: 'Test',
        message: 'Message'
      })
      
      // 驗證預設值
      expect(confirmDialog.value.type).toBe('warning')
      expect(confirmDialog.value.confirmText).toBeTruthy()
      expect(confirmDialog.value.cancelText).toBeTruthy()
      
      // 清理
      confirmDialog.value.onCancel()
    })

    it('showConfirmDialog 應接受自定義選項', async () => {
      const { confirmDialog, showConfirmDialog } = useUIState()
      
      showConfirmDialog({
        title: 'Custom Title',
        message: 'Custom Message',
        type: 'danger',
        confirmText: 'Delete',
        cancelText: 'Keep'
      })
      
      expect(confirmDialog.value.type).toBe('danger')
      expect(confirmDialog.value.confirmText).toBe('Delete')
      expect(confirmDialog.value.cancelText).toBe('Keep')
      
      // 清理
      confirmDialog.value.onCancel()
    })
  })

  describe('Property 30: Toast 自動消失', () => {
    it('showToast 後 toast.show 應在指定時間後變為 false', () => {
      const { toast, showToast } = useUIState()
      
      // 顯示 Toast（預設 3000ms）
      showToast('Test message', 'success')
      
      // 驗證 Toast 已顯示
      expect(toast.value.show).toBe(true)
      expect(toast.value.message).toBe('Test message')
      expect(toast.value.type).toBe('success')
      
      // 快進 2999ms - Toast 應仍顯示
      vi.advanceTimersByTime(2999)
      expect(toast.value.show).toBe(true)
      
      // 快進 1ms（總共 3000ms）- Toast 應消失
      vi.advanceTimersByTime(1)
      expect(toast.value.show).toBe(false)
    })

    it('showToast 應支持自定義持續時間', () => {
      const { toast, showToast } = useUIState()
      
      // 顯示 Toast，自定義 5000ms
      showToast('Custom duration', 'warning', 5000)
      
      expect(toast.value.show).toBe(true)
      
      // 快進 4999ms - Toast 應仍顯示
      vi.advanceTimersByTime(4999)
      expect(toast.value.show).toBe(true)
      
      // 快進 1ms（總共 5000ms）- Toast 應消失
      vi.advanceTimersByTime(1)
      expect(toast.value.show).toBe(false)
    })

    it('showToast 應支持所有 Toast 類型', () => {
      fc.assert(
        fc.property(
          fc.constantFrom<ToastType>('success', 'error', 'warning'),
          fc.string({ minLength: 1, maxLength: 100 }),
          (toastType, message) => {
            const { toast, showToast } = useUIState()
            
            showToast(message, toastType)
            
            return (
              toast.value.show === true &&
              toast.value.message === message &&
              toast.value.type === toastType
            )
          }
        ),
        { numRuns: 50 }
      )
    })

    it('連續 showToast 應覆蓋前一個 Toast', () => {
      const { toast, showToast } = useUIState()
      
      showToast('First message', 'success')
      expect(toast.value.message).toBe('First message')
      
      showToast('Second message', 'error')
      expect(toast.value.message).toBe('Second message')
      expect(toast.value.type).toBe('error')
    })

    it('P1-FIX: 連續 showToast 應清除前一個計時器，防止 Toast 提前消失', () => {
      const { toast, showToast } = useUIState()
      
      // 第一個 Toast，3000ms
      showToast('First message', 'success', 3000)
      expect(toast.value.show).toBe(true)
      
      // 快進 2000ms
      vi.advanceTimersByTime(2000)
      expect(toast.value.show).toBe(true)
      
      // 第二個 Toast，3000ms（應該清除第一個計時器）
      showToast('Second message', 'error', 3000)
      expect(toast.value.show).toBe(true)
      expect(toast.value.message).toBe('Second message')
      
      // 快進 1000ms（總共 3000ms 從第一個 Toast 開始）
      // 如果沒有清除計時器，第一個 Toast 的計時器會在這裡觸發
      vi.advanceTimersByTime(1000)
      
      // Toast 應該仍然顯示（因為第二個 Toast 的計時器還有 2000ms）
      expect(toast.value.show).toBe(true)
      
      // 再快進 2000ms（第二個 Toast 的計時器到期）
      vi.advanceTimersByTime(2000)
      expect(toast.value.show).toBe(false)
    })
  })

  describe('Property 32: 統一錯誤處理', () => {
    it('handleError 應顯示錯誤 toast', () => {
      const { toast, handleError } = useUIState()
      
      const error = new Error('Test error message')
      handleError(error)
      
      expect(toast.value.show).toBe(true)
      expect(toast.value.type).toBe('error')
      expect(toast.value.message).toBe('Test error message')
    })

    it('handleError 應處理字串錯誤', () => {
      const { toast, handleError } = useUIState()
      
      handleError('String error')
      
      expect(toast.value.show).toBe(true)
      expect(toast.value.type).toBe('error')
      expect(toast.value.message).toBe('String error')
    })

    it('handleError 應使用 fallbackMessage 當錯誤無訊息時', () => {
      const { toast, handleError } = useUIState()
      
      handleError(null, 'Fallback message')
      
      expect(toast.value.show).toBe(true)
      expect(toast.value.type).toBe('error')
      expect(toast.value.message).toBe('Fallback message')
    })

    it('handleError 應處理帶有 message 屬性的物件', () => {
      const { toast, handleError } = useUIState()
      
      handleError({ message: 'Object error message' })
      
      expect(toast.value.show).toBe(true)
      expect(toast.value.type).toBe('error')
      expect(toast.value.message).toBe('Object error message')
    })

    it('handleError 應使用預設訊息當無法提取錯誤訊息時', () => {
      const { toast, handleError } = useUIState()
      
      handleError({})
      
      expect(toast.value.show).toBe(true)
      expect(toast.value.type).toBe('error')
      // 應有預設錯誤訊息
      expect(toast.value.message).toBeTruthy()
    })
  })

  describe('Filter 相關功能', () => {
    it('toggleFilter 應切換 openFilter 狀態', () => {
      const { openFilter, toggleFilter } = useUIState()
      
      expect(openFilter.value).toBe(null)
      
      toggleFilter('subscription')
      expect(openFilter.value).toBe('subscription')
      
      toggleFilter('subscription')
      expect(openFilter.value).toBe(null)
    })

    it('toggleFilter 切換到不同篩選器應更新 openFilter', () => {
      const { openFilter, toggleFilter } = useUIState()
      
      toggleFilter('subscription')
      expect(openFilter.value).toBe('subscription')
      
      toggleFilter('provider')
      expect(openFilter.value).toBe('provider')
    })

    it('closeAllFilters 應關閉所有篩選器', () => {
      const { openFilter, toggleFilter, closeAllFilters } = useUIState()
      
      toggleFilter('subscription')
      expect(openFilter.value).toBe('subscription')
      
      closeAllFilters()
      expect(openFilter.value).toBe(null)
    })
  })

  describe('初始狀態', () => {
    it('應有正確的初始值', () => {
      const { 
        activeMenu, 
        isMobileMenuOpen, 
        openFilter, 
        toast, 
        confirmDialog 
      } = useUIState()
      
      expect(activeMenu.value).toBe('dashboard')
      expect(isMobileMenuOpen.value).toBe(false)
      expect(openFilter.value).toBe(null)
      expect(toast.value.show).toBe(false)
      expect(confirmDialog.value.show).toBe(false)
    })
  })

  describe('P0-3: Toast Timer Cleanup', () => {
    it('cleanupToast 應清除 Toast 計時器並隱藏 Toast', () => {
      const { toast, showToast, cleanupToast } = useUIState()
      
      showToast('Test message', 'success', 5000)
      expect(toast.value.show).toBe(true)
      
      // 快進 2000ms
      vi.advanceTimersByTime(2000)
      expect(toast.value.show).toBe(true)
      
      // 調用 cleanup
      cleanupToast()
      
      // Toast 應立即隱藏
      expect(toast.value.show).toBe(false)
      
      // 快進剩餘時間，確保沒有錯誤發生（計時器已被清除）
      vi.advanceTimersByTime(3000)
      expect(toast.value.show).toBe(false)
    })

    it('cleanupToast 在沒有活動 Toast 時應安全調用', () => {
      const { toast, cleanupToast } = useUIState()
      
      expect(toast.value.show).toBe(false)
      
      // 應該不會拋出錯誤
      expect(() => cleanupToast()).not.toThrow()
      expect(toast.value.show).toBe(false)
    })
  })
})
