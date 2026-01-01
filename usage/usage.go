package usage

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"github.com/google/uuid"
	"kiro-manager/awssso"
	"kiro-manager/kiroversion"
	"kiro-manager/machineid"
	"kiro-manager/settings"
)

// HTTP 請求超時設定
const httpTimeout = 10 * time.Second

const (
	// API endpoint
	usageLimitsURL = "https://q.us-east-1.amazonaws.com/getUsageLimits"
	// Query parameters
	originParam       = "AI_EDITOR"
	resourceTypeParam = "AGENTIC_REQUEST"
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

// UsageLimitsResponse API 響應結構
type UsageLimitsResponse struct {
	SubscriptionInfo   SubscriptionInfo `json:"subscriptionInfo"`
	UsageBreakdownList []UsageBreakdown `json:"usageBreakdownList"`
}

// SubscriptionInfo 訂閱資訊結構
type SubscriptionInfo struct {
	SubscriptionTitle string `json:"subscriptionTitle"`
	Type              string `json:"type"`
}

// FreeTrialInfo 免費試用資訊
type FreeTrialInfo struct {
	UsageLimitWithPrecision   float64 `json:"usageLimitWithPrecision"`
	CurrentUsageWithPrecision float64 `json:"currentUsageWithPrecision"`
	FreeTrialStatus           string  `json:"freeTrialStatus"`
}

// Bonus 獎勵額度
type Bonus struct {
	BonusCode    string  `json:"bonusCode"`
	UsageLimit   float64 `json:"usageLimit"`
	CurrentUsage float64 `json:"currentUsage"`
	Status       string  `json:"status"`
}

// UsageBreakdown 用量明細結構
type UsageBreakdown struct {
	UsageLimitWithPrecision   float64        `json:"usageLimitWithPrecision"`
	CurrentUsageWithPrecision float64        `json:"currentUsageWithPrecision"`
	DisplayName               string         `json:"displayName"`
	FreeTrialInfo             *FreeTrialInfo `json:"freeTrialInfo"`
	Bonuses                   []Bonus        `json:"bonuses"`
}

// UsageInfo 計算後的用量資訊
type UsageInfo struct {
	SubscriptionTitle string  // 訂閱類型名稱
	UsageLimit        float64 // 總額度
	CurrentUsage      float64 // 已使用
	Balance           float64 // 餘額 = UsageLimit - CurrentUsage
	IsLowBalance      bool    // 餘額低於 20%
}

// CalculateBalance 從 API 響應計算餘額（使用預設閾值 0.2）
// Property 1: Balance Calculation Correctness
// Validates: Requirements 1.3
// 計算邏輯：
// - 總額度 = Σ(usageLimitWithPrecision + freeTrialInfo?.usageLimitWithPrecision + Σ(bonuses[].usageLimit))
// - 總使用 = Σ(currentUsageWithPrecision + freeTrialInfo?.currentUsageWithPrecision + Σ(bonuses[].currentUsage))
func CalculateBalance(response *UsageLimitsResponse) *UsageInfo {
	return CalculateBalanceWithThreshold(response, 0.2)
}

// CalculateBalanceWithThreshold 從 API 響應計算餘額（使用指定閾值）
// threshold: 低餘額閾值（0.0 ~ 1.0），例如 0.2 表示餘額低於 20% 時為低餘額
func CalculateBalanceWithThreshold(response *UsageLimitsResponse, threshold float64) *UsageInfo {
	if response == nil {
		return &UsageInfo{}
	}

	var totalUsageLimit float64
	var totalCurrentUsage float64

	for _, breakdown := range response.UsageBreakdownList {
		// 基本額度
		totalUsageLimit += breakdown.UsageLimitWithPrecision
		totalCurrentUsage += breakdown.CurrentUsageWithPrecision

		// 免費試用額度（如果存在且未過期）
		if breakdown.FreeTrialInfo != nil && breakdown.FreeTrialInfo.FreeTrialStatus != "EXPIRED" {
			totalUsageLimit += breakdown.FreeTrialInfo.UsageLimitWithPrecision
			totalCurrentUsage += breakdown.FreeTrialInfo.CurrentUsageWithPrecision
		}

		// 獎勵額度（如果存在且未耗盡）
		for _, bonus := range breakdown.Bonuses {
			if bonus.Status != "EXHAUSTED" {
				totalUsageLimit += bonus.UsageLimit
				totalCurrentUsage += bonus.CurrentUsage
			}
		}
	}

	balance := totalUsageLimit - totalCurrentUsage

	// Property 2: Low Balance Detection
	// Validates: Requirements 3.2
	// IsLowBalance = (Balance / TotalUsageLimit) < threshold
	var isLowBalance bool
	if totalUsageLimit > 0 {
		isLowBalance = (balance / totalUsageLimit) < threshold
	}

	return &UsageInfo{
		SubscriptionTitle: response.SubscriptionInfo.SubscriptionTitle,
		UsageLimit:        totalUsageLimit,
		CurrentUsage:      totalCurrentUsage,
		Balance:           balance,
		IsLowBalance:      isLowBalance,
	}
}

// GetUsageLimits 呼叫 API 取得用量資訊（使用當前系統 Machine ID）
// Requirements: 2.1, 2.2, 2.3
func GetUsageLimits(token *awssso.KiroAuthToken) (*UsageInfo, error) {
	machineID, err := machineid.GetMachineId()
	if err != nil {
		return nil, fmt.Errorf("failed to get machine id: %w", err)
	}
	return GetUsageLimitsWithMachineID(token, machineID)
}

// GetUsageLimitsWithMachineID 呼叫 API 取得用量資訊（使用指定的 Machine ID）
// machineID 應為 SHA256 雜湊後的值
// Requirements: 2.1, 2.2, 2.3
// 支援兩種認證類型：
// - social (GitHub/Google): 需要 profileArn 作為 query parameter
// - idc (AWS Identity Center): 不需要 profileArn
func GetUsageLimitsWithMachineID(token *awssso.KiroAuthToken, machineID string) (*UsageInfo, error) {
	if token == nil || token.AccessToken == "" {
		return nil, fmt.Errorf("invalid token: missing accessToken")
	}

	// social 類型需要 profileArn
	if token.AuthMethod == "social" && token.ProfileArn == "" {
		return nil, fmt.Errorf("invalid token: social auth requires profileArn")
	}

	if machineID == "" {
		return nil, fmt.Errorf("invalid machineID: empty")
	}

	// 建構 API URL with query parameters
	// Requirements: 2.2 - social 類型使用 profileArn 作為 query parameter
	apiURL, err := url.Parse(usageLimitsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	query := apiURL.Query()
	query.Set("origin", originParam)
	query.Set("resourceType", resourceTypeParam)
	// 只有 social 類型才加入 profileArn
	if token.AuthMethod == "social" && token.ProfileArn != "" {
		query.Set("profileArn", token.ProfileArn)
	}
	apiURL.RawQuery = query.Encode()

	// 建立 HTTP 請求
	req, err := http.NewRequest(http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 設定 Headers
	// Requirements: 2.1 - 使用 accessToken 作為 Bearer authorization
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	// Requirements: 2.3 - 設定 User-Agent headers
	// 格式: aws-sdk-js/1.0.0 ua/2.1 os/{os}#{osVersion} lang/js md/nodejs#{nodeVersion} api/codewhispererruntime#1.0.0 m/N,E KiroIDE-{kiroVersion}-{machineIdSHA256}
	osName := runtime.GOOS
	kiroVersion := getEffectiveKiroVersion()
	userAgent := fmt.Sprintf("aws-sdk-js/1.0.0 ua/2.1 os/%s lang/go api/codewhispererruntime#1.0.0 m/N,E KiroIDE-%s-%s",
		osName, kiroVersion, machineID)
	req.Header.Set("User-Agent", userAgent)

	// x-amz-user-agent header
	xAmzUserAgent := fmt.Sprintf("aws-sdk-js/1.0.0 KiroIDE-%s-%s", kiroVersion, machineID)
	req.Header.Set("x-amz-user-agent", xAmzUserAgent)

	// DEBUG: 顯示 User-Agent headers
	fmt.Printf("[DEBUG] User-Agent: %s\n", userAgent)
	fmt.Printf("[DEBUG] x-amz-user-agent: %s\n", xAmzUserAgent)

	// amz-sdk-invocation-id: 每次請求隨機生成 UUID
	req.Header.Set("amz-sdk-invocation-id", uuid.New().String())

	// amz-sdk-request header
	req.Header.Set("amz-sdk-request", "attempt=1; max=1")

	// Connection header
	req.Header.Set("Connection", "close")

	// 發送 HTTP GET 請求
	// Requirements: 1.4 - 設定超時以避免長時間等待
	client := &http.Client{
		Timeout: httpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 檢查 HTTP 狀態碼
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析 JSON 響應
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response UsageLimitsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// 計算餘額並返回 UsageInfo
	return CalculateBalance(&response), nil
}

// GetUsageLimitsSafe 安全地呼叫 API 取得用量資訊（使用當前系統 Machine ID）
// Property 4: Error Handling Graceful Degradation
// Validates: Requirements 1.4
// 當發生任何錯誤時，返回空的 UsageInfo 而非 panic
func GetUsageLimitsSafe(token *awssso.KiroAuthToken) *UsageInfo {
	if token == nil {
		return &UsageInfo{}
	}

	info, err := GetUsageLimits(token)
	if err != nil {
		// 錯誤時返回空的 UsageInfo，不 panic
		return &UsageInfo{}
	}

	if info == nil {
		return &UsageInfo{}
	}

	return info
}

// GetUsageLimitsSafeWithMachineID 安全地呼叫 API 取得用量資訊（使用指定的 Machine ID）
// machineID 應為 SHA256 雜湊後的值
// Property 4: Error Handling Graceful Degradation
// Validates: Requirements 1.4
// 當發生任何錯誤時，返回空的 UsageInfo 而非 panic
func GetUsageLimitsSafeWithMachineID(token *awssso.KiroAuthToken, machineID string) *UsageInfo {
	if token == nil || machineID == "" {
		fmt.Printf("[DEBUG] GetUsageLimitsSafeWithMachineID: token=%v, machineID=%s\n", token != nil, machineID)
		return &UsageInfo{}
	}

	info, err := GetUsageLimitsWithMachineID(token, machineID)
	if err != nil {
		// 錯誤時返回空的 UsageInfo，不 panic
		fmt.Printf("[DEBUG] GetUsageLimitsWithMachineID error: %v\n", err)
		return &UsageInfo{}
	}

	if info == nil {
		return &UsageInfo{}
	}

	fmt.Printf("[DEBUG] GetUsageLimitsWithMachineID success: %+v\n", info)
	return info
}
