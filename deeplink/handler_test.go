package deeplink

import (
	"testing"
	"time"
)

// TestParseDeepLinkURL_Success 測試正確解析 code 和 state
func TestParseDeepLinkURL_Success(t *testing.T) {
	rawURL := "kiro://kiro.kiroAgent/authenticate-success?code=test_code_123&state=test_state_456"

	result, err := ParseDeepLinkURL(rawURL)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Code != "test_code_123" {
		t.Errorf("expected code 'test_code_123', got '%s'", result.Code)
	}
	if result.State != "test_state_456" {
		t.Errorf("expected state 'test_state_456', got '%s'", result.State)
	}
}

// TestParseDeepLinkURL_InvalidScheme 測試非 kiro:// scheme 返回 ErrInvalidScheme
func TestParseDeepLinkURL_InvalidScheme(t *testing.T) {
	testCases := []string{
		"http://kiro.kiroAgent/authenticate-success?code=abc&state=xyz",
		"https://kiro.kiroAgent/authenticate-success?code=abc&state=xyz",
		"myapp://kiro.kiroAgent/authenticate-success?code=abc&state=xyz",
	}

	for _, rawURL := range testCases {
		_, err := ParseDeepLinkURL(rawURL)
		if err != ErrInvalidScheme {
			t.Errorf("URL '%s': expected ErrInvalidScheme, got %v", rawURL, err)
		}
	}
}

// TestParseDeepLinkURL_MissingCode 測試缺少 code 返回 ErrMissingCode
func TestParseDeepLinkURL_MissingCode(t *testing.T) {
	rawURL := "kiro://kiro.kiroAgent/authenticate-success?state=test_state"

	_, err := ParseDeepLinkURL(rawURL)

	if err != ErrMissingCode {
		t.Errorf("expected ErrMissingCode, got %v", err)
	}
}

// TestParseDeepLinkURL_MissingState 測試缺少 state 返回 ErrStateMismatch
func TestParseDeepLinkURL_MissingState(t *testing.T) {
	rawURL := "kiro://kiro.kiroAgent/authenticate-success?code=test_code"

	_, err := ParseDeepLinkURL(rawURL)

	if err != ErrStateMismatch {
		t.Errorf("expected ErrStateMismatch, got %v", err)
	}
}

// TestValidateDeepLinkURL_Valid 測試有效 URL 返回 true
func TestValidateDeepLinkURL_Valid(t *testing.T) {
	validURLs := []string{
		"kiro://kiro.kiroAgent/authenticate-success?code=abc&state=xyz",
		"kiro://kiro.kiroAgent/authenticate-success?code=123&state=456&extra=param",
	}

	for _, rawURL := range validURLs {
		if !ValidateDeepLinkURL(rawURL) {
			t.Errorf("expected URL '%s' to be valid", rawURL)
		}
	}
}

// TestValidateDeepLinkURL_Invalid 測試無效 URL 返回 false
func TestValidateDeepLinkURL_Invalid(t *testing.T) {
	invalidURLs := []string{
		"http://example.com",
		"kiro://kiro.kiroAgent/authenticate-success",
		"kiro://kiro.kiroAgent/authenticate-success?code=abc",
		"kiro://kiro.kiroAgent/authenticate-success?state=xyz",
		"not-a-url",
		"",
	}

	for _, rawURL := range invalidURLs {
		if ValidateDeepLinkURL(rawURL) {
			t.Errorf("expected URL '%s' to be invalid", rawURL)
		}
	}
}

// TestParseDeepLinkError_AccessDenied 測試解析 error=access_denied
func TestParseDeepLinkError_AccessDenied(t *testing.T) {
	rawURL := "kiro://kiro.kiroAgent/authenticate-success?error=access_denied&error_description=User%20denied%20access"

	dlError, hasError := ParseDeepLinkError(rawURL)

	if !hasError {
		t.Fatal("expected hasError to be true")
	}
	if dlError == nil {
		t.Fatal("expected dlError, got nil")
	}
	if dlError.Error != "access_denied" {
		t.Errorf("expected error 'access_denied', got '%s'", dlError.Error)
	}
	if dlError.Description != "User denied access" {
		t.Errorf("expected description 'User denied access', got '%s'", dlError.Description)
	}
}

// TestParseDeepLinkError_NoError 測試無錯誤參數返回 false
func TestParseDeepLinkError_NoError(t *testing.T) {
	rawURL := "kiro://kiro.kiroAgent/authenticate-success?code=abc&state=xyz"

	dlError, hasError := ParseDeepLinkError(rawURL)

	if hasError {
		t.Error("expected hasError to be false")
	}
	if dlError != nil {
		t.Error("expected dlError to be nil")
	}
}

// TestHandleDeepLinkCallback_Success 測試完整回調處理流程
func TestHandleDeepLinkCallback_Success(t *testing.T) {
	// 準備：儲存一個有效的 State
	testState := &OAuthState{
		State:         "valid_state_123",
		Provider:      "test_provider",
		CodeVerifier:  "test_verifier",
		CodeChallenge: "test_challenge",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(5 * time.Minute),
	}
	if err := SaveState(testState); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}
	defer ClearState()

	rawURL := "kiro://kiro.kiroAgent/authenticate-success?code=auth_code_abc&state=valid_state_123"

	result, err := HandleDeepLinkCallback(rawURL)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Code != "auth_code_abc" {
		t.Errorf("expected code 'auth_code_abc', got '%s'", result.Code)
	}
	if result.State != "valid_state_123" {
		t.Errorf("expected state 'valid_state_123', got '%s'", result.State)
	}
}

// TestHandleDeepLinkCallback_StateMismatch 測試 State 不匹配
func TestHandleDeepLinkCallback_StateMismatch(t *testing.T) {
	// 準備：儲存一個 State
	testState := &OAuthState{
		State:         "stored_state_abc",
		Provider:      "test_provider",
		CodeVerifier:  "test_verifier",
		CodeChallenge: "test_challenge",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(5 * time.Minute),
	}
	if err := SaveState(testState); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}
	defer ClearState()

	// URL 中的 state 與儲存的不同
	rawURL := "kiro://kiro.kiroAgent/authenticate-success?code=auth_code&state=different_state_xyz"

	_, err := HandleDeepLinkCallback(rawURL)

	if err != ErrStateMismatch {
		t.Errorf("expected ErrStateMismatch, got %v", err)
	}
}

// TestHandleDeepLinkCallback_StateExpired 測試 State 已過期
func TestHandleDeepLinkCallback_StateExpired(t *testing.T) {
	// 準備：儲存一個已過期的 State
	testState := &OAuthState{
		State:         "expired_state_123",
		Provider:      "test_provider",
		CodeVerifier:  "test_verifier",
		CodeChallenge: "test_challenge",
		CreatedAt:     time.Now().Add(-10 * time.Minute),
		ExpiresAt:     time.Now().Add(-5 * time.Minute), // 已過期
	}
	if err := SaveState(testState); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}
	defer ClearState()

	rawURL := "kiro://kiro.kiroAgent/authenticate-success?code=auth_code&state=expired_state_123"

	_, err := HandleDeepLinkCallback(rawURL)

	if err != ErrStateExpired {
		t.Errorf("expected ErrStateExpired, got %v", err)
	}
}

// TestHandleDeepLinkCallback_StateNotFound 測試找不到 State
func TestHandleDeepLinkCallback_StateNotFound(t *testing.T) {
	// 確保沒有 State 檔案
	ClearState()

	rawURL := "kiro://kiro.kiroAgent/authenticate-success?code=auth_code&state=some_state"

	_, err := HandleDeepLinkCallback(rawURL)

	if err != ErrStateNotFound {
		t.Errorf("expected ErrStateNotFound, got %v", err)
	}
}


// ============================================
// Callback Channel Tests
// ============================================

// TestInitCallbackChannel 驗證 channel 初始化
func TestInitCallbackChannel(t *testing.T) {
	// 重置以確保乾淨狀態
	ResetCallbackChannel()

	// 初始化 channel
	InitCallbackChannel()

	// 驗證可以發送和接收
	testResult := &DeepLinkResult{Code: "test_code", State: "test_state"}

	// 發送應該成功（不阻塞）
	SendCallback(testResult)

	// 接收應該成功
	result, err := WaitForCallback(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Code != "test_code" {
		t.Errorf("expected code 'test_code', got '%s'", result.Code)
	}

	// 清理
	ResetCallbackChannel()
}

// TestSendCallback_Success 驗證發送成功
func TestSendCallback_Success(t *testing.T) {
	ResetCallbackChannel()
	InitCallbackChannel()
	defer ResetCallbackChannel()

	testResult := &DeepLinkResult{Code: "send_test_code", State: "send_test_state"}

	// 發送不應該 panic 或阻塞
	SendCallback(testResult)

	// 驗證可以接收到
	result, err := WaitForCallback(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Code != "send_test_code" {
		t.Errorf("expected code 'send_test_code', got '%s'", result.Code)
	}
	if result.State != "send_test_state" {
		t.Errorf("expected state 'send_test_state', got '%s'", result.State)
	}
}

// TestWaitForCallback_Success 驗證接收成功
func TestWaitForCallback_Success(t *testing.T) {
	ResetCallbackChannel()
	InitCallbackChannel()
	defer ResetCallbackChannel()

	testResult := &DeepLinkResult{Code: "wait_code", State: "wait_state"}

	// 在 goroutine 中發送
	go func() {
		time.Sleep(10 * time.Millisecond)
		SendCallback(testResult)
	}()

	// 等待接收
	result, err := WaitForCallback(1 * time.Second)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Code != "wait_code" {
		t.Errorf("expected code 'wait_code', got '%s'", result.Code)
	}
}

// TestWaitForCallback_Timeout 驗證超時返回 ErrCallbackTimeout
func TestWaitForCallback_Timeout(t *testing.T) {
	ResetCallbackChannel()
	InitCallbackChannel()
	defer ResetCallbackChannel()

	// 不發送任何東西，等待超時
	result, err := WaitForCallback(50 * time.Millisecond)

	if err != ErrCallbackTimeout {
		t.Errorf("expected ErrCallbackTimeout, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

// TestSendCallback_ReplaceOld 驗證新結果替換舊結果
func TestSendCallback_ReplaceOld(t *testing.T) {
	ResetCallbackChannel()
	InitCallbackChannel()
	defer ResetCallbackChannel()

	oldResult := &DeepLinkResult{Code: "old_code", State: "old_state"}
	newResult := &DeepLinkResult{Code: "new_code", State: "new_state"}

	// 發送舊結果
	SendCallback(oldResult)

	// 發送新結果（應該替換舊的）
	SendCallback(newResult)

	// 接收應該得到新結果
	result, err := WaitForCallback(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Code != "new_code" {
		t.Errorf("expected code 'new_code', got '%s'", result.Code)
	}
	if result.State != "new_state" {
		t.Errorf("expected state 'new_state', got '%s'", result.State)
	}
}

// TestSendCallback_NilChannel 驗證 channel 未初始化時不 panic
func TestSendCallback_NilChannel(t *testing.T) {
	ResetCallbackChannel()
	// 不初始化 channel

	testResult := &DeepLinkResult{Code: "test", State: "test"}

	// 不應該 panic
	SendCallback(testResult)
}


// ============================================
// Cold Start (Pending Deep Link) Tests
// ============================================

// TestWaitForCallback_PendingDeepLink 測試冷啟動場景
// 當應用冷啟動時，deep link 結果會先被保存到 pending，
// WaitForCallback 應該優先返回 pending 結果
func TestWaitForCallback_PendingDeepLink(t *testing.T) {
	ResetCallbackChannel()
	defer ResetCallbackChannel()

	// 模擬冷啟動：在 channel 初始化前發送結果
	pendingResult := &DeepLinkResult{Code: "cold-start-code", State: "cold-start-state"}
	SetPendingDeepLink(pendingResult)

	// WaitForCallback 應該立即返回 pending 結果
	result, err := WaitForCallback(100 * time.Millisecond)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Code != "cold-start-code" {
		t.Errorf("expected code 'cold-start-code', got '%s'", result.Code)
	}
	if result.State != "cold-start-state" {
		t.Errorf("expected state 'cold-start-state', got '%s'", result.State)
	}

	// pending 應該被清除
	if GetPendingDeepLink() != nil {
		t.Error("expected pending deep link to be cleared")
	}
}

// TestSendCallback_SaveToPending 測試 channel 未初始化時保存到 pending
func TestSendCallback_SaveToPending(t *testing.T) {
	ResetCallbackChannel()
	defer ResetCallbackChannel()

	// 確保 channel 未初始化
	testResult := &DeepLinkResult{Code: "pending-code", State: "pending-state"}

	// 發送到未初始化的 channel 應該保存到 pending
	SendCallback(testResult)

	// 驗證 pending 有值
	pending := GetPendingDeepLink()
	if pending == nil {
		t.Fatal("expected pending deep link to be set")
	}
	if pending.Code != "pending-code" {
		t.Errorf("expected code 'pending-code', got '%s'", pending.Code)
	}
	if pending.State != "pending-state" {
		t.Errorf("expected state 'pending-state', got '%s'", pending.State)
	}

	// 清理
	clearPendingDeepLink()
}

// ============================================
// Error Parameter Integration Tests
// ============================================

// TestHandleDeepLinkCallback_WithError 測試錯誤參數處理
// 當 URL 包含 error 參數時，應該返回對應的錯誤
func TestHandleDeepLinkCallback_WithError(t *testing.T) {
	// 保存有效的 state（雖然有錯誤參數時不會用到）
	state := &OAuthState{
		State:         "test-state",
		Provider:      "Github",
		CodeVerifier:  "test-verifier",
		CodeChallenge: "test-challenge",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(5 * time.Minute),
	}
	if err := SaveState(state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}
	defer ClearState()

	// URL 包含 error 參數
	errorURL := "kiro://kiro.kiroAgent/authenticate-success?error=access_denied&error_description=User%20denied%20access"

	_, err := HandleDeepLinkCallback(errorURL)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// 應該包含錯誤訊息
	errStr := err.Error()
	if !contains(errStr, "access_denied") {
		t.Errorf("expected error to contain 'access_denied', got '%s'", errStr)
	}
}

// TestHandleDeepLinkCallback_WithErrorDescription 測試錯誤描述
func TestHandleDeepLinkCallback_WithErrorDescription(t *testing.T) {
	defer ClearState()

	// URL 包含 error 和 error_description
	errorURL := "kiro://kiro.kiroAgent/authenticate-success?error=server_error&error_description=Internal%20server%20error"

	_, err := HandleDeepLinkCallback(errorURL)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errStr := err.Error()
	if !contains(errStr, "server_error") {
		t.Errorf("expected error to contain 'server_error', got '%s'", errStr)
	}
	if !contains(errStr, "Internal server error") {
		t.Errorf("expected error to contain 'Internal server error', got '%s'", errStr)
	}
}

// contains 檢查字串是否包含子字串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
