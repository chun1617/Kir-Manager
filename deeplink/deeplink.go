// Package deeplink 提供 Deep Link URL Scheme 支援功能
// 用於處理 OAuth 回調和應用程式間通訊
package deeplink

import (
	"errors"
	"time"
)

// URL Scheme 相關常數
const (
	// URLScheme 定義應用程式的 URL Scheme
	URLScheme = "kiro"

	// RedirectURI 定義 OAuth 回調的完整 URI
	RedirectURI = "kiro://kiro.kiroAgent/authenticate-success"

	// StateFileName 定義 OAuth State 檔案名稱
	StateFileName = "kiro-manager-oauth-state.json"

	// StateExpiry 定義 OAuth State 的過期時間
	StateExpiry = 5 * time.Minute
)

// 錯誤類型定義
var (
	// ErrNotWindows 表示 Deep Link 功能僅支援 Windows 平台
	ErrNotWindows = errors.New("deep link only supported on Windows")

	// ErrRegistryFailed 表示 URL Scheme 註冊失敗
	ErrRegistryFailed = errors.New("failed to register URL scheme")

	// ErrStateNotFound 表示找不到 OAuth State
	ErrStateNotFound = errors.New("oauth state not found")

	// ErrStateExpired 表示 OAuth State 已過期
	ErrStateExpired = errors.New("oauth state expired")

	// ErrStateMismatch 表示 OAuth State 不匹配
	ErrStateMismatch = errors.New("oauth state mismatch")

	// ErrMissingCode 表示缺少授權碼
	ErrMissingCode = errors.New("missing authorization code")

	// ErrInvalidScheme 表示無效的 URL Scheme
	ErrInvalidScheme = errors.New("invalid URL scheme")

	// ErrCallbackTimeout 表示回調超時
	ErrCallbackTimeout = errors.New("callback timeout")
)
