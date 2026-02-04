package main

import (
	"os"
	"testing"

	"kiro-manager/backup"
)

// TestDeleteFolder_WithActiveSnapshot_MoveToUncategorized 測試當 deleteSnapshots=false 且文件夾包含活躍快照時，應該返回錯誤
// 根據規格：無論選擇「一併刪除」還是「移到未分類」，都應該檢查是否包含當前使用中的快照
func TestDeleteFolder_WithActiveSnapshot_MoveToUncategorized(t *testing.T) {
	// 清理環境
	path, _ := backup.GetFoldersPath()
	os.Remove(path)
	defer os.Remove(path)

	// 建立測試用的 App 實例
	app := NewApp()

	// 建立文件夾
	folder, err := backup.CreateFolder("舊帳號")
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	// 取得當前 Machine ID
	currentMachineID := app.GetCurrentMachineID()
	if currentMachineID == "" {
		t.Skip("無法取得當前 Machine ID，跳過測試")
	}

	// 建立一個使用當前 Machine ID 的備份
	testBackupName := "active-account-test"
	
	// 確保備份目錄存在並建立備份
	if err := backup.CreateBackup(testBackupName); err != nil {
		// 如果備份已存在，先刪除再建立
		backup.DeleteBackup(testBackupName)
		if err := backup.CreateBackup(testBackupName); err != nil {
			t.Fatalf("CreateBackup failed: %v", err)
		}
	}
	defer backup.DeleteBackup(testBackupName)

	// 更新備份的 Machine ID 為當前 Machine ID（模擬活躍快照）
	if err := backup.UpdateBackupMachineID(testBackupName, currentMachineID); err != nil {
		t.Fatalf("UpdateBackupMachineID failed: %v", err)
	}

	// 將快照分配到文件夾
	if err := backup.AssignSnapshotToFolder(testBackupName, folder.ID); err != nil {
		t.Fatalf("AssignSnapshotToFolder failed: %v", err)
	}

	// 嘗試刪除文件夾（選擇移到未分類，deleteSnapshots=false）
	result := app.DeleteFolder(folder.ID, false)

	// 應該返回錯誤，因為文件夾包含當前使用中的快照
	if result.Success {
		t.Errorf("Expected DeleteFolder to fail when folder contains active snapshot with deleteSnapshots=false, but it succeeded")
	}

	// 驗證錯誤訊息
	expectedMessage := "無法刪除包含當前使用中環境的文件夾"
	if result.Message != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, result.Message)
	}

	// 驗證文件夾沒有被刪除
	folders, err := backup.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders failed: %v", err)
	}

	found := false
	for _, f := range folders {
		if f.ID == folder.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Folder should not be deleted when it contains active snapshot")
	}
}

// TestDeleteFolder_WithActiveSnapshot_DeleteSnapshots 測試當 deleteSnapshots=true 且文件夾包含活躍快照時，應該返回錯誤
// 這是原有的行為，確保修改後仍然正常工作
func TestDeleteFolder_WithActiveSnapshot_DeleteSnapshots(t *testing.T) {
	// 清理環境
	path, _ := backup.GetFoldersPath()
	os.Remove(path)
	defer os.Remove(path)

	// 建立測試用的 App 實例
	app := NewApp()

	// 建立文件夾
	folder, err := backup.CreateFolder("舊帳號")
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	// 取得當前 Machine ID
	currentMachineID := app.GetCurrentMachineID()
	if currentMachineID == "" {
		t.Skip("無法取得當前 Machine ID，跳過測試")
	}

	// 建立一個使用當前 Machine ID 的備份
	testBackupName := "active-account-test-2"
	
	// 確保備份目錄存在並建立備份
	if err := backup.CreateBackup(testBackupName); err != nil {
		backup.DeleteBackup(testBackupName)
		if err := backup.CreateBackup(testBackupName); err != nil {
			t.Fatalf("CreateBackup failed: %v", err)
		}
	}
	defer backup.DeleteBackup(testBackupName)

	// 更新備份的 Machine ID 為當前 Machine ID（模擬活躍快照）
	if err := backup.UpdateBackupMachineID(testBackupName, currentMachineID); err != nil {
		t.Fatalf("UpdateBackupMachineID failed: %v", err)
	}

	// 將快照分配到文件夾
	if err := backup.AssignSnapshotToFolder(testBackupName, folder.ID); err != nil {
		t.Fatalf("AssignSnapshotToFolder failed: %v", err)
	}

	// 嘗試刪除文件夾（選擇一併刪除，deleteSnapshots=true）
	result := app.DeleteFolder(folder.ID, true)

	// 應該返回錯誤，因為文件夾包含當前使用中的快照
	if result.Success {
		t.Errorf("Expected DeleteFolder to fail when folder contains active snapshot with deleteSnapshots=true, but it succeeded")
	}

	// 驗證錯誤訊息
	expectedMessage := "無法刪除包含當前使用中環境的文件夾"
	if result.Message != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, result.Message)
	}
}

// TestDeleteFolder_WithoutActiveSnapshot_MoveToUncategorized 測試當文件夾不包含活躍快照時，可以正常刪除（移到未分類）
func TestDeleteFolder_WithoutActiveSnapshot_MoveToUncategorized(t *testing.T) {
	// 清理環境
	path, _ := backup.GetFoldersPath()
	os.Remove(path)
	defer os.Remove(path)

	// 建立測試用的 App 實例
	app := NewApp()

	// 建立文件夾
	folder, err := backup.CreateFolder("舊帳號")
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	// 手動添加一個不是活躍快照的 assignment
	data, _ := backup.LoadFolders()
	data.Assignments["inactive-snapshot"] = folder.ID
	backup.SaveFolders(data)

	// 嘗試刪除文件夾（選擇移到未分類）
	result := app.DeleteFolder(folder.ID, false)

	// 應該成功
	if !result.Success {
		t.Errorf("Expected DeleteFolder to succeed when folder does not contain active snapshot, but it failed: %s", result.Message)
	}

	// 驗證文件夾已被刪除
	folders, err := backup.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders failed: %v", err)
	}

	for _, f := range folders {
		if f.ID == folder.ID {
			t.Error("Folder should be deleted")
		}
	}
}
