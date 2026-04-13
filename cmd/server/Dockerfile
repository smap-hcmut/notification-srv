# syntax=docker/dockerfile:1

# =============================================================================
# STAGE 1: BUILDER
# --platform=$BUILDPLATFORM: Use host arch for Go compiler (no QEMU emulation)
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH

# Install tzdata only (git not needed with vendor mode)
RUN apk add --no-cache tzdata

# Copy dependency files first (cached layer — rarely changes)
COPY go.mod go.sum ./
COPY vendor/ vendor/

# Copy source code
COPY . .

# Build binary with vendor mode (no network, fully reproducible)
# CGO_ENABLED=0: Static binary for distroless
# -ldflags="-s -w": Strip debug info (~30-40% smaller)
# -mod=vendor: Use vendored dependencies (no download needed)
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -mod=vendor \
    -ldflags="-s -w" \
    -trimpath \
    -o notification-srv ./cmd/server

# =============================================================================
# STAGE 2: RUNTIME (Distroless Static — ~2MB, no shell, minimal attack surface)
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Ho_Chi_Minh

COPY --from=builder --chown=nonroot:nonroot /app/notification-srv .

USER nonroot:nonroot

EXPOSE 8081

ENTRYPOINT ["./notification-srv"]