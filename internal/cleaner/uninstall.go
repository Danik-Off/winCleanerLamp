package cleaner

import (
	"fmt"
	"os/exec"
	"strings"
)

// LaunchUninstaller запускает официальный деинсталлятор указанной программы
// (UninstallString из реестра Uninstall — та же команда, которую использует
// "Программы и компоненты" в Панели управления) и ждёт его завершения.
// Никаких флагов тихого удаления не добавляется: пользователь видит и
// подтверждает удаление в интерфейсе, который предоставляет сам
// производитель программы — это лишь запуск того же деинсталлятора, а не
// автоматическое молчаливое удаление.
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
		return fmt.Errorf("для %q не найдена команда удаления (UninstallString)", displayName)
	}

	file, argsStr := splitCommandLine(target.UninstallString)
	cmd := exec.Command(file, splitArgs(argsStr)...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("запуск деинсталлятора %q: %w", displayName, err)
	}
	return nil
}

// splitCommandLine разбивает UninstallString на исполняемый файл и
// остальные аргументы одной строкой. UninstallString почти всегда одного из
// двух видов: `"C:\Path With Spaces\uninst.exe" /S` (путь в кавычках) или
// `MsiExec.exe /X{GUID}` (без пробелов в имени) — оба случая покрыты.
func splitCommandLine(s string) (file, args string) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, `"`) {
		if end := strings.Index(s[1:], `"`); end >= 0 {
			return s[1 : end+1], strings.TrimSpace(s[end+2:])
		}
	}
	if idx := strings.IndexByte(s, ' '); idx >= 0 {
		return s[:idx], strings.TrimSpace(s[idx+1:])
	}
	return s, ""
}

// splitArgs — минимальный, учитывающий кавычки разбор аргументов командной
// строки (аналог того, что делает CommandLineToArgvW). Передаётся напрямую в
// exec.Command, что даёт корректное CreateProcess-совместимое экранирование
// без хождения через cmd.exe (у которого свои, несовместимые правила
// разбора кавычек) — иначе легко сломать пути с пробелами/кавычками внутри.
func splitArgs(s string) []string {
	var args []string
	var cur strings.Builder
	inQuotes := false
	for _, r := range s {
		switch {
		case r == '"':
			inQuotes = !inQuotes
		case r == ' ' && !inQuotes:
			if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		args = append(args, cur.String())
	}
	return args
}
