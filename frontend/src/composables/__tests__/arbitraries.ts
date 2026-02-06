/**
 * 測試數據生成器
 * @description 使用 fast-check 為 Property-Based Testing 提供任意值生成器
 * @see https://fast-check.dev/
 */
import * as fc from 'fast-check'
import type {
  BackupItem,
  FolderItem,
  CurrentUsageInfo,
  AppSettings,
  AutoSwitchSettings,
  AutoSwitchStatus,
  RefreshIntervalRule,
  PathDetectionResult,
  Result,
  SoftResetStatus,
} from '@/types/backup'

// ============================================================================
// 基礎生成器
// ============================================================================

/**
 * 生成有效的快照名稱
 * @description 符合命名規則的字串（非空、無非法字元）
 */
export const snapshotNameArbitrary = fc.stringMatching(/^[a-zA-Z0-9\u4e00-\u9fff_-]{1,50}$/)

/**
 * 生成 RFC3339 格式的時間字串
 */
export const rfc3339DateArbitrary = fc.integer({
  min: new Date('2020-01-01').getTime(),
  max: new Date('2030-12-31').getTime(),
}).map(ts => new Date(ts).toISOString())

/**
 * 生成 UUID 格式的字串
 */
export const uuidArbitrary = fc.uuid()

/**
 * 生成十六進制字串（用於 Machine ID）
 */
export const hexStringArbitrary = fc.array(
  fc.integer({ min: 0, max: 15 }),
  { minLength: 32, maxLength: 64 }
).map(arr => arr.map(n => n.toString(16)).join(''))

/**
 * 生成認證提供者
 */
export const providerArbitrary = fc.constantFrom('aws', 'github', 'google', '')

/**
 * 生成訂閱類型
 */
export const subscriptionTypeArbitrary = fc.constantFrom(
  'Free',
  'Pro',
  'Team',
  'Enterprise',
  ''
)


// ============================================================================
// 用量相關生成器
// ============================================================================

/**
 * 生成用量數值 (0 ~ 10000)
 */
export const usageValueArbitrary = fc.float({
  min: 0,
  max: 10000,
  noNaN: true,
})

/**
 * 生成餘額百分比 (0.0 ~ 1.0)
 */
export const balancePercentArbitrary = fc.float({
  min: 0,
  max: 1,
  noNaN: true,
})

// ============================================================================
// 核心介面生成器
// ============================================================================

/**
 * BackupItem 生成器
 * @description 生成完整的備份項目數據
 */
export const backupItemArbitrary: fc.Arbitrary<BackupItem> = fc.record({
  name: snapshotNameArbitrary,
  backupTime: rfc3339DateArbitrary,
  hasToken: fc.boolean(),
  hasMachineId: fc.boolean(),
  machineId: hexStringArbitrary,
  provider: providerArbitrary,
  isCurrent: fc.boolean(),
  isOriginalMachine: fc.boolean(),
  isTokenExpired: fc.boolean(),
  subscriptionTitle: subscriptionTypeArbitrary,
  usageLimit: usageValueArbitrary,
  currentUsage: usageValueArbitrary,
  balance: usageValueArbitrary,
  isLowBalance: fc.boolean(),
  cachedAt: rfc3339DateArbitrary,
  folderId: fc.oneof(uuidArbitrary, fc.constant('')),
})

/**
 * FolderItem 生成器
 * @description 生成完整的文件夾項目數據
 */
export const folderItemArbitrary: fc.Arbitrary<FolderItem> = fc.record({
  id: uuidArbitrary,
  name: fc.stringMatching(/^[a-zA-Z0-9\u4e00-\u9fff_-]{1,30}$/),
  createdAt: rfc3339DateArbitrary,
  order: fc.nat({ max: 100 }),
  snapshotCount: fc.nat({ max: 50 }),
})

/**
 * CurrentUsageInfo 生成器
 * @description 生成當前用量資訊
 */
export const currentUsageInfoArbitrary: fc.Arbitrary<CurrentUsageInfo> = fc.record({
  subscriptionTitle: subscriptionTypeArbitrary,
  usageLimit: usageValueArbitrary,
  currentUsage: usageValueArbitrary,
  balance: usageValueArbitrary,
  isLowBalance: fc.boolean(),
})

/**
 * Result 生成器
 * @description 生成操作結果
 */
export const resultArbitrary: fc.Arbitrary<Result> = fc.record({
  success: fc.boolean(),
  message: fc.string({ minLength: 0, maxLength: 200 }),
})


/**
 * AppSettings 生成器
 * @description 生成應用設定
 */
export const appSettingsArbitrary: fc.Arbitrary<AppSettings> = fc.record({
  lowBalanceThreshold: balancePercentArbitrary,
  kiroVersion: fc.stringMatching(/^\d+\.\d+\.\d+$/),
  useAutoDetect: fc.boolean(),
  customKiroInstallPath: fc.oneof(
    fc.constant(''),
    fc.constant('C:\\Program Files\\Kiro'),
    fc.constant('/Applications/Kiro.app')
  ),
})

/**
 * RefreshIntervalRule 生成器
 * @description 生成刷新間隔規則
 */
export const refreshIntervalRuleArbitrary: fc.Arbitrary<RefreshIntervalRule> = fc.record({
  minBalance: fc.nat({ max: 1000 }),
  maxBalance: fc.oneof(fc.nat({ max: 10000 }), fc.constant(-1)),
  interval: fc.integer({ min: 1, max: 60 }),
})

/**
 * AutoSwitchSettings 生成器
 * @description 生成自動切換設定
 */
export const autoSwitchSettingsArbitrary: fc.Arbitrary<AutoSwitchSettings> = fc.record({
  enabled: fc.boolean(),
  balanceThreshold: usageValueArbitrary,
  minTargetBalance: usageValueArbitrary,
  folderIds: fc.array(uuidArbitrary, { maxLength: 5 }),
  subscriptionTypes: fc.array(subscriptionTypeArbitrary, { maxLength: 4 }),
  refreshIntervals: fc.array(refreshIntervalRuleArbitrary, { maxLength: 5 }),
  notifyOnSwitch: fc.boolean(),
  notifyOnLowBalance: fc.boolean(),
})

/**
 * AutoSwitchStatus 生成器
 * @description 生成自動切換狀態
 */
export const autoSwitchStatusArbitrary: fc.Arbitrary<AutoSwitchStatus> = fc.record({
  status: fc.constantFrom('stopped', 'running', 'cooldown'),
  lastBalance: usageValueArbitrary,
  cooldownRemaining: fc.nat({ max: 300 }),
  switchCount: fc.nat({ max: 100 }),
})

/**
 * PathDetectionResult 生成器
 * @description 生成路徑偵測結果
 */
export const pathDetectionResultArbitrary: fc.Arbitrary<PathDetectionResult> = fc.record({
  path: fc.oneof(
    fc.constant(''),
    fc.constant('C:\\Program Files\\Kiro'),
    fc.constant('/Applications/Kiro.app'),
    fc.constant('/usr/local/bin/kiro')
  ),
  success: fc.boolean(),
  triedStrategies: fc.option(fc.array(fc.string(), { maxLength: 5 }), { nil: undefined }),
  failureReasons: fc.option(
    fc.dictionary(fc.string(), fc.string()),
    { nil: undefined }
  ),
})

/**
 * SoftResetStatus 生成器
 * @description 生成軟重置狀態
 */
export const softResetStatusArbitrary: fc.Arbitrary<SoftResetStatus> = fc.record({
  isPatched: fc.boolean(),
  hasCustomId: fc.boolean(),
  customMachineId: fc.oneof(hexStringArbitrary, fc.constant('')),
  extensionPath: fc.oneof(
    fc.constant(''),
    fc.constant('C:\\Users\\user\\.kiro\\extensions'),
    fc.constant('/home/user/.kiro/extensions'),
    fc.constant('/Users/user/.kiro/extensions')
  ),
  isSupported: fc.boolean(),
})


// ============================================================================
// 組合生成器（用於複雜測試場景）
// ============================================================================

/**
 * 生成備份列表
 * @param options 配置選項
 * @description 確保生成的備份名稱唯一，避免 Property 測試中的名稱衝突
 */
export const backupListArbitrary = (options?: {
  minLength?: number
  maxLength?: number
  withCurrentItem?: boolean
}) => {
  const { minLength = 0, maxLength = 10, withCurrentItem = false } = options ?? {}
  
  return fc.array(backupItemArbitrary, { minLength, maxLength }).map(items => {
    // 確保名稱唯一：為重複名稱添加索引後綴
    const nameCount = new Map<string, number>()
    const uniqueItems = items.map((item) => {
      const count = nameCount.get(item.name) || 0
      nameCount.set(item.name, count + 1)
      return {
        ...item,
        name: count === 0 ? item.name : `${item.name}_${count}`,
      }
    })
    
    if (withCurrentItem && uniqueItems.length > 0) {
      // 確保只有一個 isCurrent = true（使用確定性方式）
      return uniqueItems.map((item, index) => ({
        ...item,
        isCurrent: index === 0,
      }))
    }
    return uniqueItems.map(item => ({ ...item, isCurrent: false }))
  })
}

/**
 * 生成文件夾列表
 */
export const folderListArbitrary = (options?: {
  minLength?: number
  maxLength?: number
}) => {
  const { minLength = 0, maxLength = 10 } = options ?? {}
  return fc.array(folderItemArbitrary, { minLength, maxLength })
}

/**
 * 生成一致的用量數據（currentUsage <= usageLimit, balance = usageLimit - currentUsage）
 */
export const consistentUsageArbitrary = fc.record({
  usageLimit: fc.float({ min: 100, max: 10000, noNaN: true }),
  usagePercent: fc.float({ min: 0, max: 1, noNaN: true }),
}).map(({ usageLimit, usagePercent }) => {
  const currentUsage = usageLimit * usagePercent
  const balance = usageLimit - currentUsage
  return {
    usageLimit,
    currentUsage,
    balance,
    isLowBalance: balance / usageLimit < 0.2,
  }
})

/**
 * 生成帶有一致用量的 BackupItem
 */
export const consistentBackupItemArbitrary: fc.Arbitrary<BackupItem> = fc
  .tuple(backupItemArbitrary, consistentUsageArbitrary)
  .map(([item, usage]) => ({
    ...item,
    ...usage,
  }))
