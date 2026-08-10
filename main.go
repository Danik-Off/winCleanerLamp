package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/suzen/wincleanerlamp/internal/cleaner"
)

const version = "0.1.0"

func main() {
	var (
		listFlag         = flag.Bool("list", false, "показать все категории и выйти")
		scanFlag         = flag.Bool("scan", false, "только посчитать размер мусора (ничего не удалять)")
		cleanFlag        = flag.Bool("clean", false, "удалить мусор")
		yesFlag          = flag.Bool("yes", false, "не запрашивать подтверждение")
		verboseFlag      = flag.Bool("verbose", false, "подробный вывод")
		categories       = flag.String("categories", "", "список категорий через запятую (по умолчанию — все безопасные)")
		exclude          = flag.String("exclude", "", "исключить категории (через запятую)")
		minAgeHours      = flag.Int("min-age-hours", 0, "удалять файлы старше N часов (применяется ко всем категориям)")
		aggressive       = flag.Bool("aggressive", false, "включить агрессивные категории (Windows.old, $WINDOWS.~BT, event-logs, старые Downloads и т.п.)")
		leftovers        = flag.Bool("leftovers", false, "найти возможные остатки удалённых программ в AppData/ProgramData (только отчёт)")
		leftoversLog     = flag.String("leftovers-log", "", "записать неизвестные находки в JSON-файл для обновления orphan DB")
		sysinfo          = flag.Bool("sysinfo", false, "показать размеры системных файлов (hiberfil.sys, pagefile.sys, swapfile.sys, WinSxS) и советы")
		duplicates       = flag.String("duplicates", "", "найти дубликаты файлов в указанных папках (через запятую, например C:\\Users\\Me)")
		duplicatesSystem = flag.Bool("duplicates-system", false, "разрешить поиск дубликатов внутри системных папок (Program Files, Windows, ProgramData) — по умолчанию они пропускаются")
		emptyDirs        = flag.String("empty-dirs", "", "найти пустые папки в указанных папках (через запятую)")
		showEmpty        = flag.Bool("show-empty", false, "показывать в таблице категории с нулевым размером")
		parallelN        = flag.Int("parallel", 8, "число параллельных сканеров (1 = последовательно)")
		showVer          = flag.Bool("version", false, "показать версию")
		jsonFlag         = flag.Bool("json", false, "структурированный JSON-вывод вместо текста (для --list/--scan/--clean/--delete-path/--delete-dir)")

		// Flags для безопасного удаления одного файла/папки (используется GUI
		// вместо прямых fs-операций в Electron — единая проверка безопасности
		// и перемещение в Корзину для дубликатов/остатков/пустых папок).
		deletePathFlag = flag.String("delete-path", "", "безопасно удалить один файл (по умолчанию — в Корзину)")
		deleteDirFlag  = flag.String("delete-dir", "", "безопасно удалить папку целиком (по умолчанию — в Корзину)")
		permanentFlag  = flag.Bool("permanent", false, "удалить навсегда вместо перемещения в Корзину (для --delete-path/--delete-dir)")

		// Flags для менеджера автозагрузки (в первую очередь для GUI, см. docs/research-autostart.md)
		autostartList    = flag.Bool("autostart-list", false, "показать записи автозагрузки (Run-ключи, папки автозагрузки, задания планировщика при входе)")
		autostartSetID   = flag.String("autostart-set", "", "id записи автозагрузки для переключения (используется с --autostart-enable/--autostart-disable)")
		autostartEnable  = flag.Bool("autostart-enable", false, "включить запись, указанную в --autostart-set")
		autostartDisable = flag.Bool("autostart-disable", false, "выключить запись, указанную в --autostart-set")

		// Flags для учёта мусора в конфиге
		recordFlag = flag.Bool("record", false, "записать найденный мусор в конфиг-файл для последующей авточистки")
		autoClean  = flag.Bool("auto-clean", false, "автоматически очистить мусор из записей конфига (без подтверждения)")
		showJunk   = flag.Bool("show-junk", false, "показать записанный мусор из конфига")
		junkByProg = flag.String("junk-by-program", "", "показать мусор от указанной программы (например, 'chrome', 'teams')")
		configPath = flag.String("config", "", "путь к файлу конфига учёта мусора (по умолчанию: %LOCALAPPDATA%\\Win Cleaner Lamp\\junk.json)")

		// Flags для OrphanCleaner (orphaned_apps.json)
		orphanConfig     = flag.String("orphan-config", cleaner.OrphanConfigPath(), "путь к файлу orphaned_apps.json")
		orphanScan       = flag.Bool("orphan-scan", false, "проверить записи из orphaned_apps.json (найти подтверждённый мусор)")
		orphanDiscover   = flag.Bool("orphan-discover", false, "найти неизвестные папки, не связанные с установленными программами")
		orphanCleanNames = flag.String("orphan-clean", "", "удалить мусор указанных программ из orphaned_apps.json (имена через запятую)")
		orphanInfo       = flag.String("orphan-info", "", "подробная информация по программе из orphaned_apps.json")
		orphanList       = flag.Bool("orphan-list", false, "показать все записи из orphaned_apps.json")
		orphanRoots      = flag.String("orphan-roots", "", "корневые папки для discover (через ;)")
		orphanOut        = flag.String("orphan-out", "", "сохранить результат discover в файл JSON")
		orphanJSON       = flag.Bool("orphan-json", false, "вывод discover в формате JSON")
		orphanRecycle    = flag.Bool("orphan-recycle", false, "перемещать в корзину вместо удаления (для orphan-clean)")
		orphanExportReg  = flag.String("orphan-export-reg", "", "экспортировать ключи реестра перед удалением (папка)")
		orphanCacheOnly  = flag.Bool("orphan-cache-only", false, "удалять только кеш программ (безопасно, не трогает настройки)")
		orphanIncludeUD  = flag.Bool("orphan-include-user-data", false, "разрешить удаление путей, похожих на пользовательские данные (сохранения, проекты и т.п.) — по умолчанию они пропускаются")
	)
	flag.Usage = usage
	flag.Parse()

	if *showVer {
		fmt.Printf("Win Cleaner Lamp %s\n", version)
		return
	}

	if runtime.GOOS != "windows" {
		fmt.Fprintln(os.Stderr, "Предупреждение: это приложение предназначено для Windows. На другой ОС большинство путей будут отсутствовать.")
	}

	// Безопасное удаление одного файла/папки — самостоятельная команда,
	// не требует загрузки категорий. Используется GUI вместо прямых
	// fs-операций в Electron (там раньше не было вообще никакой проверки пути).
	if *deletePathFlag != "" {
		result := cleaner.DeleteFile(*deletePathFlag, *permanentFlag)
		printDeleteResult(result, *jsonFlag)
		if !result.Success {
			os.Exit(1)
		}
		return
	}
	if *deleteDirFlag != "" {
		result := cleaner.DeleteDir(*deleteDirFlag, *permanentFlag)
		printDeleteResult(result, *jsonFlag)
		if !result.Success {
			os.Exit(1)
		}
		return
	}

	// Менеджер автозагрузки — самостоятельные команды, см. docs/research-autostart.md.
	if *autostartList {
		entries := cleaner.ListAutostartEntries()
		if *jsonFlag {
			printJSON(entries)
		} else {
			printAutostartList(entries)
		}
		return
	}
	if *autostartSetID != "" {
		if *autostartEnable == *autostartDisable {
			fmt.Fprintln(os.Stderr, "укажите ровно один из флагов: --autostart-enable или --autostart-disable")
			os.Exit(2)
		}
		err := cleaner.ToggleAutostart(*autostartSetID, *autostartEnable)
		result := struct {
			Success bool   `json:"success"`
			Error   string `json:"error,omitempty"`
		}{Success: err == nil}
		if err != nil {
			result.Error = err.Error()
		}
		if *jsonFlag {
			printJSON(result)
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		} else {
			fmt.Println("Готово.")
		}
		if err != nil {
			os.Exit(1)
		}
		return
	}

	all := cleaner.AllTargets()

	// Загружаем orphaned_apps.json и добавляем кеш-таргеты в обычный scan/clean
	orphCfg, orphErr := cleaner.LoadOrphanConfig(*orphanConfig)
	if orphErr == nil && orphCfg != nil {
		cacheTargets := cleaner.CacheTargetsFromOrphanConfig(orphCfg)
		all = append(all, cacheTargets...)
		if *verboseFlag {
			fmt.Printf("Orphan DB: %d приложений загружено, %d категорий кеша добавлено.\n", len(orphCfg.Apps), len(cacheTargets))
		}
		for _, w := range orphCfg.Warnings {
			fmt.Fprintf(os.Stderr, "Предупреждение (orphaned_apps.json): %s\n", w)
		}
	} else if *verboseFlag || *scanFlag || *cleanFlag {
		fmt.Fprintf(os.Stderr, "Предупреждение: не удалось загрузить orphaned_apps.json (%s): %v\n", *orphanConfig, orphErr)
		fmt.Fprintln(os.Stderr, "Категории кеша приложений не будут доступны. Положите orphaned_apps.json рядом с exe.")
	}

	if *listFlag {
		if *jsonFlag {
			printListJSON(all)
		} else {
			printList(all)
		}
		return
	}

	// Обработка флагов учёта мусора
	if *showJunk || *junkByProg != "" {
		runShowJunk(*configPath, *junkByProg)
		if !*scanFlag && !*cleanFlag && !*recordFlag && !*autoClean {
			return
		}
	}

	if *autoClean {
		runAutoClean(*configPath, *verboseFlag)
		if !*scanFlag && !*cleanFlag && !*recordFlag {
			return
		}
	}

	if *sysinfo {
		runSysInfo()
		if !*scanFlag && !*cleanFlag && !*recordFlag && !*leftovers {
			return
		}
	}

	// ─── OrphanCleaner commands ───
	if *orphanList {
		runOrphanList(*orphanConfig)
		if !*orphanScan && !*orphanDiscover && *orphanCleanNames == "" {
			return
		}
	}

	if *orphanScan {
		runOrphanScan(*orphanConfig, *verboseFlag)
		if !*orphanDiscover && *orphanCleanNames == "" && !*scanFlag && !*cleanFlag {
			return
		}
	}

	if *orphanInfo != "" {
		runOrphanInfo(*orphanConfig, *orphanInfo)
		if !*orphanScan && !*orphanDiscover && *orphanCleanNames == "" && !*scanFlag && !*cleanFlag {
			return
		}
	}

	if *orphanDiscover {
		var roots []string
		if *orphanRoots != "" {
			roots = strings.Split(*orphanRoots, ";")
		}
		runOrphanDiscover(*orphanConfig, roots, *orphanJSON, *orphanOut)
		if *orphanCleanNames == "" && !*scanFlag && !*cleanFlag {
			return
		}
	}

	if *orphanCleanNames != "" {
		names := splitCSV(*orphanCleanNames)
		runOrphanClean(*orphanConfig, names, *orphanRecycle, *orphanExportReg, *verboseFlag, *orphanCacheOnly, *orphanIncludeUD)
		if !*scanFlag && !*cleanFlag {
			return
		}
	}

	if *leftovers {
		runLeftovers(*orphanConfig, *leftoversLog)
		if !*scanFlag && !*cleanFlag && !*recordFlag && *duplicates == "" && *emptyDirs == "" {
			return
		}
	}

	if *duplicates != "" {
		runDuplicates(*duplicates, *duplicatesSystem)
		if !*scanFlag && !*cleanFlag && !*recordFlag && *emptyDirs == "" {
			return
		}
	}

	if *emptyDirs != "" {
		runEmptyDirs(*emptyDirs)
		if !*scanFlag && !*cleanFlag {
			return
		}
	}

	if !*scanFlag && !*cleanFlag {
		usage()
		os.Exit(2)
	}

	selected := filterTargets(all, *categories, *exclude, *aggressive)
	if len(selected) == 0 {
		fmt.Fprintln(os.Stderr, "Нет категорий для обработки.")
		os.Exit(2)
	}

	opts := cleaner.Options{
		DryRun:  *scanFlag && !*cleanFlag,
		Verbose: *verboseFlag && !*jsonFlag,
		MinAge:  time.Duration(*minAgeHours) * time.Hour,
		Logger:  func(s string) { fmt.Println(s) },
	}

	// Загружаем конфиг для записи, если требуется
	var junkCfg *cleaner.JunkConfig
	var junkPath string
	if *recordFlag || *autoClean {
		junkPath = *configPath
		cfg, err := cleaner.LoadJunkConfig(junkPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка загрузки конфига: %v\n", err)
			os.Exit(2)
		}
		junkCfg = cfg
	}

	// Сначала всегда делаем сканирование, чтобы показать сводку.
	if !*jsonFlag {
		fmt.Printf("Сканирую %d категори%s...\n", len(selected), pluralRu(len(selected)))
	}
	scanOpts := opts
	scanOpts.DryRun = true
	scanOpts.Verbose = false

	start := time.Now()
	rows := parallelScan(selected, scanOpts, *parallelN, junkCfg, *recordFlag, junkPath, *jsonFlag)
	var totalBytes int64
	var totalFiles int
	for _, r := range rows {
		totalBytes += r.Bytes
		totalFiles += r.Files
	}
	scanDuration := time.Since(start)
	if !*jsonFlag {
		fmt.Printf("Сканирование заняло %.1f сек.\n\n", scanDuration.Seconds())
		printTable(rows, totalBytes, totalFiles, *showEmpty)
	}

	if opts.DryRun {
		// Если записываем в конфиг — сохраняем даже при сканировании
		if *recordFlag && junkCfg != nil {
			if err := cleaner.SaveJunkConfig(junkCfg, junkPath); err != nil {
				if !*jsonFlag {
					fmt.Fprintf(os.Stderr, "Ошибка сохранения конфига: %v\n", err)
				}
			} else if !*jsonFlag {
				fmt.Printf("Записано %d элементов мусора в %s\n", len(junkCfg.Records), junkPath)
			}
		}
		if *jsonFlag {
			printJSON(jsonScanResult{
				Categories:      rowsToJSON(rows),
				TotalBytes:      totalBytes,
				TotalFiles:      totalFiles,
				DurationSeconds: scanDuration.Seconds(),
			})
		}
		return
	}

	// Подтверждение. В --json (машинный режим, например GUI) интерактивный
	// prompt не имеет смысла и заблокировал бы процесс на чтении stdin —
	// вместо этого требуем явный --yes и выходим с ошибкой, если его нет.
	if !*yesFlag {
		if *jsonFlag {
			printJSON(map[string]any{"error": "требуется --yes в связке с --json (интерактивное подтверждение не поддерживается)"})
			os.Exit(2)
		}
		fmt.Printf("\nУдалить перечисленное? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" && line != "д" && line != "да" {
			fmt.Println("Отменено.")
			return
		}
	}

	if !*jsonFlag {
		fmt.Println("\nОчистка...")
	}
	cleanStart := time.Now()
	var cleanedBytes int64
	var cleanedFiles int
	var allErrors []string
	cleanedByCategory := make(map[string]cleaner.Report, len(rows))
	// Очищаем только непустые (те что нашли мусор при скане) — экономия времени.
	nonEmpty := make(map[string]bool, len(rows))
	for _, r := range rows {
		if r.Bytes > 0 || r.Files > 0 || r.Target.Special != "" {
			nonEmpty[r.Target.ID] = true
		}
	}
	for i, t := range selected {
		if !nonEmpty[t.ID] {
			continue
		}
		if !*jsonFlag {
			fmt.Printf("[%d/%d] → %s\n", i+1, len(selected), t.Name)
		}
		r := cleaner.Process(t, opts)
		cleanedByCategory[t.ID] = r
		cleanedBytes += r.Bytes
		cleanedFiles += r.Files
		if len(r.Errors) > 0 {
			allErrors = append(allErrors, r.Errors...)
			if *verboseFlag && !*jsonFlag {
				for _, e := range r.Errors {
					fmt.Printf("   ! %s\n", e)
				}
			}
		}
		if !*jsonFlag {
			fmt.Printf("   освобождено: %s (%d файл%s)\n", cleaner.Human(r.Bytes), r.Files, pluralFilesRu(r.Files))
		}
	}

	if !*jsonFlag {
		fmt.Printf("\nГотово. Освобождено: %s в %d файлах.\n", cleaner.Human(cleanedBytes), cleanedFiles)
		if len(allErrors) > 0 {
			fmt.Printf("Ошибок/пропусков: %d (часть файлов могла быть занята или требует прав администратора).\n", len(allErrors))
			if !*verboseFlag {
				fmt.Println("Запустите с --verbose для подробностей.")
			}
		}
	}

	// Сохраняем конфиг после очистки
	if *recordFlag && junkCfg != nil {
		if err := cleaner.SaveJunkConfig(junkCfg, junkPath); err != nil {
			if !*jsonFlag {
				fmt.Fprintf(os.Stderr, "Ошибка сохранения конфига: %v\n", err)
			}
		} else if !*jsonFlag {
			fmt.Printf("Конфиг сохранён: %s (всего записей: %d)\n", junkPath, len(junkCfg.Records))
		}
	}

	if *jsonFlag {
		categories := rowsToJSON(rows)
		for i, c := range categories {
			if r, ok := cleanedByCategory[c.ID]; ok {
				categories[i].Bytes = r.Bytes
				categories[i].Files = r.Files
				categories[i].Errors = r.Errors
			}
		}
		printJSON(jsonCleanResult{
			Categories:      categories,
			ScannedBytes:    totalBytes,
			ScannedFiles:    totalFiles,
			CleanedBytes:    cleanedBytes,
			CleanedFiles:    cleanedFiles,
			Errors:          allErrors,
			DurationSeconds: time.Since(cleanStart).Seconds(),
		})
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Win Cleaner Lamp %s — CLI очиститель мусора Windows.

Использование:
  Win Cleaner Lamp --scan                      показать сколько можно освободить
  Win Cleaner Lamp --clean                     очистить (с подтверждением)
  Win Cleaner Lamp --clean --yes               очистить без подтверждения
  Win Cleaner Lamp --list                      список категорий
  Win Cleaner Lamp --clean --categories user-temp,prefetch,recycle-bin
  Win Cleaner Lamp --clean --exclude recycle-bin,dns-cache

  # Учёт мусора в конфиге
  Win Cleaner Lamp --scan --record             сканировать и записать мусор в конфиг
  Win Cleaner Lamp --show-junk                 показать записанный мусор
  Win Cleaner Lamp --junk-by-program chrome    показать мусор от Chrome
  Win Cleaner Lamp --auto-clean                автоматически очистить из конфига

  # Поиск остатков удалённых программ (OrphanCleaner)
  Win Cleaner Lamp --orphan-list               показать все записи из orphaned_apps.json
  Win Cleaner Lamp --orphan-scan               проверить записи и найти подтверждённый мусор
  Win Cleaner Lamp --orphan-info "Имя"         подробная информация по программе
  Win Cleaner Lamp --orphan-discover           найти неизвестные папки (потенциальный мусор)
  Win Cleaner Lamp --orphan-discover --orphan-out unknown.json
  Win Cleaner Lamp --orphan-clean "Имя1,Имя2"  удалить остатки указанных программ
  Win Cleaner Lamp --orphan-clean "Имя" --orphan-recycle  удалить в корзину
  Win Cleaner Lamp --orphan-clean "Имя"  (пути с пользовательскими данными пропускаются;
                                          --orphan-include-user-data — включить и их)

  # Безопасное удаление одного файла/папки (используется GUI)
  Win Cleaner Lamp --delete-path "C:\путь\файл"   удалить файл (в Корзину)
  Win Cleaner Lamp --delete-dir "C:\путь\папка"    удалить папку (в Корзину)
  Win Cleaner Lamp --delete-path "..." --permanent удалить навсегда, минуя Корзину

  # JSON-вывод (для GUI/скриптов)
  Win Cleaner Lamp --scan --json               структурированный JSON вместо таблицы
  Win Cleaner Lamp --list --json

Флаги:
`, version)
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, `
Подсказки:
  • Для очистки C:\Windows\Temp, Prefetch, SoftwareDistribution и т.п.
    запускайте консоль от имени администратора.
  • Файлы, занятые запущенными процессами, будут пропущены.
  • Сначала используйте --scan, чтобы убедиться, что всё корректно.
  
  • Для авточистки: сначала --scan --record, затем позже --auto-clean
  • Конфиг хранится в: %LOCALAPPDATA%\winCleanerLamp\junk.json
  
  • OrphanCleaner: используйте --orphan-scan для проверки orphaned_apps.json
  • Команда --orphan-discover найдёт папки, не связанные с установленными программами
  • Безопасность: orphan-discover только собирает информацию, не удаляет ничего`)
}

func printList(all []cleaner.Target) {
	fmt.Println("Доступные категории (безопасные по умолчанию):")
	fmt.Println()
	for _, t := range all {
		if t.Aggressive {
			continue
		}
		fmt.Printf("  %s\n    %s\n    %s\n\n", t.ID, t.Name, t.Description)
	}
	fmt.Println("Агрессивные категории (только с флагом --aggressive или через --categories):")
	fmt.Println()
	for _, t := range all {
		if !t.Aggressive {
			continue
		}
		fmt.Printf("  %s\n    %s\n    %s\n\n", t.ID, t.Name, t.Description)
	}
}

// ─── JSON-вывод (--json) ───
//
// Раньше GUI (gui/electron/main.ts) разбирал текстовый вывод CLI регэкспами
// (parseCategories, parseScanOutput) — хрупко и ломалось при любой правке
// форматирования таблиц. --json даёт машинно-читаемую альтернативу для тех
// же команд.

type jsonCategoryDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Aggressive  bool   `json:"aggressive"`
}

type jsonCategoriesResult struct {
	Safe       []jsonCategoryDTO `json:"safe"`
	Aggressive []jsonCategoryDTO `json:"aggressive"`
}

type jsonCategoryResult struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Bytes         int64    `json:"bytes"`
	Files         int      `json:"files"`
	Skipped       bool     `json:"skipped,omitempty"`
	SkippedReason string   `json:"skippedReason,omitempty"`
	Errors        []string `json:"errors,omitempty"`
}

type jsonScanResult struct {
	Categories      []jsonCategoryResult `json:"categories"`
	TotalBytes      int64                `json:"totalBytes"`
	TotalFiles      int                  `json:"totalFiles"`
	DurationSeconds float64              `json:"durationSeconds"`
}

type jsonCleanResult struct {
	Categories      []jsonCategoryResult `json:"categories"`
	ScannedBytes    int64                `json:"scannedBytes"`
	ScannedFiles    int                  `json:"scannedFiles"`
	CleanedBytes    int64                `json:"cleanedBytes"`
	CleanedFiles    int                  `json:"cleanedFiles"`
	Errors          []string             `json:"errors,omitempty"`
	DurationSeconds float64              `json:"durationSeconds"`
}

// printJSON сериализует v в JSON и печатает его в stdout одной строкой.
func printJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error":%q}`+"\n", err.Error())
		return
	}
	fmt.Println(string(data))
}

func printListJSON(all []cleaner.Target) {
	result := jsonCategoriesResult{}
	for _, t := range all {
		dto := jsonCategoryDTO{ID: t.ID, Name: t.Name, Description: t.Description, Aggressive: t.Aggressive}
		if t.Aggressive {
			result.Aggressive = append(result.Aggressive, dto)
		} else {
			result.Safe = append(result.Safe, dto)
		}
	}
	printJSON(result)
}

// rowsToJSON конвертирует отчёты сканирования в DTO для JSON-вывода.
func rowsToJSON(rows []cleaner.Report) []jsonCategoryResult {
	out := make([]jsonCategoryResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, jsonCategoryResult{
			ID:            r.Target.ID,
			Name:          r.Target.Name,
			Bytes:         r.Bytes,
			Files:         r.Files,
			Skipped:       r.Skipped,
			SkippedReason: r.SkippedReason,
			Errors:        r.Errors,
		})
	}
	return out
}

// printDeleteResult выводит результат --delete-path/--delete-dir в
// человекочитаемом или JSON-виде.
func printDeleteResult(r cleaner.DeleteResult, jsonOut bool) {
	if jsonOut {
		printJSON(r)
		return
	}
	if !r.Success {
		fmt.Fprintf(os.Stderr, "Ошибка: %s\n", r.Error)
		return
	}
	if r.MovedToRecycleBin {
		fmt.Printf("Перемещено в Корзину: %s\n", r.Path)
	} else {
		fmt.Printf("Удалено безвозвратно: %s\n", r.Path)
	}
}

func printAutostartList(entries []cleaner.AutostartEntry) {
	if len(entries) == 0 {
		fmt.Println("Записей автозагрузки не найдено.")
		return
	}
	bySource := map[cleaner.AutostartSource][]cleaner.AutostartEntry{}
	for _, e := range entries {
		bySource[e.Source] = append(bySource[e.Source], e)
	}
	labels := map[cleaner.AutostartSource]string{
		cleaner.AutostartRunHKCU:             "Реестр — Run (текущий пользователь)",
		cleaner.AutostartRunHKLM:             "Реестр — Run (все пользователи)",
		cleaner.AutostartRunHKLM32:           "Реестр — Run (32-бит, WOW6432Node)",
		cleaner.AutostartStartupFolderUser:   "Папка автозагрузки (пользователь)",
		cleaner.AutostartStartupFolderCommon: "Папка автозагрузки (все пользователи)",
		cleaner.AutostartScheduledTask:       "Задания планировщика (при входе в систему)",
	}
	for _, src := range []cleaner.AutostartSource{
		cleaner.AutostartRunHKCU, cleaner.AutostartRunHKLM, cleaner.AutostartRunHKLM32,
		cleaner.AutostartStartupFolderUser, cleaner.AutostartStartupFolderCommon, cleaner.AutostartScheduledTask,
	} {
		list := bySource[src]
		if len(list) == 0 {
			continue
		}
		fmt.Printf("=== %s (%d) ===\n", labels[src], len(list))
		for _, e := range list {
			state := "выкл"
			if e.Enabled {
				state = "вкл"
			}
			fmt.Printf("  [%s] %-30s %s\n", state, e.Name, e.Command)
		}
		fmt.Println()
	}
}

func filterTargets(all []cleaner.Target, include, exclude string, aggressive bool) []cleaner.Target {
	inc := splitCSV(include)
	exc := splitCSV(exclude)
	excSet := map[string]bool{}
	for _, e := range exc {
		excSet[strings.ToLower(e)] = true
	}
	var out []cleaner.Target
	if len(inc) == 0 {
		for _, t := range all {
			if excSet[strings.ToLower(t.ID)] {
				continue
			}
			if t.Aggressive && !aggressive {
				continue
			}
			out = append(out, t)
		}
		return out
	}
	index := map[string]cleaner.Target{}
	for _, t := range all {
		index[strings.ToLower(t.ID)] = t
	}
	// Если категория явно указана через --categories — добавляем даже агрессивную.
	for _, id := range inc {
		if t, ok := index[strings.ToLower(id)]; ok && !excSet[strings.ToLower(id)] {
			out = append(out, t)
		} else if !ok {
			fmt.Fprintf(os.Stderr, "предупреждение: неизвестная категория %q\n", id)
		}
	}
	return out
}

func runLeftovers(orphanCfgPath string, logFile string) {
	fmt.Println("Поиск возможных остатков удалённых программ...")
	fmt.Println("Сканирование: AppData, ProgramData, Program Files, реестр HKCU\\Software")

	// Загружаем orphan DB если доступна
	var orphanCfg *cleaner.OrphanConfig
	if cfg, err := cleaner.LoadOrphanConfig(orphanCfgPath); err == nil {
		orphanCfg = cfg
		fmt.Printf("Orphan DB: %d записей загружено из %s\n", len(cfg.Apps), orphanCfgPath)
	}

	fmt.Println("(это эвристика — перед удалением проверьте каждый элемент вручную)")
	fmt.Println()

	result, err := cleaner.ScanLeftoversEx(cleaner.LeftoverScanOptions{
		OrphanCfg: orphanCfg,
		LogFile:   logFile,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		return
	}
	cands := result.Candidates
	if len(cands) == 0 {
		fmt.Println("Подозрительных остатков не найдено.")
		return
	}

	// Группируем по типу. Известные остатки (knownOrphans) дополнительно делим
	// на "программа уже удалена" (приоритет для очистки) и "программа всё ещё
	// установлена" (осторожнее — это не мусор, а действующие данные) — по
	// запросу: показывать в первую очередь остатки именно удалённых программ,
	// а не просто "известно/неизвестно".
	var removedProgramLeftovers, installedProgramLeftovers, unknownFolders, cacheHits, empties, regKeys []cleaner.LeftoverCandidate
	for _, c := range cands {
		switch {
		case c.Type == cleaner.LeftoverEmpty:
			empties = append(empties, c)
		case c.Type == cleaner.LeftoverRegistry:
			regKeys = append(regKeys, c)
		case c.CacheHit:
			cacheHits = append(cacheHits, c)
		case c.OrphanMatch != "" && !c.InstalledMatch:
			removedProgramLeftovers = append(removedProgramLeftovers, c)
		case c.OrphanMatch != "":
			installedProgramLeftovers = append(installedProgramLeftovers, c)
		default:
			unknownFolders = append(unknownFolders, c)
		}
	}

	var totalSize int64
	var totalCount int

	printFolderSection := func(title string, items []cleaner.LeftoverCandidate, tag string) {
		if len(items) == 0 {
			return
		}
		fmt.Printf("  === %s (%d) ===\n", title, len(items))
		fmt.Printf("  %-10s  %8s  %-30s  %s\n", "РАЗМЕР", "ФАЙЛОВ", "ПРИЛОЖЕНИЕ", "ПУТЬ")
		fmt.Printf("  %s  %s  %s  %s\n", strings.Repeat("-", 10), strings.Repeat("-", 8), strings.Repeat("-", 30), strings.Repeat("-", 50))
		for _, c := range items {
			size := cleaner.Human(c.Size)
			if c.Size == -1 {
				size = "~большой"
			}
			label := tag
			if c.OrphanMatch != "" && tag == "" {
				label = "[" + c.OrphanMatch + "]"
			} else if tag != "" && !strings.HasPrefix(tag, "[") {
				label = "[" + tag + "]"
			}
			fmt.Printf("  %-10s  %8d  %-30s  %s\n", size, c.Files, label, c.Path)
			if c.Size > 0 {
				totalSize += c.Size
			}
			totalCount++
		}
		fmt.Println()
	}

	// Кеш (безопасно удалить)
	printFolderSection("Кеш программ из orphan DB (безопасно)", cacheHits, "[кеш]")

	// В первую очередь — остатки программ, которые уже удалены из системы.
	printFolderSection("★ Остатки УДАЛЁННЫХ программ (приоритет для очистки)", removedProgramLeftovers, "")

	// Затем — записи, принадлежащие ещё установленным программам (осторожнее).
	printFolderSection("Данные ещё УСТАНОВЛЕННЫХ программ (не мусор — не удалять не глядя)", installedProgramLeftovers, "")

	// Неизвестные
	printFolderSection("Неизвестные папки (нет в orphan DB)", unknownFolders, "[?]")

	// Пустые папки
	if len(empties) > 0 {
		fmt.Printf("  === Пустые папки (%d) ===\n", len(empties))
		for _, c := range empties {
			fmt.Printf("  [пусто]  %s\n", c.Path)
			totalCount++
		}
		fmt.Println()
	}

	// Ключи реестра
	if len(regKeys) > 0 {
		fmt.Printf("  === Ключи реестра без программ (%d) ===\n", len(regKeys))
		for _, c := range regKeys {
			fmt.Printf("  [реестр]  %s\n", c.Path)
			totalCount++
		}
		fmt.Println()
	}

	fmt.Printf("  ИТОГО: %d потенциальных остатков", totalCount)
	if totalSize > 0 {
		fmt.Printf(", ~%s на диске", cleaner.Human(totalSize))
	}
	fmt.Println()

	// Информация о логировании
	if logFile != "" && len(unknownFolders) > 0 {
		fmt.Printf("\n  Неизвестные находки (%d) записаны в: %s\n", len(unknownFolders), logFile)
		fmt.Println("  Используйте этот файл для обновления orphaned_apps.json.")
	}

	// Информация об установленных программах
	fmt.Printf("\n  Установленных программ: %d", len(result.Installed))
	inDB := 0
	for _, p := range result.Installed {
		if p.InOrphanDB {
			inDB++
		}
	}
	if inDB > 0 {
		fmt.Printf(" (в orphan DB: %d)", inDB)
	}
	fmt.Println()
	fmt.Println("\n  Для удаления папок используйте GUI или Проводник / rmdir /s.")
	fmt.Println("  Для реестра: regedit или reg delete <ключ>.")
}

func runDuplicates(pathsCSV string, allowSystemDirs bool) {
	roots := splitCSV(pathsCSV)
	if len(roots) == 0 {
		fmt.Fprintln(os.Stderr, "Укажите папки для поиска дубликатов через запятую.")
		return
	}
	fmt.Printf("Поиск дубликатов файлов в: %s\n", strings.Join(roots, ", "))
	fmt.Println("(это может занять несколько минут для больших дисков)")
	fmt.Println()

	result, err := cleaner.ScanDuplicates(cleaner.DuplicateScanOptions{
		Roots:           roots,
		MinSize:         1024,
		AllowSystemDirs: allowSystemDirs,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		return
	}

	if len(result.SkippedRoots) > 0 {
		fmt.Printf("Пропущены системные папки (используйте --duplicates-system, если это осознанный выбор): %s\n\n",
			strings.Join(result.SkippedRoots, ", "))
	}

	if len(result.Groups) == 0 {
		fmt.Printf("Дубликатов не найдено (просканировано %d файлов за %.1f сек.)\n",
			result.ScannedFiles, result.Duration.Seconds())
		return
	}

	fmt.Printf("Просканировано %d файлов за %.1f сек.\n\n", result.ScannedFiles, result.Duration.Seconds())

	limit := 50
	if len(result.Groups) < limit {
		limit = len(result.Groups)
	}

	for i, g := range result.Groups[:limit] {
		header := fmt.Sprintf("  === Группа %d: %s (x%d файлов) ===", i+1, cleaner.Human(g.Size), len(g.Paths))
		if g.RiskFlag != "" {
			header += " ⚠ РИСК: " + g.RiskFlag
		}
		fmt.Println(header)
		for _, p := range g.Paths {
			fmt.Printf("    %s\n", p)
		}
		fmt.Printf("    Экономия при удалении лишних: %s\n\n", cleaner.Human(g.WasteSize))
	}

	fmt.Printf("  ИТОГО: %d групп дубликатов, %d файлов, ~%s можно освободить\n",
		len(result.Groups), result.TotalFiles, cleaner.Human(result.TotalWaste))
	if len(result.Groups) > limit {
		fmt.Printf("  (показано первые %d групп из %d)\n", limit, len(result.Groups))
	}
}

func runEmptyDirs(pathsCSV string) {
	roots := splitCSV(pathsCSV)
	if len(roots) == 0 {
		fmt.Fprintln(os.Stderr, "Укажите папки для поиска пустых директорий через запятую.")
		return
	}
	fmt.Printf("Поиск пустых папок в: %s\n", strings.Join(roots, ", "))
	fmt.Println("Правила: • Игнорируются Thumbs.db, desktop.ini и подобные")
	fmt.Println("         • Системные папки (Windows, Program Files) пропускаются")
	fmt.Println()

	result, err := cleaner.ScanEmptyDirs(cleaner.EmptyDirScanOptions{
		Roots:      roots,
		IgnoreJunk: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		return
	}

	if result.Total == 0 {
		fmt.Println("Пустых папок не найдено.")
		return
	}

	fmt.Printf("=== Пустые папки (%d) ===\n", result.Total)
	fmt.Println()
	for _, d := range result.Dirs {
		fmt.Printf("[пусто]  %s\n", d.Path)
	}
	fmt.Println()
	fmt.Printf("ИТОГО: %d пустых папок найдено\n", result.Total)
	fmt.Println("Для удаления используйте GUI (перемещение в Корзину) или команду:")
	fmt.Println("  Win Cleaner Lamp --empty-dirs \"путь\" --clean")
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func printTable(rows []cleaner.Report, totalBytes int64, totalFiles int, showEmpty bool) {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Bytes > rows[j].Bytes })

	visible := rows
	hidden := 0
	if !showEmpty {
		visible = visible[:0]
		for _, r := range rows {
			if r.Bytes == 0 && r.Files == 0 && !r.Skipped {
				hidden++
				continue
			}
			visible = append(visible, r)
		}
	}

	if len(visible) == 0 {
		fmt.Println("  Мусора не найдено.")
		return
	}

	idW := len("КАТЕГОРИЯ")
	nameW := len("НАЗВАНИЕ")
	for _, r := range visible {
		if len(r.Target.ID) > idW {
			idW = len(r.Target.ID)
		}
		if len([]rune(r.Target.Name)) > nameW {
			nameW = len([]rune(r.Target.Name))
		}
	}
	if nameW > 50 {
		nameW = 50
	}
	fmt.Printf("  %-*s  %-*s  %12s  %8s\n", idW, "КАТЕГОРИЯ", nameW, "НАЗВАНИЕ", "РАЗМЕР", "ФАЙЛОВ")
	fmt.Printf("  %s  %s  %s  %s\n",
		strings.Repeat("-", idW),
		strings.Repeat("-", nameW),
		strings.Repeat("-", 12),
		strings.Repeat("-", 8),
	)
	for _, r := range visible {
		name := truncateRunes(r.Target.Name, nameW)
		size := cleaner.Human(r.Bytes)
		if r.Skipped {
			size = "-"
		}
		fmt.Printf("  %-*s  %-*s  %12s  %8d\n", idW, r.Target.ID, nameW, padRunes(name, nameW), size, r.Files)
	}
	fmt.Printf("\n  ИТОГО: %s в %d файлах", cleaner.Human(totalBytes), totalFiles)
	if hidden > 0 {
		fmt.Printf("  (скрыто пустых категорий: %d — используйте --show-empty)", hidden)
	}
	fmt.Println()
}

// parallelScan прогоняет сканирование целей параллельно с прогресс-баром.
func parallelScan(targets []cleaner.Target, opts cleaner.Options, workers int, junkCfg *cleaner.JunkConfig, record bool, junkPath string, quiet bool) []cleaner.Report {
	if workers < 1 {
		workers = 1
	}
	if workers > len(targets) {
		workers = len(targets)
	}

	results := make([]cleaner.Report, len(targets))
	type job struct{ idx int }
	jobs := make(chan job, len(targets))
	for i := range targets {
		jobs <- job{i}
	}
	close(jobs)

	var done int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	total := len(targets)
	// quiet=true (--json) отключает прогресс-бар: он пишет "\r"-строки в
	// stdout, что при перенаправлении в файл/пайп (не TTY) ломает JSON —
	// carriage return не стирает предыдущий текст, а остаётся как есть.
	progress := func(name string) {
		mu.Lock()
		done++
		if !quiet {
			fmt.Printf("\r  [%d/%d] %-60s", done, total, truncateRunes(name, 58))
		}
		mu.Unlock()
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				t := targets[j.idx]
				results[j.idx] = cleaner.Process(t, opts)
				progress(t.Name)
			}
		}()
	}
	wg.Wait()
	if !quiet {
		fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
	}

	// Записываем все записи в конфиг
	if record && junkCfg != nil {
		for _, r := range results {
			for _, rec := range r.Records {
				junkCfg.AddRecord(rec.Path, rec.CategoryID, rec.Size, "")
			}
		}
	}

	return results
}

func runSysInfo() {
	info := cleaner.GatherSysInfo()
	fmt.Println("Системные файлы и большие каталоги:")
	fmt.Println()
	for _, e := range info {
		size := "—"
		if !e.Exists {
			size = "не найден"
		} else if e.Size == -1 {
			size = "~большой"
		} else if e.Size > 0 {
			size = cleaner.Human(e.Size)
		}
		fmt.Printf("  %-25s %12s  %s\n", e.Name, size, e.Path)
		if e.Hint != "" {
			fmt.Printf("  %s\n", e.Hint)
		}
		fmt.Println()
	}
	fmt.Println("  Эти файлы НЕ удаляются этой утилитой — только информативно.")
	fmt.Println("  Управление:")
	fmt.Println("    • hiberfil.sys  — отключить:  powercfg /h off")
	fmt.Println("    • pagefile.sys  — размер:    Система → Дополнительные → Быстродействие → Виртуальная память")
	fmt.Println("    • WinSxS        — анализ:    dism /Online /Cleanup-Image /AnalyzeComponentStore")
	fmt.Println("                      очистка:   dism /Online /Cleanup-Image /StartComponentCleanup /ResetBase")
	fmt.Println()
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

func padRunes(s string, n int) string {
	r := []rune(s)
	if len(r) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(r))
}

// "категори" + суффикс: 1 → "ю", 2–4 → "и", иначе → "й"
func pluralRu(n int) string {
	mod10 := n % 10
	mod100 := n % 100
	if mod10 == 1 && mod100 != 11 {
		return "ю"
	}
	if mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20) {
		return "и"
	}
	return "й"
}

func pluralFilesRu(n int) string {
	mod10 := n % 10
	mod100 := n % 100
	if mod10 == 1 && mod100 != 11 {
		return ""
	}
	if mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20) {
		return "а"
	}
	return "ов"
}

// runShowJunk показывает записанный мусор из конфига.
func runShowJunk(configPath, byProgram string) {
	cfg, err := cleaner.LoadJunkConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки конфига: %v\n", err)
		return
	}

	var records []cleaner.JunkRecord
	if byProgram != "" {
		records = cfg.GetRecordsByProgram(byProgram)
		fmt.Printf("Мусор от программы %q (%d записей):\n\n", byProgram, len(records))
	} else {
		records = cfg.GetNonDeletedRecords()
		fmt.Printf("Записанный мусор (%d записей):\n\n", len(records))
	}

	if len(records) == 0 {
		fmt.Println("  Записей не найдено.")
		return
	}

	// Группируем по категории
	byCat := map[string][]cleaner.JunkRecord{}
	for _, r := range records {
		byCat[r.CategoryID] = append(byCat[r.CategoryID], r)
	}

	var totalSize int64
	for cat, recs := range byCat {
		var catSize int64
		for _, r := range recs {
			catSize += r.Size
			totalSize += r.Size
		}
		fmt.Printf("  [%s] ~%s (%d файлов)\n", cat, cleaner.Human(catSize), len(recs))
		limit := 10
		if len(recs) < limit {
			limit = len(recs)
		}
		for _, r := range recs[:limit] {
			fmt.Printf("    • %s", r.Path)
			if r.ProgramHint != "" {
				fmt.Printf("  (от %s)", r.ProgramHint)
			}
			fmt.Println()
		}
		if len(recs) > limit {
			fmt.Printf("    ... и ещё %d записей\n", len(recs)-limit)
		}
		fmt.Println()
	}

	fmt.Printf("  ИТОГО: ~%s в %d записях\n", cleaner.Human(totalSize), len(records))
}

// ─── OrphanCleaner CLI runners ───

func runOrphanList(cfgPath string) {
	cfg, err := cleaner.LoadOrphanConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		return
	}
	fmt.Printf("Записи из %s (%d):\n\n", cfgPath, len(cfg.Apps))
	for i, app := range cfg.Apps {
		fmt.Printf("  %d. %s", i+1, app.DisplayName)
		if app.Publisher != "" {
			fmt.Printf(" (%s)", app.Publisher)
		}
		fmt.Println()
		for _, p := range app.InstallPaths {
			fmt.Printf("     install: %s\n", p)
		}
		for _, p := range app.AdditionalPaths {
			fmt.Printf("     extra:   %s\n", p)
		}
		for _, k := range app.RegistryKeys {
			fmt.Printf("     reg:     %s\n", k)
		}
		if app.Notes != "" {
			fmt.Printf("     notes:   %s\n", app.Notes)
		}
		fmt.Println()
	}
}

func runOrphanScan(cfgPath string, verbose bool) {
	cfg, err := cleaner.LoadOrphanConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		return
	}
	fmt.Printf("Сканирование %d записей из %s...\n", len(cfg.Apps), cfgPath)

	results := cleaner.OrphanScan(cfg, verbose)
	if len(results) == 0 {
		fmt.Println("Подтверждённых остатков не найдено.")
		return
	}

	fmt.Printf("\nНайдены остатки от %d программ:\n", len(results))
	fmt.Println(strings.Repeat("-", 70))
	var totalSize int64
	for _, r := range results {
		fmt.Printf("  %s", r.App.DisplayName)
		if r.TotalSize > 0 {
			fmt.Printf("  (%s, %d файлов)", cleaner.Human(r.TotalSize), r.TotalFiles)
		}
		if r.ProgramInstalled {
			fmt.Print("  [программа установлена]")
		} else {
			fmt.Print("  [★ программа удалена]")
		}
		fmt.Println()
		for _, p := range r.FoundPaths {
			size := cleaner.Human(p.Size)
			if p.Size == -1 {
				size = "~большой"
			}
			path := p.Path
			if p.LikelyUserData {
				path = "⚠ [ПОЛЬЗОВАТЕЛЬСКИЕ ДАННЫЕ] " + path
			}
			fmt.Printf("    [%8s] %s\n", size, path)
		}
		for _, k := range r.FoundRegKeys {
			fmt.Printf("    [реестр]   %s\n", k)
		}
		totalSize += r.TotalSize
		fmt.Println()
	}
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("ИТОГО: %d программ, ~%s на диске\n", len(results), cleaner.Human(totalSize))
	fmt.Println("Для удаления: Win Cleaner Lamp --orphan-clean \"Имя программы1,Имя программы2\"")
	fmt.Println("Пути с пометкой [ПОЛЬЗОВАТЕЛЬСКИЕ ДАННЫЕ] --orphan-clean не удаляет без --orphan-include-user-data.")
}

func runOrphanInfo(cfgPath, displayName string) {
	cfg, err := cleaner.LoadOrphanConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		return
	}
	results := cleaner.OrphanScan(cfg, false)
	info := cleaner.OrphanInfo(results, displayName)
	if info == nil {
		fmt.Fprintf(os.Stderr, "Программа %q не найдена или не имеет остатков.\n", displayName)
		return
	}

	fmt.Printf("=== %s ===\n", info.App.DisplayName)
	if info.App.Publisher != "" {
		fmt.Printf("Издатель: %s\n", info.App.Publisher)
	}
	if info.App.Notes != "" {
		fmt.Printf("Заметки: %s\n", info.App.Notes)
	}
	fmt.Println()

	if len(info.FoundPaths) > 0 {
		fmt.Println("Найденные пути:")
		for _, p := range info.FoundPaths {
			size := cleaner.Human(p.Size)
			if p.Size == -1 {
				size = "~большой"
			}
			fmt.Printf("  [%8s] %d файлов  %s\n", size, p.Files, p.Path)
		}
	}
	if len(info.FoundRegKeys) > 0 {
		fmt.Println("Найденные ключи реестра:")
		for _, k := range info.FoundRegKeys {
			fmt.Printf("  %s\n", k)
		}
	}
	fmt.Printf("\nИтого: ~%s, %d файлов\n", cleaner.Human(info.TotalSize), info.TotalFiles)
}

func runOrphanDiscover(cfgPath string, roots []string, jsonOutput bool, outFile string) {
	var orphanCfg *cleaner.OrphanConfig
	cfg, err := cleaner.LoadOrphanConfig(cfgPath)
	if err == nil {
		orphanCfg = cfg
	}

	if len(roots) == 0 {
		roots = cleaner.DefaultDiscoverRoots()
	}

	fmt.Println("Сканирование неизвестных папок...")
	fmt.Printf("Корневые папки: %s\n\n", strings.Join(roots, ", "))

	results, err := cleaner.OrphanDiscover(cleaner.DiscoverOptions{
		Roots:      roots,
		OrphanCfg:  orphanCfg,
		JSONOutput: jsonOutput,
		OutputFile: outFile,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
	}

	if len(results) == 0 {
		fmt.Println("Неизвестных папок не найдено.")
		return
	}

	if jsonOutput {
		// JSON уже мог быть сохранён в файл через OutputFile
		// Но выведем на экран тоже
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println("Найденные папки (потенциальный мусор):")
		fmt.Println(strings.Repeat("-", 70))
		for i, r := range results {
			exeStr := "нет .exe"
			if r.HasExecutable {
				exeStr = "есть .exe"
			}
			fmt.Printf("%d. %s (%d MB, %s)\n", i+1, r.Path, r.SizeMB, exeStr)
		}
		fmt.Println(strings.Repeat("-", 70))
		fmt.Printf("Всего найдено: %d папок.\n", len(results))
		fmt.Println("Для добавления в список отслеживания вручную внесите запись в orphaned_apps.json.")
	}

	if outFile != "" {
		fmt.Printf("Результат сохранён в %s\n", outFile)
	}
}

func runOrphanClean(cfgPath string, names []string, recycle bool, exportReg string, verbose bool, cacheOnly bool, includeUserData bool) {
	cfg, err := cleaner.LoadOrphanConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		return
	}

	if cacheOnly {
		fmt.Printf("Очистка кеша для: %s (безопасно, только временные файлы)\n", strings.Join(names, ", "))
	} else {
		fmt.Printf("Очистка остатков для: %s\n", strings.Join(names, ", "))
		fmt.Println("⚠  ВНИМАНИЕ: полная очистка удаляет настройки, профили и данные программы!")
		fmt.Println("   Для безопасного удаления только кеша используйте: --orphan-cache-only")
		if !includeUserData {
			fmt.Println("   Пути, похожие на пользовательские данные (сохранения, проекты и т.п.), будут пропущены.")
			fmt.Println("   Чтобы удалить и их — добавьте --orphan-include-user-data (внимательно проверьте список).")
		}
	}
	if recycle {
		fmt.Println("Режим: перемещение в корзину")
	}

	results := cleaner.OrphanClean(cfg, names, cleaner.OrphanCleanOptions{
		CacheOnly:       cacheOnly,
		Recycle:         recycle,
		ExportReg:       exportReg,
		Verbose:         verbose,
		IncludeUserData: includeUserData,
		Logger:          func(s string) { fmt.Println(s) },
	})

	if len(results) == 0 {
		fmt.Println("Программы не найдены в orphaned_apps.json.")
		return
	}

	var totalFreed int64
	var totalFiles int
	var totalErrors int
	for _, r := range results {
		fmt.Printf("\n  %s:\n", r.App.DisplayName)
		if len(r.DeletedPaths) > 0 {
			fmt.Printf("    Удалено путей: %d\n", len(r.DeletedPaths))
			for _, p := range r.DeletedPaths {
				fmt.Printf("      ✓ %s\n", p)
			}
		}
		if len(r.SkippedUserData) > 0 {
			fmt.Printf("    Пропущено как пользовательские данные: %d\n", len(r.SkippedUserData))
			for _, p := range r.SkippedUserData {
				fmt.Printf("      ⚠ %s\n", p)
			}
		}
		if len(r.DeletedKeys) > 0 {
			fmt.Printf("    Удалено ключей реестра: %d\n", len(r.DeletedKeys))
			for _, k := range r.DeletedKeys {
				fmt.Printf("      ✓ %s\n", k)
			}
		}
		if len(r.Errors) > 0 {
			fmt.Printf("    Ошибки: %d\n", len(r.Errors))
			for _, e := range r.Errors {
				fmt.Printf("      ! %s\n", e)
			}
		}
		totalFreed += r.FreedBytes
		totalFiles += r.FreedFiles
		totalErrors += len(r.Errors)
	}

	fmt.Printf("\nГотово. Освобождено: ~%s в %d файлах.", cleaner.Human(totalFreed), totalFiles)
	if totalErrors > 0 {
		fmt.Printf(" Ошибок: %d.", totalErrors)
	}
	fmt.Println()
}

// runAutoClean автоматически очищает мусор из записей конфига.
func runAutoClean(configPath string, verbose bool) {
	cfg, err := cleaner.LoadJunkConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки конфига: %v\n", err)
		return
	}

	records := cfg.GetNonDeletedRecords()
	if len(records) == 0 {
		fmt.Println("Нет записей для очистки.")
		return
	}

	fmt.Printf("Найдено %d записей мусора для автоматической очистки.\n", len(records))

	// Группируем по папкам
	byDir := map[string][]cleaner.JunkRecord{}
	for _, r := range records {
		dir := filepath.Dir(r.Path)
		byDir[dir] = append(byDir[dir], r)
	}

	var cleanedBytes int64
	var cleanedFiles int
	var errors []string

	for dir, recs := range byDir {
		fmt.Printf("\nОчистка папки: %s (%d файлов)\n", dir, len(recs))

		for _, r := range recs {
			if verbose {
				fmt.Printf("  → %s\n", r.Path)
			}
			if err := os.Remove(r.Path); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", r.Path, err))
				if verbose {
					fmt.Printf("    ! Ошибка: %v\n", err)
				}
			} else {
				cleanedBytes += r.Size
				cleanedFiles++
				// Помечаем как удалённое
				cfg.MarkDeleted(r.Path, r.CategoryID)
				if verbose {
					fmt.Printf("    ✓ Удалён\n")
				}
			}
		}
	}

	// Сохраняем конфиг
	if err := cleaner.SaveJunkConfig(cfg, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сохранения конфига: %v\n", err)
	}

	fmt.Printf("\nГотово. Освобождено: %s в %d файлах.\n", cleaner.Human(cleanedBytes), cleanedFiles)
	if len(errors) > 0 {
		fmt.Printf("Ошибок: %d\n", len(errors))
		if verbose {
			for _, e := range errors {
				fmt.Printf("  ! %s\n", e)
			}
		}
	}
}
