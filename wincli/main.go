//go:build windows

// Command wincli — ядро Win Cleaner Lamp для Windows.
//
// Сама логика (разбор флагов, вывод, JSON-протокол) живёт в internal/cli и
// общая для всех трёх ядер; windows-специфика — в internal/cleaner в файлах
// *_windows.go (категории мусора, реестр Uninstall, автозагрузка, Корзина,
// .lnk-ярлыки, hiberfil/WinSxS).
//
// Собирается как win-cleaner-lamp.exe — имя, которое ожидает GUI (Electron).
package main

import "github.com/suzen/wincleanerlamp/internal/cli"

const version = "0.1.0"

func main() {
	cli.Run(cli.App{Name: "Win Cleaner Lamp", Version: version})
}
