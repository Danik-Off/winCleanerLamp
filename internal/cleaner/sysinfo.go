package cleaner

import (
	"os"
	"time"
)

// SysInfoEntry — запись об одном системном файле или каталоге.
type SysInfoEntry struct {
	Name   string
	Path   string
	Size   int64 // -1 = не удалось вычислить (таймаут / нет доступа)
	Hint   string
	IsDir  bool
	Exists bool
}

// pathSizeFast возвращает размер файла/каталога.
// Для файлов — мгновенный os.Lstat.
// Для каталогов — dirSize с таймаутом 3 секунды.
// Возвращает (size, exists). size=-1 если таймаут или ошибка доступа.
func pathSizeFast(p string, _ bool) (int64, bool) {
	info, err := os.Lstat(p)
	if err != nil {
		return 0, false
	}
	if !info.IsDir() {
		return info.Size(), true
	}

	// Для каталогов используем горутину с таймаутом
	type result struct{ size int64 }
	ch := make(chan result, 1)
	go func() {
		sz, _ := dirSize(p)
		ch <- result{sz}
	}()

	select {
	case r := <-ch:
		return r.size, true
	case <-time.After(3 * time.Second):
		return -1, true // таймаут — каталог существует, но размер не удалось вычислить быстро
	}
}
