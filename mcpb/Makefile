.PHONY: build clean

BUILD_DIR := ../build/mcpb
MCP_BIN_WIN := ../build/mcp/taskwondo-mcp-windows-amd64.exe
MCP_BIN_DARWIN := ../build/mcp/taskwondo-mcp-darwin-arm64
VERSION ?=
BUNDLE_NAME := $(if $(VERSION),taskwondo-v$(VERSION).mcpb,taskwondo.mcpb)
BUNDLE := $(BUILD_DIR)/$(BUNDLE_NAME)

build: $(MCP_BIN_WIN) $(MCP_BIN_DARWIN) ## Build the .mcpb bundle (Windows + macOS)
	rm -rf $(BUILD_DIR)/staging
	mkdir -p $(BUILD_DIR)/staging/server
	cp manifest.json $(BUILD_DIR)/staging/
	cp $(MCP_BIN_WIN) $(BUILD_DIR)/staging/server/taskwondo-mcp.exe
	cp $(MCP_BIN_DARWIN) $(BUILD_DIR)/staging/server/taskwondo-mcp
	chmod +x $(BUILD_DIR)/staging/server/taskwondo-mcp
	cd $(BUILD_DIR)/staging && zip -r ../$(BUNDLE_NAME) .
	rm -rf $(BUILD_DIR)/staging
	@echo ""
	@echo "Bundle created: $(BUNDLE)"
	@ls -lh $(BUNDLE)

$(MCP_BIN_WIN):
	$(MAKE) -C ../mcp build-windows

$(MCP_BIN_DARWIN):
	$(MAKE) -C ../mcp build-darwin

clean:
	rm -rf $(BUILD_DIR)
