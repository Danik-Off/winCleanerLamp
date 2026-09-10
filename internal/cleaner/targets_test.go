package cleaner

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath_ExpandsWindowsStyleVars(t *testing.T) {
	t.Setenv("WCL_TEST_BASE", filepath.Join("base", "dir"))
	got := ExpandPath("%WCL_TEST_BASE%" + string(filepath.Separator) + "cache")
	want := filepath.Join("base", "dir") + string(filepath.Separator) + "cache"
	if got != want {
		t.Errorf("ExpandPath = %q, want %q", got, want)
	}
}

// Неопределённая переменная делает путь бессмысленным, и раскрывать его в
// огрызок нельзя: %LOCALAPPDATA%\Temp на Linux превратился бы в /Temp.
func TestExpandPath_MissingVarReturnsEmpty(t *testing.T) {
	if got := ExpandPath("%WCL_TEST_DEFINITELY_MISSING%/cache"); got != "" {
		t.Errorf("ExpandPath с неизвестной переменной = %q, ожидалась пустая строка", got)
	}
}

func TestExpandPath_EmptyVarReturnsEmpty(t *testing.T) {
	t.Setenv("WCL_TEST_EMPTY", "")
	if got := ExpandPath("%WCL_TEST_EMPTY%/cache"); got != "" {
		t.Errorf("ExpandPath с пустой переменной = %q, ожидалась пустая строка", got)
	}
}

func TestExpandPath_TildeExpandsToHome(t *testing.T) {
	got := ExpandPath("~" + string(filepath.Separator) + "wcl-test")
	if got == "" || strings.HasPrefix(got, "~") {
		t.Errorf("ExpandPath(~) не раскрыл домашний каталог: %q", got)
	}
	if !strings.HasSuffix(got, "wcl-test") {
		t.Errorf("ExpandPath(~) потерял хвост пути: %q", got)
	}
}

func TestExpandPath_PlainPathUnchanged(t *testing.T) {
	p := filepath.Join("plain", "path", "without", "vars")
	if got := ExpandPath(p); got != p {
		t.Errorf("ExpandPath(%q) = %q, путь без переменных меняться не должен", p, got)
	}
}

func TestTargetForbidden(t *testing.T) {
	target := Target{ForbidSubstrings: []string{"/.X11-unix", "/systemd-private-"}}
	forbidden := []string{
		filepath.Join("tmp", ".X11-unix", "X0"),
		filepath.Join("tmp", "systemd-private-abc", "tmp"),
	}
	for _, p := range forbidden {
		if !target.forbidden(string(filepath.Separator) + p) {
			t.Errorf("forbidden(%q) = false, want true", p)
		}
	}
	allowed := filepath.Join(string(filepath.Separator)+"tmp", "ordinary-file.tmp")
	if target.forbidden(allowed) {
		t.Errorf("forbidden(%q) = true, want false", allowed)
	}
}

func TestTargetForbidden_EmptyListAllowsEverything(t *testing.T) {
	var target Target
	if target.forbidden(filepath.Join("any", "path")) {
		t.Error("категория без ForbidSubstrings не должна ничего запрещать")
	}
}

// AllTargets каждого ядра обязан отдавать корректные категории: уникальные
// id и непустое описание — на них опирается GUI (--list --json).
func TestAllTargets_ValidAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, target := range AllTargets() {
		if target.ID == "" {
			t.Fatalf("категория без id: %+v", target)
		}
		if seen[target.ID] {
			t.Errorf("дублирующийся id категории: %s", target.ID)
		}
		seen[target.ID] = true
		if target.Name == "" || target.Description == "" {
			t.Errorf("категория %s без имени или описания", target.ID)
		}
		if target.Special == SpecialNone && len(target.Paths) == 0 {
			t.Errorf("категория %s без путей и без спец-действия", target.ID)
		}
		// Windows-ядро хранит пути шаблонами (%TEMP%), Linux и macOS —
		// уже готовыми; в обоих случаях после раскрытия путь обязан быть
		// абсолютным, иначе очистка ушла бы в текущий каталог.
		for _, p := range target.Paths {
			expanded := ExpandPath(p)
			if expanded != "" && !filepath.IsAbs(expanded) {
				t.Errorf("категория %s: путь %q раскрылся в неабсолютный %q", target.ID, p, expanded)
			}
		}
	}
}
