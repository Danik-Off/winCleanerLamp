//go:build linux

package cleaner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeTrashHome подменяет каталог данных пользователя, чтобы тесты не
// трогали настоящую Корзину.
func fakeTrashHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	return home
}

func TestMoveToTrash_MovesFileAndWritesInfo(t *testing.T) {
	fakeTrashHome(t)

	dir := t.TempDir()
	file := filepath.Join(dir, "junk.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := moveToTrash(file, false); err != nil {
		t.Fatalf("moveToTrash: %v", err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("файл должен исчезнуть из исходного расположения")
	}

	trashed := filepath.Join(trashDir(), "files", "junk.txt")
	if _, err := os.Stat(trashed); err != nil {
		t.Fatalf("файл не появился в Корзине: %v", err)
	}
	info, err := os.ReadFile(filepath.Join(trashDir(), "info", "junk.txt.trashinfo"))
	if err != nil {
		t.Fatalf("нет .trashinfo: %v", err)
	}
	text := string(info)
	if !strings.HasPrefix(text, "[Trash Info]") {
		t.Errorf(".trashinfo без заголовка секции: %q", text)
	}
	if !strings.Contains(text, "Path=") || !strings.Contains(text, "DeletionDate=") {
		t.Errorf(".trashinfo без обязательных полей: %q", text)
	}
}

func TestMoveToTrash_UniqueNameOnCollision(t *testing.T) {
	fakeTrashHome(t)

	dir := t.TempDir()
	for i := 0; i < 2; i++ {
		file := filepath.Join(dir, "same.txt")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := moveToTrash(file, false); err != nil {
			t.Fatalf("moveToTrash #%d: %v", i, err)
		}
	}

	entries, err := os.ReadDir(filepath.Join(trashDir(), "files"))
	if err != nil {
		t.Fatalf("чтение Корзины: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ожидалось 2 файла в Корзине, получено %d", len(entries))
	}
}

func TestEscapeTrashPath(t *testing.T) {
	got := escapeTrashPath("/home/me/Мои файлы/a b.txt")
	if strings.Contains(got, " ") {
		t.Errorf("пробелы должны быть закодированы: %q", got)
	}
	if !strings.HasPrefix(got, "/home/me/") {
		t.Errorf("разделители каталогов должны остаться как есть: %q", got)
	}
}

func TestProcessTrash_CountsAndClears(t *testing.T) {
	fakeTrashHome(t)

	dir := t.TempDir()
	file := filepath.Join(dir, "junk.txt")
	if err := os.WriteFile(file, []byte("0123456789"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := moveToTrash(file, false); err != nil {
		t.Fatalf("moveToTrash: %v", err)
	}

	target := Target{ID: "trash", Special: SpecialTrash}
	scan := Process(target, Options{DryRun: true})
	// Учитывается и сам файл, и его .trashinfo — оба освободятся.
	if scan.Bytes < 10 || scan.Files < 2 {
		t.Errorf("scan Корзины = %d байт в %d файлах, ожидались файл и его .trashinfo", scan.Bytes, scan.Files)
	}

	clean := Process(target, Options{})
	if len(clean.Errors) != 0 {
		t.Errorf("очистка Корзины вернула ошибки: %v", clean.Errors)
	}
	entries, err := os.ReadDir(filepath.Join(trashDir(), "files"))
	if err != nil {
		t.Fatalf("чтение Корзины: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Корзина должна опустеть, осталось %d записей", len(entries))
	}
}
