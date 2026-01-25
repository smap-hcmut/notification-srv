## ADDED Requirements

### Requirement: HttpOnly Cookie Authentication
The WebSocket service SHALL support authentication via HttpOnly cookies as the primary authentication method.

#### Scenario: Successful authentication with cookie
- **WHEN** a client connects to the WebSocket endpoint with a valid `smap_auth_token` cookie
- **THEN** the service SHALL extract the JWT from the cookie
- **AND** the service SHALL validate the JWT
- **AND** the service SHALL establish the WebSocket connection
- **AND** the service SHALL associate the connection with the authenticated user ID

#### Scenario: Missing cookie with query parameter fallback
- **WHEN** a client connects without a cookie but provides a valid JWT in the `token` query parameter
- **THEN** the service SHALL extract the JWT from the query parameter
- **AND** the service SHALL validate the JWT
- **AND** the service SHALL establish the WebSocket connection
- **AND** the service SHALL log a warning about using deprecated authentication method

#### Scenario: Authentication failure - no credentials
- **WHEN** a client connects without a cookie and without a query parameter
- **THEN** the service SHALL reject the connection with HTTP 401 Unauthorized
- **AND** the service SHALL return error message "missing token parameter"

#### Scenario: Authentication failure - invalid token
- **WHEN** a client connects with an invalid or expired JWT (from cookie or query parameter)
- **THEN** the service SHALL reject the connection with HTTP 401 Unauthorized
- **AND** the service SHALL return error message "invalid or expired token"
- **AND** the service SHALL log the authentication failure

### Requirement: Cookie Configuration
The WebSocket service SHALL use cookie configuration identical to the Identity service for consistent authentication across SMAP microservices.

#### Scenario: Cookie configuration loaded from environment
- **WHEN** the service starts
- **THEN** the service SHALL load cookie configuration from environment variables
- **AND** the cookie name SHALL be `smap_auth_token`
- **AND** the cookie domain SHALL be `.smap.com`
- **AND** the cookie SHALL have HttpOnly flag enabled
- **AND** the cookie SHALL have Secure flag enabled
- **AND** the cookie SHALL have SameSite=Lax policy

#### Scenario: Shared JWT secret validation
- **WHEN** validating a JWT from a cookie
- **THEN** the service SHALL use the same JWT secret as the Identity service
- **AND** the service SHALL successfully validate tokens issued by the Identity service

### Requirement: CORS with Credentials
The WebSocket service SHALL configure CORS to allow credentials from trusted origins for cross-origin WebSocket connections.

#### Scenario: WebSocket upgrade from allowed origin
- **WHEN** a client from an allowed origin (e.g., `https://smap.tantai.dev`) initiates a WebSocket connection
- **THEN** the service SHALL accept the Origin header
- **AND** the service SHALL allow the WebSocket upgrade
- **AND** cookies SHALL be sent with the request

#### Scenario: WebSocket upgrade from disallowed origin
- **WHEN** a client from a non-allowed origin initiates a WebSocket connection
- **THEN** the service SHALL reject the connection
- **AND** the service SHALL not upgrade to WebSocket

#### Scenario: Allowed origins configuration
- **WHEN** the service configures CORS
- **THEN** the allowed origins SHALL include `http://localhost:3000`
- **AND** the allowed origins SHALL include `https://smap.tantai.dev`
- **AND** the allowed origins SHALL include `https://smap-api.tantai.dev`
- **AND** wildcard origins SHALL NOT be allowed when credentials are enabled

### Requirement: Backward Compatibility
The WebSocket service SHALL maintain backward compatibility with query parameter authentication during the migration period.

#### Scenario: Query parameter authentication still works
- **WHEN** a client connects using the legacy `?token=<JWT>` query parameter
- **THEN** the service SHALL authenticate the client successfully
- **AND** the service SHALL establish the WebSocket connection
- **AND** the service SHALL function identically to cookie authentication

#### Scenario: Cookie takes precedence over query parameter
- **WHEN** a client provides both a cookie and a query parameter
- **THEN** the service SHALL use the JWT from the cookie
- **AND** the service SHALL ignore the query parameter
- **AND** the service SHALL authenticate using the cookie value

### Requirement: Security Improvements
The WebSocket service SHALL prevent token exposure in URLs and logs through HttpOnly cookie authentication.

#### Scenario: Token not exposed in server logs
- **WHEN** a client connects using cookie authentication
- **THEN** the JWT token SHALL NOT appear in URL query parameters
- **AND** the JWT token SHALL NOT be logged by web servers or proxies
- **AND** the JWT token SHALL NOT appear in browser history

#### Scenario: Token protected from XSS attacks
- **WHEN** the authentication cookie is set with HttpOnly flag
- **THEN** JavaScript code SHALL NOT be able to access the token
- **AND** XSS attacks SHALL NOT be able to steal the authentication token

#### Scenario: Token transmitted securely
- **WHEN** the authentication cookie is set with Secure flag
- **THEN** the token SHALL only be transmitted over HTTPS connections
- **AND** the token SHALL NOT be sent over unencrypted HTTP connections
