.PHONY: build build-helper build-darwin build-linux build-all install uninstall install-helper uninstall-helper gui-build gui-dmg clean run test

BINARY_NAME=dnsctl
HELPER_NAME=dnsctl-helper
BUILD_DIR=bin
INSTALL_DIR=/usr/local/bin

# Wails desktop GUI (separate module under gui/).
GUI_DIR=gui
GUI_APP=$(GUI_DIR)/build/bin/dnsctl-gui.app
GUI_DMG=$(GUI_DIR)/build/bin/dnsctl.dmg

build: build-helper
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dnsctl

build-helper:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(HELPER_NAME) ./cmd/dnsctl-helper

build-darwin:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/dnsctl
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/dnsctl

build-linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/dnsctl
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/dnsctl

build-all: build-darwin build-linux

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)"
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed successfully"

uninstall:
	@echo "Removing $(BINARY_NAME) from $(INSTALL_DIR)"
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstalled successfully"

# Install/remove the privileged helper LaunchDaemon (enables password-less
# DNS/hosts changes for non-root dnsctl). Requires sudo.
install-helper: build-helper
	@sudo packaging/install-helper.sh $(BUILD_DIR)/$(HELPER_NAME)

uninstall-helper:
	@sudo packaging/uninstall-helper.sh

# Build the Wails GUI .app (needs the Wails CLI + npm; see docs/INSTALL.md).
gui-build:
	cd $(GUI_DIR) && wails build

# Package the GUI .app into a DMG. Uses create-dmg (drag-to-Applications layout)
# when available, otherwise falls back to the built-in hdiutil. The GUI still
# needs the privileged helper installed separately (make install-helper) — it is
# not bundled in the DMG.
gui-dmg: gui-build
	@rm -f $(GUI_DMG)
	@if command -v create-dmg >/dev/null 2>&1; then \
		create-dmg --volname "dnsctl" --app-drop-link 450 120 "$(GUI_DMG)" "$(GUI_APP)"; \
	else \
		echo "create-dmg not found (brew install create-dmg for a nicer installer); using hdiutil"; \
		hdiutil create -volname "dnsctl" -srcfolder "$(GUI_APP)" -ov -format UDZO "$(GUI_DMG)"; \
	fi
	@echo "Built $(GUI_DMG)"

clean:
	@rm -rf $(BUILD_DIR)
	@go clean

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test -v ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

deps:
	go mod download
	go mod tidy

config:
	@mkdir -p ~/.config/dnsctl
	@if [ ! -f ~/.config/dnsctl/config.yaml ]; then \
		cp config.example.yaml ~/.config/dnsctl/config.yaml; \
		echo "Created config at ~/.config/dnsctl/config.yaml"; \
	else \
		echo "Config already exists at ~/.config/dnsctl/config.yaml"; \
	fi
