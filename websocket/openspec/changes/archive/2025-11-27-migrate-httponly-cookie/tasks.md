## 1. Configuration Setup
- [x] 1.1 Add `CookieConfig` struct to `config/config.go`
- [x] 1.2 Add cookie environment variables to `template.env`
- [x] 1.3 Update `k8s/configmap.yaml` with cookie configuration

## 2. Authentication Implementation
- [x] 2.1 Update `internal/websocket/handler.go` to extract JWT from cookies (primary method)
- [x] 2.2 Add query parameter fallback for backward compatibility
- [x] 2.3 Update CORS configuration in `internal/websocket/handler.go` to allow credentials
- [x] 2.4 Pass `CookieConfig` to WebSocket handler in `cmd/server/main.go`

## 3. Documentation
- [x] 3.1 Update `README.md` with cookie authentication instructions
- [x] 3.2 Add frontend integration examples (JavaScript WebSocket with credentials)
- [x] 3.3 Document migration path from query parameter to cookie authentication

## 4. Testing
- [x] 4.1 Test WebSocket connection with cookie authentication
- [x] 4.2 Test backward compatibility with query parameter authentication
- [x] 4.3 Test CORS with credentials from allowed origins
- [x] 4.4 Verify cookie is sent automatically by browser
- [x] 4.5 Test authentication failure scenarios (missing/invalid cookie)

## 5. Deployment
- [x] 5.1 Update Kubernetes ConfigMap with cookie environment variables
- [x] 5.2 Deploy to test environment
- [x] 5.3 Validate with frontend team
- [x] 5.4 Monitor authentication metrics
