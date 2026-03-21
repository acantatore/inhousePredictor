package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/naranjax/inhousepredictor/internal/auth"
	"github.com/naranjax/inhousepredictor/internal/db"
	"github.com/naranjax/inhousepredictor/internal/market"
	"github.com/naranjax/inhousepredictor/internal/migrate"
	"github.com/naranjax/inhousepredictor/internal/trade"
	"github.com/naranjax/inhousepredictor/internal/user"
	"github.com/naranjax/inhousepredictor/internal/ws"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	pool, err := db.Connect(ctx)
	if err != nil {
		logger.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := migrate.Run(ctx, pool); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Error("JWT_SECRET not set")
		os.Exit(1)
	}

	// Dependency wiring
	hub := ws.NewHub()
	go hub.Run(ctx)

	userRepo := user.NewRepository(pool)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc, jwtSecret)

	marketRepo := market.NewRepository(pool)
	payoutSvc := trade.NewPayoutService(pool)
	marketSvc := market.NewService(marketRepo, payoutSvc)
	marketHandler := market.NewHandler(marketSvc)

	tradeRepo := trade.NewRepository(pool)
	tradeSvc := trade.NewService(tradeRepo, hub)
	tradeHandler := trade.NewHandler(tradeSvc)

	r := chi.NewRouter()
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Public
	r.Post("/auth/register", userHandler.Register)
	r.Post("/auth/login", userHandler.Login)
	r.Get("/ws", ws.Handler(hub, jwtSecret, allowedOrigins()))

	// Authenticated
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(jwtSecret))

		r.Get("/me", userHandler.Me)
		r.Get("/users", userHandler.List)

		r.Get("/markets", marketHandler.List)
		r.With(auth.AdminOnly).Get("/admin/disputes", marketHandler.ListDisputes)
		r.Post("/markets", marketHandler.Create)
		r.Get("/markets/{id}", marketHandler.Get)
		r.Post("/markets/{id}/resolve", marketHandler.Resolve)
		r.Post("/markets/{id}/dispute", marketHandler.Dispute)
		r.With(auth.AdminOnly).Post("/markets/{id}/review-dispute", marketHandler.ReviewDispute)

		r.Post("/markets/{id}/trade", tradeHandler.Trade)
		r.Get("/markets/{id}/trades", tradeHandler.MarketTrades)
		r.Get("/positions", tradeHandler.MyPositions)
	})

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	if err := bootstrapAdmin(ctx, pool, os.Getenv("BOOTSTRAP_ADMIN_EMAIL")); err != nil {
		logger.Error("bootstrap admin failed", "error", err)
		os.Exit(1)
	}

	server := &http.Server{Addr: addr, Handler: r}
	go func() {
		logger.Info("server listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	hub.Shutdown()
}
