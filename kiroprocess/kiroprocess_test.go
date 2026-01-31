package kiroprocess

import (
	"testing"
)

// TestProcessInfo_ExePath 測試 ProcessInfo 結構應包含 ExePath 欄位
func TestProcessInfo_ExePath(t *testing.T) {
	// 建立 ProcessInfo 並設定 ExePath
	info := ProcessInfo{
		PID:     12345,
		Name:    "Kiro.exe",
		ExePath: "C:\\Users\\Test\\AppData\\Local\\Programs\\Kiro\\Kiro.exe",
	}

	// 驗證 ExePath 欄位存在且正確
	if info.ExePath == "" {
		t.Error("ProcessInfo.ExePath should not be empty")
	}

	expected := "C:\\Users\\Test\\AppData\\Local\\Programs\\Kiro\\Kiro.exe"
	if info.ExePath != expected {
		t.Errorf("ProcessInfo.ExePath = %q, want %q", info.ExePath, expected)
	}
}

// TestGetKiroExecutablePath_NotRunning 測試 Kiro 未運行時應返回 ErrProcessNotFound
func TestGetKiroExecutablePath_NotRunning(t *testing.T) {
	// 假設測試環境中 Kiro 未運行
	// 如果 Kiro 正在運行，此測試會跳過
	if IsKiroRunning() {
		t.Skip("Kiro is running, skipping this test")
	}

	path, err := GetKiroExecutablePath()

	// 應該返回 ErrProcessNotFound
	if err != ErrProcessNotFound {
		t.Errorf("GetKiroExecutablePath() error = %v, want %v", err, ErrProcessNotFound)
	}

	// 路徑應該為空
	if path != "" {
		t.Errorf("GetKiroExecutablePath() path = %q, want empty string", path)
	}
}

// TestGetKiroExecutablePath_Running 測試 Kiro 運行時應返回有效路徑
// 此測試僅在 Kiro 運行時執行
func TestGetKiroExecutablePath_Running(t *testing.T) {
	if !IsKiroRunning() {
		t.Skip("Kiro is not running, skipping this test")
	}

	path, err := GetKiroExecutablePath()

	if err != nil {
		t.Errorf("GetKiroExecutablePath() error = %v, want nil", err)
	}

	if path == "" {
		t.Error("GetKiroExecutablePath() returned empty path when Kiro is running")
	}

	// 路徑應該包含 Kiro
	t.Logf("Found Kiro executable path: %s", path)
}
