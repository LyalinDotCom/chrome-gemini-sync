# Chrome Gemini Sync

Run Gemini CLI inside a Chrome side panel with full browser context access.

![Platform](https://img.shields.io/badge/platform-macOS-blue)
![Go](https://img.shields.io/badge/go-1.21+-00ADD8)
![Node](https://img.shields.io/badge/node-20+-339933)

## Features

- **Terminal in Chrome**: Full terminal emulator in Chrome's side panel using xterm.js
- **Browser Context Access**: Gemini can read DOM, take screenshots, get console logs, and more
- **Auto-Start**: No manual server startup - opens automatically when you open the side panel
- **Native Messaging**: Uses Chrome's secure Native Messaging API instead of WebSocket

## Prerequisites

- **macOS** (Apple Silicon or Intel)
- **Go** 1.21 or higher
- **Node.js** 20 or higher
- **Google Chrome**
- **Gemini CLI** (optional, for MCP tools)

## Quick Start

### 1. Clone the repo

```bash
git clone https://github.com/yourusername/chrome-gemini-sync.git
cd chrome-gemini-sync
```

### 2. Load the Chrome extension

1. Open Chrome and go to `chrome://extensions`
2. Enable **Developer mode** (toggle in top right)
3. Click **Load unpacked**
4. Select the `chrome-extension` folder from this repo
5. **Copy the extension ID** (32-character string shown under the extension name)

### 3. Run the install script

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
| `get_browser_dom` | Get DOM content of active tab |
| `get_browser_url` | Get URL and title |
| `get_browser_selection` | Get highlighted text |
| `capture_browser_screenshot` | Take a screenshot |
| `execute_browser_script` | Run JavaScript |
| `modify_dom` | Modify page elements |
| `get_console_logs` | Get console errors/warnings |
| `inspect_page` | Analyze page complexity |

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

# Uninstall
make uninstall
```

## Troubleshooting

### "Native host not found" error

Run the install script to register the native host:
```bash
./install.sh
```

### Extension won't connect

1. Make sure you ran `./install.sh`
2. Reload the extension in `chrome://extensions`
3. Check logs at `/tmp/gemini-browser-host.log`

### Terminal not responding

1. Close and reopen the side panel
2. Check if the native host is running: `ps aux | grep gemini-browser`

## How It Works

1. **Chrome Extension** opens side panel → connects via Native Messaging
2. **Native Host** starts automatically → spawns PTY (shell)
3. **Terminal I/O** flows: xterm.js ↔ Native Messaging ↔ PTY
4. **Gemini CLI** calls MCP tools → connects to native host via Unix socket
5. **Browser requests** flow: MCP → Socket → Native Host → Chrome APIs → Response

## License

MIT
