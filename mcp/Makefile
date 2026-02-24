.PHONY: build clean install

BUILD_DIR := ../build/mcp
BINARY := $(BUILD_DIR)/taskwondo-mcp

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)

install:
	go install .
