//go:build windows

package kiropath

import (
	"testing"
)

// TestParseRegistryOutput 測試解析 reg query 輸出
func TestParseRegistryOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantPath string
		wantErr  bool
	}{
		{
			name: "InstallLocation found",
			output: `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Uninstall\{12345678-1234-1234-1234-123456789012}_is1
    DisplayName    REG_SZ    Kiro
    InstallLocation    REG_SZ    C:\Users\Test\AppData\Local\Programs\Kiro
    Publisher    REG_SZ    Amazon

`,
			wantPath: `C:\Users\Test\AppData\Local\Programs\Kiro`,
			wantErr:  false,
		},
		{
			name: "DisplayIcon found (exe path)",
			output: `HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Uninstall\{12345678-1234-1234-1234-123456789012}_is1
    DisplayName    REG_SZ    Kiro
    DisplayIcon    REG_SZ    C:\Program Files\Kiro\Kiro.exe,0
    Publisher    REG_SZ    Amazon

`,
			wantPath: `C:\Program Files\Kiro`,
			wantErr:  false,
		},
		{
			name: "DisplayIcon without icon index",
			output: `HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Uninstall\Kiro
    DisplayName    REG_SZ    Kiro
    DisplayIcon    REG_SZ    D:\Apps\Kiro\Kiro.exe
    Publisher    REG_SZ    Amazon

`,
			wantPath: `D:\Apps\Kiro`,
			wantErr:  false,
		},
		{
			name:     "No Kiro entry found",
			output:   `ERROR: The system was unable to find the specified registry key or value.`,
			wantPath: "",
			wantErr:  true,
		},
		{
			name:     "Empty output",
			output:   "",
			wantPath: "",
			wantErr:  true,
		},
		{
			name: "Multiple entries - first valid wins",
			output: `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Uninstall\Kiro
    DisplayName    REG_SZ    Kiro
    InstallLocation    REG_SZ    C:\Users\Test\AppData\Local\Programs\Kiro

HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Uninstall\Kiro
    DisplayName    REG_SZ    Kiro
    InstallLocation    REG_SZ    C:\Program Files\Kiro

`,
			wantPath: `C:\Users\Test\AppData\Local\Programs\Kiro`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, err := parseRegistryOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRegistryOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPath != tt.wantPath {
				t.Errorf("parseRegistryOutput() = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

// TestExtractInstallPath 測試從 DisplayIcon 提取目錄
func TestExtractInstallPath(t *testing.T) {
	tests := []struct {
		name        string
		displayIcon string
		want        string
	}{
		{
			name:        "exe path with icon index",
			displayIcon: `C:\Program Files\Kiro\Kiro.exe,0`,
			want:        `C:\Program Files\Kiro`,
		},
		{
			name:        "exe path without icon index",
			displayIcon: `C:\Users\Test\AppData\Local\Programs\Kiro\Kiro.exe`,
			want:        `C:\Users\Test\AppData\Local\Programs\Kiro`,
		},
		{
			name:        "path with spaces",
			displayIcon: `D:\My Programs\Kiro IDE\Kiro.exe,1`,
			want:        `D:\My Programs\Kiro IDE`,
		},
		{
			name:        "quoted path",
			displayIcon: `"C:\Program Files\Kiro\Kiro.exe"`,
			want:        `C:\Program Files\Kiro`,
		},
		{
			name:        "empty string",
			displayIcon: "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInstallPath(tt.displayIcon)
			if got != tt.want {
				t.Errorf("extractInstallPath(%q) = %q, want %q", tt.displayIcon, got, tt.want)
			}
		})
	}
}

// TestGetWindowsRegistryPath_Integration 整合測試（實際查詢 Registry）
// 這個測試會實際執行 reg query，結果取決於系統是否安裝了 Kiro
func TestGetWindowsRegistryPath_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	path, err := getWindowsRegistryPath()
	if err != nil {
		// 如果系統沒有安裝 Kiro，這是預期的錯誤
		t.Logf("getWindowsRegistryPath() returned error (expected if Kiro not installed): %v", err)
		return
	}

	// 如果找到路徑，驗證它是有效的目錄
	t.Logf("Found Kiro install path from registry: %s", path)
	if path == "" {
		t.Error("getWindowsRegistryPath() returned empty path without error")
	}
}
