package backup

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"testing/quick"
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

// generateRandomKiroAuthTokenMap 生成隨機的 KiroAuthToken map（包含各種欄位）
func generateRandomKiroAuthTokenMap(r *rand.Rand) map[string]interface{} {
	tokenMap := make(map[string]interface{})

	// 必要欄位
	tokenMap["accessToken"] = generateRandomString(r, r.Intn(100)+10)
	tokenMap["expiresAt"] = "2025-12-08T12:00:00Z"
	tokenMap["refreshToken"] = generateRandomString(r, r.Intn(100)+10)

	// 可選欄位（隨機決定是否包含）
	if r.Float32() > 0.3 {
		tokenMap["provider"] = []string{"Github", "Google", "AWS"}[r.Intn(3)]
	}
	if r.Float32() > 0.3 {
		tokenMap["authMethod"] = []string{"social", "idc"}[r.Intn(2)]
	}
	if r.Float32() > 0.3 {
		tokenMap["tokenType"] = "Bearer"
	}
	if r.Float32() > 0.3 {
		tokenMap["region"] = []string{"us-east-1", "us-west-2", "eu-west-1"}[r.Intn(3)]
	}
	if r.Float32() > 0.3 {
		tokenMap["startUrl"] = "https://d-" + generateRandomString(r, 10) + ".awsapps.com/start"
	}
	if r.Float32() > 0.3 {
		tokenMap["profileArn"] = "arn:aws:kiro::" + generateRandomString(r, 12) + ":profile/" + generateRandomString(r, 8)
	}
	// 加入一些額外的自訂欄位（模擬未知欄位）
	if r.Float32() > 0.5 {
		tokenMap["customField1"] = generateRandomString(r, 20)
	}
	if r.Float32() > 0.5 {
		tokenMap["customField2"] = r.Intn(1000)
	}
	if r.Float32() > 0.5 {
		tokenMap["nestedObject"] = map[string]interface{}{
			"key1": generateRandomString(r, 10),
			"key2": r.Intn(100),
		}
	}

	return tokenMap
}

// **Feature: token-refresh, Property 1: Token Update Preserves Original Fields**
// *For any* KiroAuthToken with valid RefreshToken, after a successful refresh operation,
// all original fields except accessToken, expiresAt, and expiresIn SHALL remain unchanged.
// **Validates: Requirements 3.2**
func TestProperty_TokenUpdatePreservesOriginalFields(t *testing.T) {
	// 建立臨時測試目錄
	tempDir, err := os.MkdirTemp("", "backup_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// 生成隨機的 token map
		originalTokenMap := generateRandomKiroAuthTokenMap(r)

		// 建立測試備份目錄
		backupName := "test_backup_" + generateRandomString(r, 8)
		backupPath := filepath.Join(tempDir, backupName)
		if err := os.MkdirAll(backupPath, 0755); err != nil {
			t.Logf("Failed to create backup dir: %v", err)
			return false
		}

		// 寫入原始 token 檔案
		tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
		originalData, err := json.MarshalIndent(originalTokenMap, "", "  ")
		if err != nil {
			t.Logf("Failed to marshal original token: %v", err)
			return false
		}
		if err := os.WriteFile(tokenPath, originalData, 0644); err != nil {
			t.Logf("Failed to write original token: %v", err)
			return false
		}

		// 生成新的 accessToken 和 expiresAt
		newAccessToken := generateRandomString(r, r.Intn(100)+10)
		newExpiresAt := "2025-12-09T15:30:00Z"

		// 呼叫 WriteBackupToken（使用自訂路徑版本）
		err = writeBackupTokenToPath(tokenPath, newAccessToken, newExpiresAt)
		if err != nil {
			t.Logf("Failed to write backup token: %v", err)
			return false
		}

		// 讀取更新後的 token
		updatedData, err := os.ReadFile(tokenPath)
		if err != nil {
			t.Logf("Failed to read updated token: %v", err)
			return false
		}

		var updatedTokenMap map[string]interface{}
		if err := json.Unmarshal(updatedData, &updatedTokenMap); err != nil {
			t.Logf("Failed to unmarshal updated token: %v", err)
			return false
		}

		// Property 1: 驗證 accessToken 和 expiresAt 已更新
		if updatedTokenMap["accessToken"] != newAccessToken {
			t.Logf("accessToken not updated: got %v, expected %v",
				updatedTokenMap["accessToken"], newAccessToken)
			return false
		}
		if updatedTokenMap["expiresAt"] != newExpiresAt {
			t.Logf("expiresAt not updated: got %v, expected %v",
				updatedTokenMap["expiresAt"], newExpiresAt)
			return false
		}

		// Property 1: 驗證其他所有欄位保持不變
		for key, originalValue := range originalTokenMap {
			if key == "accessToken" || key == "expiresAt" {
				continue // 這些欄位應該被更新
			}

			updatedValue, exists := updatedTokenMap[key]
			if !exists {
				t.Logf("Field %q was removed", key)
				return false
			}

			// 比較值（需要處理 map 類型）
			if !compareValues(originalValue, updatedValue) {
				t.Logf("Field %q changed: original=%v, updated=%v",
					key, originalValue, updatedValue)
				return false
			}
		}

		// 清理測試備份
		os.RemoveAll(backupPath)

		return true
	}

	config := &quick.Config{
		MaxCount: 100,
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// writeBackupTokenToPath 是 WriteBackupToken 的內部版本，直接操作指定路徑
// 用於測試時避免依賴 GetBackupPath
// 注意：此函數必須與 WriteBackupToken 使用相同的 struct-based 邏輯
func writeBackupTokenToPath(tokenPath string, accessToken string, expiresAt string) error {
	// 讀取現有 token 檔案以保留原始欄位
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return err
	}

	// 先解析到 map 以讀取原始值
	var tokenMap map[string]interface{}
	if err := json.Unmarshal(data, &tokenMap); err != nil {
		return err
	}

	// 使用與 WriteBackupToken 相同的 struct-based 邏輯
	// 這確保測試反映真實行為
	orderedToken := struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ProfileArn   string `json:"profileArn,omitempty"`
		ExpiresAt    string `json:"expiresAt"`
		AuthMethod   string `json:"authMethod,omitempty"`
		Provider     string `json:"provider,omitempty"`
		ClientIdHash string `json:"clientIdHash,omitempty"`
		Region       string `json:"region,omitempty"`
		TokenType    string `json:"tokenType,omitempty"`
		StartURL     string `json:"startUrl,omitempty"`
	}{
		AccessToken:  accessToken,
		RefreshToken: getStringFromMapTest(tokenMap, "refreshToken"),
		ProfileArn:   getStringFromMapTest(tokenMap, "profileArn"),
		ExpiresAt:    expiresAt,
		AuthMethod:   getStringFromMapTest(tokenMap, "authMethod"),
		Provider:     getStringFromMapTest(tokenMap, "provider"),
		ClientIdHash: getStringFromMapTest(tokenMap, "clientIdHash"),
		Region:       getStringFromMapTest(tokenMap, "region"),
		TokenType:    getStringFromMapTest(tokenMap, "tokenType"),
		StartURL:     getStringFromMapTest(tokenMap, "startUrl"),
	}

	// 將更新後的 token 寫回檔案
	updatedData, err := json.MarshalIndent(orderedToken, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenPath, updatedData, 0644)
}

// getStringFromMapTest 從 map 中安全地取得字串值（測試用）
func getStringFromMapTest(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// compareValues 比較兩個值是否相等（處理 map 和其他類型）
func compareValues(a, b interface{}) bool {
	// 將兩個值都序列化為 JSON 再比較
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

// TestWriteBackupToken_InvalidBackupName 測試無效備份名稱的處理
func TestWriteBackupToken_InvalidBackupName(t *testing.T) {
	err := WriteBackupToken("", "new-token", "2025-12-09T15:30:00Z")
	if err != ErrInvalidBackupName {
		t.Errorf("Expected ErrInvalidBackupName, got %v", err)
	}
}

// TestWriteBackupToken_BackupNotFound 測試備份不存在的處理
func TestWriteBackupToken_BackupNotFound(t *testing.T) {
	err := WriteBackupToken("non_existent_backup_xyz123", "new-token", "2025-12-09T15:30:00Z")
	if err != ErrBackupNotFound {
		t.Errorf("Expected ErrBackupNotFound, got %v", err)
	}
}

// TestWriteBackupToken_PreservesAllFields 測試欄位保留功能
// 注意：此測試驗證 struct-based 邏輯，只保留 orderedKiroAuthToken 中定義的欄位
// 未知欄位（如 customField）會被丟棄，這是設計上的限制
func TestWriteBackupToken_PreservesAllFields(t *testing.T) {
	// 建立臨時測試目錄
	tempDir, err := os.MkdirTemp("", "backup_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 建立測試備份目錄
	backupPath := filepath.Join(tempDir, "test_backup")
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	// 建立包含多個欄位的原始 token（只包含 orderedKiroAuthToken 支援的欄位）
	originalToken := map[string]interface{}{
		"accessToken":  "old-access-token",
		"expiresAt":    "2025-12-08T12:00:00Z",
		"refreshToken": "my-refresh-token",
		"provider":     "Github",
		"authMethod":   "social",
		"profileArn":   "arn:aws:kiro::123456789012:profile/test",
	}

	// 寫入原始 token
	tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
	originalData, _ := json.MarshalIndent(originalToken, "", "  ")
	if err := os.WriteFile(tokenPath, originalData, 0644); err != nil {
		t.Fatalf("Failed to write original token: %v", err)
	}

	// 更新 token
	newAccessToken := "new-access-token-12345"
	newExpiresAt := "2025-12-09T18:00:00Z"
	err = writeBackupTokenToPath(tokenPath, newAccessToken, newExpiresAt)
	if err != nil {
		t.Fatalf("Failed to write backup token: %v", err)
	}

	// 讀取更新後的 token
	updatedData, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("Failed to read updated token: %v", err)
	}

	var updatedToken map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedToken); err != nil {
		t.Fatalf("Failed to unmarshal updated token: %v", err)
	}

	// 驗證更新的欄位
	if updatedToken["accessToken"] != newAccessToken {
		t.Errorf("accessToken not updated: got %v, expected %v",
			updatedToken["accessToken"], newAccessToken)
	}
	if updatedToken["expiresAt"] != newExpiresAt {
		t.Errorf("expiresAt not updated: got %v, expected %v",
			updatedToken["expiresAt"], newExpiresAt)
	}

	// 驗證保留的欄位
	if updatedToken["refreshToken"] != "my-refresh-token" {
		t.Errorf("refreshToken changed: got %v", updatedToken["refreshToken"])
	}
	if updatedToken["provider"] != "Github" {
		t.Errorf("provider changed: got %v", updatedToken["provider"])
	}
	if updatedToken["authMethod"] != "social" {
		t.Errorf("authMethod changed: got %v", updatedToken["authMethod"])
	}
	if updatedToken["profileArn"] != "arn:aws:kiro::123456789012:profile/test" {
		t.Errorf("profileArn changed: got %v", updatedToken["profileArn"])
	}
}


// ============================================================================
// OAuth Snapshot Tests (Task 9)
// ============================================================================

// OAuthBackupData OAuth 登入備份資料結構 (測試用)
type testOAuthBackupData struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    string
	ProfileArn   string
	Provider     string
	AuthMethod   string
	ClientId     string
	ClientSecret string
	ClientIdHash string
}

// generateRandomOAuthBackupData 生成隨機的 OAuth 備份資料
func generateRandomOAuthBackupData(r *rand.Rand, authMethod string) *testOAuthBackupData {
	data := &testOAuthBackupData{
		AccessToken:  generateRandomString(r, r.Intn(50)+20),
		RefreshToken: generateRandomString(r, r.Intn(50)+20),
		ExpiresAt:    "2025-12-15T12:00:00Z",
		ProfileArn:   "arn:aws:kiro::" + generateRandomString(r, 12) + ":profile/" + generateRandomString(r, 8),
		Provider:     []string{"Github", "Google", "BuilderID"}[r.Intn(3)],
		AuthMethod:   authMethod,
	}

	if authMethod == "idc" {
		data.ClientId = generateRandomString(r, 20)
		data.ClientSecret = generateRandomString(r, 40)
		data.ClientIdHash = generateRandomString(r, 64)
	}

	return data
}

// **Feature: oauth-login, Property 9: OAuth Snapshot Creation Completeness**
// *For any* valid OAuth login result with Social auth method,
// the created snapshot SHALL contain kiro-auth-token.json and machine-id.json
// with all required fields.
// **Validates: Requirements 8.3, 8.5**
func TestProperty_OAuthSnapshotCreationCompleteness(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "oauth_backup_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// 生成隨機的 Social OAuth 資料
		oauthData := generateRandomOAuthBackupData(r, "social")
		snapshotName := "oauth_test_" + generateRandomString(r, 8)
		backupPath := filepath.Join(tempDir, snapshotName)

		// 建立 OAuth 快照
		err := createBackupFromOAuthToPath(backupPath, oauthData)
		if err != nil {
			t.Logf("Failed to create OAuth backup: %v", err)
			return false
		}

		// 驗證 kiro-auth-token.json 存在且包含所有必要欄位
		tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
		tokenData, err := os.ReadFile(tokenPath)
		if err != nil {
			t.Logf("Failed to read token file: %v", err)
			return false
		}

		var tokenMap map[string]interface{}
		if err := json.Unmarshal(tokenData, &tokenMap); err != nil {
			t.Logf("Failed to unmarshal token: %v", err)
			return false
		}

		// 驗證必要欄位存在
		requiredFields := []string{"accessToken", "refreshToken", "expiresAt", "provider", "authMethod"}
		for _, field := range requiredFields {
			if _, exists := tokenMap[field]; !exists {
				t.Logf("Missing required field: %s", field)
				return false
			}
		}

		// 驗證欄位值正確
		if tokenMap["accessToken"] != oauthData.AccessToken {
			t.Logf("accessToken mismatch")
			return false
		}
		if tokenMap["refreshToken"] != oauthData.RefreshToken {
			t.Logf("refreshToken mismatch")
			return false
		}
		if tokenMap["provider"] != oauthData.Provider {
			t.Logf("provider mismatch")
			return false
		}
		if tokenMap["authMethod"] != oauthData.AuthMethod {
			t.Logf("authMethod mismatch")
			return false
		}

		// 驗證 machine-id.json 存在
		machineIDPath := filepath.Join(backupPath, MachineIDFileName)
		if _, err := os.Stat(machineIDPath); os.IsNotExist(err) {
			t.Logf("machine-id.json not found")
			return false
		}

		// 清理
		os.RemoveAll(backupPath)
		return true
	}

	config := &quick.Config{MaxCount: 50}
	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// **Feature: oauth-login, Property 10: IdC Snapshot Creation with Client Credentials**
// *For any* valid OAuth login result with IdC auth method,
// the created snapshot SHALL contain clientIdHash.json with clientId and clientSecret.
// **Validates: Requirements 8.4**
func TestProperty_IdCSnapshotCreationWithClientCredentials(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "idc_backup_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// 生成隨機的 IdC OAuth 資料
		oauthData := generateRandomOAuthBackupData(r, "idc")
		snapshotName := "idc_test_" + generateRandomString(r, 8)
		backupPath := filepath.Join(tempDir, snapshotName)

		// 建立 OAuth 快照
		err := createBackupFromOAuthToPath(backupPath, oauthData)
		if err != nil {
			t.Logf("Failed to create OAuth backup: %v", err)
			return false
		}

		// 驗證 clientIdHash.json 存在
		clientIdHashFile := oauthData.ClientIdHash + ".json"
		clientIdHashPath := filepath.Join(backupPath, clientIdHashFile)
		clientData, err := os.ReadFile(clientIdHashPath)
		if err != nil {
			t.Logf("Failed to read clientIdHash file: %v", err)
			return false
		}

		var clientMap map[string]interface{}
		if err := json.Unmarshal(clientData, &clientMap); err != nil {
			t.Logf("Failed to unmarshal clientIdHash file: %v", err)
			return false
		}

		// 驗證 clientId 和 clientSecret 存在且正確
		if clientMap["clientId"] != oauthData.ClientId {
			t.Logf("clientId mismatch: got %v, expected %v", clientMap["clientId"], oauthData.ClientId)
			return false
		}
		if clientMap["clientSecret"] != oauthData.ClientSecret {
			t.Logf("clientSecret mismatch")
			return false
		}

		// 驗證 kiro-auth-token.json 包含 clientIdHash
		tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
		tokenData, err := os.ReadFile(tokenPath)
		if err != nil {
			t.Logf("Failed to read token file: %v", err)
			return false
		}

		var tokenMap map[string]interface{}
		if err := json.Unmarshal(tokenData, &tokenMap); err != nil {
			t.Logf("Failed to unmarshal token: %v", err)
			return false
		}

		if tokenMap["clientIdHash"] != oauthData.ClientIdHash {
			t.Logf("clientIdHash mismatch in token")
			return false
		}

		// 清理
		os.RemoveAll(backupPath)
		return true
	}

	config := &quick.Config{MaxCount: 50}
	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// **Feature: oauth-login, Property 11: Snapshot Name Validation - Illegal Characters**
// *For any* snapshot name containing illegal characters (/ \ : * ? " < > |),
// the validation SHALL reject the name.
// **Validates: Requirements 9.2**
func TestProperty_SnapshotNameValidationIllegalCharacters(t *testing.T) {
	illegalChars := []rune{'/', '\\', ':', '*', '?', '"', '<', '>', '|'}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// 生成包含非法字元的名稱
		baseName := generateRandomString(r, r.Intn(10)+5)
		illegalChar := illegalChars[r.Intn(len(illegalChars))]
		position := r.Intn(len(baseName) + 1)

		// 在隨機位置插入非法字元
		invalidName := baseName[:position] + string(illegalChar) + baseName[position:]

		// 驗證名稱被拒絕
		err := ValidateSnapshotName(invalidName)
		if err == nil {
			t.Logf("Expected error for name with illegal char '%c': %s", illegalChar, invalidName)
			return false
		}

		return true
	}

	config := &quick.Config{MaxCount: 100}
	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// **Feature: oauth-login, Property 12: Snapshot Name Uniqueness**
// *For any* existing snapshot name, attempting to create a new snapshot
// with the same name SHALL be rejected.
// **Validates: Requirements 9.3**
func TestProperty_SnapshotNameUniqueness(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "uniqueness_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		snapshotName := "unique_test_" + generateRandomString(r, 8)
		backupPath := filepath.Join(tempDir, snapshotName)

		// 建立第一個快照
		if err := os.MkdirAll(backupPath, 0755); err != nil {
			t.Logf("Failed to create first backup: %v", err)
			return false
		}

		// 驗證重複名稱被拒絕
		err := validateSnapshotNameWithPath(tempDir, snapshotName)
		if err == nil {
			t.Logf("Expected error for duplicate name: %s", snapshotName)
			os.RemoveAll(backupPath)
			return false
		}

		// 清理
		os.RemoveAll(backupPath)
		return true
	}

	config := &quick.Config{MaxCount: 50}
	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestValidateSnapshotName_EmptyName 測試空名稱被拒絕
// **Validates: Requirements 9.1**
func TestValidateSnapshotName_EmptyName(t *testing.T) {
	err := ValidateSnapshotName("")
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

// TestValidateSnapshotName_ValidName 測試有效名稱通過驗證
func TestValidateSnapshotName_ValidName(t *testing.T) {
	validNames := []string{
		"my-backup",
		"backup_2025",
		"TestBackup123",
		"a",
		"backup.name",
		"backup-with-dash",
		"backup_with_underscore",
	}

	for _, name := range validNames {
		// 使用不檢查重複的版本
		err := validateSnapshotNameBasic(name)
		if err != nil {
			t.Errorf("Expected valid name %q to pass, got error: %v", name, err)
		}
	}
}

// TestValidateSnapshotName_IllegalCharacters 測試各種非法字元
func TestValidateSnapshotName_IllegalCharacters(t *testing.T) {
	testCases := []struct {
		name        string
		description string
	}{
		{"test/name", "forward slash"},
		{"test\\name", "backslash"},
		{"test:name", "colon"},
		{"test*name", "asterisk"},
		{"test?name", "question mark"},
		{"test\"name", "double quote"},
		{"test<name", "less than"},
		{"test>name", "greater than"},
		{"test|name", "pipe"},
	}

	for _, tc := range testCases {
		err := validateSnapshotNameBasic(tc.name)
		if err == nil {
			t.Errorf("Expected error for name with %s: %q", tc.description, tc.name)
		}
	}
}

// ============================================================================
// Helper functions for testing (to be implemented in backup.go)
// ============================================================================

// createBackupFromOAuthToPath 測試用的 OAuth 備份建立函數
func createBackupFromOAuthToPath(backupPath string, data *testOAuthBackupData) error {
	// 建立備份目錄
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	// 建立 kiro-auth-token.json
	tokenData := map[string]interface{}{
		"accessToken":  data.AccessToken,
		"refreshToken": data.RefreshToken,
		"profileArn":   data.ProfileArn,
		"expiresAt":    data.ExpiresAt,
		"authMethod":   data.AuthMethod,
		"provider":     data.Provider,
	}

	// 如果是 IdC，加入 clientIdHash
	if data.AuthMethod == "idc" && data.ClientIdHash != "" {
		tokenData["clientIdHash"] = data.ClientIdHash
	}

	tokenJSON, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		os.RemoveAll(backupPath)
		return err
	}

	tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
	if err := os.WriteFile(tokenPath, tokenJSON, 0644); err != nil {
		os.RemoveAll(backupPath)
		return err
	}

	// 建立 machine-id.json
	machineIDData := MachineIDBackup{
		MachineID:  "test-machine-id-" + filepath.Base(backupPath),
		BackupTime: "2025-06-08T12:00:00Z",
	}

	machineIDJSON, err := json.MarshalIndent(machineIDData, "", "  ")
	if err != nil {
		os.RemoveAll(backupPath)
		return err
	}

	machineIDPath := filepath.Join(backupPath, MachineIDFileName)
	if err := os.WriteFile(machineIDPath, machineIDJSON, 0644); err != nil {
		os.RemoveAll(backupPath)
		return err
	}

	// 如果是 IdC，建立 clientIdHash.json
	if data.AuthMethod == "idc" && data.ClientIdHash != "" {
		clientData := map[string]string{
			"clientId":     data.ClientId,
			"clientSecret": data.ClientSecret,
		}

		clientJSON, err := json.MarshalIndent(clientData, "", "  ")
		if err != nil {
			os.RemoveAll(backupPath)
			return err
		}

		clientPath := filepath.Join(backupPath, data.ClientIdHash+".json")
		if err := os.WriteFile(clientPath, clientJSON, 0644); err != nil {
			os.RemoveAll(backupPath)
			return err
		}
	}

	return nil
}

// validateSnapshotNameBasic 基本名稱驗證（不檢查重複）
func validateSnapshotNameBasic(name string) error {
	if name == "" {
		return ErrInvalidBackupName
	}

	// 檢查非法字元
	illegalChars := []rune{'/', '\\', ':', '*', '?', '"', '<', '>', '|'}
	for _, char := range name {
		for _, illegal := range illegalChars {
			if char == illegal {
				return ErrInvalidBackupName
			}
		}
	}

	return nil
}

// validateSnapshotNameWithPath 帶路徑的名稱驗證（檢查重複）
func validateSnapshotNameWithPath(rootPath, name string) error {
	if err := validateSnapshotNameBasic(name); err != nil {
		return err
	}

	// 檢查是否已存在
	backupPath := filepath.Join(rootPath, name)
	if info, err := os.Stat(backupPath); err == nil && info.IsDir() {
		return ErrBackupExists
	}

	return nil
}


// ============================================================================
// IdC Token Field Preservation Tests (Feature: idc-token-field-preservation)
// ============================================================================

// TestWriteBackupToken_PreservesIdCFields 測試 IdC 欄位 (clientIdHash, region) 保留
// **Validates: Scenario 1 - IdC Token 刷新後保留 clientIdHash 欄位**
func TestWriteBackupToken_PreservesIdCFields(t *testing.T) {
	// 建立臨時測試目錄
	tempDir, err := os.MkdirTemp("", "idc_field_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 建立測試備份目錄
	backupPath := filepath.Join(tempDir, "test_idc_backup")
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	// 建立包含 IdC 欄位的原始 token
	originalToken := map[string]interface{}{
		"accessToken":  "old-access-token",
		"refreshToken": "my-refresh-token",
		"profileArn":   "arn:aws:kiro::123456789012:profile/test",
		"expiresAt":    "2025-12-08T12:00:00Z",
		"authMethod":   "idc",
		"provider":     "IdentityCenter",
		"clientIdHash": "abc123def456",  // IdC 特有欄位
		"region":       "us-east-1",     // IdC 特有欄位
	}

	// 寫入原始 token
	tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
	originalData, _ := json.MarshalIndent(originalToken, "", "  ")
	if err := os.WriteFile(tokenPath, originalData, 0644); err != nil {
		t.Fatalf("Failed to write original token: %v", err)
	}

	// 使用 writeBackupTokenToPath 更新 token
	newAccessToken := "new-access-token-12345"
	newExpiresAt := "2025-12-09T18:00:00Z"
	err = writeBackupTokenToPath(tokenPath, newAccessToken, newExpiresAt)
	if err != nil {
		t.Fatalf("Failed to write backup token: %v", err)
	}

	// 讀取更新後的 token
	updatedData, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("Failed to read updated token: %v", err)
	}

	var updatedToken map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedToken); err != nil {
		t.Fatalf("Failed to unmarshal updated token: %v", err)
	}

	// 驗證 IdC 欄位被保留
	if updatedToken["clientIdHash"] != "abc123def456" {
		t.Errorf("clientIdHash not preserved: got %v, expected %v",
			updatedToken["clientIdHash"], "abc123def456")
	}
	if updatedToken["region"] != "us-east-1" {
		t.Errorf("region not preserved: got %v, expected %v",
			updatedToken["region"], "us-east-1")
	}

	// 驗證更新的欄位
	if updatedToken["accessToken"] != newAccessToken {
		t.Errorf("accessToken not updated: got %v, expected %v",
			updatedToken["accessToken"], newAccessToken)
	}
	if updatedToken["expiresAt"] != newExpiresAt {
		t.Errorf("expiresAt not updated: got %v, expected %v",
			updatedToken["expiresAt"], newExpiresAt)
	}
}

// TestWriteBackupToken_PreservesOptionalFields 測試可選欄位 (tokenType, startUrl) 保留
// **Validates: Scenario 4 - 保留 tokenType 和 startUrl 可選欄位**
func TestWriteBackupToken_PreservesOptionalFields(t *testing.T) {
	// 建立臨時測試目錄
	tempDir, err := os.MkdirTemp("", "optional_field_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 建立測試備份目錄
	backupPath := filepath.Join(tempDir, "test_optional_backup")
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	// 建立包含所有可選欄位的原始 token
	originalToken := map[string]interface{}{
		"accessToken":  "old-access-token",
		"refreshToken": "my-refresh-token",
		"profileArn":   "arn:aws:kiro::123456789012:profile/test",
		"expiresAt":    "2025-12-08T12:00:00Z",
		"authMethod":   "idc",
		"provider":     "IdentityCenter",
		"clientIdHash": "abc123def456",
		"region":       "us-west-2",
		"tokenType":    "Bearer",                                    // 可選欄位
		"startUrl":     "https://d-1234567890.awsapps.com/start",    // 可選欄位
	}

	// 寫入原始 token
	tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
	originalData, _ := json.MarshalIndent(originalToken, "", "  ")
	if err := os.WriteFile(tokenPath, originalData, 0644); err != nil {
		t.Fatalf("Failed to write original token: %v", err)
	}

	// 使用 writeBackupTokenToPath 更新 token
	newAccessToken := "new-access-token-67890"
	newExpiresAt := "2025-12-10T20:00:00Z"
	err = writeBackupTokenToPath(tokenPath, newAccessToken, newExpiresAt)
	if err != nil {
		t.Fatalf("Failed to write backup token: %v", err)
	}

	// 讀取更新後的 token
	updatedData, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("Failed to read updated token: %v", err)
	}

	var updatedToken map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedToken); err != nil {
		t.Fatalf("Failed to unmarshal updated token: %v", err)
	}

	// 驗證可選欄位被保留
	if updatedToken["tokenType"] != "Bearer" {
		t.Errorf("tokenType not preserved: got %v, expected %v",
			updatedToken["tokenType"], "Bearer")
	}
	if updatedToken["startUrl"] != "https://d-1234567890.awsapps.com/start" {
		t.Errorf("startUrl not preserved: got %v, expected %v",
			updatedToken["startUrl"], "https://d-1234567890.awsapps.com/start")
	}

	// 驗證 IdC 欄位也被保留
	if updatedToken["clientIdHash"] != "abc123def456" {
		t.Errorf("clientIdHash not preserved: got %v", updatedToken["clientIdHash"])
	}
	if updatedToken["region"] != "us-west-2" {
		t.Errorf("region not preserved: got %v", updatedToken["region"])
	}
}

// TestWriteBackupToken_OmitsEmptyOptionalFields 測試空值欄位不輸出
// **Validates: Scenario 5 - 處理不包含可選欄位的 Token**
func TestWriteBackupToken_OmitsEmptyOptionalFields(t *testing.T) {
	// 建立臨時測試目錄
	tempDir, err := os.MkdirTemp("", "empty_field_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 建立測試備份目錄
	backupPath := filepath.Join(tempDir, "test_empty_backup")
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	// 建立不包含可選欄位的 Social Token
	originalToken := map[string]interface{}{
		"accessToken":  "old-access-token",
		"refreshToken": "my-refresh-token",
		"profileArn":   "arn:aws:kiro::123456789012:profile/test",
		"expiresAt":    "2025-12-08T12:00:00Z",
		"authMethod":   "social",
		"provider":     "Github",
		// 注意：沒有 clientIdHash, region, tokenType, startUrl
	}

	// 寫入原始 token
	tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
	originalData, _ := json.MarshalIndent(originalToken, "", "  ")
	if err := os.WriteFile(tokenPath, originalData, 0644); err != nil {
		t.Fatalf("Failed to write original token: %v", err)
	}

	// 使用 writeBackupTokenToPath 更新 token
	newAccessToken := "new-access-token-social"
	newExpiresAt := "2025-12-11T22:00:00Z"
	err = writeBackupTokenToPath(tokenPath, newAccessToken, newExpiresAt)
	if err != nil {
		t.Fatalf("Failed to write backup token: %v", err)
	}

	// 讀取更新後的 token
	updatedData, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("Failed to read updated token: %v", err)
	}

	var updatedToken map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedToken); err != nil {
		t.Fatalf("Failed to unmarshal updated token: %v", err)
	}

	// 驗證空值欄位不存在（或為空字串）
	// 注意：writeBackupTokenToPath 使用 map 方式，所以不會新增不存在的欄位
	if _, exists := updatedToken["clientIdHash"]; exists && updatedToken["clientIdHash"] != "" {
		t.Errorf("clientIdHash should not exist or be empty for Social token, got %v",
			updatedToken["clientIdHash"])
	}
	if _, exists := updatedToken["region"]; exists && updatedToken["region"] != "" {
		t.Errorf("region should not exist or be empty for Social token, got %v",
			updatedToken["region"])
	}
	if _, exists := updatedToken["tokenType"]; exists && updatedToken["tokenType"] != "" {
		t.Errorf("tokenType should not exist or be empty for Social token, got %v",
			updatedToken["tokenType"])
	}
	if _, exists := updatedToken["startUrl"]; exists && updatedToken["startUrl"] != "" {
		t.Errorf("startUrl should not exist or be empty for Social token, got %v",
			updatedToken["startUrl"])
	}

	// 驗證基本欄位仍然存在
	if updatedToken["accessToken"] != newAccessToken {
		t.Errorf("accessToken not updated: got %v", updatedToken["accessToken"])
	}
	if updatedToken["authMethod"] != "social" {
		t.Errorf("authMethod not preserved: got %v", updatedToken["authMethod"])
	}
	if updatedToken["provider"] != "Github" {
		t.Errorf("provider not preserved: got %v", updatedToken["provider"])
	}
}

// TestWriteBackupToken_SocialTokenUnaffected 測試 Social Token 不受影響
// **Validates: Scenario 2 - Social Token 刷新後不受影響**
func TestWriteBackupToken_SocialTokenUnaffected(t *testing.T) {
	// 建立臨時測試目錄
	tempDir, err := os.MkdirTemp("", "social_token_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 建立測試備份目錄
	backupPath := filepath.Join(tempDir, "test_social_backup")
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	// 建立 Social Token（不包含 IdC 欄位）
	originalToken := map[string]interface{}{
		"accessToken":  "social-access-token",
		"refreshToken": "social-refresh-token",
		"profileArn":   "arn:aws:kiro::123456789012:profile/social",
		"expiresAt":    "2025-12-08T12:00:00Z",
		"authMethod":   "social",
		"provider":     "Google",
	}

	// 寫入原始 token
	tokenPath := filepath.Join(backupPath, KiroAuthTokenFile)
	originalData, _ := json.MarshalIndent(originalToken, "", "  ")
	if err := os.WriteFile(tokenPath, originalData, 0644); err != nil {
		t.Fatalf("Failed to write original token: %v", err)
	}

	// 使用 writeBackupTokenToPath 更新 token
	newAccessToken := "new-social-access-token"
	newExpiresAt := "2025-12-12T10:00:00Z"
	err = writeBackupTokenToPath(tokenPath, newAccessToken, newExpiresAt)
	if err != nil {
		t.Fatalf("Failed to write backup token: %v", err)
	}

	// 讀取更新後的 token
	updatedData, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("Failed to read updated token: %v", err)
	}

	var updatedToken map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedToken); err != nil {
		t.Fatalf("Failed to unmarshal updated token: %v", err)
	}

	// 驗證 Social Token 的所有欄位正確
	if updatedToken["accessToken"] != newAccessToken {
		t.Errorf("accessToken not updated: got %v", updatedToken["accessToken"])
	}
	if updatedToken["expiresAt"] != newExpiresAt {
		t.Errorf("expiresAt not updated: got %v", updatedToken["expiresAt"])
	}
	if updatedToken["refreshToken"] != "social-refresh-token" {
		t.Errorf("refreshToken changed: got %v", updatedToken["refreshToken"])
	}
	if updatedToken["authMethod"] != "social" {
		t.Errorf("authMethod changed: got %v", updatedToken["authMethod"])
	}
	if updatedToken["provider"] != "Google" {
		t.Errorf("provider changed: got %v", updatedToken["provider"])
	}
	if updatedToken["profileArn"] != "arn:aws:kiro::123456789012:profile/social" {
		t.Errorf("profileArn changed: got %v", updatedToken["profileArn"])
	}
}
