#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Helper functions
info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Configuration
INSTALL_DIR="$HOME/Library/Application Support/ChromeGeminiSync"
MANIFEST_DIR="$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts"
HOST_NAME="com.gemini.browser"

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           Chrome Gemini Sync - Installer                      ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# =============================================================================
# EXTENSION ID (required argument)
# =============================================================================

EXTENSION_ID="$1"

if [[ -z "$EXTENSION_ID" ]]; then
    echo -e "${RED}ERROR: Extension ID is required${NC}"
    echo ""
    echo "Usage: ./install.sh <extension-id>"
    echo ""
    echo "To get your extension ID:"
    echo "  1. Open Chrome and go to: chrome://extensions"
    echo "  2. Enable 'Developer mode' (top right toggle)"
    echo "  3. Click 'Load unpacked' and select: chrome-extension/"
    echo "  4. Copy the ID shown under 'Chrome Gemini Sync'"
    echo "     (looks like: abcdefghijklmnopqrstuvwxyzabcdef)"
    echo ""
    echo "Then run: ./install.sh <your-extension-id>"
    exit 1
fi

# Validate extension ID format (32 lowercase letters)
if [[ ! "$EXTENSION_ID" =~ ^[a-z]{32}$ ]]; then
    error "Invalid extension ID format. Expected 32 lowercase letters.
Got: $EXTENSION_ID"
fi

info "Extension ID: $EXTENSION_ID"
echo ""

# =============================================================================
# PREREQUISITE CHECKS
# =============================================================================

echo "Checking prerequisites..."
echo ""

# Check OS - macOS only
if [[ "$(uname)" != "Darwin" ]]; then
    error "This tool only supports macOS. Detected: $(uname)"
fi
info "macOS detected: $(sw_vers -productVersion)"

# Check Chrome is installed
if [[ ! -d "/Applications/Google Chrome.app" ]]; then
    error "Google Chrome not found in /Applications. Please install Chrome first."
fi
info "Google Chrome found"

# Check Go is installed
if ! command -v go &>/dev/null; then
    echo ""
    error "Go is not installed.

To install Go on macOS:
  brew install go

Or download from: https://go.dev/dl/
Minimum version required: 1.21"
fi

# Check Go version (need 1.21+)
GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
GO_MAJOR=$(echo "$GO_VERSION" | cut -d. -f1)
GO_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)
if [[ "$GO_MAJOR" -lt 1 ]] || [[ "$GO_MAJOR" -eq 1 && "$GO_MINOR" -lt 21 ]]; then
    error "Go version $GO_VERSION is too old. Please upgrade to Go 1.21 or higher.
  brew upgrade go"
fi
info "Go $GO_VERSION found"

# Check Node.js/npm is installed
if ! command -v npm &>/dev/null; then
    echo ""
    error "npm is not installed.

To install Node.js on macOS:
  brew install node

Or download from: https://nodejs.org/
Minimum version required: Node.js 20"
fi

# Check Node.js version (need 20+)
NODE_VERSION=$(node --version | sed 's/v//' | cut -d. -f1)
if [[ "$NODE_VERSION" -lt 20 ]]; then
    error "Node.js version $(node --version) is too old. Please upgrade to Node.js 20 or higher.
  brew upgrade node"
fi
info "Node.js $(node --version) found"

echo ""
info "All prerequisites satisfied!"
echo ""

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Step 1/5: Building native host..."
cd native-host
go mod download
go build -o gemini-browser-host .
info "Native host built successfully"
cd ..

echo ""
echo "Step 2/5: Installing native host..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$MANIFEST_DIR"
# Use ditto to preserve code signature, then re-sign ad-hoc
ditto native-host/gemini-browser-host "$INSTALL_DIR/gemini-browser-host"
codesign -fs - "$INSTALL_DIR/gemini-browser-host" 2>/dev/null
chmod +x "$INSTALL_DIR/gemini-browser-host"
info "Binary installed to: $INSTALL_DIR/gemini-browser-host"

echo ""
echo "Step 3/5: Registering native messaging manifest..."
cat > "$MANIFEST_DIR/$HOST_NAME.json" << EOF
{
  "name": "$HOST_NAME",
  "description": "Chrome Gemini Sync - Terminal and browser context bridge",
  "path": "$INSTALL_DIR/gemini-browser-host",
  "type": "stdio",
  "allowed_origins": ["chrome-extension://$EXTENSION_ID/"]
}
EOF
info "Manifest registered for extension: $EXTENSION_ID"

echo ""
echo "Step 4/5: Building Chrome extension..."
cd chrome-extension
npm install --silent
npm run build
info "Chrome extension built successfully"
cd ..

echo ""
echo "Step 5/5: Linking Gemini CLI extension..."
if command -v gemini >/dev/null 2>&1; then
    gemini extensions link "$SCRIPT_DIR" 2>/dev/null && \
        info "Gemini extension linked successfully" || \
        warn "Failed to link Gemini extension. Run manually: gemini extensions link ."
else
    warn "Gemini CLI not found. After installing it, run: gemini extensions link <repo-root>"
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                   Installation Complete!                      ║"
echo "╠══════════════════════════════════════════════════════════════╣"
echo "║                                                               ║"
echo "║  Next steps:                                                  ║"
echo "║  1. Go to chrome://extensions                                 ║"
echo "║  2. Click the reload icon on 'Chrome Gemini Sync'             ║"
echo "║  3. Click the extension icon to open the side panel           ║"
echo "║                                                               ║"
echo "║  The terminal should start automatically!                     ║"
echo "║                                                               ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
