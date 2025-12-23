# Kiro Manager

> 跨平台 Kiro IDE 管理工具

一款基於 Wails + Vue 3 的桌面應用程式，提供 Kiro IDE 的帳號管理、Machine ID 備份與恢復、一鍵新機等功能。

## 功能特色

- **帳號備份與恢復** - 備份 Kiro 認證 Token 與 Machine ID，支援多帳號切換
- **一鍵新機** - 透過軟重置方式生成新的 Machine ID，跨平台支援
- **Machine ID 管理** - 跨平台取得與虛擬化系統 Machine ID
- **Kiro 進程檢測** - 自動檢測並關閉運行中的 Kiro 進程
- **自定義安裝路徑** - 支援手動指定 Kiro 安裝路徑
- **雙語言支援** - 繁體中文 / 簡體中文介面

## 一鍵新機

透過 Patch Kiro 的 `extension.js` 來攔截 Machine ID 讀取，實現虛擬化的 Machine ID。

**優點：**
- ✅ 跨平台支援（Windows / macOS / Linux）
- ✅ 不需要管理員權限
- ✅ 不修改系統 Registry，不影響其他軟體
- ✅ 可隨時還原為系統原始 Machine ID

**原理：**
1. 在 `~/.kiro/custom-machine-id` 儲存自訂的 Machine ID
2. Patch Kiro 的 extension.js，注入攔截程式碼
3. 當 Kiro 讀取 `vscode.env.machineId` 時，返回自訂值

**注意事項：**
- Kiro 更新後需要重新執行 Patch（程式會自動提示）
- 原始 extension.js 會備份為 `.kiro-manager-backup`

## 系統需求

| 功能 | Windows | macOS | Linux |
|------|---------|-------|-------|
| 一鍵新機 | ✅ | ✅ | ✅ |
| 帳號備份/恢復 | ✅ | ✅ | ✅ |
| Machine ID 讀取 | ✅ | ✅ | ✅ |

## 安裝方式

### 下載預編譯版本

前往 [Releases](https://github.com/your-repo/kiro-manager/releases) 下載對應平台的執行檔。

### 從原始碼編譯

**環境需求：**
- Go 1.21 或以上版本
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
2. 點擊「切換」按鈕
3. 程式會自動關閉 Kiro 並切換 Machine ID 與 Token

### 一鍵新機

1. 點擊「一鍵新機」按鈕
2. 程式會自動備份原始 Machine ID（首次使用時）
3. 生成新的 UUID 作為 Machine ID
4. 清除 SSO 快取

### 還原原始機器

點擊「還原」刪除自訂 Machine ID，恢復使用系統原始值。

## 專案結構

```
kiro-manager/
├── app.go              # Wails 綁定層
├── main.go             # GUI 入口點
├── main_cli.go         # CLI 入口點
├── awssso/             # AWS SSO 快取模組
├── backup/             # 帳號備份模組
├── kiropath/           # Kiro 路徑偵測
├── kiroprocess/        # Kiro 進程檢測
├── machineid/          # Machine ID 核心模組
├── settings/           # 應用程式設定模組
├── softreset/          # 一鍵新機模組（跨平台）
│   ├── softreset.go    # 自訂 Machine ID 管理
│   └── patch.go        # extension.js Patch 邏輯
└── frontend/           # Vue 3 前端
    ├── src/
    │   ├── App.vue
    │   ├── components/
    │   └── i18n/       # 國際化
    └── ...
```

## 技術棧

- **後端**: Go 1.21+
- **前端**: Vue 3 + TypeScript + Tailwind CSS
- **框架**: Wails v2
- **國際化**: vue-i18n

## 注意事項

⚠️ **Kiro 更新後**
- Kiro 更新後 extension.js 會被覆蓋，需要重新 Patch
- 程式會自動檢測 Patch 狀態並提示重新 Patch

⚠️ **安全提醒**
- 建議在執行一鍵新機前先備份當前帳號

## 授權條款

MIT License
