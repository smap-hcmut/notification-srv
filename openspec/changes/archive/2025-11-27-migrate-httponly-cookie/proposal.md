# Change: Migrate WebSocket Authentication to HttpOnly Cookies

## Why

The WebSocket service currently authenticates users via JWT tokens passed as query parameters (`/ws?token=<JWT>`). This approach has several security vulnerabilities:

1. **Token Exposure in Logs**: Query parameters are logged by proxies, load balancers, and web servers, exposing JWT tokens in plain text
2. **Browser History Leakage**: URLs with tokens are stored in browser history, making them accessible to malicious scripts or users
3. **Referrer Header Leakage**: Tokens can leak through HTTP Referrer headers when navigating to external sites
4. **No XSS Protection**: Unlike HttpOnly cookies, query parameters offer no protection against XSS attacks

The Identity service and other SMAP microservices have already migrated to HttpOnly cookie authentication. This change brings the WebSocket service into alignment with the ecosystem-wide security standard.

## What Changes

This change migrates WebSocket authentication from query parameter tokens to HttpOnly cookies while maintaining backward compatibility during the transition period.

### Core Changes
- Add `CookieConfig` to service configuration
- Update WebSocket handler to read JWT from cookies (primary) with query parameter fallback
- Configure CORS to allow credentials from trusted origins
- Update documentation and examples to use cookie-based authentication
- Add environment variables for cookie configuration

### Migration Strategy
- **Phase 1**: Cookie-first with query parameter fallback (this change)
- **Phase 2**: Remove query parameter fallback after client migration (future change)

### Breaking Changes
None during Phase 1. Query parameter authentication remains functional for backward compatibility.

## Impact

### Affected Specs
- `websocket-auth` (new capability) - WebSocket authentication requirements

### Affected Code
- `config/config.go` - Add CookieConfig struct
- `template.env` - Add cookie environment variables
- `internal/websocket/handler.go` - Update token extraction logic
- `internal/websocket/handler.go` - Update CORS configuration
- `cmd/server/main.go` - Pass cookie config to handler
- `README.md` - Update authentication documentation
- `k8s/configmap.yaml` - Add cookie configuration for Kubernetes deployment

### Dependencies
- Requires Identity service to be running with HttpOnly cookie support
- All services must share the same JWT secret and cookie configuration
- Frontend clients must use `credentials: 'include'` for WebSocket connections

### Security Improvements
- ✅ Tokens no longer exposed in URLs or logs
- ✅ HttpOnly flag prevents JavaScript access to tokens
- ✅ Secure flag ensures HTTPS-only transmission
- ✅ SameSite protection against CSRF attacks
- ✅ Consistent authentication across all SMAP services
