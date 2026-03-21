package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/naranjax/inhousepredictor/internal/auth"
	"github.com/naranjax/inhousepredictor/internal/httpx"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Handler upgrades HTTP to WebSocket and registers the client with the hub.
// Optional query param: ?market_id=<uuid> to subscribe to a single market.
func Handler(hub *Hub, secret string, allowedOrigins map[string]struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !originAllowed(r.Header.Get("Origin"), allowedOrigins) {
			httpx.WriteError(w, httpx.NewError(http.StatusForbidden, httpx.CodeForbidden, "Forbidden.", nil))
			return
		}
		token := auth.TokenFromRequest(r)
		if token == "" {
			httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
			return
		}
		if _, err := auth.ParseToken(token, secret); err != nil {
			httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", err))
			return
		}
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return originAllowed(r.Header.Get("Origin"), allowedOrigins)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		c := &client{
			send:     make(chan []byte, 256),
			marketID: r.URL.Query().Get("market_id"),
		}
		hub.subscribe <- c

		defer func() {
			hub.unsubscribe <- c
			conn.Close()
		}()

		// Write pump
		go func() {
			for data := range c.send {
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					return
				}
			}
		}()

		// Read pump — only purpose is detecting disconnects
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}
}

func originAllowed(origin string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return false
	}
	_, ok := allowed[origin]
	return ok
}
