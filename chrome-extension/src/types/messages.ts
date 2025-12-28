// Message types for Native Messaging communication

export interface NativeMessage {
  type: string;
  data?: unknown;
  cols?: number;
  rows?: number;
  requestId?: string;
  action?: string;
  params?: Record<string, unknown>;
  success?: boolean;
  error?: string;
}

export interface TerminalInputMessage extends NativeMessage {
  type: 'terminal:input';
  data: string;
}

export interface TerminalOutputMessage extends NativeMessage {
  type: 'terminal:output';
  data: string;
}

export interface TerminalResizeMessage extends NativeMessage {
  type: 'terminal:resize';
  cols: number;
  rows: number;
}

export interface BrowserContextRequest extends NativeMessage {
  type: 'browser:request';
  action: string;
  params?: Record<string, unknown>;
  requestId: string;
}

export interface BrowserContextResponse extends NativeMessage {
  type: 'browser:response';
  requestId: string;
  success: boolean;
  data?: unknown;
  error?: string;
}

export interface ConnectionStatusMessage {
  type: 'connection:status';
  status: 'connected' | 'disconnected' | 'connecting' | 'error';
  message?: string;
}

export type ExtensionMessage =
  | TerminalInputMessage
  | TerminalOutputMessage
  | TerminalResizeMessage
  | BrowserContextRequest
  | BrowserContextResponse
  | ConnectionStatusMessage
  | { type: 'ping' }
  | { type: 'pong'; connectionStatus: string };
