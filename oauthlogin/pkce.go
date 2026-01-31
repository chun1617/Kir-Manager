// Package oauthlogin 提供 OAuth 登入功能
package oauthlogin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// PKCEParams PKCE 參數結構
type PKCEParams struct {
	// CodeVerifier 原始驗證碼 (base64url 編碼)
	CodeVerifier string
	// CodeChallenge 驗證碼的 SHA256 雜湊 (base64url 編碼)
	CodeChallenge string
	// State 隨機狀態參數，用於防止 CSRF
	State string
}

// GeneratePKCE 生成 PKCE 參數
// 返回包含 code_verifier、code_challenge 和 state 的 PKCEParams
func GeneratePKCE() (*PKCEParams, error) {
	// 生成 32 bytes 隨機數據作為 code_verifier
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, err
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// 計算 SHA256 雜湊作為 code_challenge
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// 生成 16 bytes 隨機 state
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, err
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	return &PKCEParams{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		State:         state,
	}, nil
}

// ValidateState 驗證 state 參數是否匹配
func ValidateState(expected, actual string) bool {
	return expected == actual && expected != ""
}
