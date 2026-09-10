//go:build windows

package cleaner

// Платформенные данные для общих тестов: у каждой ОС свои защищённые
// каталоги, системные корни и имена папок, поэтому сами проверки живут в
// общих тестах, а конкретные пути — здесь и в testdata_linux_test.go /
// testdata_darwin_test.go.

// unsafeTestFile — файл в защищённом системном каталоге.
const unsafeTestFile = `C:\Windows\System32\drivers\etc\hosts`

// unsafeTestDir — защищённый системный каталог.
const unsafeTestDir = `C:\Program Files`

// systemDirRootCases — путь → ожидание isSystemDirRoot.
func systemDirRootCases() map[string]bool {
	return map[string]bool{
		`C:\Program Files`:           true,
		`C:\Program Files\SubApp`:    true,
		`C:\Program Files (x86)`:     true,
		`C:\Windows`:                 true,
		`C:\Windows\System32`:        true,
		`C:\ProgramData`:             true,
		`C:\ProgramData\SomeApp`:     true,
		`C:\Users\Someone\Documents`: false,
		`D:\Games`:                   false,
	}
}

// systemDirFilesForRisk — файлы внутри системного каталога: группа
// дубликатов из них обязана получать предупреждение.
func systemDirFilesForRisk() []string {
	return []string{
		`C:\Program Files\Vendor\shared.dll`,
		`C:\Program Files\Other\shared.dll`,
	}
}

// riskyExtFileNames — имена файлов с «опасным» расширением.
func riskyExtFileNames() []string { return []string{"app.exe", "app_backup.exe"} }

// skipDirNamesCases — имена каталогов, которые обход дубликатов пропускает
// и не пропускает.
func skipDirNamesCases() (mustSkip, mustNotSkip []string) {
	return []string{"windows", "program files", "program files (x86)", "node_modules", "$recycle.bin", "windows.old"},
		[]string{"documents", "photos", "my project"}
}

// userDataPathCases — пути, похожие и не похожие на пользовательские данные.
func userDataPathCases() (likely, notLikely []string) {
	return []string{
			`C:\Users\Me\AppData\Roaming\.minecraft\saves`,
			`C:\Users\Me\Documents\MyGame\Screenshots`,
			`C:\Users\Me\AppData\Roaming\.minecraft\resourcepacks`,
		}, []string{
			`C:\Users\Me\AppData\Local\SomeApp\Cache`,
			`C:\Users\Me\AppData\Roaming\SomeApp\logs`,
		}
}
