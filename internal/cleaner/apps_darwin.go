//go:build darwin

package cleaner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// В macOS «установленная программа» — это, как правило, .app-бандл в
// /Applications или ~/Applications; плюс к этому Homebrew ведёт собственный
// список формул и cask-ов. Реестра, как в Windows, нет, поэтому список
// собирается из этих источников.

// applicationDirs — где лежат приложения пользователя.
// /System/Applications намеренно не входит: это часть самой macOS.
func applicationDirs() []string {
	return nonEmpty([]string{
		"/Applications",
		"/Applications/Utilities",
		sub(userHomeDir(), "Applications"),
	})
}

// GetInstalledPrograms возвращает список установленных программ:
// .app-бандлы плюс формулы и cask-и Homebrew.
func GetInstalledPrograms(orphanCfg *OrphanConfig) []InstalledProgram {
	orphanNames := make(map[string]bool)
	if orphanCfg != nil {
		for _, app := range orphanCfg.Apps {
			orphanNames[strings.ToLower(app.DisplayName)] = true
		}
	}

	seen := make(map[string]bool)
	var programs []InstalledProgram
	add := func(p InstalledProgram) {
		key := strings.ToLower(p.DisplayName)
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		p.InOrphanDB = orphanNames[key]
		programs = append(programs, p)
	}

	casks := brewList("--cask")
	for _, token := range casks {
		add(InstalledProgram{
			DisplayName:     token,
			Publisher:       "Homebrew Cask",
			UninstallString: "brew uninstall --cask " + token,
		})
	}
	for _, formula := range brewList("--formula") {
		add(InstalledProgram{
			DisplayName:     formula,
			Publisher:       "Homebrew",
			UninstallString: "brew uninstall " + formula,
		})
	}

	for _, dir := range applicationDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.EqualFold(filepath.Ext(e.Name()), ".app") {
				continue
			}
			full := filepath.Join(dir, e.Name())
			name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			p := InstalledProgram{
				DisplayName:     name,
				Publisher:       appBundleIdentifier(full),
				InstallLocation: full,
				// Штатный способ удалить .app в macOS — переместить его в
				// Корзину; отдельного деинсталлятора у большинства программ
				// нет (см. LaunchUninstaller).
				UninstallString: "trash " + full,
			}
			add(p)
		}
	}

	sort.Slice(programs, func(i, j int) bool {
		return strings.ToLower(programs[i].DisplayName) < strings.ToLower(programs[j].DisplayName)
	})
	return programs
}

// brewList — список установленных формул или cask-ов.
func brewList(kind string) []string {
	if _, err := exec.LookPath("brew"); err != nil {
		return nil
	}
	out, err := exec.Command("brew", "list", kind, "-1").Output()
	if err != nil {
		return nil
	}
	var result []string
	for _, line := range strings.Split(string(out), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			result = append(result, name)
		}
	}
	return result
}

// appBundleIdentifier — CFBundleIdentifier из Info.plist бандла (роль
// «издателя»: com.apple.Safari, com.google.Chrome).
func appBundleIdentifier(appPath string) string {
	plist := filepath.Join(appPath, "Contents", "Info.plist")
	out, err := exec.Command("plutil", "-convert", "json", "-o", "-", plist).Output()
	if err != nil {
		return ""
	}
	var parsed struct {
		Identifier string `json:"CFBundleIdentifier"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return ""
	}
	return parsed.Identifier
}

// appBundleExecutable — путь к исполняемому файлу внутри бандла (нужен для
// поиска битых .app, см. shortcuts_darwin.go).
func appBundleExecutable(appPath string) string {
	plist := filepath.Join(appPath, "Contents", "Info.plist")
	out, err := exec.Command("plutil", "-convert", "json", "-o", "-", plist).Output()
	if err != nil {
		return ""
	}
	var parsed struct {
		Executable string `json:"CFBundleExecutable"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil || parsed.Executable == "" {
		return ""
	}
	return filepath.Join(appPath, "Contents", "MacOS", parsed.Executable)
}

// installedProgramNames — токены имён установленных программ и обратных
// доменных имён их бандлов: каталоги в ~/Library/Caches называются как раз
// по bundle id (com.spotify.client), а не по имени приложения.
func installedProgramNames() map[string]bool {
	result := map[string]bool{}
	for _, p := range GetInstalledPrograms(nil) {
		for _, tok := range tokenize(p.DisplayName) {
			result[tok] = true
		}
		if p.Publisher != "" {
			for _, tok := range tokenize(p.Publisher) {
				result[tok] = true
			}
			result[strings.ToLower(p.Publisher)] = true
		}
	}
	return result
}

// installedProgramPaths — каталоги установленных .app-бандлов.
func installedProgramPaths() map[string]bool {
	result := map[string]bool{}
	for _, p := range GetInstalledPrograms(nil) {
		if p.InstallLocation == "" {
			continue
		}
		result[strings.ToLower(filepath.Clean(p.InstallLocation))] = true
	}
	return result
}

// scanRegistryLeftovers — аналог осиротевших ключей реестра: plist-файлы
// launchd, чья программа больше не существует. Такие агенты остаются после
// удаления приложения и продолжают безрезультатно запускаться при каждом
// входе в систему.
func scanRegistryLeftovers(_ map[string]bool, whitelist map[string]bool) []LeftoverCandidate {
	var out []LeftoverCandidate
	known := registryWhitelist()
	for _, spec := range launchAgentDirs() {
		if spec.Dir == "" {
			continue
		}
		entries, err := os.ReadDir(spec.Dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".plist") {
				continue
			}
			full := filepath.Join(spec.Dir, e.Name())
			label, program := readLaunchdPlist(full)
			if program == "" || launchdProgramExists(program) {
				continue
			}
			// Служебные агенты самой системы и Homebrew запускают программу
			// через посредника — «несуществующая цель» у них ни о чём не
			// говорит.
			if low := strings.ToLower(label); known[low] || whitelist[low] {
				continue
			}
			out = append(out, LeftoverCandidate{
				Path:   full,
				Reason: "агент launchd ссылается на несуществующую программу: " + program,
				Type:   LeftoverRegistry,
			})
		}
	}
	return out
}

func launchdProgramExists(program string) bool {
	if strings.HasPrefix(program, "/") {
		_, err := os.Stat(program)
		return err == nil
	}
	_, err := exec.LookPath(program)
	return err == nil
}

// knownSystemFolders — каталоги в ~/Library и /Library, которые не являются
// остатками программ.
func knownSystemFolders() map[string]bool {
	list := []string{
		"application support", "caches", "containers", "group containers",
		"preferences", "logs", "launchagents", "launchdaemons",
		"cookies", "keychains", "fonts", "colorsync", "audio",
		"internet plug-ins", "input methods", "services", "screen savers",
		"safari", "mail", "messages", "calendars", "reminders", "assistant",
		"mobile documents", "icloud", "spelling", "webkit", "suggestions",
		"accounts", "autosave information", "coredata", "cloudstorage",
		"apple", "com.apple.bird", "developer", "printers", "widgets",
		"metadata", "sharing", "shortcuts", "sounds", "staging",
		// общие для unix-домашнего каталога
		"bin", "lib", "share", ".cache", ".config", ".local",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// programFilesWhitelist — подкаталоги /Applications, которые не являются
// остатками (папки-группировки самой системы).
func programFilesWhitelist() map[string]bool {
	list := []string{"utilities", "demos", "safari.app", "xcode.app"}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// registryWhitelist — plist-агенты, которые не считаем остатками.
func registryWhitelist() map[string]bool {
	list := []string{
		"com.apple.installer.cleanupinstaller", "com.openssh.ssh-agent",
		"homebrew.mxcl.autoupdate",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// ─── Платформенные корни и тексты (см. leftovers.go) ───

// leftoverUserDataRoots — где программы хранят настройки и данные.
func leftoverUserDataRoots() []string {
	return nonEmpty([]string{
		userAppSupportDir(),
		sub(userLibraryDir(), "Preferences"),
		sub(userLibraryDir(), "Containers"),
		sub(userLibraryDir(), "Saved Application State"),
	})
}

// leftoverProgramRoots — каталоги установки программ.
func leftoverProgramRoots() []string { return applicationDirs() }

// LeftoverScanDescription — что именно осматривает --leftovers.
func LeftoverScanDescription() string {
	return "~/Library/Application Support, Preferences, Containers, /Applications, агенты launchd"
}

// LeftoverRegistrySectionTitle и LeftoverRegistryTag — подписи для находок
// типа LeftoverRegistry (в macOS это plist-агенты launchd).
func LeftoverRegistrySectionTitle() string { return "Агенты launchd без программы" }

func LeftoverRegistryTag() string { return "launchd" }

// LeftoverRemovalHints — как удалить найденное вручную.
func LeftoverRemovalHints() []string {
	return []string{
		"Папки удобнее удалять через Finder (Cmd+Delete — в Корзину, обратимо).",
		"Перед удалением plist сначала выгрузите агент: launchctl bootout gui/$UID/<label>.",
	}
}
