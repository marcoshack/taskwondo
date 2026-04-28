.PHONY: build build-linux build-darwin build-windows clean install

BUILD_DIR := ../build/mcp
BIN_LINUX := $(BUILD_DIR)/taskwondo-mcp-linux-amd64
BIN_DARWIN := $(BUILD_DIR)/taskwondo-mcp-darwin-arm64
BIN_WINDOWS := $(BUILD_DIR)/taskwondo-mcp-windows-amd64.exe

build: build-linux build-darwin build-windows

build-linux:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BIN_LINUX) .

build-darwin:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BIN_DARWIN) .

build-windows:
	mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BIN_WINDOWS) .

clean:
	rm -f $(BIN_LINUX) $(BIN_DARWIN) $(BIN_WINDOWS)

install:
	go install .
