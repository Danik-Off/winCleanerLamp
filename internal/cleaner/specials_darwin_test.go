//go:build darwin

package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHumanSize(t *testing.T) {
	cases := map[string]int64{
		"512B":  512,
		"1KB":   1024,
		"1.5MB": 1024 * 1024 * 3 / 2,
		"2GB":   2 * 1024 * 1024 * 1024,
		"чушь":  0,
	}
	for in, want := range cases {
		if got := parseHumanSize(in); got != want {
			t.Errorf("parseHumanSize(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseBrewFreedBytes(t *testing.T) {
	gb := float64(1024 * 1024 * 1024)
	cases := []struct {
		out  string
		want int64
	}{
		{"==> This operation has freed approximately 1.2GB of disk space.\n", int64(1.2 * gb)},
		{"==> This operation would free approximately 37MB of disk space.\n", 37 * 1024 * 1024},
		{"Nothing to do\n", 0},
	}
	for _, c := range cases {
		if got := parseBrewFreedBytes(c.out); got != c.want {
			t.Errorf("parseBrewFreedBytes(%q) = %d, want %d", c.out, got, c.want)
		}
	}
}

func TestMoveToTrash_MovesFileToHomeTrash(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

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
	if _, err := os.Stat(filepath.Join(home, ".Trash", "junk.txt")); err != nil {
		t.Fatalf("файл не появился в ~/.Trash: %v", err)
	}
}

func TestUniqueTrashName_AvoidsCollision(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "same.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	got := uniqueTrashName(dir, "same.txt")
	if got == "same.txt" {
		t.Error("имя должно быть изменено при коллизии")
	}
	if filepath.Ext(got) != ".txt" {
		t.Errorf("расширение должно сохраняться, получено %q", got)
	}
}

func TestProcessTrash_CountsAndClears(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

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
	if scan.Bytes != 10 || scan.Files != 1 {
		t.Errorf("scan Корзины = %d байт в %d файлах, ожидалось 10 байт в 1 файле", scan.Bytes, scan.Files)
	}

	clean := Process(target, Options{})
	if len(clean.Errors) != 0 {
		t.Errorf("очистка Корзины вернула ошибки: %v", clean.Errors)
	}
	entries, err := os.ReadDir(filepath.Join(home, ".Trash"))
	if err != nil {
		t.Fatalf("чтение Корзины: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Корзина должна опустеть, осталось %d записей", len(entries))
	}
}
