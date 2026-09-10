//go:build darwin

package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsPathSafeToDelete_AllowsOrdinaryPaths(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{
		dir,
		filepath.Join(dir, "sub", "file.tmp"),
		"/Library/Caches/com.example.app",
		"/private/var/log/old.log.1",
	} {
		if ok, reason := IsPathSafeToDelete(p); !ok {
			t.Errorf("IsPathSafeToDelete(%q) = false (%s), ожидалось true", p, reason)
		}
	}
}

func TestIsPathSafeToDelete_RejectsForbiddenPrefixes(t *testing.T) {
	for _, p := range []string{
		"/System/Library/CoreServices",
		"/usr/bin",
		"/Applications/Safari.app",
		"/Library/Frameworks/Python.framework",
		"/Library/Keychains/System.keychain",
		"/private/var/db/dslocal",
		"/private/var/vm/sleepimage",
		"/Volumes/Backup",
	} {
		if ok, _ := IsPathSafeToDelete(p); ok {
			t.Errorf("IsPathSafeToDelete(%q) = true, ожидался отказ", p)
		}
	}
}

func TestIsPathSafeToDelete_RejectsFilesystemRoots(t *testing.T) {
	for _, p := range []string{"/", "/Users", "/Library", "/private", "/var"} {
		if ok, _ := IsPathSafeToDelete(p); ok {
			t.Errorf("IsPathSafeToDelete(%q) = true, ожидался отказ", p)
		}
	}
}

// Домашние каталоги защищены целиком, а вложенные пути в них — нет.
func TestIsPathSafeToDelete_HomeDirRootVsSubpaths(t *testing.T) {
	if ok, _ := IsPathSafeToDelete("/Users/someone"); ok {
		t.Error("домашний каталог пользователя не должен удаляться целиком")
	}
	if ok, reason := IsPathSafeToDelete("/Users/someone/Library/Caches/app"); !ok {
		t.Errorf("вложенный путь в домашнем каталоге должен быть разрешён, получено: %s", reason)
	}
}

func TestIsPathSafeToDelete_RejectsUserDataDirs(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("нет домашнего каталога")
	}
	for _, rel := range []string{"Library/Keychains", "Library/Mobile Documents", "Library/Mail", ".ssh"} {
		p := filepath.Join(home, rel)
		if ok, _ := IsPathSafeToDelete(p); ok {
			t.Errorf("IsPathSafeToDelete(%q) = true, данные пользователя удалять нельзя", p)
		}
	}
}

func TestIsPathSafeToDelete_RejectsEmptyPath(t *testing.T) {
	if ok, _ := IsPathSafeToDelete("   "); ok {
		t.Error("пустой путь должен отклоняться")
	}
}
