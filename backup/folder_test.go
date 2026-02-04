package backup

import (
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/quick"
	"unicode"

	"github.com/google/uuid"
)

// TestFolderStructure 測試 Folder 結構體欄位
func TestFolderStructure(t *testing.T) {
	folder := Folder{
		ID:        "test-uuid",
		Name:      "工作帳號",
		CreatedAt: "2024-01-01T00:00:00Z",
		Order:     0,
	}

	if folder.ID != "test-uuid" {
		t.Errorf("expected ID 'test-uuid', got '%s'", folder.ID)
	}
	if folder.Name != "工作帳號" {
		t.Errorf("expected Name '工作帳號', got '%s'", folder.Name)
	}
}

// TestFoldersDataStructure 測試 FoldersData 結構體
func TestFoldersDataStructure(t *testing.T) {
	data := FoldersData{
		Folders: []Folder{
			{ID: "uuid-1", Name: "文件夾1", CreatedAt: "2024-01-01T00:00:00Z", Order: 0},
		},
		Assignments: map[string]string{
			"snapshot-1": "uuid-1",
		},
	}

	if len(data.Folders) != 1 {
		t.Errorf("expected 1 folder, got %d", len(data.Folders))
	}
	if data.Assignments["snapshot-1"] != "uuid-1" {
		t.Errorf("expected assignment 'uuid-1', got '%s'", data.Assignments["snapshot-1"])
	}
}

// TestFolderErrors 測試錯誤類型定義
func TestFolderErrors(t *testing.T) {
	if ErrFolderNotFound == nil {
		t.Error("ErrFolderNotFound should not be nil")
	}
	if ErrFolderExists == nil {
		t.Error("ErrFolderExists should not be nil")
	}
	if ErrFolderNameEmpty == nil {
		t.Error("ErrFolderNameEmpty should not be nil")
	}
	if ErrFolderNameInvalid == nil {
		t.Error("ErrFolderNameInvalid should not be nil")
	}
}

// TestValidateFolderName_Empty 測試空名稱
func TestValidateFolderName_Empty(t *testing.T) {
	err := ValidateFolderName("")
	if err != ErrFolderNameEmpty {
		t.Errorf("expected ErrFolderNameEmpty, got %v", err)
	}
}

// TestValidateFolderName_InvalidChars 測試非法字元
func TestValidateFolderName_InvalidChars(t *testing.T) {
	invalidNames := []string{
		"工作/帳號",
		"工作\\帳號",
		"工作:帳號",
		"工作*帳號",
		"工作?帳號",
		"工作\"帳號",
		"工作<帳號",
		"工作>帳號",
		"工作|帳號",
	}

	for _, name := range invalidNames {
		err := ValidateFolderName(name)
		if err != ErrFolderNameInvalid {
			t.Errorf("expected ErrFolderNameInvalid for '%s', got %v", name, err)
		}
	}
}

// TestValidateFolderName_Valid 測試有效名稱
func TestValidateFolderName_Valid(t *testing.T) {
	validNames := []string{
		"工作帳號",
		"my-folder",
		"folder_123",
		"文件夾 (1)",
	}

	for _, name := range validNames {
		err := ValidateFolderName(name)
		if err != nil {
			t.Errorf("expected nil for '%s', got %v", name, err)
		}
	}
}

// TestGetFoldersPath 測試取得 folders.json 路徑
func TestGetFoldersPath(t *testing.T) {
	path, err := GetFoldersPath()
	if err != nil {
		t.Fatalf("GetFoldersPath failed: %v", err)
	}

	if !strings.HasSuffix(path, "folders.json") {
		t.Errorf("expected path to end with 'folders.json', got '%s'", path)
	}
}

// TestLoadFolders_NotExists 測試載入不存在的 folders.json
func TestLoadFolders_NotExists(t *testing.T) {
	// 確保檔案不存在
	path, _ := GetFoldersPath()
	os.Remove(path)

	data, err := LoadFolders()
	if err != nil {
		t.Fatalf("LoadFolders failed: %v", err)
	}

	if data == nil {
		t.Fatal("expected non-nil FoldersData")
	}
	if len(data.Folders) != 0 {
		t.Errorf("expected empty folders, got %d", len(data.Folders))
	}
	if data.Assignments == nil {
		t.Error("expected non-nil Assignments map")
	}
}

// TestSaveFolders 測試儲存 folders.json
func TestSaveFolders(t *testing.T) {
	data := &FoldersData{
		Folders: []Folder{
			{ID: "test-id", Name: "測試", CreatedAt: "2024-01-01T00:00:00Z", Order: 0},
		},
		Assignments: map[string]string{},
	}

	err := SaveFolders(data)
	if err != nil {
		t.Fatalf("SaveFolders failed: %v", err)
	}

	// 驗證可以讀回
	loaded, err := LoadFolders()
	if err != nil {
		t.Fatalf("LoadFolders failed: %v", err)
	}

	if len(loaded.Folders) != 1 {
		t.Errorf("expected 1 folder, got %d", len(loaded.Folders))
	}
	if loaded.Folders[0].Name != "測試" {
		t.Errorf("expected name '測試', got '%s'", loaded.Folders[0].Name)
	}

	// 清理
	path, _ := GetFoldersPath()
	os.Remove(path)
}


// ==================== Task 2.1: CreateFolder 測試 ====================

// TestCreateFolder_Success 測試成功建立文件夾
func TestCreateFolder_Success(t *testing.T) {
	// 清理環境
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, err := CreateFolder("工作帳號")
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	if folder.Name != "工作帳號" {
		t.Errorf("expected name '工作帳號', got '%s'", folder.Name)
	}
	if folder.ID == "" {
		t.Error("expected non-empty ID")
	}
	// 驗證 ID 是有效的 UUID
	if _, err := uuid.Parse(folder.ID); err != nil {
		t.Errorf("expected valid UUID, got '%s'", folder.ID)
	}
	if folder.Order != 0 {
		t.Errorf("expected order 0, got %d", folder.Order)
	}

	// 驗證已儲存
	data, _ := LoadFolders()
	if len(data.Folders) != 1 {
		t.Errorf("expected 1 folder in storage, got %d", len(data.Folders))
	}

	// 清理
	os.Remove(path)
}

// TestCreateFolder_EmptyName 測試空名稱
func TestCreateFolder_EmptyName(t *testing.T) {
	_, err := CreateFolder("")
	if err != ErrFolderNameEmpty {
		t.Errorf("expected ErrFolderNameEmpty, got %v", err)
	}
}

// TestCreateFolder_InvalidName 測試非法字元
func TestCreateFolder_InvalidName(t *testing.T) {
	_, err := CreateFolder("工作/帳號")
	if err != ErrFolderNameInvalid {
		t.Errorf("expected ErrFolderNameInvalid, got %v", err)
	}
}

// TestCreateFolder_Duplicate 測試重複名稱
func TestCreateFolder_Duplicate(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	_, err := CreateFolder("工作帳號")
	if err != nil {
		t.Fatalf("first CreateFolder failed: %v", err)
	}

	_, err = CreateFolder("工作帳號")
	if err != ErrFolderExists {
		t.Errorf("expected ErrFolderExists, got %v", err)
	}

	os.Remove(path)
}

// ==================== Task 2.2: RenameFolder 測試 ====================

// TestRenameFolder_Success 測試成功重新命名
func TestRenameFolder_Success(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("工作帳號")

	err := RenameFolder(folder.ID, "公司帳號")
	if err != nil {
		t.Fatalf("RenameFolder failed: %v", err)
	}

	// 驗證已更新
	data, _ := LoadFolders()
	if data.Folders[0].Name != "公司帳號" {
		t.Errorf("expected name '公司帳號', got '%s'", data.Folders[0].Name)
	}

	os.Remove(path)
}

// TestRenameFolder_NotFound 測試文件夾不存在
func TestRenameFolder_NotFound(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	err := RenameFolder("non-existent-id", "新名稱")
	if err != ErrFolderNotFound {
		t.Errorf("expected ErrFolderNotFound, got %v", err)
	}
}

// TestRenameFolder_Duplicate 測試重新命名為已存在的名稱
func TestRenameFolder_Duplicate(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder1, _ := CreateFolder("工作帳號")
	CreateFolder("個人帳號")

	err := RenameFolder(folder1.ID, "個人帳號")
	if err != ErrFolderExists {
		t.Errorf("expected ErrFolderExists, got %v", err)
	}

	os.Remove(path)
}

// TestRenameFolder_EmptyName 測試重新命名為空名稱
func TestRenameFolder_EmptyName(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("工作帳號")

	err := RenameFolder(folder.ID, "")
	if err != ErrFolderNameEmpty {
		t.Errorf("expected ErrFolderNameEmpty, got %v", err)
	}

	os.Remove(path)
}

// TestRenameFolder_InvalidName 測試重新命名為非法名稱
func TestRenameFolder_InvalidName(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("工作帳號")

	err := RenameFolder(folder.ID, "工作/帳號")
	if err != ErrFolderNameInvalid {
		t.Errorf("expected ErrFolderNameInvalid, got %v", err)
	}

	os.Remove(path)
}

// ==================== Task 2.3: DeleteFolder 測試 ====================

// TestDeleteFolder_Empty 測試刪除空文件夾
func TestDeleteFolder_Empty(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("測試帳號")

	movedSnapshots, err := DeleteFolder(folder.ID, false)
	if err != nil {
		t.Fatalf("DeleteFolder failed: %v", err)
	}

	if len(movedSnapshots) != 0 {
		t.Errorf("expected 0 moved snapshots, got %d", len(movedSnapshots))
	}

	data, _ := LoadFolders()
	if len(data.Folders) != 0 {
		t.Errorf("expected 0 folders, got %d", len(data.Folders))
	}

	os.Remove(path)
}

// TestDeleteFolder_MoveToUncategorized 測試刪除非空文件夾（移到未分類）
func TestDeleteFolder_MoveToUncategorized(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("舊帳號")

	// 手動添加 assignment
	data, _ := LoadFolders()
	data.Assignments["account-1"] = folder.ID
	data.Assignments["account-2"] = folder.ID
	SaveFolders(data)

	movedSnapshots, err := DeleteFolder(folder.ID, false)
	if err != nil {
		t.Fatalf("DeleteFolder failed: %v", err)
	}

	if len(movedSnapshots) != 2 {
		t.Errorf("expected 2 moved snapshots, got %d", len(movedSnapshots))
	}

	// 驗證 assignments 已清除
	data, _ = LoadFolders()
	if len(data.Assignments) != 0 {
		t.Errorf("expected 0 assignments, got %d", len(data.Assignments))
	}

	os.Remove(path)
}

// TestDeleteFolder_NotFound 測試刪除不存在的文件夾
func TestDeleteFolder_NotFound(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	_, err := DeleteFolder("non-existent-id", false)
	if err != ErrFolderNotFound {
		t.Errorf("expected ErrFolderNotFound, got %v", err)
	}
}

// TestDeleteFolder_WithDeleteSnapshots 測試刪除文件夾並刪除快照
func TestDeleteFolder_WithDeleteSnapshots(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("舊帳號")

	// 手動添加 assignment
	data, _ := LoadFolders()
	data.Assignments["account-1"] = folder.ID
	data.Assignments["account-2"] = folder.ID
	SaveFolders(data)

	movedSnapshots, err := DeleteFolder(folder.ID, true)
	if err != nil {
		t.Fatalf("DeleteFolder failed: %v", err)
	}

	if len(movedSnapshots) != 2 {
		t.Errorf("expected 2 snapshots returned, got %d", len(movedSnapshots))
	}

	// 驗證 assignments 已清除
	data, _ = LoadFolders()
	if len(data.Assignments) != 0 {
		t.Errorf("expected 0 assignments, got %d", len(data.Assignments))
	}

	os.Remove(path)
}

// ==================== Task 2.4: ListFolders 測試 ====================

// TestListFolders 測試列出文件夾
func TestListFolders(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder1, _ := CreateFolder("工作帳號")
	CreateFolder("個人帳號")

	// 添加 assignments
	data, _ := LoadFolders()
	data.Assignments["snapshot-1"] = folder1.ID
	data.Assignments["snapshot-2"] = folder1.ID
	data.Assignments["snapshot-3"] = folder1.ID
	SaveFolders(data)

	folders, err := ListFolders()
	if err != nil {
		t.Fatalf("ListFolders failed: %v", err)
	}

	if len(folders) != 2 {
		t.Errorf("expected 2 folders, got %d", len(folders))
	}

	// 找到工作帳號並檢查數量
	for _, f := range folders {
		if f.Name == "工作帳號" {
			if f.SnapshotCount != 3 {
				t.Errorf("expected 3 snapshots in '工作帳號', got %d", f.SnapshotCount)
			}
		}
		if f.Name == "個人帳號" {
			if f.SnapshotCount != 0 {
				t.Errorf("expected 0 snapshots in '個人帳號', got %d", f.SnapshotCount)
			}
		}
	}

	os.Remove(path)
}

// TestListFolders_Empty 測試列出空文件夾列表
func TestListFolders_Empty(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folders, err := ListFolders()
	if err != nil {
		t.Fatalf("ListFolders failed: %v", err)
	}

	if len(folders) != 0 {
		t.Errorf("expected 0 folders, got %d", len(folders))
	}
}


// ==================== Task 3.1: 快照歸屬管理測試 ====================

// TestAssignSnapshotToFolder_Success 測試成功將快照移入文件夾
func TestAssignSnapshotToFolder_Success(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("工作帳號")

	err := AssignSnapshotToFolder("my-github", folder.ID)
	if err != nil {
		t.Fatalf("AssignSnapshotToFolder failed: %v", err)
	}

	// 驗證 assignment
	data, _ := LoadFolders()
	if data.Assignments["my-github"] != folder.ID {
		t.Errorf("expected assignment to '%s', got '%s'", folder.ID, data.Assignments["my-github"])
	}

	os.Remove(path)
}

// TestAssignSnapshotToFolder_FolderNotFound 測試文件夾不存在
func TestAssignSnapshotToFolder_FolderNotFound(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	err := AssignSnapshotToFolder("my-github", "non-existent-id")
	if err != ErrFolderNotFound {
		t.Errorf("expected ErrFolderNotFound, got %v", err)
	}
}

// TestAssignSnapshotToFolder_MoveToAnotherFolder 測試從一個文件夾移到另一個
func TestAssignSnapshotToFolder_MoveToAnotherFolder(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder1, _ := CreateFolder("工作帳號")
	folder2, _ := CreateFolder("個人帳號")

	// 先分配到 folder1
	AssignSnapshotToFolder("my-github", folder1.ID)

	// 再移到 folder2
	err := AssignSnapshotToFolder("my-github", folder2.ID)
	if err != nil {
		t.Fatalf("AssignSnapshotToFolder failed: %v", err)
	}

	// 驗證已移到 folder2
	data, _ := LoadFolders()
	if data.Assignments["my-github"] != folder2.ID {
		t.Errorf("expected assignment to '%s', got '%s'", folder2.ID, data.Assignments["my-github"])
	}

	os.Remove(path)
}

// TestUnassignSnapshot_Success 測試成功將快照移至未分類
func TestUnassignSnapshot_Success(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("工作帳號")
	AssignSnapshotToFolder("my-github", folder.ID)

	err := UnassignSnapshot("my-github")
	if err != nil {
		t.Fatalf("UnassignSnapshot failed: %v", err)
	}

	// 驗證已移除
	data, _ := LoadFolders()
	if _, exists := data.Assignments["my-github"]; exists {
		t.Error("expected assignment to be removed")
	}

	os.Remove(path)
}

// TestUnassignSnapshot_NotAssigned 測試移除未分配的快照（應該成功，無操作）
func TestUnassignSnapshot_NotAssigned(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	err := UnassignSnapshot("my-github")
	if err != nil {
		t.Errorf("expected nil error for unassigned snapshot, got %v", err)
	}
}

// TestGetSnapshotFolderId 測試取得快照所屬文件夾 ID
func TestGetSnapshotFolderId(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("工作帳號")
	AssignSnapshotToFolder("my-github", folder.ID)

	folderId, err := GetSnapshotFolderId("my-github")
	if err != nil {
		t.Fatalf("GetSnapshotFolderId failed: %v", err)
	}

	if folderId != folder.ID {
		t.Errorf("expected folder ID '%s', got '%s'", folder.ID, folderId)
	}

	// 測試未分配的快照
	folderId, err = GetSnapshotFolderId("unassigned-snapshot")
	if err != nil {
		t.Fatalf("GetSnapshotFolderId failed: %v", err)
	}
	if folderId != "" {
		t.Errorf("expected empty folder ID for unassigned snapshot, got '%s'", folderId)
	}

	os.Remove(path)
}

// ==================== Task 3.2: 孤兒記錄清理測試 ====================

// TestCleanupOrphanAssignments 測試清理孤兒記錄
func TestCleanupOrphanAssignments(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("工作帳號")

	// 手動添加孤兒 assignment（快照不存在）
	data, _ := LoadFolders()
	data.Assignments["existing-snapshot"] = folder.ID
	data.Assignments["deleted-snapshot"] = folder.ID
	SaveFolders(data)

	// 模擬 existing-snapshot 存在，deleted-snapshot 不存在
	// 這需要一個檢查函數，我們傳入一個 checker
	cleaned, err := CleanupOrphanAssignments(func(name string) bool {
		return name == "existing-snapshot"
	})

	if err != nil {
		t.Fatalf("CleanupOrphanAssignments failed: %v", err)
	}

	if len(cleaned) != 1 {
		t.Errorf("expected 1 cleaned assignment, got %d", len(cleaned))
	}

	if len(cleaned) > 0 && cleaned[0] != "deleted-snapshot" {
		t.Errorf("expected 'deleted-snapshot' to be cleaned, got '%s'", cleaned[0])
	}

	// 驗證只剩下 existing-snapshot
	data, _ = LoadFolders()
	if len(data.Assignments) != 1 {
		t.Errorf("expected 1 assignment, got %d", len(data.Assignments))
	}
	if _, exists := data.Assignments["existing-snapshot"]; !exists {
		t.Error("expected 'existing-snapshot' to remain")
	}

	os.Remove(path)
}

// TestCleanupOrphanAssignments_NoOrphans 測試無孤兒記錄
func TestCleanupOrphanAssignments_NoOrphans(t *testing.T) {
	path, _ := GetFoldersPath()
	os.Remove(path)

	folder, _ := CreateFolder("工作帳號")

	data, _ := LoadFolders()
	data.Assignments["snapshot-1"] = folder.ID
	SaveFolders(data)

	// 所有快照都存在
	cleaned, err := CleanupOrphanAssignments(func(name string) bool {
		return true
	})

	if err != nil {
		t.Fatalf("CleanupOrphanAssignments failed: %v", err)
	}

	if len(cleaned) != 0 {
		t.Errorf("expected 0 cleaned assignments, got %d", len(cleaned))
	}

	os.Remove(path)
}


// ==================== Task 12: Property-Based Tests ====================

// TestProperty_FolderNameValidation 測試文件夾名稱驗證的屬性
func TestProperty_FolderNameValidation(t *testing.T) {
	// 屬性 1：任何包含非法字元的名稱都應該被拒絕
	t.Run("InvalidCharsRejected", func(t *testing.T) {
		invalidChars := []rune{'/', '\\', ':', '*', '?', '"', '<', '>', '|'}

		property := func(base string, charIndex uint8, position uint8) bool {
			if len(base) == 0 {
				return true // 跳過空字串
			}

			// 選擇一個非法字元
			invalidChar := invalidChars[int(charIndex)%len(invalidChars)]

			// 在隨機位置插入非法字元
			pos := int(position) % (len(base) + 1)
			name := base[:pos] + string(invalidChar) + base[pos:]

			err := ValidateFolderName(name)
			return err == ErrFolderNameInvalid
		}

		if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// 屬性 2：任何不包含非法字元的非空名稱都應該被接受
	t.Run("ValidNamesAccepted", func(t *testing.T) {
		invalidChars := map[rune]bool{
			'/': true, '\\': true, ':': true, '*': true,
			'?': true, '"': true, '<': true, '>': true, '|': true,
		}

		property := func(name string) bool {
			if name == "" {
				// 空名稱應該返回 ErrFolderNameEmpty
				return ValidateFolderName(name) == ErrFolderNameEmpty
			}

			// 檢查是否包含非法字元
			hasInvalidChar := false
			for _, r := range name {
				if invalidChars[r] {
					hasInvalidChar = true
					break
				}
			}

			err := ValidateFolderName(name)
			if hasInvalidChar {
				return err == ErrFolderNameInvalid
			}
			return err == nil
		}

		if err := quick.Check(property, &quick.Config{MaxCount: 200}); err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})
}

// TestProperty_FolderCRUDConsistency 測試文件夾 CRUD 操作的一致性
func TestProperty_FolderCRUDConsistency(t *testing.T) {
	// 屬性 1：建立後立即讀取應該返回相同資料
	t.Run("CreateThenReadConsistency", func(t *testing.T) {
		property := func(seed uint32) bool {
			// 清理環境
			path, _ := GetFoldersPath()
			os.Remove(path)
			defer os.Remove(path)

			// 生成有效的文件夾名稱
			name := generateValidFolderName(seed)
			if name == "" {
				return true // 跳過無效名稱
			}

			// 建立文件夾
			created, err := CreateFolder(name)
			if err != nil {
				return false
			}

			// 讀取並驗證
			folders, err := ListFolders()
			if err != nil {
				return false
			}

			// 應該找到剛建立的文件夾
			for _, f := range folders {
				if f.ID == created.ID {
					return f.Name == name
				}
			}
			return false
		}

		if err := quick.Check(property, &quick.Config{MaxCount: 50}); err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// 屬性 2：刪除後讀取應該返回 ErrFolderNotFound
	t.Run("DeleteThenReadNotFound", func(t *testing.T) {
		property := func(seed uint32) bool {
			// 清理環境
			path, _ := GetFoldersPath()
			os.Remove(path)
			defer os.Remove(path)

			// 生成有效的文件夾名稱
			name := generateValidFolderName(seed)
			if name == "" {
				return true
			}

			// 建立文件夾
			created, err := CreateFolder(name)
			if err != nil {
				return false
			}

			// 刪除文件夾
			_, err = DeleteFolder(created.ID, false)
			if err != nil {
				return false
			}

			// 嘗試重新命名已刪除的文件夾應該失敗
			err = RenameFolder(created.ID, "new-name")
			return err == ErrFolderNotFound
		}

		if err := quick.Check(property, &quick.Config{MaxCount: 50}); err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})
}

// TestProperty_AssignmentConsistency 測試快照分配的一致性
func TestProperty_AssignmentConsistency(t *testing.T) {
	// 屬性 1：分配後 GetSnapshotFolderId 應該返回正確的 folderId
	t.Run("AssignThenGetConsistency", func(t *testing.T) {
		property := func(seed uint32, snapshotSeed uint32) bool {
			// 清理環境
			path, _ := GetFoldersPath()
			os.Remove(path)
			defer os.Remove(path)

			// 生成有效的文件夾名稱和快照名稱
			folderName := generateValidFolderName(seed)
			snapshotName := generateValidSnapshotName(snapshotSeed)
			if folderName == "" || snapshotName == "" {
				return true
			}

			// 建立文件夾
			folder, err := CreateFolder(folderName)
			if err != nil {
				return false
			}

			// 分配快照
			err = AssignSnapshotToFolder(snapshotName, folder.ID)
			if err != nil {
				return false
			}

			// 驗證分配
			folderId, err := GetSnapshotFolderId(snapshotName)
			if err != nil {
				return false
			}

			return folderId == folder.ID
		}

		if err := quick.Check(property, &quick.Config{MaxCount: 50}); err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// 屬性 2：取消分配後 GetSnapshotFolderId 應該返回空字串
	t.Run("UnassignThenGetEmpty", func(t *testing.T) {
		property := func(seed uint32, snapshotSeed uint32) bool {
			// 清理環境
			path, _ := GetFoldersPath()
			os.Remove(path)
			defer os.Remove(path)

			// 生成有效的文件夾名稱和快照名稱
			folderName := generateValidFolderName(seed)
			snapshotName := generateValidSnapshotName(snapshotSeed)
			if folderName == "" || snapshotName == "" {
				return true
			}

			// 建立文件夾
			folder, err := CreateFolder(folderName)
			if err != nil {
				return false
			}

			// 分配快照
			err = AssignSnapshotToFolder(snapshotName, folder.ID)
			if err != nil {
				return false
			}

			// 取消分配
			err = UnassignSnapshot(snapshotName)
			if err != nil {
				return false
			}

			// 驗證已取消分配
			folderId, err := GetSnapshotFolderId(snapshotName)
			if err != nil {
				return false
			}

			return folderId == ""
		}

		if err := quick.Check(property, &quick.Config{MaxCount: 50}); err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})
}

// TestProperty_OrphanCleanup 測試孤兒記錄清理的屬性
func TestProperty_OrphanCleanup(t *testing.T) {
	// 屬性 1：清理後不應該存在任何孤兒記錄
	t.Run("NoOrphansAfterCleanup", func(t *testing.T) {
		property := func(seed uint32) bool {
			// 清理環境
			path, _ := GetFoldersPath()
			os.Remove(path)
			defer os.Remove(path)

			// 建立文件夾
			folder, err := CreateFolder(generateValidFolderName(seed))
			if err != nil {
				return true // 跳過無效情況
			}

			// 建立一些 assignments（模擬孤兒和有效記錄）
			r := rand.New(rand.NewSource(int64(seed)))
			numAssignments := r.Intn(10) + 1
			existingSnapshots := make(map[string]bool)

			data, _ := LoadFolders()
			for i := 0; i < numAssignments; i++ {
				snapshotName := generateValidSnapshotName(uint32(r.Int31()))
				if snapshotName == "" {
					continue
				}
				data.Assignments[snapshotName] = folder.ID
				// 隨機決定快照是否存在
				if r.Float32() > 0.5 {
					existingSnapshots[snapshotName] = true
				}
			}
			SaveFolders(data)

			// 執行清理
			_, err = CleanupOrphanAssignments(func(name string) bool {
				return existingSnapshots[name]
			})
			if err != nil {
				return false
			}

			// 驗證：所有剩餘的 assignments 都應該是存在的快照
			data, _ = LoadFolders()
			for snapshotName := range data.Assignments {
				if !existingSnapshots[snapshotName] {
					return false // 發現孤兒記錄
				}
			}

			return true
		}

		if err := quick.Check(property, &quick.Config{MaxCount: 50}); err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})

	// 屬性 2：清理不應該影響有效的 assignments
	t.Run("ValidAssignmentsPreserved", func(t *testing.T) {
		property := func(seed uint32) bool {
			// 清理環境
			path, _ := GetFoldersPath()
			os.Remove(path)
			defer os.Remove(path)

			// 建立文件夾
			folder, err := CreateFolder(generateValidFolderName(seed))
			if err != nil {
				return true
			}

			// 建立一些 assignments
			r := rand.New(rand.NewSource(int64(seed)))
			existingSnapshots := make(map[string]bool)
			expectedAssignments := make(map[string]string)

			data, _ := LoadFolders()
			for i := 0; i < 5; i++ {
				snapshotName := generateValidSnapshotName(uint32(r.Int31()))
				if snapshotName == "" {
					continue
				}
				data.Assignments[snapshotName] = folder.ID
				// 標記為存在的快照
				existingSnapshots[snapshotName] = true
				expectedAssignments[snapshotName] = folder.ID
			}

			// 添加一些孤兒記錄
			for i := 0; i < 3; i++ {
				orphanName := "orphan-" + generateValidSnapshotName(uint32(r.Int31()))
				if orphanName == "orphan-" {
					continue
				}
				data.Assignments[orphanName] = folder.ID
				// 不標記為存在
			}
			SaveFolders(data)

			// 執行清理
			_, err = CleanupOrphanAssignments(func(name string) bool {
				return existingSnapshots[name]
			})
			if err != nil {
				return false
			}

			// 驗證：所有有效的 assignments 都應該保留
			data, _ = LoadFolders()
			for snapshotName, expectedFolderId := range expectedAssignments {
				if data.Assignments[snapshotName] != expectedFolderId {
					return false
				}
			}

			return true
		}

		if err := quick.Check(property, &quick.Config{MaxCount: 50}); err != nil {
			t.Errorf("Property failed: %v", err)
		}
	})
}

// ==================== Task 13: 並發安全測試 ====================

// TestConcurrentFolderOperations 測試並發文件夾操作
func TestConcurrentFolderOperations(t *testing.T) {
	// 清理環境
	path, _ := GetFoldersPath()
	os.Remove(path)
	defer os.Remove(path)

	const numGoroutines = 10
	const operationsPerGoroutine = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine*3)

	// 並發建立文件夾
	t.Run("ConcurrentCreate", func(t *testing.T) {
		// 重置環境
		os.Remove(path)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					name := "folder-" + string(rune('A'+id)) + "-" + string(rune('0'+j))
					_, err := CreateFolder(name)
					if err != nil && err != ErrFolderExists {
						errors <- err
					}
				}
			}(i)
		}
		wg.Wait()

		// 檢查錯誤
		close(errors)
		for err := range errors {
			t.Errorf("Concurrent create error: %v", err)
		}

		// 驗證資料一致性
		folders, err := ListFolders()
		if err != nil {
			t.Fatalf("ListFolders failed: %v", err)
		}

		// 應該有 numGoroutines * operationsPerGoroutine 個文件夾
		expectedCount := numGoroutines * operationsPerGoroutine
		if len(folders) != expectedCount {
			t.Errorf("Expected %d folders, got %d", expectedCount, len(folders))
		}
	})

	// 並發分配快照
	t.Run("ConcurrentAssign", func(t *testing.T) {
		// 重置環境
		os.Remove(path)
		errors = make(chan error, numGoroutines*operationsPerGoroutine)

		// 先建立一個文件夾
		folder, err := CreateFolder("shared-folder")
		if err != nil {
			t.Fatalf("CreateFolder failed: %v", err)
		}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					snapshotName := "snapshot-" + string(rune('A'+id)) + "-" + string(rune('0'+j))
					err := AssignSnapshotToFolder(snapshotName, folder.ID)
					if err != nil {
						errors <- err
					}
				}
			}(i)
		}
		wg.Wait()

		// 檢查錯誤
		close(errors)
		for err := range errors {
			t.Errorf("Concurrent assign error: %v", err)
		}

		// 驗證資料一致性
		data, err := LoadFolders()
		if err != nil {
			t.Fatalf("LoadFolders failed: %v", err)
		}

		expectedCount := numGoroutines * operationsPerGoroutine
		if len(data.Assignments) != expectedCount {
			t.Errorf("Expected %d assignments, got %d", expectedCount, len(data.Assignments))
		}
	})

	// 並發混合操作（建立、分配、取消分配）
	t.Run("ConcurrentMixedOperations", func(t *testing.T) {
		// 重置環境
		os.Remove(path)
		errors = make(chan error, numGoroutines*operationsPerGoroutine*3)

		// 先建立一些文件夾
		var folderIDs []string
		for i := 0; i < 3; i++ {
			folder, err := CreateFolder("base-folder-" + string(rune('0'+i)))
			if err != nil {
				t.Fatalf("CreateFolder failed: %v", err)
			}
			folderIDs = append(folderIDs, folder.ID)
		}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					// 隨機選擇操作
					switch j % 3 {
					case 0:
						// 建立文件夾
						name := "mixed-folder-" + string(rune('A'+id)) + "-" + string(rune('0'+j))
						_, err := CreateFolder(name)
						if err != nil && err != ErrFolderExists {
							errors <- err
						}
					case 1:
						// 分配快照
						snapshotName := "mixed-snapshot-" + string(rune('A'+id)) + "-" + string(rune('0'+j))
						folderId := folderIDs[id%len(folderIDs)]
						err := AssignSnapshotToFolder(snapshotName, folderId)
						if err != nil {
							errors <- err
						}
					case 2:
						// 取消分配
						snapshotName := "mixed-snapshot-" + string(rune('A'+id)) + "-" + string(rune('0'+(j-1)))
						err := UnassignSnapshot(snapshotName)
						if err != nil {
							errors <- err
						}
					}
				}
			}(i)
		}
		wg.Wait()

		// 檢查錯誤
		close(errors)
		for err := range errors {
			t.Errorf("Concurrent mixed operation error: %v", err)
		}

		// 驗證資料完整性（不應該有 panic 或資料損壞）
		data, err := LoadFolders()
		if err != nil {
			t.Fatalf("LoadFolders failed: %v", err)
		}

		// 驗證 JSON 結構完整
		if data.Folders == nil {
			t.Error("Folders should not be nil")
		}
		if data.Assignments == nil {
			t.Error("Assignments should not be nil")
		}
	})
}

// ==================== 輔助函數 ====================

// generateValidFolderName 生成有效的文件夾名稱
func generateValidFolderName(seed uint32) string {
	r := rand.New(rand.NewSource(int64(seed)))
	length := r.Intn(20) + 1
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_ "

	result := make([]byte, length)
	for i := range result {
		result[i] = validChars[r.Intn(len(validChars))]
	}

	name := strings.TrimSpace(string(result))
	if name == "" {
		return "default-folder"
	}
	return name
}

// generateValidSnapshotName 生成有效的快照名稱
func generateValidSnapshotName(seed uint32) string {
	r := rand.New(rand.NewSource(int64(seed)))
	length := r.Intn(15) + 5
	validChars := "abcdefghijklmnopqrstuvwxyz0123456789-_"

	result := make([]byte, length)
	for i := range result {
		result[i] = validChars[r.Intn(len(validChars))]
	}

	return string(result)
}

// isValidFolderNameChar 檢查字元是否為有效的文件夾名稱字元
func isValidFolderNameChar(r rune) bool {
	invalidChars := map[rune]bool{
		'/': true, '\\': true, ':': true, '*': true,
		'?': true, '"': true, '<': true, '>': true, '|': true,
	}
	return !invalidChars[r] && (unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) || r == '-' || r == '_')
}
