.PHONY: build build-all build-win build-lin build-mac test vet-all lint lint-install clean install-deps

# Ядро текущей ОС. Имя win-cleaner-lamp.exe сохранено — его ищет GUI.
build:
ifeq ($(OS),Windows_NT)
	go build -v -o win-cleaner-lamp.exe ./wincli
else ifeq ($(shell uname -s),Darwin)
	go build -v -o mac-cleaner-lamp ./maccli
else
	go build -v -o lin-cleaner-lamp ./lincli
endif

# Все три ядра из-под любой ОС (cgo не используется, кросс-сборка работает).
build-all: build-win build-lin build-mac

build-win:
	GOOS=windows go build -ldflags "-s -w" -o dist/win-cleaner-lamp.exe ./wincli

build-lin:
	GOOS=linux go build -ldflags "-s -w" -o dist/lin-cleaner-lamp ./lincli

build-mac:
	GOOS=darwin go build -ldflags "-s -w" -o dist/mac-cleaner-lamp ./maccli

test:
	go test -v ./...

# Кросс-проверка: код каждого ядра должен компилироваться и проходить vet
# под свою ОС, иначе поломку заметит только CI соответствующей платформы.
vet-all:
	go vet ./...
	GOOS=linux go vet ./...
	GOOS=darwin go vet ./...

lint:
	golangci-lint run

lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

clean:
	go clean
	rm -f win-cleaner-lamp.exe lin-cleaner-lamp mac-cleaner-lamp
	rm -rf dist

install-deps: lint-install
