package oauthlogin

import (
	"testing"
	"time"
)

// TestOAuthError_ImplementsErrorInterface 驗證 OAuthError 實作 error 介面
func TestOAuthError_ImplementsErrorInterface(t *testing.T) {
	var err error = &OAuthError{
		Code:    ErrCodeTimeout,
		Message: "login timeout",
	}

	if err == nil {
		t.Error("OAuthError should implement error interface")
	}

	expected := "login timeout"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

// TestOAuthError_ErrorCodes 驗證所有預定義錯誤碼常數存在
func TestOAuthError_ErrorCodes(t *testing.T) {
	testCases := []struct {
		name     string
		code     string
		expected string
	}{
		{"ErrCodeTimeout", ErrCodeTimeout, "timeout"},
		{"ErrCodeCancelled", ErrCodeCancelled, "cancelled"},
		{"ErrCodeInvalidCode", ErrCodeInvalidCode, "invalid_code"},
		{"ErrCodeAuthFailed", ErrCodeAuthFailed, "auth_failed"},
		{"ErrCodeServerError", ErrCodeServerError, "server_error"},
		{"ErrCodeNetworkError", ErrCodeNetworkError, "network_error"},
		{"ErrCodeStateMismatch", ErrCodeStateMismatch, "state_mismatch"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.code != tc.expected {
				t.Errorf("%s = %q, want %q", tc.name, tc.code, tc.expected)
			}
		})
	}
}

// TestOAuthError_ErrorMethod 驗證 Error() 方法回傳正確訊息
func TestOAuthError_ErrorMethod(t *testing.T) {
	testCases := []struct {
		name     string
		err      *OAuthError
		expected string
	}{
		{
			name:     "with message",
			err:      &OAuthError{Code: ErrCodeTimeout, Message: "operation timed out"},
			expected: "operation timed out",
		},
		{
			name:     "empty message",
			err:      &OAuthError{Code: ErrCodeCancelled, Message: ""},
			expected: "",
		},
		{
			name:     "with code only",
			err:      &OAuthError{Code: ErrCodeAuthFailed},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Error() != tc.expected {
				t.Errorf("Error() = %q, want %q", tc.err.Error(), tc.expected)
			}
		})
	}
}

// TestLoginResult_StructFields 驗證 LoginResult 結構包含所有必要欄位
func TestLoginResult_StructFields(t *testing.T) {
	now := time.Now()
	result := LoginResult{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresIn:    3600,
		ExpiresAt:    now,
		ProfileArn:   "arn:aws:kiro::123456789012:profile/test",
		Provider:     ProviderGithub,
		AuthMethod:   AuthMethodSocial,
		ClientId:     "client-id-123",
		ClientSecret: "client-secret-456",
		ClientIdHash: "hash-789",
	}

	// 驗證所有欄位都能正確設定和讀取
	if result.AccessToken != "test-access-token" {
		t.Errorf("AccessToken = %q, want %q", result.AccessToken, "test-access-token")
	}
	if result.RefreshToken != "test-refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", result.RefreshToken, "test-refresh-token")
	}
	if result.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want %d", result.ExpiresIn, 3600)
	}
	if !result.ExpiresAt.Equal(now) {
		t.Errorf("ExpiresAt = %v, want %v", result.ExpiresAt, now)
	}
	if result.ProfileArn != "arn:aws:kiro::123456789012:profile/test" {
		t.Errorf("ProfileArn = %q, want %q", result.ProfileArn, "arn:aws:kiro::123456789012:profile/test")
	}
	if result.Provider != ProviderGithub {
		t.Errorf("Provider = %q, want %q", result.Provider, ProviderGithub)
	}
	if result.AuthMethod != AuthMethodSocial {
		t.Errorf("AuthMethod = %q, want %q", result.AuthMethod, AuthMethodSocial)
	}
	if result.ClientId != "client-id-123" {
		t.Errorf("ClientId = %q, want %q", result.ClientId, "client-id-123")
	}
	if result.ClientSecret != "client-secret-456" {
		t.Errorf("ClientSecret = %q, want %q", result.ClientSecret, "client-secret-456")
	}
	if result.ClientIdHash != "hash-789" {
		t.Errorf("ClientIdHash = %q, want %q", result.ClientIdHash, "hash-789")
	}
}

// TestProviderConstants 驗證 Provider 常數定義正確
func TestProviderConstants(t *testing.T) {
	testCases := []struct {
		name     string
		constant string
		expected string
	}{
		{"ProviderGithub", ProviderGithub, "Github"},
		{"ProviderGoogle", ProviderGoogle, "Google"},
		{"ProviderBuilderID", ProviderBuilderID, "BuilderID"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.constant != tc.expected {
				t.Errorf("%s = %q, want %q", tc.name, tc.constant, tc.expected)
			}
		})
	}
}

// TestAuthMethodConstants 驗證 AuthMethod 常數定義正確
func TestAuthMethodConstants(t *testing.T) {
	testCases := []struct {
		name     string
		constant string
		expected string
	}{
		{"AuthMethodSocial", AuthMethodSocial, "social"},
		{"AuthMethodIdC", AuthMethodIdC, "idc"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.constant != tc.expected {
				t.Errorf("%s = %q, want %q", tc.name, tc.constant, tc.expected)
			}
		})
	}
}

// TestLoginResult_ZeroValue 驗證 LoginResult 零值行為
func TestLoginResult_ZeroValue(t *testing.T) {
	var result LoginResult

	if result.AccessToken != "" {
		t.Errorf("Zero value AccessToken should be empty, got %q", result.AccessToken)
	}
	if result.RefreshToken != "" {
		t.Errorf("Zero value RefreshToken should be empty, got %q", result.RefreshToken)
	}
	if result.ExpiresIn != 0 {
		t.Errorf("Zero value ExpiresIn should be 0, got %d", result.ExpiresIn)
	}
	if !result.ExpiresAt.IsZero() {
		t.Errorf("Zero value ExpiresAt should be zero time, got %v", result.ExpiresAt)
	}
	if result.ProfileArn != "" {
		t.Errorf("Zero value ProfileArn should be empty, got %q", result.ProfileArn)
	}
	if result.Provider != "" {
		t.Errorf("Zero value Provider should be empty, got %q", result.Provider)
	}
	if result.AuthMethod != "" {
		t.Errorf("Zero value AuthMethod should be empty, got %q", result.AuthMethod)
	}
	if result.ClientId != "" {
		t.Errorf("Zero value ClientId should be empty, got %q", result.ClientId)
	}
	if result.ClientSecret != "" {
		t.Errorf("Zero value ClientSecret should be empty, got %q", result.ClientSecret)
	}
	if result.ClientIdHash != "" {
		t.Errorf("Zero value ClientIdHash should be empty, got %q", result.ClientIdHash)
	}
}

// TestOAuthError_CodeAndMessage 驗證 OAuthError 的 Code 和 Message 欄位
func TestOAuthError_CodeAndMessage(t *testing.T) {
	err := &OAuthError{
		Code:    ErrCodeNetworkError,
		Message: "connection refused",
	}

	if err.Code != ErrCodeNetworkError {
		t.Errorf("Code = %q, want %q", err.Code, ErrCodeNetworkError)
	}
	if err.Message != "connection refused" {
		t.Errorf("Message = %q, want %q", err.Message, "connection refused")
	}
}
