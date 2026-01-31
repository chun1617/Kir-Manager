// Package oauthlogin 提供 OAuth 登入功能的核心類型和錯誤定義
package oauthlogin

import "time"

// 錯誤碼常數定義
const (
	// ErrCodeTimeout 登入超時
	ErrCodeTimeout = "timeout"
	// ErrCodeCancelled 用戶取消
	ErrCodeCancelled = "cancelled"
	// ErrCodeInvalidCode 授權碼無效
	ErrCodeInvalidCode = "invalid_code"
	// ErrCodeAuthFailed 認證失敗
	ErrCodeAuthFailed = "auth_failed"
	// ErrCodeServerError 伺服器錯誤
	ErrCodeServerError = "server_error"
	// ErrCodeNetworkError 網路錯誤
	ErrCodeNetworkError = "network_error"
	// ErrCodeStateMismatch State 不匹配
	ErrCodeStateMismatch = "state_mismatch"
)

// Provider 常數定義
const (
	// ProviderGithub GitHub 提供者
	ProviderGithub = "Github"
	// ProviderGoogle Google 提供者
	ProviderGoogle = "Google"
	// ProviderBuilderID AWS Builder ID 提供者
	ProviderBuilderID = "BuilderID"
)

// AuthMethod 常數定義
const (
	// AuthMethodSocial Social 登入方式
	AuthMethodSocial = "social"
	// AuthMethodIdC AWS Identity Center 登入方式
	AuthMethodIdC = "idc"
)

// OAuthError 統一錯誤類型
// 包含錯誤碼和錯誤訊息，實作 error 介面
type OAuthError struct {
	// Code 錯誤碼，使用預定義的 ErrCode* 常數
	Code string
	// Message 錯誤訊息，提供人類可讀的錯誤描述
	Message string
}

// Error 實作 error 介面
func (e *OAuthError) Error() string {
	return e.Message
}

// LoginResult 登入結果結構
// 包含 OAuth 登入成功後的所有相關資訊
type LoginResult struct {
	// AccessToken 存取令牌
	AccessToken string
	// RefreshToken 刷新令牌
	RefreshToken string
	// ExpiresIn 有效期（秒）
	ExpiresIn int
	// ExpiresAt 過期時間
	ExpiresAt time.Time
	// ProfileArn AWS Profile ARN (Social 登入)
	ProfileArn string
	// Provider 提供者 (Github/Google/BuilderID)
	Provider string
	// AuthMethod 認證方式 (social/idc)
	AuthMethod string
	// ClientId IdC 客戶端 ID (僅 IdC)
	ClientId string
	// ClientSecret IdC 客戶端密鑰 (僅 IdC)
	ClientSecret string
	// ClientIdHash IdC 客戶端 ID 雜湊 (僅 IdC)
	ClientIdHash string
}
