package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"notification-srv/internal/alert"
	ws "notification-srv/internal/websocket"

	"github.com/gorilla/websocket"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

type testLogger struct{}

func (testLogger) Debug(context.Context, ...any)           {}
func (testLogger) Debugf(context.Context, string, ...any)  {}
func (testLogger) Info(context.Context, ...any)            {}
func (testLogger) Infof(context.Context, string, ...any)   {}
func (testLogger) Warn(context.Context, ...any)            {}
func (testLogger) Warnf(context.Context, string, ...any)   {}
func (testLogger) Error(context.Context, ...any)           {}
func (testLogger) Errorf(context.Context, string, ...any)  {}
func (testLogger) DPanic(context.Context, ...any)          {}
func (testLogger) DPanicf(context.Context, string, ...any) {}
func (testLogger) Panic(context.Context, ...any)           {}
func (testLogger) Panicf(context.Context, string, ...any)  {}
func (testLogger) Fatal(context.Context, ...any)           {}
func (testLogger) Fatalf(context.Context, string, ...any)  {}
func (l testLogger) WithTrace(context.Context) log.Logger  { return l }

type testAlertUC struct {
	err      error
	crisis   chan alert.CrisisAlertInput
	onboard  chan alert.DataOnboardingInput
	campaign chan alert.CampaignEventInput
}

func newTestAlertUC(err error) *testAlertUC {
	return &testAlertUC{
		err:      err,
		crisis:   make(chan alert.CrisisAlertInput, 1),
		onboard:  make(chan alert.DataOnboardingInput, 1),
		campaign: make(chan alert.CampaignEventInput, 1),
	}
}

func (a *testAlertUC) DispatchCrisisAlert(_ context.Context, input alert.CrisisAlertInput) error {
	a.crisis <- input
	return a.err
}

func (a *testAlertUC) DispatchDataOnboarding(_ context.Context, input alert.DataOnboardingInput) error {
	a.onboard <- input
	return a.err
}

func (a *testAlertUC) DispatchCampaignEvent(_ context.Context, input alert.CampaignEventInput) error {
	a.campaign <- input
	return a.err
}

func websocketPair(t *testing.T) (*websocket.Conn, *websocket.Conn, func()) {
	t.Helper()

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	serverConn := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		serverConn <- conn
	}))

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)

	var srv *websocket.Conn
	select {
	case srv = <-serverConn:
	case <-time.After(time.Second):
		t.Fatal("server websocket was not accepted")
	}

	return srv, client, func() {
		client.Close()
		srv.Close()
		server.Close()
	}
}

func TestNewRunShutdownAndStats(t *testing.T) {
	uc := New(testLogger{}, 5, newTestAlertUC(nil))
	impl, ok := uc.(*implUseCase)
	require.True(t, ok)
	require.Equal(t, 5, impl.maxConnections)

	go uc.Run()

	stats, err := uc.GetStats(context.Background())
	require.NoError(t, err)
	require.Equal(t, ws.HubStats{}, stats)
	require.NoError(t, uc.Shutdown(context.Background()))
	require.NoError(t, uc.OnUserConnected(context.Background(), "user-1"))
	require.NoError(t, uc.OnUserDisconnected(context.Background(), "user-1", false))
}

func TestRegister(t *testing.T) {
	tcs := map[string]struct {
		input   ws.ConnectionInput
		wantErr string
	}{
		"invalid connection type": {
			input:   ws.ConnectionInput{Conn: "bad"},
			wantErr: "invalid connection type",
		},
		"valid connection": {},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			alertUC := newTestAlertUC(nil)
			uc := New(testLogger{}, 5, alertUC)
			go uc.Run()

			if tc.wantErr == "" {
				serverConn, clientConn, cleanup := websocketPair(t)
				defer cleanup()
				tc.input = ws.ConnectionInput{UserID: "user-1", Conn: serverConn}
				defer clientConn.Close()
			}

			err := uc.Register(context.Background(), tc.input)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				stats, _ := uc.GetStats(context.Background())
				return stats.ActiveConnections == 1 && stats.TotalUniqueUsers == 1
			}, time.Second, 10*time.Millisecond)
			require.NoError(t, uc.Unregister(context.Background(), tc.input))
		})
	}
}

func TestParseChannel(t *testing.T) {
	tcs := map[string]struct {
		channel string
		want    ParsedChannel
		err     error
	}{
		"project": {
			channel: "project:project-1:user:user-1",
			want:    ParsedChannel{ChannelType: ws.ChannelTypeProject, EntityID: "project-1", UserID: "user-1"},
		},
		"campaign": {
			channel: "campaign:campaign-1:user:user-1",
			want:    ParsedChannel{ChannelType: ws.ChannelTypeCampaign, EntityID: "campaign-1", UserID: "user-1"},
		},
		"alert": {
			channel: "alert:crisis:user:user-1",
			want:    ParsedChannel{ChannelType: ws.ChannelTypeAlert, SubType: "crisis", UserID: "user-1"},
		},
		"system": {
			channel: "system:maintenance",
			want:    ParsedChannel{ChannelType: ws.ChannelTypeSystem, SubType: "maintenance"},
		},
		"too short":       {channel: "project", err: ws.ErrInvalidChannel},
		"bad project":     {channel: "project:project-1:member:user-1", err: ws.ErrInvalidChannel},
		"bad campaign":    {channel: "campaign:campaign-1:member:user-1", err: ws.ErrInvalidChannel},
		"bad alert":       {channel: "alert:crisis:member:user-1", err: ws.ErrInvalidChannel},
		"unknown channel": {channel: "unknown:value", err: ws.ErrInvalidChannel},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			got, err := parseChannel(tc.channel)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDetectMessageType(t *testing.T) {
	tcs := map[string]struct {
		payload string
		want    ws.MessageType
		err     error
	}{
		"analytics":       {payload: `{"source_id":"source-1","total_records":10}`, want: ws.MessageTypeAnalyticsPipeline},
		"data onboarding": {payload: `{"source_id":"source-1","record_count":10}`, want: ws.MessageTypeDataOnboarding},
		"crisis":          {payload: `{"alert_type":"spike"}`, want: ws.MessageTypeCrisisAlert},
		"campaign":        {payload: `{"campaign_id":"campaign-1"}`, want: ws.MessageTypeCampaignEvent},
		"system":          {payload: `{"system_event":"maintenance"}`, want: ws.MessageTypeSystem},
		"bad json":        {payload: `{`, err: errors.New("json")},
		"unknown":         {payload: `{"value":true}`, err: ws.ErrUnknownMessageType},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			got, err := detectMessageType([]byte(tc.payload))
			if tc.err != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestTransformMessage(t *testing.T) {
	uc := &implUseCase{}
	validPayloads := map[ws.MessageType]string{
		ws.MessageTypeDataOnboarding:    `{"project_id":"p","source_id":"s","source_name":"Facebook","source_type":"page","status":"completed","progress":100,"record_count":10,"error_count":0,"message":"done"}`,
		ws.MessageTypeAnalyticsPipeline: `{"project_id":"p","source_id":"s","total_records":10,"processed_count":10,"success_count":9,"failed_count":1,"progress":100,"current_phase":"done","estimated_time_ms":1}`,
		ws.MessageTypeCrisisAlert:       `{"project_id":"p","project_name":"Project","severity":"critical","alert_type":"spike","metric":"mentions","current_value":20,"threshold":10,"affected_aspects":["price"],"sample_mentions":["m"],"time_window":"1h","action_required":"act"}`,
		ws.MessageTypeCampaignEvent:     `{"campaign_id":"c","campaign_name":"Campaign","event_type":"started","resource_id":"r","resource_name":"Resource","resource_url":"https://example.test","message":"go"}`,
		ws.MessageTypeSystem:            `{"system_event":"maintenance"}`,
	}

	for msgType, payload := range validPayloads {
		t.Run(string(msgType)+" valid", func(t *testing.T) {
			got, err := uc.transformMessage(context.Background(), msgType, []byte(payload))
			require.NoError(t, err)
			require.Equal(t, msgType, got.Type)
			require.NotZero(t, got.Timestamp)
			require.NotNil(t, got.Payload)
		})
	}

	invalidPayloads := map[ws.MessageType]string{
		ws.MessageTypeDataOnboarding:    `{"source_id":1}`,
		ws.MessageTypeAnalyticsPipeline: `{"source_id":1}`,
		ws.MessageTypeCrisisAlert:       `{"project_id":1}`,
		ws.MessageTypeCampaignEvent:     `{"campaign_id":1}`,
		ws.MessageTypeSystem:            `{`,
	}
	for msgType, payload := range invalidPayloads {
		t.Run(string(msgType)+" invalid", func(t *testing.T) {
			_, err := uc.transformMessage(context.Background(), msgType, []byte(payload))
			require.ErrorIs(t, err, ws.ErrInvalidMessage)
		})
	}

	_, err := uc.transformMessage(context.Background(), "UNKNOWN", []byte(`{}`))
	require.ErrorIs(t, err, ws.ErrUnknownMessageType)
}

func TestHub(t *testing.T) {
	hub := newHub(testLogger{}, 10)
	go hub.run()

	client := &Connection{send: make(chan []byte, 2), userID: "user-1"}
	hub.register <- client
	require.Eventually(t, func() bool {
		active, unique := hub.Stats()
		return active == 1 && unique == 1
	}, time.Second, 10*time.Millisecond)

	hub.SendToUser("user-1", []byte("direct"))
	require.Equal(t, []byte("direct"), <-client.send)

	fullClient := &Connection{send: make(chan []byte), userID: "user-1"}
	hub.mu.Lock()
	hub.clients[fullClient] = true
	hub.users["user-1"][fullClient] = true
	hub.mu.Unlock()
	hub.SendToUser("user-1", []byte("skip-full"))
	require.Equal(t, []byte("skip-full"), <-client.send)

	hub.Broadcast([]byte("broadcast"))
	require.Equal(t, []byte("broadcast"), <-client.send)

	fullBroadcastClient := &Connection{send: make(chan []byte), userID: "user-2"}
	hub.mu.Lock()
	hub.clients[fullBroadcastClient] = true
	hub.users["user-2"] = map[*Connection]bool{fullBroadcastClient: true}
	hub.mu.Unlock()
	hub.Broadcast([]byte("drop-full"))

	hub.unregister <- client
}

func TestRouteMessage(t *testing.T) {
	uc := &implUseCase{hub: newHub(testLogger{}, 10)}
	userClient := &Connection{send: make(chan []byte, 1), userID: "user-1"}
	uc.hub.users["user-1"] = map[*Connection]bool{userClient: true}

	uc.routeMessage(ParsedChannel{UserID: "user-1"}, []byte("to-user"))
	require.Equal(t, []byte("to-user"), <-userClient.send)

	go uc.hub.run()
	broadcastClient := &Connection{send: make(chan []byte, 1), userID: "user-2"}
	uc.hub.register <- broadcastClient
	require.Eventually(t, func() bool {
		active, _ := uc.hub.Stats()
		return active == 1
	}, time.Second, 10*time.Millisecond)

	uc.routeMessage(ParsedChannel{ChannelType: ws.ChannelTypeSystem}, []byte("system"))
	require.Equal(t, []byte("system"), <-broadcastClient.send)
}

func TestProcessMessage(t *testing.T) {
	dispatchErr := errors.New("dispatch failed")
	tcs := map[string]struct {
		input      ws.ProcessMessageInput
		alertField string
		wantErr    string
	}{
		"invalid channel is swallowed": {
			input: ws.ProcessMessageInput{Channel: "bad", Payload: []byte(`{}`)},
		},
		"unknown type is swallowed": {
			input: ws.ProcessMessageInput{Channel: "project:p:user:user-1", Payload: []byte(`{"value":true}`)},
		},
		"transform error": {
			input:   ws.ProcessMessageInput{Channel: "project:p:user:user-1", Payload: []byte(`{"source_id":1,"record_count":10}`)},
			wantErr: "transform: invalid message format",
		},
		"data onboarding": {
			input:      ws.ProcessMessageInput{Channel: "project:p:user:user-1", Payload: []byte(`{"project_id":"p","source_id":"s","source_name":"Facebook","source_type":"page","status":"completed","progress":100,"record_count":10,"error_count":0,"message":"done"}`)},
			alertField: "onboard",
		},
		"crisis": {
			input:      ws.ProcessMessageInput{Channel: "alert:crisis:user:user-1", Payload: []byte(`{"project_id":"p","project_name":"Project","severity":"critical","alert_type":"spike","metric":"mentions","current_value":20,"threshold":10,"affected_aspects":["price"],"sample_mentions":["m"],"time_window":"1h","action_required":"act"}`)},
			alertField: "crisis",
		},
		"campaign": {
			input:      ws.ProcessMessageInput{Channel: "campaign:c:user:user-1", Payload: []byte(`{"campaign_id":"c","campaign_name":"Campaign","event_type":"started","resource_name":"Resource","resource_url":"https://example.test","message":"go"}`)},
			alertField: "campaign",
		},
		"analytics": {
			input: ws.ProcessMessageInput{Channel: "project:p:user:user-1", Payload: []byte(`{"project_id":"p","source_id":"s","total_records":10,"processed_count":10,"success_count":10,"failed_count":0,"progress":100,"current_phase":"done","estimated_time_ms":1}`)},
		},
		"system": {
			input: ws.ProcessMessageInput{Channel: "system:maintenance", Payload: []byte(`{"system_event":"maintenance"}`)},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			alertUC := newTestAlertUC(dispatchErr)
			uc := &implUseCase{hub: newHub(testLogger{}, 10), logger: testLogger{}, alertUC: alertUC}
			go uc.hub.run()
			client := &Connection{send: make(chan []byte, 1), userID: "user-1"}
			uc.hub.register <- client

			err := uc.ProcessMessage(context.Background(), tc.input)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)

			if tc.wantErr == "" && tc.input.Channel != "bad" && !strings.Contains(string(tc.input.Payload), `"value"`) {
				select {
				case raw := <-client.send:
					var output ws.NotificationOutput
					require.NoError(t, json.Unmarshal(raw, &output))
				case <-time.After(time.Second):
					t.Fatal("message was not routed")
				}
			}

			switch tc.alertField {
			case "onboard":
				requireReceive(t, alertUC.onboard)
			case "crisis":
				requireReceive(t, alertUC.crisis)
			case "campaign":
				requireReceive(t, alertUC.campaign)
			}
		})
	}
}

func TestProcessMessageMarshalError(t *testing.T) {
	oldMarshal := marshalNotification
	marshalNotification = func(any) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}
	defer func() { marshalNotification = oldMarshal }()

	uc := &implUseCase{
		hub:     newHub(testLogger{}, 10),
		logger:  testLogger{},
		alertUC: newTestAlertUC(nil),
	}

	err := uc.ProcessMessage(context.Background(), ws.ProcessMessageInput{
		Channel: "project:p:user:user-1",
		Payload: []byte(`{"project_id":"p","source_id":"s","total_records":10,"processed_count":10,"success_count":10,"failed_count":0,"progress":100,"current_phase":"done","estimated_time_ms":1}`),
	})
	require.EqualError(t, err, "marshal output: marshal failed")
}

func requireReceive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(time.Second):
		t.Fatal("expected async dispatch")
		var zero T
		return zero
	}
}

func TestReadPump(t *testing.T) {
	serverConn, clientConn, cleanup := websocketPair(t)
	defer cleanup()

	hub := newHub(testLogger{}, 10)
	go hub.run()
	conn := &Connection{hub: hub, conn: serverConn, send: make(chan []byte, 1), userID: "user-1"}
	hub.register <- conn

	done := make(chan struct{})
	go func() {
		conn.readPump()
		close(done)
	}()

	err := clientConn.WriteControl(
		websocket.PongMessage,
		[]byte("pong"),
		time.Now().Add(time.Second),
	)
	require.NoError(t, err)

	err = clientConn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "policy"),
		time.Now().Add(time.Second),
	)
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("readPump did not stop")
	}
}

func TestWritePump(t *testing.T) {
	tcs := map[string]struct {
		mode string
	}{
		"writes queued messages and close": {mode: "messages"},
		"next writer error":                {mode: "writer_error"},
		"ping error":                       {mode: "ping_error"},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			serverConn, clientConn, cleanup := websocketPair(t)
			defer cleanup()

			oldPingPeriod := pingPeriod
			pingPeriod = time.Millisecond
			defer func() { pingPeriod = oldPingPeriod }()

			conn := &Connection{conn: serverConn, send: make(chan []byte, 2), userID: "user-1"}
			done := make(chan struct{})

			switch tc.mode {
			case "messages":
				conn.send <- []byte("first")
				conn.send <- []byte("second")
				close(conn.send)
				go func() {
					conn.writePump(testLogger{})
					close(done)
				}()
				_, msg, err := clientConn.ReadMessage()
				require.NoError(t, err)
				require.Equal(t, []byte("firstsecond"), msg)
			case "writer_error":
				oldNextWriter := nextWebsocketWriter
				nextWebsocketWriter = func(*websocket.Conn, int) (io.WriteCloser, error) {
					return nil, errors.New("writer failed")
				}
				defer func() { nextWebsocketWriter = oldNextWriter }()
				conn.send <- []byte("message")
				go func() {
					conn.writePump(testLogger{})
					close(done)
				}()
			case "ping_error":
				require.NoError(t, clientConn.Close())
				go func() {
					conn.writePump(testLogger{})
					close(done)
				}()
			}

			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("writePump did not stop")
			}
		})
	}
}

func TestWritePumpCloseWriterError(t *testing.T) {
	serverConn, clientConn, cleanup := websocketPair(t)
	defer cleanup()

	oldAfterWriteMessage := afterWriteMessage
	afterWriteMessage = func() {
		_ = serverConn.NetConn().Close()
	}
	defer func() { afterWriteMessage = oldAfterWriteMessage }()

	conn := &Connection{conn: serverConn, send: make(chan []byte, 1), userID: "user-1"}
	conn.send <- []byte("message")

	done := make(chan struct{})
	go func() {
		conn.writePump(testLogger{})
		close(done)
	}()
	_ = clientConn.Close()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("writePump did not stop")
	}
}
