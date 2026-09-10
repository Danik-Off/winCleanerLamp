//go:build linux

// Command lincli — ядро Cleaner Lamp для Linux.
//
// Набор команд и JSON-протокол те же, что у wincli, — различия ОС живут в
// internal/cleaner в файлах *_linux.go: категории по XDG-спецификации
// (~/.cache, ~/.local/share/Trash), кеши пакетных менеджеров, journald,
// snap/flatpak, автозапуск через ~/.config/autostart и systemd --user,
// список пакетов через dpkg/rpm/pacman/flatpak/snap.
package main

import "github.com/suzen/wincleanerlamp/internal/cli"

const version = "0.1.0"

func main() {
	cli.Run(cli.App{Name: "Lin Cleaner Lamp", Version: version})
}
