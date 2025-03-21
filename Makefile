.PHONY: build clean windows linux macos all prepare release help

.DEFAULT_GOAL := help

build:
	go build -o goread ./cmd/goread

all: windows linux macos

prepare:
	mkdir -p release

windows: prepare
	GOOS=windows GOARCH=amd64 go build -o release/goread-windows-amd64.exe ./cmd/goread
	GOOS=windows GOARCH=386 go build -o release/goread-windows-386.exe ./cmd/goread

linux: prepare
	GOOS=linux GOARCH=amd64 go build -o release/goread-linux-amd64 ./cmd/goread
	GOOS=linux GOARCH=386 go build -o release/goread-linux-386 ./cmd/goread
	GOOS=linux GOARCH=arm64 go build -o release/goread-linux-arm64 ./cmd/goread
	GOOS=linux GOARCH=arm go build -o release/goread-linux-arm ./cmd/goread

macos: prepare
	GOOS=darwin GOARCH=amd64 go build -o release/goread-darwin-amd64 ./cmd/goread
	GOOS=darwin GOARCH=arm64 go build -o release/goread-darwin-arm64 ./cmd/goread

clean:
	rm -f goread
	rm -rf release

help:
	@echo "Usage: make [target]"
	@echo "Targets:"
	@echo "  build    - Build the binary for the current platform"
	@echo "  windows  - Build the binary for Windows"
	@echo "  linux    - Build the binary for Linux"
	@echo "  macos    - Build the binary for macOS"
	@echo "  all      - Build the binary for all platforms"
	@echo "  clean    - Clean the build artifacts"
	@echo "  help     - Show this help information"
