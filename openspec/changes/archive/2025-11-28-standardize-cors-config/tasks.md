## 1. Configuration Setup
- [x] 1.1 Add `EnvironmentConfig` struct to `config/config.go` with `ENV` field (default: "production")
- [x] 1.2 Update `template.env` to include `ENV=production` variable
- [x] 1.3 Update `k8s/configmap.yaml` to include `ENV: "production"` for production namespace
- [x] 1.4 Wire up environment config in `cmd/server/main.go` to pass to WebSocket handler

## 2. CORS Implementation
- [x] 2.1 Define `productionOrigins` constant in `internal/websocket/handler.go` (production domains only)
- [x] 2.2 Define `privateSubnets` constant with default CIDR ranges (172.16.21.0/24, 172.16.19.0/24, 192.168.1.0/24)
- [x] 2.3 Implement `isPrivateOrigin(origin string) bool` function using Go `net` package for CIDR validation
- [x] 2.4 Implement `isLocalhostOrigin(origin string) bool` function to check localhost/127.0.0.1
- [x] 2.5 Refactor `upgrader.CheckOrigin` to be environment-aware:
  - Production mode: static list check only
  - Dev/Staging mode: dynamic function checking production domains, localhost, and private subnets
- [x] 2.6 Add startup logging to indicate active CORS mode (production vs dev/staging)

## 3. Testing
- [x] 3.1 Create unit tests for `isPrivateOrigin()` with 20+ test cases (valid/invalid IPs, edge cases)
- [x] 3.2 Create unit tests for `isLocalhostOrigin()` with 15+ test cases (various localhost formats, ports)
- [x] 3.3 Create unit tests for production mode CORS (static origins only, rejects private subnets)
- [x] 3.4 Create unit tests for dev/staging mode CORS (allows production domains, localhost, private subnets)
- [x] 3.5 Create unit tests for fail-safe default (empty ENV defaults to production mode)
- [x] 3.6 Achieve 100% test coverage for new CORS functions
- [ ] 3.7 Test locally with ENV=production (verify strict mode)
- [ ] 3.8 Test locally with ENV=dev (verify permissive mode with localhost)

## 4. Documentation
- [x] 4.1 Update `README.md` with ENV configuration section
- [x] 4.2 Document CORS behavior table (production vs dev/staging modes)
- [x] 4.3 Document private subnet list and customization options
- [x] 4.4 Add security warnings about ENV configuration
- [ ] 4.5 Create K8s ConfigMap examples for dev, staging, and production environments

## 5. Deployment Validation
- [ ] 5.1 Deploy to staging with ENV=staging
- [ ] 5.2 Verify VPN access from private subnets works in staging
- [ ] 5.3 Verify production origins work in staging
- [ ] 5.4 Verify localhost access works in staging
- [ ] 5.5 Deploy to production with ENV=production
- [ ] 5.6 Verify only production origins work in production
- [ ] 5.7 Verify private subnets are blocked in production
- [ ] 5.8 Monitor logs for 24-48 hours for CORS-related issues
- [ ] 5.9 Verify startup logs show correct CORS mode

