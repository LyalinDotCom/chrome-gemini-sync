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

// getEnhancedPath returns PATH with common binary locations added
func getEnhancedPath() string {
	currentPath := os.Getenv("PATH")
	homeDir, _ := os.UserHomeDir()

	// Common paths where npm/homebrew install binaries on macOS
	extraPaths := []string{
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/usr/local/bin",
		"/usr/local/sbin",
		homeDir + "/.npm-global/bin",
		homeDir + "/bin",
		homeDir + "/.local/bin",
	}

	for _, p := range extraPaths {
		currentPath = p + ":" + currentPath
	}

	return currentPath
}

// Start starts the PTY with Gemini CLI
func (p *PTYManager) Start() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Enhanced PATH for finding gemini
	enhancedPath := getEnhancedPath()

	// Look for gemini with enhanced PATH
	geminiPath := ""
	for _, dir := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		candidate := dir + "/gemini"
		if _, err := os.Stat(candidate); err == nil {
			geminiPath = candidate
			break
		}
	}

	// Also try exec.LookPath with enhanced PATH
	if geminiPath == "" {
		os.Setenv("PATH", enhancedPath)
		if path, err := exec.LookPath("gemini"); err == nil {
			geminiPath = path
		}
	}

	if geminiPath == "" {
		log.Printf("[PTY] Gemini CLI not found, falling back to shell")
		return p.startShell()
	}

	// Create command for Gemini CLI
	p.cmd = exec.Command(geminiPath)
	p.cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"PATH="+enhancedPath,
		// Force color output
		"FORCE_COLOR=1",
	)

	// Start with PTY
	var err error
	p.ptmx, err = pty.Start(p.cmd)
	if err != nil {
		log.Printf("[PTY] Failed to start Gemini CLI: %v, falling back to shell", err)
		return p.startShell()
	}

	p.running = true
	log.Printf("[PTY] Started Gemini CLI: %s", geminiPath)

	// Read PTY output in background
	go p.readOutput()

	// Wait for process exit in background
	go func() {
		err := p.cmd.Wait()
		log.Printf("[PTY] Gemini CLI exited: %v", err)
		p.mutex.Lock()
		p.running = false
		p.mutex.Unlock()
		close(p.closeChan)
	}()

	return nil
}

// startShell starts a fallback shell (used when Gemini CLI is not available)
func (p *PTYManager) startShell() error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}

	// Start as login shell for proper initialization
	p.cmd = exec.Command(shell, "-l")
	p.cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"PATH="+getEnhancedPath(),
	)

	var err error
	p.ptmx, err = pty.Start(p.cmd)
	if err != nil {
		return err
	}

	p.running = true
	log.Printf("[PTY] Started fallback shell: %s -l", shell)

	go p.readOutput()

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
