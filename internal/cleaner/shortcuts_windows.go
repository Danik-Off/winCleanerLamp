//go:build windows

package cleaner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// isShortcutCandidate — ярлыком в Windows считается файл .lnk.
func isShortcutCandidate(p string, d os.DirEntry) bool {
	return !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".lnk")
}

// DefaultShortcutRoots — стандартные места, где Windows хранит ярлыки.
func DefaultShortcutRoots() []string {
	var roots []string
	add := func(p string) {
		if p != "" {
			roots = append(roots, p)
		}
	}
	add(ExpandPath(`%USERPROFILE%\Desktop`))
	add(`C:\Users\Public\Desktop`)
	add(ExpandPath(`%APPDATA%\Microsoft\Windows\Start Menu\Programs`))
	add(`C:\ProgramData\Microsoft\Windows\Start Menu\Programs`)
	return roots
}

// shortcutProbe — сырой ответ PowerShell на один ярлык.
type shortcutProbe struct {
	Path   string `json:"path"`
	Target string `json:"target"`
	Exists bool   `json:"exists"`
}

// shortcutBatchSize — сколько ярлыков разрешать за один вызов PowerShell.
// Реальный рабочий стол + меню Пуск (общее и пользовательское, со всеми
// подпапками производителей ПО) легко даёт несколько сотен .lnk-файлов — при
// передаче всех сразу через -EncodedCommand командная строка превышает лимит
// Windows (~32767 символов) и exec.Command падает с "The filename or
// extension is too long". Поэтому список бьётся на пакеты.
const shortcutBatchSize = 80

// resolveShortcutTargets разрешает TargetPath каждого .lnk через WScript.Shell
// и проверяет Test-Path на цель — пакетами по shortcutBatchSize файлов за один
// вызов PowerShell (см. shortcutBatchSize).
func resolveShortcutTargets(files []string) ([]BrokenShortcut, error) {
	var broken []BrokenShortcut
	for start := 0; start < len(files); start += shortcutBatchSize {
		end := start + shortcutBatchSize
		if end > len(files) {
			end = len(files)
		}
		batchBroken, err := resolveShortcutBatch(files[start:end])
		if err != nil {
			return broken, err
		}
		broken = append(broken, batchBroken...)
	}
	return broken, nil
}

// resolveShortcutBatch — один вызов PowerShell на пакет файлов (см. resolveShortcutTargets).
func resolveShortcutBatch(files []string) ([]BrokenShortcut, error) {
	var sb strings.Builder
	sb.WriteString("$sh = New-Object -ComObject WScript.Shell\n$items = @(\n")
	for i, f := range files {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString("'" + strings.ReplaceAll(f, "'", "''") + "'")
	}
	sb.WriteString(`
)
$results = foreach ($p in $items) {
  try {
    $lnk = $sh.CreateShortcut($p)
    $target = $lnk.TargetPath
    $exists = $true
    if ($target -and $target.Trim() -ne '') { $exists = Test-Path -LiteralPath $target }
    [PSCustomObject]@{ path = $p; target = $target; exists = $exists }
  } catch {
    [PSCustomObject]@{ path = $p; target = ''; exists = $true }
  }
}
$results | ConvertTo-Json -Compress
`)

	out, err := runPowerShellEncodedOutput(sb.String())
	if err != nil {
		return nil, fmt.Errorf("разрешение ярлыков: %w", err)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var probes []shortcutProbe
	if strings.HasPrefix(out, "[") {
		if err := json.Unmarshal([]byte(out), &probes); err != nil {
			return nil, fmt.Errorf("разбор JSON от PowerShell: %w", err)
		}
	} else {
		// ConvertTo-Json отдаёт голый объект (не массив), если элемент один.
		var one shortcutProbe
		if err := json.Unmarshal([]byte(out), &one); err != nil {
			return nil, fmt.Errorf("разбор JSON от PowerShell: %w", err)
		}
		probes = []shortcutProbe{one}
	}

	broken := make([]BrokenShortcut, 0, len(probes))
	for _, p := range probes {
		if p.Exists {
			continue
		}
		broken = append(broken, BrokenShortcut{
			Path:       p.Path,
			TargetPath: p.Target,
			Reason:     "цель ярлыка не найдена на диске",
		})
	}
	return broken, nil
}
