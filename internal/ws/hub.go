// Package ws provides the WebSocket broadcast hub.
// Clients may subscribe to a specific market_id or to all markets (empty string).
package ws

import (
	"context"
	"encoding/json"
	"sync"
)

type MessageType string

const (
	MsgPriceUpdate  MessageType = "price_update"
	MsgMarketUpdate MessageType = "market_update"
)

type Message struct {
	Type     MessageType `json:"type"`
	MarketID string      `json:"market_id"`
	Payload  any         `json:"payload"`
}

type PriceUpdatePayload struct {
	YesPrice  float64      `json:"yes_price"`
	NoPrice   float64      `json:"no_price"`
	LastTrade *TradeUpdate `json:"last_trade,omitempty"`
}

type TradeUpdate struct {
	Side   string  `json:"side"`
	Shares float64 `json:"shares"`
	Cost   int64   `json:"cost"`
}

type client struct {
	send     chan []byte
	marketID string // empty = subscribe to all markets
}

type Hub struct {
	mu          sync.RWMutex
	clients     map[*client]struct{}
	subscribe   chan *client
	unsubscribe chan *client
	broadcast   chan Message
	shutdown    chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*client]struct{}),
		subscribe:   make(chan *client, 256),
		unsubscribe: make(chan *client, 256),
		broadcast:   make(chan Message, 1024),
		shutdown:    make(chan struct{}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.closeAll()
			return
		case <-h.shutdown:
			h.closeAll()
			return
		case c := <-h.subscribe:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()

		case c := <-h.unsubscribe:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			h.mu.RLock()
			for c := range h.clients {
				if c.marketID == "" || c.marketID == msg.MarketID {
					select {
					case c.send <- data:
					default:
						// slow client — drop rather than block
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(msg Message) {
	h.broadcast <- msg
}

func (h *Hub) Shutdown() {
	close(h.shutdown)
}

func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		close(c.send)
		delete(h.clients, c)
	}
}
