## Context

The WebSocket service currently hardcodes CORS allowed origins in `internal/websocket/handler.go`. This approach:
- Blocks developers working from VPN-connected private subnets (172.16.x.x, 192.168.x.x)
- Requires manual IP updates when developers join/leave
- Doesn't differentiate between production and development environments
- Creates inconsistency with other SMAP microservices (Identity Service uses environment-aware CORS)

The SMAP Identity Service has implemented a standardized CORS pattern that this change will replicate.

## Goals / Non-Goals

### Goals
- Enable VPN access from private subnets in dev/staging without manual configuration
- Maintain strict production security (only production domains allowed)
- Follow fail-safe defaults (production mode if ENV not set)
- Align with SMAP Identity Service CORS pattern for consistency
- Support HttpOnly cookie authentication (requires specific origins, no wildcards)

### Non-Goals
- Changing production CORS behavior (must remain strict)
- Supporting wildcard origins (incompatible with credentials)
- Adding external dependencies for CIDR validation (use Go stdlib)
- Changing authentication mechanisms (still uses HttpOnly cookies)

## Decisions

### Decision 1: Environment Variable for CORS Mode
**What**: Use `ENV` environment variable to control CORS behavior
**Why**: 
- Simple, standard approach used across SMAP microservices
- Fail-safe: defaults to production if not set
- Clear separation: production vs non-production (dev/staging)

**Alternatives considered**:
- Separate `CORS_MODE` variable (rejected: adds complexity, ENV is standard)
- Config file (rejected: environment variables are standard for K8s deployments)

### Decision 2: Dynamic Origin Validation for Non-Production
**What**: Use `AllowOriginFunc` (CheckOrigin for WebSocket) with CIDR validation for private subnets
**Why**:
- Allows any IP in configured private subnets without hardcoding
- Supports VPN access from 172.16.x.x, 192.168.x.x ranges
- Maintains security: only specific subnets allowed, not all private IPs

**Alternatives considered**:
- Hardcode all possible IPs (rejected: unmaintainable, 762+ entries)
- Wildcard for private IPs (rejected: too permissive, security risk)
- No private subnet support (rejected: blocks developer productivity)

### Decision 3: Private Subnet Configuration
**What**: Default subnets: `172.16.21.0/24`, `172.16.19.0/24`, `192.168.1.0/24`
**Why**:
- Covers K8s cluster subnet and common private networks
- Can be customized per service if needed
- Uses Go stdlib `net` package for efficient CIDR validation

**Alternatives considered**:
- Single subnet (rejected: doesn't cover all use cases)
- All private IP ranges (rejected: too permissive)

### Decision 4: Production Mode Uses Static List
**What**: Production mode uses static `AllowedOrigins` list (no dynamic function)
**Why**:
- Simpler, more predictable for production
- Better performance (no function call per request)
- Clearer security model (explicit allow-list)

**Alternatives considered**:
- Dynamic function for all modes (rejected: unnecessary complexity in production)

## Risks / Trade-offs

### Risk: ENV Misconfiguration in Production
**Mitigation**: 
- Fail-safe default (production mode if ENV empty/undefined)
- Startup logging shows active CORS mode
- K8s ConfigMaps explicitly set ENV=production for production namespace

### Risk: Private Subnet Access in Production
**Mitigation**:
- Private subnets only allowed in dev/staging modes
- Production mode uses static list only
- Code review ensures ENV=production in production deployments

### Trade-off: Complexity vs Flexibility
- **Complexity**: Adds environment awareness and CIDR validation
- **Benefit**: Eliminates manual maintenance, enables VPN access
- **Acceptable**: Complexity is localized to CORS configuration, well-tested pattern from Identity Service

## Migration Plan

1. **Implementation**: Add environment config and refactor CORS logic
2. **Testing**: Unit tests for all origin validation scenarios
3. **Staging Deployment**: Deploy with ENV=staging, verify VPN access works
4. **Production Deployment**: Deploy with ENV=production, verify strict mode active
5. **Monitoring**: Check logs for CORS mode and any rejection issues

**Rollback**: Revert to previous hardcoded origins if issues arise (simple code revert)

## Open Questions

None - pattern is proven from SMAP Identity Service implementation.

