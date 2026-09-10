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

# Линт всех трёх ядер: файлы *_windows.go / *_linux.go / *_darwin.go
# компилируются только под свою ОС, и один прогон проверил бы одно ядро.
lint:
	GOOS=windows golangci-lint run
	GOOS=linux golangci-lint run
	GOOS=darwin golangci-lint run

# Путь модуля с /v2 обязателен: конфиг .golangci.yml в формате v2, и
# установка по старому пути поставила бы линтер v1, который его не прочитает.
lint-install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2

clean:
	go clean
	rm -f win-cleaner-lamp.exe lin-cleaner-lamp mac-cleaner-lamp
	rm -rf dist

install-deps: lint-install
