package deeplink

import (
	"testing"
	"time"
)

// TestConstants_URLScheme 驗證 URLScheme 常數正確定義
func TestConstants_URLScheme(t *testing.T) {
	expected := "kiro"
	if URLScheme != expected {
		t.Errorf("URLScheme = %q, want %q", URLScheme, expected)
	}
}

// TestConstants_RedirectURI 驗證 RedirectURI 格式正確
func TestConstants_RedirectURI(t *testing.T) {
	expected := "kiro://kiro.kiroAgent/authenticate-success"
	if RedirectURI != expected {
		t.Errorf("RedirectURI = %q, want %q", RedirectURI, expected)
	}
}

// TestConstants_StateFileName 驗證 StateFileName 正確
func TestConstants_StateFileName(t *testing.T) {
	expected := "kiro-manager-oauth-state.json"
	if StateFileName != expected {
		t.Errorf("StateFileName = %q, want %q", StateFileName, expected)
	}
}

// TestConstants_StateExpiry 驗證 StateExpiry 為 5 分鐘
func TestConstants_StateExpiry(t *testing.T) {
	expected := 5 * time.Minute
	if StateExpiry != expected {
		t.Errorf("StateExpiry = %v, want %v", StateExpiry, expected)
	}
}

// TestErrors_Defined 驗證所有錯誤類型已定義且不為 nil
func TestErrors_Defined(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrNotWindows", ErrNotWindows, "deep link only supported on Windows"},
		{"ErrRegistryFailed", ErrRegistryFailed, "failed to register URL scheme"},
		{"ErrStateNotFound", ErrStateNotFound, "oauth state not found"},
		{"ErrStateExpired", ErrStateExpired, "oauth state expired"},
		{"ErrStateMismatch", ErrStateMismatch, "oauth state mismatch"},
		{"ErrMissingCode", ErrMissingCode, "missing authorization code"},
		{"ErrInvalidScheme", ErrInvalidScheme, "invalid URL scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil, want non-nil error", tt.name)
				return
			}
			if tt.err.Error() != tt.msg {
				t.Errorf("%s.Error() = %q, want %q", tt.name, tt.err.Error(), tt.msg)
			}
		})
	}
}
