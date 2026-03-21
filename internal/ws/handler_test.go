package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/naranjax/inhousepredictor/internal/auth"
	"github.com/stretchr/testify/require"
)

func TestOriginAllowed(t *testing.T) {
	allowed := map[string]struct{}{"http://localhost:3000": {}}
	require.True(t, originAllowed("http://localhost:3000", allowed))
	require.False(t, originAllowed("http://evil.example.com", allowed))
	require.False(t, originAllowed("", allowed))
}

func TestWebsocketHandlerRejectsMissingAuth(t *testing.T) {
	hub := NewHub()
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	res := httptest.NewRecorder()

	Handler(hub, "secret", map[string]struct{}{"http://localhost:3000": {}})(res, req)
	require.Equal(t, 401, res.Code)
}

func TestWebsocketHandlerRejectsDisallowedOrigin(t *testing.T) {
	hub := NewHub()
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	req.Header.Set("Authorization", "Bearer test")
	res := httptest.NewRecorder()

	Handler(hub, "secret", map[string]struct{}{"http://localhost:3000": {}})(res, req)
	require.Equal(t, 403, res.Code)
}

func TestWebsocketHandlerAcceptsAuthenticatedConnectionAndBroadcasts(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	secret := "secret"
	token, err := auth.NewToken(uuid.New(), false, secret)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(Handler(hub, secret, map[string]struct{}{"http://localhost:3000": {}})))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?market_id=market-1"
	headers := http.Header{}
	headers.Set("Origin", "http://localhost:3000")
	headers.Set("Authorization", "Bearer "+token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))

	hub.Broadcast(Message{Type: MsgPriceUpdate, MarketID: "market-1", Payload: PriceUpdatePayload{YesPrice: 0.7, NoPrice: 0.3}})
	_, payload, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Contains(t, string(payload), `"market_id":"market-1"`)
	require.Contains(t, string(payload), `"yes_price":0.7`)
}
