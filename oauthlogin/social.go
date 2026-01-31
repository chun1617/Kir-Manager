// Package oauthlogin 提供 OAuth 登入功能
package oauthlogin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// SocialProvider Social 登入提供者類型
type SocialProvider string

// Social 登入提供者常數
const (
	// SocialProviderGithub GitHub 提供者
	SocialProviderGithub SocialProvider = "Github"
	// SocialProviderGoogle Google 提供者
	SocialProviderGoogle SocialProvider = "Google"
)

// OAuth 端點常數
const (
	// AuthBaseURL 授權基礎 URL
	AuthBaseURL = "https://prod.us-east-1.auth.desktop.kiro.dev"
	// AuthorizePath 授權路徑
	AuthorizePath = "/login"
	// TokenPath Token 交換路徑
	TokenPath = "/oauth/token"
)

// SocialLoginConfig Social 登入配置結構
type SocialLoginConfig struct {
	// Provider 登入提供者 (Github/Google)
	Provider string
	// Port 本地回調伺服器端口
	Port int
	// RedirectURI 自定義回調 URI（可選，若為空則自動生成）
	RedirectURI string
}

// SocialTokenResponse Token 回應結構
type SocialTokenResponse struct {
	// AccessToken 存取令牌
	AccessToken string `json:"accessToken"`
	// RefreshToken 刷新令牌
	RefreshToken string `json:"refreshToken"`
	// ExpiresIn 有效期（秒）
	ExpiresIn int `json:"expiresIn"`
	// ProfileArn AWS Profile ARN
	ProfileArn string `json:"profileArn"`
}

// tokenExchangeRequest Token 交換請求結構
type tokenExchangeRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
}

// BuildAuthorizationURL 建構授權 URL
// 根據配置和 PKCE 參數生成完整的授權 URL
// 參數：
//   - config: Social 登入配置
//   - pkce: PKCE 參數
//
// 返回：完整的授權 URL 字串
func BuildAuthorizationURL(config SocialLoginConfig, pkce PKCEParams) string {
	// 建構基礎 URL
	baseURL := fmt.Sprintf("%s%s", AuthBaseURL, AuthorizePath)

	// 決定 redirect_uri
	redirectURI := config.RedirectURI
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("http://localhost:%d/callback", config.Port)
	}

	// 建構查詢參數
	params := url.Values{}
	params.Set("idp", config.Provider)
	params.Set("redirect_uri", redirectURI)
	params.Set("code_challenge", pkce.CodeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("state", pkce.State)

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// ExchangeToken 執行 Token 交換
// 使用授權碼和 PKCE 參數向伺服器交換 Token
// 參數：
//   - config: Social 登入配置
//   - code: 授權碼
//   - pkce: PKCE 參數
//
// 返回：Token 回應或錯誤
func ExchangeToken(config SocialLoginConfig, code string, pkce PKCEParams) (*SocialTokenResponse, error) {
	return ExchangeTokenWithClient(http.DefaultClient, config, code, pkce)
}

// ExchangeTokenWithClient 使用自定義 HTTP 客戶端執行 Token 交換
// 允許注入 HTTP 客戶端以便測試
func ExchangeTokenWithClient(client *http.Client, config SocialLoginConfig, code string, pkce PKCEParams) (*SocialTokenResponse, error) {
	tokenURL := fmt.Sprintf("%s%s", AuthBaseURL, TokenPath)
	return ExchangeTokenWithEndpoint(client, tokenURL, config, code, pkce)
}

// ExchangeTokenWithEndpoint 使用自定義端點執行 Token 交換
// 允許注入 HTTP 客戶端和端點 URL 以便測試
func ExchangeTokenWithEndpoint(client *http.Client, tokenURL string, config SocialLoginConfig, code string, pkce PKCEParams) (*SocialTokenResponse, error) {
	// 決定 redirect_uri
	redirectURI := config.RedirectURI
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("http://localhost:%d/callback", config.Port)
	}

	// 建構請求體
	reqBody := tokenExchangeRequest{
		Code:         code,
		CodeVerifier: pkce.CodeVerifier,
		RedirectURI:  redirectURI,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeNetworkError,
			Message: fmt.Sprintf("failed to marshal request: %v", err),
		}
	}

	// 建構 HTTP 請求（使用傳入的 tokenURL 參數）
	req, err := http.NewRequest(http.MethodPost, tokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeNetworkError,
			Message: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	req.Header.Set("Content-Type", "application/json")

	// 執行請求
	resp, err := client.Do(req)
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeNetworkError,
			Message: fmt.Sprintf("failed to send request: %v", err),
		}
	}
	defer resp.Body.Close()

	// 讀取回應體
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeNetworkError,
			Message: fmt.Sprintf("failed to read response: %v", err),
		}
	}

	// 處理 HTTP 錯誤
	if resp.StatusCode != http.StatusOK {
		return nil, mapHTTPError(resp.StatusCode, body)
	}

	// 解析回應
	var tokenResponse SocialTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return &tokenResponse, nil
}

// mapHTTPError 將 HTTP 狀態碼映射到 OAuthError
// 根據 Requirements 4.4, 4.5, 4.6 的規範：
//   - 400: 授權碼無效或已過期 (ErrCodeInvalidCode)
//   - 401: 認證失敗 (ErrCodeAuthFailed)
//   - 5xx: 伺服器錯誤 (ErrCodeServerError)
func mapHTTPError(statusCode int, body []byte) *OAuthError {
	switch {
	case statusCode == http.StatusBadRequest:
		return &OAuthError{
			Code:    ErrCodeInvalidCode,
			Message: fmt.Sprintf("invalid or expired authorization code: %s", string(body)),
		}
	case statusCode == http.StatusUnauthorized:
		return &OAuthError{
			Code:    ErrCodeAuthFailed,
			Message: fmt.Sprintf("authentication failed: %s", string(body)),
		}
	case statusCode >= 500:
		return &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("server error (status %d): %s", statusCode, string(body)),
		}
	default:
		return &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("unexpected error (status %d): %s", statusCode, string(body)),
		}
	}
}
