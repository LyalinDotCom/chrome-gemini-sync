/**
 * Side Panel - Terminal UI
 * Uses xterm.js to render the terminal interface
 */

import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import type { NativeMessage, ConnectionStatusMessage } from '../types/messages';

// Terminal instance
let terminal: Terminal;
let fitAddon: FitAddon;

// Connection state
let isConnected = false;

// Debounce resize
let resizeTimeout: ReturnType<typeof setTimeout> | null = null;

/**
 * Initialize the terminal
 */
function initTerminal(): void {
  const container = document.getElementById('terminal-container');
  if (!container) {
    console.error('Terminal container not found');
    return;
  }

  // Create terminal with optimized settings
  terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: 'block',
    fontSize: 13,
    fontFamily: '"Cascadia Code", "Fira Code", Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#cccccc',
      cursor: '#ffffff',
      cursorAccent: '#1e1e1e',
      selectionBackground: '#264f78',
      selectionForeground: '#ffffff',
      black: '#1e1e1e',
      red: '#f14c4c',
      green: '#4ec9b0',
      yellow: '#dcdcaa',
      blue: '#569cd6',
      magenta: '#c586c0',
      cyan: '#9cdcfe',
      white: '#d4d4d4',
      brightBlack: '#808080',
      brightRed: '#f14c4c',
      brightGreen: '#4ec9b0',
      brightYellow: '#dcdcaa',
      brightBlue: '#569cd6',
      brightMagenta: '#c586c0',
      brightCyan: '#9cdcfe',
      brightWhite: '#ffffff'
    },
    allowProposedApi: true,
    scrollback: 10000,
    tabStopWidth: 4
  });

  // Add fit addon for responsive sizing
  fitAddon = new FitAddon();
  terminal.loadAddon(fitAddon);

  // Add web links addon for clickable URLs
  const webLinksAddon = new WebLinksAddon();
  terminal.loadAddon(webLinksAddon);

  // Open terminal in container
  terminal.open(container);

  // Initial fit
  setTimeout(() => {
    fitAddon.fit();
    sendResize();
  }, 0);

  // Handle terminal input
  terminal.onData((data) => {
    if (isConnected) {
      sendMessage({
        type: 'terminal:input',
        data
      });
    }
  });

  // Handle resize with debouncing
  const debouncedResize = () => {
    if (resizeTimeout) {
      clearTimeout(resizeTimeout);
    }
    resizeTimeout = setTimeout(() => {
      fitAddon.fit();
      sendResize();
    }, 100);
  };

  const resizeObserver = new ResizeObserver(debouncedResize);
  resizeObserver.observe(container);

  // Also handle window resize
  window.addEventListener('resize', debouncedResize);

  // Write welcome message
  terminal.writeln('\x1b[1;36m╔══════════════════════════════════════════════════╗\x1b[0m');
  terminal.writeln('\x1b[1;36m║\x1b[0m  \x1b[1;33mChrome Gemini Sync Terminal\x1b[0m                      \x1b[1;36m║\x1b[0m');
  terminal.writeln('\x1b[1;36m║\x1b[0m  Connecting to native host...                     \x1b[1;36m║\x1b[0m');
  terminal.writeln('\x1b[1;36m╚══════════════════════════════════════════════════╝\x1b[0m');
  terminal.writeln('');
}

/**
 * Send terminal resize information to native host
 */
function sendResize(): void {
  if (terminal && isConnected) {
    sendMessage({
      type: 'terminal:resize',
      cols: terminal.cols,
      rows: terminal.rows
    });
  }
}

/**
 * Send message to background script
 */
function sendMessage(message: NativeMessage): void {
  chrome.runtime.sendMessage(message).catch((error) => {
    console.error('Failed to send message:', error);
  });
}

/**
 * Update connection status UI
 */
function updateConnectionStatus(status: 'connected' | 'disconnected' | 'connecting' | 'error', message?: string): void {
  const statusElement = document.getElementById('status');
  const overlay = document.getElementById('connection-overlay');
  const statusText = statusElement?.querySelector('.status-text');
  const errorDetails = overlay?.querySelector('.error-details');
  const statusMessage = overlay?.querySelector('.status-message');
  const spinner = overlay?.querySelector('.spinner');

  if (statusElement) {
    statusElement.className = 'status-indicator ' + status;
  }

  if (statusText) {
    const statusMessages: Record<string, string> = {
      connected: 'Connected',
      disconnected: 'Disconnected',
      connecting: 'Connecting...',
      error: 'Connection Failed'
    };
    statusText.textContent = statusMessages[status] || status;
  }

  if (overlay) {
    if (status === 'connected') {
      overlay.classList.add('hidden');
      isConnected = true;
      // Send initial resize
      setTimeout(sendResize, 100);
    } else {
      overlay.classList.remove('hidden');
      isConnected = false;

      if (status === 'connecting') {
        errorDetails?.classList.add('hidden');
        spinner?.classList.remove('hidden');
        if (statusMessage) statusMessage.textContent = 'Connecting to native host...';
      } else if (status === 'error' || status === 'disconnected') {
        errorDetails?.classList.remove('hidden');
        spinner?.classList.add('hidden');
        if (statusMessage) statusMessage.textContent = '';
      }
    }
  }

  // Write status to terminal
  if (terminal) {
    if (status === 'connected') {
      terminal.writeln('\x1b[1;32m✓ Connected to native host\x1b[0m');
      terminal.writeln('');
    } else if (status === 'error' && message) {
      terminal.writeln(`\x1b[1;31m✗ ${message}\x1b[0m`);
    }
  }
}

/**
 * Handle messages from background script
 */
function handleMessage(message: NativeMessage): void {
  switch (message.type) {
    case 'terminal:output':
      if (terminal && message.data) {
        terminal.write(message.data as string);
      }
      break;

    case 'connection:status':
      const statusMessage = message as unknown as ConnectionStatusMessage;
      updateConnectionStatus(statusMessage.status, statusMessage.message);
      break;

    default:
      console.log('Unknown message type:', message);
  }
}

/**
 * Set up event listeners
 */
function setupEventListeners(): void {
  // Listen for messages from background script
  chrome.runtime.onMessage.addListener((message: NativeMessage) => {
    handleMessage(message);
  });

  // Reconnect button
  document.getElementById('reconnect-btn')?.addEventListener('click', () => {
    updateConnectionStatus('connecting');
    chrome.runtime.sendMessage({ type: 'connection:status', action: 'reconnect' });
  });

  // Retry button in overlay
  document.getElementById('retry-btn')?.addEventListener('click', () => {
    updateConnectionStatus('connecting');
    chrome.runtime.sendMessage({ type: 'connection:status', action: 'reconnect' });
  });

  // Clear terminal button
  document.getElementById('clear-btn')?.addEventListener('click', () => {
    if (terminal) {
      terminal.clear();
    }
  });
}

/**
 * Check initial connection status
 */
async function checkConnectionStatus(): Promise<void> {
  try {
    await chrome.runtime.sendMessage({ type: 'ping' });
  } catch (error) {
    console.error('Background script not responding:', error);
    updateConnectionStatus('error', 'Extension not properly initialized');
  }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  initTerminal();
  setupEventListeners();
  checkConnectionStatus();
});
