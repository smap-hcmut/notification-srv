# Refactor Plan: notification-srv → knowledge-srv Convention

## Phase 1: ✅ PKG Layer (COMPLETED)
Đã refactor tất cả packages theo 4-file pattern (interface.go, type.go, constant.go, <pkg>.go)

---

## Phase 2: CMD & CONFIG Structure

### Current Structure (notification-srv)
```
cmd/
└── server/
    ├── main.go          # Single entry point
    ├── Dockerfile
    └── deployment.yaml

config/
└── config.go            # Uses env vars only (caarlos0/env)
```

### Target Structure (knowledge-srv convention)
```
cmd/
├── api/                 # HTTP API server
│   ├── main.go
│   ├── Dockerfile
│   └── deployment.yaml
└── consumer/            # Background consumer (if needed)
    ├── main.go
    ├── Dockerfile
    └── deployment.yaml

config/
├── config.go            # Main config loader (Viper)
├── redis/
│   └── connect.go       # Redis connection helper
└── <service>/
    └── connect.go       # Other service connections
```

### Key Differences

#### 1. Config Loading
**Current (notification-srv):**
- Uses `caarlos0/env` - environment variables only
- Simple struct tags: `env:"REDIS_HOST"`
- No config file support

**Target (knowledge-srv):**
- Uses `spf13/viper` - supports YAML + env vars
- Config file: `knowledge-config.yaml`
- Env var override with `viper.AutomaticEnv()`
- Validation in `validate()` function

#### 2. CMD Structure
**Current:**
- Single `cmd/server/main.go` - WebSocket server only

**Target:**
- `cmd/api/main.go` - HTTP API server
- `cmd/consumer/main.go` - Kafka/background consumer
- Separate Dockerfiles and deployments

#### 3. Connection Helpers
**Current:**
- Direct initialization in main.go:
  ```go
  redisClient, err := redis.NewClient(redis.Config{...})
  ```

**Target:**
- Centralized in config/<service>/connect.go:
  ```go
  // config/redis/connect.go
  func Connect(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error)
  func Disconnect() error
  ```

---

## Phase 3: INTERNAL Layer Structure

### Current Structure (notification-srv)
```
internal/
├── auth/              # ⚠️ Utility layer (should be pkg?)
├── redis/             # ⚠️ Infrastructure (subscriber only)
├── server/            # ⚠️ Infrastructure (HTTP server)
├── transform/         # ⚠️ Business logic (not domain)
├── types/             # ✅ Shared types
└── websocket/         # ✅ Delivery layer (WebSocket)
```

### Target Structure (knowledge-srv convention)
```
internal/
├── <domain>/          # Domain modules (e.g., indexing, chat, search)
│   ├── delivery/
│   │   ├── http/
│   │   │   ├── handlers.go
│   │   │   ├── process_request.go
│   │   │   ├── presenters.go
│   │   │   ├── routes.go
│   │   │   ├── errors.go
│   │   │   └── new.go
│   │   └── kafka/
│   │       └── consumer/
│   ├── repository/
│   │   ├── interface.go
│   │   ├── option.go
│   │   ├── errors.go
│   │   └── postgre/
│   ├── usecase/
│   │   ├── new.go
│   │   ├── <method>.go
│   │   └── helpers.go
│   ├── interface.go
│   ├── types.go
│   └── errors.go
├── httpserver/        # HTTP server wiring
├── consumer/          # Consumer server wiring
├── middleware/        # Middleware (auth, cors, etc.)
└── model/             # Shared domain models
```

---

## Refactor Actions

### Action 1: Migrate Config System
**Priority: HIGH**

1. **Replace env parser:**
   - Remove: `github.com/caarlos0/env/v9`
   - Add: `github.com/spf13/viper`

2. **Create config file:**
   - `config/notification-config.yaml` (or `websocket-config.yaml`)
   - Support both YAML and env vars

3. **Refactor config.go:**
   - Use Viper for loading
   - Add `setDefaults()` function
   - Add `validate()` function
   - Keep struct fields but change tags

4. **Create connection helpers:**
   - `config/redis/connect.go`
   - Pattern: `Connect(ctx, cfg) (*Client, error)` + `Disconnect() error`

### Action 2: Restructure CMD
**Priority: HIGH**

1. **Rename cmd/server → cmd/api:**
   ```bash
   mv cmd/server cmd/api
   ```

2. **Update main.go:**
   - Follow knowledge-srv pattern
   - Initialize all dependencies in order:
     1. Config
     2. Logger
     3. Context with signal handling
     4. Infrastructure (Redis, etc.)
     5. Core utilities (JWT, Discord)
     6. UseCases
     7. Delivery (HTTP/WebSocket)
     8. Server

3. **Add cmd/consumer (if needed):**
   - For background jobs
   - Kafka consumer
   - Redis subscriber (move from internal/redis)

### Action 3: Refactor INTERNAL Layer
**Priority: MEDIUM**

#### 3.1 Move/Refactor Utility Layers

**internal/auth → pkg/auth:**
- Already has security utilities
- Should be reusable package
- Refactor to 4-file pattern

**internal/transform → pkg/transform:**
- Message transformation logic
- Not domain-specific
- Refactor to 4-file pattern

**internal/redis → internal/subscriber (or cmd/consumer):**
- Redis subscriber is infrastructure
- Should be part of consumer service
- Or create proper domain module

**internal/server → internal/httpserver:**
- Rename for consistency
- Follow knowledge-srv pattern

#### 3.2 Create Domain Modules (if needed)

**Option A: Keep WebSocket as main domain**
```
internal/
└── websocket/
    ├── delivery/
    │   └── ws/
    │       ├── handler.go
    │       ├── hub.go
    │       ├── connection.go
    │       └── ...
    ├── usecase/
    │   ├── new.go
    │   ├── broadcast.go
    │   └── subscribe.go
    ├── interface.go
    ├── types.go
    └── errors.go
```

**Option B: Create notification domain**
```
internal/
└── notification/
    ├── delivery/
    │   ├── ws/
    │   └── http/
    ├── usecase/
    ├── repository/ (if needed)
    ├── interface.go
    ├── types.go
    └── errors.go
```

---

## Migration Checklist

### Phase 2A: Config Migration
- [ ] Add `spf13/viper` dependency
- [ ] Create `config/notification-config.yaml`
- [ ] Refactor `config/config.go` to use Viper
- [ ] Add `setDefaults()` function
- [ ] Add `validate()` function
- [ ] Create `config/redis/connect.go`
- [ ] Test config loading (YAML + env override)

### Phase 2B: CMD Restructure
- [ ] Rename `cmd/server` → `cmd/api`
- [ ] Refactor `cmd/api/main.go` following knowledge-srv pattern
- [ ] Update Dockerfile paths
- [ ] Update deployment.yaml
- [ ] Test server startup

### Phase 3A: Move Utilities to PKG
- [ ] Move `internal/auth` → `pkg/auth`
- [ ] Refactor `pkg/auth` to 4-file pattern
- [ ] Move `internal/transform` → `pkg/transform`
- [ ] Refactor `pkg/transform` to 4-file pattern
- [ ] Update all imports

### Phase 3B: Refactor Infrastructure
- [ ] Rename `internal/server` → `internal/httpserver`
- [ ] Refactor `internal/httpserver` following knowledge-srv pattern
- [ ] Move Redis subscriber logic appropriately
- [ ] Create `internal/consumer` if needed

### Phase 3C: Refactor WebSocket Domain
- [ ] Decide on domain structure (Option A or B)
- [ ] Create proper layer separation (delivery/usecase/repository)
- [ ] Move files to correct locations
- [ ] Create interface.go, types.go, errors.go at module root
- [ ] Update all imports

---

## Notes

### Why Viper over env?
1. **Flexibility**: Supports multiple config sources (file, env, flags)
2. **Defaults**: Centralized default values
3. **Validation**: Explicit validation logic
4. **Override**: File → Env → Flag precedence
5. **Standard**: Used across SMAP services

### Why Separate CMD?
1. **Separation of Concerns**: API vs Background processing
2. **Deployment**: Different scaling strategies
3. **Resources**: Different resource requirements
4. **Monitoring**: Separate health checks and metrics

### Why Connection Helpers?
1. **Reusability**: Shared connection logic
2. **Cleanup**: Centralized disconnect logic
3. **Testing**: Easier to mock
4. **Consistency**: Same pattern across services

---

## Estimated Effort

| Phase | Tasks | Effort | Risk |
|-------|-------|--------|------|
| 2A: Config Migration | 7 | 2-3h | Low |
| 2B: CMD Restructure | 5 | 1-2h | Low |
| 3A: Move to PKG | 5 | 2-3h | Medium |
| 3B: Refactor Infrastructure | 4 | 2-3h | Medium |
| 3C: Refactor Domain | 6 | 3-4h | High |
| **Total** | **27** | **10-15h** | **Medium** |

---

## Next Steps

1. ✅ Complete Phase 1 (PKG refactor) - DONE
2. 🔄 Start Phase 2A (Config migration)
3. → Phase 2B (CMD restructure)
4. → Phase 3A (Move utilities)
5. → Phase 3B (Infrastructure)
6. → Phase 3C (Domain refactor)
