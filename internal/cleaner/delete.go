package cleaner

import (
	"fmt"
	"os"
)

// DeleteResult — результат безопасного удаления одного файла/папки.
// Используется CLI-командами --delete-path/--delete-dir и их JSON-выводом,
// а также OrphanCleaner-ом — единый формат вместо трёх разных ad-hoc
// реализаций, которые раньше жили в cleaner.go, emptydirs.go и
// gui/electron/main.ts.
type DeleteResult struct {
	Path              string `json:"path"`
	Success           bool   `json:"success"`
	MovedToRecycleBin bool   `json:"movedToRecycleBin"`
	// ScheduledForReboot — файл занят другим процессом прямо сейчас, обычное
	// удаление и перемещение в Корзину не удались, но удаление запланировано
	// на следующую перезагрузку (см. scheduleDeleteOnReboot).
	ScheduledForReboot bool   `json:"scheduledForReboot,omitempty"`
	Error              string `json:"error,omitempty"`
}

// DeleteFile безопасно удаляет один файл (не папку).
// По умолчанию (permanent=false) файл перемещается в Корзину.
func DeleteFile(path string, permanent bool) DeleteResult {
	r := DeleteResult{Path: path}

	ok, reason := IsPathSafeToDelete(path)
	if !ok {
		r.Error = reason
		return r
	}

	info, err := os.Lstat(path)
	if err != nil {
		r.Error = fmt.Sprintf("файл не найден: %v", err)
		return r
	}
	if info.IsDir() {
		r.Error = "путь является папкой, используйте --delete-dir"
		return r
	}

	if permanent {
		if err := os.Remove(path); err != nil {
			if rebootErr := scheduleDeleteOnReboot(path); rebootErr == nil {
				r.Success = true
				r.ScheduledForReboot = true
				return r
			}
			r.Error = err.Error()
			return r
		}
		r.Success = true
		return r
	}

	if err := moveToTrash(path, false); err != nil {
		// Файл, скорее всего, занят другим процессом — вместо того чтобы
		// просто вернуть ошибку, планируем удаление на следующую
		// перезагрузку (тот же приём, что используют установщики Windows).
		if rebootErr := scheduleDeleteOnReboot(path); rebootErr == nil {
			r.Success = true
			r.ScheduledForReboot = true
			return r
		}
		r.Error = err.Error()
		return r
	}
	r.Success = true
	r.MovedToRecycleBin = true
	return r
}

// DeleteDir безопасно удаляет папку целиком (вместе с содержимым).
// По умолчанию (permanent=false) папка перемещается в Корзину.
func DeleteDir(path string, permanent bool) DeleteResult {
	r := DeleteResult{Path: path}

	ok, reason := IsPathSafeToDelete(path)
	if !ok {
		r.Error = reason
		return r
	}

	info, err := os.Lstat(path)
	if err != nil {
		r.Error = fmt.Sprintf("папка не найдена: %v", err)
		return r
	}
	if !info.IsDir() {
		r.Error = "путь не является папкой, используйте --delete-path"
		return r
	}

	if permanent {
		if err := os.RemoveAll(path); err != nil {
			r.Error = err.Error()
			return r
		}
		r.Success = true
		return r
	}

	if err := moveToTrash(path, true); err != nil {
		r.Error = err.Error()
		return r
	}
	r.Success = true
	r.MovedToRecycleBin = true
	return r
}
