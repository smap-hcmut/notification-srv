## ADDED Requirements

### Requirement: CORS with Credentials
The WebSocket service SHALL configure CORS to allow credentials from trusted origins for cross-origin WebSocket connections. The service SHALL use environment-aware origin validation that differs between production and non-production environments.

#### Scenario: WebSocket upgrade from allowed origin in production
- **WHEN** a client from a production origin (e.g., `https://smap.tantai.dev`) initiates a WebSocket connection
- **AND** the service is running in production mode (ENV=production)
- **THEN** the service SHALL accept the Origin header
- **AND** the service SHALL allow the WebSocket upgrade
- **AND** cookies SHALL be sent with the request

#### Scenario: WebSocket upgrade from disallowed origin
- **WHEN** a client from a non-allowed origin initiates a WebSocket connection
- **THEN** the service SHALL reject the connection
- **AND** the service SHALL not upgrade to WebSocket

#### Scenario: Production mode origin validation
- **WHEN** the service is running in production mode (ENV=production)
- **THEN** the service SHALL only allow origins from the production origins list
- **AND** the service SHALL reject localhost origins
- **AND** the service SHALL reject private subnet origins (e.g., 172.16.x.x, 192.168.x.x)
- **AND** wildcard origins SHALL NOT be allowed when credentials are enabled

#### Scenario: Development/staging mode origin validation
- **WHEN** the service is running in development or staging mode (ENV=dev or ENV=staging)
- **THEN** the service SHALL allow production origins
- **AND** the service SHALL allow localhost origins (any port)
- **AND** the service SHALL allow origins from configured private subnets (CIDR validation)
- **AND** wildcard origins SHALL NOT be allowed when credentials are enabled

#### Scenario: Environment configuration default
- **WHEN** the ENV environment variable is not set or is empty
- **THEN** the service SHALL default to production mode
- **AND** the service SHALL use strict origin validation (production origins only)

#### Scenario: Private subnet origin validation
- **WHEN** a client from a private subnet IP (e.g., `http://172.16.21.50:3000`) initiates a WebSocket connection
- **AND** the service is running in development or staging mode
- **AND** the IP address falls within a configured private subnet CIDR range
- **THEN** the service SHALL accept the Origin header
- **AND** the service SHALL allow the WebSocket upgrade

#### Scenario: Private subnet rejection in production
- **WHEN** a client from a private subnet IP initiates a WebSocket connection
- **AND** the service is running in production mode
- **THEN** the service SHALL reject the connection
- **AND** the service SHALL not upgrade to WebSocket

#### Scenario: CORS mode logging
- **WHEN** the service starts
- **THEN** the service SHALL log the active CORS mode (production or dev/staging)
- **AND** the log message SHALL indicate whether strict or permissive validation is active

