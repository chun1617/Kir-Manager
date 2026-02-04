package backup

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// FoldersFileName 文件夾資料檔案名稱
	FoldersFileName = "folders.json"
)

var (
	// ErrFolderNotFound 文件夾不存在
	ErrFolderNotFound = errors.New("folder not found")
	// ErrFolderExists 文件夾已存在
	ErrFolderExists = errors.New("folder already exists")
	// ErrFolderNameEmpty 文件夾名稱為空
	ErrFolderNameEmpty = errors.New("folder name cannot be empty")
	// ErrFolderNameInvalid 文件夾名稱包含非法字元
	ErrFolderNameInvalid = errors.New("folder name contains invalid characters")
	// ErrFolderHasActiveSnapshot 文件夾包含活躍快照，無法刪除
	ErrFolderHasActiveSnapshot = errors.New("cannot delete folder containing active snapshot")
)

// Folder 代表一個文件夾
type Folder struct {
	ID        string `json:"id"`        // 唯一識別碼
	Name      string `json:"name"`      // 文件夾名稱
	CreatedAt string `json:"createdAt"` // 建立時間 (RFC3339 格式)
	Order     int    `json:"order"`     // 排序順序
}

// FoldersData 代表 folders.json 的完整結構
type FoldersData struct {
	Folders     []Folder          `json:"folders"`     // 文件夾列表
	Assignments map[string]string `json:"assignments"` // snapshotName -> folderId 映射
}


// invalidFolderNameChars 定義文件夾名稱中不允許的字元
var invalidFolderNameChars = []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}

// ValidateFolderName 驗證文件夾名稱
// 規則：
// - 不可為空
// - 不可包含非法字元：/ \ : * ? " < > |
func ValidateFolderName(name string) error {
	if name == "" {
		return ErrFolderNameEmpty
	}

	for _, char := range invalidFolderNameChars {
		if strings.Contains(name, char) {
			return ErrFolderNameInvalid
		}
	}

	return nil
}


// foldersMutex 保護 folders.json 的並發讀寫
var foldersMutex sync.Mutex

// GetFoldersPath 取得 folders.json 的路徑
func GetFoldersPath() (string, error) {
	rootPath, err := GetBackupRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootPath, FoldersFileName), nil
}

// LoadFolders 載入文件夾資料
// 如果檔案不存在，返回空的 FoldersData
func LoadFolders() (*FoldersData, error) {
	foldersMutex.Lock()
	defer foldersMutex.Unlock()

	return loadFoldersInternal()
}

// loadFoldersInternal 內部載入函數（不加鎖，供已持有鎖的函數使用）
func loadFoldersInternal() (*FoldersData, error) {
	path, err := GetFoldersPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &FoldersData{
				Folders:     []Folder{},
				Assignments: make(map[string]string),
			}, nil
		}
		return nil, err
	}

	var foldersData FoldersData
	if err := json.Unmarshal(data, &foldersData); err != nil {
		return nil, err
	}

	// 確保 Assignments 不為 nil
	if foldersData.Assignments == nil {
		foldersData.Assignments = make(map[string]string)
	}

	return &foldersData, nil
}

// SaveFolders 儲存文件夾資料
func SaveFolders(data *FoldersData) error {
	if data == nil {
		return nil
	}

	foldersMutex.Lock()
	defer foldersMutex.Unlock()

	return saveFoldersInternal(data)
}

// saveFoldersInternal 內部儲存函數（不加鎖，供已持有鎖的函數使用）
func saveFoldersInternal(data *FoldersData) error {
	path, err := GetFoldersPath()
	if err != nil {
		return err
	}

	// 確保目錄存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, jsonData, 0644)
}

// FolderWithCount 文件夾及其快照數量
type FolderWithCount struct {
	Folder
	SnapshotCount int `json:"snapshotCount"`
}

// CreateFolder 建立新文件夾
// 返回建立的文件夾，或錯誤（名稱無效、已存在）
func CreateFolder(name string) (*Folder, error) {
	// 驗證名稱
	if err := ValidateFolderName(name); err != nil {
		return nil, err
	}

	foldersMutex.Lock()
	defer foldersMutex.Unlock()

	// 載入現有資料
	data, err := loadFoldersInternal()
	if err != nil {
		return nil, err
	}

	// 檢查名稱是否已存在
	for _, f := range data.Folders {
		if f.Name == name {
			return nil, ErrFolderExists
		}
	}

	// 建立新文件夾
	folder := Folder{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: time.Now().Format(time.RFC3339),
		Order:     len(data.Folders),
	}

	data.Folders = append(data.Folders, folder)

	// 儲存
	if err := saveFoldersInternal(data); err != nil {
		return nil, err
	}

	return &folder, nil
}

// RenameFolder 重新命名文件夾
func RenameFolder(id, newName string) error {
	if err := ValidateFolderName(newName); err != nil {
		return err
	}

	foldersMutex.Lock()
	defer foldersMutex.Unlock()

	data, err := loadFoldersInternal()
	if err != nil {
		return err
	}

	// 檢查新名稱是否已存在（排除自己）
	for _, f := range data.Folders {
		if f.Name == newName && f.ID != id {
			return ErrFolderExists
		}
	}

	// 找到並更新
	found := false
	for i := range data.Folders {
		if data.Folders[i].ID == id {
			data.Folders[i].Name = newName
			found = true
			break
		}
	}

	if !found {
		return ErrFolderNotFound
	}

	return saveFoldersInternal(data)
}

// DeleteFolder 刪除文件夾
// deleteSnapshots: true 表示一併刪除快照，false 表示移到未分類
// 返回被移到未分類的快照名稱列表
func DeleteFolder(id string, deleteSnapshots bool) ([]string, error) {
	foldersMutex.Lock()
	defer foldersMutex.Unlock()

	data, err := loadFoldersInternal()
	if err != nil {
		return nil, err
	}

	// 找到文件夾
	folderIndex := -1
	for i, f := range data.Folders {
		if f.ID == id {
			folderIndex = i
			break
		}
	}

	if folderIndex == -1 {
		return nil, ErrFolderNotFound
	}

	// 收集該文件夾的快照
	var snapshotsInFolder []string
	for snapshotName, folderId := range data.Assignments {
		if folderId == id {
			snapshotsInFolder = append(snapshotsInFolder, snapshotName)
		}
	}

	// 處理快照
	if deleteSnapshots {
		// 刪除快照（從 assignments 移除）
		// 注意：實際刪除快照的邏輯需要在外部處理，這裡只處理 assignments
		for _, name := range snapshotsInFolder {
			delete(data.Assignments, name)
		}
	} else {
		// 移到未分類（從 assignments 移除）
		for _, name := range snapshotsInFolder {
			delete(data.Assignments, name)
		}
	}

	// 刪除文件夾
	data.Folders = append(data.Folders[:folderIndex], data.Folders[folderIndex+1:]...)

	// 儲存
	if err := saveFoldersInternal(data); err != nil {
		return nil, err
	}

	return snapshotsInFolder, nil
}

// ListFolders 列出所有文件夾及其快照數量
func ListFolders() ([]FolderWithCount, error) {
	data, err := LoadFolders()
	if err != nil {
		return nil, err
	}

	// 計算每個文件夾的快照數量
	countMap := make(map[string]int)
	for _, folderId := range data.Assignments {
		countMap[folderId]++
	}

	result := make([]FolderWithCount, len(data.Folders))
	for i, f := range data.Folders {
		result[i] = FolderWithCount{
			Folder:        f,
			SnapshotCount: countMap[f.ID],
		}
	}

	return result, nil
}


// ==================== Task 3.1: 快照歸屬管理 ====================

// AssignSnapshotToFolder 將快照分配到指定文件夾
func AssignSnapshotToFolder(snapshotName, folderId string) error {
	foldersMutex.Lock()
	defer foldersMutex.Unlock()

	data, err := loadFoldersInternal()
	if err != nil {
		return err
	}

	// 檢查文件夾是否存在
	folderExists := false
	for _, f := range data.Folders {
		if f.ID == folderId {
			folderExists = true
			break
		}
	}

	if !folderExists {
		return ErrFolderNotFound
	}

	// 更新 assignment
	data.Assignments[snapshotName] = folderId

	return saveFoldersInternal(data)
}

// UnassignSnapshot 將快照移至未分類（從 assignments 移除）
func UnassignSnapshot(snapshotName string) error {
	foldersMutex.Lock()
	defer foldersMutex.Unlock()

	data, err := loadFoldersInternal()
	if err != nil {
		return err
	}

	// 移除 assignment（如果存在）
	delete(data.Assignments, snapshotName)

	return saveFoldersInternal(data)
}

// GetSnapshotFolderId 取得快照所屬的文件夾 ID
// 如果快照未分配到任何文件夾，返回空字串
func GetSnapshotFolderId(snapshotName string) (string, error) {
	data, err := LoadFolders()
	if err != nil {
		return "", err
	}

	return data.Assignments[snapshotName], nil
}

// ==================== Task 3.2: 孤兒記錄清理 ====================

// SnapshotExistsChecker 檢查快照是否存在的函數類型
type SnapshotExistsChecker func(snapshotName string) bool

// CleanupOrphanAssignments 清理孤兒記錄（快照不存在但有 assignment）
// checker: 檢查快照是否存在的函數
// 返回被清理的快照名稱列表
func CleanupOrphanAssignments(checker SnapshotExistsChecker) ([]string, error) {
	foldersMutex.Lock()
	defer foldersMutex.Unlock()

	data, err := loadFoldersInternal()
	if err != nil {
		return nil, err
	}

	var cleaned []string
	for snapshotName := range data.Assignments {
		if !checker(snapshotName) {
			cleaned = append(cleaned, snapshotName)
			delete(data.Assignments, snapshotName)
		}
	}

	if len(cleaned) > 0 {
		if err := saveFoldersInternal(data); err != nil {
			return nil, err
		}
	}

	return cleaned, nil
}
