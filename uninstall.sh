#!/bin/bash

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Helper functions
info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
removed() { echo -e "${GREEN}[REMOVED]${NC} $1"; }
skipped() { echo -e "${YELLOW}[SKIPPED]${NC} $1 (not found)"; }

# Configuration (must match install.sh)
INSTALL_DIR="$HOME/Library/Application Support/ChromeGeminiSync"
MANIFEST_DIR="$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts"
HOST_NAME="com.gemini.browser"

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           Chrome Gemini Sync - Uninstaller                    ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# =============================================================================
# REMOVE NATIVE HOST
# =============================================================================

echo "Removing native host..."

# Remove native host binary and directory
if [[ -d "$INSTALL_DIR" ]]; then
    rm -rf "$INSTALL_DIR"
    removed "$INSTALL_DIR"
else
    skipped "$INSTALL_DIR"
fi

# Remove native messaging manifest
if [[ -f "$MANIFEST_DIR/$HOST_NAME.json" ]]; then
    rm -f "$MANIFEST_DIR/$HOST_NAME.json"
    removed "$MANIFEST_DIR/$HOST_NAME.json"
else
    skipped "$MANIFEST_DIR/$HOST_NAME.json"
fi

echo ""

# =============================================================================
# UNINSTALL GEMINI CLI EXTENSION
# =============================================================================

echo "Uninstalling Gemini CLI extension..."

if command -v gemini >/dev/null 2>&1; then
    # Try to uninstall by extension name
    if gemini extensions uninstall chrome-extension-sync 2>/dev/null; then
        removed "Gemini CLI extension (chrome-extension-sync)"
    else
        skipped "Gemini CLI extension (may not have been installed)"
    fi
else
    skipped "Gemini CLI extension (gemini CLI not installed)"
fi

echo ""

# =============================================================================
# CLEAN BUILD ARTIFACTS (optional)
# =============================================================================

echo "Cleaning build artifacts..."

# Remove chrome extension build output
if [[ -d "$SCRIPT_DIR/chrome-extension/dist" ]]; then
    rm -rf "$SCRIPT_DIR/chrome-extension/dist"
    removed "chrome-extension/dist"
else
    skipped "chrome-extension/dist"
fi

# Remove node_modules
if [[ -d "$SCRIPT_DIR/chrome-extension/node_modules" ]]; then
    rm -rf "$SCRIPT_DIR/chrome-extension/node_modules"
    removed "chrome-extension/node_modules"
else
    skipped "chrome-extension/node_modules"
fi

# Remove native host build output
if [[ -f "$SCRIPT_DIR/native-host/gemini-browser-host" ]]; then
    rm -f "$SCRIPT_DIR/native-host/gemini-browser-host"
    removed "native-host/gemini-browser-host"
else
    skipped "native-host/gemini-browser-host"
fi

echo ""

# =============================================================================
# KILL ANY RUNNING PROCESSES
# =============================================================================

echo "Stopping any running processes..."

if pgrep -f "gemini-browser-host" >/dev/null 2>&1; then
    pkill -f "gemini-browser-host" 2>/dev/null
    removed "Killed running gemini-browser-host processes"
else
    skipped "No running gemini-browser-host processes"
fi

echo ""

# =============================================================================
# MANUAL STEPS
# =============================================================================

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                   Uninstall Complete!                         ║"
echo "╠══════════════════════════════════════════════════════════════╣"
echo "║                                                               ║"
echo "║  Manual step required:                                        ║"
echo "║                                                               ║"
echo "║  Remove the Chrome extension:                                 ║"
echo "║  1. Open Chrome: chrome://extensions                          ║"
echo "║  2. Find 'Chrome Gemini Sync'                                 ║"
echo "║  3. Click 'Remove'                                            ║"
echo "║                                                               ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
