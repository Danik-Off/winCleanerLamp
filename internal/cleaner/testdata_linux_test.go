//go:build linux

package cleaner

// Платформенные данные для общих тестов, см. testdata_windows_test.go.

// unsafeTestFile — файл в защищённом системном каталоге.
const unsafeTestFile = "/usr/bin/env"

// unsafeTestDir — защищённый системный каталог.
const unsafeTestDir = "/usr/lib"

// systemDirRootCases — путь → ожидание isSystemDirRoot.
func systemDirRootCases() map[string]bool {
	return map[string]bool{
		"/usr":                  true,
		"/usr/share/app":        true,
		"/opt":                  true,
		"/opt/vendor/app":       true,
		"/var":                  true,
		"/var/log":              true,
		"/etc":                  true,
		"/home/someone":         false,
		"/home/someone/Проекты": false,
		"/mnt/data":             false,
	}
}

// systemDirFilesForRisk — файлы внутри системного каталога.
func systemDirFilesForRisk() []string {
	return []string{
		"/usr/lib/vendor/libshared.so",
		"/usr/lib/other/libshared.so",
	}
}

// riskyExtFileNames — имена файлов с «опасным» расширением.
func riskyExtFileNames() []string { return []string{"libfoo.so", "libfoo_backup.so"} }

// skipDirNamesCases — имена каталогов, которые обход дубликатов пропускает
// и не пропускает.
func skipDirNamesCases() (mustSkip, mustNotSkip []string) {
	return []string{"proc", "sys", "snap", "node_modules", ".cache", "lost+found"},
		[]string{"documents", "photos", "my project"}
}

// userDataPathCases — пути, похожие и не похожие на пользовательские данные.
func userDataPathCases() (likely, notLikely []string) {
	return []string{
			"/home/me/.minecraft/saves",
			"/home/me/Документы/MyGame/Screenshots",
			"/home/me/.minecraft/resourcepacks",
		}, []string{
			"/home/me/.cache/SomeApp/Cache",
			"/home/me/.local/share/SomeApp/logs",
		}
}
