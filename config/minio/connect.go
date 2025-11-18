package minio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"smap-api/config"
	miniopkg "smap-api/pkg/minio"
)

const (
	// defaultConnectTimeout is the maximum time to wait for initial connection
	defaultConnectTimeout = 5 * time.Second
	// defaultMaxRetries is the default number of retry attempts
	defaultMaxRetries = 3
)

var (
	instance miniopkg.MinIO
	once     sync.Once
	mu       sync.RWMutex
	initErr  error // Stores the last initialization error to allow retry
)

// Connect initializes and connects to MinIO using singleton pattern.
// If connection fails, it can be retried by calling Connect() again.
// Returns the existing instance if already connected.
func Connect(ctx context.Context, cfg config.MinIOConfig) (miniopkg.MinIO, error) {
	mu.Lock()
	defer mu.Unlock()

	// Return existing instance if already connected
	if instance != nil {
		return instance, nil
	}

	// Reset sync.Once if previous initialization failed to allow retry
	if initErr != nil {
		once = sync.Once{}
		initErr = nil
	}

	var err error
	once.Do(func() {
		// Create context with timeout for connection attempt
		connectCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
		defer cancel()

		fmt.Printf("[MinIO] Attempting to connect to %s (SSL: %v, Region: %s)...\n",
			cfg.Endpoint, cfg.UseSSL, cfg.Region)

		// Create MinIO implementation using pkg/minio
		impl, implErr := miniopkg.NewMinIO(&cfg)
		if implErr != nil {
			err = fmt.Errorf("failed to create MinIO implementation: %w", implErr)
			initErr = err
			fmt.Printf("[MinIO] ERROR: Failed to create implementation: %v\n", err)
			return
		}

		fmt.Printf("[MinIO] Verifying connection...\n")

		// Verify connection by connecting
		if connectErr := impl.Connect(connectCtx); connectErr != nil {
			err = fmt.Errorf("failed to connect to MinIO: %w", connectErr)
			initErr = err
			fmt.Printf("[MinIO] ERROR: Failed to verify connection: %v\n", connectErr)
			return
		}

		instance = impl
		fmt.Printf("[MinIO] Successfully connected to %s\n", cfg.Endpoint)
	})

	return instance, err
}

// ConnectWithRetry initializes and connects to MinIO with retry logic.
func ConnectWithRetry(ctx context.Context, cfg config.MinIOConfig, maxRetries int) (miniopkg.MinIO, error) {
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		client, err := Connect(ctx, cfg)
		if err == nil {
			return client, nil
		}

		lastErr = err
		if i < maxRetries-1 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(i)) * time.Second
			fmt.Printf("[MinIO] Connection attempt %d/%d failed, retrying in %v...\n",
				i+1, maxRetries, backoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
	}

	return nil, fmt.Errorf("failed to connect after %d retries: %w", maxRetries, lastErr)
}

// GetClient returns the singleton MinIO client instance.
// Panics if the client has not been initialized by calling Connect() first.
func GetClient() miniopkg.MinIO {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("MinIO client not initialized. Call Connect() first")
	}
	return instance
}

// Disconnect closes the MinIO connection and resets the singleton instance.
// This allows a new connection to be established by calling Connect() again.
func Disconnect(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		fmt.Printf("[MinIO] Disconnecting...\n")
		if err := instance.Close(); err != nil {
			fmt.Printf("[MinIO] ERROR: Failed to close connection: %v\n", err)
			return fmt.Errorf("failed to close MinIO connection: %w", err)
		}

		instance = nil
		initErr = nil
		once = sync.Once{} // Reset to allow reconnection
		fmt.Printf("[MinIO] Disconnected successfully\n")
	}
	return nil
}

// HealthCheck performs a health check on the MinIO connection.
// Returns an error if the connection is not initialized or the health check fails.
func HealthCheck(ctx context.Context) error {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return fmt.Errorf("MinIO client not initialized")
	}

	return instance.HealthCheck(ctx)
}

// IsConnected checks if the MinIO client instance exists.
// Note: This only checks if the instance is initialized, not if the connection is actually alive.
// Use HealthCheck() to verify the connection is working.
func IsConnected() bool {
	mu.RLock()
	defer mu.RUnlock()

	return instance != nil
}

// Reconnect closes the existing MinIO connection and establishes a new one.
// This is useful when you need to reconnect after a connection loss or configuration change.
func Reconnect(ctx context.Context, cfg config.MinIOConfig) error {
	mu.Lock()
	defer mu.Unlock()

	// Close existing connection if present
	if instance != nil {
		_ = instance.Close()
		instance = nil
	}

	// Reset sync.Once and error state to allow new connection
	once = sync.Once{}
	initErr = nil

	fmt.Printf("[MinIO] Reconnecting to %s (SSL: %v, Region: %s)...\n",
		cfg.Endpoint, cfg.UseSSL, cfg.Region)

	// Create new MinIO implementation using pkg/minio
	impl, err := miniopkg.NewMinIO(&cfg)
	if err != nil {
		fmt.Printf("[MinIO] ERROR: Failed to create new implementation: %v\n", err)
		return fmt.Errorf("failed to create new MinIO connection: %w", err)
	}

	// Verify connection with timeout
	connectCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()

	if connectErr := impl.Connect(connectCtx); connectErr != nil {
		fmt.Printf("[MinIO] ERROR: Failed to reconnect: %v\n", connectErr)
		return fmt.Errorf("failed to connect to MinIO: %w", connectErr)
	}

	instance = impl
	fmt.Printf("[MinIO] Reconnected successfully to %s\n", cfg.Endpoint)

	return nil
}
