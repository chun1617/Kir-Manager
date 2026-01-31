package oauthlogin

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestProperty_CallbackServerStateValidation 測試 Callback Server State 驗證
// Property 3: Callback Server 應正確驗證 state 參數
// - 正確的 state 應該成功接收 authorization_code
// - 錯誤的 state 應該被拒絕
func TestProperty_CallbackServerStateValidation(t *testing.T) {
	const iterations = 10

	for i := 0; i < iterations; i++ {
		// 生成 PKCE 參數
		pkce, err := GeneratePKCE()
		if err != nil {
			t.Fatalf("GeneratePKCE() failed: %v", err)
		}

		// 建立 Callback Server
		server := NewCallbackServer(pkce.State)

		// 啟動 Server
		port, err := server.Start()
		if err != nil {
			t.Fatalf("server.Start() failed: %v", err)
		}
		defer server.Stop()

		// 測試正確的 state
		testCode := fmt.Sprintf("test_code_%d", i)
		callbackURL := fmt.Sprintf("http://localhost:%d/callback?code=%s&state=%s",
			port, testCode, pkce.State)

		// 發送回調請求
		go func() {
			resp, err := http.Get(callbackURL)
			if err != nil {
				t.Errorf("HTTP GET failed: %v", err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}()

		// 等待回調結果
		result, err := server.WaitForCallback(5 * time.Second)
		if err != nil {
			t.Errorf("WaitForCallback() failed: %v", err)
			continue
		}

		if result.Code != testCode {
			t.Errorf("Code mismatch: got %s, want %s", result.Code, testCode)
		}
	}
}

// TestCallbackServer_InvalidState 測試無效 state 被拒絕
func TestCallbackServer_InvalidState(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() failed: %v", err)
	}

	server := NewCallbackServer(pkce.State)
	port, err := server.Start()
	if err != nil {
		t.Fatalf("server.Start() failed: %v", err)
	}
	defer server.Stop()

	// 使用錯誤的 state
	wrongState := "wrong_state_value"
	callbackURL := fmt.Sprintf("http://localhost:%d/callback?code=test_code&state=%s",
		port, wrongState)

	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	defer resp.Body.Close()

	// 應該返回 400 Bad Request
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid state, got %d", resp.StatusCode)
	}
}

// TestCallbackServer_Timeout 測試超時處理
func TestCallbackServer_Timeout(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() failed: %v", err)
	}

	server := NewCallbackServer(pkce.State)
	_, err = server.Start()
	if err != nil {
		t.Fatalf("server.Start() failed: %v", err)
	}
	defer server.Stop()

	// 使用很短的超時時間
	_, err = server.WaitForCallback(100 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// 驗證錯誤類型
	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Errorf("Expected *OAuthError, got %T", err)
	} else if oauthErr.Code != ErrCodeTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeTimeout, oauthErr.Code)
	}
}

// TestCallbackServer_MissingCode 測試缺少 code 參數
func TestCallbackServer_MissingCode(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() failed: %v", err)
	}

	server := NewCallbackServer(pkce.State)
	port, err := server.Start()
	if err != nil {
		t.Fatalf("server.Start() failed: %v", err)
	}
	defer server.Stop()

	// 缺少 code 參數
	callbackURL := fmt.Sprintf("http://localhost:%d/callback?state=%s", port, pkce.State)

	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	defer resp.Body.Close()

	// 應該返回 400 Bad Request
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing code, got %d", resp.StatusCode)
	}
}

// TestCallbackServer_ErrorCallback 測試錯誤回調處理
func TestCallbackServer_ErrorCallback(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() failed: %v", err)
	}

	server := NewCallbackServer(pkce.State)
	port, err := server.Start()
	if err != nil {
		t.Fatalf("server.Start() failed: %v", err)
	}
	defer server.Stop()

	// 模擬用戶取消授權的回調
	callbackURL := fmt.Sprintf("http://localhost:%d/callback?error=access_denied&state=%s",
		port, pkce.State)

	go func() {
		resp, err := http.Get(callbackURL)
		if err != nil {
			return
		}
		resp.Body.Close()
	}()

	// 等待回調結果
	_, err = server.WaitForCallback(5 * time.Second)
	if err == nil {
		t.Error("Expected error for access_denied, got nil")
	}

	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Errorf("Expected *OAuthError, got %T", err)
	} else if oauthErr.Code != ErrCodeCancelled {
		t.Errorf("Expected error code %s, got %s", ErrCodeCancelled, oauthErr.Code)
	}
}

// TestCallbackServer_GetCallbackURL 測試取得回調 URL
func TestCallbackServer_GetCallbackURL(t *testing.T) {
	server := NewCallbackServer("test_state")
	port, err := server.Start()
	if err != nil {
		t.Fatalf("server.Start() failed: %v", err)
	}
	defer server.Stop()

	expectedURL := fmt.Sprintf("http://localhost:%d/callback", port)
	actualURL := server.GetCallbackURL()

	if actualURL != expectedURL {
		t.Errorf("GetCallbackURL() = %s, want %s", actualURL, expectedURL)
	}
}
