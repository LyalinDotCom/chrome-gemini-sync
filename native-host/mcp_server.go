// MCP Server
//
// Implements the MCP (Model Context Protocol) JSON-RPC interface.
// When run with --mcp-mode, connects to the Native Host via Unix socket
// and exposes browser context tools to Gemini CLI.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
)

// MCPServer implements the MCP protocol
type MCPServer struct {
	socketPath string
	conn       net.Conn
}

// NewMCPServer creates a new MCP server
func NewMCPServer(socketPath string) *MCPServer {
	return &MCPServer{
		socketPath: socketPath,
	}
}

// JSON-RPC types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Run starts the MCP server main loop
func (s *MCPServer) Run() {
	// Connect to the Native Host socket
	if err := s.connect(); err != nil {
		log.Printf("[MCP] Failed to connect to native host: %v", err)
		// Still handle initialize - will report error on tool calls
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("[MCP] Failed to parse request: %v", err)
			continue
		}

		log.Printf("[MCP] Received: %s", req.Method)

		// Handle the request
		response := s.handleRequest(req)
		if response != nil {
			s.sendResponse(*response)
		}
	}
}

func (s *MCPServer) connect() error {
	var err error
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		s.conn, err = net.Dial("unix", s.socketPath)
		if err == nil {
			log.Printf("[MCP] Connected to native host socket")
			return nil
		}
		log.Printf("[MCP] Waiting for native host socket... (%d/%d)", i+1, maxRetries)
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("failed to connect after %d retries: %w", maxRetries, err)
}

func (s *MCPServer) handleRequest(req JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		// No response needed for notifications
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		// Unknown method - ignore to avoid noise
		return nil
	}
}

func (s *MCPServer) handleInitialize(req JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "chrome-browser-context",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
		},
	}
}

func (s *MCPServer) handleToolsList(req JSONRPCRequest) *JSONRPCResponse {
	tools := []map[string]interface{}{
		{
			"name":        "get_browser_dom",
			"description": "Get the DOM content of the active browser tab. Returns HTML, URL, and title.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector to get specific element (default: body)",
					},
				},
			},
		},
		{
			"name":        "get_browser_url",
			"description": "Get the URL and title of the active browser tab.",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "get_browser_selection",
			"description": "Get the currently selected/highlighted text in the active browser tab.",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "capture_browser_screenshot",
			"description": "Capture a screenshot of the active browser tab. Returns base64-encoded PNG.",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "execute_browser_script",
			"description": "Execute JavaScript in the active browser tab context.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"script": map[string]interface{}{
						"type":        "string",
						"description": "JavaScript code to execute",
					},
				},
				"required": []string{"script"},
			},
		},
		{
			"name":        "modify_dom",
			"description": "Modify DOM elements. Actions: setHTML, setText, setAttribute, addClass, removeClass, remove, insertBefore, insertAfter.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector to find elements",
					},
					"action": map[string]interface{}{
						"type":        "string",
						"description": "Action to perform",
						"enum":        []string{"setHTML", "setText", "setAttribute", "removeAttribute", "addClass", "removeClass", "remove", "insertBefore", "insertAfter"},
					},
					"value": map[string]interface{}{
						"type":        "string",
						"description": "Value for the action",
					},
					"attributeName": map[string]interface{}{
						"type":        "string",
						"description": "Attribute name for setAttribute/removeAttribute",
					},
					"all": map[string]interface{}{
						"type":        "boolean",
						"description": "Apply to all matching elements (default: first only)",
					},
				},
				"required": []string{"selector", "action"},
			},
		},
		{
			"name":        "get_console_logs",
			"description": "Get console logs (errors, warnings, info) from the active tab. First call attaches debugger.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"level": map[string]interface{}{
						"type":        "string",
						"description": "Filter by level",
						"enum":        []string{"all", "error", "warning", "info"},
					},
					"clear": map[string]interface{}{
						"type":        "boolean",
						"description": "Clear logs after retrieving",
					},
				},
			},
		},
		{
			"name":        "inspect_page",
			"description": "Analyze page complexity to decide whether to download DOM to file.",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *MCPServer) handleToolsCall(req JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "Invalid params")
	}

	// Map tool names to Chrome actions
	actionMap := map[string]string{
		"get_browser_dom":           "getDom",
		"get_browser_url":           "getUrl",
		"get_browser_selection":     "getSelection",
		"capture_browser_screenshot": "screenshot",
		"execute_browser_script":    "executeScript",
		"modify_dom":                "modifyDom",
		"get_console_logs":          "getConsoleLogs",
		"inspect_page":              "inspectPage",
	}

	action, ok := actionMap[params.Name]
	if !ok {
		return s.errorResponse(req.ID, -32601, fmt.Sprintf("Unknown tool: %s", params.Name))
	}

	// Check socket connection
	if s.conn == nil {
		return s.errorResponse(req.ID, -32000,
			"Not connected to Chrome. Make sure the Chrome extension is open.")
	}

	// Send request to native host via socket
	requestId := uuid.New().String()
	socketReq := SocketMessage{
		Type:      "browser:request",
		RequestId: requestId,
		Action:    action,
		Params:    params.Arguments,
	}

	reqBytes, _ := json.Marshal(socketReq)
	reqBytes = append(reqBytes, '\n')
	if _, err := s.conn.Write(reqBytes); err != nil {
		return s.errorResponse(req.ID, -32000, fmt.Sprintf("Failed to send request: %v", err))
	}

	// Read response
	reader := bufio.NewReader(s.conn)
	respLine, err := reader.ReadBytes('\n')
	if err != nil {
		return s.errorResponse(req.ID, -32000, fmt.Sprintf("Failed to read response: %v", err))
	}

	var socketResp SocketResponse
	if err := json.Unmarshal(respLine, &socketResp); err != nil {
		return s.errorResponse(req.ID, -32000, fmt.Sprintf("Failed to parse response: %v", err))
	}

	if !socketResp.Success {
		return s.errorResponse(req.ID, -32000, socketResp.Error)
	}

	// Format response based on tool
	content := s.formatToolResult(params.Name, socketResp.Data)

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": content,
		},
	}
}

func (s *MCPServer) formatToolResult(toolName string, data interface{}) []map[string]interface{} {
	// Special handling for screenshots
	if toolName == "capture_browser_screenshot" {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if dataUrl, ok := dataMap["dataUrl"].(string); ok {
				// Extract base64 data from data URL
				if len(dataUrl) > 22 { // "data:image/png;base64,"
					base64Data := dataUrl[22:]
					return []map[string]interface{}{
						{
							"type":     "image",
							"data":     base64Data,
							"mimeType": "image/png",
						},
					}
				}
			}
		}
	}

	// Default: return as JSON text
	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return []map[string]interface{}{
		{
			"type": "text",
			"text": string(jsonBytes),
		},
	}
}

func (s *MCPServer) errorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
}

func (s *MCPServer) sendResponse(resp JSONRPCResponse) {
	respBytes, err := json.Marshal(resp)
	if err != nil {
		log.Printf("[MCP] Failed to marshal response: %v", err)
		return
	}
	fmt.Printf("%s\n", respBytes)
}
