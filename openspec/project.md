# Project Context

## Purpose
The SMAP WebSocket Notification Service is a real-time communication hub that maintains persistent WebSocket connections with clients and delivers notifications from Redis Pub/Sub channels. This service acts as a bridge between backend microservices and connected users, enabling instant message delivery for notifications, status updates, and real-time events.

**Key Goals:**
- Provide low-latency real-time message delivery (<10ms typical)
- Support multiple concurrent connections per user (multiple browser tabs/devices)
- Scale horizontally to handle 10,000+ concurrent connections per instance
- Maintain high availability with automatic Redis reconnection
- Secure authentication via HttpOnly cookies (shared with Identity Service)

## Tech Stack
- **Language**: Go 1.25+
- **WebSocket**: `github.com/gorilla/websocket` v1.5.3
- **HTTP Framework**: `github.com/gin-gonic/gin` v1.11.0
- **Redis Client**: `github.com/redis/go-redis/v9` v9.7.0
- **Logging**: `go.uber.org/zap` v1.27.0 (structured JSON logging)
- **JWT**: `github.com/golang-jwt/jwt/v5` v5.2.1
- **Configuration**: `github.com/caarlos0/env/v9` v9.0.0 (environment variables)
- **Container**: Docker with distroless base image

## Project Conventions

### Code Style
- Follow Go standard formatting (`gofmt`)
- Use `golangci-lint` for code quality (if configured)
- Package naming: lowercase, no underscores (e.g., `websocket`, `redis`)
- File naming: snake_case for multi-word files (e.g., `handler.go`, `connection.go`)
- Error handling: Always check errors, use structured error types from `pkg/errors`
- Logging: Use structured logging with context, include user IDs and connection IDs in logs
- Comments: Export all public functions/types with godoc comments

### Architecture Patterns
- **Hub Pattern**: Central connection registry (`internal/websocket/hub.go`) manages user → connections mapping
- **Pub/Sub Pattern**: Redis Pub/Sub for message routing from backend services
- **Connection Pattern**: Each WebSocket connection has read/write pumps (goroutines)
- **Dependency Injection**: Pass dependencies (logger, validator, config) via constructors
- **Interface Segregation**: Define interfaces for testability (e.g., `RedisNotifier` interface)
- **Stateless Design**: No in-memory state persistence, ready for horizontal scaling
- **Graceful Shutdown**: Handle SIGINT/SIGTERM with connection cleanup

### Testing Strategy
- **Unit Tests**: Target 100% coverage for new functions (especially CORS validation, CIDR checks)
- **Integration Tests**: Test Redis Pub/Sub message flow with test Redis instance
- **Manual Testing**: Use Redis CLI or Postman for WebSocket connection testing
- **Test Files**: Co-locate with source files (`*_test.go`)
- **Test Naming**: `TestFunctionName_Scenario` format (e.g., `TestIsPrivateOrigin_K8sSubnet`)
- **Test Data**: Use table-driven tests for multiple scenarios
- **Mocking**: Use interfaces to enable dependency injection in tests

### Git Workflow
- **Branching**: Feature branches from `master` branch
- **Commits**: Conventional commits preferred (feat:, fix:, docs:, test:)
- **PR Process**: Code review required, all tests must pass
- **OpenSpec**: Use OpenSpec for change proposals (see `openspec/AGENTS.md`)
- **Archiving**: Archive completed changes to `openspec/changes/archive/YYYY-MM-DD-<name>/`

## Domain Context

### WebSocket Service Domain
- **Connections**: Each user can have multiple WebSocket connections (multiple tabs/devices)
- **Authentication**: HttpOnly cookie authentication (primary), query parameter (deprecated fallback)
- **Message Routing**: Messages published to Redis channels (`user_noti:<userID>`, `project:<projectID>:<userID>`, `job:<jobID>:<userID>`) are delivered to user's connections based on topic subscriptions
- **Connection Lifecycle**: Connect → Authenticate → Register → Receive Messages → Disconnect → Cleanup
- **Health Monitoring**: Ping/Pong keep-alive (30s interval) to detect dead connections

### SMAP Microservice Ecosystem
- **Identity Service**: Provides JWT tokens and sets HttpOnly cookies (shared secret required)
- **Other Services**: Publish notifications to Redis channels for this service to deliver
- **Frontend**: Connects via WebSocket from allowed origins (CORS-protected)
- **Redis**: Central message broker for all microservices

### CORS & Security Context
- **HttpOnly Cookies**: Required for secure authentication (no wildcard origins allowed)
- **CORS Configuration**: Environment-aware (production: strict, dev/staging: permissive with private subnets)
- **Allowed Origins**: Production domains + localhost (dev) + private subnets (dev/staging via VPN)
- **Environment Variable**: `ENV` controls CORS mode (production/dev/staging)

## Important Constraints

### Security Constraints
- **HttpOnly Cookie Requirement**: Cannot use wildcard (`*`) origins with credentials (browser security)
- **CORS Validation**: Must specify exact origins, no wildcards
- **JWT Secret**: Must match Identity Service secret for token validation
- **Cookie Configuration**: Must match Identity Service (domain, secure, sameSite)
- **Production Security**: Production mode only allows production domains (no localhost/private subnets)

### Technical Constraints
- **Stateless**: No database, all state in Redis or memory (connections)
- **Redis Dependency**: Service cannot function without Redis connection
- **Connection Limits**: Default 10,000 connections per instance (configurable)
- **Message Size**: Default 512 bytes max (configurable)
- **Go Version**: Requires Go 1.25+ (uses latest language features)

### Operational Constraints
- **Horizontal Scaling**: Multiple instances can run behind load balancer (stateless design)
- **Redis Pattern Subscription**: Uses multiple patterns (`user_noti:*`, `project:*`, `job:*`) to subscribe to all notification channels
- **Graceful Shutdown**: Must handle SIGINT/SIGTERM and close connections cleanly
- **Environment Configuration**: All configuration via environment variables (12-factor app)

## External Dependencies

### Required Services
- **Redis 7.0+**: Message broker for Pub/Sub, must be accessible and authenticated
- **Identity Service**: Provides JWT tokens and cookie configuration (shared JWT secret required)
- **Kubernetes** (production): ConfigMaps and Secrets for environment configuration

### Optional Services
- **Discord Webhooks**: For operational notifications (webhook ID/token in config)

### Integration Points
- **Redis Pub/Sub Channels**: Multiple patterns (`user_noti:<userID>`, `project:<projectID>:<userID>`, `job:<jobID>:<userID>`) for topic-based message routing
- **JWT Validation**: Uses shared secret with Identity Service
- **Cookie Domain**: `.smap.com` (shared with Identity Service)
- **CORS Origins**: Must align with frontend deployment origins

### Network Requirements
- **Inbound**: WebSocket connections on configured port (default 8081)
- **Outbound**: Redis connection (host/port from config)
- **CORS**: Must allow credentials from configured origins
