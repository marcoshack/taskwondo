.PHONY: build clean

BUILD_DIR := ../build/mcpb
MCP_BINARY := ../build/mcp/taskwondo-mcp.exe
VERSION ?=
BUNDLE_NAME := $(if $(VERSION),taskwondo-v$(VERSION).mcpb,taskwondo.mcpb)
BUNDLE := $(BUILD_DIR)/$(BUNDLE_NAME)

build: $(MCP_BINARY) ## Build the .mcpb bundle
	rm -rf $(BUILD_DIR)/staging
	mkdir -p $(BUILD_DIR)/staging/server
	cp manifest.json $(BUILD_DIR)/staging/
	cp $(MCP_BINARY) $(BUILD_DIR)/staging/server/
	cd $(BUILD_DIR)/staging && zip -r ../$(BUNDLE_NAME) .
	rm -rf $(BUILD_DIR)/staging
	@echo ""
	@echo "Bundle created: $(BUNDLE)"
	@ls -lh $(BUNDLE)

$(MCP_BINARY):
	$(MAKE) -C ../mcp build-windows

clean:
	rm -rf $(BUILD_DIR)
