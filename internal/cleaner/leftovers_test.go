package cleaner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTokenize(t *testing.T) {
	toks := tokenize("Google Chrome")
	want := map[string]bool{"google": true, "chrome": true, "googlechrome": true}
	got := map[string]bool{}
	for _, tk := range toks {
		got[tk] = true
	}
	for w := range want {
		if !got[w] {
			t.Errorf("tokenize(\"Google Chrome\") не содержит %q, получено %v", w, toks)
		}
	}
}

func TestTokenize_DropsShortWords(t *testing.T) {
	toks := tokenize("a b VLC")
	for _, tk := range toks {
		if tk == "a" || tk == "b" {
			t.Errorf("токены короче 3 символов должны отбрасываться, получено %v", toks)
		}
	}
}

func TestMatchesInstalled(t *testing.T) {
	installed := map[string]bool{"chrome": true, "googlechrome": true}

	if !matchesInstalled("Google Chrome", installed) {
		t.Error("папка 'Google Chrome' должна совпасть с установленным 'chrome'")
	}
	if matchesInstalled("Completely Unrelated Folder", installed) {
		t.Error("не связанная папка не должна совпадать")
	}
}

func TestTypeOrder(t *testing.T) {
	if typeOrder(LeftoverFolder) >= typeOrder(LeftoverEmpty) {
		t.Error("LeftoverFolder должен идти раньше LeftoverEmpty")
	}
	if typeOrder(LeftoverEmpty) >= typeOrder(LeftoverRegistry) {
		t.Error("LeftoverEmpty должен идти раньше LeftoverRegistry")
	}
}

func TestIsDirEffectivelyEmpty(t *testing.T) {
	dir := t.TempDir()
	junk := map[string]bool{"thumbs.db": true, "desktop.ini": true}

	if !isDirEffectivelyEmpty(dir, junk) {
		t.Error("пустая папка должна считаться пустой")
	}

	if err := os.WriteFile(filepath.Join(dir, "desktop.ini"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if !isDirEffectivelyEmpty(dir, junk) {
		t.Error("папка только с desktop.ini должна считаться пустой")
	}

	if err := os.WriteFile(filepath.Join(dir, "real.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if isDirEffectivelyEmpty(dir, junk) {
		t.Error("папка с реальным файлом не должна считаться пустой")
	}
}

func TestIsDirEffectivelyEmpty_NestedEmptySubdirs(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub1", "sub2"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if !isDirEffectivelyEmpty(dir, nil) {
		t.Error("вложенные пустые папки в сумме должны считаться пустыми")
	}

	if err := os.WriteFile(filepath.Join(dir, "sub1", "sub2", "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if isDirEffectivelyEmpty(dir, nil) {
		t.Error("файл во вложенной папке должен делать родителя непустым")
	}
}

func TestDirSizeWithTimeout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("12345"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("1234567890"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	size, files := dirSizeWithTimeout(dir, 2*time.Second)
	if files != 2 {
		t.Errorf("files = %d, want 2", files)
	}
	if size != 15 {
		t.Errorf("size = %d, want 15", size)
	}
}

func TestScanEmptyFolders(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "empty1"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "nonempty"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nonempty", "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	results := scanEmptyFolders([]string{root})
	if len(results) != 1 {
		t.Fatalf("ожидалась 1 пустая папка, получено %d: %+v", len(results), results)
	}
	if filepath.Base(results[0].Path) != "empty1" {
		t.Errorf("неожиданный результат: %+v", results[0])
	}
	if results[0].Type != LeftoverEmpty {
		t.Errorf("Type = %v, want LeftoverEmpty", results[0].Type)
	}
}

func TestScanOrphanCachePaths(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "f.dat"), []byte("12345"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg := &OrphanConfig{Apps: []OrphanApp{
		{DisplayName: "TestApp", CachePaths: []string{cacheDir}},
	}}

	results := scanOrphanCachePaths(cfg)
	if len(results) != 1 {
		t.Fatalf("ожидался 1 результат, получено %d", len(results))
	}
	if !results[0].CacheHit {
		t.Error("CacheHit должен быть true")
	}
	if results[0].OrphanMatch != "TestApp" {
		t.Errorf("OrphanMatch = %q, want TestApp", results[0].OrphanMatch)
	}
	if results[0].Size != 5 {
		t.Errorf("Size = %d, want 5", results[0].Size)
	}
}
