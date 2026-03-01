.PHONY: build build-windows clean install

BUILD_DIR := ../build/mcp
BINARY := $(BUILD_DIR)/taskwondo-mcp
BINARY_WIN := $(BUILD_DIR)/taskwondo-mcp.exe

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BINARY) .

build-windows:
	mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_WIN) .

clean:
	rm -f $(BINARY) $(BINARY_WIN)

install:
	go install .
