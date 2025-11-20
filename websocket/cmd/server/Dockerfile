# Stage 1: Builder
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=1.0.0" \
    -o websocket-server \
    ./cmd/server

# Stage 2: Runtime (Distroless)
FROM gcr.io/distroless/static-debian12

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /build/websocket-server /

# Expose WebSocket port
EXPOSE 8081

# Run as non-root user
USER 65532:65532

# Run the binary
ENTRYPOINT ["/websocket-server"]
