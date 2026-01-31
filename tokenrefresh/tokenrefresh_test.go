package tokenrefresh

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"kiro-manager/awssso"
)

// generateRandomString 生成指定長度的隨機字串
func generateRandomString(r *rand.Rand, length int) string {
	if length <= 0 {
		return ""
	}
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[r.Intn(len(charset))]
	}
	return string(result)
}

// generateRandomSocialResponse 生成隨機的 Social 刷新回應
func generateRandomSocialResponse(r *rand.Rand) SocialRefreshResponse {
	return SocialRefreshResponse{
		AccessToken:  generateRandomString(r, r.Intn(100)+10), // 10-109 字元
		ExpiresIn:    r.Intn(86400) + 60,                      // 60 秒到 24 小時
		RefreshToken: generateRandomString(r, r.Intn(100)+10),
		ProfileArn:   "arn:aws:kiro::" + generateRandomString(r, 12) + ":profile/" + generateRandomString(r, 8),
	}
}

// **Feature: token-refresh, Property 5: Response Field Extraction (Social)**
// *For any* valid Social refresh API response, the TokenInfo SHALL contain
// the accessToken, expiresIn, and profileArn fields from the response.
// **Validates: Requirements 5.1**
func TestProperty_SocialResponseFieldExtraction(t *testing.T) {
	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		socialResp := generateRandomSocialResponse(r)

		// 序列化為 JSON
		jsonData, err := json.Marshal(socialResp)
		if err != nil {
			t.Logf("Failed to marshal response: %v", err)
			return false
		}

		// 記錄解析前的時間（用於驗證 ExpiresAt）
		beforeParse := time.Now()

		// 使用 ParseSocialResponse 解析
		tokenInfo, err := ParseSocialResponse(jsonData)
		if err != nil {
			t.Logf("Failed to parse response: %v", err)
			return false
		}

		afterParse := time.Now()

		// Property 5.1: accessToken 必須正確提取
		if tokenInfo.AccessToken != socialResp.AccessToken {
			t.Logf("AccessToken mismatch: got %q, expected %q",
				tokenInfo.AccessToken, socialResp.AccessToken)
			return false
		}

		// Property 5.1: expiresIn 必須正確提取
		if tokenInfo.ExpiresIn != socialResp.ExpiresIn {
			t.Logf("ExpiresIn mismatch: got %d, expected %d",
				tokenInfo.ExpiresIn, socialResp.ExpiresIn)
			return false
		}

		// Property 5.1: profileArn 必須正確提取
		if tokenInfo.ProfileArn != socialResp.ProfileArn {
			t.Logf("ProfileArn mismatch: got %q, expected %q",
				tokenInfo.ProfileArn, socialResp.ProfileArn)
			return false
		}

		// Property 5.3: ExpiresAt 必須在合理範圍內
		// ExpiresAt 應該在 beforeParse + expiresIn 和 afterParse + expiresIn 之間
		expectedMinExpiresAt := beforeParse.Add(time.Duration(socialResp.ExpiresIn) * time.Second)
		expectedMaxExpiresAt := afterParse.Add(time.Duration(socialResp.ExpiresIn) * time.Second)

		if tokenInfo.ExpiresAt.Before(expectedMinExpiresAt) || tokenInfo.ExpiresAt.After(expectedMaxExpiresAt) {
			t.Logf("ExpiresAt out of range: got %v, expected between %v and %v",
				tokenInfo.ExpiresAt, expectedMinExpiresAt, expectedMaxExpiresAt)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// **Feature: token-refresh, Property 2: ExpiresAt Calculation Correctness**
// *For any* successful refresh response with expiresIn value N, the calculated
// ExpiresAt SHALL be within 1 second of (current_time + N seconds) and formatted as RFC3339.
// **Validates: Requirements 5.3**
func TestProperty_ExpiresAtCalculationCorrectness(t *testing.T) {
	f := func(expiresIn int) bool {
		// 限制 expiresIn 在合理範圍內（1 秒到 30 天）
		if expiresIn < 1 {
			expiresIn = 1
		}
		if expiresIn > 2592000 {
			expiresIn = 2592000
		}

		// 測試 CalculateExpiresAt（返回 time.Time）
		before := time.Now()
		result := CalculateExpiresAt(expiresIn)
		after := time.Now()

		// 計算預期的最小和最大值
		expectedMin := before.Add(time.Duration(expiresIn) * time.Second)
		expectedMax := after.Add(time.Duration(expiresIn) * time.Second)

		// Property 2: 結果應該在預期範圍內（within 1 second）
		if result.Before(expectedMin) || result.After(expectedMax) {
			t.Logf("ExpiresAt out of range: got %v, expected between %v and %v",
				result, expectedMin, expectedMax)
			return false
		}

		// 驗證時間差在 1 秒內
		diff := math.Abs(result.Sub(expectedMin).Seconds())
		if diff > 1.0 {
			t.Logf("ExpiresAt diff too large: %f seconds", diff)
			return false
		}

		// 測試 CalculateExpiresAtString（返回 RFC3339 字串）
		beforeStr := time.Now()
		resultStr := CalculateExpiresAtString(expiresIn)
		afterStr := time.Now()

		// Property 2: 結果必須是有效的 RFC3339 格式
		parsedTime, err := time.Parse(time.RFC3339, resultStr)
		if err != nil {
			t.Logf("Failed to parse RFC3339 string %q: %v", resultStr, err)
			return false
		}

		// 驗證解析後的時間在預期範圍內
		// 注意：RFC3339 格式化會截斷亞秒精度，所以需要允許 1 秒的誤差
		expectedMinStr := beforeStr.Add(time.Duration(expiresIn) * time.Second).Add(-1 * time.Second)
		expectedMaxStr := afterStr.Add(time.Duration(expiresIn) * time.Second).Add(1 * time.Second)

		if parsedTime.Before(expectedMinStr) || parsedTime.After(expectedMaxStr) {
			t.Logf("Parsed ExpiresAt out of range: got %v, expected between %v and %v",
				parsedTime, expectedMinStr, expectedMaxStr)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestParseSocialResponse_InvalidJSON 測試無效 JSON 的處理
func TestParseSocialResponse_InvalidJSON(t *testing.T) {
	invalidJSONs := [][]byte{
		[]byte(""),
		[]byte("not json"),
		[]byte("{invalid}"),
		[]byte("[1,2,3]"),
	}

	for _, jsonData := range invalidJSONs {
		_, err := ParseSocialResponse(jsonData)
		if err == nil {
			t.Errorf("Expected error for invalid JSON: %s", string(jsonData))
		}

		// 確認錯誤類型是 RefreshError
		if _, ok := err.(*RefreshError); !ok {
			t.Errorf("Expected RefreshError, got %T", err)
		}
	}
}

// TestParseSocialResponse_EmptyFields 測試空欄位的處理
func TestParseSocialResponse_EmptyFields(t *testing.T) {
	// 空的 JSON 物件
	jsonData := []byte(`{}`)
	tokenInfo, err := ParseSocialResponse(jsonData)
	if err != nil {
		t.Errorf("Unexpected error for empty object: %v", err)
	}

	// 應該返回空值
	if tokenInfo.AccessToken != "" {
		t.Errorf("Expected empty AccessToken, got %q", tokenInfo.AccessToken)
	}
	if tokenInfo.ExpiresIn != 0 {
		t.Errorf("Expected zero ExpiresIn, got %d", tokenInfo.ExpiresIn)
	}
	if tokenInfo.ProfileArn != "" {
		t.Errorf("Expected empty ProfileArn, got %q", tokenInfo.ProfileArn)
	}
}


// generateRandomIdCResponse 生成隨機的 IdC 刷新回應
func generateRandomIdCResponse(r *rand.Rand) IdCRefreshResponse {
	tokenTypes := []string{"Bearer", "bearer", "JWT"}
	return IdCRefreshResponse{
		AccessToken:  generateRandomString(r, r.Intn(200)+50), // 50-249 字元（IdC token 通常較長）
		ExpiresIn:    r.Intn(28800) + 3600,                    // 1 小時到 8 小時
		TokenType:    tokenTypes[r.Intn(len(tokenTypes))],
		RefreshToken: generateRandomString(r, r.Intn(100)+10),
	}
}

// **Feature: token-refresh, Property 5: Response Field Extraction (IdC)**
// *For any* valid IdC refresh API response, the TokenInfo SHALL contain
// the accessToken, expiresIn, and tokenType fields from the response.
// **Validates: Requirements 5.2**
func TestProperty_IdCResponseFieldExtraction(t *testing.T) {
	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		idcResp := generateRandomIdCResponse(r)

		// 序列化為 JSON
		jsonData, err := json.Marshal(idcResp)
		if err != nil {
			t.Logf("Failed to marshal response: %v", err)
			return false
		}

		// 記錄解析前的時間（用於驗證 ExpiresAt）
		beforeParse := time.Now()

		// 使用 ParseIdCResponse 解析
		tokenInfo, err := ParseIdCResponse(jsonData)
		if err != nil {
			t.Logf("Failed to parse response: %v", err)
			return false
		}

		afterParse := time.Now()

		// Property 5.2: accessToken 必須正確提取
		if tokenInfo.AccessToken != idcResp.AccessToken {
			t.Logf("AccessToken mismatch: got %q, expected %q",
				tokenInfo.AccessToken, idcResp.AccessToken)
			return false
		}

		// Property 5.2: expiresIn 必須正確提取
		if tokenInfo.ExpiresIn != idcResp.ExpiresIn {
			t.Logf("ExpiresIn mismatch: got %d, expected %d",
				tokenInfo.ExpiresIn, idcResp.ExpiresIn)
			return false
		}

		// Property 5.2: tokenType 必須正確提取
		if tokenInfo.TokenType != idcResp.TokenType {
			t.Logf("TokenType mismatch: got %q, expected %q",
				tokenInfo.TokenType, idcResp.TokenType)
			return false
		}

		// Property 5.3: ExpiresAt 必須在合理範圍內
		// ExpiresAt 應該在 beforeParse + expiresIn 和 afterParse + expiresIn 之間
		expectedMinExpiresAt := beforeParse.Add(time.Duration(idcResp.ExpiresIn) * time.Second)
		expectedMaxExpiresAt := afterParse.Add(time.Duration(idcResp.ExpiresIn) * time.Second)

		if tokenInfo.ExpiresAt.Before(expectedMinExpiresAt) || tokenInfo.ExpiresAt.After(expectedMaxExpiresAt) {
			t.Logf("ExpiresAt out of range: got %v, expected between %v and %v",
				tokenInfo.ExpiresAt, expectedMinExpiresAt, expectedMaxExpiresAt)
			return false
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestParseIdCResponse_InvalidJSON 測試無效 JSON 的處理
func TestParseIdCResponse_InvalidJSON(t *testing.T) {
	invalidJSONs := [][]byte{
		[]byte(""),
		[]byte("not json"),
		[]byte("{invalid}"),
		[]byte("[1,2,3]"),
	}

	for _, jsonData := range invalidJSONs {
		_, err := ParseIdCResponse(jsonData)
		if err == nil {
			t.Errorf("Expected error for invalid JSON: %s", string(jsonData))
		}

		// 確認錯誤類型是 RefreshError
		if _, ok := err.(*RefreshError); !ok {
			t.Errorf("Expected RefreshError, got %T", err)
		}
	}
}

// TestParseIdCResponse_EmptyFields 測試空欄位的處理
func TestParseIdCResponse_EmptyFields(t *testing.T) {
	// 空的 JSON 物件
	jsonData := []byte(`{}`)
	tokenInfo, err := ParseIdCResponse(jsonData)
	if err != nil {
		t.Errorf("Unexpected error for empty object: %v", err)
	}

	// 應該返回空值
	if tokenInfo.AccessToken != "" {
		t.Errorf("Expected empty AccessToken, got %q", tokenInfo.AccessToken)
	}
	if tokenInfo.ExpiresIn != 0 {
		t.Errorf("Expected zero ExpiresIn, got %d", tokenInfo.ExpiresIn)
	}
	if tokenInfo.TokenType != "" {
		t.Errorf("Expected empty TokenType, got %q", tokenInfo.TokenType)
	}
}


// **Feature: token-refresh, Property 3: Authentication Type Routing**
// *For any* KiroAuthToken, the refresh function SHALL route to Social refresh
// when AuthMethod is "social" and to IdC refresh when AuthMethod is "idc"
// or when StartURL and Region are present.
// **Validates: Requirements 2.1, 2.2, 2.4**
func TestProperty_AuthenticationTypeRouting(t *testing.T) {
	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// 生成隨機的 KiroAuthToken
		token := generateRandomKiroAuthToken(r)

		// 偵測認證類型
		authType := DetectAuthType(token)

		// 驗證路由邏輯
		switch {
		// 規則 1: AuthMethod 為 "social" 時應路由到 social
		case strings.ToLower(token.AuthMethod) == "social":
			if authType != "social" {
				t.Logf("Expected 'social' for AuthMethod='social', got %q", authType)
				return false
			}

		// 規則 2: AuthMethod 為 "idc" 或 "identitycenter" 時應路由到 idc
		case strings.ToLower(token.AuthMethod) == "idc" || strings.ToLower(token.AuthMethod) == "identitycenter":
			if authType != "idc" {
				t.Logf("Expected 'idc' for AuthMethod=%q, got %q", token.AuthMethod, authType)
				return false
			}

		// 規則 3: 有 StartURL 和 Region 時應路由到 idc
		case token.AuthMethod == "" && token.StartURL != "" && token.Region != "":
			if authType != "idc" {
				t.Logf("Expected 'idc' for StartURL=%q and Region=%q, got %q",
					token.StartURL, token.Region, authType)
				return false
			}

		// 規則 4: 有 Provider 時應路由到 social
		case token.AuthMethod == "" && token.Provider != "" && token.StartURL == "":
			if authType != "social" {
				t.Logf("Expected 'social' for Provider=%q, got %q", token.Provider, authType)
				return false
			}

		// 規則 5: 有 ProfileArn 但沒有 StartURL 時應路由到 social
		case token.AuthMethod == "" && token.ProfileArn != "" && token.StartURL == "":
			if authType != "social" {
				t.Logf("Expected 'social' for ProfileArn=%q without StartURL, got %q",
					token.ProfileArn, authType)
				return false
			}

		// 規則 6: 其他情況應返回 unknown
		case token.AuthMethod == "" && token.StartURL == "" && token.Provider == "" && token.ProfileArn == "":
			if authType != "unknown" {
				t.Logf("Expected 'unknown' for empty token, got %q", authType)
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// generateRandomKiroAuthToken 生成隨機的 KiroAuthToken 用於測試
func generateRandomKiroAuthToken(r *rand.Rand) *awssso.KiroAuthToken {
	// 定義可能的 AuthMethod 值（包含空值以測試其他判斷邏輯）
	authMethods := []string{"", "social", "Social", "SOCIAL", "idc", "IdC", "IDC", "identitycenter", "IdentityCenter", "unknown", "other"}
	providers := []string{"", "Github", "Google", "AWS", "Microsoft"}
	regions := []string{"", "us-east-1", "us-west-2", "eu-west-1", "ap-northeast-1"}

	token := &awssso.KiroAuthToken{
		AuthMethod: authMethods[r.Intn(len(authMethods))],
	}

	// 隨機決定是否填充其他欄位
	if r.Float32() > 0.3 {
		token.AccessToken = generateRandomString(r, r.Intn(100)+10)
	}
	if r.Float32() > 0.3 {
		token.RefreshToken = generateRandomString(r, r.Intn(100)+10)
	}
	if r.Float32() > 0.5 {
		token.Provider = providers[r.Intn(len(providers))]
	}
	if r.Float32() > 0.5 {
		token.Region = regions[r.Intn(len(regions))]
	}
	if r.Float32() > 0.5 {
		token.StartURL = "https://d-" + generateRandomString(r, 10) + ".awsapps.com/start"
	}
	if r.Float32() > 0.5 {
		token.ProfileArn = "arn:aws:kiro::" + generateRandomString(r, 12) + ":profile/" + generateRandomString(r, 8)
	}

	return token
}

// TestDetectAuthType_SocialExplicit 測試明確的 Social 認證類型
func TestDetectAuthType_SocialExplicit(t *testing.T) {
	testCases := []struct {
		name     string
		token    *awssso.KiroAuthToken
		expected string
	}{
		{
			name:     "AuthMethod=social",
			token:    &awssso.KiroAuthToken{AuthMethod: "social"},
			expected: "social",
		},
		{
			name:     "AuthMethod=Social (大小寫)",
			token:    &awssso.KiroAuthToken{AuthMethod: "Social"},
			expected: "social",
		},
		{
			name:     "AuthMethod=SOCIAL (全大寫)",
			token:    &awssso.KiroAuthToken{AuthMethod: "SOCIAL"},
			expected: "social",
		},
		{
			name:     "Provider=Github (無 AuthMethod)",
			token:    &awssso.KiroAuthToken{Provider: "Github"},
			expected: "social",
		},
		{
			name:     "ProfileArn 存在 (無 StartURL)",
			token:    &awssso.KiroAuthToken{ProfileArn: "arn:aws:kiro::123456789012:profile/test"},
			expected: "social",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DetectAuthType(tc.token)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestDetectAuthType_IdCExplicit 測試明確的 IdC 認證類型
func TestDetectAuthType_IdCExplicit(t *testing.T) {
	testCases := []struct {
		name     string
		token    *awssso.KiroAuthToken
		expected string
	}{
		{
			name:     "AuthMethod=idc",
			token:    &awssso.KiroAuthToken{AuthMethod: "idc"},
			expected: "idc",
		},
		{
			name:     "AuthMethod=IdC (大小寫)",
			token:    &awssso.KiroAuthToken{AuthMethod: "IdC"},
			expected: "idc",
		},
		{
			name:     "AuthMethod=IDC (全大寫)",
			token:    &awssso.KiroAuthToken{AuthMethod: "IDC"},
			expected: "idc",
		},
		{
			name:     "AuthMethod=identitycenter",
			token:    &awssso.KiroAuthToken{AuthMethod: "identitycenter"},
			expected: "idc",
		},
		{
			name:     "AuthMethod=IdentityCenter (大小寫)",
			token:    &awssso.KiroAuthToken{AuthMethod: "IdentityCenter"},
			expected: "idc",
		},
		{
			name:     "StartURL 和 Region 存在 (無 AuthMethod)",
			token:    &awssso.KiroAuthToken{StartURL: "https://d-123456.awsapps.com/start", Region: "us-east-1"},
			expected: "idc",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DetectAuthType(tc.token)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestDetectAuthType_Unknown 測試未知認證類型
func TestDetectAuthType_Unknown(t *testing.T) {
	testCases := []struct {
		name     string
		token    *awssso.KiroAuthToken
		expected string
	}{
		{
			name:     "空 token",
			token:    &awssso.KiroAuthToken{},
			expected: "unknown",
		},
		{
			name:     "nil token",
			token:    nil,
			expected: "unknown",
		},
		{
			name:     "只有 AccessToken",
			token:    &awssso.KiroAuthToken{AccessToken: "some-token"},
			expected: "unknown",
		},
		{
			name:     "未知的 AuthMethod",
			token:    &awssso.KiroAuthToken{AuthMethod: "unknown-method"},
			expected: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DetectAuthType(tc.token)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestRefreshAccessToken_NilToken 測試 nil token 的處理
func TestRefreshAccessToken_NilToken(t *testing.T) {
	_, err := RefreshAccessToken(nil, "test-machine-id")
	if err == nil {
		t.Error("Expected error for nil token")
	}

	refreshErr, ok := err.(*RefreshError)
	if !ok {
		t.Errorf("Expected RefreshError, got %T", err)
	}

	if refreshErr.Message != "Token 不可為空" {
		t.Errorf("Expected 'Token 不可為空', got %q", refreshErr.Message)
	}
}

// TestRefreshAccessToken_EmptyMachineId 測試空 machineId 的處理
func TestRefreshAccessToken_EmptyMachineId(t *testing.T) {
	token := &awssso.KiroAuthToken{
		AuthMethod:   "social",
		RefreshToken: "some-refresh-token",
	}

	_, err := RefreshAccessToken(token, "")
	if err == nil {
		t.Error("Expected error for empty machineId")
	}

	refreshErr, ok := err.(*RefreshError)
	if !ok {
		t.Errorf("Expected RefreshError, got %T", err)
	}

	if refreshErr.Message != "machineId 不可為空" {
		t.Errorf("Expected 'machineId 不可為空', got %q", refreshErr.Message)
	}
}

// TestRefreshAccessToken_EmptyRefreshToken 測試空 RefreshToken 的處理
func TestRefreshAccessToken_EmptyRefreshToken(t *testing.T) {
	// Social 認證但沒有 RefreshToken
	token := &awssso.KiroAuthToken{
		AuthMethod: "social",
	}

	_, err := RefreshAccessToken(token, "test-machine-id")
	if err == nil {
		t.Error("Expected error for empty RefreshToken")
	}

	refreshErr, ok := err.(*RefreshError)
	if !ok {
		t.Errorf("Expected RefreshError, got %T", err)
	}

	if refreshErr.Message != "RefreshToken 不可為空" {
		t.Errorf("Expected 'RefreshToken 不可為空', got %q", refreshErr.Message)
	}
}

// TestRefreshAccessToken_UnknownAuthType 測試未知認證類型的處理
func TestRefreshAccessToken_UnknownAuthType(t *testing.T) {
	token := &awssso.KiroAuthToken{
		RefreshToken: "some-refresh-token",
		// 沒有任何可識別認證類型的欄位
	}

	_, err := RefreshAccessToken(token, "test-machine-id")
	if err == nil {
		t.Error("Expected error for unknown auth type")
	}

	refreshErr, ok := err.(*RefreshError)
	if !ok {
		t.Errorf("Expected RefreshError, got %T", err)
	}

	if !strings.Contains(refreshErr.Message, "不支援的認證類型") {
		t.Errorf("Expected error message to contain '不支援的認證類型', got %q", refreshErr.Message)
	}
}


// **Feature: token-refresh, Property 4: HTTP Error Code Mapping**
// *For any* HTTP error response with status code C, the returned RefreshError
// SHALL have Code equal to C and an appropriate user-friendly Message based
// on the status code category (401/403 → invalid token, 429 → rate limit, 5xx → server error).
// **Validates: Requirements 4.1, 4.2, 4.3**
func TestProperty_HTTPErrorCodeMapping(t *testing.T) {
	f := func(statusCode int) bool {
		// 限制狀態碼在有效的 HTTP 錯誤範圍內（100-599）
		// 排除 200 系列（成功狀態碼）
		if statusCode < 100 || statusCode > 599 || (statusCode >= 200 && statusCode < 300) {
			// 將無效的狀態碼映射到有效範圍
			statusCode = (statusCode%400 + 400) % 500
			if statusCode >= 200 && statusCode < 300 {
				statusCode = 400
			}
		}

		// 呼叫 MapHTTPError
		refreshErr := MapHTTPError(statusCode, "test error body")

		// Property 4: Code 必須等於輸入的 statusCode
		if refreshErr.Code != statusCode {
			t.Logf("Code mismatch: got %d, expected %d", refreshErr.Code, statusCode)
			return false
		}

		// Property 4.1: HTTP 401/403 應映射為「Token 已失效，請重新登入 Kiro」
		if statusCode == 401 || statusCode == 403 {
			expectedMsg := "Token 已失效，請重新登入 Kiro"
			if refreshErr.Message != expectedMsg {
				t.Logf("Message mismatch for %d: got %q, expected %q",
					statusCode, refreshErr.Message, expectedMsg)
				return false
			}
		}

		// Property 4.2: HTTP 429 應映射為「請求過於頻繁，請稍後再試」
		if statusCode == 429 {
			expectedMsg := "請求過於頻繁，請稍後再試"
			if refreshErr.Message != expectedMsg {
				t.Logf("Message mismatch for 429: got %q, expected %q",
					refreshErr.Message, expectedMsg)
				return false
			}
		}

		// Property 4.3: HTTP 5xx 應映射為「伺服器暫時無法使用，請稍後再試」
		if statusCode >= 500 && statusCode < 600 {
			expectedMsg := "伺服器暫時無法使用，請稍後再試"
			if refreshErr.Message != expectedMsg {
				t.Logf("Message mismatch for %d: got %q, expected %q",
					statusCode, refreshErr.Message, expectedMsg)
				return false
			}
		}

		// 其他狀態碼應映射為「Token 刷新失敗 (HTTP xxx): body」格式
		if statusCode != 401 && statusCode != 403 && statusCode != 429 &&
			!(statusCode >= 500 && statusCode < 600) {
			expectedPrefix := fmt.Sprintf("Token 刷新失敗 (HTTP %d):", statusCode)
			if !strings.HasPrefix(refreshErr.Message, expectedPrefix) {
				t.Logf("Message mismatch for %d: got %q, expected prefix %q",
					statusCode, refreshErr.Message, expectedPrefix)
				return false
			}
		}

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestMapHTTPError_SpecificCodes 測試特定 HTTP 狀態碼的映射
func TestMapHTTPError_SpecificCodes(t *testing.T) {
	testCases := []struct {
		name        string
		statusCode  int
		expectedMsg string
	}{
		// 需求 4.1: HTTP 401/403 映射為「Token 已失效，請重新登入 Kiro」
		{
			name:        "HTTP 401 Unauthorized",
			statusCode:  401,
			expectedMsg: "Token 已失效，請重新登入 Kiro",
		},
		{
			name:        "HTTP 403 Forbidden",
			statusCode:  403,
			expectedMsg: "Token 已失效，請重新登入 Kiro",
		},
		// 需求 4.2: HTTP 429 映射為「請求過於頻繁，請稍後再試」
		{
			name:        "HTTP 429 Too Many Requests",
			statusCode:  429,
			expectedMsg: "請求過於頻繁，請稍後再試",
		},
		// 需求 4.3: HTTP 5xx 映射為「伺服器暫時無法使用，請稍後再試」
		{
			name:        "HTTP 500 Internal Server Error",
			statusCode:  500,
			expectedMsg: "伺服器暫時無法使用，請稍後再試",
		},
		{
			name:        "HTTP 502 Bad Gateway",
			statusCode:  502,
			expectedMsg: "伺服器暫時無法使用，請稍後再試",
		},
		{
			name:        "HTTP 503 Service Unavailable",
			statusCode:  503,
			expectedMsg: "伺服器暫時無法使用，請稍後再試",
		},
		{
			name:        "HTTP 504 Gateway Timeout",
			statusCode:  504,
			expectedMsg: "伺服器暫時無法使用，請稍後再試",
		},
		{
			name:        "HTTP 599 (邊界值)",
			statusCode:  599,
			expectedMsg: "伺服器暫時無法使用，請稍後再試",
		},
		// 其他狀態碼（包含詳細錯誤資訊）
		{
			name:        "HTTP 400 Bad Request",
			statusCode:  400,
			expectedMsg: "Token 刷新失敗 (HTTP 400): test body",
		},
		{
			name:        "HTTP 404 Not Found",
			statusCode:  404,
			expectedMsg: "Token 刷新失敗 (HTTP 404): test body",
		},
		{
			name:        "HTTP 408 Request Timeout",
			statusCode:  408,
			expectedMsg: "Token 刷新失敗 (HTTP 408): test body",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			refreshErr := MapHTTPError(tc.statusCode, "test body")

			// 驗證 Code
			if refreshErr.Code != tc.statusCode {
				t.Errorf("Code mismatch: got %d, expected %d",
					refreshErr.Code, tc.statusCode)
			}

			// 驗證 Message
			if refreshErr.Message != tc.expectedMsg {
				t.Errorf("Message mismatch: got %q, expected %q",
					refreshErr.Message, tc.expectedMsg)
			}
		})
	}
}

// TestMapHTTPError_5xxRange 測試所有 5xx 狀態碼都正確映射
func TestMapHTTPError_5xxRange(t *testing.T) {
	expectedMsg := "伺服器暫時無法使用，請稍後再試"

	for statusCode := 500; statusCode < 600; statusCode++ {
		refreshErr := MapHTTPError(statusCode, "")

		if refreshErr.Code != statusCode {
			t.Errorf("Code mismatch for %d: got %d", statusCode, refreshErr.Code)
		}

		if refreshErr.Message != expectedMsg {
			t.Errorf("Message mismatch for %d: got %q, expected %q",
				statusCode, refreshErr.Message, expectedMsg)
		}
	}
}
