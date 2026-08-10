package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteFile_RejectsUnsafePath(t *testing.T) {
	r := DeleteFile(`C:\Windows\System32\drivers\etc\hosts`, true)
	if r.Success {
		t.Fatal("DeleteFile на защищённом пути должен провалиться")
	}
	if r.Error == "" {
		t.Error("ожидалась причина отказа")
	}
}

func TestDeleteFile_RejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	r := DeleteFile(dir, true)
	if r.Success {
		t.Fatal("DeleteFile на папке должен провалиться (используйте DeleteDir)")
	}
}

func TestDeleteFile_RejectsMissingFile(t *testing.T) {
	dir := t.TempDir()
	r := DeleteFile(filepath.Join(dir, "not-there.txt"), true)
	if r.Success {
		t.Fatal("DeleteFile на несуществующем файле должен провалиться")
	}
}

func TestDeleteFile_PermanentRemovesRealFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "junk.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := DeleteFile(file, true)
	if !r.Success {
		t.Fatalf("DeleteFile(permanent) должен успешно удалить обычный файл: %s", r.Error)
	}
	if r.MovedToRecycleBin {
		t.Error("permanent=true не должен идти через Корзину")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("файл должен быть физически удалён")
	}
}

func TestDeleteDir_RejectsUnsafePath(t *testing.T) {
	r := DeleteDir(`C:\Program Files`, true)
	if r.Success {
		t.Fatal("DeleteDir на защищённом пути должен провалиться")
	}
}

func TestDeleteDir_RejectsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notadir.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	r := DeleteDir(file, true)
	if r.Success {
		t.Fatal("DeleteDir на файле должен провалиться (используйте DeleteFile)")
	}
}

func TestDeleteDir_PermanentRemovesRealDir(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "victim")
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := DeleteDir(dir, true)
	if !r.Success {
		t.Fatalf("DeleteDir(permanent) должен успешно удалить обычную папку: %s", r.Error)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("папка должна быть физически удалена вместе с содержимым")
	}
}

// TestDeleteFile_RecycleBinRoundTrip — интеграционный тест реального
// перемещения в Корзину через PowerShell. Пропускается в -short и если
// окружение не поддерживает Корзину (например, headless CI без Explorer).
func TestDeleteFile_RecycleBinRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("пропущено в -short: обращается к реальной Корзине через PowerShell")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "recycle-me.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := DeleteFile(file, false)
	if !r.Success {
		t.Skipf("Корзина недоступна в этом окружении: %s", r.Error)
	}
	if !r.MovedToRecycleBin {
		t.Error("permanent=false должен перемещать в Корзину, а не удалять навсегда")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("файл должен исчезнуть из исходного расположения после перемещения в Корзину")
	}
}
