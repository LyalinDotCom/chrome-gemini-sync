// Native Messaging Protocol Handler
//
// Chrome's Native Messaging uses a simple protocol:
// - Messages are JSON
// - Each message is prefixed with a 4-byte little-endian length
// - Max message size is 1MB (1024*1024 bytes)

package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const MaxMessageSize = 1024 * 1024 // 1MB

// Message represents a Native Messaging message
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	Cols      interface{} `json:"cols,omitempty"`
	Rows      interface{} `json:"rows,omitempty"`
	RequestId interface{} `json:"requestId,omitempty"`
	Action    string      `json:"action,omitempty"`
	Params    interface{} `json:"params,omitempty"`
	Success   bool        `json:"success,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// ReadNativeMessage reads a length-prefixed JSON message from the reader
func ReadNativeMessage(r io.Reader) (*Message, error) {
	// Read 4-byte length prefix (little-endian)
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}

	if length > MaxMessageSize {
		return nil, fmt.Errorf("message too large: %d bytes (max %d)", length, MaxMessageSize)
	}

	// Read the JSON message
	msgBytes := make([]byte, length)
	if _, err := io.ReadFull(r, msgBytes); err != nil {
		return nil, fmt.Errorf("failed to read message body: %w", err)
	}

	// Parse JSON
	var msg Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message JSON: %w", err)
	}

	return &msg, nil
}

// WriteNativeMessage writes a length-prefixed JSON message to the writer
func WriteNativeMessage(w io.Writer, msg Message) error {
	// Serialize to JSON
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if len(msgBytes) > MaxMessageSize {
		return fmt.Errorf("message too large: %d bytes (max %d)", len(msgBytes), MaxMessageSize)
	}

	// Write 4-byte length prefix (little-endian)
	length := uint32(len(msgBytes))
	if err := binary.Write(w, binary.LittleEndian, length); err != nil {
		return fmt.Errorf("failed to write message length: %w", err)
	}

	// Write the JSON message
	if _, err := w.Write(msgBytes); err != nil {
		return fmt.Errorf("failed to write message body: %w", err)
	}

	return nil
}

// BrowserRequest represents a request to be sent to Chrome
type BrowserRequest struct {
	Type      string      `json:"type"`
	Action    string      `json:"action"`
	Params    interface{} `json:"params,omitempty"`
	RequestId string      `json:"requestId"`
}

// BrowserResponse represents a response from Chrome
type BrowserResponse struct {
	Type      string      `json:"type"`
	RequestId string      `json:"requestId"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}
