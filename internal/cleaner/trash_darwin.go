//go:build darwin

package cleaner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// trashDir — Корзина текущего пользователя (~/.Trash).
func trashDir() string { return sub(userHomeDir(), ".Trash") }

// moveToTrash перемещает файл или папку в ~/.Trash обычным переименованием.
//
// Осознанное отличие от Finder: «Положить обратно» для таких объектов
// работать не будет — исходный путь Finder хранит в собственных метаданных,
// а не в самом файле. Зато не нужен ни AppleScript, ни запущенный Finder
// (в ssh-сессии его может не быть вовсе), и файл остаётся на месте, откуда
// пользователь может забрать его вручную.
//
// Объекты с других томов в ~/.Trash не переезжают: там своя Корзина
// (/Volumes/<том>/.Trashes/<uid>), и перенос между томами — это копирование
// целиком. В этом случае возвращается понятная ошибка с подсказкой
// про --permanent.
func moveToTrash(path string, _ bool) error {
	dir := trashDir()
	if dir == "" {
		return fmt.Errorf("не удалось определить ~/.Trash (нет $HOME)")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("создание %s: %w", dir, err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	target := filepath.Join(dir, uniqueTrashName(dir, filepath.Base(abs)))
	if err := os.Rename(abs, target); err != nil {
		return fmt.Errorf("перемещение в Корзину %s: %w (другой том? используйте --permanent)", abs, err)
	}
	return nil
}

// uniqueTrashName подбирает свободное имя в Корзине.
func uniqueTrashName(dir, base string) string {
	name := base
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 1; ; i++ {
		if _, err := os.Lstat(filepath.Join(dir, name)); os.IsNotExist(err) {
			return name
		}
		name = fmt.Sprintf("%s %d%s", stem, i, ext)
	}
}
