/**
 * Background Service Worker
 *
 * Uses Chrome Native Messaging to communicate with the native host.
 * The native host is automatically started when we connect.
 *
 * Handles:
 * - Terminal I/O (forwarding between side panel and native host PTY)
 * - Browser context requests (DOM, screenshots, console logs, etc.)
 */

import type {
  NativeMessage,
  BrowserContextRequest,
  BrowserContextResponse,
  ExtensionMessage,
} from '../types/messages';

const NATIVE_HOST_NAME = 'com.gemini.browser';

let port: chrome.runtime.Port | null = null;
let connectionStatus: 'connected' | 'disconnected' | 'connecting' | 'error' = 'disconnected';

// Console logs storage per tab (using debugger API)
interface ConsoleLogEntry {
  level: 'error' | 'warning' | 'info' | 'log' | 'debug';
  text: string;
  timestamp: number;
  url?: string;
  lineNumber?: number;
  stackTrace?: string;
}

const consoleLogs = new Map<number, ConsoleLogEntry[]>();
const attachedTabs = new Set<number>();
const MAX_LOGS_PER_TAB = 500;

/**
 * Connect to the native host
 */
function connectToNativeHost(): void {
  if (port !== null) {
    return;
  }

  console.log('[Background] Connecting to native host:', NATIVE_HOST_NAME);
  connectionStatus = 'connecting';
  broadcastToExtension({ type: 'connection:status', status: 'connecting' });

  try {
    port = chrome.runtime.connectNative(NATIVE_HOST_NAME);

    port.onMessage.addListener((message: NativeMessage) => {
      handleNativeMessage(message);
    });

    port.onDisconnect.addListener(() => {
      const error = chrome.runtime.lastError;
      console.log('[Background] Native host disconnected:', error?.message);
      port = null;
      connectionStatus = 'disconnected';

      if (error?.message?.includes('not found')) {
        broadcastToExtension({
          type: 'connection:status',
          status: 'error',
          message: 'Native host not installed. Run install.sh first.'
        });
      } else {
        broadcastToExtension({ type: 'connection:status', status: 'disconnected' });
        // Try to reconnect after a delay
        setTimeout(connectToNativeHost, 2000);
      }
    });

    // Connection established
    connectionStatus = 'connected';
    broadcastToExtension({ type: 'connection:status', status: 'connected' });
    console.log('[Background] Connected to native host');

  } catch (error) {
    console.error('[Background] Failed to connect:', error);
    connectionStatus = 'error';
    broadcastToExtension({
      type: 'connection:status',
      status: 'error',
      message: 'Failed to connect to native host'
    });
  }
}

/**
 * Send message to native host
 */
function sendToNativeHost(message: NativeMessage): boolean {
  if (port === null) {
    console.warn('[Background] Cannot send message, not connected');
    return false;
  }
  port.postMessage(message);
  return true;
}

/**
 * Broadcast message to all extension contexts (side panel)
 */
function broadcastToExtension(message: ExtensionMessage | NativeMessage): void {
  chrome.runtime.sendMessage(message).catch(() => {
    // Ignore errors when no listeners are available
  });
}

/**
 * Handle messages from native host
 */
async function handleNativeMessage(message: NativeMessage): Promise<void> {
  switch (message.type) {
    case 'terminal:output':
      // Forward terminal output to side panel
      broadcastToExtension(message);
      break;

    case 'browser:request':
      // Native host is requesting browser context
      await handleBrowserContextRequest(message as BrowserContextRequest);
      break;

    default:
      console.log('[Background] Unknown message type:', message.type);
  }
}

/**
 * Handle browser context requests from native host
 */
async function handleBrowserContextRequest(request: BrowserContextRequest): Promise<void> {
  console.log('[Background] Browser context request:', request.action);

  try {
    let response: BrowserContextResponse;

    switch (request.action) {
      case 'getDom':
        response = await getActiveTabDom(request);
        break;
      case 'getSelection':
        response = await getActiveTabSelection(request);
        break;
      case 'getUrl':
        response = await getActiveTabUrl(request);
        break;
      case 'screenshot':
        response = await captureActiveTabScreenshot(request);
        break;
      case 'executeScript':
        response = await executeScriptInTab(request);
        break;
      case 'modifyDom':
        response = await modifyDomInTab(request);
        break;
      case 'getConsoleLogs':
        response = await getConsoleLogs(request);
        break;
      case 'inspectPage':
        response = await inspectPage(request);
        break;
      default:
        response = {
          type: 'browser:response',
          requestId: request.requestId,
          success: false,
          error: `Unknown action: ${request.action}`
        };
    }

    sendToNativeHost(response);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    sendToNativeHost({
      type: 'browser:response',
      requestId: request.requestId,
      success: false,
      error: errorMessage
    });
  }
}

/**
 * Get the active tab
 */
async function getActiveTab(): Promise<chrome.tabs.Tab> {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.id) {
    throw new Error('No active tab found');
  }

  const url = tab.url || '';
  if (url.startsWith('chrome://') || url.startsWith('chrome-extension://') ||
      url.startsWith('edge://') || url.startsWith('about:') ||
      url.startsWith('devtools://')) {
    throw new Error(`Cannot access restricted page: ${url.split('/')[0]}//...`);
  }

  return tab;
}

/**
 * Get DOM from active tab
 */
async function getActiveTabDom(request: BrowserContextRequest): Promise<BrowserContextResponse> {
  const tab = await getActiveTab();

  const results = await chrome.scripting.executeScript({
    target: { tabId: tab.id! },
    func: (options: { selector?: string }) => {
      const selector = options?.selector || 'body';
      const element = document.querySelector(selector);
      if (!element) {
        return { html: null, error: `Element not found: ${selector}` };
      }
      return {
        html: element.outerHTML,
        url: window.location.href,
        title: document.title,
        selector
      };
    },
    args: [request.params as { selector?: string } || {}]
  });

  const result = results[0]?.result;
  if (result?.error) {
    return {
      type: 'browser:response',
      requestId: request.requestId,
      success: false,
      error: result.error
    };
  }

  return {
    type: 'browser:response',
    requestId: request.requestId,
    success: true,
    data: result
  };
}

/**
 * Get selected text from active tab
 */
async function getActiveTabSelection(request: BrowserContextRequest): Promise<BrowserContextResponse> {
  const tab = await getActiveTab();

  const results = await chrome.scripting.executeScript({
    target: { tabId: tab.id! },
    func: () => {
      const selection = window.getSelection();
      return {
        text: selection?.toString() || '',
        url: window.location.href,
        title: document.title
      };
    }
  });

  return {
    type: 'browser:response',
    requestId: request.requestId,
    success: true,
    data: results[0]?.result
  };
}

/**
 * Get URL of active tab
 */
async function getActiveTabUrl(request: BrowserContextRequest): Promise<BrowserContextResponse> {
  const tab = await getActiveTab();

  return {
    type: 'browser:response',
    requestId: request.requestId,
    success: true,
    data: {
      url: tab.url,
      title: tab.title,
      id: tab.id
    }
  };
}

/**
 * Capture screenshot of active tab
 */
async function captureActiveTabScreenshot(request: BrowserContextRequest): Promise<BrowserContextResponse> {
  try {
    const dataUrl = await chrome.tabs.captureVisibleTab({
      format: 'png',
      quality: 90
    });

    return {
      type: 'browser:response',
      requestId: request.requestId,
      success: true,
      data: {
        dataUrl,
        format: 'png'
      }
    };
  } catch (error) {
    return {
      type: 'browser:response',
      requestId: request.requestId,
      success: false,
      error: error instanceof Error ? error.message : 'Failed to capture screenshot'
    };
  }
}

/**
 * Execute script in active tab
 */
async function executeScriptInTab(request: BrowserContextRequest): Promise<BrowserContextResponse> {
  const tab = await getActiveTab();
  const script = (request.params as { script?: string })?.script;

  if (!script) {
    return {
      type: 'browser:response',
      requestId: request.requestId,
      success: false,
      error: 'No script provided'
    };
  }

  try {
    const wrappedScript = `
      (function() {
        try {
          ${script}
        } catch (e) {
          return { __error: e.message || 'Script execution failed' };
        }
      })();
    `;

    const results = await chrome.scripting.executeScript({
      target: { tabId: tab.id! },
      world: 'MAIN',
      func: (code: string) => {
        const scriptEl = document.createElement('script');
        scriptEl.textContent = code;
        document.documentElement.appendChild(scriptEl);
        scriptEl.remove();
        return { success: true };
      },
      args: [wrappedScript]
    });

    return {
      type: 'browser:response',
      requestId: request.requestId,
      success: true,
      data: results[0]?.result
    };
  } catch (error) {
    return {
      type: 'browser:response',
      requestId: request.requestId,
      success: false,
      error: error instanceof Error ? error.message : 'Failed to execute script'
    };
  }
}

/**
 * Modify DOM elements in active tab
 */
async function modifyDomInTab(request: BrowserContextRequest): Promise<BrowserContextResponse> {
  const tab = await getActiveTab();

  const params = request.params as {
    selector?: string;
    action?: string;
    value?: string;
    attributeName?: string;
    all?: boolean;
  };

  if (!params?.selector || !params?.action) {
    return {
      type: 'browser:response',
      requestId: request.requestId,
      success: false,
      error: 'Missing selector or action'
    };
  }

  try {
    const results = await chrome.scripting.executeScript({
      target: { tabId: tab.id! },
      func: (selector: string, action: string, value: string | null, attributeName: string | null, all: boolean) => {
        try {
          const elements = all
            ? Array.from(document.querySelectorAll(selector))
            : [document.querySelector(selector)].filter(Boolean) as Element[];

          if (elements.length === 0) {
            return { success: false, error: `No elements found: ${selector}` };
          }

          let modifiedCount = 0;
          for (const element of elements) {
            switch (action) {
              case 'setHTML':
                (element as HTMLElement).innerHTML = value || '';
                break;
              case 'setText':
                (element as HTMLElement).textContent = value || '';
                break;
              case 'setAttribute':
                if (!attributeName) return { success: false, error: 'attributeName required' };
                element.setAttribute(attributeName, value || '');
                break;
              case 'removeAttribute':
                if (!attributeName) return { success: false, error: 'attributeName required' };
                element.removeAttribute(attributeName);
                break;
              case 'addClass':
                if (!value) return { success: false, error: 'class name required' };
                element.classList.add(value);
                break;
              case 'removeClass':
                if (!value) return { success: false, error: 'class name required' };
                element.classList.remove(value);
                break;
              case 'remove':
                element.remove();
                break;
              case 'insertBefore':
                if (!value) return { success: false, error: 'HTML content required' };
                element.insertAdjacentHTML('beforebegin', value);
                break;
              case 'insertAfter':
                if (!value) return { success: false, error: 'HTML content required' };
                element.insertAdjacentHTML('afterend', value);
                break;
              default:
                return { success: false, error: `Unknown action: ${action}` };
            }
            modifiedCount++;
          }

          return { success: true, modifiedCount, message: `Modified ${modifiedCount} element(s)` };
        } catch (e) {
          return { success: false, error: e instanceof Error ? e.message : 'DOM modification failed' };
        }
      },
      args: [params.selector, params.action, params.value ?? null, params.attributeName ?? null, params.all ?? false]
    });

    const result = results[0]?.result;
    if (!result?.success) {
      return {
        type: 'browser:response',
        requestId: request.requestId,
        success: false,
        error: result?.error || 'DOM modification failed'
      };
    }

    return {
      type: 'browser:response',
      requestId: request.requestId,
      success: true,
      data: result
    };
  } catch (error) {
    return {
      type: 'browser:response',
      requestId: request.requestId,
      success: false,
      error: error instanceof Error ? error.message : 'Failed to modify DOM'
    };
  }
}

/**
 * Attach debugger to tab for console logs
 */
async function attachDebuggerToTab(tabId: number): Promise<void> {
  if (attachedTabs.has(tabId)) return;

  await chrome.debugger.attach({ tabId }, '1.3');
  attachedTabs.add(tabId);
  consoleLogs.set(tabId, []);

  await chrome.debugger.sendCommand({ tabId }, 'Log.enable');
  await chrome.debugger.sendCommand({ tabId }, 'Runtime.enable');
}

/**
 * Get console logs for active tab
 */
async function getConsoleLogs(request: BrowserContextRequest): Promise<BrowserContextResponse> {
  const tab = await getActiveTab();
  const tabId = tab.id!;

  const params = request.params as {
    level?: 'all' | 'error' | 'warning' | 'info';
    clear?: boolean;
  };

  if (!attachedTabs.has(tabId)) {
    try {
      await attachDebuggerToTab(tabId);
      await new Promise(resolve => setTimeout(resolve, 100));
    } catch (error) {
      return {
        type: 'browser:response',
        requestId: request.requestId,
        success: false,
        error: `Failed to attach debugger: ${error instanceof Error ? error.message : 'Unknown error'}`
      };
    }
  }

  let logs = consoleLogs.get(tabId) || [];

  if (params?.level && params.level !== 'all') {
    const levelMap: Record<string, string[]> = {
      'error': ['error'],
      'warning': ['warning'],
      'info': ['info', 'log']
    };
    const allowedLevels = levelMap[params.level] || [];
    logs = logs.filter(log => allowedLevels.includes(log.level));
  }

  if (params?.clear) {
    consoleLogs.set(tabId, []);
  }

  return {
    type: 'browser:response',
    requestId: request.requestId,
    success: true,
    data: {
      logs,
      tabId,
      url: tab.url,
      isCapturing: attachedTabs.has(tabId)
    }
  };
}

/**
 * Inspect page complexity
 */
async function inspectPage(request: BrowserContextRequest): Promise<BrowserContextResponse> {
  const tab = await getActiveTab();

  const results = await chrome.scripting.executeScript({
    target: { tabId: tab.id! },
    func: () => {
      const htmlLength = document.documentElement.outerHTML.length;
      const elementCount = document.getElementsByTagName('*').length;
      const bodyTextLength = document.body.innerText.length;
      const isLarge = htmlLength > 50000 || elementCount > 1000;

      const metaDesc = document.querySelector('meta[name="description"]')?.getAttribute('content') || '';
      const h1s = Array.from(document.querySelectorAll('h1')).map(h => h.innerText.trim()).filter(Boolean);

      return {
        url: window.location.href,
        title: document.title,
        stats: { htmlLength, elementCount, bodyTextLength },
        meta: { description: metaDesc, h1Count: h1s.length, h1Example: h1s[0] || null },
        recommendation: isLarge ? 'download' : 'direct',
        reason: isLarge
          ? `Page is large (${Math.round(htmlLength/1024)}KB, ${elementCount} elements). Use output: "file".`
          : 'Page is small. Safe to read directly.'
      };
    }
  });

  return {
    type: 'browser:response',
    requestId: request.requestId,
    success: true,
    data: results[0]?.result
  };
}

// Handle debugger events
chrome.debugger.onEvent.addListener((source, method, params) => {
  const tabId = source.tabId;
  if (!tabId || !attachedTabs.has(tabId)) return;

  let entry: ConsoleLogEntry | null = null;

  if (method === 'Log.entryAdded') {
    const logEntry = (params as { entry: { level: string; text: string; url?: string; lineNumber?: number } }).entry;
    entry = {
      level: logEntry.level as ConsoleLogEntry['level'],
      text: logEntry.text,
      timestamp: Date.now(),
      url: logEntry.url,
      lineNumber: logEntry.lineNumber
    };
  }

  if (method === 'Runtime.consoleAPICalled') {
    const consoleEvent = params as { type: string; args?: Array<{ type: string; value?: unknown; description?: string }> };
    const levelMap: Record<string, ConsoleLogEntry['level']> = {
      'log': 'log', 'info': 'info', 'warn': 'warning', 'warning': 'warning',
      'error': 'error', 'debug': 'debug'
    };

    const text = consoleEvent.args?.map((arg) => {
      if (arg.type === 'string') return String(arg.value);
      if (arg.type === 'number' || arg.type === 'boolean') return String(arg.value);
      if (arg.type === 'undefined') return 'undefined';
      return arg.description || `[${arg.type}]`;
    }).join(' ') || '';

    entry = {
      level: levelMap[consoleEvent.type] || 'log',
      text,
      timestamp: Date.now()
    };
  }

  if (method === 'Runtime.exceptionThrown') {
    const exception = (params as { exceptionDetails: { text?: string; exception?: { description?: string }; url?: string; lineNumber?: number } }).exceptionDetails;
    entry = {
      level: 'error',
      text: exception.text || exception.exception?.description || 'Unknown error',
      timestamp: Date.now(),
      url: exception.url,
      lineNumber: exception.lineNumber
    };
  }

  if (entry) {
    const logs = consoleLogs.get(tabId) || [];
    logs.push(entry);
    if (logs.length > MAX_LOGS_PER_TAB) {
      logs.splice(0, logs.length - MAX_LOGS_PER_TAB);
    }
    consoleLogs.set(tabId, logs);
  }
});

// Clean up on debugger detach
chrome.debugger.onDetach.addListener((source) => {
  if (source.tabId) {
    attachedTabs.delete(source.tabId);
  }
});

// Clean up on tab close
chrome.tabs.onRemoved.addListener((tabId) => {
  attachedTabs.delete(tabId);
  consoleLogs.delete(tabId);
});

// Listen for messages from side panel
chrome.runtime.onMessage.addListener((message: ExtensionMessage, _sender, sendResponse) => {
  if (message.type === 'ping') {
    sendResponse({ type: 'pong', connectionStatus });
    setTimeout(() => broadcastToExtension({ type: 'connection:status', status: connectionStatus }), 100);
    return true;
  }

  if (message.type === 'terminal:input' || message.type === 'terminal:resize') {
    sendToNativeHost(message as NativeMessage);
    sendResponse({ success: true });
    return true;
  }

  if (message.type === 'connection:status' && (message as { action?: string }).action === 'reconnect') {
    port = null;
    connectToNativeHost();
    sendResponse({ success: true });
    return true;
  }

  return false;
});

// Handle extension icon click
chrome.action.onClicked.addListener((tab) => {
  if (tab.id) {
    chrome.sidePanel.open({ tabId: tab.id });
  }
});

// Initialize connection on service worker start
connectToNativeHost();

console.log('[Background] Service worker initialized');
