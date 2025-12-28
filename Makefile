.PHONY: all build clean install uninstall test dev

# Default target
all: build

# Build everything
build: build-native build-extension

# Build native host
build-native:
	@echo "Building native host..."
	cd native-host && go build -o gemini-browser-host .

# Build Chrome extension
build-extension:
	@echo "Building Chrome extension..."
	cd chrome-extension && npm install && npm run build

# Development mode - watch for changes
dev:
	@echo "Starting development mode..."
	cd chrome-extension && npm run dev

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f native-host/gemini-browser-host
	rm -rf chrome-extension/dist
	rm -rf chrome-extension/node_modules

# Install (runs install.sh)
install:
	./install.sh

# Uninstall
uninstall:
	@echo "Uninstalling Chrome Gemini Sync..."
	rm -rf "$(HOME)/Library/Application Support/ChromeGeminiSync"
	rm -f "$(HOME)/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.gemini.browser.json"
	@echo "Done. Please manually remove the extension from Chrome."

# Test native host
test-native:
	@echo "Testing native host build..."
	cd native-host && go build -o gemini-browser-host .
	@echo "Native host builds successfully"

# Test extension build
test-extension:
	@echo "Testing extension build..."
	cd chrome-extension && npm install && npm run typecheck && npm run build
	@echo "Extension builds successfully"

# Run all tests
test: test-native test-extension
	@echo "All tests passed!"

# Format code
fmt:
	cd native-host && go fmt ./...

# Lint
lint:
	cd native-host && go vet ./...
	cd chrome-extension && npm run lint

# Show help
help:
	@echo "Chrome Gemini Sync - Build Commands"
	@echo ""
	@echo "  make build      - Build everything"
	@echo "  make install    - Run install script"
	@echo "  make uninstall  - Remove installed files"
	@echo "  make dev        - Start development mode (watch)"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make test       - Run tests"
	@echo "  make lint       - Lint code"
	@echo "  make help       - Show this help"
