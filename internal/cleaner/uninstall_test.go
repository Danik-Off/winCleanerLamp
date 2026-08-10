package cleaner

import (
	"reflect"
	"testing"
)

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantFile string
		wantArgs string
	}{
		{"quoted path with args", `"C:\Program Files\App\uninst.exe" /S`, `C:\Program Files\App\uninst.exe`, "/S"},
		{"quoted path no args", `"C:\Program Files\App\uninst.exe"`, `C:\Program Files\App\uninst.exe`, ""},
		{"unquoted no spaces with args", `MsiExec.exe /X{GUID}`, "MsiExec.exe", "/X{GUID}"},
		{"unquoted no args", `C:\uninst.exe`, `C:\uninst.exe`, ""},
		{"leading/trailing whitespace", `  "C:\App\u.exe" /S  `, `C:\App\u.exe`, "/S"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, args := splitCommandLine(tt.in)
			if file != tt.wantFile {
				t.Errorf("file = %q, want %q", file, tt.wantFile)
			}
			if args != tt.wantArgs {
				t.Errorf("args = %q, want %q", args, tt.wantArgs)
			}
		})
	}
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single flag", "/S", []string{"/S"}},
		{"guid flag", "/X{1234-ABCD}", []string{"/X{1234-ABCD}"}},
		{"multiple flags", "/S /norestart", []string{"/S", "/norestart"}},
		{"quoted arg with space", `/dir "C:\some path"`, []string{"/dir", `C:\some path`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitArgs(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitArgs(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestLaunchUninstaller_RejectsUnknownProgram(t *testing.T) {
	err := LaunchUninstaller("Definitely Not An Installed Program XYZ123")
	if err == nil {
		t.Fatal("ожидалась ошибка для несуществующей программы")
	}
}
