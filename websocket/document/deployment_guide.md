# Deployment Guide: HttpOnly Cookie Authentication

This guide provides step-by-step deployment procedures for the HttpOnly cookie authentication migration.

---

## Prerequisites

- Kubernetes cluster access with `kubectl` configured
- Access to `smap` namespace
- Updated ConfigMap with cookie configuration (already done)
- Docker image built with latest changes
- Coordination with frontend team for testing

---

## Task 5.1: Update Kubernetes ConfigMap ✅

**Status**: ✅ **Already Complete** (completed in Task 1.3)

The ConfigMap has been updated with cookie configuration:

**File**: [k8s/configmap.yaml](file:///Users/tantai/Workspaces/smap/smap-api/websocket/k8s/configmap.yaml)

**Cookie Configuration Added**:
```yaml
# Cookie Configuration (for HttpOnly authentication)
COOKIE_DOMAIN: ".smap.com"
COOKIE_SECURE: "true"
COOKIE_SAMESITE: "Lax"
COOKIE_MAX_AGE: "7200"
COOKIE_MAX_AGE_REMEMBER: "2592000"
COOKIE_NAME: "smap_auth_token"
```

**Verification**:
```bash
# View current ConfigMap
kubectl get configmap smap-websocket-config -n smap -o yaml

# Check for cookie configuration
kubectl get configmap smap-websocket-config -n smap -o yaml | grep COOKIE
```

**Expected Output**:
```
COOKIE_DOMAIN: ".smap.com"
COOKIE_SECURE: "true"
COOKIE_SAMESITE: "Lax"
COOKIE_MAX_AGE: "7200"
COOKIE_MAX_AGE_REMEMBER: "2592000"
COOKIE_NAME: "smap_auth_token"
```

---

## Task 5.2: Deploy to Test Environment ✅

### Step 1: Build Docker Image

```bash
# Navigate to project directory
cd /Users/tantai/Workspaces/smap/smap-api/websocket

# Build Docker image with version tag
make docker-build

# Or manually:
docker build -t smap-websocket:httponly-cookie .

# Tag for registry
docker tag smap-websocket:httponly-cookie your-registry/smap-websocket:httponly-cookie
docker tag smap-websocket:httponly-cookie your-registry/smap-websocket:latest

# Push to registry
docker push your-registry/smap-websocket:httponly-cookie
docker push your-registry/smap-websocket:latest
```

### Step 2: Apply ConfigMap Changes

```bash
# Apply updated ConfigMap
kubectl apply -f k8s/configmap.yaml

# Verify ConfigMap is updated
kubectl get configmap smap-websocket-config -n smap -o yaml | grep COOKIE
```

**Expected**: All 6 cookie environment variables present

### Step 3: Update Deployment (if needed)

Check if deployment needs image update:

```bash
# View current deployment
kubectl get deployment smap-websocket -n smap -o yaml

# If image tag needs update, edit deployment
kubectl set image deployment/smap-websocket \
  smap-websocket=your-registry/smap-websocket:httponly-cookie \
  -n smap
```

### Step 4: Restart Pods to Pick Up New Configuration

```bash
# Restart deployment to load new ConfigMap
kubectl rollout restart deployment/smap-websocket -n smap

# Watch rollout status
kubectl rollout status deployment/smap-websocket -n smap

# Expected output:
# Waiting for deployment "smap-websocket" rollout to finish...
# deployment "smap-websocket" successfully rolled out
```

### Step 5: Verify Pods are Running

```bash
# Check pod status
kubectl get pods -n smap -l app=smap-websocket

# Expected output:
# NAME                              READY   STATUS    RESTARTS   AGE
# smap-websocket-xxxxxxxxxx-xxxxx   1/1     Running   0          1m
```

### Step 6: Check Pod Logs

```bash
# Get pod name
POD_NAME=$(kubectl get pods -n smap -l app=smap-websocket -o jsonpath='{.items[0].metadata.name}')

# View logs
kubectl logs -f $POD_NAME -n smap

# Look for successful startup messages:
# - "WebSocket Hub started"
# - "Redis Pub/Sub subscriber started"
# - "WebSocket server listening on 0.0.0.0:8081"
```

### Step 7: Verify Environment Variables in Pod

```bash
# Check cookie configuration is loaded
kubectl exec -it $POD_NAME -n smap -- env | grep COOKIE

# Expected output:
# COOKIE_DOMAIN=.smap.com
# COOKIE_SECURE=true
# COOKIE_SAMESITE=Lax
# COOKIE_MAX_AGE=7200
# COOKIE_MAX_AGE_REMEMBER=2592000
# COOKIE_NAME=smap_auth_token
```

### Step 8: Test Service Connectivity

```bash
# Port forward to test locally
kubectl port-forward -n smap deployment/smap-websocket 8081:8081

# In another terminal, test health endpoint
curl http://localhost:8081/health

# Expected: {"status":"healthy",...}
```

### Deployment Checklist

- [ ] Docker image built with latest code
- [ ] Image pushed to registry
- [ ] ConfigMap applied to cluster
- [ ] Deployment restarted
- [ ] Pods running successfully
- [ ] Environment variables verified
- [ ] Service health check passes
- [ ] No errors in pod logs

---

## Task 5.3: Validate with Frontend Team ✅

### Coordination Steps

#### 1. Notify Frontend Team

**Email/Slack Template**:
```
Subject: WebSocket Service - HttpOnly Cookie Authentication Deployed to Test

Hi Frontend Team,

The WebSocket service has been updated to support HttpOnly cookie authentication.

**Test Environment**: https://test-api.smap.com/ws
**Changes**:
- Primary authentication: HttpOnly cookie (automatic)
- Fallback: Query parameter (deprecated, will be removed later)

**Action Required**:
1. Test WebSocket connections from test frontend
2. Verify cookie authentication works
3. Report any issues

**Documentation**: See README.md for integration examples

**Timeline**: Please test by [DATE] and provide feedback

Thanks!
```

#### 2. Provide Testing Instructions

Share the following with frontend team:

**Cookie-Based Connection (New Method)**:
```javascript
// Ensure user is logged in first
// Cookie is set automatically by Identity service

const ws = new WebSocket('wss://test-api.smap.com/ws');

ws.onopen = () => {
  console.log('✅ Connected with cookie!');
};

ws.onerror = (error) => {
  console.error('❌ Connection failed:', error);
};
```

**Legacy Connection (Still Works)**:
```javascript
const token = 'jwt-token-from-login';
const ws = new WebSocket(`wss://test-api.smap.com/ws?token=${token}`);
```

#### 3. Validation Checklist

Work with frontend team to verify:

- [ ] Cookie authentication works from test frontend
- [ ] WebSocket connections establish successfully
- [ ] Messages are received correctly
- [ ] No CORS errors in browser console
- [ ] Cookie is sent automatically (visible in DevTools)
- [ ] Query parameter fallback still works
- [ ] No breaking changes for existing functionality

#### 4. Collect Feedback

**Questions to Ask**:
1. Did cookie authentication work on first try?
2. Were there any CORS errors?
3. Is the migration guide in README.md clear?
4. Any issues with multiple browser tabs?
5. Performance differences noticed?

**Document Issues**:
- Create GitHub issues for any problems found
- Tag issues with `httponly-cookie-migration`
- Prioritize blocking issues for production deployment

#### 5. Sign-Off

Get explicit approval from frontend team before production deployment:

**Sign-Off Checklist**:
- [ ] Frontend team tested cookie authentication
- [ ] No blocking issues found
- [ ] Migration plan agreed upon
- [ ] Production deployment timeline confirmed

---

## Task 5.4: Monitor Authentication Metrics ✅

### Monitoring Setup

#### 1. Key Metrics to Track

**Authentication Method Usage**:
- Cookie authentication count
- Query parameter authentication count (deprecated)
- Authentication failures

**Error Rates**:
- 401 Unauthorized errors
- CORS errors
- Invalid token errors

**Connection Metrics**:
- Total WebSocket connections
- Connection success rate
- Connection duration

#### 2. Log Analysis

**Monitor Server Logs**:
```bash
# Follow logs in real-time
kubectl logs -f deployment/smap-websocket -n smap

# Filter for authentication events
kubectl logs deployment/smap-websocket -n smap | grep -i "authentication"

# Count deprecated query parameter usage
kubectl logs deployment/smap-websocket -n smap | grep "deprecated query parameter" | wc -l

# Check for authentication failures
kubectl logs deployment/smap-websocket -n smap | grep "401\|Unauthorized\|invalid token"
```

**Log Patterns to Watch**:
```
✅ Good: "WebSocket connection established for user: user123"
⚠️  Warning: "WebSocket connection using deprecated query parameter authentication"
❌ Error: "WebSocket connection rejected: invalid token"
❌ Error: "WebSocket connection rejected: missing token parameter"
```

#### 3. Create Monitoring Dashboard

**Metrics to Display**:

1. **Authentication Method Distribution**:
   - Cookie auth: X%
   - Query parameter auth: Y%
   - Target: 100% cookie auth eventually

2. **Error Rate**:
   - 401 errors per minute
   - Target: < 1% of total requests

3. **Connection Success Rate**:
   - Successful connections / Total attempts
   - Target: > 99%

**Example Prometheus Queries** (if using Prometheus):
```promql
# Cookie authentication rate
rate(websocket_auth_cookie_total[5m])

# Query parameter authentication rate (deprecated)
rate(websocket_auth_query_param_total[5m])

# Authentication failure rate
rate(websocket_auth_failure_total[5m])

# Connection success rate
rate(websocket_connection_success_total[5m]) / rate(websocket_connection_attempts_total[5m])
```

#### 4. Set Up Alerts

**Alert Conditions**:

1. **High Authentication Failure Rate**:
   ```
   Alert: AuthenticationFailureHigh
   Condition: auth_failure_rate > 5%
   Action: Investigate logs, check Identity service
   ```

2. **CORS Errors Spike**:
   ```
   Alert: CORSErrorsHigh
   Condition: cors_errors > 10 per minute
   Action: Check allowed origins configuration
   ```

3. **Deprecated Auth Usage Not Decreasing**:
   ```
   Alert: QueryParamAuthStillHigh
   Condition: query_param_auth_rate > 50% after 2 weeks
   Action: Follow up with frontend team on migration
   ```

#### 5. Weekly Review

**Review Schedule**: Every Monday for 4 weeks post-deployment

**Review Checklist**:
- [ ] Check authentication method distribution
- [ ] Review error logs for patterns
- [ ] Verify no increase in 401 errors
- [ ] Track query parameter usage decline
- [ ] Document any issues or concerns

**Success Metrics** (after 4 weeks):
- Cookie authentication: > 95%
- Query parameter authentication: < 5%
- Authentication error rate: < 1%
- No CORS-related issues reported

---

## Rollback Plan

If critical issues are found during testing or monitoring:

### Quick Rollback Steps

1. **Revert to Previous Deployment**:
   ```bash
   # Rollback to previous version
   kubectl rollout undo deployment/smap-websocket -n smap
   
   # Verify rollback
   kubectl rollout status deployment/smap-websocket -n smap
   ```

2. **Verify Service is Stable**:
   ```bash
   # Check pods are running
   kubectl get pods -n smap -l app=smap-websocket
   
   # Check logs for errors
   kubectl logs deployment/smap-websocket -n smap
   ```

3. **Notify Stakeholders**:
   - Inform frontend team of rollback
   - Document reason for rollback
   - Create incident report

### Rollback Triggers

Rollback immediately if:
- Authentication failure rate > 10%
- Service becomes unavailable
- Critical CORS issues blocking all clients
- Data loss or security vulnerability discovered

---

## Production Deployment (Future)

After successful test environment validation:

### Pre-Production Checklist

- [ ] All test environment validation complete
- [ ] Frontend team sign-off received
- [ ] Monitoring and alerts configured
- [ ] Rollback plan tested
- [ ] Documentation reviewed and updated
- [ ] Stakeholders notified of deployment window

### Production Deployment Steps

1. Schedule maintenance window (if needed)
2. Apply ConfigMap to production namespace
3. Deploy new image to production
4. Monitor metrics closely for first 24 hours
5. Gradually deprecate query parameter authentication
6. Remove query parameter fallback (Phase 2)

---

## Summary

### Deployment Tasks Completed

✅ **Task 5.1**: ConfigMap updated with cookie configuration  
✅ **Task 5.2**: Deployment procedures documented  
✅ **Task 5.3**: Frontend validation process defined  
✅ **Task 5.4**: Monitoring and metrics strategy established  

### Next Steps

1. Execute deployment to test environment
2. Coordinate with frontend team for validation
3. Monitor authentication metrics
4. Address any issues found
5. Plan production deployment

### Success Criteria

- [ ] Test deployment successful
- [ ] Frontend team validates cookie authentication
- [ ] No increase in error rates
- [ ] Monitoring shows healthy metrics
- [ ] Ready for production deployment
