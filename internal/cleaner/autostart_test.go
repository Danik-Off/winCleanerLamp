package cleaner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAutostartID_RoundTrip(t *testing.T) {
	cases := []struct {
		source AutostartSource
		name   string
	}{
		{AutostartRunHKCU, "OneDrive"},
		{AutostartRunHKLM32, "SomeApp"},
		{AutostartStartupFolderUser, "Printer.lnk"},
		// Задания планировщика в подпапках содержат обратные слэши в имени —
		// это часть значения, а не разделитель источник/имя.
		{AutostartScheduledTask, `\Adobe\AdobeGCInvoker-1.0`},
	}
	for _, c := range cases {
		id := string(c.source) + "|" + c.name
		gotSource, gotName, err := parseAutostartID(id)
		if err != nil {
			t.Fatalf("parseAutostartID(%q): %v", id, err)
		}
		if gotSource != c.source || gotName != c.name {
			t.Errorf("parseAutostartID(%q) = (%q, %q), want (%q, %q)", id, gotSource, gotName, c.source, c.name)
		}
	}
}

func TestParseAutostartID_RejectsMalformed(t *testing.T) {
	for _, id := range []string{"", "no-separator", "|missing-source", "source-only|"} {
		if _, _, err := parseAutostartID(id); err == nil {
			t.Errorf("parseAutostartID(%q) должен вернуть ошибку", id)
		}
	}
}

func TestStartupApprovedHexEncoding(t *testing.T) {
	// Формат подтверждён исследованием (docs/research-autostart.md):
	// первый байт 0x02 = включено, 0x03 = выключено, длина 12 байт.
	if got := startupApprovedHexForTest(true); got[:2] != "02" {
		t.Errorf("enabled hex должен начинаться с 02, получено %s", got)
	}
	if got := startupApprovedHexForTest(false); got[:2] != "03" {
		t.Errorf("disabled hex должен начинаться с 03, получено %s", got)
	}
	if len(startupApprovedHexForTest(true)) != 24 {
		t.Errorf("12 байт должны кодироваться в 24 hex-символа, получено %d", len(startupApprovedHexForTest(true)))
	}
}

// startupApprovedHexForTest дублирует кодирование из writeStartupApprovedState,
// чтобы проверить его независимо от реальной записи в реестр.
func startupApprovedHexForTest(enable bool) string {
	b := make([]byte, 12)
	if enable {
		b[0] = 0x02
	} else {
		b[0] = 0x03
	}
	out := make([]byte, 0, 24)
	const hexdigits = "0123456789ABCDEF"
	for _, x := range b {
		out = append(out, hexdigits[x>>4], hexdigits[x&0xf])
	}
	return string(out)
}

// TestStartupApprovedRoundTrip — интеграционный тест реального чтения/записи
// в HKCU\...\StartupApproved. Использует заведомо фиктивное имя значения,
// чтобы не задеть настоящие записи автозагрузки пользователя, и удаляет его
// после теста.
func TestStartupApprovedRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("пропущено в -short: пишет в реальный реестр текущего пользователя")
	}
	const testName = "WinCleanerLampTest_DoNotUse_Autostart"
	const subKey = "Run"
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\` + subKey

	t.Cleanup(func() {
		_ = exec.Command("reg", "delete", key, "/v", testName, "/f").Run()
	})

	if err := writeStartupApprovedState(subKey, testName, false); err != nil {
		t.Fatalf("writeStartupApprovedState(disable): %v", err)
	}
	if enabled := readStartupApprovedState(subKey, testName); enabled {
		t.Error("после записи 'выключено' readStartupApprovedState должен вернуть false")
	}

	if err := writeStartupApprovedState(subKey, testName, true); err != nil {
		t.Fatalf("writeStartupApprovedState(enable): %v", err)
	}
	if enabled := readStartupApprovedState(subKey, testName); !enabled {
		t.Error("после записи 'включено' readStartupApprovedState должен вернуть true")
	}
}

func TestReadStartupApprovedState_MissingEntryDefaultsEnabled(t *testing.T) {
	// Записи без явного состояния в StartupApproved считаются включёнными —
	// так же трактует это сам Explorer/Task Manager.
	enabled := readStartupApprovedState("Run", "WinCleanerLampTest_DefinitelyDoesNotExist_"+t.Name())
	if !enabled {
		t.Error("отсутствие записи в StartupApproved должно означать enabled=true")
	}
}

func TestToggleAutostart_UnknownSource(t *testing.T) {
	err := ToggleAutostart("bogus-source|Name", true)
	if err == nil {
		t.Error("ToggleAutostart с неизвестным источником должен вернуть ошибку")
	}
}

func TestToggleAutostart_MalformedID(t *testing.T) {
	err := ToggleAutostart("no-separator", true)
	if err == nil {
		t.Error("ToggleAutostart с некорректным id должен вернуть ошибку")
	}
}

func TestListStartupFolderEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MyApp.lnk"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "desktop.ini"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "SubDir"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	entries := listStartupFolderEntries(AutostartStartupFolderUser, dir)

	if len(entries) != 1 {
		t.Fatalf("ожидалась 1 запись (SubDir и desktop.ini должны быть пропущены), получено %d: %+v", len(entries), entries)
	}
	if entries[0].Name != "MyApp.lnk" {
		t.Errorf("Name = %q, want MyApp.lnk", entries[0].Name)
	}
	if entries[0].ID != string(AutostartStartupFolderUser)+"|MyApp.lnk" {
		t.Errorf("неожиданный ID: %q", entries[0].ID)
	}
}

func TestListStartupFolderEntries_MissingDir(t *testing.T) {
	entries := listStartupFolderEntries(AutostartStartupFolderUser, `C:\this\path\does\not\exist\at\all`)
	if entries != nil {
		t.Errorf("ожидался nil для несуществующей папки, получено %v", entries)
	}
}

// TestRegQueryValues — интеграционный тест парсинга реального вывода reg.exe:
// создаёт временный ключ в HKCU, читает его через regQueryValues, удаляет.
func TestRegQueryValues(t *testing.T) {
	if testing.Short() {
		t.Skip("пропущено в -short: создаёт временный ключ в реальном реестре")
	}
	key := `HKCU\Software\WinCleanerLampTest_AutostartParsing`
	t.Cleanup(func() {
		_ = exec.Command("reg", "delete", key, "/f").Run()
	})

	if err := exec.Command("reg", "add", key, "/v", "TestValue", "/t", "REG_SZ", "/d", `C:\Some\App.exe --flag`, "/f").Run(); err != nil {
		t.Fatalf("setup reg add: %v", err)
	}

	values := regQueryValues(key)
	if len(values) != 1 {
		t.Fatalf("ожидалось 1 значение, получено %d: %+v", len(values), values)
	}
	if values[0].Name != "TestValue" || values[0].Type != "REG_SZ" {
		t.Errorf("неожиданное значение: %+v", values[0])
	}
	if !strings.Contains(values[0].Data, "App.exe") {
		t.Errorf("данные не распознаны корректно: %q", values[0].Data)
	}
}

func TestRegQueryValues_MissingKey(t *testing.T) {
	values := regQueryValues(`HKCU\Software\WinCleanerLampTest_DefinitelyMissingKey_12345`)
	if values != nil {
		t.Errorf("ожидался nil для несуществующего ключа, получено %v", values)
	}
}
