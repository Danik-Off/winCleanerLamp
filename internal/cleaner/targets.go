package cleaner

import (
	"os"
	"path/filepath"
	"strings"
)

// Target описывает одну категорию мусора.
type Target struct {
	ID          string   // короткий идентификатор для --categories
	Name        string   // человекочитаемое имя
	Description string   // что именно чистится и почему это безопасно
	Paths       []string // пути (могут содержать %ENV% и ~)
	// Если true — содержимое директории удаляется, но сама директория сохраняется.
	KeepRoot bool
	// Минимальный возраст файла в часах. 0 = без ограничения.
	MinAgeHours int
	// Если путь содержит один из этих подстрок после раскрытия — пропускаем (защита).
	ForbidSubstrings []string
	// Специальный обработчик (для нефайловых действий, напр. корзина / DNS).
	Special SpecialAction
	// Aggressive — категория потенциально опасна / требует явного согласия.
	// Включается только с флагом --aggressive.
	Aggressive bool
}

// SpecialAction — нефайловое действие категории (очистка через системную
// утилиту вместо обхода дерева файлов). Значения объявлены для всех ОС в
// одном месте, а реализованы каждым ядром отдельно: см. processSpecial в
// specials_windows.go / specials_linux.go / specials_darwin.go. Категория с
// неизвестным текущему ядру Special просто не попадает в его AllTargets.
type SpecialAction string

const (
	SpecialNone SpecialAction = ""

	// Общие для всех ОС (разные реализации).
	SpecialDNSCache       SpecialAction = "dns_cache"
	SpecialThumbnailCache SpecialAction = "thumbnail_cache"

	// Windows.
	SpecialRecycleBin       SpecialAction = "recycle_bin"
	SpecialEventLogs        SpecialAction = "event_logs"
	SpecialComponentCleanup SpecialAction = "component_cleanup"

	// Linux и macOS: Корзина по стандарту FreeDesktop (~/.local/share/Trash)
	// и ~/.Trash соответственно.
	SpecialTrash SpecialAction = "trash"

	// Linux.
	SpecialJournalVacuum SpecialAction = "journal_vacuum"
	SpecialPackageCache  SpecialAction = "package_cache"
	SpecialFlatpakUnused SpecialAction = "flatpak_unused"
	SpecialSnapDisabled  SpecialAction = "snap_disabled"

	// macOS.
	SpecialQuickLookCache SpecialAction = "quicklook_cache"
	SpecialBrewCleanup    SpecialAction = "brew_cleanup"
	SpecialLocalSnapshots SpecialAction = "local_snapshots"
)

// AllTargets — перечень категорий мусора текущей ОС. Реализация своя у
// каждого ядра: targets_windows.go, targets_linux.go, targets_darwin.go.

// ExpandPath раскрывает переменные окружения (%VAR% в стиле Windows и
// $VAR/${VAR} в стиле Unix) и ~ в начале пути. Формат %VAR% понимается на
// всех ОС — так orphaned_apps.json, написанный для Windows, остаётся
// читаемым и другими ядрами.
// Если хотя бы одна переменная в пути не определена, возвращается пустая
// строка, а не обрубок вроде "\Temp" или "/Temp": на Linux и macOS
// %LOCALAPPDATA% и подобные не существуют, и без этой проверки
// windows-путь из orphaned_apps.json превращался бы в путь от корня диска.
//
// $VAR раскрывается только там, где это принято (Linux/macOS): в Windows
// доллар — обычный символ имени папки (C:\$WINDOWS.~BT, C:\$GetCurrent), и
// раньше os.Expand съедал такие пути целиком, превращая их в "C:\".
func ExpandPath(p string) string {
	out, ok := expandVars(p)
	if !ok {
		return ""
	}
	if strings.HasPrefix(out, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			out = filepath.Join(home, strings.TrimPrefix(out, "~"))
		}
	}
	return out
}

// expandVars раскрывает %VAR% (везде) и $VAR/${VAR} (только если
// expandDollarVars). Второе значение = false, если какая-то переменная не
// определена или пуста.
func expandVars(p string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(p); {
		switch {
		case p[i] == '%':
			j := strings.IndexByte(p[i+1:], '%')
			if j <= 0 {
				b.WriteByte(p[i])
				i++
				continue
			}
			v, ok := lookupEnvNonEmpty(p[i+1 : i+1+j])
			if !ok {
				return "", false
			}
			b.WriteString(v)
			i += j + 2
		case expandDollarVars && p[i] == '$':
			name, next := parseDollarName(p, i)
			if name == "" {
				b.WriteByte(p[i])
				i++
				continue
			}
			v, ok := lookupEnvNonEmpty(name)
			if !ok {
				return "", false
			}
			b.WriteString(v)
			i = next
		default:
			b.WriteByte(p[i])
			i++
		}
	}
	return b.String(), true
}

// parseDollarName разбирает $NAME или ${NAME}, начиная с позиции доллара.
// Возвращает имя переменной и индекс следующего за ней символа; пустое имя
// означает "это не ссылка на переменную".
func parseDollarName(p string, i int) (string, int) {
	if i+1 >= len(p) {
		return "", i + 1
	}
	if p[i+1] == '{' {
		end := strings.IndexByte(p[i+2:], '}')
		if end <= 0 {
			return "", i + 1
		}
		return p[i+2 : i+2+end], i + 2 + end + 1
	}
	j := i + 1
	for j < len(p) && (p[j] == '_' || (p[j] >= 'a' && p[j] <= 'z') || (p[j] >= 'A' && p[j] <= 'Z') || (p[j] >= '0' && p[j] <= '9')) {
		j++
	}
	if j == i+1 {
		return "", i + 1
	}
	return p[i+1 : j], j
}

func lookupEnvNonEmpty(name string) (string, bool) {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}
