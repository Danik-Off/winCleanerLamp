package cleaner

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf16"
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
	Error             string `json:"error,omitempty"`
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
			r.Error = err.Error()
			return r
		}
		r.Success = true
		return r
	}

	if err := moveToRecycleBin(path, false); err != nil {
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

	if err := moveToRecycleBin(path, true); err != nil {
		r.Error = err.Error()
		return r
	}
	r.Success = true
	r.MovedToRecycleBin = true
	return r
}

// moveToRecycleBin перемещает файл или папку в Корзину через PowerShell
// (Microsoft.VisualBasic.FileIO.FileSystem — единственный официально
// поддерживаемый способ без внешних зависимостей).
//
// Путь корректно экранируется для одинарных PowerShell-строк (единственный
// спецсимвол там — сама кавычка, экранируется удвоением), а весь скрипт
// затем передаётся через -EncodedCommand (UTF-16LE + Base64) вместо -Command.
// Раньше три разных места в проекте (orphan.go, emptydirs.go, main.ts)
// экранировали путь по-разному и местами вообще никак (main.ts делал
// `replace(/"/g, '\"')`, что не меняет строку) — теперь это одна проверенная
// реализация.
func moveToRecycleBin(path string, isDir bool) error {
	method := "DeleteFile"
	if isDir {
		method = "DeleteDirectory"
	}
	escaped := strings.ReplaceAll(path, "'", "''")
	script := fmt.Sprintf(
		"Add-Type -AssemblyName Microsoft.VisualBasic; [Microsoft.VisualBasic.FileIO.FileSystem]::%s('%s', 'OnlyErrorDialogs', 'SendToRecycleBin')",
		method, escaped,
	)
	return runPowerShellEncoded(script)
}

// runPowerShellEncoded запускает PowerShell-скрипт через -EncodedCommand,
// что исключает любые проблемы с квотированием на границе Go↔powershell.exe
// (в отличие от -Command, где путь со спецсимволами мог ломать разбор
// аргументов ещё до исполнения самого скрипта).
func runPowerShellEncoded(script string) error {
	u16 := utf16.Encode([]rune(script))
	buf := make([]byte, len(u16)*2)
	for i, v := range u16 {
		binary.LittleEndian.PutUint16(buf[i*2:], v)
	}
	encoded := base64.StdEncoding.EncodeToString(buf)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-EncodedCommand", encoded)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
