package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsPathSafeToDelete_AllowsOrdinaryPaths(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "junk.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	for _, p := range []string{dir, file} {
		ok, reason := IsPathSafeToDelete(p)
		if !ok {
			t.Errorf("IsPathSafeToDelete(%q) = false (%s), want true", p, reason)
		}
	}
}

func TestIsPathSafeToDelete_RejectsForbiddenPrefixes(t *testing.T) {
	cases := []string{
		`C:\Windows\System32`,
		`C:\Windows\System32\drivers\etc\hosts`,
		`c:\windows\system32\subdir\file.txt`, // регистр не важен
		`C:\Windows\SysWOW64`,
		`C:\Windows\SysWOW64\evil.dll`,
		`C:\Windows\WinSxS`,
		`C:\Windows\WinSxS\some-component`,
		`C:\Windows\Installer`,
		`C:\Windows\Installer\somepackage.msi`,
		`C:\Windows\Servicing`,
		`C:\Windows\Boot`,
		`C:\Program Files`,
		`C:\Program Files\SomeApp\file.exe`,
		`C:\Program Files (x86)`,
		`C:\Program Files (x86)\SomeApp`,
		`C:\ProgramData\Microsoft\Windows\Start Menu`,
		`C:\Users\Default`,
		`C:\Users\Default\NTUSER.DAT`,
		`C:\Users\Public`,
		`C:\Users\All Users`,
		`C:\PerfLogs`,
	}
	for _, p := range cases {
		ok, reason := IsPathSafeToDelete(p)
		if ok {
			t.Errorf("IsPathSafeToDelete(%q) = true, want false (защищённый путь)", p)
		}
		if reason == "" {
			t.Errorf("IsPathSafeToDelete(%q): ожидалась причина отказа", p)
		}
	}
}

func TestIsPathSafeToDelete_AllowsLegitimateWindowsSubpaths(t *testing.T) {
	// Эти пути НЕ должны блокироваться — встроенные категории (targets.go)
	// целенаправленно чистят именно их. Регрессия здесь означает, что
	// windows-temp/prefetch/cbs-logs и т.п. перестанут работать.
	cases := []string{
		`C:\Windows\Temp`,
		`C:\Windows\Temp\somefile.tmp`,
		`C:\Windows\Prefetch`,
		`C:\Windows\Logs\CBS`,
		`C:\Windows\Panther`,
		`C:\Windows\LiveKernelReports`,
		`C:\Windows.old`, // не является подпутём C:\Windows (нет разделителя)
	}
	for _, p := range cases {
		ok, _ := IsPathSafeToDelete(p)
		if !ok {
			t.Errorf("IsPathSafeToDelete(%q) = false, want true (легитимная категория)", p)
		}
	}
}

func TestIsPathSafeToDelete_FontCacheException(t *testing.T) {
	cases := []string{
		`C:\Windows\System32\FNTCACHE.DAT`,
		`c:\windows\system32\fntcache.dat`,
	}
	for _, p := range cases {
		ok, reason := IsPathSafeToDelete(p)
		if !ok {
			t.Errorf("IsPathSafeToDelete(%q) = false (%s), want true (точечное исключение font-cache)", p, reason)
		}
	}
}

func TestIsPathSafeToDelete_RejectsDriveRoot(t *testing.T) {
	// Примечание: "C:" без слеша НЕ тестируем — в Windows это "текущий
	// каталог на диске C:" (см. filepath.Abs), а не корень диска; такое
	// значение никогда не встречается в Target.Paths/--delete-path.
	for _, p := range []string{`C:\`, `D:\`} {
		ok, _ := IsPathSafeToDelete(p)
		if ok {
			t.Errorf("IsPathSafeToDelete(%q) = true, want false (корень диска)", p)
		}
	}
}

func TestIsPathSafeToDelete_RejectsUNCPaths(t *testing.T) {
	for _, p := range []string{`\\server\share`, `\\?\C:\Windows`} {
		ok, _ := IsPathSafeToDelete(p)
		if ok {
			t.Errorf("IsPathSafeToDelete(%q) = true, want false (UNC/расширенный путь)", p)
		}
	}
}

func TestIsPathSafeToDelete_RejectsEmptyPath(t *testing.T) {
	ok, _ := IsPathSafeToDelete("")
	if ok {
		t.Error("IsPathSafeToDelete(\"\") = true, want false")
	}
}

func TestIsPathSafeToDelete_RejectsHomeDirRoot(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("не удалось определить домашнюю папку")
	}
	ok, _ := IsPathSafeToDelete(home)
	if ok {
		t.Errorf("IsPathSafeToDelete(%q) = true, want false (корень профиля пользователя)", home)
	}

	// Но вложенные пути внутри профиля — легитимные цели (AppData и т.п.).
	sub := filepath.Join(home, "AppData", "Local", "Temp")
	ok, _ = IsPathSafeToDelete(sub)
	if !ok {
		t.Errorf("IsPathSafeToDelete(%q) = false, want true (подпапка профиля)", sub)
	}
}

func TestIsPathSafeToDelete_SymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "escape-link")
	target := `C:\Windows\System32`

	if err := os.Symlink(target, link); err != nil {
		t.Skipf("символические ссылки недоступны в этом окружении: %v", err)
	}

	ok, reason := IsPathSafeToDelete(link)
	if ok {
		t.Errorf("IsPathSafeToDelete(%q) = true, want false (симлинк ведёт в защищённую зону: %s)", link, target)
	}
	if reason == "" {
		t.Error("ожидалась причина отказа для симлинка")
	}
}

func TestCheckStaticPathSafety_CaseInsensitive(t *testing.T) {
	ok, _ := checkStaticPathSafety(`C:\PROGRAM FILES\App`)
	if ok {
		t.Error("checkStaticPathSafety должен игнорировать регистр при сравнении с запрещёнными префиксами")
	}
}
