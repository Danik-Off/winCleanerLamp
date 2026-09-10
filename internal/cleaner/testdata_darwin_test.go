//go:build darwin

package cleaner

// Платформенные данные для общих тестов, см. testdata_windows_test.go.

// unsafeTestFile — файл в защищённом системном каталоге.
const unsafeTestFile = "/usr/bin/env"

// unsafeTestDir — защищённый системный каталог.
const unsafeTestDir = "/Applications"

// systemDirRootCases — путь → ожидание isSystemDirRoot.
func systemDirRootCases() map[string]bool {
	return map[string]bool{
		"/System":                   true,
		"/System/Library":           true,
		"/usr":                      true,
		"/Applications":             true,
		"/Applications/Safari.app":  true,
		"/Library":                  true,
		"/Library/Caches":           true,
		"/Users/someone/Documents":  false,
		"/Users/someone/Projects":   false,
		"/Volumes/External/Backups": true,
	}
}

// systemDirFilesForRisk — файлы внутри системного каталога.
func systemDirFilesForRisk() []string {
	return []string{
		"/Library/Frameworks/Vendor.framework/Vendor",
		"/Library/Frameworks/Other.framework/Vendor",
	}
}

// riskyExtFileNames — имена файлов с «опасным» расширением.
func riskyExtFileNames() []string { return []string{"libfoo.dylib", "libfoo_backup.dylib"} }

// skipDirNamesCases — имена каталогов, которые обход дубликатов пропускает
// и не пропускает.
func skipDirNamesCases() (mustSkip, mustNotSkip []string) {
	return []string{"library", "applications", "system", ".trash", "node_modules", ".fseventsd"},
		[]string{"documents", "photos", "my project"}
}

// userDataPathCases — пути, похожие и не похожие на пользовательские данные.
func userDataPathCases() (likely, notLikely []string) {
	return []string{
			"/Users/me/Library/Application Support/minecraft/saves",
			"/Users/me/Documents/MyGame/Screenshots",
			"/Users/me/Library/Application Support/minecraft/resourcepacks",
		}, []string{
			"/Users/me/Library/Caches/SomeApp/Cache",
			"/Users/me/Library/Logs/SomeApp/logs",
		}
}
