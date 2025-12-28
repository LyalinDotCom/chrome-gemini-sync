// Chrome-Gemini Sync Native Host
//
// This binary runs in two modes:
// 1. Native Messaging mode (default): Launched by Chrome extension
//    - Handles terminal I/O via PTY
//    - Routes browser context requests from MCP clients
//    - Creates Unix socket for MCP client connections
//
// 2. MCP Server mode (--mcp-mode): Launched by Gemini CLI
//    - Implements MCP JSON-RPC protocol
//    - Connects to Native Host via Unix socket for browser context

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const (
	SocketPath = "/tmp/gemini-browser.sock"
	LogFile    = "/tmp/gemini-browser-host.log"
)

var (
	mcpMode = flag.Bool("mcp-mode", false, "Run as MCP server (for Gemini CLI)")
	debug   = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()

	// Setup logging
	setupLogging()

	if *mcpMode {
		log.Println("[Main] Starting in MCP Server mode")
		runMCPMode()
	} else {
		log.Println("[Main] Starting in Native Messaging mode")
		runNativeMessagingMode()
	}
}

func setupLogging() {
	logFile, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Can't log to file, use stderr (but be careful - Native Messaging uses stdin/stdout)
		log.SetOutput(os.Stderr)
		return
	}
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func runNativeMessagingMode() {
	// Clean up old socket if exists
	os.Remove(SocketPath)

	// Create the bridge that coordinates everything
	bridge := NewBrowserBridge()

	// Start Unix socket server for MCP clients
	socketServer := NewSocketServer(SocketPath, bridge)
	go socketServer.Start()

	// Start PTY manager
	ptyManager := NewPTYManager()
	if err := ptyManager.Start(); err != nil {
		log.Fatalf("[Main] Failed to start PTY: %v", err)
	}

	// Connect PTY output to Native Messaging
	go func() {
		for output := range ptyManager.OutputChan() {
			msg := Message{
				Type: "terminal:output",
				Data: output,
			}
			if err := WriteNativeMessage(os.Stdout, msg); err != nil {
				log.Printf("[Main] Failed to write terminal output: %v", err)
			}
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("[Main] Shutting down...")
		ptyManager.Stop()
		socketServer.Stop()
		os.Remove(SocketPath)
		os.Exit(0)
	}()

	// Main loop: read from Chrome (Native Messaging) and dispatch
	for {
		msg, err := ReadNativeMessage(os.Stdin)
		if err != nil {
			log.Printf("[Main] Failed to read Native Message: %v", err)
			break
		}

		log.Printf("[Main] Received message type: %s", msg.Type)

		switch msg.Type {
		case "terminal:input":
			// Forward to PTY
			if data, ok := msg.Data.(string); ok {
				ptyManager.Write([]byte(data))
			}

		case "terminal:resize":
			// Resize PTY
			if cols, ok := msg.Cols.(float64); ok {
				if rows, ok := msg.Rows.(float64); ok {
					ptyManager.Resize(int(cols), int(rows))
				}
			}

		case "browser:response":
			// Forward response to waiting MCP client
			if reqID, ok := msg.RequestId.(string); ok {
				bridge.HandleResponse(reqID, *msg)
			}

		default:
			log.Printf("[Main] Unknown message type: %s", msg.Type)
		}
	}
}

func runMCPMode() {
	// In MCP mode, we connect to the Native Host's socket
	// and implement the MCP JSON-RPC protocol
	mcpServer := NewMCPServer(SocketPath)
	mcpServer.Run()
}

// GetInstallDir returns the installation directory for the native host
func GetInstallDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Library", "Application Support", "ChromeGeminiSync")
}
