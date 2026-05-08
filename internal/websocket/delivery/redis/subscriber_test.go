package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"notification-srv/internal/websocket"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

type redisTestLogger struct {
	errorLogs int
}

func (redisTestLogger) Debug(context.Context, ...any)          {}
func (redisTestLogger) Debugf(context.Context, string, ...any) {}
func (redisTestLogger) Info(context.Context, ...any)           {}
func (redisTestLogger) Infof(context.Context, string, ...any)  {}
func (redisTestLogger) Warn(context.Context, ...any)           {}
func (redisTestLogger) Warnf(context.Context, string, ...any)  {}
func (l *redisTestLogger) Error(context.Context, ...any)       { l.errorLogs++ }
func (l *redisTestLogger) Errorf(context.Context, string, ...any) {
	l.errorLogs++
}
func (redisTestLogger) DPanic(context.Context, ...any)          {}
func (redisTestLogger) DPanicf(context.Context, string, ...any) {}
func (redisTestLogger) Panic(context.Context, ...any)           {}
func (redisTestLogger) Panicf(context.Context, string, ...any)  {}
func (redisTestLogger) Fatal(context.Context, ...any)           {}
func (redisTestLogger) Fatalf(context.Context, string, ...any)  {}
func (l *redisTestLogger) WithTrace(context.Context) log.Logger { return l }

type fakeRedisClient struct {
	client *goredis.Client
}

func (f *fakeRedisClient) Set(context.Context, string, interface{}, time.Duration) error { return nil }
func (f *fakeRedisClient) Get(context.Context, string) (string, error)                   { return "", nil }
func (f *fakeRedisClient) Delete(context.Context, ...string) error                       { return nil }
func (f *fakeRedisClient) Exists(context.Context, string) (bool, error)                  { return false, nil }
func (f *fakeRedisClient) TTL(context.Context, string) (time.Duration, error)            { return 0, nil }
func (f *fakeRedisClient) Close() error                                                  { return f.client.Close() }
func (f *fakeRedisClient) Ping(ctx context.Context) error                                { return f.client.Ping(ctx).Err() }
func (f *fakeRedisClient) GetClient() *goredis.Client                                    { return f.client }

type fakeWebSocketUseCase struct {
	err    error
	inputs chan websocket.ProcessMessageInput
}

func newFakeWebSocketUseCase(err error) *fakeWebSocketUseCase {
	return &fakeWebSocketUseCase{err: err, inputs: make(chan websocket.ProcessMessageInput, 2)}
}

func (u *fakeWebSocketUseCase) Run()                           {}
func (u *fakeWebSocketUseCase) Shutdown(context.Context) error { return nil }
func (u *fakeWebSocketUseCase) Register(context.Context, websocket.ConnectionInput) error {
	return nil
}
func (u *fakeWebSocketUseCase) Unregister(context.Context, websocket.ConnectionInput) error {
	return nil
}
func (u *fakeWebSocketUseCase) GetStats(context.Context) (websocket.HubStats, error) {
	return websocket.HubStats{}, nil
}
func (u *fakeWebSocketUseCase) ProcessMessage(_ context.Context, input websocket.ProcessMessageInput) error {
	u.inputs <- input
	return u.err
}
func (u *fakeWebSocketUseCase) OnUserConnected(context.Context, string) error { return nil }
func (u *fakeWebSocketUseCase) OnUserDisconnected(context.Context, string, bool) error {
	return nil
}

func newRedisClient(t *testing.T) (*miniredis.Miniredis, *goredis.Client) {
	t.Helper()
	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		client.Close()
		server.Close()
	})
	return server, client
}

func TestNew(t *testing.T) {
	logger := &redisTestLogger{}
	uc := newFakeWebSocketUseCase(nil)
	redisClient := &fakeRedisClient{}

	got := New(redisClient, uc, logger)
	sub, ok := got.(*subscriber)
	require.True(t, ok)
	require.Equal(t, redisClient, sub.redis)
	require.Equal(t, uc, sub.uc)
	require.Equal(t, logger, sub.logger)
	require.NotNil(t, sub.quit)
}

func TestStartListenAndShutdown(t *testing.T) {
	_, client := newRedisClient(t)
	uc := newFakeWebSocketUseCase(nil)
	sub := New(&fakeRedisClient{client: client}, uc, &redisTestLogger{}).(*subscriber)

	require.NoError(t, sub.Start())
	require.NoError(t, client.Publish(context.Background(), "project:project-1:user:user-1", "payload").Err())

	select {
	case got := <-uc.inputs:
		require.Equal(t, "project:project-1:user:user-1", got.Channel)
		require.Equal(t, []byte("payload"), got.Payload)
	case <-time.After(time.Second):
		t.Fatal("message was not processed")
	}

	require.NoError(t, sub.Shutdown(context.Background()))
}

func TestStartSubscribeError(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 10 * time.Millisecond,
		ReadTimeout: 10 * time.Millisecond,
	})
	defer client.Close()

	sub := New(&fakeRedisClient{client: client}, newFakeWebSocketUseCase(nil), &redisTestLogger{}).(*subscriber)
	err := sub.Start()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to subscribe")
}

func TestListenUnexpectedClose(t *testing.T) {
	_, client := newRedisClient(t)
	logger := &redisTestLogger{}
	sub := New(&fakeRedisClient{client: client}, newFakeWebSocketUseCase(nil), logger).(*subscriber)

	require.NoError(t, sub.Start())
	require.NoError(t, sub.pubsub.Close())

	require.Eventually(t, func() bool {
		return logger.errorLogs > 0
	}, time.Second, 10*time.Millisecond)
}

func TestShutdownWithoutPubSub(t *testing.T) {
	sub := New(&fakeRedisClient{}, newFakeWebSocketUseCase(nil), &redisTestLogger{}).(*subscriber)
	require.NoError(t, sub.Shutdown(context.Background()))
}

func TestShutdownPubSubCloseError(t *testing.T) {
	expectedErr := errors.New("close failed")
	oldClosePubSub := closePubSub
	closePubSub = func(*goredis.PubSub) error {
		return expectedErr
	}
	defer func() { closePubSub = oldClosePubSub }()

	logger := &redisTestLogger{}
	sub := New(&fakeRedisClient{}, newFakeWebSocketUseCase(nil), logger).(*subscriber)
	sub.pubsub = &goredis.PubSub{}

	require.NoError(t, sub.Shutdown(context.Background()))
	require.Equal(t, 1, logger.errorLogs)
}

func TestHandleMessage(t *testing.T) {
	expectedErr := errors.New("process failed")
	logger := &redisTestLogger{}
	uc := newFakeWebSocketUseCase(expectedErr)
	sub := New(&fakeRedisClient{}, uc, logger).(*subscriber)

	sub.handleMessage(context.Background(), &goredis.Message{
		Channel: "system:maintenance",
		Payload: "payload",
	})

	require.Equal(t, 1, logger.errorLogs)
	require.Equal(t, websocket.ProcessMessageInput{
		Channel: "system:maintenance",
		Payload: []byte("payload"),
	}, <-uc.inputs)
}
