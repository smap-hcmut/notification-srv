# Frontend Integration Guide: HttpOnly Cookie WebSocket Authentication

Complete guide for frontend developers to integrate WebSocket connections with HttpOnly cookie authentication.

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Detailed Integration](#detailed-integration)
5. [Framework-Specific Examples](#framework-specific-examples)
6. [Troubleshooting](#troubleshooting)
7. [Best Practices](#best-practices)
8. [Migration from Query Parameters](#migration-from-query-parameters)

---

## Overview

The WebSocket service uses **HttpOnly cookie authentication** for secure, automatic credential handling. This eliminates the need to manually pass JWT tokens in URLs.

### Key Benefits

✅ **Automatic Authentication**: Browser handles cookies automatically  
✅ **Enhanced Security**: Tokens never exposed in URLs or JavaScript  
✅ **Simplified Code**: No manual token management required  
✅ **XSS Protection**: HttpOnly flag prevents script access  
✅ **CSRF Protection**: SameSite policy provides additional security  

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│ 1. User logs in via Identity service                       │
│    POST /identity/authentication/login                      │
│    → Server sets HttpOnly cookie: smap_auth_token          │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Frontend connects to WebSocket                          │
│    new WebSocket('ws://api.smap.com/ws')                    │
│    → Browser automatically sends cookie                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. WebSocket service validates cookie                      │
│    → Extracts JWT from cookie                               │
│    → Validates token                                         │
│    → Establishes WebSocket connection                       │
└─────────────────────────────────────────────────────────────┘
```

---

## Prerequisites

### 1. User Must Be Authenticated

Before connecting to WebSocket, ensure the user is logged in via the Identity service:

```javascript
// Login first to set the cookie
const response = await fetch('https://api.smap.com/identity/authentication/login', {
  method: 'POST',
  credentials: 'include', // CRITICAL: Include credentials
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    email: 'user@example.com',
    password: 'password123',
  }),
});

if (response.ok) {
  // Cookie is now set automatically
  // Ready to connect to WebSocket
}
```

### 2. Frontend Must Be on Allowed Origin

Your frontend must be served from one of these origins:
- `http://localhost:3000` (development)
- `http://127.0.0.1:3000` (development)
- `https://smap.tantai.dev` (production)
- `https://smap-api.tantai.dev` (production)

### 3. HTTPS in Production

In production, cookies require HTTPS:
- Development: `ws://localhost:8081/ws` (HTTP/WS)
- Production: `wss://api.smap.com/ws` (HTTPS/WSS)

---

## Quick Start

### Minimal Example

```javascript
// 1. Ensure user is logged in (cookie is set)
// 2. Connect to WebSocket
const ws = new WebSocket('ws://localhost:8081/ws');

ws.onopen = () => {
  console.log('✅ Connected!');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('📨 Received:', message);
};

ws.onerror = (error) => {
  console.error('❌ Error:', error);
};

ws.onclose = () => {
  console.log('🔌 Connection closed');
};
```

**That's it!** No token management, no URL parameters, no manual cookie handling.

---

## Detailed Integration

### Complete Authentication Flow

```javascript
class WebSocketClient {
  constructor(url) {
    this.url = url;
    this.ws = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
  }

  async connect() {
    try {
      // Create WebSocket connection
      this.ws = new WebSocket(this.url);

      // Connection opened
      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0; // Reset on successful connection
        this.onConnected();
      };

      // Message received
      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          this.handleMessage(message);
        } catch (error) {
          console.error('Failed to parse message:', error);
        }
      };

      // Connection error
      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        this.onError(error);
      };

      // Connection closed
      this.ws.onclose = (event) => {
        console.log('WebSocket closed:', event.code, event.reason);
        this.onDisconnected(event);
        
        // Attempt reconnection
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnect();
        }
      };
    } catch (error) {
      console.error('Failed to create WebSocket:', error);
    }
  }

  reconnect() {
    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
    
    setTimeout(() => {
      this.connect();
    }, delay);
  }

  send(data) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    } else {
      console.warn('WebSocket not connected');
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  // Override these methods in your implementation
  onConnected() {
    console.log('Override onConnected()');
  }

  handleMessage(message) {
    console.log('Override handleMessage():', message);
  }

  onError(error) {
    console.log('Override onError():', error);
  }

  onDisconnected(event) {
    console.log('Override onDisconnected():', event);
  }
}

// Usage
const wsClient = new WebSocketClient('ws://localhost:8081/ws');

wsClient.onConnected = () => {
  console.log('✅ Connected to WebSocket');
};

wsClient.handleMessage = (message) => {
  console.log('📨 Message:', message);
  // Handle different message types
  switch (message.type) {
    case 'notification':
      showNotification(message.payload);
      break;
    case 'update':
      updateUI(message.payload);
      break;
    default:
      console.log('Unknown message type:', message.type);
  }
};

wsClient.onError = (error) => {
  console.error('❌ WebSocket error:', error);
  // Show error to user
};

wsClient.onDisconnected = (event) => {
  console.log('🔌 Disconnected:', event.code);
  // Update UI to show disconnected state
};

// Connect
wsClient.connect();
```

---

## Framework-Specific Examples

### React

#### Using Hooks

```javascript
import { useEffect, useState, useRef } from 'react';

function useWebSocket(url) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState(null);
  const ws = useRef(null);

  useEffect(() => {
    // Create WebSocket connection
    ws.current = new WebSocket(url);

    ws.current.onopen = () => {
      console.log('WebSocket connected');
      setIsConnected(true);
    };

    ws.current.onmessage = (event) => {
      const message = JSON.parse(event.data);
      setLastMessage(message);
    };

    ws.current.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.current.onclose = () => {
      console.log('WebSocket disconnected');
      setIsConnected(false);
    };

    // Cleanup on unmount
    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [url]);

  const sendMessage = (data) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(data));
    }
  };

  return { isConnected, lastMessage, sendMessage };
}

// Usage in component
function NotificationComponent() {
  const { isConnected, lastMessage, sendMessage } = useWebSocket('ws://localhost:8081/ws');

  useEffect(() => {
    if (lastMessage) {
      console.log('New message:', lastMessage);
      // Handle message
    }
  }, [lastMessage]);

  return (
    <div>
      <div>Status: {isConnected ? '🟢 Connected' : '🔴 Disconnected'}</div>
      {lastMessage && (
        <div>Last message: {JSON.stringify(lastMessage)}</div>
      )}
    </div>
  );
}
```

#### Using Context

```javascript
import React, { createContext, useContext, useEffect, useState } from 'react';

const WebSocketContext = createContext(null);

export function WebSocketProvider({ children, url }) {
  const [ws, setWs] = useState(null);
  const [isConnected, setIsConnected] = useState(false);
  const [messages, setMessages] = useState([]);

  useEffect(() => {
    const websocket = new WebSocket(url);

    websocket.onopen = () => {
      console.log('WebSocket connected');
      setIsConnected(true);
    };

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data);
      setMessages((prev) => [...prev, message]);
    };

    websocket.onclose = () => {
      console.log('WebSocket disconnected');
      setIsConnected(false);
    };

    setWs(websocket);

    return () => {
      websocket.close();
    };
  }, [url]);

  const sendMessage = (data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(data));
    }
  };

  return (
    <WebSocketContext.Provider value={{ ws, isConnected, messages, sendMessage }}>
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocketContext() {
  return useContext(WebSocketContext);
}

// Usage
function App() {
  return (
    <WebSocketProvider url="ws://localhost:8081/ws">
      <NotificationList />
    </WebSocketProvider>
  );
}

function NotificationList() {
  const { isConnected, messages } = useWebSocketContext();

  return (
    <div>
      <div>Status: {isConnected ? 'Connected' : 'Disconnected'}</div>
      <ul>
        {messages.map((msg, index) => (
          <li key={index}>{JSON.stringify(msg)}</li>
        ))}
      </ul>
    </div>
  );
}
```

### Vue.js

```javascript
// composables/useWebSocket.js
import { ref, onMounted, onUnmounted } from 'vue';

export function useWebSocket(url) {
  const isConnected = ref(false);
  const lastMessage = ref(null);
  let ws = null;

  const connect = () => {
    ws = new WebSocket(url);

    ws.onopen = () => {
      console.log('WebSocket connected');
      isConnected.value = true;
    };

    ws.onmessage = (event) => {
      lastMessage.value = JSON.parse(event.data);
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
      isConnected.value = false;
    };
  };

  const sendMessage = (data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(data));
    }
  };

  const disconnect = () => {
    if (ws) {
      ws.close();
    }
  };

  onMounted(() => {
    connect();
  });

  onUnmounted(() => {
    disconnect();
  });

  return {
    isConnected,
    lastMessage,
    sendMessage,
  };
}

// Usage in component
<template>
  <div>
    <div>Status: {{ isConnected ? 'Connected' : 'Disconnected' }}</div>
    <div v-if="lastMessage">
      Last message: {{ lastMessage }}
    </div>
  </div>
</template>

<script setup>
import { useWebSocket } from '@/composables/useWebSocket';

const { isConnected, lastMessage, sendMessage } = useWebSocket('ws://localhost:8081/ws');
</script>
```

### Angular

```typescript
// websocket.service.ts
import { Injectable } from '@angular/core';
import { Observable, Subject } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class WebSocketService {
  private ws: WebSocket | null = null;
  private messagesSubject = new Subject<any>();
  private connectionSubject = new Subject<boolean>();

  public messages$ = this.messagesSubject.asObservable();
  public connection$ = this.connectionSubject.asObservable();

  connect(url: string): void {
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.connectionSubject.next(true);
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.messagesSubject.next(message);
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.connectionSubject.next(false);
    };
  }

  sendMessage(data: any): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

// Usage in component
import { Component, OnInit, OnDestroy } from '@angular/core';
import { WebSocketService } from './websocket.service';

@Component({
  selector: 'app-notifications',
  template: `
    <div>
      <div>Status: {{ isConnected ? 'Connected' : 'Disconnected' }}</div>
      <div *ngFor="let message of messages">
        {{ message | json }}
      </div>
    </div>
  `
})
export class NotificationsComponent implements OnInit, OnDestroy {
  isConnected = false;
  messages: any[] = [];

  constructor(private wsService: WebSocketService) {}

  ngOnInit(): void {
    this.wsService.connect('ws://localhost:8081/ws');

    this.wsService.connection$.subscribe(connected => {
      this.isConnected = connected;
    });

    this.wsService.messages$.subscribe(message => {
      this.messages.push(message);
    });
  }

  ngOnDestroy(): void {
    this.wsService.disconnect();
  }
}
```

---

## Troubleshooting

### Issue 1: Connection Rejected - "missing token parameter"

**Cause**: User not logged in or cookie not set

**Solution**:
```javascript
// Ensure user is logged in first
async function ensureAuthenticated() {
  const response = await fetch('https://api.smap.com/identity/authentication/me', {
    credentials: 'include',
  });

  if (!response.ok) {
    // User not logged in, redirect to login
    window.location.href = '/login';
    return false;
  }

  return true;
}

// Use before connecting to WebSocket
if (await ensureAuthenticated()) {
  const ws = new WebSocket('ws://localhost:8081/ws');
}
```

### Issue 2: CORS Error

**Cause**: Frontend not served from allowed origin

**Solution**:
- Ensure your frontend is on `localhost:3000` (development) or `smap.tantai.dev` (production)
- Check browser console for exact CORS error
- Verify WebSocket URL matches your environment

### Issue 3: Cookie Not Sent

**Cause**: Missing `credentials: 'include'` in login request

**Solution**:
```javascript
// WRONG - Cookie not set
fetch('/identity/authentication/login', {
  method: 'POST',
  body: JSON.stringify({ email, password }),
});

// CORRECT - Cookie set automatically
fetch('/identity/authentication/login', {
  method: 'POST',
  credentials: 'include', // ← REQUIRED
  body: JSON.stringify({ email, password }),
});
```

### Issue 4: Connection Works Locally But Not in Production

**Cause**: Using `ws://` instead of `wss://` in production

**Solution**:
```javascript
// Use environment-based URL
const WS_URL = process.env.NODE_ENV === 'production'
  ? 'wss://api.smap.com/ws'
  : 'ws://localhost:8081/ws';

const ws = new WebSocket(WS_URL);
```

### Issue 5: Multiple Tabs/Windows

**Behavior**: Each tab creates a separate WebSocket connection

**This is normal and expected**. The server supports multiple connections per user.

```javascript
// Each tab will have its own connection
// Server tracks all connections for the same user
```

---

## Best Practices

### 1. Environment Configuration

```javascript
// config.js
const config = {
  development: {
    wsUrl: 'ws://localhost:8081/ws',
    apiUrl: 'http://localhost:8080',
  },
  production: {
    wsUrl: 'wss://api.smap.com/ws',
    apiUrl: 'https://api.smap.com',
  },
};

export const WS_URL = config[process.env.NODE_ENV].wsUrl;
export const API_URL = config[process.env.NODE_ENV].apiUrl;
```

### 2. Reconnection Strategy

```javascript
class ReconnectingWebSocket {
  constructor(url, options = {}) {
    this.url = url;
    this.maxReconnectAttempts = options.maxReconnectAttempts || 5;
    this.reconnectInterval = options.reconnectInterval || 1000;
    this.reconnectDecay = options.reconnectDecay || 1.5;
    this.reconnectAttempts = 0;
    this.ws = null;
  }

  connect() {
    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      console.log('Connected');
      this.reconnectAttempts = 0;
    };

    this.ws.onclose = () => {
      console.log('Disconnected');
      this.reconnect();
    };
  }

  reconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectInterval * Math.pow(this.reconnectDecay, this.reconnectAttempts);

    console.log(`Reconnecting in ${delay}ms...`);
    setTimeout(() => this.connect(), delay);
  }
}
```

### 3. Message Type Handling

```javascript
function handleWebSocketMessage(message) {
  switch (message.type) {
    case 'notification':
      showNotification(message.payload);
      break;
    case 'update':
      updateData(message.payload);
      break;
    case 'error':
      handleError(message.payload);
      break;
    default:
      console.warn('Unknown message type:', message.type);
  }
}
```

### 4. Connection State Management

```javascript
const ConnectionState = {
  CONNECTING: 0,
  OPEN: 1,
  CLOSING: 2,
  CLOSED: 3,
};

function getConnectionStatus(ws) {
  switch (ws.readyState) {
    case ConnectionState.CONNECTING:
      return 'Connecting...';
    case ConnectionState.OPEN:
      return 'Connected';
    case ConnectionState.CLOSING:
      return 'Closing...';
    case ConnectionState.CLOSED:
      return 'Disconnected';
    default:
      return 'Unknown';
  }
}
```

---

## Migration from Query Parameters

If you're migrating from the old query parameter authentication:

### Before (Deprecated)

```javascript
// OLD METHOD - Don't use
const token = localStorage.getItem('jwt_token');
const ws = new WebSocket(`ws://localhost:8081/ws?token=${token}`);
```

### After (Current)

```javascript
// NEW METHOD - Use this
const ws = new WebSocket('ws://localhost:8081/ws');
// Cookie sent automatically!
```

### Migration Checklist

- [ ] Remove token from WebSocket URL
- [ ] Remove token storage in localStorage/sessionStorage
- [ ] Ensure login uses `credentials: 'include'`
- [ ] Test WebSocket connection without query parameter
- [ ] Remove token management code
- [ ] Update documentation

---

## Complete Example Application

```javascript
// app.js - Complete WebSocket integration example
class NotificationApp {
  constructor() {
    this.ws = null;
    this.isConnected = false;
    this.notifications = [];
  }

  async init() {
    // Check if user is authenticated
    if (!(await this.checkAuth())) {
      window.location.href = '/login';
      return;
    }

    // Connect to WebSocket
    this.connectWebSocket();

    // Setup UI event listeners
    this.setupUI();
  }

  async checkAuth() {
    try {
      const response = await fetch('https://api.smap.com/identity/authentication/me', {
        credentials: 'include',
      });
      return response.ok;
    } catch (error) {
      console.error('Auth check failed:', error);
      return false;
    }
  }

  connectWebSocket() {
    const wsUrl = process.env.NODE_ENV === 'production'
      ? 'wss://api.smap.com/ws'
      : 'ws://localhost:8081/ws';

    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log('✅ WebSocket connected');
      this.isConnected = true;
      this.updateConnectionStatus('Connected');
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };

    this.ws.onerror = (error) => {
      console.error('❌ WebSocket error:', error);
      this.updateConnectionStatus('Error');
    };

    this.ws.onclose = () => {
      console.log('🔌 WebSocket disconnected');
      this.isConnected = false;
      this.updateConnectionStatus('Disconnected');
      // Attempt reconnection after 5 seconds
      setTimeout(() => this.connectWebSocket(), 5000);
    };
  }

  handleMessage(message) {
    console.log('📨 Received message:', message);

    switch (message.type) {
      case 'notification':
        this.addNotification(message.payload);
        break;
      case 'update':
        this.handleUpdate(message.payload);
        break;
      default:
        console.warn('Unknown message type:', message.type);
    }
  }

  addNotification(notification) {
    this.notifications.unshift(notification);
    this.renderNotifications();
    this.showToast(notification.title);
  }

  handleUpdate(data) {
    console.log('Update received:', data);
    // Handle data updates
  }

  updateConnectionStatus(status) {
    const statusEl = document.getElementById('connection-status');
    if (statusEl) {
      statusEl.textContent = status;
      statusEl.className = status.toLowerCase();
    }
  }

  renderNotifications() {
    const container = document.getElementById('notifications');
    if (!container) return;

    container.innerHTML = this.notifications
      .map(
        (notif) => `
        <div class="notification">
          <h3>${notif.title}</h3>
          <p>${notif.message}</p>
          <small>${new Date(notif.timestamp).toLocaleString()}</small>
        </div>
      `
      )
      .join('');
  }

  showToast(message) {
    // Show toast notification
    console.log('Toast:', message);
  }

  setupUI() {
    // Setup any UI event listeners
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

// Initialize app
const app = new NotificationApp();
app.init();

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
  app.disconnect();
});
```

---

## Summary

### Key Takeaways

1. ✅ **No manual token handling** - Browser manages cookies automatically
2. ✅ **Login first** - Ensure user is authenticated before connecting
3. ✅ **Use `credentials: 'include'`** - Required for cookie transmission
4. ✅ **Environment-aware URLs** - Use `ws://` for dev, `wss://` for production
5. ✅ **Handle reconnection** - Implement reconnection logic for reliability

### Quick Reference

```javascript
// Login
await fetch('/identity/authentication/login', {
  method: 'POST',
  credentials: 'include', // ← Required
  body: JSON.stringify({ email, password }),
});

// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8081/ws');
// Cookie sent automatically!

// Handle messages
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log(message);
};
```

### Need Help?

- **Documentation**: See [README.md](../README.md)
- **Testing Guide**: See [manual_testing_guide.md](manual_testing_guide.md)
- **Deployment**: See [deployment_guide.md](deployment_guide.md)

---

**Last Updated**: 2025-11-28  
**Version**: 1.0 - HttpOnly Cookie Authentication
