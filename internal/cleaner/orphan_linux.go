//go:build linux

package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// registryKeyExists — в Linux реестра нет. Поле registryKeys в
// orphaned_apps.json заполнено windows-путями, поэтому здесь всегда false:
// иначе Linux-ядро сообщало бы о «найденных» ключах, которых не существует.
func registryKeyExists(_ string) bool { return false }

// DefaultDiscoverRoots — где искать неизвестные каталоги: пользовательские
// каталоги настроек/данных/кеша и /opt (туда ставятся программы мимо
// пакетного менеджера — именно они и оставляют мусор).
func DefaultDiscoverRoots() []string {
	return nonEmpty([]string{
		userConfigDir(),
		userDataDir(),
		userCacheDir(),
		sub(userHomeDir(), ".var/app"),
		"/opt",
	})
}

// dirHasExecutable проверяет, есть ли в каталоге исполняемый файл (до 2
// уровней вглубь). В Linux исполняемость — это бит x, а не расширение, как
// .exe в Windows.
func dirHasExecutable(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if found {
			return filepath.SkipAll
		}
		rel, _ := filepath.Rel(root, path)
		depth := strings.Count(rel, string(filepath.Separator))
		if d.IsDir() {
			if depth > 2 {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		if info.Mode().Perm()&0o111 != 0 || isLinuxAppBundleFile(d.Name()) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// isLinuxAppBundleFile — файлы, которые означают «здесь установлена
// программа», даже если бит x снят (например, после копирования с
// FAT-раздела).
func isLinuxAppBundleFile(name string) bool {
	low := strings.ToLower(name)
	for _, ext := range []string{".appimage", ".so", ".jar"} {
		if strings.HasSuffix(low, ext) {
			return true
		}
	}
	return false
}

// discoverWhitelist — каталоги, которые точно не мусор: служебные каталоги
// XDG, окружений рабочего стола и системных служб.
func discoverWhitelist() map[string]bool {
	out := knownSystemFolders()
	for _, s := range []string{
		"containerd", "docker", "kubernetes", "snapd", "flatpak",
		"google", "mozilla", "microsoft", "jetbrains", "nvidia",
		"lost+found", "chrome", "chromium",
	} {
		out[s] = true
	}
	for s := range programFilesWhitelist() {
		out[s] = true
	}
	return out
}
