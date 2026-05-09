.PHONY: build test lint clean install-deps

build:
	go build -v

test:
	go test -v ./...

lint:
	golangci-lint run

lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

clean:
	go clean
	rm -f wincleanerlamp.exe

install-deps: lint-install
