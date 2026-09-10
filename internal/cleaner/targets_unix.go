//go:build linux || darwin

package cleaner

import (
	"os"
	"path/filepath"
)

// Помощники для сборки списка категорий Linux- и macOS-ядер. В отличие от
// Windows, где пути хранятся шаблонами (%TEMP%), здесь пути собираются сразу
// готовыми — и любой из них может оказаться неизвестным (не определён $HOME
// или XDG-переменная), поэтому пустые значения отбрасываются, а не
// превращаются в относительные пути.

// sub — путь base/rel; пустая строка, если base неизвестен. Без этой
// проверки filepath.Join("", "cache") дал бы относительный путь "cache" и
// очистка ушла бы в текущий каталог.
func sub(base, rel string) string {
	if base == "" {
		return ""
	}
	return filepath.Join(base, filepath.FromSlash(rel))
}

// paths собирает список путей категории, отбрасывая пустые.
func paths(list ...string) []string { return nonEmpty(list) }

// dropEmptyTargets убирает файловые категории, у которых не осталось ни
// одного пути (например, весь Xcode на машине без Xcode). Категории-действия
// (Special) остаются: у них путей нет по определению.
func dropEmptyTargets(targets []Target) []Target {
	out := make([]Target, 0, len(targets))
	for _, t := range targets {
		if t.Special == SpecialNone && len(t.Paths) == 0 {
			continue
		}
		out = append(out, t)
	}
	return out
}

// userHomeDir — домашний каталог или "" (ошибка вызывающему не нужна).
func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
