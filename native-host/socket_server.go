// Unix Socket Server
//
// Provides a Unix domain socket for MCP clients to connect to.
// When running in MCP mode, the client connects to this socket
// to communicate with the Chrome-connected native host.

package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"os"
	"sync"
)

// SocketServer manages the Unix socket for MCP client connections
type SocketServer struct {
	path     string
	bridge   *BrowserBridge
	listener net.Listener
	clients  map[net.Conn]bool
	mutex    sync.Mutex
	running  bool
}

// NewSocketServer creates a new socket server
func NewSocketServer(path string, bridge *BrowserBridge) *SocketServer {
	return &SocketServer{
		path:    path,
		bridge:  bridge,
		clients: make(map[net.Conn]bool),
	}
}

// Start starts the socket server
func (s *SocketServer) Start() error {
	// Remove old socket if exists
	os.Remove(s.path)

	var err error
	s.listener, err = net.Listen("unix", s.path)
	if err != nil {
		return err
	}

	// Make socket world-readable/writable for MCP clients
	os.Chmod(s.path, 0777)

	s.running = true
	log.Printf("[Socket] Listening on %s", s.path)

	for s.running {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.running {
				log.Printf("[Socket] Accept error: %v", err)
			}
			continue
		}

		s.mutex.Lock()
		s.clients[conn] = true
		s.mutex.Unlock()

		log.Println("[Socket] MCP client connected")
		go s.handleClient(conn)
	}

	return nil
}

// handleClient handles a connected MCP client
func (s *SocketServer) handleClient(conn net.Conn) {
	defer func() {
		s.mutex.Lock()
		delete(s.clients, conn)
		s.mutex.Unlock()
		conn.Close()
		log.Println("[Socket] MCP client disconnected")
	}()

	reader := bufio.NewReader(conn)
	for {
		// Read line (each MCP request is a JSON line)
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}

		// Parse the socket message
		var socketMsg SocketMessage
		if err := json.Unmarshal(line, &socketMsg); err != nil {
			log.Printf("[Socket] Failed to parse message: %v", err)
			continue
		}

		// Handle the request
		response := s.handleRequest(socketMsg)

		// Send response
		respBytes, _ := json.Marshal(response)
		respBytes = append(respBytes, '\n')
		conn.Write(respBytes)
	}
}

// SocketMessage represents a message over the Unix socket
type SocketMessage struct {
	Type      string      `json:"type"`
	RequestId string      `json:"requestId"`
	Action    string      `json:"action,omitempty"`
	Params    interface{} `json:"params,omitempty"`
}

// SocketResponse represents a response over the Unix socket
type SocketResponse struct {
	Type      string      `json:"type"`
	RequestId string      `json:"requestId"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// handleRequest handles a request from an MCP client
func (s *SocketServer) handleRequest(msg SocketMessage) SocketResponse {
	log.Printf("[Socket] Handling request: %s (%s)", msg.Action, msg.RequestId)

	// Forward to Chrome via the bridge
	response, err := s.bridge.Request(msg.Action, msg.Params, msg.RequestId)
	if err != nil {
		return SocketResponse{
			Type:      "browser:response",
			RequestId: msg.RequestId,
			Success:   false,
			Error:     err.Error(),
		}
	}

	return SocketResponse{
		Type:      "browser:response",
		RequestId: msg.RequestId,
		Success:   response.Success,
		Data:      response.Data,
		Error:     response.Error,
	}
}

// Stop stops the socket server
func (s *SocketServer) Stop() {
	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}

	s.mutex.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.mutex.Unlock()

	os.Remove(s.path)
}
