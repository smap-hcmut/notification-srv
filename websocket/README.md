# WebSocket Notification Service

> Real-time notification hub using WebSocket and Redis Pub/Sub

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Redis](https://img.shields.io/badge/Redis-7.0+-DC382D?style=flat&logo=redis)](https://redis.io/)
[![WebSocket](https://img.shields.io/badge/WebSocket-RFC6455-4FC08D?style=flat)](https://datatracker.ietf.org/doc/html/rfc6455)

---

## Overview

**WebSocket Service** is a lightweight, scalable notification hub that maintains persistent WebSocket connections and delivers real-time messages from Redis Pub/Sub to connected clients.

### Key Features

- WebSocket Server with JWT authentication  
- Redis Pub/Sub integration for message routing  
- Multiple connections per user (multiple browser tabs)  
- Auto reconnection to Redis with retry logic  
- Ping/Pong keep-alive (30s interval)  
- Health & Metrics endpoints  
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

## Endpoints

### WebSocket Connection
**GET** `/ws?token=<JWT_TOKEN>`

```javascript
const ws = new WebSocket(`ws://localhost:8081/ws?token=${token}`);
ws.onmessage = (event) => console.log(JSON.parse(event.data));
```

### Health Check
**GET** `/health`

### Metrics
**GET** `/metrics`

---

## Testing

### With Example Client
```bash
go run tests/client_example.go YOUR_JWT_TOKEN
```

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
WS_PORT=8081
REDIS_HOST=localhost
REDIS_PASSWORD=21042004
JWT_SECRET_KEY=your-secret
WS_MAX_CONNECTIONS=10000
```

---

## Documentation

- **[Technical Requirements](document/proposal.md)** - Vietnamese specification
- **[Integration Guide](document/integration.md)** - How to integrate

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

**Built for SMAP Graduation Project**  
*Last updated: 2025-11-21*
