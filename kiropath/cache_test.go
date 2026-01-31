package kiropath

import (
	"sync"
	"testing"
)

// TestGetPathCache_Empty 測試初始狀態應返回空字串
func TestGetPathCache_Empty(t *testing.T) {
	// 確保測試前清除快取
	InvalidatePathCache()

	got := getPathCache()
	if got != "" {
		t.Errorf("getPathCache() = %q, want empty string", got)
	}
}

// TestSetPathCache_AndGet 測試設定後應能取得
func TestSetPathCache_AndGet(t *testing.T) {
	// 確保測試前清除快取
	InvalidatePathCache()

	testPath := "/test/kiro/path"
	setPathCache(testPath)

	got := getPathCache()
	if got != testPath {
		t.Errorf("getPathCache() = %q, want %q", got, testPath)
	}

	// 清理
	InvalidatePathCache()
}

// TestInvalidatePathCache 測試清除後應返回空字串
func TestInvalidatePathCache(t *testing.T) {
	// 先設定一個值
	testPath := "/test/kiro/path"
	setPathCache(testPath)

	// 確認已設定
	if got := getPathCache(); got != testPath {
		t.Fatalf("setPathCache failed, got %q, want %q", got, testPath)
	}

	// 清除快取
	InvalidatePathCache()

	// 確認已清除
	got := getPathCache()
	if got != "" {
		t.Errorf("after InvalidatePathCache(), getPathCache() = %q, want empty string", got)
	}
}

// TestPathCache_Concurrency 測試並發安全
func TestPathCache_Concurrency(t *testing.T) {
	// 確保測試前清除快取
	InvalidatePathCache()

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 3) // 3 種操作

	// 並發寫入
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				setPathCache("/path/from/goroutine")
			}
		}(i)
	}

	// 並發讀取
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = getPathCache()
			}
		}(i)
	}

	// 並發清除
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				InvalidatePathCache()
			}
		}(i)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	// 如果沒有 panic 或 race condition，測試通過
	// 清理
	InvalidatePathCache()
}

// TestPathCache_SetEmptyString 測試設定空字串的行為
func TestPathCache_SetEmptyString(t *testing.T) {
	// 先設定一個有效值
	setPathCache("/valid/path")

	// 設定空字串應該等同於清除
	setPathCache("")

	got := getPathCache()
	if got != "" {
		t.Errorf("after setPathCache(\"\"), getPathCache() = %q, want empty string", got)
	}
}
