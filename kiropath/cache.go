package kiropath

import (
	"sync"
)

var (
	// pathCache 儲存已偵測到的 Kiro 安裝路徑
	pathCache string
	// pathCacheMu 保護 pathCache 的讀寫鎖
	pathCacheMu sync.RWMutex
	// cacheValid 標記快取是否有效
	cacheValid bool
)

// getPathCache 取得快取的路徑
// 如果快取無效或為空，返回空字串
func getPathCache() string {
	pathCacheMu.RLock()
	defer pathCacheMu.RUnlock()

	if !cacheValid {
		return ""
	}
	return pathCache
}

// setPathCache 設定快取的路徑
// 如果傳入空字串，等同於清除快取
func setPathCache(path string) {
	pathCacheMu.Lock()
	defer pathCacheMu.Unlock()

	if path == "" {
		pathCache = ""
		cacheValid = false
		return
	}

	pathCache = path
	cacheValid = true
}

// InvalidatePathCache 清除路徑快取
// 此函數為公開函數，供 settings 模組在設定變更時調用
func InvalidatePathCache() {
	pathCacheMu.Lock()
	defer pathCacheMu.Unlock()

	pathCache = ""
	cacheValid = false
}
