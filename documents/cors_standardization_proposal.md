# CORS Standardization Proposal for SMAP Microservices

**Author**: SMAP Identity Service Team
**Date**: November 28, 2025
**Version**: 1.0
**Status**: Proposed

---

## Executive Summary

This document proposes a standardized approach to CORS (Cross-Origin Resource Sharing) configuration across all SMAP microservices. The solution is based on the implementation in the SMAP Identity Service, which provides environment-aware CORS validation while maintaining security in production.

### Key Benefits
- ✅ **Developer Productivity**: Enable VPN access from private subnets without manual IP configuration
- ✅ **Security First**: Strict CORS validation in production, permissive only in dev/staging
- ✅ **Zero Maintenance**: No need to update configs when developers join/leave
- ✅ **Fail-Safe**: Defaults to production mode for security
- ✅ **Consistent**: Same patterns across all microservices

---

## Problem Statement

### Current Challenges

1. **Hardcoded Origin Lists**
   - Must manually add each developer's IP to CORS configuration
   - 254 IPs × 3 subnets = 762 potential entries (unmaintainable)
   - Developers blocked when working via VPN from private networks

2. **HttpOnly Cookie Constraints**
   - Requires `AllowCredentials: true`
   - Cannot use wildcard (`*`) origins with credentials (browser security)
   - Must specify exact origins

3. **Environment Differences**
   - Production needs strict security (only production domains)
   - Development needs flexibility (localhost + VPN access)
   - No standardized way to differentiate between environments

### Impact
- Reduced developer productivity (forced to work on localhost only)
- Manual configuration overhead (DevOps bottleneck)
- Inconsistent CORS handling across microservices
- Security risks from ad-hoc workarounds

---

## Proposed Solution

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│          Environment Variable: ENV                  │
│   Values: "production" | "staging" | "dev"         │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
         ┌─────────────────────┐
         │  DefaultCORSConfig  │
         │   (environment)     │
         └──────────┬──────────┘
                    │
        ┌───────────┴───────────┐
        │                       │
        ▼                       ▼
┌──────────────┐      ┌────────────────────┐
│  Production  │      │  Dev/Staging       │
│              │      │                    │
│ Static list  │      │ AllowOriginFunc    │
│ of domains   │      │ (dynamic check)    │
└──────────────┘      └─────────┬──────────┘
                                │
                    ┌───────────┼───────────┐
                    │           │           │
                    ▼           ▼           ▼
              Production   Localhost   Private
               Domains                 Subnets
```

### Core Components

#### 1. Environment Configuration

**Environment Variable**: `ENV`

| Value | CORS Mode | Use Case |
|-------|-----------|----------|
| `production` | Strict (static origins only) | Production deployments |
| `staging` | Permissive (dynamic validation) | Staging/QA environments |
| `dev` | Permissive (dynamic validation) | Local development |
| *(empty)* | Defaults to `production` | Fail-safe security |

#### 2. Origin Validation Logic

**Production Mode** (`ENV=production`):
```go
config.AllowedOrigins = productionOrigins
// Only: https://smap.tantai.dev, https://smap-api.tantai.dev
```

**Non-Production Mode** (`ENV=dev` or `ENV=staging`):
```go
config.AllowOriginFunc = func(origin string) bool {
    // 1. Check production domains
    // 2. Check localhost (any port)
    // 3. Check private subnets (CIDR validation)
    return allowed
}
```

#### 3. Private Subnet Configuration

Default subnets (can be customized per microservice):
- `172.16.21.0/24` - K8s cluster subnet
- `172.16.19.0/24` - Private network 1
- `192.168.1.0/24` - Private network 2

**CIDR Validation**: Uses Go's `net` package for efficient IP-in-subnet checks (~microseconds)

#### 4. Security Features

1. **Fail-Safe Default**: Missing/empty ENV → production mode
2. **No Wildcards**: Always specific origins (HttpOnly cookie compatible)
3. **Environment Isolation**: Private subnets only in non-production
4. **Audit Trail**: Startup logging shows active CORS mode
5. **Type-Safe**: stdlib-based validation (no external dependencies)

---

## Implementation Guide

### Phase 1: Core Implementation (Per Microservice)

#### Step 1: Add Environment Configuration

```go
// config/config.go
type Config struct {
    Environment EnvironmentConfig
    // ... other fields
}

type EnvironmentConfig struct {
    Name string `env:"ENV" envDefault:"production"`
}
```

#### Step 2: Update CORS Middleware

```go
// internal/middleware/cors.go

// Add private subnet and production origin constants
var privateSubnets = []string{
    "172.16.21.0/24",
    "172.16.19.0/24",
    "192.168.1.0/24",
}

var productionOrigins = []string{
    "https://your-service.tantai.dev",
}

// Add helper functions
func isPrivateOrigin(origin string) bool {
    u, _ := url.Parse(origin)
    ip := net.ParseIP(u.Hostname())
    if ip == nil {
        return false
    }

    for _, cidr := range privateSubnets {
        _, subnet, _ := net.ParseCIDR(cidr)
        if subnet.Contains(ip) {
            return true
        }
    }
    return false
}

func isLocalhostOrigin(origin string) bool {
    return strings.HasPrefix(origin, "http://localhost") ||
           strings.HasPrefix(origin, "http://127.0.0.1") ||
           strings.HasPrefix(origin, "https://localhost") ||
           strings.HasPrefix(origin, "https://127.0.0.1")
}

// Refactor CORS config
func DefaultCORSConfig(environment string) CORSConfig {
    if environment == "" {
        environment = "production"
    }

    config := CORSConfig{
        AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"Origin", "Content-Type", "Authorization", "Accept"},
        AllowCredentials: true,
        MaxAge: 86400,
    }

    if environment == "production" {
        config.AllowedOrigins = productionOrigins
        return config
    }

    // Non-production: dynamic validation
    config.AllowOriginFunc = func(origin string) bool {
        // Check production domains
        for _, allowed := range productionOrigins {
            if origin == allowed {
                return true
            }
        }

        // Check localhost
        if isLocalhostOrigin(origin) {
            return true
        }

        // Check private subnets
        if isPrivateOrigin(origin) {
            return true
        }

        return false
    }

    return config
}
```

#### Step 3: Wire Up Configuration

```go
// cmd/api/main.go or main entrypoint
httpServer, err := httpserver.New(logger, httpserver.Config{
    Environment: cfg.Environment.Name,  // Pass ENV from config
    // ... other config
})
```

```go
// internal/httpserver/handler.go
func (srv HTTPServer) registerMiddlewares(mw middleware.Middleware) {
    corsConfig := middleware.DefaultCORSConfig(srv.environment)
    srv.gin.Use(middleware.CORS(corsConfig))

    // Log CORS mode
    ctx := context.Background()
    if srv.environment == "production" {
        srv.l.Infof(ctx, "CORS mode: production (strict origins only)")
    } else {
        srv.l.Infof(ctx, "CORS mode: %s (permissive - allows localhost and private subnets)", srv.environment)
    }
}
```

#### Step 4: Update Environment Files

```bash
# template.env
ENV=production  # Values: production | staging | dev
```

#### Step 5: Create Kubernetes ConfigMaps

See `deployment/k8s/configmap-*.yaml` examples in SMAP Identity Service.

### Phase 2: Testing

#### Unit Tests Template

```go
// internal/middleware/cors_test.go
func TestIsPrivateOrigin(t *testing.T) {
    tests := []struct {
        name   string
        origin string
        want   bool
    }{
        {"K8s subnet IP", "http://172.16.21.50:3000", true},
        {"Public IP", "http://1.2.3.4:3000", false},
        // ... more cases
    }
    // ... test implementation
}

func TestIsLocalhostOrigin(t *testing.T) { /* ... */ }
func TestDefaultCORSConfig(t *testing.T) { /* ... */ }
func TestDefaultCORSConfigAllowOriginFunc(t *testing.T) { /* ... */ }
```

**Coverage Target**: 100% for new CORS functions

### Phase 3: Documentation

1. **README.md**: Add ENV configuration section
2. **Deployment Docs**: Document K8s ConfigMap setup
3. **Security Notes**: Explain production vs non-production modes
4. **Developer Guide**: How to test from VPN

### Phase 4: Rollout Strategy

**Per Microservice**:
1. Implement changes in feature branch
2. Run unit tests (100% coverage target)
3. Test locally with ENV=production and ENV=dev
4. Deploy to staging with ENV=staging
5. Verify VPN access works
6. Deploy to production with ENV=production
7. Monitor logs for 24-48 hours

**Validation Checklist**:
- [ ] Production origins work in all modes
- [ ] Localhost works in dev/staging only
- [ ] Private subnets work in dev/staging only
- [ ] Private subnets blocked in production
- [ ] HttpOnly cookies still function correctly
- [ ] Startup logs show correct CORS mode
- [ ] No CORS-related errors in logs

---

## Microservice Adoption Roadmap

### Priority 1: High-Traffic Services (Week 1-2)
- **Identity Service** ✅ (Reference Implementation)
- **API Gateway** (if applicable)
- **Auth Service** (if separate from identity)

### Priority 2: Core Services (Week 3-4)
- **User Service**
- **Payment Service**
- **Notification Service**

### Priority 3: Supporting Services (Week 5-6)
- **Analytics Service**
- **Logging Service**
- **Admin Service**

### Timeline Overview

| Week | Milestone | Deliverable |
|------|-----------|-------------|
| 1 | Reference Implementation | Identity Service complete with tests and docs |
| 2-3 | High-Priority Services | 3 services migrated, validated in staging |
| 4-5 | Core Services | 3 more services migrated |
| 6 | Supporting Services | Remaining services migrated |
| 7 | Production Rollout | All services deployed to production |
| 8 | Monitoring & Refinement | Collect feedback, address issues |

---

## Customization Guidelines

### Per-Microservice Customization

**Production Origins**: Update for each service
```go
var productionOrigins = []string{
    "https://your-service.tantai.dev",
    "https://another-domain.com",
}
```

**Private Subnets**: Adjust if network topology differs
```go
var privateSubnets = []string{
    "172.16.21.0/24",  // Your K8s cluster
    "10.0.0.0/16",     // Your private network
}
```

**Allowed Headers**: Add service-specific headers
```go
AllowedHeaders: []string{
    "Origin",
    "Content-Type",
    "Authorization",
    "X-Service-Specific-Header",  // Custom header
}
```

### Framework-Specific Adaptations

**For Non-Go Services** (Node.js, Python, etc.):
- Implement same logic in respective language
- Use equivalent CIDR validation libraries
- Maintain same environment variable conventions
- Follow same production/dev/staging modes

**For Different HTTP Frameworks**:
- Adapt middleware registration to framework patterns
- Ensure AllowOriginFunc equivalent exists or implement custom logic
- Maintain fail-safe defaults

---

## Security Considerations

### Threat Model

| Threat | Mitigation |
|--------|------------|
| **Permissive CORS in production** | Default to production mode, require explicit ENV=dev |
| **CSRF attacks from dev env** | Dev environments not internet-accessible (VPN-only) |
| **IP spoofing in private subnets** | Private subnets behind VPN, not publicly routable |
| **ENV misconfiguration** | Startup logging, fail-safe default, automated tests |

### Best Practices

1. **Never** set `ENV=dev` or `ENV=staging` in production
2. **Always** validate ENV value on startup and log the mode
3. **Test** CORS behavior in each environment before deployment
4. **Monitor** CORS rejection logs for anomalies
5. **Audit** ConfigMaps/environment configs regularly
6. **Review** private subnet list when network changes

### Compliance

- ✅ **OWASP**: Follows CORS security best practices
- ✅ **Browser Security**: Compatible with credential-based authentication
- ✅ **Zero Trust**: Explicit allow-list (no wildcards)
- ✅ **Least Privilege**: Minimal access in production

---

## Metrics & Monitoring

### Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Developer VPN Access | 100% of devs can access via VPN | Survey + access logs |
| CORS Config Changes | 90% reduction in manual updates | DevOps ticket volume |
| Production Security Incidents | 0 CORS-related breaches | Security audit logs |
| Deployment Time | <5 min per service | CI/CD metrics |

### Monitoring Queries

**Startup Log Check** (verify CORS mode):
```
service:identity-api "CORS mode:"
```

**CORS Rejection Rate** (should be low):
```
service:identity-api level:warn "origin rejected"
```

**Environment Misconfiguration** (should be 0):
```
service:* env:production "CORS mode: dev"
```

### Alerting

- **Critical**: Production service with ENV=dev or ENV=staging
- **Warning**: CORS rejection rate >5% in production
- **Info**: CORS mode different from expected for environment

---

## Replication Checklist

Use this checklist when implementing CORS standardization in a new microservice:

### Planning
- [ ] Review this proposal and SMAP Identity Service implementation
- [ ] Identify service-specific production origins
- [ ] Confirm private subnet CIDR ranges
- [ ] Check if service uses HttpOnly cookies (requires `AllowCredentials: true`)

### Implementation
- [ ] Add `EnvironmentConfig` to config package
- [ ] Update template.env with ENV variable
- [ ] Implement `isPrivateOrigin()` function
- [ ] Implement `isLocalhostOrigin()` function
- [ ] Define `privateSubnets` and `productionOrigins` variables
- [ ] Refactor `DefaultCORSConfig()` to accept environment parameter
- [ ] Update CORS middleware to handle `AllowOriginFunc`
- [ ] Wire up environment config in main entrypoint
- [ ] Add startup logging for CORS mode

### Testing
- [ ] Create unit tests for `isPrivateOrigin()` (20+ cases)
- [ ] Create unit tests for `isLocalhostOrigin()` (15+ cases)
- [ ] Create unit tests for `DefaultCORSConfig()` (5+ cases)
- [ ] Create unit tests for `AllowOriginFunc` (10+ cases)
- [ ] Achieve 100% test coverage for new functions
- [ ] Test locally with ENV=production
- [ ] Test locally with ENV=dev

### Documentation
- [ ] Update README.md with ENV configuration
- [ ] Add CORS behavior table (production/staging/dev)
- [ ] Document private subnet list
- [ ] Add security warnings
- [ ] Create K8s ConfigMap examples (dev, staging, production)

### Deployment
- [ ] Deploy to staging with ENV=staging
- [ ] Verify VPN access from private subnets
- [ ] Verify production origins work
- [ ] Deploy to production with ENV=production
- [ ] Verify only production origins work
- [ ] Monitor logs for 24-48 hours
- [ ] Collect developer feedback

### Post-Deployment
- [ ] Update service documentation
- [ ] Share learnings with team
- [ ] Report any customizations needed
- [ ] Update this proposal if patterns changed

---

## Support & Resources

### Reference Implementation
- **Service**: SMAP Identity Service (`identity/`)
- **Files to Review**:
  - `config/config.go` - Environment configuration
  - `internal/middleware/cors.go` - CORS middleware with environment awareness
  - `internal/middleware/cors_test.go` - Comprehensive unit tests (70+ cases)
  - `deployment/k8s/configmap-*.yaml` - K8s configuration examples
  - `README.md` - ENV documentation

### Team Contacts
- **Technical Lead**: [Name/Email]
- **DevOps Team**: [Team Channel]
- **Security Team**: [Security Contact]

### Useful Links
- [CORS with Credentials - MDN](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [HttpOnly Cookie Security - OWASP](https://owasp.org/www-community/HttpOnly)
- [Go net Package](https://pkg.go.dev/net)

---

## Appendix: Code Examples

### Complete CORS Middleware Example (Go)

See `internal/middleware/cors.go` in SMAP Identity Service for full implementation.

### Kubernetes ConfigMap Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: service-config
  namespace: smap-production
data:
  ENV: "production"  # CRITICAL: Must be production in prod
  # ... other config
```

### Testing Example

```go
func TestProductionModeRejectsPrivateOrigins(t *testing.T) {
    config := DefaultCORSConfig("production")

    // Should have no dynamic function
    if config.AllowOriginFunc != nil {
        t.Error("Production mode should not have AllowOriginFunc")
    }

    // Should only have production origins
    if len(config.AllowedOrigins) != len(productionOrigins) {
        t.Errorf("Expected %d origins, got %d",
            len(productionOrigins), len(config.AllowedOrigins))
    }
}
```

---

## Conclusion

This standardized CORS approach provides a secure, maintainable, and developer-friendly solution for all SMAP microservices. By following this proposal:

1. **Developers gain flexibility** to work from anywhere via VPN
2. **DevOps reduces overhead** with zero manual IP configuration
3. **Security remains strong** with fail-safe defaults and production isolation
4. **Consistency improves** across all microservices

The SMAP Identity Service implementation serves as a battle-tested reference that can be replicated across the entire microservice ecosystem with minimal effort.

### Next Steps

1. Review and approve this proposal
2. Begin Priority 1 service migrations (Week 2)
3. Collect feedback and refine patterns
4. Complete rollout across all services (8 weeks)
5. Establish this as standard practice for new microservices

---

**Document Version History**:
- v1.0 (2025-11-28): Initial proposal based on SMAP Identity Service implementation
