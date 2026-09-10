//go:build linux

package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func writeDesktopFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return p
}

func TestReadDesktopEntry_ParsesMainSectionOnly(t *testing.T) {
	p := writeDesktopFile(t, t.TempDir(), "app.desktop", `# комментарий
[Desktop Entry]
Type=Application
Name=Тестовое приложение
Name[en]=Test App
Exec=/usr/bin/test-app --flag %U
Hidden=false

[Desktop Action New]
Name=Новое окно
Exec=/usr/bin/other
`)
	keys := readDesktopEntry(p)
	if keys["Name"] != "Тестовое приложение" {
		t.Errorf("Name = %q, локализованный ключ не должен перекрывать основной", keys["Name"])
	}
	if keys["Exec"] != "/usr/bin/test-app --flag %U" {
		t.Errorf("Exec = %q", keys["Exec"])
	}
	if keys["Type"] != "Application" {
		t.Errorf("Type = %q", keys["Type"])
	}
}

func TestDesktopEntryEnabled(t *testing.T) {
	cases := []struct {
		keys map[string]string
		want bool
	}{
		{map[string]string{}, true},
		{map[string]string{"Hidden": "true"}, false},
		{map[string]string{"Hidden": "false"}, true},
		{map[string]string{"X-GNOME-Autostart-enabled": "false"}, false},
		{map[string]string{"X-GNOME-Autostart-enabled": "true"}, true},
	}
	for _, c := range cases {
		if got := desktopEntryEnabled(c.keys); got != c.want {
			t.Errorf("desktopEntryEnabled(%v) = %v, want %v", c.keys, got, c.want)
		}
	}
}

func TestSetDesktopHidden_RoundTrip(t *testing.T) {
	p := writeDesktopFile(t, t.TempDir(), "app.desktop", `[Desktop Entry]
Type=Application
Name=App
Exec=/usr/bin/app
`)
	if err := setDesktopHidden(p, true); err != nil {
		t.Fatalf("setDesktopHidden(true): %v", err)
	}
	if desktopEntryEnabled(readDesktopEntry(p)) {
		t.Error("после Hidden=true запись должна считаться выключенной")
	}
	if err := setDesktopHidden(p, false); err != nil {
		t.Fatalf("setDesktopHidden(false): %v", err)
	}
	keys := readDesktopEntry(p)
	if !desktopEntryEnabled(keys) {
		t.Error("после Hidden=false запись должна считаться включённой")
	}
	// Остальные ключи обязаны пережить правку — файл принадлежит не нам.
	if keys["Exec"] != "/usr/bin/app" || keys["Name"] != "App" {
		t.Errorf("правка Hidden потеряла другие ключи: %v", keys)
	}
}

func TestDesktopExecPath(t *testing.T) {
	cases := []struct {
		keys map[string]string
		want string
	}{
		{map[string]string{"Exec": "/usr/bin/app --flag %U"}, "/usr/bin/app"},
		{map[string]string{"Exec": `"/opt/My App/app" %F`}, "/opt/My App/app"},
		{map[string]string{"Exec": "env LANG=C /usr/bin/app"}, "/usr/bin/app"},
		{map[string]string{"TryExec": "/usr/bin/real", "Exec": "/usr/bin/wrapper"}, "/usr/bin/real"},
		{map[string]string{}, ""},
	}
	for _, c := range cases {
		if got := desktopExecPath(c.keys); got != c.want {
			t.Errorf("desktopExecPath(%v) = %q, want %q", c.keys, got, c.want)
		}
	}
}

func TestListDesktopAutostart(t *testing.T) {
	dir := t.TempDir()
	writeDesktopFile(t, dir, "enabled.desktop", "[Desktop Entry]\nType=Application\nName=Enabled\nExec=/bin/true\n")
	writeDesktopFile(t, dir, "disabled.desktop", "[Desktop Entry]\nType=Application\nName=Disabled\nExec=/bin/true\nHidden=true\n")
	writeDesktopFile(t, dir, "ignored.txt", "не ярлык")

	entries := listDesktopAutostart(AutostartDesktopUser, dir)
	if len(entries) != 2 {
		t.Fatalf("ожидалось 2 записи (.desktop), получено %d", len(entries))
	}
	byName := map[string]AutostartEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if !byName["Enabled"].Enabled {
		t.Error("запись без Hidden должна быть включена")
	}
	if byName["Disabled"].Enabled {
		t.Error("запись с Hidden=true должна быть выключена")
	}
	// ID должен разбираться обратно — на нём построен --autostart-set.
	source, name, err := parseAutostartID(byName["Enabled"].ID)
	if err != nil || source != AutostartDesktopUser || name != "enabled.desktop" {
		t.Errorf("parseAutostartID(%q) = (%v, %q, %v)", byName["Enabled"].ID, source, name, err)
	}
}

func TestToggleAutostart_UnknownSource(t *testing.T) {
	if err := ToggleAutostart("unknown-source|name", true); err == nil {
		t.Error("ожидалась ошибка для неизвестного источника")
	}
}

func TestToggleAutostart_MalformedID(t *testing.T) {
	for _, id := range []string{"", "no-separator", "|", "source|"} {
		if err := ToggleAutostart(id, true); err == nil {
			t.Errorf("ожидалась ошибка для некорректного id %q", id)
		}
	}
}
