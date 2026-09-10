//go:build darwin

package cleaner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LaunchUninstaller удаляет программу способом, принятым в macOS.
//
// Как и в Windows-ядре, приоритет у деинсталлятора производителя: если
// внутри бандла (или рядом с ним) лежит собственный Uninstaller.app, он
// запускается через `open -W`, и подтверждение спрашивает он сам.
//
// Иначе:
//   - формулы и cask-и Homebrew удаляются штатным brew uninstall;
//   - обычный .app перемещается в Корзину — это и есть штатное удаление в
//     macOS, и, в отличие от rm -rf, оно обратимо: программу можно вернуть
//     из Корзины.
//
// Остатки в ~/Library (настройки, кеши) при этом не трогаются — для них
// есть --orphan-scan/--orphan-clean, которые показывают пути до удаления.
func LaunchUninstaller(displayName string) error {
	programs := GetInstalledPrograms(nil)
	var target *InstalledProgram
	for i := range programs {
		if strings.EqualFold(programs[i].DisplayName, displayName) {
			target = &programs[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("программа %q не найдена среди установленных", displayName)
	}

	if target.InstallLocation != "" {
		if uninstaller := findBundledUninstaller(target.InstallLocation); uninstaller != "" {
			if out, err := exec.Command("open", "-W", uninstaller).CombinedOutput(); err != nil {
				return fmt.Errorf("запуск деинсталлятора %q: %v: %s", uninstaller, err, strings.TrimSpace(string(out)))
			}
			return nil
		}
	}

	cmd := strings.Fields(target.UninstallString)
	switch {
	case len(cmd) >= 2 && cmd[0] == "brew":
		if _, err := exec.LookPath("brew"); err != nil {
			return fmt.Errorf("brew не найден в PATH: %w", err)
		}
		if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %v: %s", target.UninstallString, err, strings.TrimSpace(string(out)))
		}
		return nil

	case target.InstallLocation != "":
		if ok, reason := IsPathSafeToDelete(target.InstallLocation); !ok {
			return fmt.Errorf("удаление %q отменено: %s", displayName, reason)
		}
		if err := moveToTrash(target.InstallLocation, true); err != nil {
			return fmt.Errorf("перемещение %q в Корзину: %w", displayName, err)
		}
		return nil

	default:
		return fmt.Errorf("для %q не известен способ удаления", displayName)
	}
}

// findBundledUninstaller ищет собственный деинсталлятор программы: обычно
// это Uninstall*.app внутри бандла (Contents/Resources) или рядом с ним в
// той же папке.
func findBundledUninstaller(appPath string) string {
	dirs := []string{
		filepath.Join(appPath, "Contents", "Resources"),
		filepath.Dir(appPath),
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := strings.ToLower(e.Name())
			if !strings.HasSuffix(name, ".app") || !strings.HasPrefix(name, "uninstall") {
				continue
			}
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}
