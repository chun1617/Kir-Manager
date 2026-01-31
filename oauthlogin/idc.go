// Package oauthlogin 提供 OAuth 登入功能
package oauthlogin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// IdC API 端點常數
const (
	// IdCRegisterURL 設備註冊端點
	IdCRegisterURL = "https://oidc.us-east-1.amazonaws.com/client/register"
	// IdCDeviceAuthURL 設備授權端點
	IdCDeviceAuthURL = "https://oidc.us-east-1.amazonaws.com/device_authorization"
	// IdCTokenURL Token 端點
	IdCTokenURL = "https://oidc.us-east-1.amazonaws.com/token"
)

// IdC 錯誤碼常數
const (
	// IdCErrAuthorizationPending 授權等待中
	IdCErrAuthorizationPending = "authorization_pending"
	// IdCErrAccessDenied 存取被拒絕
	IdCErrAccessDenied = "access_denied"
	// IdCErrExpiredToken Token 已過期
	IdCErrExpiredToken = "expired_token"
	// IdCErrSlowDown 請求過於頻繁
	IdCErrSlowDown = "slow_down"
)

// IdCClientCredentials IdC 客戶端憑證結構
type IdCClientCredentials struct {
	// ClientId 客戶端 ID
	ClientId string `json:"clientId"`
	// ClientSecret 客戶端密鑰
	ClientSecret string `json:"clientSecret"`
}

// DeviceRegistrationRequest 設備註冊請求結構
type DeviceRegistrationRequest struct {
	// ClientName 客戶端名稱
	ClientName string `json:"clientName"`
	// ClientType 客戶端類型（固定為 "public"）
	ClientType string `json:"clientType"`
	// GrantTypes 授權類型列表
	GrantTypes []string `json:"grantTypes"`
	// IssuerUrl 發行者 URL
	IssuerUrl string `json:"issuerUrl"`
}

// DeviceAuthorizationRequest 設備授權請求結構
type DeviceAuthorizationRequest struct {
	// ClientId 客戶端 ID
	ClientId string `json:"clientId"`
	// ClientSecret 客戶端密鑰
	ClientSecret string `json:"clientSecret"`
	// StartUrl 起始 URL
	StartUrl string `json:"startUrl"`
}

// DeviceAuthorizationResponse 設備授權回應結構
type DeviceAuthorizationResponse struct {
	// DeviceCode 設備碼
	DeviceCode string `json:"deviceCode"`
	// UserCode 用戶碼
	UserCode string `json:"userCode"`
	// VerificationUri 驗證 URI
	VerificationUri string `json:"verificationUri"`
	// VerificationUriComplete 完整驗證 URI（含用戶碼）
	VerificationUriComplete string `json:"verificationUriComplete"`
	// ExpiresIn 過期時間（秒）
	ExpiresIn int `json:"expiresIn"`
	// Interval 輪詢間隔（秒）
	Interval int `json:"interval"`
}

// TokenPollingRequest Token 輪詢請求結構
type TokenPollingRequest struct {
	// ClientId 客戶端 ID
	ClientId string `json:"clientId"`
	// ClientSecret 客戶端密鑰
	ClientSecret string `json:"clientSecret"`
	// GrantType 授權類型
	GrantType string `json:"grantType"`
	// DeviceCode 設備碼
	DeviceCode string `json:"deviceCode"`
}

// IdCTokenResponse IdC Token 回應結構
type IdCTokenResponse struct {
	// AccessToken 存取令牌
	AccessToken string `json:"accessToken"`
	// RefreshToken 刷新令牌
	RefreshToken string `json:"refreshToken"`
	// IdToken ID 令牌
	IdToken string `json:"idToken"`
	// ExpiresIn 有效期（秒）
	ExpiresIn int `json:"expiresIn"`
}

// IdCErrorResponse IdC 錯誤回應結構
type IdCErrorResponse struct {
	// Error 錯誤碼
	Error string `json:"error"`
	// ErrorDescription 錯誤描述
	ErrorDescription string `json:"error_description,omitempty"`
}

// setIdCHeaders 設定 IdC API 請求的標準 Headers
func setIdCHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", "oidc.us-east-1.amazonaws.com")
	req.Header.Set("x-amz-user-agent", "aws-sdk-js/3.738.0 ua/2.1 os/other lang/js api/sso-oidc#3.738.0 m/E KiroIDE")
	req.Header.Set("User-Agent", "node")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")
}

// RegisterDeviceClient 執行設備註冊
// 向 IdC 註冊���備客戶端，取得 clientId 和 clientSecret
// 參數：
//   - clientName: 客戶端名稱（例如 "Kiro Manager"）
//   - issuerUrl: 發行者 URL（例如 "https://view.awsapps.com/start"）
//
// 返回：客戶端憑證或錯誤
func RegisterDeviceClient(clientName, issuerUrl string) (*IdCClientCredentials, error) {
	return RegisterDeviceClientWithClient(http.DefaultClient, clientName, issuerUrl)
}

// RegisterDeviceClientWithClient 使用自定義 HTTP 客戶端執行設備註冊
func RegisterDeviceClientWithClient(client *http.Client, clientName, issuerUrl string) (*IdCClientCredentials, error) {
	return RegisterDeviceClientWithEndpoint(client, IdCRegisterURL, clientName, issuerUrl)
}

// RegisterDeviceClientWithEndpoint 使用自定義端點執行設備註冊
// 允許注入 HTTP 客戶端和端點 URL 以便測試
func RegisterDeviceClientWithEndpoint(client *http.Client, endpoint, clientName, issuerUrl string) (*IdCClientCredentials, error) {
	// 建構請求體
	reqBody := DeviceRegistrationRequest{
		ClientName: clientName,
		ClientType: "public",
		GrantTypes: []string{"device_code", "refresh_token"},
		IssuerUrl:  issuerUrl,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeNetworkError,
			Message: fmt.Sprintf("failed to marshal request: %v", err),
		}
	}

	// 建構 HTTP 請求
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeNetworkError,
			Message: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	setIdCHeaders(req)

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
	var creds IdCClientCredentials
	if err := json.Unmarshal(body, &creds); err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return &creds, nil
}

// StartDeviceAuthorization 啟動設備授權流程
// 使用客戶端憑證向 IdC 請求設備授權
// 參數：
//   - creds: 客戶端憑證
//   - startUrl: 起始 URL
//
// 返回：設備授權回應或錯誤
func StartDeviceAuthorization(creds *IdCClientCredentials, startUrl string) (*DeviceAuthorizationResponse, error) {
	return StartDeviceAuthorizationWithClient(http.DefaultClient, creds, startUrl)
}

// StartDeviceAuthorizationWithClient 使用自定義 HTTP 客戶端啟動設備授權
func StartDeviceAuthorizationWithClient(client *http.Client, creds *IdCClientCredentials, startUrl string) (*DeviceAuthorizationResponse, error) {
	return StartDeviceAuthorizationWithEndpoint(client, IdCDeviceAuthURL, creds, startUrl)
}

// StartDeviceAuthorizationWithEndpoint 使用自定義端點啟動設備授權
// 允許注入 HTTP 客戶端和端點 URL 以便測試
func StartDeviceAuthorizationWithEndpoint(client *http.Client, endpoint string, creds *IdCClientCredentials, startUrl string) (*DeviceAuthorizationResponse, error) {
	// 建構請求體
	reqBody := DeviceAuthorizationRequest{
		ClientId:     creds.ClientId,
		ClientSecret: creds.ClientSecret,
		StartUrl:     startUrl,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeNetworkError,
			Message: fmt.Sprintf("failed to marshal request: %v", err),
		}
	}

	// 建構 HTTP 請求
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeNetworkError,
			Message: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	setIdCHeaders(req)

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
	var authResp DeviceAuthorizationResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return &authResp, nil
}

// PollForToken 輪詢 Token
// 使用設備碼輪詢 IdC 以取得 Token
// 參數：
//   - ctx: context，用於取消操作
//   - creds: 客戶端憑證
//   - authResp: 設備授權回應
//
// 返回：Token 回應或錯誤
func PollForToken(ctx context.Context, creds *IdCClientCredentials, authResp *DeviceAuthorizationResponse) (*IdCTokenResponse, error) {
	return PollForTokenWithClient(ctx, http.DefaultClient, creds, authResp)
}

// PollForTokenWithClient 使用自定義 HTTP 客戶端輪詢 Token
func PollForTokenWithClient(ctx context.Context, client *http.Client, creds *IdCClientCredentials, authResp *DeviceAuthorizationResponse) (*IdCTokenResponse, error) {
	return PollForTokenWithEndpoint(ctx, client, IdCTokenURL, creds, authResp)
}

// PollForTokenWithEndpoint 使用自定義端點輪詢 Token
// 允許注入 HTTP 客戶端和端點 URL 以便測試
func PollForTokenWithEndpoint(ctx context.Context, client *http.Client, endpoint string, creds *IdCClientCredentials, authResp *DeviceAuthorizationResponse) (*IdCTokenResponse, error) {
	// 計算輪詢間隔
	interval := time.Duration(authResp.Interval) * time.Second
	if interval < time.Second {
		interval = time.Second
	}

	// 建構請求體
	reqBody := TokenPollingRequest{
		ClientId:     creds.ClientId,
		ClientSecret: creds.ClientSecret,
		GrantType:    "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode:   authResp.DeviceCode,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeNetworkError,
			Message: fmt.Sprintf("failed to marshal request: %v", err),
		}
	}

	// 輪詢循環
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 嘗試取得 Token
		tokenResp, err := pollTokenOnce(ctx, client, endpoint, jsonBody)
		if err == nil {
			return tokenResp, nil
		}

		// 檢查是否為可重試的錯誤
		oauthErr, ok := err.(*OAuthError)
		if !ok {
			return nil, err
		}

		// 處理不同的錯誤狀態
		switch oauthErr.Code {
		case IdCErrAuthorizationPending:
			// 繼續輪詢
		case IdCErrSlowDown:
			// 增加間隔
			interval += 5 * time.Second
			ticker.Reset(interval)
		default:
			// 其他錯誤直接返回
			return nil, err
		}

		// 等待下一次輪詢或 context 取消
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return nil, &OAuthError{
					Code:    ErrCodeTimeout,
					Message: "polling timeout",
				}
			}
			return nil, &OAuthError{
				Code:    ErrCodeCancelled,
				Message: "polling cancelled",
			}
		case <-ticker.C:
			// 繼續輪詢
		}
	}
}

// pollTokenOnce 執行單次 Token 輪詢
func pollTokenOnce(ctx context.Context, client *http.Client, endpoint string, jsonBody []byte) (*IdCTokenResponse, error) {
	// 檢查 context 是否已取消
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &OAuthError{
				Code:    ErrCodeTimeout,
				Message: "polling timeout",
			}
		}
		return nil, &OAuthError{
			Code:    ErrCodeCancelled,
			Message: "polling cancelled",
		}
	default:
	}

	// 建構 HTTP 請求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeNetworkError,
			Message: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	setIdCHeaders(req)

	// 執行請求
	resp, err := client.Do(req)
	if err != nil {
		// 檢查是否為 context 取消導致的錯誤
		if ctx.Err() != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return nil, &OAuthError{
					Code:    ErrCodeTimeout,
					Message: "polling timeout",
				}
			}
			return nil, &OAuthError{
				Code:    ErrCodeCancelled,
				Message: "polling cancelled",
			}
		}
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
		return nil, mapIdCError(resp.StatusCode, body)
	}

	// 解析回應
	var tokenResp IdCTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, &OAuthError{
			Code:    ErrCodeServerError,
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return &tokenResp, nil
}

// mapIdCError 將 IdC API 錯誤映射到 OAuthError
// 處理 authorization_pending、access_denied、expired_token 等狀態
func mapIdCError(statusCode int, body []byte) *OAuthError {
	// 嘗試解析錯誤回應
	var errResp IdCErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil {
		switch errResp.Error {
		case IdCErrAuthorizationPending:
			return &OAuthError{
				Code:    IdCErrAuthorizationPending,
				Message: "authorization pending, please complete the login in your browser",
			}
		case IdCErrSlowDown:
			return &OAuthError{
				Code:    IdCErrSlowDown,
				Message: "polling too fast, slowing down",
			}
		case IdCErrAccessDenied:
			return &OAuthError{
				Code:    ErrCodeAuthFailed,
				Message: "access denied by user",
			}
		case IdCErrExpiredToken:
			return &OAuthError{
				Code:    ErrCodeTimeout,
				Message: "device code expired",
			}
		}
	}

	// 回退到通用 HTTP 錯誤映射
	return mapHTTPError(statusCode, body)
}
