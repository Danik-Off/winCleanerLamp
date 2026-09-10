//go:build !linux

// Заглушка для сборки под другими ОС, см. wincli/unsupported.go.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "lincli — ядро для Linux. Для этой ОС соберите ./wincli (Windows) или ./maccli (macOS).")
	os.Exit(1)
}
