//go:build linux

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
		"/tmp/some-cache-dir",
		"/var/tmp/older-file",
		"/var/log/old.log.1",
	} {
		if ok, reason := IsPathSafeToDelete(p); !ok {
			t.Errorf("IsPathSafeToDelete(%q) = false (%s), ожидалось true", p, reason)
		}
	}
}

func TestIsPathSafeToDelete_RejectsForbiddenPrefixes(t *testing.T) {
	for _, p := range []string{
		"/usr",
		"/usr/lib/systemd",
		"/etc/passwd",
		"/boot/grub",
		"/proc/1",
		"/var/lib/dpkg",
		"/var/spool/cron",
		"/snap/core",
		"/media/usb",
	} {
		if ok, _ := IsPathSafeToDelete(p); ok {
			t.Errorf("IsPathSafeToDelete(%q) = true, ожидался отказ", p)
		}
	}
}

func TestIsPathSafeToDelete_RejectsFilesystemRoots(t *testing.T) {
	for _, p := range []string{"/", "/home", "/root", "/var"} {
		if ok, _ := IsPathSafeToDelete(p); ok {
			t.Errorf("IsPathSafeToDelete(%q) = true, ожидался отказ", p)
		}
	}
}

// Домашние каталоги защищены целиком, а вложенные пути в них — нет: именно
// они и есть цели очистки (~/.cache и т.п.).
func TestIsPathSafeToDelete_HomeDirRootVsSubpaths(t *testing.T) {
	if ok, _ := IsPathSafeToDelete("/home/someone"); ok {
		t.Error("домашний каталог пользователя не должен удаляться целиком")
	}
	if ok, reason := IsPathSafeToDelete("/home/someone/.cache/app"); !ok {
		t.Errorf("вложенный путь в домашнем каталоге должен быть разрешён, получено: %s", reason)
	}
}

func TestIsPathSafeToDelete_RejectsUserKeyDirs(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("нет домашнего каталога")
	}
	for _, rel := range []string{".ssh", ".gnupg"} {
		p := filepath.Join(home, rel)
		if ok, _ := IsPathSafeToDelete(p); ok {
			t.Errorf("IsPathSafeToDelete(%q) = true, ключи пользователя удалять нельзя", p)
		}
	}
}

func TestIsPathSafeToDelete_RejectsEmptyPath(t *testing.T) {
	if ok, _ := IsPathSafeToDelete("   "); ok {
		t.Error("пустой путь должен отклоняться")
	}
}
