package kiropath

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestSearchInPath_NotFound 測試 PATH 中沒有 Kiro 時應返回錯誤
func TestSearchInPath_NotFound(t *testing.T) {
	// 保存原始 PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// 設定一個不包含 Kiro 的 PATH
	tempDir := t.TempDir()
	os.Setenv("PATH", tempDir)

	_, err := searchInPath()
	if err == nil {
		t.Error("searchInPath() should return error when Kiro not in PATH")
	}
}

// TestSearchInPath_EmptyPath 測試 PATH 為空時應返回錯誤
func TestSearchInPath_EmptyPath(t *testing.T) {
	// 保存原始 PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// 設定空 PATH
	os.Setenv("PATH", "")

	_, err := searchInPath()
	if err == nil {
		t.Error("searchInPath() should return error when PATH is empty")
	}
}

// TestSearchInPath_Found 測試在 PATH 中找到 Kiro 執行檔
func TestSearchInPath_Found(t *testing.T) {
	// 保存原始 PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// 建立臨時目錄結構模擬 Kiro 安裝
	tempDir := t.TempDir()
	var execName string
	if runtime.GOOS == "windows" {
		execName = "Kiro.exe"
	} else {
		execName = "kiro"
	}

	// 建立假的執行檔
	execPath := filepath.Join(tempDir, execName)
	if err := os.WriteFile(execPath, []byte("fake executable"), 0755); err != nil {
		t.Fatalf("Failed to create fake executable: %v", err)
	}

	// 設定 PATH 包含臨時目錄
	os.Setenv("PATH", tempDir)

	result, err := searchInPath()
	if err != nil {
		t.Errorf("searchInPath() returned error: %v", err)
	}
	if result != tempDir {
		t.Errorf("searchInPath() = %q, want %q", result, tempDir)
	}
}

// TestSearchInPath_MultipleDirectories 測試 PATH 包含多個目錄時的搜索
func TestSearchInPath_MultipleDirectories(t *testing.T) {
	// 保存原始 PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// 建立多個臨時目錄
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()
	tempDir3 := t.TempDir()

	var execName string
	if runtime.GOOS == "windows" {
		execName = "Kiro.exe"
	} else {
		execName = "kiro"
	}

	// 只在第二個目錄建立執行檔
	execPath := filepath.Join(tempDir2, execName)
	if err := os.WriteFile(execPath, []byte("fake executable"), 0755); err != nil {
		t.Fatalf("Failed to create fake executable: %v", err)
	}

	// 設定 PATH 包含多個目錄
	pathSeparator := string(os.PathListSeparator)
	newPath := tempDir1 + pathSeparator + tempDir2 + pathSeparator + tempDir3
	os.Setenv("PATH", newPath)

	result, err := searchInPath()
	if err != nil {
		t.Errorf("searchInPath() returned error: %v", err)
	}
	if result != tempDir2 {
		t.Errorf("searchInPath() = %q, want %q", result, tempDir2)
	}
}

// TestSearchInPath_FirstMatchWins 測試多個目錄都有 Kiro 時返回第一個
func TestSearchInPath_FirstMatchWins(t *testing.T) {
	// 保存原始 PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// 建立多個臨時目錄
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	var execName string
	if runtime.GOOS == "windows" {
		execName = "Kiro.exe"
	} else {
		execName = "kiro"
	}

	// 在兩個目錄都建立執行檔
	for _, dir := range []string{tempDir1, tempDir2} {
		execPath := filepath.Join(dir, execName)
		if err := os.WriteFile(execPath, []byte("fake executable"), 0755); err != nil {
			t.Fatalf("Failed to create fake executable: %v", err)
		}
	}

	// 設定 PATH
	pathSeparator := string(os.PathListSeparator)
	newPath := tempDir1 + pathSeparator + tempDir2
	os.Setenv("PATH", newPath)

	result, err := searchInPath()
	if err != nil {
		t.Errorf("searchInPath() returned error: %v", err)
	}
	// 應該返回第一個找到的目錄
	if result != tempDir1 {
		t.Errorf("searchInPath() = %q, want %q (first match)", result, tempDir1)
	}
}

// TestSearchInPath_NonExistentDirectory 測試 PATH 包含不存在的目錄時應跳過
func TestSearchInPath_NonExistentDirectory(t *testing.T) {
	// 保存原始 PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// 建立一個有效的臨時目錄
	tempDir := t.TempDir()

	var execName string
	if runtime.GOOS == "windows" {
		execName = "Kiro.exe"
	} else {
		execName = "kiro"
	}

	// 建立執行檔
	execPath := filepath.Join(tempDir, execName)
	if err := os.WriteFile(execPath, []byte("fake executable"), 0755); err != nil {
		t.Fatalf("Failed to create fake executable: %v", err)
	}

	// 設定 PATH 包含不存在的目錄和有效目錄
	pathSeparator := string(os.PathListSeparator)
	nonExistentDir := filepath.Join(t.TempDir(), "nonexistent_subdir")
	newPath := nonExistentDir + pathSeparator + tempDir
	os.Setenv("PATH", newPath)

	result, err := searchInPath()
	if err != nil {
		t.Errorf("searchInPath() returned error: %v", err)
	}
	if result != tempDir {
		t.Errorf("searchInPath() = %q, want %q", result, tempDir)
	}
}
