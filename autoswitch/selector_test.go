package autoswitch

import (
	"testing"
)

// 測試用快照資料
func testSnapshots() []CandidateSnapshot {
	return []CandidateSnapshot{
		{Name: "帳號A", Balance: 3, SubscriptionType: "Free", FolderId: "folder-1"},
		{Name: "帳號B", Balance: 150, SubscriptionType: "Pro", FolderId: "folder-1"},
		{Name: "帳號C", Balance: 80, SubscriptionType: "Pro", FolderId: "folder-2"},
		{Name: "帳號D", Balance: 30, SubscriptionType: "Enterprise", FolderId: "folder-2"},
		{Name: "帳號E", Balance: 200, SubscriptionType: "Enterprise", FolderId: "folder-3"},
	}
}

// TestFilterByFolder 驗證文件夾篩選
func TestFilterByFolder(t *testing.T) {
	config := &AutoSwitchSettings{
		Enabled:          true,
		MinTargetBalance: 0,
		FolderIds:        []string{"folder-1"},
	}

	candidates := FilterCandidates(config, "帳號A", testSnapshots())

	// 應該只有 folder-1 的快照（排除當前的帳號A）
	if len(candidates) != 1 {
		t.Errorf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Name != "帳號B" {
		t.Errorf("expected 帳號B, got %s", candidates[0].Name)
	}
}

// TestFilterBySubscription 驗證訂閱篩選
func TestFilterBySubscription(t *testing.T) {
	config := &AutoSwitchSettings{
		Enabled:           true,
		MinTargetBalance:  0,
		SubscriptionTypes: []string{"Pro"},
	}

	candidates := FilterCandidates(config, "帳號A", testSnapshots())

	// 應該只有 Pro 訂閱的快照
	if len(candidates) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(candidates))
	}
	for _, c := range candidates {
		if c.SubscriptionType != "Pro" {
			t.Errorf("expected Pro subscription, got %s", c.SubscriptionType)
		}
	}
}

// TestFilterByMinBalance 驗證最低餘額篩選
func TestFilterByMinBalance(t *testing.T) {
	config := &AutoSwitchSettings{
		Enabled:          true,
		MinTargetBalance: 50,
	}

	candidates := FilterCandidates(config, "帳號A", testSnapshots())

	// 應該只有餘額 >= 50 的快照
	for _, c := range candidates {
		if c.Balance < 50 {
			t.Errorf("expected balance >= 50, got %f for %s", c.Balance, c.Name)
		}
	}

	// 帳號B(150), 帳號C(80), 帳號E(200) 符合條件
	if len(candidates) != 3 {
		t.Errorf("expected 3 candidates, got %d", len(candidates))
	}
}

// TestCombinedFilters 驗證組合篩選
func TestCombinedFilters(t *testing.T) {
	config := &AutoSwitchSettings{
		Enabled:           true,
		MinTargetBalance:  50,
		FolderIds:         []string{"folder-1", "folder-2"},
		SubscriptionTypes: []string{"Pro"},
	}

	candidates := FilterCandidates(config, "帳號A", testSnapshots())

	// 應該只有：
	// - 餘額 >= 50
	// - 在 folder-1 或 folder-2
	// - 訂閱類型為 Pro
	// 符合條件：帳號B(150, Pro, folder-1), 帳號C(80, Pro, folder-2)
	if len(candidates) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(candidates))
	}

	// 驗證按餘額降序排列
	if candidates[0].Name != "帳號B" {
		t.Errorf("expected first candidate to be 帳號B, got %s", candidates[0].Name)
	}
	if candidates[1].Name != "帳號C" {
		t.Errorf("expected second candidate to be 帳號C, got %s", candidates[1].Name)
	}
}

// TestSelectBestCandidate 驗證選擇餘額最高者
func TestSelectBestCandidate(t *testing.T) {
	candidates := []CandidateSnapshot{
		{Name: "帳號B", Balance: 150},
		{Name: "帳號C", Balance: 80},
		{Name: "帳號E", Balance: 200},
	}

	// 先排序
	config := &AutoSwitchSettings{MinTargetBalance: 0}
	sorted := FilterCandidates(config, "", candidates)

	best := SelectBestCandidate(sorted)
	if best == nil {
		t.Fatal("expected a candidate, got nil")
	}
	if best.Name != "帳號E" {
		t.Errorf("expected 帳號E (highest balance), got %s", best.Name)
	}
}

// TestSelectBestCandidate_Empty 驗證空列表
func TestSelectBestCandidate_Empty(t *testing.T) {
	best := SelectBestCandidate(nil)
	if best != nil {
		t.Error("expected nil for empty candidates")
	}

	best = SelectBestCandidate([]CandidateSnapshot{})
	if best != nil {
		t.Error("expected nil for empty slice")
	}
}

// TestFilterCandidates_ExcludesCurrent 驗證排除當前快照
func TestFilterCandidates_ExcludesCurrent(t *testing.T) {
	config := &AutoSwitchSettings{
		Enabled:          true,
		MinTargetBalance: 0,
	}

	candidates := FilterCandidates(config, "帳號B", testSnapshots())

	for _, c := range candidates {
		if c.Name == "帳號B" {
			t.Error("current snapshot should be excluded")
		}
	}
}

// TestFilterCandidates_NilConfig 驗證 nil 設定
func TestFilterCandidates_NilConfig(t *testing.T) {
	candidates := FilterCandidates(nil, "帳號A", testSnapshots())
	if candidates != nil {
		t.Error("expected nil for nil config")
	}
}

// TestFilterCandidates_EmptySnapshots 驗證空快照列表
func TestFilterCandidates_EmptySnapshots(t *testing.T) {
	config := &AutoSwitchSettings{Enabled: true}
	candidates := FilterCandidates(config, "帳號A", nil)
	if candidates != nil {
		t.Error("expected nil for empty snapshots")
	}

	candidates = FilterCandidates(config, "帳號A", []CandidateSnapshot{})
	if candidates != nil {
		t.Error("expected nil for empty slice")
	}
}

// TestFilterCandidates_SortedByBalance 驗證按餘額降序排列
func TestFilterCandidates_SortedByBalance(t *testing.T) {
	config := &AutoSwitchSettings{
		Enabled:          true,
		MinTargetBalance: 0,
	}

	candidates := FilterCandidates(config, "", testSnapshots())

	// 驗證降序排列
	for i := 0; i < len(candidates)-1; i++ {
		if candidates[i].Balance < candidates[i+1].Balance {
			t.Errorf("candidates not sorted by balance: %f < %f",
				candidates[i].Balance, candidates[i+1].Balance)
		}
	}
}
