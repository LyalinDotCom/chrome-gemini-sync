# Chrome Gemini Sync

Run Gemini CLI inside a Chrome side panel with full browser context access.

> **Disclaimer**: This is a personal prototype and exploration project. It is **not an official product**, comes with **no support or warranty**, and is provided as-is. I am **not accepting pull requests** or feature requests. Feel free to fork if you want to build on it.

## Known Issues & Warnings

- **Visual bugs**: Gemini CLI has various visual quirks when running in the browser terminal. I'm planning to explore fixes in the future.
- **macOS Apple Silicon only**: Built and tested exclusively on macOS with Apple Silicon. I have no idea if it works on Intel Macs, Windows, or Linux.
- **For developers only**: This is designed for developers who understand the risks of running these tools. I have built no safety mechanisms and provide no warranty. All source code is available for your inspection.

![Platform](https://img.shields.io/badge/platform-macOS_Apple_Silicon-blue)
![Go](https://img.shields.io/badge/go-1.21+-00ADD8)
![Node](https://img.shields.io/badge/node-20+-339933)

## Features

- **Terminal in Chrome**: Full terminal emulator in Chrome's side panel using xterm.js
- **Browser Context Access**: Gemini can read DOM, take screenshots, get console logs, and more
- **Auto-Start**: No manual server startup - opens automatically when you open the side panel
- **Native Messaging**: Uses Chrome's secure Native Messaging API instead of WebSocket

## Prerequisites

- **macOS** (Apple Silicon - Intel untested)
- **Go** 1.21 or higher
- **Node.js** 20 or higher
- **Google Chrome**
- **Gemini CLI** (optional, for MCP tools)

## Quick Start

### 1. Clone and build

```bash
git clone https://github.com/yourusername/chrome-gemini-sync.git
cd chrome-gemini-sync
./install.sh
```

This builds the Chrome extension and signs the binaries for macOS.

### 2. Load the extension in Chrome

1. Open Chrome and go to `chrome://extensions`
2. Enable **Developer mode** (toggle in top right)
3. Click **Load unpacked**
4. Select the `chrome-extension` folder from this repo
5. **Copy the extension ID** (32-character string shown under the extension name)

### 3. Complete installation

```bash
./install.sh <your-extension-id>
```

Example:
```bash
./install.sh abcdefghijklmnopqrstuvwxyzabcdef
```

### 4. Reload and open

1. Go back to `chrome://extensions`
2. Click the **reload** icon on "Chrome Gemini Sync"
3. Click the extension icon in your toolbar to open the side panel

The terminal starts automatically!

## Architecture

```
Chrome Extension (Side Panel)
       │
       │ Native Messaging (auto-starts)
       ▼
Native Host (Go binary)
├── PTY Manager (runs shell)
├── Browser Bridge
└── Unix Socket Server
       ▲
       │ Unix Socket
       ▼
MCP Server (--mcp-mode)
       ▲
       │ MCP Protocol
       ▼
Gemini CLI
```

**Key insight**: Chrome's Native Messaging API automatically starts the native host when the extension connects. No manual "start the server" step needed.

## Project Structure

```
chrome-gemini-sync/
├── install.sh                 # One-command setup
├── Makefile                   # Build commands
├── chrome-extension/          # Chrome Extension
│   ├── manifest.json
│   ├── src/
│   │   ├── background/        # Native Messaging client
│   │   ├── sidepanel/         # Terminal UI (xterm.js)
│   │   └── types/
│   └── dist/                  # Built files
├── native-host/               # Go binary
│   ├── main.go                # Entry point
│   ├── native_messaging.go    # Chrome protocol
│   ├── pty_manager.go         # Terminal
│   ├── socket_server.go       # MCP bridge
│   ├── mcp_server.go          # MCP tools
│   └── browser_bridge.go      # Request routing
├── gemini-extension.json      # Gemini CLI extension config
└── gemini-extension.md        # MCP tool documentation
```

## MCP Tools Available

When using Gemini CLI with this extension, you get these browser context tools:

| Tool | Description |
|------|-------------|
| `get_page_text` | Get visible text content (best for reading pages) |
| `get_browser_dom` | Get DOM/HTML content of active tab |
| `get_browser_url` | Get URL and title |
| `get_browser_selection` | Get highlighted text |
| `capture_browser_screenshot` | Take a screenshot |
| `execute_browser_script` | Run JavaScript and get results |
| `modify_dom` | Modify page elements |
| `get_console_logs` | Get console errors/warnings |
| `inspect_page` | Analyze page complexity |
| `save_page_to_file` | Download large pages for offline analysis |

## Uninstall

To completely remove Chrome Gemini Sync:

```bash
./uninstall.sh
```

This removes the native host, messaging manifest, build artifacts, and Gemini CLI extension link. It will prompt you to manually remove the Chrome extension.

## Development

```bash
# Build everything
make build

# Development mode (watch for changes)
make dev

# Clean build artifacts
make clean

# Run tests
make test
```

## Troubleshooting

### "Native host not found" error

Run the install script with your extension ID to register the native host:
```bash
./install.sh <your-extension-id>
```

### Extension won't connect

1. Make sure you ran `./install.sh <extension-id>` (with your ID)
2. Reload the extension in `chrome://extensions`
3. Check logs at `/tmp/gemini-browser-host.log`

### Terminal not responding

1. Close and reopen the side panel
2. Check if the native host is running: `ps aux | grep gemini-browser`

### Text duplication in Gemini CLI output

If you see duplicated text when Gemini CLI responds (same lines appearing twice), this is caused by Gemini CLI's **alternate buffer mode** which doesn't work well in embedded terminal environments.

**Fix:** Add this to your `~/.gemini/settings.json`:

```json
{
  "ui": {
    "useAlternateBuffer": false
  }
}
```

If the file doesn't exist, create it. If it already has content, add the `"ui"` section to it.

This disables the alternate screen buffer while keeping all other UI elements (banner, tips, colors, etc.).

## How It Works

1. **Chrome Extension** opens side panel → connects via Native Messaging
2. **Native Host** starts automatically → spawns PTY (shell)
3. **Terminal I/O** flows: xterm.js ↔ Native Messaging ↔ PTY
4. **Gemini CLI** calls MCP tools → connects to native host via Unix socket
5. **Browser requests** flow: MCP → Socket → Native Host → Chrome APIs → Response
