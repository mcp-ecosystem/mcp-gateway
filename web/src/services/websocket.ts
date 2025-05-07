import toast from 'react-hot-toast';
import { v4 as uuidv4 } from 'uuid';

import { ToolCall } from '../types/message';

export interface WebSocketMessage {
  type: 'system' | 'message' | 'stream' | 'tool_call' | 'tool_result';
  content: string;
  sender: 'user' | 'bot';
  timestamp: number;
  id: string;
  tools?: Array<{
    name: string;
    description: string;
    parameters: {
      properties: Record<string, unknown>;
      required: string[];
    };
  }>;
  toolCalls?: ToolCall[];
  toolResult?: {
    toolCallId: string
    name: string;
    result: string;
  };
}

export class WebSocketService {
  private ws: WebSocket | null = null;
  private messageHandlers: Array<(message: WebSocketMessage) => void> = [];
  private streamHandlers: Array<(chunk: string) => void> = [];
  private sessionId: string = uuidv4();

  constructor() {
    this.cleanup();
  }

  clearMessageHandlers() {
    this.messageHandlers = [];
    this.streamHandlers = [];
  }

  cleanup() {
    // Clear existing connection if any
    this.disconnect();
    // Clear message handlers
    this.clearMessageHandlers();
    // Generate new session ID
    this.sessionId = uuidv4();
  }

  switchChat(sessionId: string) {
    // Clear existing connection
    this.disconnect();
    // Don't clear message handlers here
    // this.clearMessageHandlers();
    // Set new session ID
    this.sessionId = sessionId;
    // Don't connect immediately, wait for first message
    return Promise.resolve();
  }

  connect() {
    if (this.ws) {
      return Promise.resolve();
    }

    return new Promise<void>((resolve) => {
      const token = window.localStorage.getItem('token');
      this.ws = new WebSocket(`${import.meta.env.VITE_WS_BASE_URL}/chat?sessionId=${this.sessionId}&token=${token}`);

      this.ws.onopen = () => {
        resolve();
      };

      this.ws.onmessage = (event) => {
        const message = JSON.parse(event.data) as WebSocketMessage;
        if (message.type === 'stream') {
          this.streamHandlers.forEach(handler => handler(message.content));
        } else {
          this.messageHandlers.forEach(handler => handler(message));
        }
      };

      this.ws.onclose = () => {
        toast.error('WebSocket 连接已断开', {
          duration: 3000,
          position: 'bottom-right',
        });
        this.ws = null;
      };

      this.ws.onerror = (error) => {
        toast.error('WebSocket 发生错误' + error, {
          duration: 3000,
          position: 'bottom-right',
        });
      };
    });
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  async sendMessage(content: string, tools?: WebSocketMessage['tools']) {
    // Connect to WebSocket if not already connected
    if (!this.ws) {
      await this.connect();
    }

    const message: WebSocketMessage = {
      type: 'message',
      content,
      sender: 'user',
      timestamp: Date.now(),
      id: uuidv4(),
      tools,
    };

    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  async sendToolResult(name: string, toolCallId: string, result: string) {
    // Connect to WebSocket if not already connected
    if (!this.ws) {
      await this.connect();
    }

    const message: WebSocketMessage = {
      type: 'tool_result',
      content: '',
      sender: 'user',
      timestamp: Date.now(),
      id: uuidv4(),
      toolResult: {
        toolCallId,
        name,
        result
      }
    };

    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  onMessage(handler: (message: WebSocketMessage) => void) {
    this.messageHandlers.push(handler);
    return () => {
      this.messageHandlers = this.messageHandlers.filter(h => h !== handler);
    };
  }

  onStream(handler: (chunk: string) => void) {
    this.streamHandlers.push(handler);
    return () => {
      this.streamHandlers = this.streamHandlers.filter(h => h !== handler);
    };
  }

  getSessionId(): string {
    return this.sessionId;
  }
}

// Create a singleton instance
export const wsService = new WebSocketService();
