# Testing Guide: HttpOnly Cookie Authentication

This document provides testing procedures for the HttpOnly cookie authentication system.

---

## Table of Contents

1. [Manual Testing](#manual-testing)
2. [Integration Testing](#integration-testing)
3. [Frontend Testing](#frontend-testing)
4. [Troubleshooting Tests](#troubleshooting-tests)

---

## Manual Testing

### Prerequisites

- Identity service running on `http://localhost:8080` or `https://smap-api.tantai.dev`
- Valid test user credentials
- cURL or Postman installed

### 1. Test Login - Cookie is Set

**Test**: Verify that login sets the HttpOnly cookie correctly.

```bash
# Login and save cookies
curl -i -X POST http://localhost:8080/identity/authentication/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "remember": false
  }' \
  -c cookies.txt

# Expected Response Headers:
# HTTP/1.1 200 OK
# Set-Cookie: smap_auth_token=eyJhbGc...; Path=/identity; Domain=.smap.com; Max-Age=7200; HttpOnly; Secure; SameSite=Lax
```

**Verification Checklist**:
- [x] Status code is 200
- [x] `Set-Cookie` header is present
- [x] Cookie name is `smap_auth_token`
- [x] `HttpOnly` flag is set
- [x] `Secure` flag is set
- [x] `SameSite=Lax` is set
- [x] `Max-Age=7200` (2 hours) for normal login
- [x] Response body contains user info (without token field)

### 2. Test Remember Me - Extended Cookie Lifetime

```bash
# Login with remember=true
curl -i -X POST http://localhost:8080/identity/authentication/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "remember": true
  }' \
  -c cookies-remember.txt

# Check Max-Age in response
# Expected: Max-Age=2592000 (30 days)
```

**Verification**:
- [x] `Max-Age=2592000` (30 days) when remember=true

### 3. Test Authenticated Request with Cookie

**Test**: Verify that subsequent requests use the cookie automatically.

```bash
# Use the cookie from previous login
curl -X GET http://localhost:8080/identity/authentication/me \
  -b cookies.txt

# Expected Response:
# {
#   "code": 200,
#   "data": {
#     "id": "user-id",
#     "email": "test@example.com",
#     "full_name": "Test User",
#     "role": "USER"
#   }
# }
```

**Verification**:
- [x] Status code is 200
- [x] Response contains user information
- [x] No Authorization header needed

### 4. Test Logout - Cookie Expiration

```bash
# Logout using the cookie
curl -i -X POST http://localhost:8080/identity/authentication/logout \
  -b cookies.txt

# Expected Response Headers:
# HTTP/1.1 200 OK
# Set-Cookie: smap_auth_token=; Path=/identity; Domain=.smap.com; Max-Age=-1; HttpOnly; Secure
```

**Verification**:
- [x] Status code is 200
- [x] `Set-Cookie` header has `Max-Age=-1`
- [x] Cookie value is empty

### 5. Test Request After Logout

```bash
# Try to access /me after logout
curl -X GET http://localhost:8080/identity/authentication/me \
  -b cookies.txt

# Expected Response:
# {
#   "code": 401,
#   "message": "Unauthorized"
# }
```

**Verification**:
- [x] Status code is 401 (Unauthorized)

### 6. Test Backward Compatibility - Authorization Header

**Test**: Verify that Bearer token still works during migration period.

```bash
# First, extract token from login response (if still available)
# Or use a pre-existing token for testing

TOKEN="your-jwt-token-here"

curl -X GET http://localhost:8080/identity/authentication/me \
  -H "Authorization: Bearer $TOKEN"

# Expected: 200 OK with user data
```

**Verification**:
- [x] Bearer token authentication still works
- [x] This ensures backward compatibility during migration

---

## Integration Testing

### Test Suite Structure

Create integration tests in `/internal/authentication/delivery/http/integration_test.go`:

```go
package http_test

import (
    "testing"
    "net/http"
    "net/http/httptest"
)

func TestLoginSetsCookie(t *testing.T) {
    // Setup test server
    // Login request
    // Assert cookie is set with correct attributes
}

func TestAuthenticatedRequestWithCookie(t *testing.T) {
    // Login to get cookie
    // Make authenticated request with cookie
    // Assert request succeeds
}

func TestLogoutExpiresCookie(t *testing.T) {
    // Login to get cookie
    // Logout
    // Assert cookie is expired (MaxAge=-1)
}

func TestGetMeReturnsUserInfo(t *testing.T) {
    // Login to get cookie
    // Call /me endpoint
    // Assert user info returned
}

func TestFallbackToAuthorizationHeader(t *testing.T) {
    // Create valid JWT token
    // Make request with Authorization header (no cookie)
    // Assert request succeeds
}
```

### Running Integration Tests

```bash
# Run all tests
go test ./internal/authentication/delivery/http/... -v

# Run specific test
go test ./internal/authentication/delivery/http/... -v -run TestLoginSetsCookie

# Run with coverage
go test ./internal/authentication/delivery/http/... -cover
```

---

## Frontend Testing

### Setup Test Environment

```javascript
import axios from 'axios';

// Configure axios for cookie authentication
const api = axios.create({
  baseURL: 'http://localhost:8080/identity',
  withCredentials: true  // REQUIRED for cookies
});
```

### Test 1: Login Flow

```javascript
describe('Cookie Authentication', () => {
  it('should login and store cookie automatically', async () => {
    const response = await api.post('/authentication/login', {
      email: 'test@example.com',
      password: 'password123'
    });

    // Assertions
    expect(response.status).toBe(200);
    expect(response.data.data.user).toBeDefined();
    expect(response.data.data.token).toBeUndefined(); // No token in response

    // Cookie is stored automatically by browser
  });
});
```

### Test 2: Authenticated Request

```javascript
it('should make authenticated requests with cookie', async () => {
  // Login first
  await api.post('/authentication/login', {
    email: 'test@example.com',
    password: 'password123'
  });

  // Make authenticated request (cookie sent automatically)
  const response = await api.get('/authentication/me');

  expect(response.status).toBe(200);
  expect(response.data.data.id).toBeDefined();
  expect(response.data.data.email).toBe('test@example.com');
});
```

### Test 3: Logout Flow

```javascript
it('should logout and clear cookie', async () => {
  // Login
  await api.post('/authentication/login', {
    email: 'test@example.com',
    password: 'password123'
  });

  // Logout
  const logoutResponse = await api.post('/authentication/logout');
  expect(logoutResponse.status).toBe(200);

  // Try to access protected endpoint
  try {
    await api.get('/authentication/me');
    fail('Should have thrown 401');
  } catch (error) {
    expect(error.response.status).toBe(401);
  }
});
```

### Test 4: CORS Credentials

```javascript
it('should handle CORS correctly with credentials', async () => {
  // This test verifies CORS headers are set correctly
  const response = await api.options('/authentication/me');

  expect(response.headers['access-control-allow-credentials']).toBe('true');
  expect(response.headers['access-control-allow-origin']).toBeDefined();
});
```

---

## Troubleshooting Tests

### Issue: Cookie Not Being Set

**Symptom**: Login succeeds but no Set-Cookie header

**Possible Causes**:
1. Cookie configuration missing in handler
2. Response not properly setting cookie

**Debug**:
```bash
# Check response headers
curl -i http://localhost:8080/identity/authentication/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass"}' \
  | grep -i "set-cookie"
```

### Issue: 401 on Authenticated Requests

**Symptom**: Requests fail with 401 even with valid cookie

**Possible Causes**:
1. Cookie not being sent by client
2. Cookie name mismatch
3. JWT verification failing

**Debug**:
```bash
# Check if cookie is being sent
curl -v http://localhost:8080/identity/authentication/me \
  -b cookies.txt 2>&1 | grep "Cookie:"

# Should see: Cookie: smap_auth_token=<JWT>
```

### Issue: CORS Errors in Browser

**Symptom**: "Access-Control-Allow-Origin" errors in console

**Possible Causes**:
1. `withCredentials: true` not set in frontend
2. CORS middleware not configured correctly
3. Wildcard origin with credentials

**Debug**:
```javascript
// Check axios configuration
console.log(api.defaults.withCredentials); // Should be true

// Check CORS headers in response
fetch('http://localhost:8080/identity/authentication/me', {
  credentials: 'include'
})
.then(response => {
  console.log(response.headers.get('access-control-allow-credentials'));
  // Should be 'true'
});
```

### Issue: Cookie Expiry Issues

**Symptom**: Cookie expires too quickly or doesn't persist

**Debug**:
```bash
# Check cookie Max-Age
curl -i http://localhost:8080/identity/authentication/login \
  -d '{"email":"test@example.com","password":"pass","remember":true}' \
  | grep "Max-Age"

# Normal login: Max-Age=7200 (2 hours)
# Remember me: Max-Age=2592000 (30 days)
```

---

## Test Coverage Goals

### Minimum Coverage

- [x] Login sets cookie correctly
- [x] Cookie has correct attributes (HttpOnly, Secure, SameSite)
- [x] Authenticated requests work with cookie
- [x] Logout expires cookie
- [x] /me endpoint returns user info
- [x] Authorization header fallback works

### Extended Coverage

- [x] Remember Me extends cookie lifetime
- [x] Invalid cookies return 401
- [x] Expired tokens return 401
- [x] Cookie takes precedence over Authorization header
- [x] CORS credentials are handled correctly
- [x] Cookie domain and path are correct

---

## Continuous Integration

### GitHub Actions Example

```yaml
name: Cookie Auth Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run Integration Tests
        run: |
          go test ./internal/authentication/delivery/http/... -v
          go test ./internal/middleware/... -v
        env:
          JWT_SECRET: test-secret
          COOKIE_NAME: smap_auth_token
```

---

## Summary

### Manual Testing Checklist

- [ ] Login sets HttpOnly cookie
- [ ] Cookie has correct security attributes
- [ ] Authenticated requests work with cookie
- [ ] Logout expires cookie
- [ ] /me endpoint returns user info
- [ ] Authorization header fallback works (during migration)
- [ ] CORS credentials work correctly

### Automated Testing Checklist

- [ ] Unit tests for cookie-setting logic
- [ ] Integration tests for auth flow
- [ ] Middleware tests for cookie extraction
- [ ] Frontend E2E tests with cookie authentication
- [ ] CORS credential tests

### CI/CD Integration

- [ ] Tests run on every push
- [ ] Coverage reports generated
- [ ] Test failures block merges
- [ ] Integration tests with real database

---

**Note**: These tests should be run before removing the Authorization header fallback to ensure all clients have migrated successfully.