//go:build windows

package cleaner

import (
	"fmt"
	"syscall"
	"unsafe"
)

// MOVEFILE_DELAY_UNTIL_REBOOT — стандартный флаг WinAPI MoveFileEx: файл
// удаляется не сейчас, а при следующей загрузке системы (до запуска
// большинства процессов, значит и до блокировки файла ими). Это тот же
// приём, которым десятилетиями пользуются установщики Windows для замены/
// удаления DLL и файлов, занятых другим процессом прямо сейчас — известный
// как "pending file rename operations" (реестр
// HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\PendingFileRenameOperations).
const movefileDelayUntilReboot = 0x4

var (
	modkernel32     = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileExW = modkernel32.NewProc("MoveFileExW")
)

// scheduleDeleteOnReboot просит Windows удалить path при следующей
// перезагрузке. Используется только как fallback, когда обычное удаление
// не удалось (файл занят другим процессом) — раньше такие файлы просто
// пропускались с ошибкой в отчёте, теперь пользователь получает реальный
// путь избавиться от них без необходимости искать и закрывать процесс вручную.
func scheduleDeleteOnReboot(path string) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("UTF16PtrFromString: %w", err)
	}
	r1, _, callErr := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(p)),
		0, // lpNewFileName = NULL — файл будет удалён, а не перемещён/переименован
		movefileDelayUntilReboot,
	)
	if r1 == 0 {
		return fmt.Errorf("MoveFileExW: %w", callErr)
	}
	return nil
}
