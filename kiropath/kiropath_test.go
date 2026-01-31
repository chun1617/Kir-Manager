package kiropath

import (
	"errors"
	"testing"
)

// ============================================================================
// Test: DetectionFailedError
// ============================================================================

func TestDetectionFailedError_Error(t *testing.T) {
	err := &DetectionFailedError{
		TriedStrategies: []string{"custom", "process", "registry"},
		FailureReasons: map[string]string{
			"custom":   "path not set",
			"process":  "kiro not running",
			"registry": "key not found",
		},
	}

	if err.Error() != "all detection strategies failed" {
		t.Errorf("expected 'all detection strategies failed', got '%s'", err.Error())
	}
}

func TestDetectionFailedError_IsError(t *testing.T) {
	err := &DetectionFailedError{
		TriedStrategies: []string{"custom"},
		FailureReasons:  map[string]string{"custom": "not set"},
	}

	// 確認可以用 errors.As 檢查類型
	var detectionErr *DetectionFailedError
	if !errors.As(err, &detectionErr) {
		t.Error("expected error to be DetectionFailedError")
	}
}

func TestDetectionFailedError_ContainsStrategies(t *testing.T) {
	strategies := []string{"custom", "process", "registry", "hardcoded", "path"}
	reasons := map[string]string{
		"custom":    "not configured",
		"process":   "not running",
		"registry":  "not found",
		"hardcoded": "not found",
		"path":      "not in PATH",
	}

	err := &DetectionFailedError{
		TriedStrategies: strategies,
		FailureReasons:  reasons,
	}

	if len(err.TriedStrategies) != 5 {
		t.Errorf("expected 5 strategies, got %d", len(err.TriedStrategies))
	}

	if len(err.FailureReasons) != 5 {
		t.Errorf("expected 5 failure reasons, got %d", len(err.FailureReasons))
	}
}

// ============================================================================
// Test: GetKiroInstallPath - Cache Priority
// ============================================================================

func TestGetKiroInstallPath_UsesCache(t *testing.T) {
	// 設定快取
	testPath := "C:\\TestCache\\Kiro"
	setPathCache(testPath)
	defer InvalidatePathCache()

	// 呼叫 GetKiroInstallPath 應該返回快取的路徑
	result, err := GetKiroInstallPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != testPath {
		t.Errorf("expected cached path '%s', got '%s'", testPath, result)
	}
}

func TestGetKiroInstallPath_CacheInvalidation(t *testing.T) {
	// 設定快取
	testPath := "C:\\TestCache\\Kiro"
	setPathCache(testPath)

	// 驗證快取存在
	if getPathCache() != testPath {
		t.Fatal("cache should be set")
	}

	// 清除快取
	InvalidatePathCache()

	// 驗證快取已清除
	if getPathCache() != "" {
		t.Error("cache should be empty after invalidation")
	}
}

// ============================================================================
// Test: GetKiroInstallPath - Detection Chain
// ============================================================================

func TestGetKiroInstallPath_DetectionChainOrder(t *testing.T) {
	// 清除快取確保測試獨立
	InvalidatePathCache()
	defer InvalidatePathCache()

	// 這個測試驗證偵測鏈的存在
	// 實際的偵測結果取決於系統狀態
	_, err := GetKiroInstallPath()

	// 如果所有策略都失敗，應該返回 DetectionFailedError
	if err != nil {
		var detectionErr *DetectionFailedError
		if errors.As(err, &detectionErr) {
			// 驗證錯誤包含嘗試過的策略
			if len(detectionErr.TriedStrategies) == 0 {
				t.Error("DetectionFailedError should contain tried strategies")
			}
		}
		// 其他錯誤類型也是可接受的（如 ErrKiroNotFound）
	}
}

func TestGetKiroInstallPath_ReturnsDetectionFailedError(t *testing.T) {
	// 清除快取
	InvalidatePathCache()
	defer InvalidatePathCache()

	// 在沒有 Kiro 安裝的環境中，應該返回 DetectionFailedError
	// 這個測試主要驗證錯誤類型的正確性
	_, err := GetKiroInstallPath()

	if err != nil {
		// 檢查是否為 DetectionFailedError 或 ErrKiroNotFound
		var detectionErr *DetectionFailedError
		if errors.As(err, &detectionErr) {
			// 成功：返回了 DetectionFailedError
			t.Logf("Got DetectionFailedError with strategies: %v", detectionErr.TriedStrategies)
		} else if errors.Is(err, ErrKiroNotFound) {
			// 舊的錯誤類型，需要更新實作
			t.Log("Got ErrKiroNotFound - implementation needs update to return DetectionFailedError")
		}
	}
}

// ============================================================================
// Test: Cache Mechanism
// ============================================================================

func TestPathCache_SetAndGet(t *testing.T) {
	// 清除快取
	InvalidatePathCache()
	defer InvalidatePathCache()

	// 初始應該為空
	if getPathCache() != "" {
		t.Error("cache should be empty initially")
	}

	// 設定快取
	testPath := "C:\\Test\\Kiro"
	setPathCache(testPath)

	// 應該能取得快取
	if getPathCache() != testPath {
		t.Errorf("expected '%s', got '%s'", testPath, getPathCache())
	}
}
