package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultShortcutRoots_ReturnsNonEmpty(t *testing.T) {
	roots := DefaultShortcutRoots()
	if len(roots) == 0 {
		t.Fatal("ожидались стандартные корни для ярлыков, получен пустой список")
	}
	for _, r := range roots {
		if r == "" {
			t.Error("DefaultShortcutRoots не должен содержать пустые пути")
		}
	}
}

func TestScanBrokenShortcuts_NoLnkFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "not-a-shortcut.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	result, err := ScanBrokenShortcuts([]string{dir})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if result.Scanned != 0 {
		t.Errorf("Scanned = %d, ожидалось 0 (нет .lnk файлов)", result.Scanned)
	}
	if len(result.Broken) != 0 {
		t.Errorf("Broken = %v, ожидался пустой список", result.Broken)
	}
}

func TestScanBrokenShortcuts_MissingRootIgnored(t *testing.T) {
	result, err := ScanBrokenShortcuts([]string{filepath.Join(t.TempDir(), "definitely-missing")})
	if err != nil {
		t.Fatalf("несуществующий корень не должен приводить к ошибке: %v", err)
	}
	if result.Scanned != 0 || len(result.Broken) != 0 {
		t.Errorf("ожидался пустой результат для несуществующего корня, получено %+v", result)
	}
}

// TestScanBrokenShortcuts_DetectsBrokenTarget — интеграционный тест,
// создаёт настоящий .lnk через PowerShell/WScript.Shell и проверяет, что
// сканер находит его как битый. Пропускается в -short.
func TestScanBrokenShortcuts_DetectsBrokenTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("пропущено в -short: требует реального PowerShell/WScript.Shell")
	}

	dir := t.TempDir()
	lnkPath := filepath.Join(dir, "broken.lnk")
	missingTarget := filepath.Join(dir, "does-not-exist.exe")

	escapedLnk := replaceAllQuotes(lnkPath)
	escapedTarget := replaceAllQuotes(missingTarget)
	createScript := "$sh = New-Object -ComObject WScript.Shell; $s = $sh.CreateShortcut('" + escapedLnk + "'); $s.TargetPath = '" + escapedTarget + "'; $s.Save()"
	if err := runPowerShellEncoded(createScript); err != nil {
		t.Skipf("не удалось создать тестовый ярлык (WScript.Shell недоступен?): %v", err)
	}

	result, err := ScanBrokenShortcuts([]string{dir})
	if err != nil {
		t.Fatalf("ScanBrokenShortcuts: %v", err)
	}
	if result.Scanned != 1 {
		t.Fatalf("Scanned = %d, ожидался 1 ярлык", result.Scanned)
	}
	if len(result.Broken) != 1 {
		t.Fatalf("ожидался 1 битый ярлык, получено %d: %+v", len(result.Broken), result.Broken)
	}
	if result.Broken[0].Path != lnkPath {
		t.Errorf("Path = %q, ожидалось %q", result.Broken[0].Path, lnkPath)
	}
}

func replaceAllQuotes(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\'' {
			out = append(out, '\'', '\'')
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
