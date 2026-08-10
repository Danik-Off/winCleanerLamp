//go:build !windows

package cleaner

import "errors"

// scheduleDeleteOnReboot — заглушка для не-Windows сборок (только чтобы
// пакет собирался кроссплатформенно, например для go vet/lint в CI).
// Приложение всё равно рассчитано только на Windows.
func scheduleDeleteOnReboot(_ string) error {
	return errors.New("отложенное удаление при перезагрузке поддерживается только на Windows")
}
