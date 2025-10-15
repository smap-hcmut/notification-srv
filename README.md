## Smap API

### Overview

Smap API is a high-performance Golang backend designed for low-latency APIs, event-driven processing, and secure access control. It exposes an HTTP API, produces/consumes events via RabbitMQ, and integrates with MongoDB and Redis. The codebase prioritizes concurrency, predictable memory footprint, and operational robustness.

### Core Capabilities

- API Layer and RBAC: Authentication via JWT and internal keys; request handling via Gin; structured logging via Zap; configurable CORS, recovery, and error handling middleware.
- Event Orchestration: Producers and consumers using RabbitMQ for asynchronous workflows, decoupling long-running tasks from the HTTP path.
- Caching and Session State: Redis client with configurable pool sizing and standalone/cluster modes.
- Persistence: MongoDB for operational data models (users, roles, sessions, uploads, etc.).
- Notifications and Monitoring: Discord webhook integration for error/incident reporting; SMTP configuration for email delivery.
- OAuth: Google/Facebook/GitLab OAuth configuration plumbing for social sign-in flows.

### Process Topology

- API Server (cmd/api):
  - Loads configuration and secrets from environment variables.
  - Initializes Zap logger, Encrypter, MongoDB, Redis, RabbitMQ, Discord webhook, SMTP options.
  - Starts an HTTP server with mapped handlers and graceful shutdown.
- Consumer (cmd/consumer):
  - Loads identical configuration and secrets.
  - Connects to MongoDB, Redis, RabbitMQ.
  - Initializes OAuth config and runs a long-lived consumer service to process events.

### Key Architectural Modules

- Configuration (config/): Centralized typed config loaded via env (caarlos0/env). Includes HTTP server, logger, Mongo, Redis, RabbitMQ, JWT, encrypter, internal key, OAuth providers, SMTP, WebSocket, Discord.
- HTTP Server (internal/httpserver/): Gin setup, handler mapping, lifecycle management, graceful shutdown.
- Auth and RBAC (internal/auth/): Use cases and HTTP delivery for authentication/authorization flows.
- Users/Roles/Sessions (internal/user, internal/role, internal/session): Use cases, repositories (Mongo), and HTTP delivery for identity and access management.
- Eventing (pkg/rabbitmq, internal/auth/delivery/rabbitmq, internal/core/smtp/rabbitmq): AMQP connection management, channel utilities, and event producers/consumers.
- Persistence (internal/appconfig/mongo, pkg/mongo): Mongo connection, monitoring, and option helpers.
- Cache (pkg/redis): Redis client with pool configuration.
- Logging (pkg/log): Zap logger wrappers.
- Utilities (pkg/encrypter, pkg/email, pkg/response, pkg/paginator, pkg/util, etc.): Encryption, templated emails, response helpers, pagination, validation, time utilities.

### Non-Functional Priorities

- Performance: Concurrency-first design (Golang), minimal allocation patterns in hot paths, configurable pool sizes.
- Latency: Fast request handling via Gin, Redis cache, asynchronous event offloading via RabbitMQ.
- Reliability: Graceful shutdown, connection lifecycle management, health checks via Docker Compose for local development.
- Observability: Structured logs; Discord webhook for incident notifications.

## Getting Started

### Prerequisites

- Go 1.21+ (recommended)
- Docker and Docker Compose (for local dependencies)
- Make

### Clone and bootstrap

```bash
git clone <your-fork-or-repo-url>
cd smap-api
cp env.template .env
# edit .env with real secrets and endpoints
```

### Start local dependencies (Redis, RabbitMQ)

```bash
make build-docker-compose
# or directly: docker compose up --build -d
```

### Run services

- Run API server:

```bash
make run-api
# generates Swagger and runs: go run cmd/api/main.go
```

- Run Consumer:

```bash
make run-consumer
# runs: go run cmd/consumer/main.go
```

### Generate Swagger

```bash
make swagger
# swag init -g cmd/api/main.go
```

## Configuration

Configuration is environment-driven. The following blocks summarize the most important variables. See `config/config.go` and `env.template` for the full list.

- HTTP server and logger:
  - HOST, APP_PORT, API_MODE
  - LOGGER_LEVEL, LOGGER_MODE, LOGGER_ENCODING
- Security:
  - JWT_SECRET
  - ENCRYPT_KEY (for symmetric encryption)
  - INTERNAL_KEY (for internal requests)
- MongoDB:
  - MONGODB_DATABASE
  - MONGODB_ENCODED_URI (full connection string, optionally encrypted)
  - MONGODB_ENABLE_MONITORING
- Redis:
  - REDIS_ADDR (comma-separated or array-like), REDIS_PASSWORD, REDIS_DB
  - REDIS_STANDALONE, REDIS_POOL_SIZE, REDIS_POOL_TIMEOUT, REDIS_MIN_IDLE_CONNS
- RabbitMQ:
  - RABBITMQ_URL
- SMTP (email):
  - SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD, SMTP_FROM, SMTP_FROM_NAME
- OAuth:
  - GOOGLE_OAUTH_* / FACEBOOK_OAUTH_* / GITLAB_OAUTH_*
- WebSocket:
  - WS_READ_BUFFER_SIZE, WS_WRITE_BUFFER_SIZE, WS_MAX_MESSAGE_SIZE, WS_PONG_WAIT, WS_PING_PERIOD, WS_WRITE_WAIT
- Discord:
  - DISCORD_REPORT_BUG_ID, DISCORD_REPORT_BUG_TOKEN

Note: The `env.template` also includes PostgreSQL and MinIO placeholders for future integrations; the current codebase primarily uses MongoDB, Redis, RabbitMQ, SMTP, and OAuth providers.

## Deployment

- Docker: `cmd/api/Dockerfile` builds the API container. Use the provided `docker-compose.yml` for local infra only (Redis, RabbitMQ).
- Kubernetes: See `deployment/deployment.yaml` for a reference manifest and `deployment/smap-api.ngtantai.pro.conf` for an nginx site configuration example.
- CI/CD: `Jenkinsfile` contains a pipeline example for build/deploy automation.

## Project Structure (high level)

```
cmd/
  api/            # HTTP API entrypoint
  consumer/       # RabbitMQ consumer entrypoint
config/           # Typed environment configuration
internal/
  httpserver/     # Gin server, lifecycle
  auth/           # Auth use cases and delivery
  user/           # User use cases, repo, delivery
  role/           # Role use cases, repo, delivery
  session/        # Session use cases, repo, delivery
  consumer/       # Consumer server setup
pkg/
  log/            # Zap logger
  rabbitmq/       # AMQP utilities
  redis/          # Redis client and options
  mongo/          # Mongo helpers
  email/          # SMTP templates and helpers
  encrypter/      # Symmetric encryption
  response/       # Response helpers
  paginator/      # Pagination utilities
```

## API Documentation

- Swagger/OpenAPI is generated from annotations in `cmd/api/main.go` and HTTP handlers.
- Use `make swagger` to refresh `docs/swagger.*` artifacts.

## Operational Notes

- Graceful shutdown: Both API and consumer processes handle SIGINT/SIGTERM.
- Connection lifecycle: Mongo, Redis, RabbitMQ connections are created at startup and closed on shutdown; Redis, RabbitMQ have health checks in local Compose setup.
- Secrets: Do not commit real `.env` values. Use secrets managers in production.

## Roadmap and Extensibility

- Add ClickHouse and PostgreSQL adapters for analytical and relational workloads.
- Expand Alert Service with rule evaluation and delivery guarantees (DLQs, retries).
- Integrate API Gateway pattern in Visualization service or move to dedicated gateway.
- Enhance observability (metrics, tracing) and structured error taxonomies.

## License

Proprietary – all rights reserved unless otherwise noted.
