//go:build darwin

package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// forbiddenPathPrefixes — каталоги macOS, удаление которых (включая любые
// вложенные пути) всегда запрещено:
//   - /System, /bin, /sbin, /usr, /Library/Frameworks, /Library/Extensions —
//     системные компоненты (часть из них и так защищена SIP, но ошибку лучше
//     поймать до вызова ФС);
//   - /Applications — установленные программы (их удаляет пользователь или
//     --uninstall-launch, а не очистка мусора);
//   - /private/var/db, /private/var/vm — служебные базы и файл подкачки;
//     при этом /private/var/log, /private/var/folders (TMPDIR) и
//     /private/var/tmp разрешены — это цели очистки;
//   - /Volumes — смонтированные диски, включая Time Machine;
//   - ~/Library/Keychains, ~/Library/Mobile Documents (iCloud Drive),
//     ~/Library/Mail, ~/Library/Messages, ~/.ssh, ~/.gnupg — данные и ключи
//     пользователя, которые ни одна эвристика не должна считать мусором.
var forbiddenPathPrefixes = buildForbiddenPathPrefixes()

func buildForbiddenPathPrefixes() []string {
	list := []string{
		"/system", "/bin", "/sbin", "/usr", "/cores",
		"/library/frameworks", "/library/extensions", "/library/keychains",
		"/library/security", "/library/startupitems",
		"/applications", "/volumes", "/network",
		"/private/var/db", "/private/var/vm", "/private/var/root",
		"/private/etc", "/etc", "/var/db", "/var/vm",
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		home = strings.ToLower(filepath.Clean(home))
		list = append(list,
			filepath.Join(home, "library/keychains"),
			filepath.Join(home, "library/mobile documents"),
			filepath.Join(home, "library/mail"),
			filepath.Join(home, "library/messages"),
			filepath.Join(home, "library/photos"),
			filepath.Join(home, ".ssh"),
			filepath.Join(home, ".gnupg"),
		)
	}
	return list
}

// pathSafetyExceptions — точечные исключения из forbiddenPathPrefixes.
// В macOS-ядре таких пока нет.
var pathSafetyExceptions = map[string]bool{}

// platformPathSafety — macOS-специфичные проверки.
func platformPathSafety(abs, _ string) (bool, string) {
	if abs == "/" {
		return false, "нельзя удалить корень файловой системы"
	}
	// /Users и /Users/<user> целиком.
	if abs == "/Users" || abs == "/private" || abs == "/var" || abs == "/Library" {
		return false, "нельзя удалить системный каталог верхнего уровня: " + abs
	}
	if strings.HasPrefix(abs, "/Users/") && strings.Count(abs, "/") == 2 {
		return false, "нельзя удалить домашний каталог пользователя целиком"
	}
	return true, ""
}
