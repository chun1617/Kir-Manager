package deeplink

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// OAuthState 定義 OAuth State 結構
type OAuthState struct {
	State         string    `json:"state"`
	Provider      string    `json:"provider"`
	CodeVerifier  string    `json:"code_verifier"`
	CodeChallenge string    `json:"code_challenge"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// SaveState 將 State 參數持久化到臨時檔案
func SaveState(state *OAuthState) error {
	statePath := getStatePath()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0600)
}

// LoadState 從臨時檔案讀取 State 參數
func LoadState() (*OAuthState, error) {
	statePath := getStatePath()

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrStateNotFound
		}
		return nil, err
	}

	var state OAuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// ClearState 刪除臨時檔案
func ClearState() error {
	statePath := getStatePath()

	err := os.Remove(statePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// IsStateExpired 檢查 State 是否已過期
func IsStateExpired(state *OAuthState) bool {
	return time.Now().After(state.ExpiresAt)
}

// ValidateState 驗證 State 是否匹配
func ValidateState(state *OAuthState, expectedState string) error {
	if state.State != expectedState {
		return ErrStateMismatch
	}
	return nil
}

// getStatePath 取得臨時檔案路徑
func getStatePath() string {
	return filepath.Join(os.TempDir(), StateFileName)
}
