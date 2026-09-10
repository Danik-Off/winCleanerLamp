//go:build linux

package cleaner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// LaunchUninstaller удаляет программу штатным средством её источника —
// пакетным менеджером, flatpak или snap (команда берётся из
// UninstallString, которую сформировал сам GetInstalledPrograms, а не из
// произвольной строки: команда собирается по фиксированному шаблону и
// запускается без оболочки, поэтому подставить в неё что-то постороннее
// нельзя).
//
// Отличие от Windows принципиальное, и его стоит держать в голове: там
// запускается интерфейс деинсталлятора производителя, и подтверждение
// спрашивает он. В Linux у пакетных менеджеров такого интерфейса нет, а
// работают они от root. Поэтому:
//   - системные пакеты (apt/dnf/pacman) запускаются через pkexec — графический
//     запрос пароля и есть точка подтверждения пользователем;
//   - если pkexec недоступен и мы не root, ничего не выполняется: возвращается
//     ошибка с точной командой, которую пользователь выполнит сам;
//   - flatpak и snap в пользовательской области выполняются напрямую.
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
	if target.UninstallString == "" {
		return fmt.Errorf("для %q не известен способ удаления (программа установлена не пакетным менеджером)", displayName)
	}

	args := strings.Fields(target.UninstallString)
	if len(args) < 2 {
		return fmt.Errorf("некорректная команда удаления для %q: %q", displayName, target.UninstallString)
	}
	if _, err := exec.LookPath(args[0]); err != nil {
		return fmt.Errorf("%s не найден в PATH: %w", args[0], err)
	}

	if !needsRootToUninstall(args[0]) || os.Geteuid() == 0 {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("удаление %q (%s): %v: %s", displayName, target.UninstallString, err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	if _, err := exec.LookPath("pkexec"); err != nil {
		return fmt.Errorf("удаление %q требует прав root; выполните вручную: sudo %s", displayName, target.UninstallString)
	}
	out, err := exec.Command("pkexec", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pkexec %s: %v: %s", target.UninstallString, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// needsRootToUninstall — команде нужен root. flatpak в пользовательской
// установке и так работает без него, snap и системные пакетные менеджеры —
// нет.
func needsRootToUninstall(tool string) bool {
	switch tool {
	case "flatpak":
		return false
	default:
		return true
	}
}
