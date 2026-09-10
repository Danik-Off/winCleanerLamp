//go:build linux

package cleaner

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Источники автозагрузки Linux. В отличие от Windows реестра здесь два
// принципиально разных механизма: XDG Autostart (.desktop-файлы, которые
// запускает сеанс рабочего стола) и пользовательские юниты systemd,
// работающие независимо от рабочего стола.
const (
	AutostartDesktopUser   AutostartSource = "autostart-user"
	AutostartDesktopSystem AutostartSource = "autostart-system"
	AutostartSystemdUser   AutostartSource = "systemd-user"
)

// AutostartSourceOrder — порядок вывода источников в CLI.
func AutostartSourceOrder() []AutostartSource {
	return []AutostartSource{AutostartDesktopUser, AutostartDesktopSystem, AutostartSystemdUser}
}

// AutostartSourceLabel — человекочитаемое имя источника.
func AutostartSourceLabel(s AutostartSource) string {
	switch s {
	case AutostartDesktopUser:
		return "XDG Autostart (пользователь, ~/.config/autostart)"
	case AutostartDesktopSystem:
		return "XDG Autostart (система, /etc/xdg/autostart)"
	case AutostartSystemdUser:
		return "Пользовательские юниты systemd (systemctl --user)"
	}
	return string(s)
}

// systemAutostartDirs — общесистемные каталоги XDG Autostart. Порядок важен:
// первым идёт то, что определено в $XDG_CONFIG_DIRS.
func systemAutostartDirs() []string {
	var dirs []string
	for _, d := range strings.Split(os.Getenv("XDG_CONFIG_DIRS"), ":") {
		if filepath.IsAbs(d) {
			dirs = append(dirs, filepath.Join(d, "autostart"))
		}
	}
	if len(dirs) == 0 {
		dirs = []string{"/etc/xdg/autostart"}
	}
	return dirs
}

func userAutostartDir() string { return sub(userConfigDir(), "autostart") }

// ListAutostartEntries собирает записи автозагрузки: .desktop-файлы
// пользователя и системы плюс включённые/выключенные пользовательские юниты
// systemd.
func ListAutostartEntries() []AutostartEntry {
	var entries []AutostartEntry

	userDir := userAutostartDir()
	userFiles := map[string]bool{}
	for _, e := range listDesktopAutostart(AutostartDesktopUser, userDir) {
		userFiles[e.Name] = true
		entries = append(entries, e)
	}

	// Системные записи, которые пользователь ещё не переопределил своим
	// файлом: одноимённый файл в ~/.config/autostart полностью заменяет
	// системный, и показывать обе записи было бы враньём.
	for _, dir := range systemAutostartDirs() {
		for _, e := range listDesktopAutostart(AutostartDesktopSystem, dir) {
			if userFiles[e.Name] {
				continue
			}
			entries = append(entries, e)
		}
	}

	entries = append(entries, listSystemdUserUnits()...)
	return entries
}

// ToggleAutostart включает/выключает запись автозагрузки по её ID.
//
// Для .desktop это ключ Hidden в файле пользователя (так же поступают
// GNOME Tweaks и KDE «Автозапуск»): исходный файл не удаляется, действие
// полностью обратимо. Системная запись при выключении не трогается —
// в ~/.config/autostart создаётся перекрывающая копия с Hidden=true.
// Для systemd — штатные systemctl --user enable/disable.
func ToggleAutostart(id string, enable bool) error {
	source, name, err := parseAutostartID(id)
	if err != nil {
		return err
	}
	switch source {
	case AutostartDesktopUser:
		dir := userAutostartDir()
		if dir == "" {
			return fmt.Errorf("не удалось определить ~/.config/autostart")
		}
		return setDesktopHidden(filepath.Join(dir, name), !enable)
	case AutostartDesktopSystem:
		return overrideSystemAutostart(name, enable)
	case AutostartSystemdUser:
		verb := "disable"
		if enable {
			verb = "enable"
		}
		out, err := exec.Command("systemctl", "--user", verb, name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("systemctl --user %s %s: %v: %s", verb, name, err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("неизвестный источник автозагрузки: %s", source)
	}
}

// ─── XDG Autostart (.desktop) ───

func listDesktopAutostart(source AutostartSource, dir string) []AutostartEntry {
	if dir == "" {
		return nil
	}
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]AutostartEntry, 0, len(dirEntries))
	for _, e := range dirEntries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".desktop") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		keys := readDesktopEntry(full)
		enabled := desktopEntryEnabled(keys)
		// Системную запись мог выключить пользовательский файл-перекрытие.
		if source == AutostartDesktopSystem {
			if override := sub(userAutostartDir(), e.Name()); override != "" {
				if _, err := os.Stat(override); err == nil {
					enabled = desktopEntryEnabled(readDesktopEntry(override))
				}
			}
		}
		name := keys["Name"]
		if name == "" {
			name = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		}
		out = append(out, AutostartEntry{
			ID:        string(source) + "|" + e.Name(),
			Source:    source,
			Name:      name,
			Command:   keys["Exec"],
			Location:  full,
			Enabled:   enabled,
			CanToggle: true,
		})
	}
	return out
}

// readDesktopEntry читает ключи секции [Desktop Entry]. Полноценный разбор
// ini здесь не нужен: интересуют Name, Exec, TryExec, Hidden и
// X-GNOME-Autostart-enabled, а локализованные варианты (Name[ru]) намеренно
// пропускаются — в отчёте нужен стабильный идентификатор, а не перевод.
func readDesktopEntry(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	keys := map[string]string{}
	inSection := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inSection = line == "[Desktop Entry]"
			continue
		}
		if !inSection {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if strings.Contains(k, "[") {
			continue // локализованный ключ
		}
		if _, exists := keys[k]; !exists {
			keys[k] = strings.TrimSpace(v)
		}
	}
	return keys
}

// desktopEntryEnabled — запись считается выключенной, если Hidden=true или
// X-GNOME-Autostart-enabled=false. Отсутствие обоих ключей означает
// «включена» — так же это трактуют сами окружения рабочего стола.
func desktopEntryEnabled(keys map[string]string) bool {
	if strings.EqualFold(keys["Hidden"], "true") {
		return false
	}
	if v, ok := keys["X-GNOME-Autostart-enabled"]; ok && strings.EqualFold(v, "false") {
		return false
	}
	return true
}

// setDesktopHidden выставляет Hidden=<hidden> в секции [Desktop Entry],
// сохраняя остальные строки файла как есть.
func setDesktopHidden(path string, hidden bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("чтение %s: %w", path, err)
	}
	value := "false"
	if hidden {
		value = "true"
	}

	lines := strings.Split(string(data), "\n")
	inSection, replaced := false, false
	sectionEnd := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			if inSection && sectionEnd < 0 {
				sectionEnd = i
			}
			inSection = trimmed == "[Desktop Entry]"
			continue
		}
		if !inSection {
			continue
		}
		if k, _, ok := strings.Cut(trimmed, "="); ok && strings.TrimSpace(k) == "Hidden" {
			lines[i] = "Hidden=" + value
			replaced = true
			break
		}
	}
	if !replaced {
		if sectionEnd < 0 {
			sectionEnd = len(lines)
		}
		rest := append([]string{"Hidden=" + value}, lines[sectionEnd:]...)
		lines = append(lines[:sectionEnd:sectionEnd], rest...)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

// overrideSystemAutostart создаёт (или правит) пользовательскую копию
// системного .desktop-файла — штатный для XDG способ выключить системную
// запись, не имея прав на /etc.
func overrideSystemAutostart(fileName string, enable bool) error {
	userDir := userAutostartDir()
	if userDir == "" {
		return fmt.Errorf("не удалось определить ~/.config/autostart")
	}
	override := filepath.Join(userDir, fileName)

	if _, err := os.Stat(override); err != nil {
		var source string
		for _, dir := range systemAutostartDirs() {
			candidate := filepath.Join(dir, fileName)
			if _, err := os.Stat(candidate); err == nil {
				source = candidate
				break
			}
		}
		if source == "" {
			return fmt.Errorf("системная запись автозагрузки %q не найдена", fileName)
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("чтение %s: %w", source, err)
		}
		if err := os.MkdirAll(userDir, 0o755); err != nil {
			return fmt.Errorf("создание %s: %w", userDir, err)
		}
		if err := os.WriteFile(override, data, 0o644); err != nil {
			return fmt.Errorf("запись %s: %w", override, err)
		}
	}
	return setDesktopHidden(override, !enable)
}

// ─── systemd --user ───

// listSystemdUserUnits перечисляет пользовательские юниты, состояние которых
// пользователь может менять (enabled/disabled). Статические и generated
// юниты пропускаются: их нельзя включить или выключить через systemctl.
func listSystemdUserUnits() []AutostartEntry {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil
	}
	out, err := exec.Command("systemctl", "--user", "list-unit-files",
		"--type=service,timer", "--no-legend", "--no-pager", "--plain").Output()
	if err != nil {
		return nil
	}
	var entries []AutostartEntry
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		unit, state := fields[0], fields[1]
		if state != "enabled" && state != "disabled" {
			continue
		}
		entries = append(entries, AutostartEntry{
			ID:        string(AutostartSystemdUser) + "|" + unit,
			Source:    AutostartSystemdUser,
			Name:      unit,
			Command:   "systemctl --user start " + unit,
			Location:  "systemd --user",
			Enabled:   state == "enabled",
			CanToggle: true,
		})
	}
	return entries
}
