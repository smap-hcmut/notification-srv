## Context

The WebSocket service currently authenticates via JWT tokens in query parameters (`/ws?token=<JWT>`), which exposes tokens in logs, browser history, and referrer headers. The Identity service and other SMAP microservices have migrated to HttpOnly cookies for improved security.

This change aligns the WebSocket service with the ecosystem-wide authentication standard while maintaining backward compatibility during client migration.

### Constraints
- WebSocket protocol limitations: Cannot set cookies during upgrade handshake
- Must maintain backward compatibility for existing clients
- Shared authentication with Identity service (same JWT secret and cookie configuration)
- CORS requirements for cross-origin WebSocket connections with credentials

### Stakeholders
- Frontend developers (need to update WebSocket client code)
- DevOps team (Kubernetes configuration updates)
- Security team (improved authentication security)

## Goals / Non-Goals

### Goals
- Migrate WebSocket authentication to HttpOnly cookies
- Maintain backward compatibility with query parameter authentication
- Align cookie configuration with Identity service and other microservices
- Enable CORS with credentials for trusted origins
- Improve security by preventing token exposure in URLs and logs

### Non-Goals
- Removing query parameter fallback (deferred to Phase 2)
- Implementing custom cookie management (use Gin's built-in cookie handling)
- Changing JWT token format or validation logic
- Modifying WebSocket message protocol

## Decisions

### Decision 1: Cookie-First with Query Parameter Fallback
**What**: Check for JWT in cookies first, fall back to query parameter if not found

**Why**: 
- Enables gradual client migration without breaking existing connections
- Provides clear migration path for frontend teams
- Reduces deployment risk

**Alternatives Considered**:
- Cookie-only (rejected: breaks existing clients immediately)
- Query parameter-only (rejected: doesn't improve security)
- Dual authentication (rejected: adds unnecessary complexity)

### Decision 2: Reuse Identity Service Cookie Configuration
**What**: Use identical cookie settings as Identity service (`smap_auth_token`, `.smap.com` domain)

**Why**:
- Single sign-on experience across all services
- Simplified configuration management
- Consistent security posture

**Alternatives Considered**:
- Separate WebSocket cookie (rejected: requires separate authentication flow)
- Service-specific cookie domain (rejected: breaks cross-service authentication)

### Decision 3: CORS Configuration with Specific Origins
**What**: Configure CORS to allow credentials from specific trusted origins (not wildcard)

**Why**:
- Browser security requirement: credentials cannot be used with wildcard origins
- Explicit origin list improves security
- Aligns with other SMAP microservices

**Alternatives Considered**:
- Wildcard origin with credentials (rejected: browsers block this)
- No CORS (rejected: breaks cross-origin WebSocket connections)

### Decision 4: Gin Cookie Extraction
**What**: Use Gin's `c.Cookie()` method to extract JWT from cookies

**Why**:
- Built-in, well-tested functionality
- Consistent with HTTP middleware patterns
- Simplifies code

**Alternatives Considered**:
- Manual cookie parsing from headers (rejected: reinvents the wheel)
- Custom cookie middleware (rejected: unnecessary abstraction)

## Technical Approach

### Authentication Flow

```
┌─────────────┐                    ┌──────────────────┐
│   Browser   │  1. Login via      │ Identity Service │
│             │     Identity       │                  │
│             │ ─────────────────► │ Set-Cookie:      │
│             │ ◄───────────────── │ smap_auth_token  │
└─────────────┘                    └──────────────────┘
       │
       │ 2. WebSocket connection with cookie
       │    ws://api.smap.com/ws
       │    Cookie: smap_auth_token=<JWT>
       ▼
┌─────────────────────────────────────────────────────┐
│ WebSocket Service                                   │
│                                                     │
│ 1. Extract token from cookie (primary)             │
│ 2. Fallback to query parameter (backward compat)   │
│ 3. Validate JWT                                     │
│ 4. Upgrade to WebSocket if valid                   │
└─────────────────────────────────────────────────────┘
```

### Token Extraction Logic

```go
// Priority order:
// 1. Cookie (preferred)
token, err := c.Cookie(cookieConfig.Name)
if err != nil || token == "" {
    // 2. Query parameter (fallback for backward compatibility)
    token = c.Query("token")
}

if token == "" {
    return ErrMissingToken
}

// Validate JWT (existing logic unchanged)
userID, err := jwtValidator.ExtractUserID(token)
```

### CORS Configuration

```go
upgrader := websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        allowedOrigins := []string{
            "http://localhost:3000",
            "https://smap.tantai.dev",
            "https://smap-api.tantai.dev",
        }
        for _, allowed := range allowedOrigins {
            if origin == allowed {
                return true
            }
        }
        return false
    },
}
```

## Risks / Trade-offs

### Risk 1: Cookie Not Sent During WebSocket Upgrade
**Risk**: Browsers may not send cookies with WebSocket upgrade requests from certain origins

**Mitigation**: 
- Configure CORS properly with specific origins
- Test with all supported browsers
- Maintain query parameter fallback during migration period

### Risk 2: Client Migration Coordination
**Risk**: Frontend teams may not update WebSocket clients immediately

**Mitigation**:
- Keep query parameter fallback functional
- Document migration clearly
- Provide migration timeline
- Monitor authentication method usage

### Risk 3: Configuration Mismatch Across Services
**Risk**: Different cookie configurations between services breaks authentication

**Mitigation**:
- Use centralized configuration template
- Document required environment variables
- Add validation checks at startup
- Share configuration via Kubernetes ConfigMap

## Migration Plan

### Phase 1: Implementation (This Change)
1. Add cookie configuration to WebSocket service
2. Update authentication logic (cookie-first, query parameter fallback)
3. Configure CORS for credentials
4. Deploy to test environment
5. Validate with frontend team
6. Deploy to production

### Phase 2: Client Migration (Future)
1. Update frontend WebSocket clients to rely on cookies
2. Monitor authentication method usage
3. Communicate deprecation timeline for query parameters
4. Remove query parameter fallback (separate OpenSpec change)

### Rollback Plan
If issues arise:
1. Revert to previous deployment (query parameter only)
2. Cookie configuration is additive, so rollback is safe
3. No database migrations required

## Validation

### Manual Testing
1. Login via Identity service to obtain cookie
2. Connect to WebSocket without query parameter
3. Verify connection succeeds using cookie
4. Test with query parameter (backward compatibility)
5. Test CORS from allowed origins

### Integration Testing
- Test cookie extraction logic
- Test query parameter fallback
- Test authentication failure scenarios
- Test CORS with credentials

## Open Questions

None. All design decisions have been made based on existing Identity service implementation and SMAP microservice patterns.
