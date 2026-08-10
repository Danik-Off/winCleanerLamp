package cleaner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanDuplicates_FindsIdenticalFiles(t *testing.T) {
	dir := t.TempDir()
	content := []byte("одинаковое содержимое для теста дубликатов")

	mustWrite := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
			t.Fatalf("setup %s: %v", name, err)
		}
	}
	mustWrite("a.txt")
	mustWrite("b.txt")
	if err := os.WriteFile(filepath.Join(dir, "unique.txt"), []byte("совсем другое содержимое файла"), 0o644); err != nil {
		t.Fatalf("setup unique.txt: %v", err)
	}

	result, err := ScanDuplicates(DuplicateScanOptions{
		Roots:   []string{dir},
		MinSize: 1,
	})
	if err != nil {
		t.Fatalf("ScanDuplicates: %v", err)
	}

	if len(result.Groups) != 1 {
		t.Fatalf("ожидалась 1 группа дубликатов, получено %d: %+v", len(result.Groups), result.Groups)
	}
	g := result.Groups[0]
	if len(g.Paths) != 2 {
		t.Errorf("в группе должно быть 2 файла, получено %d", len(g.Paths))
	}
	wantWaste := int64(len(content)) * 1 // (2 файла - 1) * размер
	if g.WasteSize != wantWaste {
		t.Errorf("WasteSize = %d, want %d", g.WasteSize, wantWaste)
	}
	if g.RiskFlag != "" {
		t.Errorf("обычные .txt файлы в пользовательской папке не должны иметь RiskFlag, получено %q", g.RiskFlag)
	}
}

func TestScanDuplicates_RespectsMinSize(t *testing.T) {
	dir := t.TempDir()
	content := []byte("x") // 1 байт

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), content, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), content, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	result, err := ScanDuplicates(DuplicateScanOptions{
		Roots:   []string{dir},
		MinSize: 1024, // файлы меньше — не считаем дубликатами
	})
	if err != nil {
		t.Fatalf("ScanDuplicates: %v", err)
	}
	if len(result.Groups) != 0 {
		t.Errorf("файлы меньше MinSize не должны попадать в результат, получено %d групп", len(result.Groups))
	}
}

func TestScanDuplicates_NoFalsePositivesForUniqueFiles(t *testing.T) {
	dir := t.TempDir()
	for i, s := range []string{"первый файл", "второй файл", "третий файл"} {
		name := filepath.Join(dir, "f"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(name, []byte(s), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	result, err := ScanDuplicates(DuplicateScanOptions{Roots: []string{dir}, MinSize: 1})
	if err != nil {
		t.Fatalf("ScanDuplicates: %v", err)
	}
	if len(result.Groups) != 0 {
		t.Errorf("уникальные файлы не должны давать группы дубликатов, получено %d", len(result.Groups))
	}
}

func TestIsSystemDirRoot(t *testing.T) {
	cases := map[string]bool{
		`C:\Program Files`:           true,
		`C:\Program Files\SubApp`:    true,
		`C:\Program Files (x86)`:     true,
		`C:\Windows`:                 true,
		`C:\Windows\System32`:        true,
		`C:\ProgramData`:             true,
		`C:\ProgramData\SomeApp`:     true,
		`C:\Users\Someone\Documents`: false,
		`D:\Games`:                   false,
	}
	for path, want := range cases {
		got := isSystemDirRoot(path)
		if got != want {
			t.Errorf("isSystemDirRoot(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestDuplicateGroupRiskFlag_ExecutableFlagged(t *testing.T) {
	dir := t.TempDir()
	flag := duplicateGroupRiskFlag([]string{
		filepath.Join(dir, "app.exe"),
		filepath.Join(dir, "app_backup.exe"),
	})
	if flag == "" {
		t.Error("исполняемые файлы должны получать RiskFlag")
	}
}

func TestDuplicateGroupRiskFlag_OrdinaryFilesClean(t *testing.T) {
	dir := t.TempDir()
	flag := duplicateGroupRiskFlag([]string{
		filepath.Join(dir, "photo1.jpg"),
		filepath.Join(dir, "photo2.jpg"),
	})
	if flag != "" {
		t.Errorf("обычные файлы в пользовательской папке не должны получать RiskFlag, получено %q", flag)
	}
}

func TestDuplicateGroupRiskFlag_SystemDirFlagged(t *testing.T) {
	flag := duplicateGroupRiskFlag([]string{
		`C:\Program Files\Vendor\shared.dll`,
		`C:\Program Files\Other\shared.dll`,
	})
	if flag == "" {
		t.Error("файлы внутри Program Files должны получать RiskFlag независимо от расширения")
	}
}

func TestScanDuplicates_HashCacheReusesUnchangedFiles(t *testing.T) {
	// Изолируем кэш от реального %USERPROFILE% пользователя.
	fakeHome := t.TempDir()
	t.Setenv("USERPROFILE", fakeHome)

	dir := t.TempDir()
	content := make([]byte, 5000)
	for i := range content {
		content[i] = byte(i % 251)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.bin"), content, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.bin"), content, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	opts := DuplicateScanOptions{Roots: []string{dir}, MinSize: 1}

	first, err := ScanDuplicates(opts)
	if err != nil {
		t.Fatalf("первый скан: %v", err)
	}
	if len(first.Groups) != 1 {
		t.Fatalf("ожидалась 1 группа на первом скане, получено %d", len(first.Groups))
	}
	firstHash := first.Groups[0].Hash

	if _, err := os.Stat(dupHashCachePath()); err != nil {
		t.Fatalf("кэш-файл не создан после скана: %v", err)
	}

	// Второй скан должен переиспользовать кэш и дать тот же результат.
	second, err := ScanDuplicates(opts)
	if err != nil {
		t.Fatalf("второй скан: %v", err)
	}
	if len(second.Groups) != 1 || second.Groups[0].Hash != firstHash {
		t.Fatalf("второй скан (из кэша) дал другой результат: %+v", second.Groups)
	}

	// Изменённый файл (тот же размер, другое содержимое) должен пересчитаться,
	// а не взяться из устаревшего кэша по размеру. Явно меняем mtime, чтобы
	// не зависеть от разрешения таймера файловой системы между записями.
	changed := make([]byte, len(content))
	for i := range changed {
		changed[i] = byte((i + 1) % 251)
	}
	aPath := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(aPath, changed, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	newTime := time.Now().Add(1 * time.Hour)
	if err := os.Chtimes(aPath, newTime, newTime); err != nil {
		t.Fatalf("setup chtimes: %v", err)
	}

	third, err := ScanDuplicates(opts)
	if err != nil {
		t.Fatalf("третий скан: %v", err)
	}
	if len(third.Groups) != 0 {
		t.Fatalf("после изменения файла (тот же размер) дубликаты не должны находиться — кэш должен был инвалидироваться по mtime, получено %+v", third.Groups)
	}
}

func TestSkipDir(t *testing.T) {
	mustSkip := []string{"windows", "program files", "program files (x86)", "node_modules", "$recycle.bin", "windows.old"}
	for _, name := range mustSkip {
		if !skipDir(name) {
			t.Errorf("skipDir(%q) = false, want true", name)
		}
	}
	mustNotSkip := []string{"documents", "photos", "my project"}
	for _, name := range mustNotSkip {
		if skipDir(name) {
			t.Errorf("skipDir(%q) = true, want false", name)
		}
	}
}
