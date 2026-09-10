//go:build darwin

// Command maccli — ядро Cleaner Lamp для macOS.
//
// Набор команд и JSON-протокол те же, что у wincli, — различия ОС живут в
// internal/cleaner в файлах *_darwin.go: категории ~/Library/Caches и
// ~/Library/Logs, DerivedData/Simulator Xcode, кеш Homebrew, Корзина
// (~/.Trash), автозапуск через LaunchAgents/LaunchDaemons, список программ
// по .app-бандлам и Homebrew Cask.
package main

import "github.com/suzen/wincleanerlamp/internal/cli"

const version = "0.1.0"

func main() {
	cli.Run(cli.App{Name: "Mac Cleaner Lamp", Version: version})
}
