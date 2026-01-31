package deeplink

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveState_Success(t *testing.T) {
	// Arrange
	state := &OAuthState{
		State:         "test-state-123",
		Provider:      "github",
		CodeVerifier:  "test-verifier",
		CodeChallenge: "test-challenge",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(StateExpiry),
	}

	// Cleanup before and after test
	ClearState()
	defer ClearState()

	// Act
	err := SaveState(state)

	// Assert
	if err != nil {
		t.Fatalf("SaveState() error = %v, want nil", err)
	}

	// Verify file exists
	statePath := filepath.Join(os.TempDir(), StateFileName)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Errorf("State file not created at %s", statePath)
	}
}

func TestLoadState_Success(t *testing.T) {
	// Arrange
	expectedState := &OAuthState{
		State:         "test-state-456",
		Provider:      "github",
		CodeVerifier:  "test-verifier-456",
		CodeChallenge: "test-challenge-456",
		CreatedAt:     time.Now().Truncate(time.Second),
		ExpiresAt:     time.Now().Add(StateExpiry).Truncate(time.Second),
	}

	// Cleanup before and after test
	ClearState()
	defer ClearState()

	// Save state first
	if err := SaveState(expectedState); err != nil {
		t.Fatalf("Failed to save state for test: %v", err)
	}

	// Act
	loadedState, err := LoadState()

	// Assert
	if err != nil {
		t.Fatalf("LoadState() error = %v, want nil", err)
	}

	if loadedState.State != expectedState.State {
		t.Errorf("LoadState().State = %v, want %v", loadedState.State, expectedState.State)
	}
	if loadedState.Provider != expectedState.Provider {
		t.Errorf("LoadState().Provider = %v, want %v", loadedState.Provider, expectedState.Provider)
	}
	if loadedState.CodeVerifier != expectedState.CodeVerifier {
		t.Errorf("LoadState().CodeVerifier = %v, want %v", loadedState.CodeVerifier, expectedState.CodeVerifier)
	}
	if loadedState.CodeChallenge != expectedState.CodeChallenge {
		t.Errorf("LoadState().CodeChallenge = %v, want %v", loadedState.CodeChallenge, expectedState.CodeChallenge)
	}
}

func TestLoadState_NotFound(t *testing.T) {
	// Arrange - ensure no state file exists
	ClearState()

	// Act
	_, err := LoadState()

	// Assert
	if err != ErrStateNotFound {
		t.Errorf("LoadState() error = %v, want %v", err, ErrStateNotFound)
	}
}

func TestIsStateExpired_NotExpired(t *testing.T) {
	// Arrange
	state := &OAuthState{
		State:     "test-state",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(StateExpiry),
	}

	// Act
	expired := IsStateExpired(state)

	// Assert
	if expired {
		t.Errorf("IsStateExpired() = true, want false for non-expired state")
	}
}

func TestIsStateExpired_Expired(t *testing.T) {
	// Arrange
	state := &OAuthState{
		State:     "test-state",
		CreatedAt: time.Now().Add(-10 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}

	// Act
	expired := IsStateExpired(state)

	// Assert
	if !expired {
		t.Errorf("IsStateExpired() = false, want true for expired state")
	}
}

func TestClearState_Success(t *testing.T) {
	// Arrange - create a state file first
	state := &OAuthState{
		State:     "test-state-to-clear",
		Provider:  "github",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(StateExpiry),
	}
	if err := SaveState(state); err != nil {
		t.Fatalf("Failed to save state for test: %v", err)
	}

	// Verify file exists before clearing
	statePath := filepath.Join(os.TempDir(), StateFileName)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatalf("State file should exist before clearing")
	}

	// Act
	err := ClearState()

	// Assert
	if err != nil {
		t.Fatalf("ClearState() error = %v, want nil", err)
	}

	// Verify file no longer exists
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Errorf("State file should not exist after clearing")
	}
}

func TestValidateState_Match(t *testing.T) {
	// Arrange
	state := &OAuthState{
		State:     "matching-state-123",
		Provider:  "github",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(StateExpiry),
	}

	// Act
	err := ValidateState(state, "matching-state-123")

	// Assert
	if err != nil {
		t.Errorf("ValidateState() error = %v, want nil for matching state", err)
	}
}

func TestValidateState_Mismatch(t *testing.T) {
	// Arrange
	state := &OAuthState{
		State:     "original-state",
		Provider:  "github",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(StateExpiry),
	}

	// Act
	err := ValidateState(state, "different-state")

	// Assert
	if err != ErrStateMismatch {
		t.Errorf("ValidateState() error = %v, want %v for mismatched state", err, ErrStateMismatch)
	}
}
