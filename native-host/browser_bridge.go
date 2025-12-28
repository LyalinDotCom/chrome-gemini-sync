// Browser Bridge
//
// Coordinates communication between MCP clients (via socket)
// and the Chrome extension (via Native Messaging).
// Manages pending requests and routes responses.

package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

const RequestTimeout = 30 * time.Second

// BrowserBridge manages request/response correlation
type BrowserBridge struct {
	pending map[string]chan *Message
	mutex   sync.RWMutex
}

// NewBrowserBridge creates a new browser bridge
func NewBrowserBridge() *BrowserBridge {
	return &BrowserBridge{
		pending: make(map[string]chan *Message),
	}
}

// Request sends a request to Chrome and waits for response
func (b *BrowserBridge) Request(action string, params interface{}, requestId string) (*Message, error) {
	if requestId == "" {
		requestId = uuid.New().String()
	}

	// Create response channel
	respChan := make(chan *Message, 1)
	b.mutex.Lock()
	b.pending[requestId] = respChan
	b.mutex.Unlock()

	// Ensure cleanup
	defer func() {
		b.mutex.Lock()
		delete(b.pending, requestId)
		b.mutex.Unlock()
	}()

	// Send request to Chrome via Native Messaging
	req := Message{
		Type:      "browser:request",
		Action:    action,
		Params:    params,
		RequestId: requestId,
	}

	log.Printf("[Bridge] Sending request to Chrome: %s (%s)", action, requestId)
	if err := WriteNativeMessage(os.Stdout, req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		log.Printf("[Bridge] Received response for: %s", requestId)
		return resp, nil
	case <-time.After(RequestTimeout):
		return nil, fmt.Errorf("request timeout after %v", RequestTimeout)
	}
}

// HandleResponse routes a response from Chrome to the waiting request
func (b *BrowserBridge) HandleResponse(requestId string, msg Message) {
	b.mutex.RLock()
	respChan, ok := b.pending[requestId]
	b.mutex.RUnlock()

	if ok {
		select {
		case respChan <- &msg:
			log.Printf("[Bridge] Routed response for: %s", requestId)
		default:
			log.Printf("[Bridge] Response channel full for: %s", requestId)
		}
	} else {
		log.Printf("[Bridge] No pending request for: %s", requestId)
	}
}

// GetPendingCount returns the number of pending requests
func (b *BrowserBridge) GetPendingCount() int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return len(b.pending)
}
