//go:build linux

package cleaner

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// В Linux нет единого реестра установленных программ, поэтому список
// собирается из всех источников, которые есть в системе: системный пакетный
// менеджер (dpkg/rpm/pacman/apk), flatpak, snap и .desktop-файлы приложений,
// установленных мимо пакетного менеджера (AppImage, ручная распаковка в
// /opt). Дубликаты по имени убираются.

// packageSource — один источник списка программ.
type packageSource struct {
	tool string
	args []string
	// parse разбирает строку вывода в запись; ok=false — строку пропустить.
	parse func(line string) (InstalledProgram, bool)
}

func packageSources() []packageSource {
	return []packageSource{
		{
			tool: "dpkg-query",
			args: []string{"-W", "-f", "${Package}\t${Maintainer}\t${db:Status-Status}\n"},
			parse: func(line string) (InstalledProgram, bool) {
				f := strings.Split(line, "\t")
				if len(f) < 3 || f[2] != "installed" {
					return InstalledProgram{}, false
				}
				return InstalledProgram{
					DisplayName:     f[0],
					Publisher:       f[1],
					UninstallString: "apt-get remove " + f[0],
				}, true
			},
		},
		{
			tool: "rpm",
			args: []string{"-qa", "--qf", "%{NAME}\t%{VENDOR}\n"},
			parse: func(line string) (InstalledProgram, bool) {
				f := strings.Split(line, "\t")
				if len(f) < 1 || f[0] == "" {
					return InstalledProgram{}, false
				}
				p := InstalledProgram{DisplayName: f[0], UninstallString: "dnf remove " + f[0]}
				if len(f) > 1 {
					p.Publisher = f[1]
				}
				return p, true
			},
		},
		{
			tool: "pacman",
			args: []string{"-Qq"},
			parse: func(line string) (InstalledProgram, bool) {
				name := strings.TrimSpace(line)
				if name == "" {
					return InstalledProgram{}, false
				}
				return InstalledProgram{
					DisplayName:     name,
					UninstallString: "pacman -Rns " + name,
				}, true
			},
		},
		{
			tool: "flatpak",
			args: []string{"list", "--app", "--columns=application,name,origin"},
			parse: func(line string) (InstalledProgram, bool) {
				f := strings.Split(line, "\t")
				if len(f) < 2 || f[0] == "" {
					return InstalledProgram{}, false
				}
				p := InstalledProgram{
					DisplayName:     f[1],
					UninstallString: "flatpak uninstall " + f[0],
				}
				if len(f) > 2 {
					p.Publisher = f[2]
				}
				// Данные flatpak-приложения лежат в ~/.var/app/<id>.
				p.InstallLocation = sub(userHomeDir(), ".var/app/"+f[0])
				return p, true
			},
		},
	}
}

// GetInstalledPrograms возвращает список установленных программ из всех
// доступных источников.
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

	for _, src := range packageSources() {
		if _, err := exec.LookPath(src.tool); err != nil {
			continue
		}
		out, err := exec.Command(src.tool, src.args...).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			if p, ok := src.parse(line); ok {
				add(p)
			}
		}
	}

	for _, p := range snapPrograms() {
		add(p)
	}
	for _, p := range desktopPrograms() {
		add(p)
	}

	sort.Slice(programs, func(i, j int) bool {
		return strings.ToLower(programs[i].DisplayName) < strings.ToLower(programs[j].DisplayName)
	})
	return programs
}

// snapPrograms разбирает табличный вывод `snap list` (у snap нет формата
// вывода с разделителями, поэтому колонки берутся по пробелам).
func snapPrograms() []InstalledProgram {
	if _, err := exec.LookPath("snap"); err != nil {
		return nil
	}
	out, err := exec.Command("snap", "list", "--unicode=never", "--color=never").Output()
	if err != nil {
		return nil
	}
	var programs []InstalledProgram
	for i, line := range strings.Split(string(out), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 5 {
			continue
		}
		programs = append(programs, InstalledProgram{
			DisplayName:     f[0],
			Publisher:       f[4],
			InstallLocation: filepath.Join("/snap", f[0]),
			UninstallString: "snap remove " + f[0],
		})
	}
	return programs
}

// desktopApplicationDirs — каталоги с ярлыками приложений (XDG).
func desktopApplicationDirs() []string {
	dirs := []string{
		sub(userDataDir(), "applications"),
		"/usr/share/applications",
		"/usr/local/share/applications",
		"/var/lib/flatpak/exports/share/applications",
		sub(userDataDir(), "flatpak/exports/share/applications"),
	}
	return nonEmpty(dirs)
}

// desktopPrograms — приложения, у которых есть ярлык, но нет записи в
// пакетном менеджере (AppImage, распакованные в /opt сборки). Именно они
// чаще всего и оставляют после себя каталоги-«остатки».
func desktopPrograms() []InstalledProgram {
	var programs []InstalledProgram
	for _, dir := range desktopApplicationDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".desktop") {
				continue
			}
			keys := readDesktopEntry(filepath.Join(dir, e.Name()))
			if keys["Name"] == "" || strings.EqualFold(keys["NoDisplay"], "true") {
				continue
			}
			p := InstalledProgram{DisplayName: keys["Name"]}
			if exe := desktopExecPath(keys); filepath.IsAbs(exe) {
				p.InstallLocation = filepath.Dir(exe)
			}
			programs = append(programs, p)
		}
	}
	return programs
}

// desktopExecPath — путь к исполняемому файлу из ключей TryExec/Exec, без
// аргументов и подстановок вида %U/%F.
func desktopExecPath(keys map[string]string) string {
	if try := splitDesktopExec(keys["TryExec"]); len(try) > 0 {
		return try[0]
	}
	for _, arg := range splitDesktopExec(keys["Exec"]) {
		// env VAR=value program — пропускаем обёртку и присваивания.
		if arg == "env" || strings.Contains(arg, "=") {
			continue
		}
		if strings.HasPrefix(arg, "%") {
			continue
		}
		return arg
	}
	return ""
}

// splitDesktopExec разбирает значение ключа Exec на аргументы по правилам
// Desktop Entry Specification: аргумент с пробелом или другим особым
// символом заключается в двойные кавычки, а внутри них символы " ` $ \
// экранируются обратным слэшем.
//
// strings.Fields здесь не годится: путь вида "/opt/My App/app" она разорвала
// бы по пробелу, и проверка «программа существует» всегда давала бы ложное
// «не найдена» — ярлык попадал бы в битые, а его каталог в остатки.
func splitDesktopExec(line string) []string {
	var args []string
	var current strings.Builder
	inQuotes, hasToken := false, false

	flush := func() {
		if hasToken {
			args = append(args, current.String())
			current.Reset()
			hasToken = false
		}
	}

	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case inQuotes && c == '\\' && i+1 < len(runes):
			i++
			current.WriteRune(runes[i])
			hasToken = true
		case c == '"':
			inQuotes = !inQuotes
			hasToken = true // "" — это пустой аргумент, а не отсутствие его
		case !inQuotes && (c == ' ' || c == '\t'):
			flush()
		default:
			current.WriteRune(c)
			hasToken = true
		}
	}
	flush()
	return args
}

// installedProgramNames — токены имён установленных программ для эвристики
// «папка похожа на установленную программу».
func installedProgramNames() map[string]bool {
	result := map[string]bool{}
	for _, p := range GetInstalledPrograms(nil) {
		for _, tok := range tokenize(p.DisplayName) {
			result[tok] = true
		}
	}
	// Имена бинарников в PATH: каталог ~/.config/htop принадлежит htop, даже
	// если пакет называется иначе.
	for _, dir := range []string{"/usr/bin", "/usr/local/bin", sub(userHomeDir(), ".local/bin")} {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := strings.ToLower(e.Name())
			if len(name) >= 3 {
				result[name] = true
			}
		}
	}
	return result
}

// installedProgramPaths — каталоги установки известных программ.
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

// scanRegistryLeftovers — аналог поиска осиротевших ключей реестра: ярлыки
// .desktop и записи автозагрузки, чья программа больше не установлена
// (TryExec/Exec указывает в никуда). В Windows-ядре это ключи HKCU\Software,
// здесь — файлы, но роль та же: запись осталась, программы нет.
func scanRegistryLeftovers(_ map[string]bool, whitelist map[string]bool) []LeftoverCandidate {
	var out []LeftoverCandidate
	known := registryWhitelist()
	dirs := append(desktopApplicationDirs(), nonEmpty([]string{userAutostartDir()})...)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".desktop") {
				continue
			}
			// Обёртки самих окружений рабочего стола запускают программу
			// через посредника, и «несуществующая цель» у них — норма.
			base := strings.ToLower(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
			if known[base] || whitelist[base] {
				continue
			}
			full := filepath.Join(dir, e.Name())
			keys := readDesktopEntry(full)
			target := desktopExecPath(keys)
			if target == "" || desktopTargetExists(target) {
				continue
			}
			out = append(out, LeftoverCandidate{
				Path:   full,
				Reason: "ярлык ссылается на несуществующую программу: " + target,
				Type:   LeftoverRegistry,
			})
		}
	}
	return out
}

// desktopTargetExists — цель ярлыка существует: либо это абсолютный путь,
// либо команда, найденная в PATH.
func desktopTargetExists(target string) bool {
	if strings.ContainsRune(target, filepath.Separator) {
		_, err := os.Stat(target)
		return err == nil
	}
	_, err := exec.LookPath(target)
	return err == nil
}

// knownSystemFolders — каталоги в ~/.config и ~/.local/share, которые не
// являются остатками программ.
func knownSystemFolders() map[string]bool {
	list := []string{
		// XDG и окружения рабочего стола
		"applications", "autostart", "icons", "fonts", "themes", "mime",
		"desktop-directories", "user-dirs.dirs", "user-dirs.locale",
		"dconf", "gtk-2.0", "gtk-3.0", "gtk-4.0", "qt5ct", "qt6ct",
		"gnome", "gnome-control-center", "gnome-session", "kde", "kdeglobals",
		"plasma", "xfce4", "cinnamon", "mate", "lxqt", "i3", "sway", "hypr",
		"systemd", "pulse", "pipewire", "wireplumber", "environment.d",
		"keyrings", "gnupg", "ssh", "nautilus", "dolphin", "evolution",
		"trash", "recently-used.xbel", "flatpak", "glib-2.0", "menus",
		"ibus", "fcitx", "fcitx5", "xdg-desktop-portal", "session",
		// прочее общее
		"bin", "lib", "share", "state", "cache", "backgrounds", "sounds",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// programFilesWhitelist — подкаталоги /opt и /usr/local, которые точно не
// остатки программ.
func programFilesWhitelist() map[string]bool {
	list := []string{
		"bin", "sbin", "lib", "lib64", "libexec", "include", "share",
		"src", "etc", "games", "man", "doc", "containerd", "systemd",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// registryWhitelist — .desktop-ярлыки, которые не считаем остатками, даже
// если цель не нашлась (обёртки и записи самих окружений).
func registryWhitelist() map[string]bool {
	list := []string{
		"xdg-desktop-portal", "im-launch", "gnome-keyring", "pulseaudio",
		"orca-autostart", "at-spi-dbus-bus", "user-dirs-update-gtk",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

// ─── Платформенные корни и тексты (см. leftovers.go) ───

// leftoverUserDataRoots — каталоги, где программы хранят настройки и данные.
func leftoverUserDataRoots() []string {
	return nonEmpty([]string{
		userConfigDir(),
		userDataDir(),
		sub(userHomeDir(), ".var/app"),
	})
}

// leftoverProgramRoots — каталоги установки программ мимо пакетного
// менеджера. /usr сюда намеренно не входит: там всё принадлежит пакетам.
func leftoverProgramRoots() []string {
	return []string{"/opt"}
}

// LeftoverScanDescription — что именно осматривает --leftovers.
func LeftoverScanDescription() string {
	return "~/.config, ~/.local/share, ~/.var/app, /opt, ярлыки .desktop"
}

// LeftoverRegistrySectionTitle и LeftoverRegistryTag — подписи для находок
// типа LeftoverRegistry (в Linux это .desktop-ярлыки без программы).
func LeftoverRegistrySectionTitle() string { return "Ярлыки .desktop без программы" }

func LeftoverRegistryTag() string { return "ярлык" }

// LeftoverRemovalHints — как удалить найденное вручную.
func LeftoverRemovalHints() []string {
	return []string{
		"Для удаления папок используйте GUI или rm -rf (перед этим проверьте путь).",
		"Ярлыки .desktop удаляются как обычные файлы; системные — только с sudo.",
	}
}
