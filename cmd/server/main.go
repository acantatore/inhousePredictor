package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/naranjax/inhousepredictor/internal/auth"
	"github.com/naranjax/inhousepredictor/internal/db"
	"github.com/naranjax/inhousepredictor/internal/market"
	"github.com/naranjax/inhousepredictor/internal/trade"
	"github.com/naranjax/inhousepredictor/internal/user"
	"github.com/naranjax/inhousepredictor/internal/ws"
)

func main() {
	ctx := context.Background()

	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set")
	}

	// Dependency wiring
	hub := ws.NewHub()
	go hub.Run()

	userRepo := user.NewRepository(pool)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc, jwtSecret)

	marketRepo := market.NewRepository(pool)
	marketSvc := market.NewService(marketRepo)
	marketHandler := market.NewHandler(marketSvc)

	tradeRepo := trade.NewRepository(pool)
	tradeSvc := trade.NewService(tradeRepo, hub)
	tradeHandler := trade.NewHandler(tradeSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Public
	r.Post("/auth/register", userHandler.Register)
	r.Post("/auth/login", userHandler.Login)
	r.Get("/ws", ws.Handler(hub))

	// Authenticated
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(jwtSecret))

		r.Get("/me", userHandler.Me)

		r.Get("/markets", marketHandler.List)
		r.Post("/markets", marketHandler.Create)
		r.Get("/markets/{id}", marketHandler.Get)
		r.Post("/markets/{id}/resolve", marketHandler.Resolve)
		r.Post("/markets/{id}/dispute", marketHandler.Dispute)

		r.Post("/markets/{id}/trade", tradeHandler.Trade)
		r.Get("/markets/{id}/trades", tradeHandler.MarketTrades)
		r.Get("/positions", tradeHandler.MyPositions)
	})

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
