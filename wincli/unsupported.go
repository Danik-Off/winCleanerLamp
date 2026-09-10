//go:build !windows

// Заглушка, чтобы `go build ./...` и `go vet ./...` под другой ОС не падали
// на «build constraints exclude all Go files». Для Linux и macOS собирайте
// соответствующие ядра: ./lincli и ./maccli.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "wincli — ядро для Windows. Для этой ОС соберите ./lincli (Linux) или ./maccli (macOS).")
	os.Exit(1)
}
