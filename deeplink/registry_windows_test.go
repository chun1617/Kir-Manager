//go:build windows

package deeplink

import (
	"strings"
	"testing"
)

// TestIsDeepLinkSupported_Windows 驗證 Windows 平台返回 true
func TestIsDeepLinkSupported_Windows(t *testing.T) {
	result := IsDeepLinkSupported()
	if !result {
		t.Error("IsDeepLinkSupported() should return true on Windows")
	}
}

// TestParseRegistryOutput 測試 Registry 輸出解析
func TestParseRegistryOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantValue   string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid_default_value",
			output: `HKEY_CURRENT_USER\Software\Classes\kiro
    (Default)    REG_SZ    URL:Kiro Manager Protocol
`,
			wantValue: "URL:Kiro Manager Protocol",
			wantErr:   false,
		},
		{
			name: "valid_command_value",
			output: `HKEY_CURRENT_USER\Software\Classes\kiro\shell\open\command
    (Default)    REG_SZ    "C:\Program Files\Kiro Manager\kiro-manager.exe" "%1"
`,
			wantValue: `"C:\Program Files\Kiro Manager\kiro-manager.exe" "%1"`,
			wantErr:   false,
		},
		{
			name:        "empty_output",
			output:      "",
			wantValue:   "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "error_not_found",
			output:      "ERROR: The system was unable to find the specified registry key or value.",
			wantValue:   "",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "no_default_value",
			output: `HKEY_CURRENT_USER\Software\Classes\kiro
    URL Protocol    REG_SZ    
`,
			wantValue:   "",
			wantErr:     true,
			errContains: "no default value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRegistryDefaultValue(tt.output)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseRegistryDefaultValue() expected error containing %q, got nil", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseRegistryDefaultValue() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("parseRegistryDefaultValue() unexpected error = %v", err)
				return
			}
			if got != tt.wantValue {
				t.Errorf("parseRegistryDefaultValue() = %q, want %q", got, tt.wantValue)
			}
		})
	}
}

// TestBuildRegAddCommand 測試 reg add 命令建構
func TestBuildRegAddCommand(t *testing.T) {
	tests := []struct {
		name     string
		regPath  string
		value    string
		wantArgs []string
	}{
		{
			name:    "scheme_default_value",
			regPath: `HKCU\Software\Classes\kiro`,
			value:   "URL:Kiro Manager Protocol",
			wantArgs: []string{
				"add",
				`HKCU\Software\Classes\kiro`,
				"/ve",
				"/t", "REG_SZ",
				"/d", "URL:Kiro Manager Protocol",
				"/f",
			},
		},
		{
			name:    "url_protocol_value",
			regPath: `HKCU\Software\Classes\kiro`,
			value:   "",
			wantArgs: []string{
				"add",
				`HKCU\Software\Classes\kiro`,
				"/ve",
				"/t", "REG_SZ",
				"/d", "",
				"/f",
			},
		},
		{
			name:    "command_value_with_path",
			regPath: `HKCU\Software\Classes\kiro\shell\open\command`,
			value:   `"C:\Program Files\Kiro Manager\kiro-manager.exe" "%1"`,
			wantArgs: []string{
				"add",
				`HKCU\Software\Classes\kiro\shell\open\command`,
				"/ve",
				"/t", "REG_SZ",
				"/d", `"C:\Program Files\Kiro Manager\kiro-manager.exe" "%1"`,
				"/f",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs := buildRegAddArgs(tt.regPath, tt.value)
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("buildRegAddArgs() returned %d args, want %d", len(gotArgs), len(tt.wantArgs))
				t.Errorf("got: %v", gotArgs)
				t.Errorf("want: %v", tt.wantArgs)
				return
			}
			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("buildRegAddArgs()[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

// TestBuildRegQueryArgs 測試 reg query 命令建構
func TestBuildRegQueryArgs(t *testing.T) {
	tests := []struct {
		name     string
		regPath  string
		wantArgs []string
	}{
		{
			name:    "query_scheme_key",
			regPath: `HKCU\Software\Classes\kiro`,
			wantArgs: []string{
				"query",
				`HKCU\Software\Classes\kiro`,
				"/ve",
			},
		},
		{
			name:    "query_command_key",
			regPath: `HKCU\Software\Classes\kiro\shell\open\command`,
			wantArgs: []string{
				"query",
				`HKCU\Software\Classes\kiro\shell\open\command`,
				"/ve",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs := buildRegQueryArgs(tt.regPath)
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("buildRegQueryArgs() returned %d args, want %d", len(gotArgs), len(tt.wantArgs))
				return
			}
			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("buildRegQueryArgs()[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

// TestBuildCommandValue 測試命令值建構
func TestBuildCommandValue(t *testing.T) {
	tests := []struct {
		name    string
		exePath string
		want    string
	}{
		{
			name:    "simple_path",
			exePath: `C:\kiro-manager.exe`,
			want:    `"C:\kiro-manager.exe" "%1"`,
		},
		{
			name:    "path_with_spaces",
			exePath: `C:\Program Files\Kiro Manager\kiro-manager.exe`,
			want:    `"C:\Program Files\Kiro Manager\kiro-manager.exe" "%1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCommandValue(tt.exePath)
			if got != tt.want {
				t.Errorf("buildCommandValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractExePathFromCommand 測試從命令值提取執行檔路徑
func TestExtractExePathFromCommand(t *testing.T) {
	tests := []struct {
		name        string
		commandVal  string
		wantPath    string
		wantErr     bool
		errContains string
	}{
		{
			name:       "quoted_path_with_arg",
			commandVal: `"C:\Program Files\Kiro Manager\kiro-manager.exe" "%1"`,
			wantPath:   `C:\Program Files\Kiro Manager\kiro-manager.exe`,
			wantErr:    false,
		},
		{
			name:       "simple_quoted_path",
			commandVal: `"C:\kiro-manager.exe" "%1"`,
			wantPath:   `C:\kiro-manager.exe`,
			wantErr:    false,
		},
		{
			name:        "empty_value",
			commandVal:  "",
			wantPath:    "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "invalid_format",
			commandVal:  "invalid",
			wantPath:    "",
			wantErr:     true,
			errContains: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractExePathFromCommand(tt.commandVal)
			if tt.wantErr {
				if err == nil {
					t.Errorf("extractExePathFromCommand() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("extractExePathFromCommand() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("extractExePathFromCommand() unexpected error = %v", err)
				return
			}
			if got != tt.wantPath {
				t.Errorf("extractExePathFromCommand() = %q, want %q", got, tt.wantPath)
			}
		})
	}
}
