//go:build linux

package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// forbiddenPathPrefixes — каталоги Linux, удаление которых (включая любые
// вложенные пути) всегда запрещено.
//
// Логика та же, что и в Windows-ядре, но список другой и составлен по FHS:
//   - бинарники и библиотеки системы: /bin, /sbin, /lib*, /usr, /opt
//   - конфигурация и загрузка: /etc, /boot
//   - виртуальные ФС ядра: /proc, /sys, /dev, /run
//   - состояние пакетного менеджера и служб: /var/lib, /var/spool, /var/backups
//     (при этом /var/cache, /var/log и /var/tmp НЕ запрещены — это как раз
//     легитимные цели очистки)
//   - чужие точки монтирования: /media, /mnt, /snap
//   - ключи пользователя: ~/.ssh, ~/.gnupg
//
// /home и /root в список НЕ входят: внутри них лежат как раз легитимные
// цели очистки (~/.cache, ~/.local/share/Trash). Сами эти каталоги целиком
// защищены отдельно, в platformPathSafety и checkStaticPathSafety.
var forbiddenPathPrefixes = buildForbiddenPathPrefixes()

func buildForbiddenPathPrefixes() []string {
	list := []string{
		"/bin", "/sbin", "/lib", "/lib32", "/lib64", "/libx32",
		"/usr", "/opt", "/etc", "/boot", "/efi",
		"/proc", "/sys", "/dev", "/run",
		"/var/lib", "/var/spool", "/var/backups", "/var/local", "/var/opt",
		"/media", "/mnt", "/snap", "/srv",
		"/lost+found",
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		home = strings.ToLower(filepath.Clean(home))
		// Ключи и секреты: не мусор ни при каких эвристиках.
		list = append(list,
			filepath.Join(home, ".ssh"),
			filepath.Join(home, ".gnupg"),
			filepath.Join(home, ".password-store"),
			filepath.Join(home, ".pki"),
		)
	}
	return list
}

// pathSafetyExceptions — точечные исключения из forbiddenPathPrefixes.
// В Linux-ядре таких пока нет: всё, что чистится, лежит вне запрещённых
// каталогов.
var pathSafetyExceptions = map[string]bool{}

// platformPathSafety — unix-специфичные проверки.
func platformPathSafety(abs, _ string) (bool, string) {
	if abs == "/" {
		return false, "нельзя удалить корень файловой системы"
	}
	// /tmp и /var/tmp здесь намеренно разрешены: их содержимое чистится
	// категориями с KeepRoot=true, сами каталоги при этом не удаляются.
	if abs == "/home" || abs == "/root" || abs == "/var" {
		return false, "нельзя удалить системный каталог верхнего уровня: " + abs
	}
	// /home/<user> целиком (не только домашний каталог текущего
	// пользователя, который отдельно проверяется в checkStaticPathSafety).
	if strings.HasPrefix(abs, "/home/") && strings.Count(abs, "/") == 2 {
		return false, "нельзя удалить домашний каталог пользователя целиком"
	}
	return true, ""
}
