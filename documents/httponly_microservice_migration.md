# Microservices Migration Guide: HttpOnly Cookie Authentication

> **Guide for applying HttpOnly cookie authentication to other SMAP microservices**

This document provides step-by-step instructions for migrating other microservices in the SMAP ecosystem to use the same HttpOnly cookie authentication implemented in the Identity service.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Architecture Overview](#architecture-overview)
3. [Step-by-Step Migration](#step-by-step-migration)
4. [Configuration](#configuration)
5. [Testing](#testing)
6. [Common Pitfalls](#common-pitfalls)
7. [Troubleshooting](#troubleshooting)

---

## Prerequisites

Before migrating a microservice, ensure you have:

- Completed the Identity service migration (reference implementation)
- Understanding of your service's current authentication flow
- Access to service configuration and environment variables
- Test environment for validation

---

## Architecture Overview

### Current Architecture (Bearer Token)

```
┌─────────────┐                    ┌──────────────────┐
│   Browser   │  Authorization:    │   Microservice   │
│             │  Bearer <token>    │   (Orders, etc)  │
│             │ ─────────────────► │                  │
└─────────────┘                    └──────────────────┘
```

### New Architecture (Cookie-Based)

```
┌─────────────┐                    ┌──────────────────┐
│   Browser   │  Cookie:           │   Microservice   │
│             │  smap_auth_token   │   (Orders, etc)  │
│             │ ─────────────────► │                  │
└─────────────┘                    └──────────────────┘
```

**Key Points**:
- All services share the **same JWT secret** (centralized authentication)
- All services use the **same cookie name** (`smap_auth_token`)
- All services use the **same cookie domain** (`.smap.com`)
- Token verification logic remains **identical**

---

## Step-by-Step Migration

Follow these steps to migrate any SMAP microservice:

### Step 1: Add Cookie Configuration to Config

**File**: `config/config.go`

```go
type Config struct {
    // ... existing fields ...

    // Authentication & Security Configuration
    JWT            JWTConfig
    Cookie         CookieConfig  // ← ADD THIS

    // ... other fields ...
}

// Add this struct (copy from Identity service)
type CookieConfig struct {
    Domain         string `env:"COOKIE_DOMAIN" envDefault:".smap.com"`
    Secure         bool   `env:"COOKIE_SECURE" envDefault:"true"`
    SameSite       string `env:"COOKIE_SAMESITE" envDefault:"Lax"`
    MaxAge         int    `env:"COOKIE_MAX_AGE" envDefault:"7200"`
    MaxAgeRemember int    `env:"COOKIE_MAX_AGE_REMEMBER" envDefault:"2592000"`
    Name           string `env:"COOKIE_NAME" envDefault:"smap_auth_token"`
}
```

**File**: `template.env`

```env
# Cookie Configuration (for HttpOnly authentication)
COOKIE_DOMAIN=.smap.com
COOKIE_SECURE=true
COOKIE_SAMESITE=Lax
COOKIE_MAX_AGE=7200
COOKIE_MAX_AGE_REMEMBER=2592000
COOKIE_NAME=smap_auth_token
```

---

### Step 2: Update Middleware to Read from Cookie

**File**: `internal/middleware/new.go`

```go
import (
    "smap-api/config"
    pkgLog "smap-api/pkg/log"
    pkgScope "smap-api/pkg/scope"
)

type Middleware struct {
    l            pkgLog.Logger
    jwtManager   pkgScope.Manager
    cookieConfig config.CookieConfig  // ← ADD THIS
}

func New(l pkgLog.Logger, jwtManager pkgScope.Manager, cookieConfig config.CookieConfig) Middleware {
    return Middleware{
        l:            l,
        jwtManager:   jwtManager,
        cookieConfig: cookieConfig,  // ← ADD THIS
    }
}
```

**File**: `internal/middleware/middleware.go`

```go
func (m Middleware) Auth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ← CHANGE: Cookie-first authentication strategy
        // First, attempt to read token from cookie (preferred method)
        tokenString, err := c.Cookie(m.cookieConfig.Name)

        // Fallback to Authorization header for backward compatibility
        // TODO: Remove this fallback after all clients have migrated
        if err != nil || tokenString == "" {
            tokenString = strings.ReplaceAll(c.GetHeader("Authorization"), "Bearer ", "")
        }

        // ← REST REMAINS UNCHANGED
        if tokenString == "" {
            response.Unauthorized(c)
            c.Abort()
            return
        }

        payload, err := m.jwtManager.Verify(tokenString)
        if err != nil {
            response.Unauthorized(c)
            c.Abort()
            return
        }

        ctx := c.Request.Context()
        ctx = scope.SetPayloadToContext(ctx, payload)
        sc := scope.NewScope(payload)
        ctx = scope.SetScopeToContext(ctx, sc)
        c.Request = c.Request.WithContext(ctx)

        c.Next()
    }
}
```

---

### Step 3: Update CORS Configuration

**File**: `internal/middleware/cors.go`

```go
// Update DefaultCORSConfig function
func DefaultCORSConfig() CORSConfig {
    return CORSConfig{
        // ← CHANGE: Replace wildcard with specific origins
        AllowedOrigins: []string{
            "http://localhost",
            "http://127.0.0.1",
            "http://localhost:3000",
            "http://127.0.0.1:3000",
            "https://smap.tantai.dev",
            "https://smap-api.tantai.dev",
            "https://smap-api.tantai.dev/*",
        },
        AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
        AllowedHeaders: []string{
            "Origin",
            "Content-Type",
            "Content-Length",
            "Accept-Encoding",
            "X-CSRF-Token",
            "Authorization",
            "Accept",
            "X-Requested-With",
            "lang",
        },
        ExposedHeaders:   []string{"Content-Length"},
        AllowCredentials: true,  // ← CHANGE: Must be true for cookies
        MaxAge:           86400,
    }
}
```

---

### Step 4: Update HTTP Server Integration

**File**: `internal/httpserver/new.go`

```go
type HTTPServer struct {
    // ... existing fields ...
    cookieConfig config.CookieConfig  // ← ADD THIS
    // ... other fields ...
}

type Config struct {
    // ... existing fields ...
    CookieConfig config.CookieConfig  // ← ADD THIS
    // ... other fields ...
}

func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
    srv := &HTTPServer{
        // ... existing fields ...
        cookieConfig: cfg.CookieConfig,  // ← ADD THIS
        // ... other fields ...
    }
    // ... rest unchanged ...
}
```

**File**: `internal/httpserver/handler.go`

```go
func (srv HTTPServer) mapHandlers() error {
    scopeManager := scope.New(srv.jwtSecretKey)
    mw := middleware.New(srv.l, scopeManager, srv.cookieConfig)  // ← PASS COOKIE CONFIG

    // ... rest unchanged ...
}
```

**File**: `cmd/api/main.go`

```go
httpServer, err := httpserver.New(logger, httpserver.Config{
    // ... existing config ...
    CookieConfig: cfg.Cookie,  // ← ADD THIS
    // ... other config ...
})
```

---

### Step 5: Update Swagger Security Definitions

**File**: `cmd/api/main.go`

Add security definitions to Swagger annotations:

```go
// @title       Your Service Name API
// @description Your Service Description
// @version     1
// @host        your-service.tantai.dev
// @schemes     https
// @BasePath    /your-base-path
//
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name smap_auth_token
// @description Authentication token stored in HttpOnly cookie. Set automatically by Identity service /login endpoint.
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Legacy Bearer token authentication (deprecated - use cookie authentication instead). Format: "Bearer {token}"
func main() {
    // ... main code ...
}
```

Update all protected endpoints:

```go
// Change from:
// @Security Bearer

// To:
// @Security CookieAuth
```

---

### Step 6: Update Documentation

**File**: `README.md`

Add authentication section explaining:
1. Cookie-based authentication is required
2. How to configure HTTP clients (`withCredentials: true`)
3. Cookie configuration options
4. Migration instructions from Bearer token

**Example**:
```markdown
## Authentication

This service uses **HttpOnly cookie authentication** shared with the Identity service.

### Frontend Setup

```javascript
const api = axios.create({
  baseURL: 'https://your-service.tantai.dev',
  withCredentials: true  // Required for cookies
});
```

See the [Identity Service README](../identity/README.md) for complete authentication documentation.
```

---

## Configuration

### Environment Variables

All services must use identical cookie configuration:

```env
# Shared across all SMAP microservices
COOKIE_NAME=smap_auth_token
COOKIE_DOMAIN=.smap.com
COOKIE_SECURE=true
COOKIE_SAMESITE=Lax
COOKIE_MAX_AGE=7200
COOKIE_MAX_AGE_REMEMBER=2592000

# Service-specific JWT secret (must match Identity service)
JWT_SECRET=<same-secret-as-identity-service>
```

### Critical Configuration Rules

1. **JWT Secret**: MUST be identical across all services
2. **Cookie Name**: MUST be identical (`smap_auth_token`)
3. **Cookie Domain**: MUST be identical (`.smap.com`)
4. **CORS Origins**: MUST include all frontend domains

---

## Testing

### Manual Testing Checklist

#### 1. Test Cookie Reception
```bash
# Login via Identity service
curl -i -X POST https://smap-api.tantai.dev/identity/authentication/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  -c cookies.txt

# Verify Set-Cookie header is present
# Cookie should be: smap_auth_token=<JWT>; HttpOnly; Secure; SameSite=Lax
```

#### 2. Test Your Service with Cookie
```bash
# Call your service endpoint with the cookie
curl -X GET https://your-service.tantai.dev/your-endpoint \
  -b cookies.txt

# Should return authenticated response
```

#### 3. Test CORS with Credentials
```javascript
// In browser console on https://smap.tantai.dev
fetch('https://your-service.tantai.dev/your-endpoint', {
  credentials: 'include'
})
.then(r => r.json())
.then(console.log);

// Should succeed without CORS errors
```

#### 4. Test Backward Compatibility
```bash
# Get token from login response (if still available during migration)
TOKEN="<your-jwt-token>"

# Test with Authorization header
curl -X GET https://your-service.tantai.dev/your-endpoint \
  -H "Authorization: Bearer $TOKEN"

# Should still work during migration period
```

### Integration Test Example

```go
func TestCookieAuthentication(t *testing.T) {
    // Setup test server
    router := setupTestRouter()
    w := httptest.NewRecorder()

    // Login and capture cookie
    loginReq := httptest.NewRequest("POST", "/authentication/login", loginBody)
    router.ServeHTTP(w, loginReq)

    cookies := w.Result().Cookies()
    assert.NotEmpty(t, cookies)

    authCookie := findCookie(cookies, "smap_auth_token")
    assert.NotNil(t, authCookie)
    assert.True(t, authCookie.HttpOnly)

    // Test authenticated request with cookie
    req := httptest.NewRequest("GET", "/your-endpoint", nil)
    req.AddCookie(authCookie)

    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
}
```

---

## Common Pitfalls

### 1. CORS Misconfiguration

Wrong:
```go
AllowedOrigins: []string{"*"}
AllowCredentials: true  // Browser will block this!
```

Correct:
```go
AllowedOrigins: []string{"https://smap.tantai.dev", "http://localhost:3000"}
AllowCredentials: true
```

### 2. Cookie Domain Mismatch

Wrong: Different services use different cookie domains
- Service A: `COOKIE_DOMAIN=servicea.smap.com`
- Service B: `COOKIE_DOMAIN=serviceb.smap.com`

Correct: All services use the same parent domain
- All services: `COOKIE_DOMAIN=.smap.com`

### 3. JWT Secret Mismatch

Wrong: Each service has its own JWT secret

Correct: All services share the same JWT secret from Identity service

### 4. Missing `withCredentials` in Frontend

Wrong:
```javascript
axios.get('/api/endpoint')  // Cookie not sent!
```

Correct:
```javascript
const api = axios.create({
  baseURL: 'https://api.smap.com',
  withCredentials: true  // Cookies included
});
```

### 5. Cookie Scope Too Narrow

Wrong: `Path: /specific-endpoint` - Cookie only sent to one endpoint

Correct: `Path: /` or `Path: /service-base` - Cookie sent to all service endpoints

---

## Troubleshooting

### Issue: "401 Unauthorized" after migration

**Possible Causes**:
1. Frontend not sending credentials
2. CORS blocking cookie transmission
3. Cookie domain mismatch
4. JWT secret mismatch between services

**Solution**:
```javascript
// Check browser console for CORS errors
// Verify withCredentials is set
const api = axios.create({
  withCredentials: true  // Make sure this is present
});

// Check cookie is being sent (Network tab → Request Headers)
// Should see: Cookie: smap_auth_token=<JWT>
```

### Issue: CORS Error "Credentials flag is true, but Access-Control-Allow-Credentials is false"

**Cause**: CORS middleware not setting `AllowCredentials: true`

**Solution**:
```go
// internal/middleware/cors.go
func DefaultCORSConfig() CORSConfig {
    return CORSConfig{
        AllowCredentials: true,  // Must be true!
        // ...
    }
}
```

### Issue: Cookie not being set after login

**Possible Causes**:
1. Identity service not setting cookie properly
2. Secure flag enabled but testing on HTTP
3. Domain mismatch (frontend and API on different domains)

**Solution**:
```bash
# Check login response headers
curl -i -X POST https://smap-api.tantai.dev/identity/authentication/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass"}' \
  | grep Set-Cookie

# Should see: Set-Cookie: smap_auth_token=...; HttpOnly; Secure; SameSite=Lax
```

### Issue: Different services can't share authentication

**Cause**: Cookie configuration mismatch

**Solution**: Verify ALL services have identical configuration:
```bash
# Check each service's .env file
grep "COOKIE_" .env

# Should output (identical across all services):
COOKIE_NAME=smap_auth_token
COOKIE_DOMAIN=.smap.com
COOKIE_SECURE=true
COOKIE_SAMESITE=Lax
```

---

## Migration Checklist

Use this checklist when migrating each microservice:

### Pre-Migration
- [ ] Backup current service configuration
- [ ] Document current authentication flow
- [ ] Review Identity service migration as reference
- [ ] Set up test environment

### Code Changes
- [ ] Add `CookieConfig` to `config/config.go`
- [ ] Update middleware to accept cookie config
- [ ] Implement cookie-first token extraction
- [ ] Update CORS to enable credentials
- [ ] Pass cookie config through HTTP server layers
- [ ] Update Swagger security definitions
- [ ] Update all `@Security` annotations

### Configuration
- [ ] Add cookie env vars to `template.env`
- [ ] Verify JWT secret matches Identity service
- [ ] Configure allowed CORS origins
- [ ] Set correct cookie domain (`.smap.com`)

### Documentation
- [ ] Update README with authentication instructions
- [ ] Add frontend integration examples
- [ ] Document breaking changes
- [ ] Update API documentation

### Testing
- [ ] Test login flow (Identity service)
- [ ] Test cookie is received and stored
- [ ] Test authenticated requests with cookie
- [ ] Test CORS with credentials
- [ ] Test backward compatibility (Bearer token)
- [ ] Test cross-service authentication

### Deployment
- [ ] Deploy to test environment
- [ ] Validate with frontend team
- [ ] Monitor for authentication errors
- [ ] Document any issues encountered

### Post-Migration
- [ ] Monitor authentication metrics
- [ ] Collect feedback from frontend team
- [ ] Plan removal of Bearer token fallback
- [ ] Update monitoring/alerting rules

---

## Summary

The migration to HttpOnly cookie authentication involves:

1. Shared Configuration: All services use identical cookie and JWT settings
2. Middleware Update: Cookie-first token extraction with Bearer fallback
3. CORS Update: Enable credentials with specific origins
4. Integration: Wire cookie config through server layers
5. Testing: Verify authentication works across all services

**Key Principle**: Cookie authentication is centralized in Identity service. Other services only need to validate the shared JWT token from the cookie.

---

## Questions?

If you encounter issues during migration:

1. Reference the Identity service implementation
2. Check the [troubleshooting section](#troubleshooting)
3. Verify all configuration values match across services
4. Review [common pitfalls](#common-pitfalls)

The Identity service serves as the reference implementation for all migrations.