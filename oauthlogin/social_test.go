// Package oauthlogin 提供 OAuth 登入功能的測試
package oauthlogin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"testing/quick"
)

// TestProperty4_AuthorizationURLParameterCompleteness 驗證 BuildAuthorizationURL 生成的 URL 包含所有必要參數
// Property 4: Authorization URL Parameter Completeness
// 對於任意有效的 SocialLoginConfig 和 PKCEParams，生成的授權 URL 必須包含：
// - idp: Provider 名稱
// - redirect_uri: 回調 URL
// - code_challenge: PKCE challenge
// - code_challenge_method: S256
// - state: 隨機 state
func TestProperty4_AuthorizationURLParameterCompleteness(t *testing.T) {
	f := func(port uint16, providerIdx uint8) bool {
		// 確保 port 在有效範圍內
		if port == 0 {
			port = 8080
		}

		// 選擇 provider
		providers := []string{ProviderGithub, ProviderGoogle}
		provider := providers[int(providerIdx)%len(providers)]

		// 建立配置
		config := SocialLoginConfig{
			Provider:    provider,
			Port:        int(port),
			RedirectURI: "",
		}

		// 生成 PKCE 參數
		pkce, err := GeneratePKCE()
		if err != nil {
			t.Logf("GeneratePKCE failed: %v", err)
			return false
		}

		// 建構授權 URL
		authURL := BuildAuthorizationURL(config, *pkce)

		// 解析 URL
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			t.Logf("Failed to parse URL: %v", err)
			return false
		}

		// 驗證所有必要參數存在
		query := parsedURL.Query()

		// 檢查 idp 參數
		idp := query.Get("idp")
		if idp != provider {
			t.Logf("idp mismatch: expected %s, got %s", provider, idp)
			return false
		}

		// 檢查 redirect_uri 參數
		redirectURI := query.Get("redirect_uri")
		if redirectURI == "" {
			t.Logf("redirect_uri is empty")
			return false
		}

		// 檢查 code_challenge 參數
		codeChallenge := query.Get("code_challenge")
		if codeChallenge != pkce.CodeChallenge {
			t.Logf("code_challenge mismatch: expected %s, got %s", pkce.CodeChallenge, codeChallenge)
			return false
		}

		// 檢查 code_challenge_method 參數
		codeChallengeMethod := query.Get("code_challenge_method")
		if codeChallengeMethod != "S256" {
			t.Logf("code_challenge_method mismatch: expected S256, got %s", codeChallengeMethod)
			return false
		}

		// 檢查 state 參數
		state := query.Get("state")
		if state != pkce.State {
			t.Logf("state mismatch: expected %s, got %s", pkce.State, state)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property 4 failed: %v", err)
	}
}

// TestProperty5_TokenResponseParsingRoundTrip 驗證 Token 回應解析正確性
// Property 5: Token Response Parsing Round-Trip
// 對於任意有效的 Token 回應 JSON，解析後的結構應該保留所有欄位值
func TestProperty5_TokenResponseParsingRoundTrip(t *testing.T) {
	f := func(accessToken, refreshToken, profileArn string, expiresIn uint16) bool {
		// 確保 expiresIn 在合理範圍內
		if expiresIn == 0 {
			expiresIn = 3600
		}

		// 建立原始回應 JSON
		originalResponse := map[string]interface{}{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
			"expiresIn":    int(expiresIn),
			"profileArn":   profileArn,
		}

		jsonData, err := json.Marshal(originalResponse)
		if err != nil {
			t.Logf("Failed to marshal JSON: %v", err)
			return false
		}

		// 解析 JSON 到 SocialTokenResponse
		var tokenResponse SocialTokenResponse
		if err := json.Unmarshal(jsonData, &tokenResponse); err != nil {
			t.Logf("Failed to unmarshal JSON: %v", err)
			return false
		}

		// 驗證所有欄位正確解析
		if tokenResponse.AccessToken != accessToken {
			t.Logf("AccessToken mismatch: expected %s, got %s", accessToken, tokenResponse.AccessToken)
			return false
		}

		if tokenResponse.RefreshToken != refreshToken {
			t.Logf("RefreshToken mismatch: expected %s, got %s", refreshToken, tokenResponse.RefreshToken)
			return false
		}

		if tokenResponse.ExpiresIn != int(expiresIn) {
			t.Logf("ExpiresIn mismatch: expected %d, got %d", expiresIn, tokenResponse.ExpiresIn)
			return false
		}

		if tokenResponse.ProfileArn != profileArn {
			t.Logf("ProfileArn mismatch: expected %s, got %s", profileArn, tokenResponse.ProfileArn)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property 5 failed: %v", err)
	}
}

// TestBuildAuthorizationURL_BaseURL 驗證授權 URL 的基礎 URL 正確
func TestBuildAuthorizationURL_BaseURL(t *testing.T) {
	config := SocialLoginConfig{
		Provider: ProviderGithub,
		Port:     8080,
	}

	pkce, _ := GeneratePKCE()
	authURL := BuildAuthorizationURL(config, *pkce)

	parsedURL, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	expectedHost := "prod.us-east-1.auth.desktop.kiro.dev"
	if parsedURL.Host != expectedHost {
		t.Errorf("Host mismatch: expected %s, got %s", expectedHost, parsedURL.Host)
	}

	if parsedURL.Scheme != "https" {
		t.Errorf("Scheme mismatch: expected https, got %s", parsedURL.Scheme)
	}
}

// TestBuildAuthorizationURL_RedirectURI 驗證 redirect_uri 格式正確
func TestBuildAuthorizationURL_RedirectURI(t *testing.T) {
	testCases := []struct {
		name        string
		port        int
		expectedURI string
	}{
		{"port 8080", 8080, "http://localhost:8080/callback"},
		{"port 3000", 3000, "http://localhost:3000/callback"},
		{"port 9999", 9999, "http://localhost:9999/callback"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := SocialLoginConfig{
				Provider: ProviderGithub,
				Port:     tc.port,
			}

			pkce, _ := GeneratePKCE()
			authURL := BuildAuthorizationURL(config, *pkce)

			parsedURL, _ := url.Parse(authURL)
			redirectURI := parsedURL.Query().Get("redirect_uri")

			if redirectURI != tc.expectedURI {
				t.Errorf("redirect_uri mismatch: expected %s, got %s", tc.expectedURI, redirectURI)
			}
		})
	}
}

// TestMapHTTPError_400 驗證 400 錯誤映射到 ErrCodeInvalidCode
func TestMapHTTPError_400(t *testing.T) {
	oauthErr := mapHTTPError(400, []byte("invalid code"))

	if oauthErr.Code != ErrCodeInvalidCode {
		t.Errorf("Expected error code %s, got %s", ErrCodeInvalidCode, oauthErr.Code)
	}
}

// TestMapHTTPError_401 驗證 401 錯誤映射到 ErrCodeAuthFailed
func TestMapHTTPError_401(t *testing.T) {
	oauthErr := mapHTTPError(401, []byte("unauthorized"))

	if oauthErr.Code != ErrCodeAuthFailed {
		t.Errorf("Expected error code %s, got %s", ErrCodeAuthFailed, oauthErr.Code)
	}
}

// TestMapHTTPError_5xx 驗證 5xx 錯誤映射到 ErrCodeServerError
func TestMapHTTPError_5xx(t *testing.T) {
	testCases := []int{500, 501, 502, 503, 504}

	for _, statusCode := range testCases {
		t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
			oauthErr := mapHTTPError(statusCode, []byte("server error"))

			if oauthErr.Code != ErrCodeServerError {
				t.Errorf("Expected error code %s, got %s", ErrCodeServerError, oauthErr.Code)
			}
		})
	}
}

// TestExchangeToken_HTTPErrorMapping 驗證 ExchangeToken 的 HTTP 錯誤映射
func TestExchangeToken_HTTPErrorMapping(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		expectedCode string
	}{
		{"400 Bad Request", 400, ErrCodeInvalidCode},
		{"401 Unauthorized", 401, ErrCodeAuthFailed},
		{"500 Internal Server Error", 500, ErrCodeServerError},
		{"502 Bad Gateway", 502, ErrCodeServerError},
		{"503 Service Unavailable", 503, ErrCodeServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 建立 mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte("error response"))
			}))
			defer server.Close()

			// 建立使用 mock server 的配置
			config := SocialLoginConfig{
				Provider:    ProviderGithub,
				Port:        8080,
				RedirectURI: "http://localhost:8080/callback",
			}

			pkce, _ := GeneratePKCE()

			// 使用 ExchangeTokenWithEndpoint 測試
			_, err := ExchangeTokenWithEndpoint(http.DefaultClient, server.URL, config, "test_code", *pkce)
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}

			oauthErr, ok := err.(*OAuthError)
			if !ok {
				t.Fatalf("Expected *OAuthError, got %T", err)
			}

			if oauthErr.Code != tc.expectedCode {
				t.Errorf("Expected error code %s, got %s", tc.expectedCode, oauthErr.Code)
			}
		})
	}
}

// TestExchangeToken_Success 驗證成功的 Token 交換
func TestExchangeToken_Success(t *testing.T) {
	expectedResponse := SocialTokenResponse{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresIn:    3600,
		ProfileArn:   "arn:aws:iam::123456789012:user/test",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 驗證請求方法
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// 驗證 Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// 返回成功回應
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	config := SocialLoginConfig{
		Provider:    ProviderGithub,
		Port:        8080,
		RedirectURI: "http://localhost:8080/callback",
	}

	pkce, _ := GeneratePKCE()

	result, err := ExchangeTokenWithEndpoint(http.DefaultClient, server.URL, config, "test_code", *pkce)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.AccessToken != expectedResponse.AccessToken {
		t.Errorf("AccessToken mismatch: expected %s, got %s", expectedResponse.AccessToken, result.AccessToken)
	}

	if result.RefreshToken != expectedResponse.RefreshToken {
		t.Errorf("RefreshToken mismatch: expected %s, got %s", expectedResponse.RefreshToken, result.RefreshToken)
	}

	if result.ExpiresIn != expectedResponse.ExpiresIn {
		t.Errorf("ExpiresIn mismatch: expected %d, got %d", expectedResponse.ExpiresIn, result.ExpiresIn)
	}

	if result.ProfileArn != expectedResponse.ProfileArn {
		t.Errorf("ProfileArn mismatch: expected %s, got %s", expectedResponse.ProfileArn, result.ProfileArn)
	}
}
