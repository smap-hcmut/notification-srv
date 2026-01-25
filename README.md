# WebSocket Notification Service

> Real-time notification hub using WebSocket and Redis Pub/Sub

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Redis](https://img.shields.io/badge/Redis-7.0+-DC382D?style=flat&logo=redis)](https://redis.io/)
[![WebSocket](https://img.shields.io/badge/WebSocket-RFC6455-4FC08D?style=flat)](https://datatracker.ietf.org/doc/html/rfc6455)

---

## Overview

**WebSocket Service** is a lightweight, scalable notification hub that maintains persistent WebSocket connections and delivers real-time messages from Redis Pub/Sub to connected clients.

### Key Features

- **HttpOnly Cookie Authentication** - Secure JWT authentication via cookies
- WebSocket Server with automatic credential handling
- Redis Pub/Sub integration for message routing
- Multiple connections per user (multiple browser tabs)
- Auto reconnection to Redis with retry logic
- Ping/Pong keep-alive (30s interval)
- Health check endpoint
- Graceful shutdown handling
- Docker support with optimized build
- Horizontal scaling ready

---

## Architecture

```
┌─────────────────┐         ┌──────────────┐         ┌─────────────────┐
│ Other Services  │ ──────► │    Redis     │ ──────► │ WebSocket Hub   │
│  (Publishers)   │ PUBLISH │  (Pub/Sub)   │ CONSUME │   (This Service)│
└─────────────────┘         └──────────────┘         └────────┬────────┘
                                                               │ WebSocket
                                                               ▼
                                                      ┌─────────────────┐
                                                      │  Web Clients    │
                                                      └─────────────────┘
```

---

## Quick Start

### Running Locally

```bash
# 1. Copy environment template
cp template.env .env

# 2. Edit configuration
nano .env

# 3. Start Redis
redis-server --requirepass 21042004

# 4. Run service
make run
```

### Using Docker

```bash
make docker-build
make docker-run
```

---

## Authentication

This service uses **HttpOnly cookie authentication** shared with the Identity service for improved security.

### How It Works

1. **Login via Identity Service**: User authenticates and receives `smap_auth_token` cookie
2. **Automatic Cookie Transmission**: Browser automatically sends cookie with WebSocket connection
3. **Secure Token Handling**: JWT never exposed in URLs or browser history

### Frontend Integration

#### Recommended: Cookie-Based Authentication (Secure)

```javascript
// No token needed in URL - cookie sent automatically!
const ws = new WebSocket("ws://localhost:8081/ws");

ws.onopen = () => {
  console.log("Connected with cookie authentication!");
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log("Received:", message);
};

ws.onerror = (error) => {
  console.error("WebSocket error:", error);
};
```

**Requirements**:

- User must be logged in via Identity service (cookie automatically set)
- Frontend must be served from allowed origin (localhost:3000, smap.tantai.dev)
- Browser automatically includes cookies with WebSocket connections

#### Legacy: Query Parameter Authentication (Deprecated)

> ⚠️ **Deprecated**: This method is maintained for backward compatibility but will be removed in a future version.

```javascript
// Old method - token exposed in URL
const token = "your-jwt-token";
const ws = new WebSocket(`ws://localhost:8081/ws?token=${token}`);
```

**Security Issues**:

- Token exposed in URL and logs
- Token stored in browser history
- Vulnerable to referrer header leakage

### Migration Guide

**For Frontend Developers**:

1. **Remove token from WebSocket URL**:

   ```javascript
   // Before
   const ws = new WebSocket(`ws://api.smap.com/ws?token=${token}`);

   // After
   const ws = new WebSocket("ws://api.smap.com/ws");
   ```

2. **Ensure user is authenticated**: Login via Identity service first

   ```javascript
   // Login to get cookie
   await fetch("https://api.smap.com/identity/authentication/login", {
     method: "POST",
     credentials: "include", // Important!
     headers: { "Content-Type": "application/json" },
     body: JSON.stringify({ email, password }),
   });

   // Cookie is now set, connect to WebSocket
   const ws = new WebSocket("ws://api.smap.com/ws");
   ```

3. **Test the connection**: Verify cookie authentication works before removing query parameter

### Troubleshooting

#### Connection Rejected: "missing token parameter"

- **Cause**: Not logged in or cookie not set
- **Solution**: Login via Identity service first

#### Connection Rejected: "invalid or expired token"

- **Cause**: Cookie expired or invalid JWT
- **Solution**: Re-login to get fresh cookie

#### CORS Errors

- **Cause**: Frontend not served from allowed origin
- **Solution**: Ensure frontend is on localhost:3000 or smap.tantai.dev

---

## Endpoints

### WebSocket Connection

**GET** `/ws`

Authentication via HttpOnly cookie (automatic) or query parameter (deprecated).

### Health Check

**GET** `/health`

---

## Testing

### With Redis CLI

```bash
redis-cli -a 21042004
PUBLISH user_noti:user123 '{"type":"notification","payload":{"title":"Hello"}}'
```

### With Postman

1. Create WebSocket Request
2. URL: `ws://localhost:8081/ws?token=YOUR_TOKEN`
3. Connect

---

## Configuration

Key environment variables (see `template.env` for full list):

```bash
# Environment (controls CORS behavior)
ENV=production  # Values: production | staging | dev

# Server
WS_PORT=8081

# Redis
REDIS_HOST=localhost
REDIS_PASSWORD=21042004

# Authentication (must match Identity service)
JWT_SECRET_KEY=your-secret
COOKIE_NAME=smap_auth_token
COOKIE_DOMAIN=.smap.com
COOKIE_SECURE=true

# WebSocket
WS_MAX_CONNECTIONS=10000
```

**Important**: Cookie configuration must match Identity service for authentication to work.

### Environment Configuration (`ENV`)

The `ENV` variable controls CORS (Cross-Origin Resource Sharing) behavior:

| Value        | CORS Mode                | Allowed Origins                                                                    |
| ------------ | ------------------------ | ---------------------------------------------------------------------------------- |
| `production` | Strict                   | Production domains only (`https://smap.tantai.dev`, `https://smap-api.tantai.dev`) |
| `staging`    | Permissive               | Production domains + localhost + private subnets (VPN access)                      |
| `dev`        | Permissive               | Production domains + localhost + private subnets (VPN access)                      |
| _(empty)_    | Defaults to `production` | Production domains only (fail-safe)                                                |

**Security Notes**:

- ⚠️ **Never** set `ENV=dev` or `ENV=staging` in production deployments
- Production mode blocks localhost and private subnet access for security
- Dev/staging modes allow VPN access from private subnets (172.16.x.x, 192.168.x.x)
- Default behavior is production mode if `ENV` is not set (fail-safe)

**Private Subnets** (allowed in dev/staging only):

- `172.16.21.0/24` - K8s cluster subnet
- `172.16.19.0/24` - Private network 1
- `192.168.1.0/24` - Private network 2

These can be customized by modifying `privateSubnets` in `internal/websocket/handler.go`.

---

## Documentation

See `documents/` directory for detailed documentation including integration guides and technical specifications.

---

## Make Commands

```bash
make run                # Run locally
make build              # Build binary
make docker-build       # Build Docker image
make test               # Run tests
make help               # Show all commands
```

---

## Security

### HttpOnly Cookie Benefits

- **No Token Exposure**: JWT never appears in URLs or logs
- **XSS Protection**: HttpOnly flag prevents JavaScript access
- **HTTPS Only**: Secure flag ensures encrypted transmission
- **CSRF Protection**: SameSite=Lax policy
- **Automatic Handling**: Browser manages cookie lifecycle

### CORS Configuration

CORS behavior is controlled by the `ENV` environment variable (see [Configuration](#environment-configuration-env) above).

**Production Mode** (`ENV=production`):

- Only production domains allowed: `https://smap.tantai.dev`, `https://smap-api.tantai.dev`
- Localhost and private subnets are blocked

**Dev/Staging Mode** (`ENV=dev` or `ENV=staging`):

- Production domains allowed
- Localhost allowed (any port): `http://localhost:*`, `http://127.0.0.1:*`
- Private subnets allowed (VPN access): `172.16.21.0/24`, `172.16.19.0/24`, `192.168.1.0/24`

**Startup Logging**: The service logs the active CORS mode on startup:

```
CORS mode: production (strict origins only)
```

or

```
CORS mode: dev (permissive - allows localhost and private subnets)
```

---

**Built for SMAP Graduation Project**  
_Last updated: 2026-01-25_
