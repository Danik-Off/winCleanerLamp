//go:build linux

package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanBrokenShortcuts_DetectsMissingExec(t *testing.T) {
	dir := t.TempDir()
	writeDesktopFile(t, dir, "broken.desktop", "[Desktop Entry]\nType=Application\nName=Broken\nExec=/nonexistent/path/to/app\n")
	writeDesktopFile(t, dir, "ok.desktop", "[Desktop Entry]\nType=Application\nName=Ok\nExec=/bin/sh\n")
	writeDesktopFile(t, dir, "link.desktop", "[Desktop Entry]\nType=Link\nName=Site\nURL=https://example.com\n")

	result, err := ScanBrokenShortcuts([]string{dir})
	if err != nil {
		t.Fatalf("ScanBrokenShortcuts: %v", err)
	}
	if result.Scanned != 3 {
		t.Errorf("Scanned = %d, ожидалось 3 файла .desktop", result.Scanned)
	}
	if len(result.Broken) != 1 {
		t.Fatalf("ожидался ровно один битый ярлык, получено %d: %+v", len(result.Broken), result.Broken)
	}
	if filepath.Base(result.Broken[0].Path) != "broken.desktop" {
		t.Errorf("битым признан не тот ярлык: %s", result.Broken[0].Path)
	}
}

func TestScanBrokenShortcuts_CommandInPathIsNotBroken(t *testing.T) {
	dir := t.TempDir()
	writeDesktopFile(t, dir, "sh.desktop", "[Desktop Entry]\nType=Application\nName=Shell\nExec=sh -c true\n")

	result, err := ScanBrokenShortcuts([]string{dir})
	if err != nil {
		t.Fatalf("ScanBrokenShortcuts: %v", err)
	}
	if len(result.Broken) != 0 {
		t.Errorf("команда, найденная в PATH, не должна считаться битой: %+v", result.Broken)
	}
}

func TestScanBrokenShortcuts_MissingRootIgnored(t *testing.T) {
	result, err := ScanBrokenShortcuts([]string{filepath.Join(t.TempDir(), "no-such-dir")})
	if err != nil {
		t.Fatalf("несуществующий корень не должен приводить к ошибке: %v", err)
	}
	if result.Scanned != 0 || len(result.Broken) != 0 {
		t.Errorf("ожидался пустой результат, получено %+v", result)
	}
}

func TestScanRegistryLeftovers_FindsOrphanDesktopFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	appsDir := filepath.Join(home, ".local", "share", "applications")
	writeDesktopFile(t, appsDir, "gone.desktop", "[Desktop Entry]\nType=Application\nName=Gone\nExec=/nonexistent/app\n")
	writeDesktopFile(t, appsDir, "alive.desktop", "[Desktop Entry]\nType=Application\nName=Alive\nExec=/bin/sh\n")

	// В системных каталогах (/usr/share/applications) на машине сборки тоже
	// могут найтись осиротевшие ярлыки, поэтому проверяем не количество, а
	// присутствие нужного и отсутствие рабочего.
	var foundGone, foundAlive bool
	for _, c := range scanRegistryLeftovers(nil, nil) {
		if c.Type != LeftoverRegistry {
			t.Errorf("неверный тип находки: %s", c.Type)
		}
		switch filepath.Base(c.Path) {
		case "gone.desktop":
			foundGone = true
		case "alive.desktop":
			foundAlive = true
		}
	}
	if !foundGone {
		t.Error("ярлык с несуществующим Exec должен попасть в остатки")
	}
	if foundAlive {
		t.Error("ярлык с существующим Exec не должен попадать в остатки")
	}
}

func TestDirHasExecutable(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "bin")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	plain := filepath.Join(sub, "readme.txt")
	if err := os.WriteFile(plain, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if dirHasExecutable(dir) {
		t.Error("каталог без исполняемых файлов не должен считаться установленной программой")
	}

	binary := filepath.Join(sub, "app")
	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if !dirHasExecutable(dir) {
		t.Error("файл с битом x должен распознаваться как исполняемый")
	}
}
