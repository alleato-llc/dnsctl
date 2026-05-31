.PHONY: build build-helper build-darwin build-linux build-all install clean run test

BINARY_NAME=dnsctl
HELPER_NAME=dnsctl-helper
BUILD_DIR=bin
INSTALL_DIR=/usr/local/bin

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
