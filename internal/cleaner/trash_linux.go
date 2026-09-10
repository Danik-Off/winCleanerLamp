//go:build linux

package cleaner

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// trashDir — каталог Корзины текущего пользователя по FreeDesktop Trash
// Specification: $XDG_DATA_HOME/Trash (обычно ~/.local/share/Trash).
func trashDir() string {
	data := userDataDir()
	if data == "" {
		return ""
	}
	return filepath.Join(data, "Trash")
}

// moveToTrash перемещает файл или папку в Корзину по FreeDesktop Trash
// Specification, а не вызывает gio/kioclient: внешняя утилита есть не в
// каждом окружении (и её нет вовсе на сервере без рабочего стола), а формат
// самой Корзины простой и стабильный — файл переносится в Trash/files, а
// рядом, в Trash/info, кладётся <имя>.trashinfo с исходным путём и датой,
// чтобы файловый менеджер умел восстановить файл на место.
//
// Спецификация допускает Корзину только в пределах той же файловой системы:
// если файл лежит на другом разделе (внешний диск, отдельный /home),
// os.Rename вернёт EXDEV. Копировать гигабайты ради «удаления» смысла нет —
// возвращаем понятную ошибку с подсказкой про --permanent.
func moveToTrash(path string, _ bool) error {
	dir := trashDir()
	if dir == "" {
		return fmt.Errorf("не удалось определить каталог Корзины (нет $HOME/$XDG_DATA_HOME)")
	}
	filesDir := filepath.Join(dir, "files")
	infoDir := filepath.Join(dir, "info")
	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		return fmt.Errorf("создание %s: %w", filesDir, err)
	}
	if err := os.MkdirAll(infoDir, 0o700); err != nil {
		return fmt.Errorf("создание %s: %w", infoDir, err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	name := uniqueTrashName(filesDir, infoDir, filepath.Base(abs))

	// Сначала .trashinfo: если запись метаданных не удалась, файл остаётся
	// на месте, и пользователь не теряет возможность его найти.
	info := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
		escapeTrashPath(abs), time.Now().Format("2006-01-02T15:04:05"))
	infoPath := filepath.Join(infoDir, name+".trashinfo")
	if err := os.WriteFile(infoPath, []byte(info), 0o600); err != nil {
		return fmt.Errorf("запись %s: %w", infoPath, err)
	}

	if err := os.Rename(abs, filepath.Join(filesDir, name)); err != nil {
		_ = os.Remove(infoPath)
		return fmt.Errorf("перемещение в Корзину %s: %w (другая файловая система? используйте --permanent)", abs, err)
	}
	return nil
}

// uniqueTrashName подбирает имя, свободное и в files, и в info.
func uniqueTrashName(filesDir, infoDir, base string) string {
	name := base
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 1; ; i++ {
		_, errFile := os.Lstat(filepath.Join(filesDir, name))
		_, errInfo := os.Lstat(filepath.Join(infoDir, name+".trashinfo"))
		if os.IsNotExist(errFile) && os.IsNotExist(errInfo) {
			return name
		}
		name = fmt.Sprintf("%s.%d%s", stem, i, ext)
	}
}

// escapeTrashPath кодирует путь для поля Path в .trashinfo: спецификация
// требует percent-encoding по правилам URI, при этом разделители каталогов
// остаются как есть.
func escapeTrashPath(p string) string {
	parts := strings.Split(p, string(filepath.Separator))
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
