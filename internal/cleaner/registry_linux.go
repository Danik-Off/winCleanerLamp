//go:build linux

package cleaner

import "errors"

// В Linux реестра нет. Записи registryKeys в orphaned_apps.json относятся к
// Windows, и Linux-ядро их не трогает: registryKeyExists всегда возвращает
// false, поэтому до deleteRegistryKey дело не доходит, а явный запрос
// экспорта (--orphan-export-reg) честно сообщает, что экспортировать нечего.

const registryExportExt = ".reg"

var errNoRegistry = errors.New("реестр Windows отсутствует в этой ОС — экспорт и удаление ключей не выполняются")

func exportRegistryKey(_, _ string) error { return errNoRegistry }

func deleteRegistryKey(_ string) error { return errNoRegistry }
