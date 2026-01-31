// Package oauthlogin 提供 OAuth 登入功能的測試
package oauthlogin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/quick"
	"time"
)

// TestProperty6_IdCDeviceRegistrationRequestFormat 驗證設備註冊請求格式正確
// Property 6: IdC Device Registration Request Format
// 對於任意有效的 clientName 和 issuerUrl，生成的請求必須包含：
// - clientName: 客戶端名稱
// - clientType: "public"
// - grantTypes: ["device_code", "refresh_token"]
// - issuerUrl: 發行者 URL
func TestProperty6_IdCDeviceRegistrationRequestFormat(t *testing.T) {
	f := func(clientName, issuerUrl string) bool {
		// 跳過空字串
		if clientName == "" || issuerUrl == "" {
			return true
		}

		// 建立 mock server 來捕獲請求
		var capturedRequest DeviceRegistrationRequest
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 解析請求體
			if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
				t.Logf("Failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// 返回成功回應
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(IdCClientCredentials{
				ClientId:     "test_client_id",
				ClientSecret: "test_client_secret",
			})
		}))
		defer server.Close()

		// 執行設備註冊
		_, err := RegisterDeviceClientWithEndpoint(http.DefaultClient, server.URL, clientName, issuerUrl)
		if err != nil {
			t.Logf("RegisterDeviceClient failed: %v", err)
			return false
		}

		// 驗證請求格式
		if capturedRequest.ClientName != clientName {
			t.Logf("ClientName mismatch: expected %s, got %s", clientName, capturedRequest.ClientName)
			return false
		}

		if capturedRequest.ClientType != "public" {
			t.Logf("ClientType mismatch: expected public, got %s", capturedRequest.ClientType)
			return false
		}

		if len(capturedRequest.GrantTypes) != 2 ||
			capturedRequest.GrantTypes[0] != "device_code" ||
			capturedRequest.GrantTypes[1] != "refresh_token" {
			t.Logf("GrantTypes mismatch: expected [device_code, refresh_token], got %v", capturedRequest.GrantTypes)
			return false
		}

		if capturedRequest.IssuerUrl != issuerUrl {
			t.Logf("IssuerUrl mismatch: expected %s, got %s", issuerUrl, capturedRequest.IssuerUrl)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Errorf("Property 6 failed: %v", err)
	}
}

// TestProperty7_IdCTokenPollingRequestFormat 驗證 Token 輪詢請求格式正確
// Property 7: IdC Token Polling Request Format
// 對於任意有效的 clientId、clientSecret 和 deviceCode，生成的請求必須包含：
// - clientId: 客戶端 ID
// - clientSecret: 客戶端密鑰
// - grantType: "urn:ietf:params:oauth:grant-type:device_code"
// - deviceCode: 設備碼
func TestProperty7_IdCTokenPollingRequestFormat(t *testing.T) {
	f := func(clientId, clientSecret, deviceCode string) bool {
		// 跳過空字串
		if clientId == "" || clientSecret == "" || deviceCode == "" {
			return true
		}

		// 建立 mock server 來捕獲請求
		var capturedRequest TokenPollingRequest
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 解析請求體
			if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
				t.Logf("Failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// 返回成功回應
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(IdCTokenResponse{
				AccessToken:  "test_access_token",
				RefreshToken: "test_refresh_token",
				ExpiresIn:    3600,
			})
		}))
		defer server.Close()

		// 建立憑證
		creds := &IdCClientCredentials{
			ClientId:     clientId,
			ClientSecret: clientSecret,
		}

		// 建立授權回應
		authResp := &DeviceAuthorizationResponse{
			DeviceCode: deviceCode,
			Interval:   1,
			ExpiresIn:  300,
		}

		// 執行 Token 輪詢（使用短超時）
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := PollForTokenWithEndpoint(ctx, http.DefaultClient, server.URL, creds, authResp)
		if err != nil {
			t.Logf("PollForToken failed: %v", err)
			return false
		}

		// 驗證請求格式
		if capturedRequest.ClientId != clientId {
			t.Logf("ClientId mismatch: expected %s, got %s", clientId, capturedRequest.ClientId)
			return false
		}

		if capturedRequest.ClientSecret != clientSecret {
			t.Logf("ClientSecret mismatch: expected %s, got %s", clientSecret, capturedRequest.ClientSecret)
			return false
		}

		expectedGrantType := "urn:ietf:params:oauth:grant-type:device_code"
		if capturedRequest.GrantType != expectedGrantType {
			t.Logf("GrantType mismatch: expected %s, got %s", expectedGrantType, capturedRequest.GrantType)
			return false
		}

		if capturedRequest.DeviceCode != deviceCode {
			t.Logf("DeviceCode mismatch: expected %s, got %s", deviceCode, capturedRequest.DeviceCode)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Errorf("Property 7 failed: %v", err)
	}
}

// TestProperty8_IdCTokenResponseParsing 驗證 IdC Token 回應解析正確性
// Property 8: IdC Token Response Parsing
// 對於任意有效的 Token 回應 JSON，解析後的結構應該保留所有欄位值
func TestProperty8_IdCTokenResponseParsing(t *testing.T) {
	f := func(accessToken, refreshToken, idToken string, expiresIn uint16) bool {
		// 確保 expiresIn 在合理範圍內
		if expiresIn == 0 {
			expiresIn = 3600
		}

		// 建立原始回應 JSON
		originalResponse := map[string]interface{}{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
			"idToken":      idToken,
			"expiresIn":    int(expiresIn),
		}

		jsonData, err := json.Marshal(originalResponse)
		if err != nil {
			t.Logf("Failed to marshal JSON: %v", err)
			return false
		}

		// 解析 JSON 到 IdCTokenResponse
		var tokenResponse IdCTokenResponse
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

		if tokenResponse.IdToken != idToken {
			t.Logf("IdToken mismatch: expected %s, got %s", idToken, tokenResponse.IdToken)
			return false
		}

		if tokenResponse.ExpiresIn != int(expiresIn) {
			t.Logf("ExpiresIn mismatch: expected %d, got %d", expiresIn, tokenResponse.ExpiresIn)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property 8 failed: %v", err)
	}
}

// TestRegisterDeviceClient_Success 驗證成功的設備註冊
func TestRegisterDeviceClient_Success(t *testing.T) {
	expectedResponse := IdCClientCredentials{
		ClientId:     "test_client_id_12345",
		ClientSecret: "test_client_secret_67890",
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

	result, err := RegisterDeviceClientWithEndpoint(http.DefaultClient, server.URL, "Kiro Manager", "https://view.awsapps.com/start")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ClientId != expectedResponse.ClientId {
		t.Errorf("ClientId mismatch: expected %s, got %s", expectedResponse.ClientId, result.ClientId)
	}

	if result.ClientSecret != expectedResponse.ClientSecret {
		t.Errorf("ClientSecret mismatch: expected %s, got %s", expectedResponse.ClientSecret, result.ClientSecret)
	}
}

// TestStartDeviceAuthorization_Success 驗證成功的設備授權
func TestStartDeviceAuthorization_Success(t *testing.T) {
	expectedResponse := DeviceAuthorizationResponse{
		DeviceCode:              "device_code_12345",
		UserCode:                "ABCD-EFGH",
		VerificationUri:         "https://device.sso.us-east-1.amazonaws.com/",
		VerificationUriComplete: "https://device.sso.us-east-1.amazonaws.com/?user_code=ABCD-EFGH",
		ExpiresIn:               600,
		Interval:                5,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 驗證請求方法
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// 返回成功回應
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	creds := &IdCClientCredentials{
		ClientId:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	result, err := StartDeviceAuthorizationWithEndpoint(http.DefaultClient, server.URL, creds, "https://view.awsapps.com/start")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.DeviceCode != expectedResponse.DeviceCode {
		t.Errorf("DeviceCode mismatch: expected %s, got %s", expectedResponse.DeviceCode, result.DeviceCode)
	}

	if result.UserCode != expectedResponse.UserCode {
		t.Errorf("UserCode mismatch: expected %s, got %s", expectedResponse.UserCode, result.UserCode)
	}

	if result.VerificationUri != expectedResponse.VerificationUri {
		t.Errorf("VerificationUri mismatch: expected %s, got %s", expectedResponse.VerificationUri, result.VerificationUri)
	}

	if result.VerificationUriComplete != expectedResponse.VerificationUriComplete {
		t.Errorf("VerificationUriComplete mismatch: expected %s, got %s", expectedResponse.VerificationUriComplete, result.VerificationUriComplete)
	}

	if result.ExpiresIn != expectedResponse.ExpiresIn {
		t.Errorf("ExpiresIn mismatch: expected %d, got %d", expectedResponse.ExpiresIn, result.ExpiresIn)
	}

	if result.Interval != expectedResponse.Interval {
		t.Errorf("Interval mismatch: expected %d, got %d", expectedResponse.Interval, result.Interval)
	}
}

// TestPollForToken_Success 驗證成功的 Token 輪詢
func TestPollForToken_Success(t *testing.T) {
	expectedResponse := IdCTokenResponse{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		IdToken:      "test_id_token",
		ExpiresIn:    3600,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	creds := &IdCClientCredentials{
		ClientId:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	authResp := &DeviceAuthorizationResponse{
		DeviceCode: "test_device_code",
		Interval:   1,
		ExpiresIn:  300,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := PollForTokenWithEndpoint(ctx, http.DefaultClient, server.URL, creds, authResp)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.AccessToken != expectedResponse.AccessToken {
		t.Errorf("AccessToken mismatch: expected %s, got %s", expectedResponse.AccessToken, result.AccessToken)
	}

	if result.RefreshToken != expectedResponse.RefreshToken {
		t.Errorf("RefreshToken mismatch: expected %s, got %s", expectedResponse.RefreshToken, result.RefreshToken)
	}

	if result.IdToken != expectedResponse.IdToken {
		t.Errorf("IdToken mismatch: expected %s, got %s", expectedResponse.IdToken, result.IdToken)
	}

	if result.ExpiresIn != expectedResponse.ExpiresIn {
		t.Errorf("ExpiresIn mismatch: expected %d, got %d", expectedResponse.ExpiresIn, result.ExpiresIn)
	}
}

// TestPollForToken_AuthorizationPending 驗證 authorization_pending 狀態處理
func TestPollForToken_AuthorizationPending(t *testing.T) {
	callCount := 0
	expectedResponse := IdCTokenResponse{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresIn:    3600,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			// 前兩次返回 authorization_pending
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
			return
		}
		// 第三次返回成功
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	creds := &IdCClientCredentials{
		ClientId:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	authResp := &DeviceAuthorizationResponse{
		DeviceCode: "test_device_code",
		Interval:   1, // 1 秒間隔
		ExpiresIn:  300,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := PollForTokenWithEndpoint(ctx, http.DefaultClient, server.URL, creds, authResp)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if callCount < 3 {
		t.Errorf("Expected at least 3 calls, got %d", callCount)
	}

	if result.AccessToken != expectedResponse.AccessToken {
		t.Errorf("AccessToken mismatch: expected %s, got %s", expectedResponse.AccessToken, result.AccessToken)
	}
}

// TestPollForToken_AccessDenied 驗證 access_denied 狀態處理
func TestPollForToken_AccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "access_denied"})
	}))
	defer server.Close()

	creds := &IdCClientCredentials{
		ClientId:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	authResp := &DeviceAuthorizationResponse{
		DeviceCode: "test_device_code",
		Interval:   1,
		ExpiresIn:  300,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := PollForTokenWithEndpoint(ctx, http.DefaultClient, server.URL, creds, authResp)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Fatalf("Expected *OAuthError, got %T", err)
	}

	if oauthErr.Code != ErrCodeAuthFailed {
		t.Errorf("Expected error code %s, got %s", ErrCodeAuthFailed, oauthErr.Code)
	}
}

// TestPollForToken_ExpiredToken 驗證 expired_token 狀態處理
func TestPollForToken_ExpiredToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "expired_token"})
	}))
	defer server.Close()

	creds := &IdCClientCredentials{
		ClientId:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	authResp := &DeviceAuthorizationResponse{
		DeviceCode: "test_device_code",
		Interval:   1,
		ExpiresIn:  300,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := PollForTokenWithEndpoint(ctx, http.DefaultClient, server.URL, creds, authResp)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Fatalf("Expected *OAuthError, got %T", err)
	}

	if oauthErr.Code != ErrCodeTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeTimeout, oauthErr.Code)
	}
}

// TestPollForToken_ContextCancellation 驗證 context 取消處理
func TestPollForToken_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 總是返回 authorization_pending
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
	}))
	defer server.Close()

	creds := &IdCClientCredentials{
		ClientId:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	authResp := &DeviceAuthorizationResponse{
		DeviceCode: "test_device_code",
		Interval:   1,
		ExpiresIn:  300,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := PollForTokenWithEndpoint(ctx, http.DefaultClient, server.URL, creds, authResp)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Fatalf("Expected *OAuthError, got %T", err)
	}

	if oauthErr.Code != ErrCodeCancelled && oauthErr.Code != ErrCodeTimeout {
		t.Errorf("Expected error code %s or %s, got %s", ErrCodeCancelled, ErrCodeTimeout, oauthErr.Code)
	}
}

// TestRegisterDeviceClient_HTTPError 驗證設備註冊的 HTTP 錯誤處理
func TestRegisterDeviceClient_HTTPError(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		expectedCode string
	}{
		{"400 Bad Request", 400, ErrCodeInvalidCode},
		{"401 Unauthorized", 401, ErrCodeAuthFailed},
		{"500 Internal Server Error", 500, ErrCodeServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte("error response"))
			}))
			defer server.Close()

			_, err := RegisterDeviceClientWithEndpoint(http.DefaultClient, server.URL, "Kiro Manager", "https://view.awsapps.com/start")
			if err == nil {
				t.Fatal("Expected error, got nil")
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

// TestIdCHeaders 驗證 IdC API 請求包含正確的 Headers
func TestIdCHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 驗證必要的 Headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		if r.Header.Get("x-amz-user-agent") == "" {
			t.Error("Expected x-amz-user-agent header to be set")
		}

		if r.Header.Get("User-Agent") != "node" {
			t.Errorf("Expected User-Agent node, got %s", r.Header.Get("User-Agent"))
		}

		// 返回成功回應
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IdCClientCredentials{
			ClientId:     "test_client_id",
			ClientSecret: "test_client_secret",
		})
	}))
	defer server.Close()

	_, err := RegisterDeviceClientWithEndpoint(http.DefaultClient, server.URL, "Kiro Manager", "https://view.awsapps.com/start")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
