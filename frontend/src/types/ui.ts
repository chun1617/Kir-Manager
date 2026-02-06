/**
 * UI 狀態相關類型定義
 * @description 用於 useUIState composable
 */

/** 菜單類型 */
export type MenuType = 'dashboard' | 'settings' | 'oauth'

/** Toast 類型 */
export type ToastType = 'success' | 'error' | 'warning'

/** 確認對話框類型 */
export type ConfirmDialogType = 'warning' | 'danger' | 'info'

/** Toast 狀態 */
export interface ToastState {
  show: boolean
  message: string
  type: ToastType
}

/** 確認對話框狀態 */
export interface ConfirmDialogState {
  show: boolean
  title: string
  message: string
  type: ConfirmDialogType
  confirmText: string
  cancelText: string
  onConfirm: () => void
  onCancel: () => void
}

/** 確認對話框選項 */
export interface ConfirmDialogOptions {
  title: string
  message: string
  type?: ConfirmDialogType
  confirmText?: string
  cancelText?: string
}

// ============================================================================
// 自動切換事件類型
// ============================================================================

/** 自動切換事件類型 */
export type AutoSwitchEventType = 
  | 'switch'
  | 'switch_fail'
  | 'low_balance'
  | 'cooldown'
  | 'max_switch'
  | 'no_candidates'

/** 自動切換事件資料 */
export interface AutoSwitchEventData {
  from?: string
  to?: string
}

/** 自動切換事件 */
export interface AutoSwitchEvent {
  Type: AutoSwitchEventType
  Message?: string
  Data?: AutoSwitchEventData
}

// ============================================================================
// useUIState 類型
// ============================================================================

/** useUIState 返回類型 */
export interface UIStateReturn {
  // 狀態
  activeMenu: import('vue').Ref<MenuType>
  isMobileMenuOpen: import('vue').Ref<boolean>
  openFilter: import('vue').Ref<string | null>
  toast: import('vue').Ref<ToastState>
  confirmDialog: import('vue').Ref<ConfirmDialogState>
  
  // 方法
  setActiveMenu: (menu: MenuType) => void
  toggleMobileMenu: () => void
  toggleFilter: (name: string) => void
  closeAllFilters: () => void
  showToast: (message: string, type: ToastType, duration?: number) => void
  cleanupToast: () => void
  showConfirmDialog: (options: ConfirmDialogOptions) => Promise<boolean>
  handleError: (error: unknown, fallbackMessage?: string) => void
}
