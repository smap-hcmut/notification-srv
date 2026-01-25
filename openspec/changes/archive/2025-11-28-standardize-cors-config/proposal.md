# Change: Standardize CORS Configuration with Environment Awareness

## Why

The WebSocket service currently uses hardcoded CORS origin lists that require manual updates when developers work from VPN-connected private subnets. This creates maintenance overhead and blocks developer productivity. The service needs environment-aware CORS validation that:
- Maintains strict security in production (only production domains)
- Allows flexible access in dev/staging (localhost + private subnets via VPN)
- Eliminates manual IP configuration
- Follows the standardized CORS pattern used across SMAP microservices

## What Changes

- **MODIFIED**: CORS origin validation becomes environment-aware (production vs dev/staging modes)
- **ADDED**: Environment configuration (`ENV` variable) to control CORS behavior
- **ADDED**: Dynamic origin validation for private subnets in non-production environments
- **ADDED**: Startup logging to indicate active CORS mode
- **MODIFIED**: CORS origin checking logic to support CIDR-based private subnet validation

**BREAKING**: None - production behavior remains identical (strict origins only)

## Impact

- **Affected specs**: `websocket-auth` (CORS with Credentials requirement)
- **Affected code**:
  - `internal/websocket/handler.go` - Replace hardcoded CheckOrigin with environment-aware logic
  - `config/config.go` - Add EnvironmentConfig struct
  - `template.env` - Add ENV variable
  - `k8s/configmap.yaml` - Add ENV configuration
- **Benefits**:
  - Developers can access service via VPN without manual IP configuration
  - Zero maintenance for CORS config when developers join/leave
  - Consistent with SMAP Identity Service CORS pattern
  - Fail-safe defaults (production mode if ENV not set)

