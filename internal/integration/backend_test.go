package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/naranjax/inhousepredictor/internal/httpx"
	"github.com/naranjax/inhousepredictor/internal/market"
	"github.com/naranjax/inhousepredictor/internal/testutil"
	"github.com/naranjax/inhousepredictor/internal/trade"
	"github.com/naranjax/inhousepredictor/internal/user"
)

func TestUserCreateMapsDuplicateEmailToConflict(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := user.NewRepository(pool)
	ctx := context.Background()

	first, err := repo.Create(ctx, "Jane", "jane@example.com", "hash")
	require.NoError(t, err)
	require.NotNil(t, first)

	_, err = repo.Create(ctx, "Jane Two", "jane@example.com", "hash")
	require.Error(t, err)
	appErr := &httpx.Error{}
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, httpx.CodeConflict, appErr.Code)
}

func TestMarketCreateDebitsCreatorAndSeedsBalancedPool(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	creatorID := createUser(t, pool, "creator@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)

	repo := market.NewRepository(pool)
	svc := market.NewService(repo, nil)
	ctx := context.Background()

	m, err := svc.Create(ctx, market.CreateParams{
		Question:         "Will launch happen this month?",
		Description:      "Test market",
		Category:         market.CategoryGeneral,
		CreatorID:        creatorID,
		ResolverID:       resolverID,
		InitialLiquidity: 250,
		ClosesAt:         time.Now().Add(2 * time.Hour),
		ResolvesAt:       time.Now().Add(4 * time.Hour),
	})
	require.NoError(t, err)
	require.NotNil(t, m)
	require.InDelta(t, 0.5, m.YesPrice, 1e-9)

	var balance int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT balance FROM users WHERE id = $1`, creatorID).Scan(&balance))
	require.Equal(t, int64(9750), balance)

	var yesReserve, noReserve float64
	require.NoError(t, pool.QueryRow(ctx, `SELECT yes_reserve, no_reserve FROM pools WHERE market_id = $1`, m.ID).Scan(&yesReserve, &noReserve))
	require.Equal(t, 250.0, yesReserve)
	require.Equal(t, 250.0, noReserve)
}

func TestTradeExecuteBlocksCreatorAndClosedMarkets(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	creatorID := createUser(t, pool, "creator@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	traderID := createUser(t, pool, "trader@example.com", false)
	marketID := createMarket(t, pool, creatorID, resolverID, time.Now().Add(-1*time.Hour), time.Now().Add(2*time.Hour))

	t.Run("creator cannot trade own market", func(t *testing.T) {
		setMarketClosesAt(t, pool, marketID, time.Now().Add(1*time.Hour))
		service := trade.NewService(trade.NewRepository(pool), nil)
		_, err := service.Execute(context.Background(), trade.TradeRequest{UserID: creatorID, MarketID: marketID, Side: trade.SideYes, Cost: 100})
		appErr := &httpx.Error{}
		require.ErrorAs(t, err, &appErr)
		require.Equal(t, httpx.CodeCreatorRestricted, appErr.Code)
	})

	t.Run("closed market rejects trade", func(t *testing.T) {
		service := trade.NewService(trade.NewRepository(pool), nil)
		_, err := service.Execute(context.Background(), trade.TradeRequest{UserID: traderID, MarketID: marketID, Side: trade.SideYes, Cost: 100})
		appErr := &httpx.Error{}
		require.ErrorAs(t, err, &appErr)
		require.Equal(t, httpx.CodeMarketClosed, appErr.Code)
	})
}

func TestPayoutIsIdempotent(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	creatorID := createUser(t, pool, "creator@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	traderID := createUser(t, pool, "trader@example.com", false)
	marketID := createMarket(t, pool, creatorID, resolverID, time.Now().Add(1*time.Hour), time.Now().Add(2*time.Hour))
	require.NoError(t, makeTrade(t, pool, traderID, marketID, trade.SideYes, 200))

	payoutSvc := trade.NewPayoutService(pool)
	require.NoError(t, payoutSvc.Payout(context.Background(), marketID, "yes"))

	var firstBalance int64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT balance FROM users WHERE id = $1`, traderID).Scan(&firstBalance))

	require.NoError(t, payoutSvc.Payout(context.Background(), marketID, "yes"))

	var secondBalance int64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT balance FROM users WHERE id = $1`, traderID).Scan(&secondBalance))
	require.Equal(t, firstBalance, secondBalance)
}

func TestDisputeDeadlineIsEnforced(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	creatorID := createUser(t, pool, "creator@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	userID := createUser(t, pool, "trader@example.com", false)
	marketID := createMarket(t, pool, creatorID, resolverID, time.Now().Add(1*time.Hour), time.Now().Add(2*time.Hour))

	_, err := pool.Exec(context.Background(), `
		UPDATE markets
		SET status = 'resolved', outcome = 'yes', evidence_url = 'https://example.com/evidence',
		    resolved_at = NOW() - INTERVAL '72 hours', dispute_deadline = NOW() - INTERVAL '24 hours'
		WHERE id = $1
	`, marketID)
	require.NoError(t, err)

	repo := market.NewRepository(pool)
	err = repo.Dispute(context.Background(), marketID, userID, "Too late")
	appErr := &httpx.Error{}
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, httpx.CodeDisputeClosed, appErr.Code)
}

func createUser(t *testing.T, pool *pgxpool.Pool, email string, isAdmin bool) uuid.UUID {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	var id uuid.UUID
	require.NoError(t, pool.QueryRow(context.Background(), `
		INSERT INTO users (name, email, password_hash, is_admin)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, email, email, string(hash), isAdmin).Scan(&id))
	return id
}

func createMarket(t *testing.T, pool *pgxpool.Pool, creatorID, resolverID uuid.UUID, closesAt, resolvesAt time.Time) uuid.UUID {
	t.Helper()
	var marketID uuid.UUID
	require.NoError(t, pool.QueryRow(context.Background(), `
		INSERT INTO markets (question, description, category, creator_id, resolver_id, initial_liquidity, closes_at, resolves_at)
		VALUES ('Will tests pass?', 'Integration test market', 'general', $1, $2, 200, $3, $4)
		RETURNING id
	`, creatorID, resolverID, closesAt, resolvesAt).Scan(&marketID))
	_, err := pool.Exec(context.Background(), `
		INSERT INTO pools (market_id, yes_reserve, no_reserve, total_collateral)
		VALUES ($1, 200, 200, 200)
	`, marketID)
	require.NoError(t, err)
	return marketID
}

func setMarketClosesAt(t *testing.T, pool *pgxpool.Pool, marketID uuid.UUID, closesAt time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `UPDATE markets SET closes_at = $1 WHERE id = $2`, closesAt, marketID)
	require.NoError(t, err)
}

func makeTrade(t *testing.T, pool *pgxpool.Pool, userID, marketID uuid.UUID, side trade.Side, cost int64) error {
	t.Helper()
	service := trade.NewService(trade.NewRepository(pool), nil)
	_, err := service.Execute(context.Background(), trade.TradeRequest{UserID: userID, MarketID: marketID, Side: side, Cost: cost})
	return err
}
