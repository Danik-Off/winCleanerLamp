package cleaner

import (
	"fmt"
	"strings"
)

// AutostartSource — откуда взята запись автозагрузки. Набор значений свой у
// каждой ОС (реестр Run и планировщик в Windows, ~/.config/autostart и
// systemd --user в Linux, LaunchAgents/LaunchDaemons в macOS) и объявлен в
// autostart_windows.go / autostart_linux.go / autostart_darwin.go — там же
// живут ListAutostartEntries, ToggleAutostart, AutostartSourceOrder и
// AutostartSourceLabel.
type AutostartSource string

// AutostartEntry — одна запись автозагрузки. Формат одинаков для всех ядер,
// поэтому GUI разбирает JSON --autostart-list без оглядки на ОС.
type AutostartEntry struct {
	ID        string          `json:"id"` // <source>|<name> — используется для ToggleAutostart
	Source    AutostartSource `json:"source"`
	Name      string          `json:"name"`
	Command   string          `json:"command,omitempty"`
	Location  string          `json:"location"`
	Enabled   bool            `json:"enabled"`
	CanToggle bool            `json:"canToggle"`
}

// parseAutostartID разбирает ID вида "<source>|<name>".
func parseAutostartID(id string) (AutostartSource, string, error) {
	parts := strings.SplitN(id, "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("некорректный id записи автозагрузки: %q", id)
	}
	return AutostartSource(parts[0]), parts[1], nil
}
