//go:build windows

package cleaner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func registryKeyExists(key string) bool {
	err := exec.Command("reg", "query", key, "/ve").Run()
	if err != nil {
		// Пробуем без /ve — может быть просто ключ без значения по умолчанию
		err = exec.Command("reg", "query", key).Run()
	}
	return err == nil
}

// DefaultDiscoverRoots — корневые папки по умолчанию.
func DefaultDiscoverRoots() []string {
	roots := []string{
		`C:\Program Files`,
		`C:\Program Files (x86)`,
		`C:\ProgramData`,
	}
	if appdata := ExpandPath(`%APPDATA%`); appdata != "" {
		roots = append(roots, appdata)
	}
	if localAppdata := ExpandPath(`%LOCALAPPDATA%`); localAppdata != "" {
		roots = append(roots, localAppdata)
	}
	return roots
}

// dirHasExecutable проверяет, содержит ли папка .exe файлы (до 2 уровней вглубь).
func dirHasExecutable(root string) bool {
	depth := 0
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if found {
			return filepath.SkipAll
		}
		// Ограничиваем глубину
		rel, _ := filepath.Rel(root, path)
		depth = strings.Count(rel, string(filepath.Separator))
		if d.IsDir() && depth > 2 {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".exe") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// discoverWhitelist — папки, которые точно не мусор (системные/вендорские).
func discoverWhitelist() map[string]bool {
	list := []string{
		// Windows / системные
		"windowsapps", "microsoft", "common files", "internet explorer",
		"windows defender", "windows defender advanced threat protection",
		"windows mail", "windows media player", "windows multimedia platform",
		"windows nt", "windows photo viewer", "windows portable devices",
		"windows security", "windows sidebar", "windowspowershell",
		"microsoft.net", "msbuild", "reference assemblies", "dotnet",
		"iis", "iis express", "microsoft sdks", "microsoft sql server",
		"microsoft visual studio", "uninstall information", "windows kits",
		"package cache", "softwaredistrribution", "microsoft update health tools",
		// Вендоры
		"nvidia corporation", "nvidia", "realtek", "intel", "amd",
		"dell", "hp", "lenovo", "asus", "acer", "logitech", "razer",
		"google", "mozilla", "adobe", "apple", "oracle", "jetbrains",
		// AppData системные
		"microsoft", "windows", "packages", "temp", "tmp",
		"d3dscache", "connecteddevicesplatform", "comms", "crashdumps",
		"virtualstore", "programs", "application data", "history",
		"desktop", "downloads", "diagnosis", "publishers",
		".default", "default", "default user", "public",
		"ssh", "regid.1991-06.com.microsoft", "usoshared", "usoprivate",
		"windowsholographicdevices", "placeholdertilelogofolder",
		"local", "locallow", "roaming",
	}
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}
