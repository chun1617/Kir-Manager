package oauthlogin

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

// TestProperty_PKCEParameterGenerationValidity 測試 PKCE 參數生成有效性
// Property 1: 每次生成的 PKCE 參數都應該是有效的
// - code_verifier 長度應為 43 字元 (32 bytes base64url = 43 chars)
// - code_challenge 長度應為 43 字元 (32 bytes SHA256 base64url = 43 chars)
// - state 長度應為 22 字元 (16 bytes base64url = 22 chars)
// - 多次生成應產生不同的值
func TestProperty_PKCEParameterGenerationValidity(t *testing.T) {
	const iterations = 100
	seenVerifiers := make(map[string]bool)
	seenChallenges := make(map[string]bool)
	seenStates := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		params, err := GeneratePKCE()
		if err != nil {
			t.Fatalf("GeneratePKCE() failed: %v", err)
		}

		// 驗證 code_verifier 長度 (32 bytes -> 43 chars base64url)
		if len(params.CodeVerifier) != 43 {
			t.Errorf("CodeVerifier length = %d, want 43", len(params.CodeVerifier))
		}

		// 驗證 code_challenge 長度 (32 bytes SHA256 -> 43 chars base64url)
		if len(params.CodeChallenge) != 43 {
			t.Errorf("CodeChallenge length = %d, want 43", len(params.CodeChallenge))
		}

		// 驗證 state 長度 (16 bytes -> 22 chars base64url)
		if len(params.State) != 22 {
			t.Errorf("State length = %d, want 22", len(params.State))
		}

		// 驗證 base64url 解碼有效性
		if _, err := base64.RawURLEncoding.DecodeString(params.CodeVerifier); err != nil {
			t.Errorf("CodeVerifier is not valid base64url: %v", err)
		}
		if _, err := base64.RawURLEncoding.DecodeString(params.CodeChallenge); err != nil {
			t.Errorf("CodeChallenge is not valid base64url: %v", err)
		}
		if _, err := base64.RawURLEncoding.DecodeString(params.State); err != nil {
			t.Errorf("State is not valid base64url: %v", err)
		}

		// 記錄唯一性
		seenVerifiers[params.CodeVerifier] = true
		seenChallenges[params.CodeChallenge] = true
		seenStates[params.State] = true
	}

	// 驗證唯一性 - 100 次生成應該產生接近 100 個不同的值
	if len(seenVerifiers) < iterations*9/10 {
		t.Errorf("CodeVerifier uniqueness too low: %d/%d", len(seenVerifiers), iterations)
	}
	if len(seenChallenges) < iterations*9/10 {
		t.Errorf("CodeChallenge uniqueness too low: %d/%d", len(seenChallenges), iterations)
	}
	if len(seenStates) < iterations*9/10 {
		t.Errorf("State uniqueness too low: %d/%d", len(seenStates), iterations)
	}
}

// TestProperty_PKCECodeChallengeRoundTrip 測試 code_challenge 計算正確性
// Property 2: code_challenge 應該是 code_verifier 的 SHA256 雜湊
func TestProperty_PKCECodeChallengeRoundTrip(t *testing.T) {
	const iterations = 100

	for i := 0; i < iterations; i++ {
		params, err := GeneratePKCE()
		if err != nil {
			t.Fatalf("GeneratePKCE() failed: %v", err)
		}

		// 手動計算 code_challenge
		hash := sha256.Sum256([]byte(params.CodeVerifier))
		expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

		if params.CodeChallenge != expectedChallenge {
			t.Errorf("CodeChallenge mismatch:\ngot:  %s\nwant: %s",
				params.CodeChallenge, expectedChallenge)
		}
	}
}

// TestValidateState 測試 state 驗證函數
func TestValidateState(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   string
		want     bool
	}{
		{"matching states", "abc123", "abc123", true},
		{"mismatched states", "abc123", "xyz789", false},
		{"empty expected", "", "abc123", false},
		{"empty actual", "abc123", "", false},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateState(tt.expected, tt.actual); got != tt.want {
				t.Errorf("ValidateState(%q, %q) = %v, want %v",
					tt.expected, tt.actual, got, tt.want)
			}
		})
	}
}
