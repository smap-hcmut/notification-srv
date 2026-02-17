# SMAP Notification Service

Core service for SMAP (Social Media Analytics Platform) handling Real-Time Notifications (WebSocket) and Critical Alerts (Discord).

---

## Architecture

```
                    ┌────────────────────────┐
                    │  Backend Services      │
                    │ (Crawler, Analyzer...) │
                    └───────────┬────────────┘
                                │ PUBLISH
                                ▼
                    ┌────────────────────────┐
                    │      Redis Pub/Sub     │
                    └───────────┬────────────┘
                                │ SUBSCRIBE
    ┌───────────────────────────▼───────────────────────────┐
    │                   notification-srv                    │
    │                                                       │
    │   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐ │
    │   │  WebSocket  │   │   Alert     │   │   Redis     │ │
    │   │   Domain    │◄──┤   Domain    │◄──┤  Delivery   │ │
    │   └──────┬──────┘   └──────┬──────┘   └─────────────┘ │
    │          │                 │                          │
    └──────────┼─────────────────┼──────────────────────────┘
               │                 │
      WebSocket│Push             │Webhook
               ▼                 ▼
        ┌────────────┐     ┌────────────┐
        │  Browser   │     │  Discord   │
        │ Dashboards │     │  Channel   │
        └────────────┘     └────────────┘

    Shared:  internal/model  (Scope, Constants)
    Pkg:     pkg/discord, pkg/redis, pkg/jwt...
```

**2 Core Domains:**

- **WebSocket**: Manages connections/hubs, transforms Redis messages, and routes them to connected users.
- **Alert**: Dispatches critical system and business alerts to Discord channels using rich Embeds.

---

## Tech Stack

| Component | Technology | Purpose |
| :--- | :--- | :--- |
| **Language** | Go 1.25+ | Backend Service |
| **Framework** | Gin | HTTP/WebSocket Routing |
| **WebSocket** | Gorilla/Websocket | Connection handling |
| **Broker** | Redis Pub/Sub | Message ingestion from backend |
| **Auth** | JWT (HS256) | Security via HttpOnly Cookie |
| **Alerts** | Discord Webhooks | Critical notifications |
| **Config** | Viper | Configuration management |

---

## Features

- **Real-time Updates**: Push notifications for Data Onboarding, Analytics Pipelines, and Campaign Events.
- **Crisis Alerts**: Automatic detection and dispatch of high-severity alerts (Sentiment Spikes) to Discord.
- **Smart Routing**: Messages are filtered by Project ID and User ID.
- **Robust Auth**: Secure connection upgrade using JWT validation.
- **Graceful Shutdown**: Clean disconnection handling to prevent client errors.
- **Scalable Hub**: Goroutine-per-client model ensuring high concurrency.

---

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose available
- Redis 7+
- Discord Webhook URL (for alerts)

### 1. Clone & Configure

```bash
git clone <repository-url>
cd notification-srv

# Copy config template
cp config/config.example.yaml config/config.yaml

# Edit with your secrets (Redis, JWT, Discord)
nano config/config.yaml
```

### 2. Configure Discord

1. Create a Webhook in your Discord Server (Channel Settings > Integrations > Webhooks).
2. Update `config/config.yaml`:

```yaml
discord:
  webhook_url: "https://discord.com/api/webhooks/..."
  enabled: true
```

### 3. Run Services

```bash
# Start Redis (if using docker-compose)
docker-compose up -d redis

# Run API service
make run
```

### 4. Test

```bash
# Health check
curl http://localhost:8080/health
# Returns: {"status":"healthy", "redis":"connected", ...}

# Connect WebSocket (requires valid token)
wscat -c "ws://localhost:8080/ws?token=VALID_JWT"
```

---

## Configuration

Key settings in `config/config.yaml`:

```yaml
# Environment
environment: "development" # development | production

# Server
server:
  port: 8080

# WebSocket
websocket:
  max_connections: 10000
  read_buffer_size: 1024
  write_buffer_size: 1024
  allowed_origins: ["*"]

# Redis
redis:
  host: "localhost"
  port: 6379
  password: ""

# Discord Alerting
discord:
  webhook_url: "https://discord.com/api/webhooks/..."
```

---

## API & Events

### WebSocket Endpoint

- `GET /ws`
  - **Headers**: `Cookie: smap_auth_token=...` OR **Query**: `?token=...`
  - **Query Params**: `?project_id=...` (optional filter)

### Supported Events (Redis Channels)

- `DATA_ONBOARDING`
- `ANALYTICS_PIPELINE`
- `CRISIS_ALERT`
- `CAMPAIGN_EVENT`
- `SYSTEM`

See [documents/notification.md](documents/notification.md) for detailed payload structures.

---

## Project Structure

```
notification-srv/
├── cmd/
│   └── api/              # Main entry point
├── config/               # Configuration loading
├── internal/
│   ├── websocket/        # Domain: Real-time hub
│   ├── alert/            # Domain: Discord dispatching
│   ├── httpserver/       # Router, Health checks
│   ├── middleware/       # Auth, CORS
│   └── ...
├── pkg/
│   ├── discord/          # Discord client
│   ├── redis/            # Redis client
│   └── ...
├── documents/            # Architecture & Plans
└── README.md             # This file
```

---

## License

Part of SMAP graduation project.

---

**Last Updated**: 17/02/2026
