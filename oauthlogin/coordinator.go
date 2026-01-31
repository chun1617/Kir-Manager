// Package oauthlogin 提供 OAuth 登入功能
package oauthlogin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"kiro-manager/deeplink"
)

// SocialLoginCoordinatorConfig Social 登入協調器配置
type SocialLoginCoordinatorConfig struct {
	// Provider 登入提供者 (Github/Google)
	Provider string
	// TokenURL 自定義 Token 端點 URL（用於測試）
	TokenURL string
	// Timeout 登入超時時間
	Timeout time.Duration
	// OpenBrowser 是否自動開啟瀏覽器
	OpenBrowser bool
	// HTTPClient 自定義 HTTP 客戶端（用於測試）
	HTTPClient *http.Client
}

// IdCLoginCoordinatorConfig IdC 登入協調器配置
type IdCLoginCoordinatorConfig struct {
	// StartURL IdC 起始 URL
	StartURL string
	// ClientName 客戶端名稱
	ClientName string
	// RegisterURL 自定義註冊端點 URL（用於測試）
	RegisterURL string
	// DeviceAuthURL 自定義設備授權端點 URL（用於測試）
	DeviceAuthURL string
	// TokenURL 自定義 Token 端點 URL（用於測試）
	TokenURL string
	// Timeout 登入超時時間
	Timeout time.Duration
	// OpenBrowser 是否自動開啟瀏覽器
	OpenBrowser bool
	// HTTPClient 自定義 HTTP 客戶端（用於測試）
	HTTPClient *http.Client
}

// openBrowser 跨平台開啟瀏覽器
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// 使用 rundll32 避免 URL 中的 & 被 cmd 解釋為命令分隔符
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux 和其他
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

// SocialLogin 執行 Social 登入流程
// 整合 PKCE、Callback Server、Token 交換和瀏覽器開啟邏輯
// 參數：
//   - ctx: context，用於取消操作
//   - config: Social 登入協調器配置
//
// 返回：登入結果或錯誤
func SocialLogin(ctx context.Context, config SocialLoginCoordinatorConfig) (*LoginResult, error) {
	// 1. 生成 PKCE 參數
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("failed to generate PKCE: %v", err),
		}
	}

	// 2. 啟動本地 Callback Server
	callbackServer := NewCallbackServer(pkce.State)
	port, err := callbackServer.Start()
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("failed to start callback server: %v", err),
		}
	}
	defer callbackServer.Stop()

	// 3. 建構授權 URL
	socialConfig := SocialLoginConfig{
		Provider: config.Provider,
		Port:     port,
	}
	authURL := BuildAuthorizationURL(socialConfig, *pkce)

	// 4. 開啟瀏覽器
	if config.OpenBrowser {
		if err := openBrowser(authURL); err != nil {
			return nil, &OAuthError{
				Code:    ErrCodeServerError,
				Message: fmt.Sprintf("failed to open browser: %v", err),
			}
		}
	}

	// 5. 等待回調
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	callbackResult, err := callbackServer.WaitForCallback(timeout)
	if err != nil {
		return nil, err
	}

	// 6. 驗證 state 參數
	if !ValidateState(pkce.State, callbackResult.State) {
		return nil, &OAuthError{
			Code:    ErrCodeStateMismatch,
			Message: "state parameter mismatch",
		}
	}

	// 7. 執行 Token 交換
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = fmt.Sprintf("%s%s", AuthBaseURL, TokenPath)
	}

	tokenResp, err := ExchangeTokenWithEndpoint(httpClient, tokenURL, socialConfig, callbackResult.Code, *pkce)
	if err != nil {
		return nil, err
	}

	// 8. 建構並返回 LoginResult
	return &LoginResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		ProfileArn:   tokenResp.ProfileArn,
		Provider:     config.Provider,
		AuthMethod:   AuthMethodSocial,
	}, nil
}

// SocialLoginWithDeepLink 使用 Deep Link 執行 Social 登入
// 此函數用於 Windows 平台，使用 kiro:// URL Scheme 作為 redirect_uri
// 參數：
//   - ctx: context，用於取消操作
//   - config: Social 登入協調器配置
//
// 返回：登入結果或錯誤
func SocialLoginWithDeepLink(ctx context.Context, config SocialLoginCoordinatorConfig) (*LoginResult, error) {
	// 1. 生成 PKCE 參數
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("failed to generate PKCE: %v", err),
		}
	}

	// 2. 持久化 State 到臨時檔案
	oauthState := &deeplink.OAuthState{
		State:         pkce.State,
		Provider:      config.Provider,
		CodeVerifier:  pkce.CodeVerifier,
		CodeChallenge: pkce.CodeChallenge,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(deeplink.StateExpiry),
	}
	if err := deeplink.SaveState(oauthState); err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("failed to save state: %v", err),
		}
	}

	// 3. 建構授權 URL (使用 kiro:// redirect_uri)
	socialConfig := SocialLoginConfig{
		Provider:    config.Provider,
		RedirectURI: deeplink.RedirectURI,
	}
	authURL := BuildAuthorizationURL(socialConfig, *pkce)

	// 4. 開啟瀏覽器
	if config.OpenBrowser {
		if err := openBrowser(authURL); err != nil {
			deeplink.ClearState()
			return nil, &OAuthError{
				Code:    ErrCodeServerError,
				Message: fmt.Sprintf("failed to open browser: %v", err),
			}
		}
	}

	// 5. 等待 deep link 回調
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	callbackResult, err := deeplink.WaitForCallback(timeout)
	if err != nil {
		deeplink.ClearState()
		if err == deeplink.ErrCallbackTimeout {
			return nil, &OAuthError{
				Code:    ErrCodeTimeout,
				Message: "login timeout",
			}
		}
		return nil, &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("callback error: %v", err),
		}
	}

	// 6. 清理臨時檔案
	defer deeplink.ClearState()

	// 7. 執行 Token 交換
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = fmt.Sprintf("%s%s", AuthBaseURL, TokenPath)
	}

	// 使用持久化的 PKCE 參數
	savedPKCE := PKCEParams{
		CodeVerifier:  oauthState.CodeVerifier,
		CodeChallenge: oauthState.CodeChallenge,
		State:         oauthState.State,
	}

	tokenResp, err := ExchangeTokenWithEndpoint(httpClient, tokenURL, socialConfig, callbackResult.Code, savedPKCE)
	if err != nil {
		return nil, err
	}

	// 8. 建構並返回 LoginResult
	return &LoginResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		ProfileArn:   tokenResp.ProfileArn,
		Provider:     config.Provider,
		AuthMethod:   AuthMethodSocial,
	}, nil
}

// SocialLoginWithSimulatedCallback 使用模擬回調執行 Social 登入（用於測試）
// 此函數跳過實際的瀏覽器授權流程，直接使用提供的授權碼
func SocialLoginWithSimulatedCallback(ctx context.Context, config SocialLoginCoordinatorConfig, authCode string) (*LoginResult, error) {
	// 1. 生成 PKCE 參數
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("failed to generate PKCE: %v", err),
		}
	}

	// 2. 建構 Social 配置
	socialConfig := SocialLoginConfig{
		Provider:    config.Provider,
		Port:        8080, // 測試用固定端口
		RedirectURI: "http://localhost:8080/callback",
	}

	// 3. 執行 Token 交換
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = fmt.Sprintf("%s%s", AuthBaseURL, TokenPath)
	}

	tokenResp, err := ExchangeTokenWithEndpoint(httpClient, tokenURL, socialConfig, authCode, *pkce)
	if err != nil {
		return nil, err
	}

	// 4. 建構並返回 LoginResult
	return &LoginResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		ProfileArn:   tokenResp.ProfileArn,
		Provider:     config.Provider,
		AuthMethod:   AuthMethodSocial,
	}, nil
}

// SocialLoginWithMismatchedState 模擬 State 不匹配的情況（用於測試）
func SocialLoginWithMismatchedState(ctx context.Context, config SocialLoginCoordinatorConfig) (*LoginResult, error) {
	// 直接返回 State 不匹配錯誤
	return nil, &OAuthError{
		Code:    ErrCodeStateMismatch,
		Message: "state parameter mismatch",
	}
}

// IdCLogin 執行 IdC 登入流程
// 整合設備註冊、設備授權、Token 輪詢和瀏覽器開啟邏輯
// 參數：
//   - ctx: context，用於取消操作
//   - config: IdC 登入協調器配置
//
// 返回：登入結果或錯誤
func IdCLogin(ctx context.Context, config IdCLoginCoordinatorConfig) (*LoginResult, error) {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// 設定超時
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	// 建立帶超時的 context
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 1. 註冊設備客戶端
	registerURL := config.RegisterURL
	if registerURL == "" {
		registerURL = IdCRegisterURL
	}

	clientName := config.ClientName
	if clientName == "" {
		clientName = "Kiro Manager"
	}

	creds, err := RegisterDeviceClientWithEndpoint(httpClient, registerURL, clientName, config.StartURL)
	if err != nil {
		return nil, err
	}

	// 2. 啟動設備授權
	deviceAuthURL := config.DeviceAuthURL
	if deviceAuthURL == "" {
		deviceAuthURL = IdCDeviceAuthURL
	}

	authResp, err := StartDeviceAuthorizationWithEndpoint(httpClient, deviceAuthURL, creds, config.StartURL)
	if err != nil {
		return nil, err
	}

	// 3. 開啟瀏覽器至 verificationUriComplete
	if config.OpenBrowser {
		if err := openBrowser(authResp.VerificationUriComplete); err != nil {
			return nil, &OAuthError{
				Code:    ErrCodeServerError,
				Message: fmt.Sprintf("failed to open browser: %v", err),
			}
		}
	}

	// 4. 輪詢 Token
	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = IdCTokenURL
	}

	tokenResp, err := PollForTokenWithEndpoint(ctx, httpClient, tokenURL, creds, authResp)
	if err != nil {
		return nil, err
	}

	// 5. 計算 ClientIdHash
	hash := sha256.Sum256([]byte(creds.ClientId))
	clientIdHash := hex.EncodeToString(hash[:])

	// 6. 建構並返回 LoginResult
	return &LoginResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		Provider:     ProviderBuilderID,
		AuthMethod:   AuthMethodIdC,
		ClientId:     creds.ClientId,
		ClientSecret: creds.ClientSecret,
		ClientIdHash: clientIdHash,
	}, nil
}
