# WebSocket Notification Service - Overview

## Introduction

The WebSocket Notification Service is a real-time communication hub built with Go that maintains persistent WebSocket connections with clients and delivers notifications from Redis Pub/Sub channels. This service acts as a bridge between backend services and connected users, enabling instant message delivery.

## Purpose

This service solves the problem of real-time notification delivery in distributed microservice architectures. Backend services publish messages to Redis channels, and this service ensures those messages reach connected users immediately through WebSocket connections.

## Key Features

### Core Functionality
- WebSocket server with JWT-based authentication
- Redis Pub/Sub integration for message routing
- Multi-connection support per user (multiple browser tabs/devices)
- Real-time message delivery with sub-millisecond latency
- Pattern-based subscription to user notification channels

### Reliability
- Automatic reconnection to Redis with exponential backoff
- Connection health monitoring via Ping/Pong keep-alive
- Graceful shutdown handling with connection cleanup
- Message delivery tracking and failure handling
- Thread-safe concurrent connection management

### Scalability
- Horizontal scaling ready (stateless design)
- Configurable connection limits (default 10,000)
- Connection pooling for Redis
- Buffered channels to prevent blocking
- Optimized goroutine usage for concurrent handling

### Operations
- Health check endpoint for monitoring
- Metrics endpoint for observability
- Structured logging with configurable levels
- Docker support with multi-stage builds
- Environment-based configuration

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Other Backend Services                   │
│              (API, Workers, Scheduled Jobs)                 │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ PUBLISH to channels:
                     │ user_noti:user123
                     │ user_noti:user456
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                        Redis Pub/Sub                        │
│              (Message Broker & Channel Manager)             │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ Pattern Subscribe: user_noti:*
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              WebSocket Notification Service                 │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Redis Subscriber                       │    │
│  │  - Listens to user_noti:* pattern                   │    │
│  │  - Deserializes JSON messages                       │    │
│  │  - Routes to Hub by user ID                         │    │
│  └──────────────────┬──────────────────────────────────┘    │ 
│                     │                                       │
│                     ▼                                       │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                  Hub (Connection Registry)          │    │
│  │  - Manages user → connections mapping               │    │
│  │  - Broadcasts messages to user connections          │    │
│  │  - Handles registration/unregistration              │    │
│  │  - Tracks metrics and statistics                    │    │
│  └──────────────────┬──────────────────────────────────┘    │
│                     │                                       │
│                     ▼                                       │
│  ┌─────────────────────────────────────────────────────┐    │
│  │               Individual Connections                │    │
│  │  - Read Pump: Handles Pong & disconnection          │    │
│  │  - Write Pump: Sends messages & Ping frames         │    │
│  │  - Buffered send channels (256 messages)            │    │
│  └─────────────────────────────────────────────────────┘    │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ WebSocket Protocol
                     │ ws://host:8081/ws?token=JWT
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                    Web Clients                              │
│              (Browsers, Mobile Apps)                        │
└─────────────────────────────────────────────────────────────┘
```

## Technology Stack

### Core Technologies
- **Language**: Go 1.25+
- **WebSocket**: gorilla/websocket
- **Redis Client**: go-redis/v9
- **HTTP Framework**: Gin
- **Logging**: Uber Zap

### Development & Deployment
- **Configuration**: Environment variables with caarlos0/env
- **Container**: Docker with distroless base image
- **Authentication**: JWT (golang-jwt/jwt)

## Use Cases

### Notification Delivery
- Order status updates
- Payment confirmations
- System alerts and warnings
- User mentions and messages
- Real-time status changes

### Real-Time Updates
- Live data synchronization
- Dashboard metric updates
- Task completion notifications
- Collaborative editing events

### Multi-Device Support
- Synchronized notifications across devices
- Multiple browser tabs receiving same messages
- Mobile and desktop simultaneous connections

## System Requirements

### Runtime Requirements
- Go 1.25 or higher
- Redis 7.0 or higher
- Network access to Redis server
- JWT secret key for authentication

### Resource Requirements
- **Memory**: ~50MB base + ~5KB per connection
- **CPU**: <5% idle, ~20% at max load (10,000 connections)
- **Network**: Minimal bandwidth (JSON messages only)
- **Storage**: None (stateless service)

### Recommended Configuration
- **Development**: 1 CPU core, 256MB RAM
- **Production**: 2+ CPU cores, 1GB+ RAM
- **Max Connections**: 10,000 per instance (configurable)

## Performance Characteristics

### Latency
- WebSocket message delivery: <1ms (local network)
- Redis to client: <10ms (typical)
- Authentication: <5ms per connection

### Throughput
- Message handling: >100,000 messages/second
- Concurrent connections: 10,000 per instance
- Connection establishment: ~1,000/second

### Scalability
- Horizontal: Deploy multiple instances behind load balancer
- Vertical: Increase connection limit per instance
- Redis: Single Redis instance can support multiple service instances

## Security Features

### Authentication
- JWT token validation on connection
- Configurable secret key
- Token expiration checking
- User ID extraction from token claims

### Network Security
- TLS support for Redis connections
- CORS configuration for WebSocket
- Non-root container user (UID 65532)
- Distroless container image (no shell)

### Operational Security
- No secrets in source code
- Environment-based configuration
- Structured audit logging
- Connection limit enforcement

## High-Level Workflow

### Client Connection Flow
1. Client obtains JWT token from authentication service
2. Client initiates WebSocket connection: `ws://host:8081/ws?token=JWT`
3. Service validates JWT and extracts user ID
4. Service creates Connection object and registers with Hub
5. Service starts read and write pumps for the connection
6. Client receives connection confirmation

### Message Delivery Flow
1. Backend service publishes message to Redis: `PUBLISH user_noti:user123 {...}`
2. Redis Subscriber receives message via pattern subscription
3. Subscriber parses message and extracts user ID from channel name
4. Subscriber sends message to Hub for user ID
5. Hub looks up all connections for that user
6. Hub sends message to each connection's send channel
7. Write pump sends message through WebSocket to client
8. Client receives and processes message

### Disconnection Flow
1. Client closes connection or network fails
2. Read pump detects error and triggers cleanup
3. Connection unregisters from Hub
4. Hub removes connection from user's connection list
5. If last connection, Hub removes user entry
6. Resources are cleaned up (channels closed)

## Project Structure

```
websocket/
├── cmd/
│   └── server/
│       ├── main.go                 # Entry point with lifecycle management
│       └── Dockerfile              # Multi-stage optimized build
├── config/
│   └── config.go                   # Configuration loading from environment
├── internal/
│   ├── websocket/
│   │   ├── hub.go                 # Connection registry and message router
│   │   ├── connection.go          # Individual WebSocket connection handler
│   │   ├── handler.go             # HTTP handler for WebSocket upgrade
│   │   ├── message.go             # Message types and serialization
│   │   └── errors.go              # WebSocket-specific errors
│   ├── redis/
│   │   └── subscriber.go          # Redis Pub/Sub listener and router
│   └── server/
│       ├── server.go              # HTTP server setup and lifecycle
│       ├── health.go              # Health check endpoint
│       └── metrics.go             # Metrics endpoint
├── pkg/
│   ├── redis/
│   │   ├── client.go              # Redis client wrapper with pooling
│   │   └── types.go               # Redis configuration types
│   ├── jwt/
│   │   ├── validator.go           # JWT validation and parsing
│   │   └── types.go               # JWT configuration and claims
│   ├── log/
│   │   ├── new.go                 # Logger factory
│   │   └── zap.go                 # Zap logger implementation
│   ├── discord/                   # Optional Discord webhook integration
│   ├── errors/                    # Shared error types
│   ├── locale/                    # Localization support
│   └── response/                  # Standard response formatting
├── tests/
│   ├── client_example.go          # Example WebSocket client for testing
│   └── generate_token.go          # JWT token generator for testing
├── scripts/
│   ├── build.sh                   # Build automation script
│   └── quick_test.sh              # Quick test script
├── document/                      # Documentation (you are here)
├── go.mod                         # Go module dependencies
├── go.sum                         # Dependency checksums
├── Makefile                       # Build and development commands
├── template.env                   # Environment variable template
└── README.md                      # Quick start guide
```

## Configuration Management

All configuration is done through environment variables. Key categories:

### Server Configuration
- Host and port binding
- Operation mode (debug/release)

### Redis Configuration
- Connection details (host, port, password)
- TLS settings
- Connection pool parameters

### WebSocket Configuration
- Ping/Pong intervals
- Read/write timeouts
- Message size limits
- Connection limits

### Authentication Configuration
- JWT secret key
- Token validation rules

### Logging Configuration
- Log level (debug, info, warn, error)
- Log encoding (json, console)
- Log mode (development, production)

See `template.env` for complete list of configuration options.

## Operational Characteristics

### Startup
- Loads configuration from environment
- Initializes logger with structured output
- Connects to Redis and validates connection
- Initializes JWT validator
- Starts Hub goroutine
- Starts Redis Subscriber
- Starts HTTP server
- Reports "ready" status

### Runtime
- Accepts WebSocket connections continuously
- Maintains connection health via Ping/Pong
- Routes messages from Redis to connected users
- Tracks metrics for monitoring
- Handles connection failures gracefully
- Logs important events and errors

### Shutdown
- Receives interrupt signal (SIGINT/SIGTERM)
- Stops accepting new connections
- Shuts down Redis Subscriber
- Closes all WebSocket connections gracefully
- Shuts down HTTP server
- Waits up to 30 seconds for cleanup
- Exits cleanly

## Monitoring & Observability

### Health Check Endpoint
```
GET /health
```
Returns system health including Redis connectivity, connection counts, and uptime.

### Metrics Endpoint
```
GET /metrics
```
Returns operational metrics including:
- Active connections
- Unique users connected
- Messages received from Redis
- Messages sent to clients
- Failed message deliveries

### Logs
Structured JSON logs with levels:
- **INFO**: Normal operations (connections, disconnections)
- **WARN**: Recoverable issues (full buffers, reconnections)
- **ERROR**: Serious errors (authentication failures, Redis errors)
- **DEBUG**: Detailed debugging (message routing, raw data)
