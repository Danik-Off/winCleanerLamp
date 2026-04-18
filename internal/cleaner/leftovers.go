package cleaner

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// LeftoverCandidate — папка, похожая на остаток от удалённой программы.
type LeftoverCandidate struct {
	Path   string
	Size   int64
	Files  int
	Reason string // почему помечена (например, "нет в Uninstall registry")
}

// ScanLeftovers ищет папки в AppData\Roaming, AppData\Local, ProgramData,
// которые не соответствуют ни одной установленной программе (HKLM/HKCU Uninstall)
// и не входят в whitelist системных/встроенных имён.
//
// ВНИМАНИЕ: это эвристика — возвращается только для просмотра, не удаляется автоматически.
func ScanLeftovers() ([]LeftoverCandidate, error) {
	installed := installedProgramNames()
	whitelist := knownSystemFolders()

	roots := []string{
		ExpandPath(`%APPDATA%`),
		ExpandPath(`%LOCALAPPDATA%`),
		`C:\ProgramData`,
	}

	var out []LeftoverCandidate
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			low := strings.ToLower(name)
			if whitelist[low] {
				continue
			}
			if matchesInstalled(name, installed) {
				continue
			}
			full := filepath.Join(root, name)
			size, files := dirSize(full)
			if size == 0 && files == 0 {
				continue
			}
			out = append(out, LeftoverCandidate{
				Path:   full,
				Size:   size,
				Files:  files,
				Reason: "нет в списке установленных программ",
			})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	return out, nil
}

// installedProgramNames читает DisplayName из реестра (Uninstall и Publisher папки).
func installedProgramNames() map[string]bool {
	keys := []string{
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	result := map[string]bool{}
	for _, k := range keys {
		out, err := exec.Command("reg", "query", k, "/s", "/v", "DisplayName").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			// формат: "    DisplayName    REG_SZ    My App Name"
			if !strings.HasPrefix(line, "DisplayName") {
				continue
			}
			idx := strings.Index(line, "REG_SZ")
			if idx < 0 {
				continue
			}
			name := strings.TrimSpace(line[idx+len("REG_SZ"):])
			if name == "" {
				continue
			}
			for _, tok := range tokenize(name) {
				result[tok] = true
			}
		}
	}
	// Также добавим вендоров из HKLM\SOFTWARE (первый уровень) — часто имя папки AppData совпадает с вендором.
	if out, err := exec.Command("reg", "query", `HKLM\SOFTWARE`).Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(strings.ToUpper(line), "HKEY_LOCAL_MACHINE\\SOFTWARE\\") {
				continue
			}
			parts := strings.Split(line, `\`)
			if len(parts) == 0 {
				continue
			}
			last := strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
			if last != "" {
				result[last] = true
			}
		}
	}
	return result
}

// tokenize режет DisplayName на значимые токены (слова >2 символов, lowercase).
func tokenize(s string) []string {
	s = strings.ToLower(s)
	var out []string
	var b strings.Builder
	flush := func() {
		w := b.String()
		b.Reset()
		if len(w) >= 3 {
			out = append(out, w)
		}
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	// Плюс целая строка без пробелов
	out = append(out, strings.ReplaceAll(s, " ", ""))
	return out
}

// matchesInstalled: true если имя папки пересекается с любым токеном установленного ПО.
func matchesInstalled(folder string, installed map[string]bool) bool {
	toks := tokenize(folder)
	for _, t := range toks {
		if installed[t] {
			return true
		}
	}
	// также: подстрочная проверка
	low := strings.ToLower(folder)
	for k := range installed {
		if len(k) >= 4 && strings.Contains(low, k) {
			return true
		}
	}
	return false
}

// knownSystemFolders — имена папок в AppData/ProgramData, которые не являются остатками программ.
func knownSystemFolders() map[string]bool {
	list := []string{
		// общие системные / встроенные
		"microsoft", "windows", "windowsapps", "packages", "temp", "tmp",
		"d3dscache", "connecteddevicesplatform", "comms", "crashdumps",
		"virtualstore", "programs", "application data", "history", "desktop",
		"downloaded installations", "downloads", "diagnosis", "publishers",
		// ProgramData системные
		"ssh", "regid.1991-06.com.microsoft", "usoshared",
		// популярные вендоры, у которых AppData-папка остаётся навсегда
		"adobe", "google", "mozilla", "apple", "realtek", "intel", "nvidia",
		"amd", "dell", "hp", "lenovo", "asus", "acer", "logitech", "razer",
		"oracle", "ibm", "sun", "jetbrains", "docker", "kubernetes",
		"notepad++", "vlc", "7-zip", "git", "github",
		// часто встречающиеся
		"local", "locallow", "roaming",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}
