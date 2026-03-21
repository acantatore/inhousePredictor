package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("http request", "method", r.Method, "path", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}

func allowedOrigins() map[string]struct{} {
	allowed := map[string]struct{}{}
	for _, origin := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		allowed["http://localhost:3000"] = struct{}{}
		allowed["http://127.0.0.1:3000"] = struct{}{}
		allowed["http://localhost:5173"] = struct{}{}
		allowed["http://127.0.0.1:5173"] = struct{}{}
		allowed["http://localhost:4173"] = struct{}{}
		allowed["http://127.0.0.1:4173"] = struct{}{}
		allowed["http://localhost:8080"] = struct{}{}
		allowed["http://127.0.0.1:8080"] = struct{}{}
	}
	return allowed
}

func bootstrapAdmin(ctx context.Context, db *pgxpool.Pool, email string) error {
	if strings.TrimSpace(email) == "" {
		return nil
	}
	var adminCount int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE is_admin = true`).Scan(&adminCount); err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	if adminCount > 0 {
		return nil
	}
	tag, err := db.Exec(ctx, `UPDATE users SET is_admin = true WHERE email = $1`, strings.TrimSpace(email))
	if err != nil {
		return fmt.Errorf("promote bootstrap admin: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("bootstrap admin email %q not found", email)
	}
	return nil
}
