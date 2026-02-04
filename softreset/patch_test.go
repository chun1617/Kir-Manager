package softreset

import (
	"strings"
	"testing"
)

// Task 3.1: 測試 V4 Patch 程式碼結構
func TestPatchCode_ContainsGetCustomMachineIdFunction(t *testing.T) {
	if !strings.Contains(patchCode, "function getCustomMachineId()") {
		t.Error("patchCode should contain getCustomMachineId function definition")
	}
}

func TestPatchCode_ContainsFormatValidationRegex(t *testing.T) {
	if !strings.Contains(patchCode, "/^[a-f0-9]{64}$/i") {
		t.Error("patchCode should contain format validation regex /^[a-f0-9]{64}$/i")
	}
}

func TestPatchCode_ContainsControlCharacterCleanup(t *testing.T) {
	// 檢查控制字元清理正則
	if !strings.Contains(patchCode, `[\x00-\x1F\x7F]`) {
		t.Error("patchCode should contain control character cleanup regex")
	}
}

func TestPatchCode_ContainsDynamicReadLogic(t *testing.T) {
	// V4 不應該有 let customMachineId = null 的靜態變數
	if strings.Contains(patchCode, "let customMachineId = null") {
		t.Error("V4 patchCode should not have static customMachineId variable")
	}
	// V4 不應該有 if (!customMachineId) return 的提前退出
	if strings.Contains(patchCode, "if (!customMachineId) return") {
		t.Error("V4 patchCode should not have early return based on static variable")
	}
}

// Task 3.2: 測試版本標記系統
func TestPatchMarker_IsV4(t *testing.T) {
	expected := "/* KIRO_MANAGER_PATCH_V4 */"
	if PatchMarker != expected {
		t.Errorf("PatchMarker should be %q, got %q", expected, PatchMarker)
	}
}

func TestOldPatchMarkerV3_Exists(t *testing.T) {
	expected := "/* KIRO_MANAGER_PATCH_V3 */"
	if OldPatchMarkerV3 != expected {
		t.Errorf("OldPatchMarkerV3 should be %q, got %q", expected, OldPatchMarkerV3)
	}
}

func TestIsOldPatched_DetectsV3(t *testing.T) {
	// 這個測試需要模擬檔案系統，暫時跳過
	// 實際測試會在整合測試中進行
	t.Skip("Requires file system mocking")
}

// Task 3.3: 測試 patchCode 包含 fs.promises.readFile 攔截
func TestPatchCode_ContainsFsPromisesReadFileInterception(t *testing.T) {
	if !strings.Contains(patchCode, "fs.promises.readFile") {
		t.Error("patchCode should contain fs.promises.readFile interception")
	}
}

// 測試 patchCode 包含錯誤處理邏輯
func TestPatchCode_ContainsErrorHandling(t *testing.T) {
	// 應該有 try-catch
	if !strings.Contains(patchCode, "try {") || !strings.Contains(patchCode, "catch") {
		t.Error("patchCode should contain try-catch error handling")
	}
	// 應該有 ENOENT 檢查
	if !strings.Contains(patchCode, "ENOENT") {
		t.Error("patchCode should handle ENOENT error")
	}
}

// 測試 patchCode 包含警告日誌
func TestPatchCode_ContainsWarningLogs(t *testing.T) {
	if !strings.Contains(patchCode, "[KIRO_PATCH]") {
		t.Error("patchCode should contain [KIRO_PATCH] warning prefix")
	}
}
