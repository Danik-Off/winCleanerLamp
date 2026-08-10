package cleaner

import (
	"path/filepath"
	"testing"
)

func TestLoadJunkConfig_MissingFileReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.json")

	cfg, err := LoadJunkConfig(path)
	if err != nil {
		t.Fatalf("LoadJunkConfig на несуществующем файле не должен возвращать ошибку: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg == nil")
	}
	if len(cfg.Records) != 0 {
		t.Errorf("ожидался пустой список записей, получено %d", len(cfg.Records))
	}
	if cfg.Stats == nil {
		t.Error("Stats должен быть инициализирован (не nil)")
	}
}

func TestSaveAndLoadJunkConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "junk.json")

	cfg, err := LoadJunkConfig(path)
	if err != nil {
		t.Fatalf("LoadJunkConfig: %v", err)
	}
	cfg.AddRecord(`C:\Temp\a.tmp`, "user-temp", 1024, "chrome")
	cfg.AddRecord(`C:\Temp\b.tmp`, "user-temp", 2048, "")

	if err := SaveJunkConfig(cfg, path); err != nil {
		t.Fatalf("SaveJunkConfig: %v", err)
	}

	loaded, err := LoadJunkConfig(path)
	if err != nil {
		t.Fatalf("LoadJunkConfig (после сохранения): %v", err)
	}
	if len(loaded.Records) != 2 {
		t.Fatalf("ожидалось 2 записи после round-trip, получено %d", len(loaded.Records))
	}
	if loaded.Stats["user-temp"] != 3072 {
		t.Errorf("Stats[user-temp] = %d, want 3072", loaded.Stats["user-temp"])
	}
}

func TestJunkConfig_AddRecordUpdatesExisting(t *testing.T) {
	cfg := &JunkConfig{Stats: map[string]int64{}}
	cfg.AddRecord(`C:\a.tmp`, "cat", 100, "")
	cfg.AddRecord(`C:\a.tmp`, "cat", 200, "hint") // тот же путь+категория — обновление, не дубль

	if len(cfg.Records) != 1 {
		t.Fatalf("ожидалась 1 запись после обновления существующей, получено %d", len(cfg.Records))
	}
	if cfg.Records[0].Size != 200 || cfg.Records[0].ProgramHint != "hint" {
		t.Errorf("запись не обновилась: %+v", cfg.Records[0])
	}
}

func TestJunkConfig_MarkDeletedAndGetNonDeleted(t *testing.T) {
	cfg := &JunkConfig{Stats: map[string]int64{}}
	cfg.AddRecord(`C:\a.tmp`, "cat", 100, "")
	cfg.AddRecord(`C:\b.tmp`, "cat", 200, "")

	cfg.MarkDeleted(`C:\a.tmp`, "cat")

	remaining := cfg.GetNonDeletedRecords()
	if len(remaining) != 1 {
		t.Fatalf("ожидалась 1 неудалённая запись, получено %d", len(remaining))
	}
	if remaining[0].Path != `C:\b.tmp` {
		t.Errorf("неожиданная оставшаяся запись: %+v", remaining[0])
	}
}

func TestJunkConfig_RemoveRecordUpdatesStats(t *testing.T) {
	cfg := &JunkConfig{Stats: map[string]int64{}}
	cfg.AddRecord(`C:\a.tmp`, "cat", 100, "")
	cfg.AddRecord(`C:\b.tmp`, "cat", 200, "")

	cfg.RemoveRecord(`C:\a.tmp`, "cat")

	if len(cfg.Records) != 1 {
		t.Fatalf("ожидалась 1 запись после удаления, получено %d", len(cfg.Records))
	}
	if cfg.Stats["cat"] != 200 {
		t.Errorf("Stats[cat] = %d, want 200 (100 должно было вычесться)", cfg.Stats["cat"])
	}
}

func TestJunkConfig_GetRecordsByCategoryAndProgram(t *testing.T) {
	cfg := &JunkConfig{Stats: map[string]int64{}}
	cfg.AddRecord(`C:\a.tmp`, "cache-a", 100, "chrome")
	cfg.AddRecord(`C:\b.tmp`, "cache-b", 200, "firefox")
	cfg.AddRecord(`C:\c.tmp`, "cache-a", 300, "chrome")

	byCategory := cfg.GetRecordsByCategory("cache-a")
	if len(byCategory) != 2 {
		t.Errorf("GetRecordsByCategory(cache-a): ожидалось 2, получено %d", len(byCategory))
	}

	byProgram := cfg.GetRecordsByProgram("chrome")
	if len(byProgram) != 2 {
		t.Errorf("GetRecordsByProgram(chrome): ожидалось 2, получено %d", len(byProgram))
	}
}
