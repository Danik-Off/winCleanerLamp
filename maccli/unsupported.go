//go:build !darwin

// Заглушка для сборки под другими ОС, см. wincli/unsupported.go.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "maccli — ядро для macOS. Для этой ОС соберите ./wincli (Windows) или ./lincli (Linux).")
	os.Exit(1)
}
