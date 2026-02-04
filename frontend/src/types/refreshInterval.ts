/**
 * 刷新頻率規則
 * @description 定義特定餘額範圍內的監控間隔
 * @requirements 1.1, 1.2, 3.1, 3.2, 3.3
 */
export interface RefreshRule {
  /** 唯一識別碼 */
  id: string
  /** 餘額下限 (>= 0) */
  minBalance: number
  /** 餘額上限 (-1 表示無上限) */
  maxBalance: number
  /** 刷新間隔 (分鐘, >= 1) */
  interval: number
}

/**
 * 規則驗證結果
 * @description 驗證操作的結果結構
 */
export interface ValidationResult {
  /** 驗證是否通過 */
  valid: boolean
  /** 錯誤訊息 i18n key */
  error?: string
}

/**
 * 規則操作類型
 * @description 定義可執行的規則操作
 */
export type RuleOperation = 'add' | 'update' | 'delete'
