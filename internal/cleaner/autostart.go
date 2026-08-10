package cleaner

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AutostartSource — откуда взята запись автозагрузки.
// См. docs/research-autostart.md для обоснования списка источников и того,
// как Windows хранит состояние включено/выключено (StartupApproved).
type AutostartSource string

const (
	AutostartRunHKCU             AutostartSource = "run-hkcu"
	AutostartRunHKLM             AutostartSource = "run-hklm"
	AutostartRunHKLM32           AutostartSource = "run-hklm32" // WOW6432Node
	AutostartStartupFolderUser   AutostartSource = "startup-folder-user"
	AutostartStartupFolderCommon AutostartSource = "startup-folder-common"
	AutostartScheduledTask       AutostartSource = "scheduled-task"
)

// AutostartEntry — одна запись автозагрузки.
type AutostartEntry struct {
	ID        string          `json:"id"` // <source>|<name> — используется для ToggleAutostart
	Source    AutostartSource `json:"source"`
	Name      string          `json:"name"`
	Command   string          `json:"command,omitempty"`
	Location  string          `json:"location"`
	Enabled   bool            `json:"enabled"`
	CanToggle bool            `json:"canToggle"`
}

// ListAutostartEntries собирает все известные автозапускаемые записи:
// реестровые Run-ключи (HKCU/HKLM/WOW6432Node), обе папки автозагрузки и
// задания планировщика с триггером "при входе в систему".
func ListAutostartEntries() []AutostartEntry {
	var entries []AutostartEntry

	for _, spec := range runKeySpecs() {
		approvedSub := "Run"
		if spec.Source == AutostartRunHKLM32 {
			approvedSub = "Run32"
		}
		for _, v := range regQueryValues(spec.Key) {
			if v.Type != "REG_SZ" && v.Type != "REG_EXPAND_SZ" {
				continue
			}
			entries = append(entries, AutostartEntry{
				ID:        string(spec.Source) + "|" + v.Name,
				Source:    spec.Source,
				Name:      v.Name,
				Command:   v.Data,
				Location:  spec.Key,
				Enabled:   readStartupApprovedState(approvedSub, v.Name),
				CanToggle: true,
			})
		}
	}

	entries = append(entries, listStartupFolderEntries(
		AutostartStartupFolderUser, ExpandPath(`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`))...)
	entries = append(entries, listStartupFolderEntries(
		AutostartStartupFolderCommon, `C:\ProgramData\Microsoft\Windows\Start Menu\Programs\Startup`)...)

	entries = append(entries, listLogonScheduledTasks()...)

	return entries
}

// ToggleAutostart включает/выключает запись автозагрузки по её ID.
// Для Run-ключей и папок автозагрузки это запись в
// HKCU\...\StartupApproved (как это делает сам Task Manager — исходная
// команда не удаляется, действие полностью обратимо). Для заданий
// планировщика — штатное состояние задания через schtasks.
func ToggleAutostart(id string, enable bool) error {
	source, name, err := parseAutostartID(id)
	if err != nil {
		return err
	}
	switch source {
	case AutostartRunHKCU, AutostartRunHKLM:
		return writeStartupApprovedState("Run", name, enable)
	case AutostartRunHKLM32:
		return writeStartupApprovedState("Run32", name, enable)
	case AutostartStartupFolderUser, AutostartStartupFolderCommon:
		return writeStartupApprovedState("StartupFolder", name, enable)
	case AutostartScheduledTask:
		flag := "/Enable"
		if !enable {
			flag = "/Disable"
		}
		cmd := exec.Command("schtasks", "/Change", "/TN", name, flag)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("неизвестный источник автозагрузки: %s", source)
	}
}

func parseAutostartID(id string) (AutostartSource, string, error) {
	parts := strings.SplitN(id, "|", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", "", fmt.Errorf("некорректный id записи автозагрузки: %q", id)
	}
	return AutostartSource(parts[0]), parts[1], nil
}

// ─── Run-ключи ───

type runKeySpec struct {
	Source AutostartSource
	Key    string
}

func runKeySpecs() []runKeySpec {
	return []runKeySpec{
		{AutostartRunHKCU, `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`},
		{AutostartRunHKLM, `HKLM\Software\Microsoft\Windows\CurrentVersion\Run`},
		{AutostartRunHKLM32, `HKLM\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`},
	}
}

type regValue struct {
	Name string
	Type string
	Data string
}

// regQueryValues парсит вывод "reg query <key>" (без /s — только значения
// самого ключа, не подключей).
func regQueryValues(key string) []regValue {
	out, err := exec.Command("reg", "query", key).Output()
	if err != nil {
		return nil
	}
	var values []regValue
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, "    ") {
			continue // заголовок ключа/пустая строка
		}
		for _, t := range []string{"REG_EXPAND_SZ", "REG_SZ", "REG_BINARY", "REG_DWORD"} {
			marker := "    " + t + "    "
			idx := strings.Index(line, marker)
			if idx < 0 {
				continue
			}
			values = append(values, regValue{
				Name: strings.TrimSpace(line[:idx]),
				Type: t,
				Data: strings.TrimSpace(line[idx+len(marker):]),
			})
			break
		}
	}
	return values
}

// ─── Папки автозагрузки ───

func listStartupFolderEntries(source AutostartSource, dir string) []AutostartEntry {
	var out []AutostartEntry
	if dir == "" {
		return out
	}
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range dirEntries {
		if e.IsDir() || strings.EqualFold(e.Name(), "desktop.ini") {
			continue
		}
		name := e.Name()
		full := filepath.Join(dir, name)
		out = append(out, AutostartEntry{
			ID:        string(source) + "|" + name,
			Source:    source,
			Name:      name,
			Command:   full,
			Location:  full,
			Enabled:   readStartupApprovedState("StartupFolder", name),
			CanToggle: true,
		})
	}
	return out
}

// ─── Задания планировщика с триггером "при входе в систему" ───

// listLogonScheduledTasks находит задания через PowerShell (Get-ScheduledTask),
// а не текстовый "schtasks /query /v /fo csv" — вывод schtasks локализован
// (на русской Windows строка триггера будет на русском), а имена CIM-классов
// в PowerShell — нет. См. docs/research-autostart.md.
func listLogonScheduledTasks() []AutostartEntry {
	const script = `@(Get-ScheduledTask | Where-Object { $_.Triggers | Where-Object { $_.CimClass.CimClassName -eq 'MSFT_TaskLogonTrigger' } } | Select-Object TaskName, TaskPath, @{N='State';E={$_.State.ToString()}}) | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return nil
	}
	var raw []struct {
		TaskName string `json:"TaskName"`
		TaskPath string `json:"TaskPath"`
		State    string `json:"State"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil
	}
	result := make([]AutostartEntry, 0, len(raw))
	for _, t := range raw {
		full := t.TaskPath + t.TaskName
		result = append(result, AutostartEntry{
			ID:        string(AutostartScheduledTask) + "|" + full,
			Source:    AutostartScheduledTask,
			Name:      t.TaskName,
			Command:   full,
			Location:  full,
			Enabled:   !strings.EqualFold(t.State, "Disabled"),
			CanToggle: true,
		})
	}
	return result
}

// ─── StartupApproved (реестровое состояние включено/выключено) ───

// readStartupApprovedState читает 12-байтовое REG_BINARY значение из
// HKCU\...\StartupApproved\<subKey>. Отсутствие записи означает "включено"
// (так же трактует это сам Explorer/Task Manager).
func readStartupApprovedState(subKey, name string) bool {
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\` + subKey
	out, err := exec.Command("reg", "query", key, "/v", name).Output()
	if err != nil {
		return true
	}
	idx := strings.Index(string(out), "REG_BINARY")
	if idx < 0 {
		return true
	}
	hexStr := strings.Fields(string(out)[idx+len("REG_BINARY"):])
	if len(hexStr) == 0 || len(hexStr[0]) < 2 {
		return true
	}
	data, err := hex.DecodeString(hexStr[0])
	if err != nil || len(data) == 0 {
		return true
	}
	// 0x02 = включено, 0x03 = выключено (см. docs/research-autostart.md)
	return data[0] != 0x03
}

// writeStartupApprovedState записывает состояние включено/выключено.
func writeStartupApprovedState(subKey, name string, enable bool) error {
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\` + subKey
	b := make([]byte, 12)
	if enable {
		b[0] = 0x02
	} else {
		b[0] = 0x03
	}
	hexData := strings.ToUpper(hex.EncodeToString(b))
	cmd := exec.Command("reg", "add", key, "/v", name, "/t", "REG_BINARY", "/d", hexData, "/f")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
