# Manual Testing Guide: HttpOnly Cookie Authentication

This document provides step-by-step manual testing procedures for the HttpOnly cookie authentication implementation.

---

## Prerequisites

- WebSocket service running locally or on test environment
- Identity service running with HttpOnly cookie support
- Valid test user credentials
- Browser with developer tools (Chrome/Firefox recommended)
- cURL or similar HTTP client

---

## Test 1: WebSocket Connection with Cookie Authentication ✅

### Objective
Verify that WebSocket connections work with HttpOnly cookie authentication.

### Steps

1. **Login via Identity Service**:
   ```bash
   curl -i -X POST http://localhost:8080/identity/authentication/login \
     -H "Content-Type: application/json" \
     -d '{
       "email": "test@example.com",
       "password": "password123"
     }' \
     -c cookies.txt
   ```

2. **Verify Cookie is Set**:
   ```bash
   cat cookies.txt | grep smap_auth_token
   ```
   
   **Expected**: Cookie `smap_auth_token` is present with JWT value

3. **Connect to WebSocket with Cookie**:
   
   **Using Browser Console** (recommended):
   ```javascript
   // Open browser console on http://localhost:3000 (or allowed origin)
   const ws = new WebSocket('ws://localhost:8081/ws');
   
   ws.onopen = () => {
     console.log('✅ Connected with cookie authentication!');
   };
   
   ws.onmessage = (event) => {
     console.log('📨 Received:', JSON.parse(event.data));
   };
   
   ws.onerror = (error) => {
     console.error('❌ Error:', error);
   };
   
   ws.onclose = () => {
     console.log('🔌 Connection closed');
   };
   ```

4. **Verify Connection Established**:
   - Check console for "Connected with cookie authentication!" message
   - Check WebSocket service logs for successful connection
   - Verify no "deprecated query parameter" warning in logs

### Expected Results
- ✅ WebSocket connection established successfully
- ✅ No token in URL
- ✅ Cookie sent automatically by browser
- ✅ No authentication errors in logs

### Success Criteria
- [ ] Connection succeeds without query parameter
- [ ] Cookie is sent with WebSocket upgrade request
- [ ] User is authenticated correctly
- [ ] No errors in browser console or server logs

---

## Test 2: Backward Compatibility with Query Parameter Authentication ✅

### Objective
Verify that legacy query parameter authentication still works during migration period.

### Steps

1. **Get JWT Token**:
   ```bash
   # Extract token from login response or use existing token
   TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
   ```

2. **Connect with Query Parameter**:
   ```javascript
   const token = 'your-jwt-token-here';
   const ws = new WebSocket(`ws://localhost:8081/ws?token=${token}`);
   
   ws.onopen = () => {
     console.log('✅ Connected with query parameter (deprecated)');
   };
   ```

3. **Check Server Logs**:
   ```bash
   # Look for deprecation warning
   tail -f logs/websocket.log | grep "deprecated query parameter"
   ```

### Expected Results
- ✅ Connection succeeds with query parameter
- ✅ Warning logged: "WebSocket connection using deprecated query parameter authentication"
- ✅ Functionality identical to cookie authentication

### Success Criteria
- [ ] Query parameter authentication works
- [ ] Deprecation warning appears in logs
- [ ] No functional differences from cookie auth

---

## Test 3: CORS with Credentials from Allowed Origins ✅

### Objective
Verify CORS configuration allows credentials from trusted origins and blocks others.

### Test 3.1: Allowed Origin (localhost:3000)

1. **Serve Frontend on Allowed Origin**:
   ```bash
   # Start simple HTTP server on port 3000
   python3 -m http.server 3000
   ```

2. **Create Test HTML File** (`test-ws.html`):
   ```html
   <!DOCTYPE html>
   <html>
   <head><title>WebSocket Test</title></head>
   <body>
     <h1>WebSocket Cookie Auth Test</h1>
     <button onclick="connect()">Connect</button>
     <div id="status"></div>
     <script>
       function connect() {
         const ws = new WebSocket('ws://localhost:8081/ws');
         const status = document.getElementById('status');
         
         ws.onopen = () => {
           status.innerHTML = '✅ Connected from localhost:3000';
           status.style.color = 'green';
         };
         
         ws.onerror = (error) => {
           status.innerHTML = '❌ Connection failed: ' + error;
           status.style.color = 'red';
         };
       }
     </script>
   </body>
   </html>
   ```

3. **Open in Browser**: `http://localhost:3000/test-ws.html`

4. **Click Connect Button**

### Expected Results
- ✅ Connection succeeds
- ✅ No CORS errors in browser console
- ✅ Status shows "Connected from localhost:3000"

### Test 3.2: Disallowed Origin

1. **Serve Frontend on Different Port**:
   ```bash
   python3 -m http.server 8000
   ```

2. **Open**: `http://localhost:8000/test-ws.html`

3. **Click Connect Button**

### Expected Results
- ❌ Connection rejected
- ❌ CORS error in browser console
- ❌ WebSocket upgrade fails

### Success Criteria
- [ ] Allowed origins (localhost:3000, 127.0.0.1:3000) can connect
- [ ] Disallowed origins are rejected
- [ ] CORS errors appear for blocked origins

---

## Test 4: Verify Cookie is Sent Automatically by Browser 🔍

### Objective
Confirm that browsers automatically include cookies with WebSocket connections.

### Steps

1. **Login to Set Cookie** (from Test 1)

2. **Open Browser DevTools**:
   - Chrome: F12 → Network tab → WS filter
   - Firefox: F12 → Network tab → WS filter

3. **Connect to WebSocket**:
   ```javascript
   const ws = new WebSocket('ws://localhost:8081/ws');
   ```

4. **Inspect WebSocket Request Headers**:
   - Click on WebSocket connection in Network tab
   - Go to "Headers" section
   - Look for "Cookie" header

### Expected Results
- ✅ Cookie header present in request
- ✅ Cookie value: `smap_auth_token=<JWT>`
- ✅ Cookie sent automatically (not manually added)

### Success Criteria
- [ ] Cookie header visible in DevTools
- [ ] Cookie contains JWT token
- [ ] No manual cookie handling required in code

---

## Test 5: Authentication Failure Scenarios ❌

### Test 5.1: Missing Cookie and Query Parameter

**Steps**:
1. Clear all cookies
2. Attempt WebSocket connection without query parameter:
   ```javascript
   const ws = new WebSocket('ws://localhost:8081/ws');
   ```

**Expected**:
- ❌ Connection rejected with 401 Unauthorized
- ❌ Error message: "missing token parameter"
- ❌ `ws.onerror` triggered

### Test 5.2: Invalid/Expired Token in Cookie

**Steps**:
1. Manually set invalid cookie:
   ```javascript
   document.cookie = "smap_auth_token=invalid-token; path=/";
   const ws = new WebSocket('ws://localhost:8081/ws');
   ```

**Expected**:
- ❌ Connection rejected with 401 Unauthorized
- ❌ Error message: "invalid or expired token"
- ❌ Server logs: "WebSocket connection rejected: invalid token"

### Test 5.3: Expired JWT Token

**Steps**:
1. Use token that has expired (past `exp` claim)
2. Attempt connection

**Expected**:
- ❌ Connection rejected with 401 Unauthorized
- ❌ Error message: "invalid or expired token"

### Success Criteria
- [ ] Missing credentials return 401
- [ ] Invalid tokens return 401
- [ ] Expired tokens return 401
- [ ] Error messages are clear and helpful

---

## Test Summary Checklist

### Cookie Authentication
- [ ] Test 1: WebSocket connection with cookie succeeds
- [ ] Test 2: Query parameter fallback works (backward compatibility)
- [ ] Test 3: CORS allows credentials from allowed origins
- [ ] Test 3: CORS blocks disallowed origins

### Browser Behavior
- [ ] Test 4: Cookie sent automatically by browser
- [ ] Test 4: Cookie visible in DevTools Network tab

### Error Handling
- [ ] Test 5.1: Missing credentials rejected (401)
- [ ] Test 5.2: Invalid token rejected (401)
- [ ] Test 5.3: Expired token rejected (401)

---

## Notes for Testers

1. **Cookie Scope**: Cookies are domain-specific. Ensure frontend and WebSocket service share the same domain or use proper cookie domain configuration.

2. **HTTPS in Production**: In production, `COOKIE_SECURE=true` requires HTTPS. Use `ws://` for local testing, `wss://` for production.

3. **Browser Differences**: Test in multiple browsers (Chrome, Firefox, Safari) as cookie handling may vary slightly.

4. **Debugging Tips**:
   - Use browser DevTools Network tab to inspect WebSocket headers
   - Check server logs for authentication warnings/errors
   - Verify cookie is set correctly after login (Application tab → Cookies)

---

## Automated Testing (Future)

For automated testing, consider:
- Integration tests using Go's `httptest` package
- WebSocket client tests with cookie jar
- CORS validation tests
- Token expiration tests

See `document/httponly_cookie.md` for integration test examples.
