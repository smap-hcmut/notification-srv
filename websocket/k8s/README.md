# Kubernetes Ingress Configuration

## Tại sao có 2 Ingress files?

### `ingress.yaml` - API Services

Dành cho Identity và Project services với:

- Rewrite rules: `/identity/...` → `/...`
- Standard HTTP/HTTPS
- Timeouts ngắn hơn

### `ingress-websocket.yaml` - WebSocket Service

Dành riêng cho WebSocket với:

- **KHÔNG có rewrite rules** (preserve original path)
- WebSocket-specific timeouts (3600s)
- Direct pass-through để tránh protocol downgrade

## Vấn đề với Shared Ingress

Khi WebSocket và API services dùng chung 1 Ingress:

```yaml
# ❌ Có vấn đề
nginx.ingress.kubernetes.io/rewrite-target: /$2
paths:
  - path: /identity(/|$)(.*) # OK - cần rewrite
  - path: /ws(/|$)(.*) # PROBLEM - rewrite gây conflict
```

**Vấn đề:**

- Rewrite rules có thể làm mất WebSocket upgrade headers
- Protocol có thể bị downgrade từ WSS → WS
- Path rewriting conflict với WebSocket handshake

## Giải pháp: Separate Ingress

```yaml
# ✅ Ingress 1: API services (có rewrite)
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: smap-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$2
spec:
  rules:
    - host: smap-api.tantai.dev
      paths:
        - path: /identity(/|$)(.*)
        - path: /project(/|$)(.*)

---
# ✅ Ingress 2: WebSocket (KHÔNG rewrite)
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: smap-websocket-ingress
  annotations:
    nginx.ingress.kubernetes.io/websocket-services: "smap-websocket"
    # NO rewrite annotation
spec:
  rules:
    - host: smap-api.tantai.dev
      paths:
        - path: /ws
          pathType: Prefix
```

## Deployment

### Apply cả 2 Ingress

```bash
# Apply API services ingress
kubectl apply -f websocket/k8s/ingress.yaml

# Apply WebSocket ingress
kubectl apply -f websocket/k8s/ingress-websocket.yaml
```

### Verify

```bash
# Check cả 2 ingress
kubectl get ingress -n smap

# Expected output:
# NAME                      CLASS    HOSTS                   ADDRESS
# smap-ingress              <none>   smap-api.tantai.dev     10.x.x.x
# smap-websocket-ingress    <none>   smap-api.tantai.dev     10.x.x.x

# Describe để xem chi tiết
kubectl describe ingress smap-ingress -n smap
kubectl describe ingress smap-websocket-ingress -n smap
```

### Test

```bash
# Test API endpoint (có rewrite)
curl https://smap-api.tantai.dev/identity/health

# Test WebSocket (không rewrite)
wscat -c wss://smap-api.tantai.dev/ws
```

## Lợi ích của Separate Ingress

1. **Tránh conflict giữa rewrite rules và WebSocket**
2. **Dedicated configuration** cho từng service type
3. **Dễ debug** - logs và metrics riêng biệt
4. **Flexibility** - có thể tune riêng cho WebSocket
5. **Security** - không cần dùng configuration-snippet (bị disable)

## Troubleshooting

### Issue: Snippet directives disabled

```
Error: nginx.ingress.kubernetes.io/configuration-snippet annotation cannot be used
```

**Giải pháp:** Dùng separate ingress thay vì snippet

### Issue: WebSocket vẫn fallback về WS

**Check:**

1. Ingress có `websocket-services` annotation không?
2. Path có conflict với rewrite rules không?
3. Timeouts đủ lớn không?

```bash
# Debug ingress
kubectl describe ingress smap-websocket-ingress -n smap

# Check logs
kubectl logs -n ingress-nginx <ingress-controller-pod>
```

### Issue: 404 Not Found

**Nguyên nhân:** Path không match

**Check:**

```bash
# WebSocket ingress phải có
path: /ws
pathType: Prefix

# Không phải
path: /ws(/|$)(.*)  # ❌ Có rewrite pattern
```

## Notes

- Cả 2 ingress đều point đến cùng host: `smap-api.tantai.dev`
- NGINX Ingress Controller sẽ merge rules từ cả 2 ingress
- Priority: Specific paths (WebSocket `/ws`) được check trước generic patterns
- Không cần TLS config trong ingress nếu SSL termination ở external NGINX
