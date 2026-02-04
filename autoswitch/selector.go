package autoswitch

import (
	"sort"
)

// CandidateSnapshot 候選快照結構
type CandidateSnapshot struct {
	Name             string  `json:"name"`
	Balance          float64 `json:"balance"`
	SubscriptionType string  `json:"subscriptionType"`
	FolderId         string  `json:"folderId"`
}

// FilterCandidates 篩選符合條件的候選快照
// 參數：
//   - config: 自動切換設定
//   - currentName: 當前快照名稱（會被排除）
//   - allSnapshots: 所有可用快照
//
// 返回：符合條件的候選快照列表（按餘額降序排列）
func FilterCandidates(config *AutoSwitchSettings, currentName string, allSnapshots []CandidateSnapshot) []CandidateSnapshot {
	if config == nil || len(allSnapshots) == 0 {
		return nil
	}

	var candidates []CandidateSnapshot

	for _, snapshot := range allSnapshots {
		// 排除當前快照
		if snapshot.Name == currentName {
			continue
		}

		// 檢查最低餘額要求
		if snapshot.Balance < config.MinTargetBalance {
			continue
		}

		// 檢查文件夾篩選
		if len(config.FolderIds) > 0 {
			if !containsString(config.FolderIds, snapshot.FolderId) {
				continue
			}
		}

		// 檢查訂閱類型篩選
		if len(config.SubscriptionTypes) > 0 {
			if !containsString(config.SubscriptionTypes, snapshot.SubscriptionType) {
				continue
			}
		}

		candidates = append(candidates, snapshot)
	}

	// 按餘額降序排列
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Balance > candidates[j].Balance
	})

	return candidates
}

// SelectBestCandidate 選擇餘額最高的候選
// 返回 nil 表示沒有可用候選
func SelectBestCandidate(candidates []CandidateSnapshot) *CandidateSnapshot {
	if len(candidates) == 0 {
		return nil
	}
	// 假設已按餘額降序排列，返回第一個
	return &candidates[0]
}

// containsString 檢查字串切片是否包含指定字串
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
