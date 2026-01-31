# Kiro Manager

> 跨平台 Kiro IDE 管理工具 | v0.2.2

一款基於 Wails + Vue 3 的桌面應用程式，提供 Kiro IDE 的帳號管理、Machine ID 備份與恢復、一鍵新機等功能。

## 功能特色

- **帳號備份與恢復** - 備份 Kiro 認證 Token 與 Machine ID，支援多帳號切換
- **一鍵新機** - 透過重置方式生成新的 Machine ID，跨平台支援
- **Machine ID 管理** - 跨平台取得與虛擬化系統 Machine ID
- **用量查詢** - 顯示訂閱類型、總額度、已使用、餘額，支援低餘額警告（閾值可配置）
- **Token 自動刷新** - 過期時自動刷新，支援 Social (GitHub/Google) 和 IdC (AWS Identity Center) 認證
- **批量操作** - 支援批量刪除、批量刷新餘額、批量重新生成機器碼
- **Kiro 進程檢測** - 自動檢測並關閉運行中的 Kiro 進程
- **自定義安裝路徑** - 支援手動指定 Kiro 安裝路徑，或自動偵測
- **雙語言支援** - 繁體中文 / 簡體中文介面

## 一鍵新機

透過 Patch Kiro 的 `extension.js` 來攔截 Machine ID 讀取，實現虛擬化的 Machine ID。

**優點：**
- ✅ 跨平台支援（Windows / macOS / Linux）
- ✅ 不需要管理員權限
- ✅ 不修改系統 Registry，不影響其他軟體
- ✅ 可隨時還原為系統原始 Machine ID

**原理：**
1. 在 `~/.kiro/custom-machine-id` 儲存自訂的 Machine ID（SHA256 雜湊值）
2. 在 `~/.kiro/custom-machine-id-raw` 儲存原始 UUID（供 UI 顯示）
3. Patch Kiro 的 extension.js，注入多層攔截程式碼（V3 版本）
4. 攔截 `vscode.env.machineId`、`node-machine-id`、`child_process`、`fs` 等讀取方式

**注意事項：**
- Kiro 更新後需要重新執行 Patch（程式會自動提示）
- 原始 extension.js 會備份為 `.kiro-manager-backup`

## 系統需求

| 功能 | Windows | macOS | Linux |
|------|---------|-------|-------|
| 一鍵新機 | ✅ | ✅ | ✅ |
| 帳號備份/恢復 | ✅ | ✅ | ✅ |
| Machine ID 讀取 | ✅ | ✅ | ✅ |
| 進程檢測 | ✅ | ⚠️ | ⚠️ |

> ⚠️ macOS/Linux 進程檢測功能開發中

## 安裝方式

### 下載預編譯版本

前往 [Releases](https://github.com/your-repo/kiro-manager/releases) 下載對應平台的執行檔。

### 從原始碼編譯

**環境需求：**
- Go 1.25.5 或以上版本
- Node.js 18+
- Wails CLI

```bash
# 安裝 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 克隆專案
git clone https://github.com/your-repo/kiro-manager.git
cd kiro-manager

# 開發模式
wails dev

# 編譯發布版本
wails build
```

## 使用說明

### 備份帳號

1. 確保已登入 Kiro IDE
2. 開啟 Kiro Manager
3. 輸入備份名稱，點擊「建立備份」
4. 備份將儲存於執行檔同層的 `backups/` 目錄

### 切換帳號

1. 從備份列表選擇要切換的帳號
2. 點擊「載入」按鈕
3. 程式會自動關閉 Kiro 並切換 Machine ID 與 Token

### 一鍵新機

1. 點擊「一鍵新機」按鈕
2. 程式會自動備份原始 Machine ID（首次使用時）
3. 生成新的 UUID 作為 Machine ID
4. 清除 SSO 快取

### 還原原始機器

點擊「還原出廠」刪除自訂 Machine ID，恢復使用系統原始值。

### 批量操作

1. 在環境快照列表中勾選多個項目
2. 使用表格上方的批量操作按鈕：
   - **批量刷新餘額** - 序列查詢各帳號餘額（跳過冷卻中項目）
   - **批量刷新機器碼** - 為選中項目生成新的 UUID
   - **批量刪除** - 刪除選中的備份

## 用量計算

餘額計算邏輯：
- **總額度** = 基本額度 + 免費試用額度（未過期時）+ 獎勵額度（未耗盡時）
- 當 `bonus.Status == "EXHAUSTED"` 時，獎勵額度**不計入**總額度
- 當 `freeTrialInfo.FreeTrialStatus == "EXPIRED"` 時，免費試用額度**不計入**總額度
- **低餘額警告閾值**可在設定中調整（預設 20%）
- **刷新冷卻期** 60 秒，防止頻繁 API 呼叫

## 專案結構

```
kiro-manager/
├── app.go              # Wails 綁定層（27 個 API）
├── main.go             # GUI 入口點
├── main_cli.go         # CLI 入口點
├── awssso/             # AWS SSO 快取模組
├── backup/             # 帳號備份模組
├── kiropath/           # Kiro 路徑偵測
│   ├── kiropath.go     # 跨平台路徑偵測
│   └── cache.go        # 路徑快取機制
├── kiroprocess/        # Kiro 進程檢測
├── kiroversion/        # Kiro 版本偵測
├── machineid/          # Machine ID 核心模組
├── settings/           # 應用程式設定模組
├── softreset/          # 一鍵新機模組（跨平台）
│   ├── softreset.go    # 自訂 Machine ID 管理
│   └── patch.go        # extension.js Patch 邏輯（V3）
├── tokenrefresh/       # Token 刷新模組
│   └── tokenrefresh.go # Social/IdC 雙認證支援
├── usage/              # 用量查詢模組
├── internal/
│   └── cmdutil/        # 命令工具（視窗隱藏）
└── frontend/           # Vue 3 前端
    ├── src/
    │   ├── App.vue
    │   ├── components/
    │   └── i18n/       # 國際化
    └── ...
```

## 技術棧

- **後端**: Go 1.25.5+
- **前端**: Vue 3 + TypeScript + Tailwind CSS
- **框架**: Wails v2
- **國際化**: vue-i18n
- **Kiro 預設版本**: 0.8.206

## 注意事項

⚠️ **Kiro 更新後**
- Kiro 更新後 extension.js 會被覆蓋，需要重新 Patch
- 程式會自動檢測 Patch 狀態並提示重新 Patch

⚠️ **安全提醒**
- 建議在執行一鍵新機前先備份當前帳號

⚠️ **認證類型**
- **Social 認證** (GitHub/Google)：使用 `profileArn` 進行 API 呼叫
- **IdC 認證** (AWS Identity Center)：使用 `clientIdHash` 關聯 clientId/clientSecret

## 授權條款

MIT License
