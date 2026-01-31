package deeplink

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// 全域回調 channel 和同步機制
var (
	callbackChan chan *DeepLinkResult
	callbackOnce sync.Once
	callbackMu   sync.Mutex
)

// 冷啟動時的待處理 deep link 結果
var (
	pendingDeepLink *DeepLinkResult
	pendingMu       sync.Mutex
)

// SetPendingDeepLink 設定待處理的 deep link 結果（冷啟動用）
func SetPendingDeepLink(result *DeepLinkResult) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	pendingDeepLink = result
}

// GetPendingDeepLink 取得待處理的 deep link 結果
func GetPendingDeepLink() *DeepLinkResult {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	return pendingDeepLink
}

// clearPendingDeepLink 清除待處理的 deep link 結果
func clearPendingDeepLink() {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	pendingDeepLink = nil
}

// InitCallbackChannel 初始化回調 channel
// 確保只初始化一次
func InitCallbackChannel() {
	callbackOnce.Do(func() {
		callbackChan = make(chan *DeepLinkResult, 1)
	})
}

// SendCallback 發送回調結果到 channel
// 非阻塞發送，如果 channel 已滿則丟棄舊的並發送新的
// 若 channel 未初始化（冷啟動場景），保存到 pending
func SendCallback(result *DeepLinkResult) {
	callbackMu.Lock()
	defer callbackMu.Unlock()

	// 若 channel 未初始化，保存到 pending（冷啟動場景）
	if callbackChan == nil {
		SetPendingDeepLink(result)
		return
	}

	select {
	case callbackChan <- result:
	default:
		// channel 已滿，丟棄舊的結果
		select {
		case <-callbackChan:
		default:
		}
		callbackChan <- result
	}
}

// WaitForCallback 等待回調結果 (帶超時)
// 返回結果或超時錯誤
// 優先檢查 pending 結果（冷啟動場景）
func WaitForCallback(timeout time.Duration) (*DeepLinkResult, error) {
	// 先檢查是否有 pending 結果（冷啟動場景）
	if pending := GetPendingDeepLink(); pending != nil {
		clearPendingDeepLink()
		return pending, nil
	}

	InitCallbackChannel()

	select {
	case result := <-callbackChan:
		return result, nil
	case <-time.After(timeout):
		return nil, ErrCallbackTimeout
	}
}

// ResetCallbackChannel 重置回調 channel (用於測試)
func ResetCallbackChannel() {
	callbackMu.Lock()
	defer callbackMu.Unlock()

	if callbackChan != nil {
		close(callbackChan)
	}
	callbackChan = nil
	callbackOnce = sync.Once{}
}

// DeepLinkResult 定義 Deep Link 解析結果
type DeepLinkResult struct {
	Code  string
	State string
}

// DeepLinkError 定義 Deep Link 錯誤
type DeepLinkError struct {
	Error       string
	Description string
}

// ParseDeepLinkURL 解析 deep link URL
// URL 格式: kiro://kiro.kiroAgent/authenticate-success?code=xxx&state=yyy
func ParseDeepLinkURL(rawURL string) (*DeepLinkResult, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, ErrInvalidScheme
	}

	// 驗證 scheme
	if strings.ToLower(parsedURL.Scheme) != URLScheme {
		return nil, ErrInvalidScheme
	}

	// 取得查詢參數
	query := parsedURL.Query()

	code := query.Get("code")
	if code == "" {
		return nil, ErrMissingCode
	}

	state := query.Get("state")
	if state == "" {
		return nil, ErrStateMismatch
	}

	return &DeepLinkResult{
		Code:  code,
		State: state,
	}, nil
}

// ValidateDeepLinkURL 驗證 URL 格式是否正確
func ValidateDeepLinkURL(rawURL string) bool {
	result, err := ParseDeepLinkURL(rawURL)
	if err != nil {
		return false
	}
	return result.Code != "" && result.State != ""
}

// HandleDeepLinkCallback 處理 deep link 回調
// 1. 先檢查是否有錯誤參數
// 2. 解析 URL
// 3. 載入持久化的 State
// 4. 驗證 State 匹配
// 5. 檢查 State 是否過期
// 6. 返回結果
func HandleDeepLinkCallback(rawURL string) (*DeepLinkResult, error) {
	// 1. 先檢查是否有錯誤參數
	if dlErr, hasError := ParseDeepLinkError(rawURL); hasError {
		return nil, fmt.Errorf("oauth error: %s - %s", dlErr.Error, dlErr.Description)
	}

	// 2. 解析 URL
	result, err := ParseDeepLinkURL(rawURL)
	if err != nil {
		return nil, err
	}

	// 3. 載入持久化的 State
	savedState, err := LoadState()
	if err != nil {
		return nil, err
	}

	// 4. 驗證 State 匹配
	if err := ValidateState(savedState, result.State); err != nil {
		return nil, err
	}

	// 5. 檢查 State 是否過期
	if IsStateExpired(savedState) {
		return nil, ErrStateExpired
	}

	// 6. 返回結果
	return result, nil
}

// ParseDeepLinkError 解析 URL 中的錯誤參數
// URL 格式: kiro://...?error=access_denied&error_description=...
func ParseDeepLinkError(rawURL string) (*DeepLinkError, bool) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, false
	}

	query := parsedURL.Query()

	errorCode := query.Get("error")
	if errorCode == "" {
		return nil, false
	}

	return &DeepLinkError{
		Error:       errorCode,
		Description: query.Get("error_description"),
	}, true
}
