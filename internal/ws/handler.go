package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: restrict to company domain in prod
		return true
	},
}

// Handler upgrades HTTP to WebSocket and registers the client with the hub.
// Optional query param: ?market_id=<uuid> to subscribe to a single market.
func Handler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
