import { ref } from 'vue'
import type {
  MenuType,
  ToastType,
  ToastState,
  ConfirmDialogState,
  ConfirmDialogOptions,
  UIStateReturn
} from '@/types/ui'

/** Toast 計時器 ID（P1-FIX: 防止計時器累積） */
let toastTimeoutId: ReturnType<typeof setTimeout> | null = null

/**
 * UI 狀態管理 Composable
 * @description 管理應用程式的 UI 狀態，包括菜單、Toast、確認對話框等
 * @requirements 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 7.8, 8.2
 */
export function useUIState(): UIStateReturn {
  // ============================================
  // 狀態定義
  // ============================================
  
  /** 當前活動菜單 */
  const activeMenu = ref<MenuType>('dashboard')
  
  /** 移動端菜單開關狀態 */
  const isMobileMenuOpen = ref(false)
  
  /** 當前開啟的篩選器名稱 */
  const openFilter = ref<string | null>(null)
  
  /** Toast 通知狀態 */
  const toast = ref<ToastState>({
    show: false,
    message: '',
    type: 'success'
  })
  
  /** 確認對話框狀態 */
  const confirmDialog = ref<ConfirmDialogState>({
    show: false,
    title: '',
    message: '',
    type: 'warning',
    confirmText: '確認',
    cancelText: '取消',
    onConfirm: () => {},
    onCancel: () => {}
  })

  // ============================================
  // 菜單相關方法
  // ============================================
  
  /**
   * 設定活動菜單
   * @description 切換菜單時自動關閉移動端菜單
   * @requirements 7.1 - 菜單切換
   * @param menu 目標菜單
   */
  const setActiveMenu = (menu: MenuType): void => {
    activeMenu.value = menu
    // Property 28: 菜單切換時關閉移動端菜單
    isMobileMenuOpen.value = false
  }
  
  /**
   * 切換移動端菜單
   * @description 反轉移動端菜單的開關狀態
   * @requirements 7.2 - 移動端菜單控制
   */
  const toggleMobileMenu = (): void => {
    // Property 11: Toggle 狀態反轉
    isMobileMenuOpen.value = !isMobileMenuOpen.value
  }

  // ============================================
  // 篩選器相關方法
  // ============================================
  
  /**
   * 切換篩選器
   * @description 開啟指定篩選器，或關閉已開啟的篩選器
   * @requirements 7.3 - 篩選器控制
   * @param name 篩選器名稱
   */
  const toggleFilter = (name: string): void => {
    openFilter.value = openFilter.value === name ? null : name
  }
  
  /**
   * 關閉所有篩選器
   * @requirements 7.4 - 篩選器控制
   */
  const closeAllFilters = (): void => {
    openFilter.value = null
  }

  // ============================================
  // Toast 相關方法
  // ============================================
  
  /**
   * 顯示 Toast 通知
   * @description 顯示 Toast 並在指定時間後自動消失
   * @requirements 7.5 - Toast 通知
   * @param message 訊息內容
   * @param type Toast 類型
   * @param duration 持續時間（毫秒），預設 3000ms
   */
  const showToast = (
    message: string,
    type: ToastType,
    duration: number = 3000
  ): void => {
    // P1-FIX: 清除前一個計時器，防止 Toast 提前消失
    if (toastTimeoutId) {
      clearTimeout(toastTimeoutId)
      toastTimeoutId = null
    }
    
    toast.value = { show: true, message, type }
    // Property 30: Toast 自動消失
    toastTimeoutId = setTimeout(() => {
      toast.value.show = false
      toastTimeoutId = null
    }, duration)
  }

  /**
   * 清理 Toast 計時器
   * @description 清除 Toast 計時器並隱藏 Toast，用於組件卸載時防止 Memory Leak
   * @requirements P0-3 - Toast Timer Memory Leak 修復
   */
  const cleanupToast = (): void => {
    if (toastTimeoutId) {
      clearTimeout(toastTimeoutId)
      toastTimeoutId = null
    }
    toast.value.show = false
  }

  // ============================================
  // 確認對話框相關方法
  // ============================================
  
  /**
   * 顯示確認對話框
   * @description 顯示確認對話框並返回 Promise
   * @requirements 7.6 - 確認對話框
   * @param options 對話框選項
   * @returns Promise<boolean> - 用戶確認返回 true，取消返回 false
   */
  const showConfirmDialog = (options: ConfirmDialogOptions): Promise<boolean> => {
    return new Promise((resolve) => {
      // Property 29: 確認對話框 Promise 解析
      confirmDialog.value = {
        show: true,
        title: options.title,
        message: options.message,
        type: options.type || 'warning',
        confirmText: options.confirmText || '確認',
        cancelText: options.cancelText || '取消',
        onConfirm: () => {
          confirmDialog.value.show = false
          resolve(true)
        },
        onCancel: () => {
          confirmDialog.value.show = false
          resolve(false)
        }
      }
    })
  }

  // ============================================
  // 錯誤處理方法
  // ============================================
  
  /**
   * 統一錯誤處理
   * @description 提取錯誤訊息並顯示錯誤 Toast
   * @requirements 8.2 - 統一錯誤處理
   * @param error 錯誤物件
   * @param fallbackMessage 備用訊息
   */
  const handleError = (error: unknown, fallbackMessage?: string): void => {
    // Property 32: 統一錯誤處理
    let message: string
    
    if (error instanceof Error) {
      message = error.message
    } else if (typeof error === 'string') {
      message = error
    } else if (error && typeof error === 'object' && 'message' in error) {
      message = String((error as { message: unknown }).message)
    } else if (fallbackMessage) {
      message = fallbackMessage
    } else {
      message = '發生未知錯誤'
    }
    
    showToast(message, 'error')
  }

  // ============================================
  // 返回公開 API
  // ============================================
  
  return {
    // 狀態
    activeMenu,
    isMobileMenuOpen,
    openFilter,
    toast,
    confirmDialog,
    
    // 方法
    setActiveMenu,
    toggleMobileMenu,
    toggleFilter,
    closeAllFilters,
    showToast,
    cleanupToast,
    showConfirmDialog,
    handleError
  }
}
