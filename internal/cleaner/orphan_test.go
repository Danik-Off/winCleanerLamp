package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateOrphanConfig_DetectsDuplicates(t *testing.T) {
	cfg := &OrphanConfig{Apps: []OrphanApp{
		{DisplayName: "Minecraft (Java Edition)"},
		{DisplayName: "minecraft (java edition)"}, // регистр не важен
		{DisplayName: "Unique App"},
	}}
	warnings := validateOrphanConfig(cfg)
	if len(warnings) != 1 {
		t.Fatalf("ожидалось 1 предупреждение о дубле, получено %d: %v", len(warnings), warnings)
	}
}

func TestValidateOrphanConfig_NoWarningsWhenUnique(t *testing.T) {
	cfg := &OrphanConfig{Apps: []OrphanApp{
		{DisplayName: "App A"},
		{DisplayName: "App B"},
	}}
	if warnings := validateOrphanConfig(cfg); len(warnings) != 0 {
		t.Errorf("не ожидалось предупреждений, получено %v", warnings)
	}
}

func TestSaveAndLoadOrphanConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orphaned_apps.json")

	cfg := &OrphanConfig{Apps: []OrphanApp{
		{
			DisplayName:     "Test App",
			InstallPaths:    []string{`%APPDATA%\TestApp`},
			CachePaths:      []string{`%APPDATA%\TestApp\Cache`},
			UserDataPaths:   []string{`%APPDATA%\TestApp\saves`},
			AdditionalPaths: []string{`%APPDATA%\TestApp\extra`},
		},
	}}

	if err := SaveOrphanConfig(cfg, path); err != nil {
		t.Fatalf("SaveOrphanConfig: %v", err)
	}

	loaded, err := LoadOrphanConfig(path)
	if err != nil {
		t.Fatalf("LoadOrphanConfig: %v", err)
	}
	if len(loaded.Apps) != 1 || loaded.Apps[0].DisplayName != "Test App" {
		t.Fatalf("неожиданное содержимое после round-trip: %+v", loaded.Apps)
	}
	if len(loaded.Apps[0].UserDataPaths) != 1 {
		t.Errorf("UserDataPaths не сохранились: %+v", loaded.Apps[0])
	}
}

func TestLoadOrphanConfig_AcceptsPlainArrayFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orphaned_apps.json")
	// Legacy-формат: просто массив без обёртки {"apps": [...]}
	data := `[{"displayName": "Legacy App", "installPaths": []}]`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := LoadOrphanConfig(path)
	if err != nil {
		t.Fatalf("LoadOrphanConfig(массив): %v", err)
	}
	if len(cfg.Apps) != 1 || cfg.Apps[0].DisplayName != "Legacy App" {
		t.Fatalf("неожиданное содержимое: %+v", cfg.Apps)
	}
}

func TestLoadOrphanConfig_MissingFile(t *testing.T) {
	_, err := LoadOrphanConfig(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil {
		t.Fatal("ожидалась ошибка для несуществующего файла")
	}
}

func TestSanitizeID(t *testing.T) {
	cases := map[string]string{
		"Google Chrome":   "google-chrome",
		"App! With@#Junk": "app-withjunk",
	}
	for in, want := range cases {
		if got := sanitizeID(in); got != want {
			t.Errorf("sanitizeID(%q) = %q, want %q", in, got, want)
		}
	}
	long := "this-is-a-very-long-display-name-that-should-be-truncated-somewhere"
	if got := sanitizeID(long); len(got) > 30 {
		t.Errorf("sanitizeID должен обрезать до 30 символов, получено %d: %q", len(got), got)
	}
}

func TestIsLikelyUserDataPath(t *testing.T) {
	likely := []string{
		`C:\Users\Me\AppData\Roaming\.minecraft\saves`,
		`C:\Users\Me\Documents\MyGame\Screenshots`,
		`C:\Users\Me\AppData\Roaming\.minecraft\resourcepacks`,
	}
	for _, p := range likely {
		if !isLikelyUserDataPath(p) {
			t.Errorf("isLikelyUserDataPath(%q) = false, want true", p)
		}
	}
	notLikely := []string{
		`C:\Users\Me\AppData\Local\SomeApp\Cache`,
		`C:\Users\Me\AppData\Roaming\SomeApp\logs`,
	}
	for _, p := range notLikely {
		if isLikelyUserDataPath(p) {
			t.Errorf("isLikelyUserDataPath(%q) = true, want false", p)
		}
	}
}

func TestCacheTargetsFromOrphanConfig(t *testing.T) {
	cfg := &OrphanConfig{Apps: []OrphanApp{
		{DisplayName: "App With Cache", CachePaths: []string{`%APPDATA%\App\Cache`}},
		{DisplayName: "App Without Cache"},
	}}
	targets := CacheTargetsFromOrphanConfig(cfg)
	if len(targets) != 1 {
		t.Fatalf("ожидался 1 target (только для приложения с cachePaths), получено %d", len(targets))
	}
	if !targets[0].KeepRoot {
		t.Error("orphan cache targets должны иметь KeepRoot=true")
	}
}

func TestCacheTargetsFromOrphanConfig_NilConfig(t *testing.T) {
	if got := CacheTargetsFromOrphanConfig(nil); got != nil {
		t.Errorf("ожидался nil для nil-конфига, получено %v", got)
	}
}

// setupOrphanFixture создаёт временную "программу" с реальными файлами на
// диске для end-to-end проверки scanOneOrphan/cleanOneOrphan без обращения
// к реестру/установленным программам.
func setupOrphanFixture(t *testing.T) (OrphanApp, string) {
	t.Helper()
	dir := t.TempDir()

	cacheDir := filepath.Join(dir, "cache")
	dataDir := filepath.Join(dir, "data")
	savesDir := filepath.Join(dir, "saves") // похоже на пользовательские данные по имени

	for _, d := range []string{cacheDir, dataDir, savesDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("setup dir %s: %v", d, err)
		}
	}
	mustWrite := func(dir, name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0o644); err != nil {
			t.Fatalf("setup file: %v", err)
		}
	}
	mustWrite(cacheDir, "cache.dat")
	mustWrite(dataDir, "settings.ini")
	mustWrite(savesDir, "world1.dat")

	app := OrphanApp{
		DisplayName:     "WinCleanerLampTestFixtureApp",
		InstallPaths:    []string{dataDir},
		CachePaths:      []string{cacheDir},
		AdditionalPaths: []string{savesDir}, // намеренно НЕ в userDataPaths — ловим эвристикой по имени
	}
	return app, dir
}

func TestScanOneOrphan_FindsPathsAndFlagsUserData(t *testing.T) {
	app, _ := setupOrphanFixture(t)

	result := scanOneOrphan(app)

	if len(result.FoundPaths) != 3 {
		t.Fatalf("ожидалось 3 найденных пути (data, cache, saves), получено %d: %+v", len(result.FoundPaths), result.FoundPaths)
	}

	var sawUserData bool
	for _, p := range result.FoundPaths {
		if filepath.Base(p.Path) == "saves" {
			sawUserData = true
			if !p.LikelyUserData {
				t.Error("папка 'saves' должна быть помечена LikelyUserData даже без явного userDataPaths")
			}
		}
	}
	if !sawUserData {
		t.Fatal("не нашли путь saves в результатах")
	}
}

func TestCleanOneOrphan_CacheOnlyTouchesOnlyCache(t *testing.T) {
	app, _ := setupOrphanFixture(t)

	result := cleanOneOrphan(app, OrphanCleanOptions{CacheOnly: true})

	if len(result.Errors) != 0 {
		t.Fatalf("неожиданные ошибки: %v", result.Errors)
	}
	if len(result.DeletedPaths) != 1 {
		t.Fatalf("cache-only должен удалить ровно 1 путь (cache), получено %d: %v", len(result.DeletedPaths), result.DeletedPaths)
	}
	for _, p := range app.CachePaths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("кеш должен быть удалён: %s", p)
		}
	}
	for _, p := range app.InstallPaths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("installPaths не должны трогаться в cache-only режиме: %s (%v)", p, err)
		}
	}
}

func TestCleanOneOrphan_FullCleanSkipsUserDataByDefault(t *testing.T) {
	app, _ := setupOrphanFixture(t)

	result := cleanOneOrphan(app, OrphanCleanOptions{CacheOnly: false})

	if len(result.SkippedUserData) != 1 {
		t.Fatalf("ожидался 1 пропущенный путь (saves, похож на пользовательские данные), получено %d: %v", len(result.SkippedUserData), result.SkippedUserData)
	}
	for _, p := range app.AdditionalPaths { // saves
		if _, err := os.Stat(p); err != nil {
			t.Errorf("saves не должен быть удалён без --orphan-include-user-data: %s (%v)", p, err)
		}
	}
	for _, p := range app.InstallPaths { // data — не похоже на пользовательские данные, должен удалиться
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("installPaths должны быть удалены при полной очистке: %s", p)
		}
	}
}

func TestCleanOneOrphan_FullCleanIncludesUserDataWhenRequested(t *testing.T) {
	app, _ := setupOrphanFixture(t)

	result := cleanOneOrphan(app, OrphanCleanOptions{CacheOnly: false, IncludeUserData: true})

	if len(result.SkippedUserData) != 0 {
		t.Errorf("с IncludeUserData=true ничего не должно пропускаться, пропущено: %v", result.SkippedUserData)
	}
	for _, p := range app.AdditionalPaths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("saves должен быть удалён при явном IncludeUserData=true: %s", p)
		}
	}
}

func TestCleanOneOrphan_RejectsUnsafePath(t *testing.T) {
	app := OrphanApp{
		DisplayName:  "Malicious",
		InstallPaths: []string{`C:\Windows\System32`},
	}
	result := cleanOneOrphan(app, OrphanCleanOptions{CacheOnly: false})
	if len(result.DeletedPaths) != 0 {
		t.Errorf("небезопасный путь не должен удаляться: %v", result.DeletedPaths)
	}
	if len(result.Errors) == 0 {
		t.Error("ожидалась ошибка о небезопасном пути")
	}
}
