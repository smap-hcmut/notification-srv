package websocket

import (
	"net/http"
	"testing"
)

func TestIsPrivateOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		// K8s subnet (172.16.21.0/24)
		{"K8s subnet IP - valid", "http://172.16.21.50:3000", true},
		{"K8s subnet IP - lower bound", "http://172.16.21.0:3000", true},
		{"K8s subnet IP - upper bound", "http://172.16.21.255:3000", true},
		{"K8s subnet IP - no port", "http://172.16.21.100", true},
		{"K8s subnet IP - HTTPS", "https://172.16.21.50:3000", true},
		{"K8s subnet IP - out of range", "http://172.16.20.50:3000", false},
		{"K8s subnet IP - out of range", "http://172.16.22.50:3000", false},

		// Private network 1 (172.16.19.0/24)
		{"Private network 1 - valid", "http://172.16.19.50:3000", true},
		{"Private network 1 - lower bound", "http://172.16.19.0:3000", true},
		{"Private network 1 - upper bound", "http://172.16.19.255:3000", true},
		{"Private network 1 - out of range", "http://172.16.18.50:3000", false},
		{"Private network 1 - out of range", "http://172.16.20.50:3000", false},

		// Private network 2 (192.168.1.0/24)
		{"Private network 2 - valid", "http://192.168.1.50:3000", true},
		{"Private network 2 - lower bound", "http://192.168.1.0:3000", true},
		{"Private network 2 - upper bound", "http://192.168.1.255:3000", true},
		{"Private network 2 - out of range", "http://192.168.0.50:3000", false},
		{"Private network 2 - out of range", "http://192.168.2.50:3000", false},

		// Edge cases
		{"Public IP", "http://1.2.3.4:3000", false},
		{"Invalid origin URL", "not-a-url", false},
		{"Empty origin", "", false},
		{"Domain name (not IP)", "http://example.com:3000", false},
		{"IPv6 address", "http://[2001:db8::1]:3000", false},
		{"Localhost IP", "http://127.0.0.1:3000", false}, // Not in private subnets
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPrivateOrigin(tt.origin)
			if got != tt.want {
				t.Errorf("isPrivateOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestIsLocalhostOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		// localhost variants
		{"localhost HTTP - port 3000", "http://localhost:3000", true},
		{"localhost HTTP - port 8080", "http://localhost:8080", true},
		{"localhost HTTP - port 80", "http://localhost:80", true},
		{"localhost HTTP - no port", "http://localhost", true},
		{"localhost HTTPS - port 3000", "https://localhost:3000", true},
		{"localhost HTTPS - port 443", "https://localhost:443", true},
		{"localhost HTTPS - no port", "https://localhost", true},

		// 127.0.0.1 variants
		{"127.0.0.1 HTTP - port 3000", "http://127.0.0.1:3000", true},
		{"127.0.0.1 HTTP - port 8080", "http://127.0.0.1:8080", true},
		{"127.0.0.1 HTTP - port 80", "http://127.0.0.1:80", true},
		{"127.0.0.1 HTTP - no port", "http://127.0.0.1", true},
		{"127.0.0.1 HTTPS - port 3000", "https://127.0.0.1:3000", true},
		{"127.0.0.1 HTTPS - port 443", "https://127.0.0.1:443", true},
		{"127.0.0.1 HTTPS - no port", "https://127.0.0.1", true},

		// Negative cases
		{"Production domain", "https://smap.tantai.dev", false},
		{"Private subnet IP", "http://172.16.21.50:3000", false},
		{"Public IP", "http://1.2.3.4:3000", false},
		{"Domain name", "http://example.com:3000", false},
		{"Empty origin", "", false},
		{"Invalid URL", "not-a-url", false},
		{"localhost in domain", "http://mylocalhost.com:3000", false},
		{"127.0.0.1 in domain", "http://127.0.0.1.example.com:3000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLocalhostOrigin(tt.origin)
			if got != tt.want {
				t.Errorf("isLocalhostOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestCreateUpgrader_ProductionMode(t *testing.T) {
	upgrader := createUpgrader("production")

	// Production origins should be allowed
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{"Production origin 1", "https://smap.tantai.dev", true},
		{"Production origin 2", "https://smap-api.tantai.dev", true},
		{"Localhost - should be rejected", "http://localhost:3000", false},
		{"127.0.0.1 - should be rejected", "http://127.0.0.1:3000", false},
		{"Private subnet - should be rejected", "http://172.16.21.50:3000", false},
		{"Invalid origin", "https://evil.com", false},
		{"Empty origin", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header: make(http.Header),
			}
			req.Header.Set("Origin", tt.origin)
			got := upgrader.CheckOrigin(req)
			if got != tt.want {
				t.Errorf("CheckOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestCreateUpgrader_DevMode(t *testing.T) {
	upgrader := createUpgrader("dev")

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		// Production origins should work
		{"Production origin 1", "https://smap.tantai.dev", true},
		{"Production origin 2", "https://smap-api.tantai.dev", true},

		// Localhost should work
		{"Localhost HTTP", "http://localhost:3000", true},
		{"Localhost HTTPS", "https://localhost:3000", true},
		{"127.0.0.1 HTTP", "http://127.0.0.1:3000", true},
		{"127.0.0.1 HTTPS", "https://127.0.0.1:3000", true},

		// Private subnets should work
		{"K8s subnet", "http://172.16.21.50:3000", true},
		{"Private network 1", "http://172.16.19.50:3000", true},
		{"Private network 2", "http://192.168.1.50:3000", true},

		// Invalid origins should be rejected
		{"Public IP", "http://1.2.3.4:3000", false},
		{"Invalid domain", "https://evil.com", false},
		{"Empty origin", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header: make(http.Header),
			}
			req.Header.Set("Origin", tt.origin)
			got := upgrader.CheckOrigin(req)
			if got != tt.want {
				t.Errorf("CheckOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestCreateUpgrader_StagingMode(t *testing.T) {
	upgrader := createUpgrader("staging")

	// Staging should behave like dev
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{"Production origin", "https://smap.tantai.dev", true},
		{"Localhost", "http://localhost:3000", true},
		{"Private subnet", "http://172.16.21.50:3000", true},
		{"Invalid origin", "https://evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header: make(http.Header),
			}
			req.Header.Set("Origin", tt.origin)
			got := upgrader.CheckOrigin(req)
			if got != tt.want {
				t.Errorf("CheckOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestCreateUpgrader_FailSafeDefault(t *testing.T) {
	// Empty environment should default to production
	upgrader := createUpgrader("")

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{"Production origin - should work", "https://smap.tantai.dev", true},
		{"Localhost - should be rejected", "http://localhost:3000", false},
		{"Private subnet - should be rejected", "http://172.16.21.50:3000", false},
		{"Invalid origin - should be rejected", "https://evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header: make(http.Header),
			}
			req.Header.Set("Origin", tt.origin)
			got := upgrader.CheckOrigin(req)
			if got != tt.want {
				t.Errorf("CheckOrigin(%q) = %v, want %v (empty ENV should default to production)", tt.origin, got, tt.want)
			}
		})
	}
}
