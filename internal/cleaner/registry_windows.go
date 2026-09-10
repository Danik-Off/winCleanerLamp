//go:build windows

package cleaner

import "os/exec"

// registryExportExt — расширение файла резервной копии, которую
// --orphan-export-reg делает перед удалением ключей.
const registryExportExt = ".reg"

// exportRegistryKey сохраняет ветку реестра в .reg-файл (reg export) —
// страховка перед удалением остатков программы.
func exportRegistryKey(key, outFile string) error {
	return exec.Command("reg", "export", key, outFile, "/y").Run()
}

// deleteRegistryKey удаляет ветку реестра целиком (reg delete /f).
func deleteRegistryKey(key string) error {
	return exec.Command("reg", "delete", key, "/f").Run()
}
