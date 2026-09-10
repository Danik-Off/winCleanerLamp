//go:build darwin

package cleaner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Источники автозагрузки macOS: агенты и демоны launchd (plist-файлы) плюс
// «Объекты входа» из Системных настроек.
const (
	AutostartLaunchAgentUser   AutostartSource = "launch-agent-user"
	AutostartLaunchAgentSystem AutostartSource = "launch-agent-system"
	AutostartLaunchDaemon      AutostartSource = "launch-daemon"
	AutostartLoginItem         AutostartSource = "login-item"
)

// AutostartSourceOrder — порядок вывода источников в CLI.
func AutostartSourceOrder() []AutostartSource {
	return []AutostartSource{
		AutostartLaunchAgentUser, AutostartLaunchAgentSystem,
		AutostartLaunchDaemon, AutostartLoginItem,
	}
}

// AutostartSourceLabel — человекочитаемое имя источника.
func AutostartSourceLabel(s AutostartSource) string {
	switch s {
	case AutostartLaunchAgentUser:
		return "LaunchAgents (пользователь, ~/Library/LaunchAgents)"
	case AutostartLaunchAgentSystem:
		return "LaunchAgents (система, /Library/LaunchAgents)"
	case AutostartLaunchDaemon:
		return "LaunchDaemons (системные службы)"
	case AutostartLoginItem:
		return "Объекты входа (Системные настройки)"
	}
	return string(s)
}

// launchAgentDirs — каталоги plist-файлов и соответствующие им источники.
func launchAgentDirs() []struct {
	Source AutostartSource
	Dir    string
} {
	return []struct {
		Source AutostartSource
		Dir    string
	}{
		{AutostartLaunchAgentUser, sub(userLibraryDir(), "LaunchAgents")},
		{AutostartLaunchAgentSystem, "/Library/LaunchAgents"},
		{AutostartLaunchDaemon, "/Library/LaunchDaemons"},
	}
}

// ListAutostartEntries собирает агенты и демоны launchd и объекты входа.
// Каталоги /System/Library/Launch* намеренно не читаются: это компоненты
// самой macOS, защищённые SIP, и предлагать их выключить бессмысленно.
func ListAutostartEntries() []AutostartEntry {
	var entries []AutostartEntry
	disabled := launchctlDisabled()

	for _, spec := range launchAgentDirs() {
		if spec.Dir == "" {
			continue
		}
		files, err := os.ReadDir(spec.Dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.EqualFold(filepath.Ext(f.Name()), ".plist") {
				continue
			}
			full := filepath.Join(spec.Dir, f.Name())
			label, program := readLaunchdPlist(full)
			if label == "" {
				label = strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
			}
			entries = append(entries, AutostartEntry{
				ID:        string(spec.Source) + "|" + label,
				Source:    spec.Source,
				Name:      label,
				Command:   program,
				Location:  full,
				Enabled:   !disabled[label],
				CanToggle: true,
			})
		}
	}

	entries = append(entries, listLoginItems()...)
	return entries
}

// ToggleAutostart включает/выключает запись автозагрузки по её ID.
//
// Для launchd это launchctl enable/disable — состояние хранится в базе
// самого launchd, plist-файл не меняется, и действие полностью обратимо.
// Объекты входа не переключаются: в macOS их можно только добавить или
// удалить, а удаление — не то же самое, что «выключить», поэтому
// CanToggle у них false и здесь возвращается понятная ошибка.
func ToggleAutostart(id string, enable bool) error {
	source, label, err := parseAutostartID(id)
	if err != nil {
		return err
	}
	verb := "disable"
	if enable {
		verb = "enable"
	}
	switch source {
	case AutostartLaunchAgentUser, AutostartLaunchAgentSystem:
		target := fmt.Sprintf("gui/%d/%s", os.Getuid(), label)
		out, err := exec.Command("launchctl", verb, target).CombinedOutput()
		if err != nil {
			return fmt.Errorf("launchctl %s %s: %v: %s", verb, target, err, strings.TrimSpace(string(out)))
		}
		return nil
	case AutostartLaunchDaemon:
		target := "system/" + label
		out, err := exec.Command("launchctl", verb, target).CombinedOutput()
		if err != nil {
			return fmt.Errorf("launchctl %s %s (нужен sudo): %v: %s", verb, target, err, strings.TrimSpace(string(out)))
		}
		return nil
	case AutostartLoginItem:
		return fmt.Errorf("объекты входа нельзя выключить, только удалить — сделайте это в Системных настройках → Основные → Объекты входа")
	default:
		return fmt.Errorf("неизвестный источник автозагрузки: %s", source)
	}
}

// launchctlDisabled — метки, выключенные пользователем
// (launchctl print-disabled). Вывод имеет вид `"label" => disabled`.
func launchctlDisabled() map[string]bool {
	result := map[string]bool{}
	for _, domain := range []string{fmt.Sprintf("gui/%d", os.Getuid()), "system"} {
		out, err := exec.Command("launchctl", "print-disabled", domain).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			label, state, ok := strings.Cut(line, "=>")
			if !ok {
				continue
			}
			label = strings.Trim(strings.TrimSpace(label), `"`)
			if label == "" {
				continue
			}
			// В разных версиях macOS это "true"/"false" или
			// "disabled"/"enabled".
			state = strings.TrimSpace(state)
			if state == "true" || state == "disabled" {
				result[label] = true
			}
		}
	}
	return result
}

// readLaunchdPlist достаёт Label и первый аргумент запуска
// (Program или ProgramArguments[0]) из plist-файла. plist бывает и
// бинарным, поэтому он сначала переводится в JSON штатным plutil, и лишь
// если plutil недоступен — читается как XML-текст.
func readLaunchdPlist(path string) (label, program string) {
	if out, err := exec.Command("plutil", "-convert", "json", "-o", "-", path).Output(); err == nil {
		var parsed struct {
			Label            string   `json:"Label"`
			Program          string   `json:"Program"`
			ProgramArguments []string `json:"ProgramArguments"`
		}
		if err := json.Unmarshal(out, &parsed); err == nil {
			program = parsed.Program
			if program == "" && len(parsed.ProgramArguments) > 0 {
				program = parsed.ProgramArguments[0]
			}
			return parsed.Label, program
		}
	}
	return readLaunchdPlistXML(path)
}

// readLaunchdPlistXML — запасной разбор текстового plist: ищем значение,
// идущее за <key>Label</key> и <key>Program</key>.
func readLaunchdPlistXML(path string) (label, program string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	text := string(data)
	value := func(key string) string {
		idx := strings.Index(text, "<key>"+key+"</key>")
		if idx < 0 {
			return ""
		}
		rest := text[idx:]
		start := strings.Index(rest, "<string>")
		if start < 0 {
			return ""
		}
		rest = rest[start+len("<string>"):]
		end := strings.Index(rest, "</string>")
		if end < 0 {
			return ""
		}
		return strings.TrimSpace(rest[:end])
	}
	program = value("Program")
	if program == "" {
		program = value("ProgramArguments")
	}
	return value("Label"), program
}

// listLoginItems перечисляет объекты входа через System Events. Если
// AppleScript недоступен (нет разрешения на автоматизацию или сессия без
// графики), список просто будет пустым — это не ошибка очистки.
func listLoginItems() []AutostartEntry {
	if _, err := exec.LookPath("osascript"); err != nil {
		return nil
	}
	const script = `tell application "System Events" to get the name of every login item`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil
	}
	var entries []AutostartEntry
	for _, name := range strings.Split(strings.TrimSpace(string(out)), ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		entries = append(entries, AutostartEntry{
			ID:        string(AutostartLoginItem) + "|" + name,
			Source:    AutostartLoginItem,
			Name:      name,
			Location:  "Системные настройки → Объекты входа",
			Enabled:   true,
			CanToggle: false,
		})
	}
	return entries
}
