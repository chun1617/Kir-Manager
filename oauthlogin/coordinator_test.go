// Package oauthlogin 提供 OAuth 登入功能
package oauthlogin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestSocialLogin_Success 測試 Social 登入成功流程
func TestSocialLogin_Success(t *testing.T) {
	// 建立模擬 Token 端點
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// 返回成功的 Token 回應
		resp := SocialTokenResponse{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    3600,
			ProfileArn:   "arn:aws:iam::123456789012:user/test",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	// 建立協調器配置
	config := SocialLoginCoordinatorConfig{
		Provider:     ProviderGithub,
		TokenURL:     tokenServer.URL,
		Timeout:      10 * time.Second,
		OpenBrowser:  false, // 測試時不開啟瀏覽器
		HTTPClient:   tokenServer.Client(),
	}

	// 執行登入（使用模擬回調）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := SocialLoginWithSimulatedCallback(ctx, config, "test-auth-code")
	if err != nil {
		t.Fatalf("SocialLogin failed: %v", err)
	}

	// 驗證結果
	if result.AccessToken != "test-access-token" {
		t.Errorf("expected access token 'test-access-token', got '%s'", result.AccessToken)
	}
	if result.RefreshToken != "test-refresh-token" {
		t.Errorf("expected refresh token 'test-refresh-token', got '%s'", result.RefreshToken)
	}
	if result.Provider != ProviderGithub {
		t.Errorf("expected provider '%s', got '%s'", ProviderGithub, result.Provider)
	}
	if result.AuthMethod != AuthMethodSocial {
		t.Errorf("expected auth method '%s', got '%s'", AuthMethodSocial, result.AuthMethod)
	}
}

// TestSocialLogin_TokenExchangeError 測試 Token 交換失敗
func TestSocialLogin_TokenExchangeError(t *testing.T) {
	// 建立模擬 Token 端點（返回錯誤）
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid_grant"))
	}))
	defer tokenServer.Close()

	config := SocialLoginCoordinatorConfig{
		Provider:     ProviderGithub,
		TokenURL:     tokenServer.URL,
		Timeout:      10 * time.Second,
		OpenBrowser:  false,
		HTTPClient:   tokenServer.Client(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := SocialLoginWithSimulatedCallback(ctx, config, "invalid-code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Fatalf("expected OAuthError, got %T", err)
	}
	if oauthErr.Code != ErrCodeInvalidCode {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeInvalidCode, oauthErr.Code)
	}
}

// TestSocialLogin_StateMismatch 測試 State 不匹配
func TestSocialLogin_StateMismatch(t *testing.T) {
	config := SocialLoginCoordinatorConfig{
		Provider:    ProviderGithub,
		Timeout:     10 * time.Second,
		OpenBrowser: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := SocialLoginWithMismatchedState(ctx, config)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Fatalf("expected OAuthError, got %T", err)
	}
	if oauthErr.Code != ErrCodeStateMismatch {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeStateMismatch, oauthErr.Code)
	}
}

// TestIdCLogin_Success 測試 IdC 登入成功流程
func TestIdCLogin_Success(t *testing.T) {
	// 建立模擬 IdC 端點
	registerCalled := false
	deviceAuthCalled := false
	tokenCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/register":
			registerCalled = true
			resp := IdCClientCredentials{
				ClientId:     "test-client-id",
				ClientSecret: "test-client-secret",
			}
			json.NewEncoder(w).Encode(resp)

		case "/device_authorization":
			deviceAuthCalled = true
			resp := DeviceAuthorizationResponse{
				DeviceCode:              "test-device-code",
				UserCode:                "TEST-CODE",
				VerificationUri:         "https://device.sso.us-east-1.amazonaws.com/",
				VerificationUriComplete: "https://device.sso.us-east-1.amazonaws.com/?user_code=TEST-CODE",
				ExpiresIn:               600,
				Interval:                1,
			}
			json.NewEncoder(w).Encode(resp)

		case "/token":
			tokenCalled = true
			resp := IdCTokenResponse{
				AccessToken:  "test-idc-access-token",
				RefreshToken: "test-idc-refresh-token",
				IdToken:      "test-id-token",
				ExpiresIn:    3600,
			}
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 建立協調器配置
	config := IdCLoginCoordinatorConfig{
		StartURL:          "https://test.awsapps.com/start",
		ClientName:        "Kiro Manager Test",
		RegisterURL:       server.URL + "/register",
		DeviceAuthURL:     server.URL + "/device_authorization",
		TokenURL:          server.URL + "/token",
		Timeout:           10 * time.Second,
		OpenBrowser:       false, // 測試時不開啟瀏覽器
		HTTPClient:        server.Client(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := IdCLogin(ctx, config)
	if err != nil {
		t.Fatalf("IdCLogin failed: %v", err)
	}

	// 驗證所有端點都被呼叫
	if !registerCalled {
		t.Error("register endpoint was not called")
	}
	if !deviceAuthCalled {
		t.Error("device_authorization endpoint was not called")
	}
	if !tokenCalled {
		t.Error("token endpoint was not called")
	}

	// 驗證結果
	if result.AccessToken != "test-idc-access-token" {
		t.Errorf("expected access token 'test-idc-access-token', got '%s'", result.AccessToken)
	}
	if result.RefreshToken != "test-idc-refresh-token" {
		t.Errorf("expected refresh token 'test-idc-refresh-token', got '%s'", result.RefreshToken)
	}
	if result.Provider != ProviderBuilderID {
		t.Errorf("expected provider '%s', got '%s'", ProviderBuilderID, result.Provider)
	}
	if result.AuthMethod != AuthMethodIdC {
		t.Errorf("expected auth method '%s', got '%s'", AuthMethodIdC, result.AuthMethod)
	}
	if result.ClientId != "test-client-id" {
		t.Errorf("expected client id 'test-client-id', got '%s'", result.ClientId)
	}
	if result.ClientSecret != "test-client-secret" {
		t.Errorf("expected client secret 'test-client-secret', got '%s'", result.ClientSecret)
	}

	// 驗證 ClientIdHash
	expectedHash := sha256.Sum256([]byte("test-client-id"))
	expectedHashStr := hex.EncodeToString(expectedHash[:])
	if result.ClientIdHash != expectedHashStr {
		t.Errorf("expected client id hash '%s', got '%s'", expectedHashStr, result.ClientIdHash)
	}
}

// TestIdCLogin_RegisterError 測試設備註冊失敗
func TestIdCLogin_RegisterError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	config := IdCLoginCoordinatorConfig{
		StartURL:      "https://test.awsapps.com/start",
		ClientName:    "Kiro Manager Test",
		RegisterURL:   server.URL + "/register",
		Timeout:       10 * time.Second,
		OpenBrowser:   false,
		HTTPClient:    server.Client(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := IdCLogin(ctx, config)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Fatalf("expected OAuthError, got %T", err)
	}
	if oauthErr.Code != ErrCodeServerError {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeServerError, oauthErr.Code)
	}
}

// TestIdCLogin_Timeout 測試 IdC 登入超時
func TestIdCLogin_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/register":
			resp := IdCClientCredentials{
				ClientId:     "test-client-id",
				ClientSecret: "test-client-secret",
			}
			json.NewEncoder(w).Encode(resp)

		case "/device_authorization":
			resp := DeviceAuthorizationResponse{
				DeviceCode:              "test-device-code",
				UserCode:                "TEST-CODE",
				VerificationUri:         "https://device.sso.us-east-1.amazonaws.com/",
				VerificationUriComplete: "https://device.sso.us-east-1.amazonaws.com/?user_code=TEST-CODE",
				ExpiresIn:               600,
				Interval:                1,
			}
			json.NewEncoder(w).Encode(resp)

		case "/token":
			// 持續返回 authorization_pending
			w.WriteHeader(http.StatusBadRequest)
			resp := IdCErrorResponse{
				Error:            IdCErrAuthorizationPending,
				ErrorDescription: "authorization pending",
			}
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := IdCLoginCoordinatorConfig{
		StartURL:      "https://test.awsapps.com/start",
		ClientName:    "Kiro Manager Test",
		RegisterURL:   server.URL + "/register",
		DeviceAuthURL: server.URL + "/device_authorization",
		TokenURL:      server.URL + "/token",
		Timeout:       2 * time.Second, // 短超時以加速測試
		OpenBrowser:   false,
		HTTPClient:    server.Client(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := IdCLogin(ctx, config)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Fatalf("expected OAuthError, got %T", err)
	}
	if oauthErr.Code != ErrCodeTimeout {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeTimeout, oauthErr.Code)
	}
}
