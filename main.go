package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
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
		listFlag    = flag.Bool("list", false, "показать все категории и выйти")
		scanFlag    = flag.Bool("scan", false, "только посчитать размер мусора (ничего не удалять)")
		cleanFlag   = flag.Bool("clean", false, "удалить мусор")
		yesFlag     = flag.Bool("yes", false, "не запрашивать подтверждение")
		verboseFlag = flag.Bool("verbose", false, "подробный вывод")
		categories  = flag.String("categories", "", "список категорий через запятую (по умолчанию — все безопасные)")
		exclude     = flag.String("exclude", "", "исключить категории (через запятую)")
		minAgeHours = flag.Int("min-age-hours", 0, "удалять файлы старше N часов (применяется ко всем категориям)")
		aggressive  = flag.Bool("aggressive", false, "включить агрессивные категории (Windows.old, $WINDOWS.~BT, event-logs, старые Downloads и т.п.)")
		leftovers   = flag.Bool("leftovers", false, "найти возможные остатки удалённых программ в AppData/ProgramData (только отчёт)")
		sysinfo     = flag.Bool("sysinfo", false, "показать размеры системных файлов (hiberfil.sys, pagefile.sys, swapfile.sys, WinSxS) и советы")
		showEmpty   = flag.Bool("show-empty", false, "показывать в таблице категории с нулевым размером")
		parallelN   = flag.Int("parallel", 8, "число параллельных сканеров (1 = последовательно)")
		showVer     = flag.Bool("version", false, "показать версию")
	)
	flag.Usage = usage
	flag.Parse()

	if *showVer {
		fmt.Printf("winCleanerLamp %s\n", version)
		return
	}

	if runtime.GOOS != "windows" {
		fmt.Fprintln(os.Stderr, "Предупреждение: это приложение предназначено для Windows. На другой ОС большинство путей будут отсутствовать.")
	}

	all := cleaner.AllTargets()

	if *listFlag {
		printList(all)
		return
	}

	if *sysinfo {
		runSysInfo()
		if !*scanFlag && !*cleanFlag && !*leftovers {
			return
		}
	}

	if *leftovers {
		runLeftovers()
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
		Verbose: *verboseFlag,
		MinAge:  time.Duration(*minAgeHours) * time.Hour,
		Logger:  func(s string) { fmt.Println(s) },
	}

	// Сначала всегда делаем сканирование, чтобы показать сводку.
	fmt.Printf("Сканирую %d категори%s...\n", len(selected), pluralRu(len(selected)))
	scanOpts := opts
	scanOpts.DryRun = true
	scanOpts.Verbose = false

	start := time.Now()
	rows := parallelScan(selected, scanOpts, *parallelN)
	var totalBytes int64
	var totalFiles int
	for _, r := range rows {
		totalBytes += r.Bytes
		totalFiles += r.Files
	}
	fmt.Printf("Сканирование заняло %.1f сек.\n\n", time.Since(start).Seconds())
	printTable(rows, totalBytes, totalFiles, *showEmpty)

	if opts.DryRun {
		return
	}

	// Подтверждение
	if !*yesFlag {
		fmt.Printf("\nУдалить перечисленное? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" && line != "д" && line != "да" {
			fmt.Println("Отменено.")
			return
		}
	}

	fmt.Println("\nОчистка...")
	var cleanedBytes int64
	var cleanedFiles int
	var allErrors []string
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
		fmt.Printf("[%d/%d] → %s\n", i+1, len(selected), t.Name)
		r := cleaner.Process(t, opts)
		cleanedBytes += r.Bytes
		cleanedFiles += r.Files
		if len(r.Errors) > 0 {
			allErrors = append(allErrors, r.Errors...)
			if *verboseFlag {
				for _, e := range r.Errors {
					fmt.Printf("   ! %s\n", e)
				}
			}
		}
		fmt.Printf("   освобождено: %s (%d файл%s)\n", cleaner.Human(r.Bytes), r.Files, pluralFilesRu(r.Files))
	}

	fmt.Printf("\nГотово. Освобождено: %s в %d файлах.\n", cleaner.Human(cleanedBytes), cleanedFiles)
	if len(allErrors) > 0 {
		fmt.Printf("Ошибок/пропусков: %d (часть файлов могла быть занята или требует прав администратора).\n", len(allErrors))
		if !*verboseFlag {
			fmt.Println("Запустите с --verbose для подробностей.")
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `winCleanerLamp %s — CLI очиститель мусора Windows.

Использование:
  wincleanerlamp --scan                      показать сколько можно освободить
  wincleanerlamp --clean                     очистить (с подтверждением)
  wincleanerlamp --clean --yes               очистить без подтверждения
  wincleanerlamp --list                      список категорий
  wincleanerlamp --clean --categories user-temp,prefetch,recycle-bin
  wincleanerlamp --clean --exclude recycle-bin,dns-cache

Флаги:
`, version)
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, `
Подсказки:
  • Для очистки C:\Windows\Temp, Prefetch, SoftwareDistribution и т.п.
    запускайте консоль от имени администратора.
  • Файлы, занятые запущенными процессами, будут пропущены.
  • Сначала используйте --scan, чтобы убедиться, что всё корректно.`)
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

func runLeftovers() {
	fmt.Println("Поиск возможных остатков удалённых программ в AppData/ProgramData...")
	fmt.Println("(это эвристика — перед удалением проверьте каждую папку вручную)")
	fmt.Println()
	cands, err := cleaner.ScanLeftovers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		return
	}
	if len(cands) == 0 {
		fmt.Println("Подозрительных папок не найдено.")
		return
	}
	var total int64
	for _, c := range cands {
		total += c.Size
	}
	fmt.Printf("  %-10s  %8s  %s\n", "РАЗМЕР", "ФАЙЛОВ", "ПАПКА")
	fmt.Printf("  %s  %s  %s\n", strings.Repeat("-", 10), strings.Repeat("-", 8), strings.Repeat("-", 50))
	for _, c := range cands {
		fmt.Printf("  %-10s  %8d  %s\n", cleaner.Human(c.Size), c.Files, c.Path)
	}
	fmt.Printf("\n  ИТОГО потенциальных остатков: %s в %d папках\n", cleaner.Human(total), len(cands))
	fmt.Println("\nДля удаления выбранных папок используйте стандартный Проводник / rmdir /s.")
	fmt.Println("Автоматическое удаление не выполняется намеренно — слишком высок риск ложных срабатываний.")
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
func parallelScan(targets []cleaner.Target, opts cleaner.Options, workers int) []cleaner.Report {
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
	progress := func(name string) {
		mu.Lock()
		done++
		fmt.Printf("\r  [%d/%d] %-60s", done, total, truncateRunes(name, 58))
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
	fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
	return results
}

func runSysInfo() {
	info := cleaner.GatherSysInfo()
	fmt.Println("Системные файлы и большие каталоги:")
	fmt.Println()
	for _, e := range info {
		size := "—"
		if e.Size > 0 {
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
