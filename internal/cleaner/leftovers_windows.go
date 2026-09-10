//go:build windows

package cleaner

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// GetInstalledPrograms возвращает список установленных программ из реестра.
func GetInstalledPrograms(orphanCfg *OrphanConfig) []InstalledProgram {
	orphanNames := make(map[string]bool)
	if orphanCfg != nil {
		for _, app := range orphanCfg.Apps {
			orphanNames[strings.ToLower(app.DisplayName)] = true
		}
	}

	keys := []string{
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	seen := make(map[string]bool)
	var programs []InstalledProgram
	for _, k := range keys {
		out, err := exec.Command("reg", "query", k, "/s").Output()
		if err != nil {
			continue
		}
		var curName, curPublisher, curLocation, curUninstall string
		flush := func() {
			if curName != "" && !seen[strings.ToLower(curName)] {
				seen[strings.ToLower(curName)] = true
				programs = append(programs, InstalledProgram{
					DisplayName:     curName,
					Publisher:       curPublisher,
					InstallLocation: curLocation,
					InOrphanDB:      orphanNames[strings.ToLower(curName)],
					UninstallString: curUninstall,
				})
			}
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				flush()
				curName, curPublisher, curLocation, curUninstall = "", "", "", ""
				continue
			}
			if strings.HasPrefix(line, "DisplayName") {
				if idx := strings.Index(line, "REG_SZ"); idx >= 0 {
					curName = strings.TrimSpace(line[idx+len("REG_SZ"):])
				}
			}
			if strings.HasPrefix(line, "Publisher") {
				if idx := strings.Index(line, "REG_SZ"); idx >= 0 {
					curPublisher = strings.TrimSpace(line[idx+len("REG_SZ"):])
				}
			}
			if strings.HasPrefix(line, "InstallLocation") {
				if idx := strings.Index(line, "REG_SZ"); idx >= 0 {
					curLocation = strings.TrimSpace(line[idx+len("REG_SZ"):])
				}
			}
			// UninstallString предпочтительнее QuietUninstallString: показывает
			// пользователю интерфейс деинсталлятора производителя вместо
			// молчаливого удаления без подтверждения.
			if strings.HasPrefix(line, "UninstallString") {
				if idx := strings.Index(line, "REG_SZ"); idx >= 0 {
					curUninstall = strings.TrimSpace(line[idx+len("REG_SZ"):])
				}
			}
		}
		flush() // последняя запись в выводе reg query не завершается пустой строкой
	}
	sort.Slice(programs, func(i, j int) bool {
		return strings.ToLower(programs[i].DisplayName) < strings.ToLower(programs[j].DisplayName)
	})
	return programs
}

// scanRegistryLeftovers — ищет ключи HKCU\Software, которые не соответствуют установленным программам.
func scanRegistryLeftovers(installed map[string]bool, whitelist map[string]bool) []LeftoverCandidate {
	var out []LeftoverCandidate
	regRoots := []string{
		`HKCU\Software`,
	}
	for _, regRoot := range regRoots {
		cmdOut, err := exec.Command("reg", "query", regRoot).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(cmdOut), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || !strings.Contains(strings.ToUpper(line), strings.ToUpper(regRoot)) {
				continue
			}
			parts := strings.Split(line, `\`)
			if len(parts) < 3 {
				continue
			}
			keyName := parts[len(parts)-1]
			low := strings.ToLower(strings.TrimSpace(keyName))
			if low == "" {
				continue
			}
			if whitelist[low] || registryWhitelist()[low] {
				continue
			}
			if matchesInstalled(keyName, installed) {
				continue
			}
			out = append(out, LeftoverCandidate{
				Path:   line,
				Size:   0,
				Files:  0,
				Reason: "ключ реестра без установленной программы",
				Type:   LeftoverRegistry,
			})
		}
	}
	return out
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

// installedProgramPaths читает InstallLocation из реестра Uninstall.
func installedProgramPaths() map[string]bool {
	keys := []string{
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	result := map[string]bool{}
	for _, k := range keys {
		out, err := exec.Command("reg", "query", k, "/s", "/v", "InstallLocation").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "InstallLocation") {
				continue
			}
			idx := strings.Index(line, "REG_SZ")
			if idx < 0 {
				continue
			}
			loc := strings.TrimSpace(line[idx+len("REG_SZ"):])
			if loc == "" {
				continue
			}
			result[strings.ToLower(filepath.Clean(loc))] = true
		}
	}
	return result
}

// knownSystemFolders — имена папок в AppData/ProgramData, которые не являются остатками программ.
func knownSystemFolders() map[string]bool {
	list := []string{
		// общие системные / встроенные
		"microsoft", "windows", "windowsapps", "packages", "temp", "tmp",
		"d3dscache", "connecteddevicesplatform", "comms", "crashdumps",
		"virtualstore", "programs", "application data", "history", "desktop",
		"downloaded installations", "downloads", "diagnosis", "publishers",
		".default", "default", "default user", "public",
		// ProgramData системные
		"ssh", "regid.1991-06.com.microsoft", "usoshared", "usoprivate",
		"placeholdertilelogofolder",
		"package cache", "softwaredistrribution", "windows defender",
		"windows security health", "windowsholographicdevices",
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

// programFilesWhitelist — папки в Program Files, которые точно системные.
func programFilesWhitelist() map[string]bool {
	list := []string{
		"common files", "internet explorer", "microsoft update health tools",
		"windows defender", "windows defender advanced threat protection",
		"windows mail", "windows media player", "windows multimedia platform",
		"windows nt", "windows photo viewer", "windows portable devices",
		"windows security", "windows sidebar", "windowsapps",
		"windowspowershell", "microsoft.net", "msbuild", "reference assemblies",
		"dotnet", "iis", "iis express", "microsoft sdks", "microsoft sql server",
		"microsoft visual studio", "uninstall information",
		"windows kits", "nvidia corporation", "realtek", "intel",
		"amd", "dell", "hp", "lenovo",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// registryWhitelist — ключи HKCU\Software, которые всегда присутствуют.
func registryWhitelist() map[string]bool {
	list := []string{
		"microsoft", "classes", "policies", "registeredapplications",
		"wine", "wow6432node", "defaultuserext", "im providers",
		"appdata", "intel", "nvidia corporation", "realtek",
		"khronos", "opengl", "amd", "google", "mozilla",
		"adobe", "apple", "apple computer, inc.", "apple inc.",
		"java", "javafx", "javasoft", "oracle",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// ─── Платформенные корни и тексты (см. leftovers.go) ───

// leftoverUserDataRoots — каталоги, где Windows-программы хранят свои данные.
func leftoverUserDataRoots() []string {
	roots := []string{
		ExpandPath("%APPDATA%"),
		ExpandPath("%LOCALAPPDATA%"),
	}
	if pd := ExpandPath("%PROGRAMDATA%"); pd != "" {
		roots = append(roots, pd)
	}
	return nonEmpty(roots)
}

// leftoverProgramRoots — каталоги установки программ.
func leftoverProgramRoots() []string {
	return nonEmpty([]string{
		ExpandPath("%PROGRAMFILES%"),
		ExpandPath("%PROGRAMFILES(X86)%"),
	})
}

// LeftoverScanDescription — что именно осматривает --leftovers.
func LeftoverScanDescription() string {
	return `AppData, ProgramData, Program Files, реестр HKCU\Software`
}

// LeftoverRegistrySectionTitle и LeftoverRegistryTag — подписи для находок
// типа LeftoverRegistry (в Windows это ключи реестра).
func LeftoverRegistrySectionTitle() string {
	return "Ключи реестра без программ"
}

func LeftoverRegistryTag() string { return "реестр" }

// LeftoverRemovalHints — как удалить найденное вручную.
func LeftoverRemovalHints() []string {
	return []string{
		"Для удаления папок используйте GUI или Проводник / rmdir /s.",
		"Для реестра: regedit или reg delete <ключ>.",
	}
}
