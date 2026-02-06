/**
 * 備份相關類型定義
 * @description 從 App.vue 提取的核心介面，用於備份管理功能
 * @see frontend/wailsjs/go/models.ts - Wails 自動生成的類型（作為參考）
 */

/**
 * 備份項目
 * @description 代表一個快照備份的完整資訊
 */
export interface BackupItem {
  /** 快照名稱 */
  name: string
  /** 備份時間 (RFC3339 格式) */
  backupTime: string
  /** 是否有 Token */
  hasToken: boolean
  /** 是否有 Machine ID */
  hasMachineId: boolean
  /** Machine ID 值 */
  machineId: string
  /** 認證提供者 (如 'aws', 'github') */
  provider: string
  /** 是否為當前使用中的快照 */
  isCurrent: boolean
  /** Machine ID 是否與原始機器相同 */
  isOriginalMachine: boolean
  /** Token 是否已過期 */
  isTokenExpired: boolean
  /** 訂閱類型名稱 */
  subscriptionTitle: string
  /** 總額度 */
  usageLimit: number
  /** 已使用額度 */
  currentUsage: number
  /** 餘額 */
  balance: number
  /** 是否為低餘額狀態 */
  isLowBalance: boolean
  /** 快取時間 (RFC3339 格式) */
  cachedAt: string
  /** 所屬文件夾 ID */
  folderId: string
}

/**
 * 文件夾項目
 * @description 代表一個快照分類文件夾
 */
export interface FolderItem {
  /** 唯一識別碼 */
  id: string
  /** 文件夾名稱 */
  name: string
  /** 建立時間 (RFC3339 格式) */
  createdAt: string
  /** 排序順序 */
  order: number
  /** 包含的快照數量 */
  snapshotCount: number
}

/**
 * 操作結果
 * @description 通用的操作結果結構
 */
export interface Result {
  /** 操作是否成功 */
  success: boolean
  /** 結果訊息 */
  message: string
}

/**
 * 當前用量資訊
 * @description 當前帳號的用量統計
 */
export interface CurrentUsageInfo {
  /** 訂閱類型名稱 */
  subscriptionTitle: string
  /** 總額度 */
  usageLimit: number
  /** 已使用額度 */
  currentUsage: number
  /** 餘額 */
  balance: number
  /** 是否為低餘額狀態 */
  isLowBalance: boolean
}

/**
 * 應用設定
 * @description 應用程式的全域設定
 */
export interface AppSettings {
  /** 低餘額閾值 (0.0 ~ 1.0) */
  lowBalanceThreshold: number
  /** Kiro IDE 版本號 */
  kiroVersion: string
  /** 是否使用自動偵測版本號 */
  useAutoDetect: boolean
  /** 自定義 Kiro 安裝路徑 */
  customKiroInstallPath: string
}

/**
 * 刷新間隔規則
 * @description 定義特定餘額範圍內的監控間隔
 */
export interface RefreshIntervalRule {
  /** 餘額下限 */
  minBalance: number
  /** 餘額上限 */
  maxBalance: number
  /** 刷新間隔 (分鐘) */
  interval: number
}

/**
 * 自動切換設定
 * @description 自動切換功能的完整設定
 */
export interface AutoSwitchSettings {
  /** 是否啟用自動切換 */
  enabled: boolean
  /** 觸發切換的餘額閾值 */
  balanceThreshold: number
  /** 目標快照的最低餘額要求 */
  minTargetBalance: number
  /** 允許切換的文件夾 ID 列表 */
  folderIds: string[]
  /** 允許切換的訂閱類型列表 */
  subscriptionTypes: string[]
  /** 刷新間隔規則列表 */
  refreshIntervals: RefreshIntervalRule[]
  /** 切換時是否通知 */
  notifyOnSwitch: boolean
  /** 低餘額時是否通知 */
  notifyOnLowBalance: boolean
}

/**
 * 自動切換狀態
 * @description 自動切換監控器的當前狀態
 */
export interface AutoSwitchStatus {
  /** 狀態 ('stopped' | 'running' | 'cooldown') */
  status: 'stopped' | 'running' | 'cooldown'
  /** 最後檢測的餘額 */
  lastBalance: number
  /** 冷卻剩餘時間 (秒) */
  cooldownRemaining: number
  /** 已切換次數 */
  switchCount: number
}

/**
 * 路徑偵測結果
 * @description Kiro 安裝路徑偵測的結果
 */
export interface PathDetectionResult {
  /** 偵測到的路徑 */
  path: string
  /** 偵測是否成功 */
  success: boolean
  /** 嘗試過的策略列表 */
  triedStrategies?: string[]
  /** 各策略的失敗原因 */
  failureReasons?: Record<string, string>
}

/**
 * 軟重置狀態
 * @description 軟重置功能的當前狀態
 */
export interface SoftResetStatus {
  /** Extension 是否已 Patch */
  isPatched: boolean
  /** 是否有自定義機器碼 */
  hasCustomId: boolean
  /** 自定義機器碼值 */
  customMachineId: string
  /** Extension 路徑 */
  extensionPath: string
  /** 是否支援軟重置 */
  isSupported: boolean
}

/**
 * 批量操作結果
 * @description 批量操作的詳細結果，包含成功/失敗/跳過的項目統計
 */
export interface BatchResult {
  /** 整體操作是否成功（無失敗項目時為 true） */
  success: boolean
  /** 成功處理的項目數量 */
  successCount: number
  /** 失敗的項目列表 */
  failedItems: Array<{ name: string; error: string }>
  /** 跳過的項目數量（用於 batchRefreshUsage 的冷卻期項目） */
  skippedCount?: number
  /** 是否所有選中項目都在冷卻期（用於 batchRefreshUsage） */
  allInCooldown?: boolean
}
