package softreset

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"kiro-manager/kiropath"
)

const (
	// PatchMarker 用於識別是否已 patch 的標記
	PatchMarker    = "/* KIRO_MANAGER_PATCH_V4 */"
	PatchEndMarker = "/* END_KIRO_MANAGER_PATCH */"
	BackupSuffix   = ".kiro-manager-backup"
	// OldPatchMarker 用於識別舊版 patch，需要重新 patch
	OldPatchMarker   = "/* KIRO_MANAGER_PATCH_V1 */"
	OldPatchMarkerV2 = "/* KIRO_MANAGER_PATCH_V2 */"
	OldPatchMarkerV3 = "/* KIRO_MANAGER_PATCH_V3 */"
)

var (
	ErrExtensionNotFound = errors.New("extension.js not found")
	ErrAlreadyPatched    = errors.New("extension.js is already patched")
	ErrNotPatched        = errors.New("extension.js is not patched")
	ErrBackupNotFound    = errors.New("backup file not found")
)

// patchCode 注入的 JavaScript 程式碼
// V4: 動態讀取 - 每次訪問時從檔案讀取，無需重啟即可生效
const patchCode = `/* KIRO_MANAGER_PATCH_V4 */
(function() {
  const fs = require('fs');
  const path = require('path');
  const os = require('os');
  const childProcess = require('child_process');
  const customIdPath = path.join(os.homedir(), '.kiro', 'custom-machine-id');

  // V4: 動態讀取函數，每次調用都從檔案讀取
  function getCustomMachineId() {
    try {
      let content = fs.readFileSync(customIdPath, 'utf8');
      // 移除控制字元
      content = content.replace(/[\x00-\x1F\x7F]/g, '');
      // trim 空白
      content = content.trim();
      // 空內容檢查
      if (!content) return null;
      // 格式驗證：64 字元 hex
      if (!/^[a-f0-9]{64}$/i.test(content)) {
        console.warn('[KIRO_PATCH] Invalid machine ID format, ignoring');
        return null;
      }
      return content;
    } catch (err) {
      if (err.code !== 'ENOENT') {
        console.warn('[KIRO_PATCH] Failed to read custom-machine-id:', err.code || err.message);
      }
      return null;
    }
  }

  // 1. 攔截 Module._load（vscode.env.machineId 和 node-machine-id）
  const Module = require('module');
  const originalLoad = Module._load;
  Module._load = function(request, parent, isMain) {
    const mod = originalLoad.call(this, request, parent, isMain);
    if (request === 'vscode') {
      const originalEnv = mod.env;
      return new Proxy(mod, {
        get(target, prop) {
          if (prop === 'env') {
            return new Proxy(originalEnv, {
              get(envTarget, envProp) {
                if (envProp === 'machineId') {
                  const customId = getCustomMachineId();
                  return customId !== null ? customId : envTarget[envProp];
                }
                return envTarget[envProp];
              }
            });
          }
          return target[prop];
        }
      });
    }
    if (mod && typeof mod === 'object' && (typeof mod.machineIdSync === 'function' || typeof mod.machineId === 'function')) {
      const originalMachineIdSync = mod.machineIdSync;
      const originalMachineId = mod.machineId;
      return new Proxy(mod, {
        get(target, prop) {
          if (prop === 'machineIdSync') {
            return function() {
              const customId = getCustomMachineId();
              return customId !== null ? customId : originalMachineIdSync.call(target);
            };
          }
          if (prop === 'machineId') {
            return async function() {
              const customId = getCustomMachineId();
              return customId !== null ? customId : originalMachineId.call(target);
            };
          }
          return target[prop];
        }
      });
    }
    return mod;
  };

  // 2. 攔截 child_process（針對 @opentelemetry 和其他直接執行命令的模組）
  const machineIdPatterns = [
    'REG.exe QUERY', 'REG QUERY', 'MachineGuid',
    'ioreg', 'IOPlatformExpertDevice',
    'kenv', 'smbios.system.uuid', 'kern.hostuuid'
  ];
  const isMachineIdCmd = (cmd) => cmd && machineIdPatterns.some(p => cmd.includes(p));

  const originalExec = childProcess.exec;
  childProcess.exec = function(cmd, options, callback) {
    if (isMachineIdCmd(cmd)) {
      const customId = getCustomMachineId();
      if (customId !== null) {
        if (typeof options === 'function') { callback = options; options = {}; }
        setImmediate(() => callback && callback(null, customId, ''));
        return { on: () => {}, stdout: { on: () => {} }, stderr: { on: () => {} } };
      }
    }
    return originalExec.apply(this, arguments);
  };

  const originalExecSync = childProcess.execSync;
  childProcess.execSync = function(cmd, options) {
    if (isMachineIdCmd(cmd)) {
      const customId = getCustomMachineId();
      if (customId !== null) return Buffer.from(customId);
    }
    return originalExecSync.apply(this, arguments);
  };

  // 3. 攔截 fs（針對 Linux /etc/machine-id）
  const machineIdPaths = ['/etc/machine-id', '/var/lib/dbus/machine-id', '/etc/hostid'];
  const isMachineIdPath = (p) => p && machineIdPaths.some(mp => String(p).includes(mp));

  const originalReadFile = fs.readFile;
  fs.readFile = function(filePath, options, callback) {
    if (isMachineIdPath(filePath)) {
      const customId = getCustomMachineId();
      if (customId !== null) {
        if (typeof options === 'function') { callback = options; }
        setImmediate(() => callback && callback(null, customId));
        return;
      }
    }
    return originalReadFile.apply(this, arguments);
  };

  const originalReadFileSync = fs.readFileSync;
  fs.readFileSync = function(filePath, options) {
    if (isMachineIdPath(filePath)) {
      const customId = getCustomMachineId();
      if (customId !== null) return customId;
    }
    return originalReadFileSync.apply(this, arguments);
  };

  if (fs.promises) {
    const originalPromisesReadFile = fs.promises.readFile;
    fs.promises.readFile = async function(filePath, options) {
      if (isMachineIdPath(filePath)) {
        const customId = getCustomMachineId();
        if (customId !== null) return customId;
      }
      return originalPromisesReadFile.apply(this, arguments);
    };
  }
})();
/* END_KIRO_MANAGER_PATCH */
`


// GetExtensionJSPath 取得 extension.js 的路徑
func GetExtensionJSPath() (string, error) {
	installPath, err := kiropath.GetKiroInstallPath()
	if err != nil {
		return "", err
	}

	var extensionPath string
	switch runtime.GOOS {
	case "windows":
		// Windows: {install}/resources/app/extensions/kiro.kiro-agent/dist/extension.js
		extensionPath = filepath.Join(installPath, "resources", "app", "extensions", "kiro.kiro-agent", "dist", "extension.js")
	case "darwin":
		// macOS: {install}/Contents/Resources/app/extensions/kiro.kiro-agent/dist/extension.js
		extensionPath = filepath.Join(installPath, "Contents", "Resources", "app", "extensions", "kiro.kiro-agent", "dist", "extension.js")
	case "linux":
		// Linux: {install}/resources/app/extensions/kiro.kiro-agent/dist/extension.js
		extensionPath = filepath.Join(installPath, "resources", "app", "extensions", "kiro.kiro-agent", "dist", "extension.js")
	default:
		return "", errors.New("unsupported platform: " + runtime.GOOS)
	}

	if _, err := os.Stat(extensionPath); os.IsNotExist(err) {
		return "", ErrExtensionNotFound
	}

	return extensionPath, nil
}

// IsPatched 檢查 extension.js 是否已被 patch（當前版本）
func IsPatched() (bool, error) {
	extPath, err := GetExtensionJSPath()
	if err != nil {
		return false, err
	}

	// 只讀取檔案開頭部分來檢查
	file, err := os.Open(extPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// 讀取前 1KB 來檢查標記
	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}

	return strings.Contains(string(buf[:n]), PatchMarker), nil
}

// IsOldPatched 檢查 extension.js 是否被舊版 patch（V1, V2 或 V3）
func IsOldPatched() (bool, error) {
	extPath, err := GetExtensionJSPath()
	if err != nil {
		return false, err
	}

	file, err := os.Open(extPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}

	content := string(buf[:n])
	// 有舊版標記（V1, V2 或 V3）但沒有新版標記（V4）
	hasOldPatch := strings.Contains(content, OldPatchMarker) ||
		strings.Contains(content, OldPatchMarkerV2) ||
		strings.Contains(content, OldPatchMarkerV3)
	hasCurrentPatch := strings.Contains(content, PatchMarker)
	return hasOldPatch && !hasCurrentPatch, nil
}

// BackupExtensionJS 備份原始 extension.js
func BackupExtensionJS() error {
	extPath, err := GetExtensionJSPath()
	if err != nil {
		return err
	}

	backupPath := extPath + BackupSuffix

	// 如果備份已存在，不覆蓋
	if _, err := os.Stat(backupPath); err == nil {
		return nil
	}

	return copyFile(extPath, backupPath)
}

// RestoreExtensionJS 從備份還原 extension.js
func RestoreExtensionJS() error {
	extPath, err := GetExtensionJSPath()
	if err != nil {
		return err
	}

	backupPath := extPath + BackupSuffix

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return ErrBackupNotFound
	}

	// 還原檔案
	if err := copyFile(backupPath, extPath); err != nil {
		return err
	}

	// 還原成功後刪除備份檔案
	_ = os.Remove(backupPath)

	return nil
}

// PatchExtensionJS 在 extension.js 開頭注入攔截程式碼
func PatchExtensionJS() error {
	extPath, err := GetExtensionJSPath()
	if err != nil {
		return err
	}

	// 檢查是否已是最新版 patch
	patched, err := IsPatched()
	if err != nil {
		return err
	}
	if patched {
		return nil // 已經是最新版 patch，不重複處理
	}

	// 檢查是否有舊版 patch，需要先移除
	oldPatched, err := IsOldPatched()
	if err != nil {
		return err
	}
	if oldPatched {
		// 移除舊版 patch
		if err := UnpatchExtensionJS(); err != nil {
			return err
		}
	}

	// 備份原始檔案
	if err := BackupExtensionJS(); err != nil {
		return err
	}

	// 讀取原始內容
	content, err := os.ReadFile(extPath)
	if err != nil {
		return err
	}

	// 在開頭加入 patch 程式碼
	newContent := patchCode + string(content)

	// 寫回檔案
	return os.WriteFile(extPath, []byte(newContent), 0644)
}

// UnpatchExtensionJS 移除注入的程式碼
func UnpatchExtensionJS() error {
	extPath, err := GetExtensionJSPath()
	if err != nil {
		return err
	}

	// 檢查是否有任何版本的 patch
	patched, err := IsPatched()
	if err != nil {
		return err
	}
	oldPatched, err := IsOldPatched()
	if err != nil {
		return err
	}
	if !patched && !oldPatched {
		return nil // 沒有任何 patch，不需要處理
	}

	// 讀取內容
	content, err := os.ReadFile(extPath)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// 找到 patch 結束標記的位置
	endIdx := strings.Index(contentStr, PatchEndMarker)
	if endIdx == -1 {
		// 找不到結束標記，嘗試從備份還原
		return RestoreExtensionJS()
	}

	// 移除 patch 程式碼（包含結束標記和換行）
	endIdx += len(PatchEndMarker)
	if endIdx < len(contentStr) && contentStr[endIdx] == '\n' {
		endIdx++
	}

	newContent := contentStr[endIdx:]

	return os.WriteFile(extPath, []byte(newContent), 0644)
}

// copyFile 複製檔案
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}
