package tokenrefresh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"kiro-manager/awssso"
	"kiro-manager/kiroversion"
	"kiro-manager/settings"
)

// API 端點常數
const (
	SocialRefreshURL = "https://prod.us-east-1.auth.desktop.kiro.dev/refreshToken"
	IdCRefreshURL    = "https://oidc.us-east-1.amazonaws.com/token"
)

// getEffectiveKiroVersion 取得有效的 Kiro 版本號
// 如果啟用自動偵測，則從 Kiro 執行檔讀取版本；否則使用設定中的自定義值
func getEffectiveKiroVersion() string {
	if settings.IsAutoDetectEnabled() {
		// 嘗試自動偵測
		if version, err := kiroversion.GetKiroVersion(); err == nil && version != "" {
			return version
		}
		// 偵測失敗時回退到設定值
	}
	return settings.GetKiroVersion()
}

// TokenInfo 刷新後的 Token 資訊
type TokenInfo struct {
	AccessToken string    `json:"accessToken"` // 新的 AccessToken
	ExpiresAt   time.Time `json:"expiresAt"`   // 過期時間（計算後）
	ExpiresIn   int       `json:"expiresIn"`   // 有效期（秒）
	ProfileArn  string    `json:"profileArn"`  // Profile ARN（僅 Social）
	TokenType   string    `json:"tokenType"`   // Token 類型（僅 IdC）
}

// RefreshError 刷新錯誤類型
type RefreshError struct {
	Code    int    // HTTP 狀態碼（0 表示非 HTTP 錯誤）
	Message string // 使用者友善的錯誤訊息
	Cause   error  // 底層錯誤（用於除錯）
}

// Error 實作 error 介面
func (e *RefreshError) Error() string {
	return e.Message
}

// Unwrap 支援 errors.Unwrap
func (e *RefreshError) Unwrap() error {
	return e.Cause
}

// SocialRefreshRequest Social 刷新請求
type SocialRefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// SocialRefreshResponse Social 刷新回應
type SocialRefreshResponse struct {
	AccessToken  string `json:"accessToken"`
	ExpiresIn    int    `json:"expiresIn"`
	RefreshToken string `json:"refreshToken"`
	ProfileArn   string `json:"profileArn"`
}


// IdCRefreshRequest IdC 刷新請求
// 注意：AWS IdC OIDC API 使用 camelCase 欄位名稱
type IdCRefreshRequest struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	GrantType    string `json:"grantType"` // 固定為 "refresh_token"
	RefreshToken string `json:"refreshToken"`
}

// IdCRefreshResponse IdC 刷新回應
// 注意：AWS IdC OIDC API 回應使用 camelCase 欄位名稱
type IdCRefreshResponse struct {
	AccessToken  string `json:"accessToken"`
	ExpiresIn    int    `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

// CalculateExpiresAt 計算過期時間
// 將 expiresIn 秒數加到當前時間
// 需求: 5.3
func CalculateExpiresAt(expiresIn int) time.Time {
	return time.Now().Add(time.Duration(expiresIn) * time.Second)
}

// CalculateExpiresAtString 計算過期時間並格式化為 Kiro 期望的 UTC 毫秒格式
// 將 expiresIn 秒數加到當前時間，並將結果格式化為 "2006-01-02T15:04:05.000Z" 格式
// 需求: 5.3
func CalculateExpiresAtString(expiresIn int) string {
	return CalculateExpiresAt(expiresIn).UTC().Format("2006-01-02T15:04:05.000Z")
}

// MapHTTPError 將 HTTP 狀態碼映射為使用者友善的錯誤訊息
// 需求: 4.1, 4.2, 4.3
// - HTTP 401/403 映射為「Token 已失效，請重新登入 Kiro」
// - HTTP 429 映射為「請求過於頻繁，請稍後再試」
// - HTTP 5xx 映射為「伺服器暫時無法使用，請稍後再試」
func MapHTTPError(statusCode int, body string) *RefreshError {
	var message string
	switch {
	case statusCode == 401 || statusCode == 403:
		message = "Token 已失效，請重新登入 Kiro"
	case statusCode == 429:
		message = "請求過於頻繁，請稍後再試"
	case statusCode >= 500 && statusCode < 600:
		message = "伺服器暫時無法使用，請稍後再試"
	default:
		// 包含 HTTP 狀態碼和回應內容以便除錯
		message = fmt.Sprintf("Token 刷新失敗 (HTTP %d): %s", statusCode, truncateString(body, 200))
	}
	return &RefreshError{
		Code:    statusCode,
		Message: message,
	}
}

// truncateString 截斷字串到指定長度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// RefreshSocialToken 使用 Social 認證方式刷新 Token
// 發送 POST 請求到 Social 刷新端點，解析回應並返回新的 Token 資訊
// machineId 參數應為對應環境快照的 Machine ID 的 SHA256 雜湊值
func RefreshSocialToken(refreshToken string, machineId string) (*TokenInfo, error) {
	// 驗證參數
	if machineId == "" {
		return nil, &RefreshError{
			Code:    0,
			Message: "machineId 不可為空",
		}
	}

	// 建立請求 body
	reqBody := SocialRefreshRequest{
		RefreshToken: refreshToken,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "無法序列化請求",
			Cause:   err,
		}
	}

	// 建立 HTTP 請求
	req, err := http.NewRequest("POST", SocialRefreshURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "無法建立請求",
			Cause:   err,
		}
	}

	// 設定必要的 Headers（與 Kiro IDE 一致）
	req.Header.Set("User-Agent", "KiroIDE-"+getEffectiveKiroVersion()+"-"+machineId)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Encoding", "br, gzip, deflate")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "*")
	req.Header.Set("Sec-Fetch-Mode", "cors")

	// 發送請求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "網路連線失敗: " + err.Error(),
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	// 讀取回應 body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "無法讀取回應",
			Cause:   err,
		}
	}

	// 處理 HTTP 錯誤（需求 4.1, 4.2, 4.3）
	if resp.StatusCode != http.StatusOK {
		return nil, MapHTTPError(resp.StatusCode, string(body))
	}

	// 解析 JSON 回應
	var socialResp SocialRefreshResponse
	if err := json.Unmarshal(body, &socialResp); err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "無法解析伺服器回應",
			Cause:   err,
		}
	}

	// 建立 TokenInfo 並計算 ExpiresAt
	return &TokenInfo{
		AccessToken: socialResp.AccessToken,
		ExpiresIn:   socialResp.ExpiresIn,
		ExpiresAt:   CalculateExpiresAt(socialResp.ExpiresIn),
		ProfileArn:  socialResp.ProfileArn,
	}, nil
}

// ParseSocialResponse 解析 Social 刷新回應 JSON
// 此函數用於測試，將 JSON bytes 解析為 TokenInfo
func ParseSocialResponse(jsonData []byte) (*TokenInfo, error) {
	var socialResp SocialRefreshResponse
	if err := json.Unmarshal(jsonData, &socialResp); err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "無法解析伺服器回應",
			Cause:   err,
		}
	}

	return &TokenInfo{
		AccessToken: socialResp.AccessToken,
		ExpiresIn:   socialResp.ExpiresIn,
		ExpiresAt:   CalculateExpiresAt(socialResp.ExpiresIn),
		ProfileArn:  socialResp.ProfileArn,
	}, nil
}


// RefreshIdCToken 使用 IdC 認證方式刷新 Token
// 發送 POST 請求到 IdC 刷新端點，包含必要的 Headers
// 需求: 2.2, 2.3, 5.2, 5.3
func RefreshIdCToken(refreshToken, clientID, clientSecret string) (*TokenInfo, error) {
	// 建立請求 body
	reqBody := IdCRefreshRequest{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "無法序列化請求",
			Cause:   err,
		}
	}

	// 建立 HTTP 請求
	req, err := http.NewRequest("POST", IdCRefreshURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "無法建立請求",
			Cause:   err,
		}
	}

	// 設定必要的 Headers（需求 2.3）
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", "oidc.us-east-1.amazonaws.com")
	req.Header.Set("x-amz-user-agent", "aws-sdk-js/3.738.0 KiroIDE")
	req.Header.Set("User-Agent", "aws-sdk-js/3.738.0 ua/2.1 os/win32#10.0.26100 lang/js md/nodejs#22.21.1 api/sso-oidc#3.738.0 m/E KiroIDE")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "close")
	req.Header.Set("amz-sdk-invocation-id", uuid.New().String())
	req.Header.Set("amz-sdk-request", "attempt=1; max=4")

	// 發送請求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "網路連線失敗: " + err.Error(),
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	// 讀取回應 body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "無法讀取回應",
			Cause:   err,
		}
	}

	// 處理 HTTP 錯誤（需求 4.1, 4.2, 4.3）
	if resp.StatusCode != http.StatusOK {
		return nil, MapHTTPError(resp.StatusCode, string(body))
	}

	// 解析 JSON 回應
	var idcResp IdCRefreshResponse
	if err := json.Unmarshal(body, &idcResp); err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "無法解析伺服器回應",
			Cause:   err,
		}
	}

	// 建立 TokenInfo 並計算 ExpiresAt（需求 5.2, 5.3）
	return &TokenInfo{
		AccessToken: idcResp.AccessToken,
		ExpiresIn:   idcResp.ExpiresIn,
		ExpiresAt:   CalculateExpiresAt(idcResp.ExpiresIn),
		TokenType:   idcResp.TokenType,
	}, nil
}

// ParseIdCResponse 解析 IdC 刷新回應 JSON
// 此函數用於測試，將 JSON bytes 解析為 TokenInfo
func ParseIdCResponse(jsonData []byte) (*TokenInfo, error) {
	var idcResp IdCRefreshResponse
	if err := json.Unmarshal(jsonData, &idcResp); err != nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "無法解析伺服器回應",
			Cause:   err,
		}
	}

	return &TokenInfo{
		AccessToken: idcResp.AccessToken,
		ExpiresIn:   idcResp.ExpiresIn,
		ExpiresAt:   CalculateExpiresAt(idcResp.ExpiresIn),
		TokenType:   idcResp.TokenType,
	}, nil
}


// RefreshAccessToken 刷新 AccessToken
// 根據 token 中的 AuthMethod 判斷使用 Social 或 IdC 刷新方式
// machineId 參數應為對應環境快照的 Machine ID 的 SHA256 雜湊值
// 需求: 2.4
func RefreshAccessToken(token *awssso.KiroAuthToken, machineId string) (*TokenInfo, error) {
	return RefreshAccessTokenWithCredentials(token, machineId, "", "")
}

// RefreshAccessTokenFromBackup 從備份目錄刷新 AccessToken
// 與 RefreshAccessToken 類似，但 IdC 認證時會使用提供的 clientId 和 clientSecret
// 而不是從系統的 SSO cache 讀取
func RefreshAccessTokenFromBackup(token *awssso.KiroAuthToken, machineId string, clientID, clientSecret string) (*TokenInfo, error) {
	return RefreshAccessTokenWithCredentials(token, machineId, clientID, clientSecret)
}

// RefreshAccessTokenWithCredentials 刷新 AccessToken（內部實作）
// 如果提供了 clientID 和 clientSecret，IdC 認證時會直接使用
// 否則會從 SSO cache 讀取
func RefreshAccessTokenWithCredentials(token *awssso.KiroAuthToken, machineId string, clientID, clientSecret string) (*TokenInfo, error) {
	if token == nil {
		return nil, &RefreshError{
			Code:    0,
			Message: "Token 不可為空",
		}
	}

	if machineId == "" {
		return nil, &RefreshError{
			Code:    0,
			Message: "machineId 不可為空",
		}
	}

	// 偵測認證類型
	authType := DetectAuthType(token)

	switch authType {
	case "social":
		// Social 認證路由到 RefreshSocialToken
		if token.RefreshToken == "" {
			return nil, &RefreshError{
				Code:    0,
				Message: "RefreshToken 不可為空",
			}
		}
		return RefreshSocialToken(token.RefreshToken, machineId)

	case "idc":
		// IdC 認證路由到 RefreshIdCToken
		if token.RefreshToken == "" {
			return nil, &RefreshError{
				Code:    0,
				Message: "RefreshToken 不可為空",
			}
		}
		// 如果沒有提供 clientID 和 clientSecret，從 SSO cache 讀取
		if clientID == "" || clientSecret == "" {
			var err error
			clientID, clientSecret, err = getIdCCredentials(token)
			if err != nil {
				return nil, err
			}
		}
		return RefreshIdCToken(token.RefreshToken, clientID, clientSecret)

	default:
		return nil, &RefreshError{
			Code:    0,
			Message: "不支援的認證類型: " + authType,
		}
	}
}

// DetectAuthType 偵測 token 的認證類型
// 根據 AuthMethod 欄位或其他特徵判斷是 Social 還是 IdC
func DetectAuthType(token *awssso.KiroAuthToken) string {
	if token == nil {
		return "unknown"
	}

	// 優先使用 AuthMethod 欄位
	if token.AuthMethod != "" {
		authMethod := strings.ToLower(token.AuthMethod)
		if authMethod == "social" {
			return "social"
		}
		if authMethod == "idc" || authMethod == "identitycenter" {
			return "idc"
		}
	}

	// 如果沒有 AuthMethod，根據其他特徵判斷
	// IdC 認證通常有 StartURL 和 Region 欄位
	if token.StartURL != "" && token.Region != "" {
		return "idc"
	}

	// Social 認證通常有 Provider 欄位（如 Github, Google）
	if token.Provider != "" {
		return "social"
	}

	// 如果有 ProfileArn 但沒有 StartURL，可能是 Social
	if token.ProfileArn != "" && token.StartURL == "" {
		return "social"
	}

	return "unknown"
}

// getIdCCredentials 從 SSO cache 中取得 IdC 的 clientId 和 clientSecret
func getIdCCredentials(token *awssso.KiroAuthToken) (clientID, clientSecret string, err error) {
	// 優先使用 clientIdHash 來查找對應的文件（BuilderId 使用此方式）
	if token.ClientIdHash != "" {
		clientIdHashFile := token.ClientIdHash + ".json"
		cacheFile, err := awssso.ReadCacheFile(clientIdHashFile)
		if err == nil && cacheFile.ClientID != "" && cacheFile.ClientSecret != "" {
			return cacheFile.ClientID, cacheFile.ClientSecret, nil
		}
	}

	// 回退：嘗試從 SSO cache 中的其他檔案讀取 clientId 和 clientSecret
	// 這些資訊通常存在於以 startUrl 的 hash 命名的檔案中
	files, err := awssso.ListCacheFiles()
	if err != nil {
		return "", "", &RefreshError{
			Code:    0,
			Message: "無法讀取 SSO 快取目錄",
			Cause:   err,
		}
	}

	// 遍歷所有快取檔案，尋找包含 clientId 和 clientSecret 的檔案
	for _, file := range files {
		if file == awssso.KiroAuthTokenFile {
			continue // 跳過 kiro-auth-token.json
		}

		cacheFile, err := awssso.ReadCacheFile(file)
		if err != nil {
			continue
		}

		// 檢查是否有 clientId 和 clientSecret
		if cacheFile.ClientID != "" && cacheFile.ClientSecret != "" {
			// 如果有 startUrl，確認與 token 的 startUrl 匹配
			if token.StartURL != "" && cacheFile.StartURL != "" {
				if cacheFile.StartURL == token.StartURL {
					return cacheFile.ClientID, cacheFile.ClientSecret, nil
				}
			} else {
				// 沒有 startUrl 可比對，直接使用找到的第一個
				return cacheFile.ClientID, cacheFile.ClientSecret, nil
			}
		}
	}

	return "", "", &RefreshError{
		Code:    0,
		Message: "找不到 IdC 認證所需的 clientId 和 clientSecret",
	}
}
