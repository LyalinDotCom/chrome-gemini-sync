// PTY Manager
//
// Manages the pseudo-terminal for running the shell.
// Handles spawning, I/O, and resizing.

package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

// PTYManager manages a pseudo-terminal
type PTYManager struct {
	cmd       *exec.Cmd
	ptmx      *os.File
	outputCh  chan string
	running   bool
	mutex     sync.Mutex
	closeChan chan struct{}
}

// NewPTYManager creates a new PTY manager
func NewPTYManager() *PTYManager {
	return &PTYManager{
		outputCh:  make(chan string, 100),
		closeChan: make(chan struct{}),
	}
}

// Start starts the PTY with a shell
func (p *PTYManager) Start() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Determine shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh" // Default to zsh on macOS
	}

	// Create command
	p.cmd = exec.Command(shell)
	p.cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)

	// Start with PTY
	var err error
	p.ptmx, err = pty.Start(p.cmd)
	if err != nil {
		return err
	}

	p.running = true
	log.Printf("[PTY] Started shell: %s", shell)

	// Read PTY output in background
	go p.readOutput()

	// Wait for process exit in background
	go func() {
		err := p.cmd.Wait()
		log.Printf("[PTY] Shell exited: %v", err)
		p.mutex.Lock()
		p.running = false
		p.mutex.Unlock()
		close(p.closeChan)
	}()

	return nil
}

// readOutput reads from PTY and sends to output channel
func (p *PTYManager) readOutput() {
	buf := make([]byte, 4096)
	for {
		n, err := p.ptmx.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("[PTY] Read error: %v", err)
			}
			return
		}
		if n > 0 {
			select {
			case p.outputCh <- string(buf[:n]):
			default:
				// Channel full, drop data
				log.Println("[PTY] Output channel full, dropping data")
			}
		}
	}
}

// Write sends data to the PTY
func (p *PTYManager) Write(data []byte) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.ptmx == nil {
		return nil
	}

	_, err := p.ptmx.Write(data)
	return err
}

// Resize resizes the PTY
func (p *PTYManager) Resize(cols, rows int) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.ptmx == nil {
		return nil
	}

	log.Printf("[PTY] Resizing to %dx%d", cols, rows)
	return pty.Setsize(p.ptmx, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

// OutputChan returns the output channel
func (p *PTYManager) OutputChan() <-chan string {
	return p.outputCh
}

// IsRunning returns whether the PTY is running
func (p *PTYManager) IsRunning() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.running
}

// Stop stops the PTY
func (p *PTYManager) Stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.ptmx != nil {
		p.ptmx.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
	p.running = false
}
