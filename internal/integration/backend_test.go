package integration_test

import (
	"context"
	"strings"
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

func TestUserListReturnsLightweightSearchResults(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	createUser(t, pool, "zoe@example.com", false)
	createUser(t, pool, "resolver.anna@example.com", true)
	createUser(t, pool, "alex@example.com", false)

	repo := user.NewRepository(pool)
	users, err := repo.List(context.Background(), "anna", 10)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "resolver.anna@example.com", users[0].Email)
	require.True(t, users[0].IsAdmin)

	allUsers, err := repo.List(context.Background(), "", 2)
	require.NoError(t, err)
	require.Len(t, allUsers, 2)
	require.LessOrEqual(t, len(allUsers), 2)
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
		setMarketClosesAt(t, pool, marketID, time.Now().Add(-1*time.Hour))
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

func TestMultiOptionMarketTradeAndSnapshots(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	creatorID := createUser(t, pool, "creator@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	traderID := createUser(t, pool, "trader@example.com", false)
	marketRepo := market.NewRepository(pool)
	svc := market.NewService(marketRepo, trade.NewPayoutService(pool))
	m, err := svc.Create(context.Background(), market.CreateParams{
		Question:         "Which region wins Q3 growth?",
		Description:      "Three-way market",
		Category:         market.CategoryFinancials,
		Options:          []string{"LatAm", "EMEA", "NA"},
		CreatorID:        creatorID,
		ResolverID:       resolverID,
		InitialLiquidity: 900,
		ClosesAt:         time.Now().Add(24 * time.Hour),
		ResolvesAt:       time.Now().Add(48 * time.Hour),
	})
	require.NoError(t, err)
	require.Len(t, m.Options, 3)
	tradeSvc := trade.NewService(trade.NewRepository(pool), nil)
	optionID := m.Options[2].ID
	tradeResult, err := tradeSvc.Execute(context.Background(), trade.TradeRequest{UserID: traderID, MarketID: m.ID, OptionID: &optionID, Cost: 300})
	require.NoError(t, err)
	require.Equal(t, "NA", tradeResult.OptionLabel)
	refetched, _, err := marketRepo.GetByID(context.Background(), m.ID)
	require.NoError(t, err)
	require.Len(t, refetched.Options, 3)
	var leader string
	var maxBps int
	for _, option := range refetched.Options {
		if option.ProbabilityBps > maxBps {
			maxBps = option.ProbabilityBps
			leader = option.Label
		}
	}
	require.Equal(t, "NA", leader)
	require.GreaterOrEqual(t, len(refetched.Snapshots), 2)
}

func TestPositionsIncludeResolvedOutcome(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	creatorID := createUser(t, pool, "creator@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	traderID := createUser(t, pool, "trader@example.com", false)
	marketID := createMarket(t, pool, creatorID, resolverID, time.Now().Add(1*time.Hour), time.Now().Add(2*time.Hour))
	require.NoError(t, makeTrade(t, pool, traderID, marketID, trade.SideYes, 200))

	_, err := pool.Exec(context.Background(), `
		UPDATE markets SET status = 'resolved', outcome = 'yes', evidence_url = 'https://example.com/evidence', resolved_at = NOW(), dispute_deadline = NOW() - INTERVAL '1 hour'
		WHERE id = $1
	`, marketID)
	require.NoError(t, err)

	require.NoError(t, trade.NewPayoutService(pool).FinalizeEligiblePayouts(context.Background()))

	positions, err := trade.NewRepository(pool).GetPositions(context.Background(), traderID)
	require.NoError(t, err)
	require.Len(t, positions, 1)
	require.NotNil(t, positions[0].MarketOutcome)
	require.Equal(t, "yes", *positions[0].MarketOutcome)
	require.Equal(t, "resolved", positions[0].MarketStatus)
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

func TestSellSharesInBinaryMarket(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	creatorID := createUser(t, pool, "creator@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	traderID := createUser(t, pool, "trader@example.com", false)
	marketID := createMarket(t, pool, creatorID, resolverID, time.Now().Add(1*time.Hour), time.Now().Add(2*time.Hour))

	// Get initial balance
	var initialBalance int64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT balance FROM users WHERE id = $1`, traderID).Scan(&initialBalance))

	// Buy YES shares with 200 points
	// Due to CPMM pricing, buying pushes the YES price up, so we get fewer shares than 200
	require.NoError(t, makeTrade(t, pool, traderID, marketID, trade.SideYes, 200))

	// Get balance after buy
	var balanceAfterBuy int64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT balance FROM users WHERE id = $1`, traderID).Scan(&balanceAfterBuy))
	require.Less(t, balanceAfterBuy, initialBalance, "Balance should decrease after buying")

	// Get position after buy
	var yesShares float64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT yes_shares FROM positions WHERE user_id = $1 AND market_id = $2`, traderID, marketID).Scan(&yesShares))
	require.Greater(t, yesShares, 0.0, "Should have YES shares")
	sharesBought := yesShares

	// Sell half of the shares
	// Due to CPMM slippage, selling gives back less than half the original cost
	// This is expected behavior - the market maker extracts value from price movement
	service := trade.NewService(trade.NewRepository(pool), nil)
	sharesToSell := int64(sharesBought / 2)
	_, err := service.Execute(context.Background(), trade.TradeRequest{UserID: traderID, MarketID: marketID, Side: trade.SideSellYes, Cost: sharesToSell})
	require.NoError(t, err)

	// Get balance after sell
	var balanceAfterSell int64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT balance FROM users WHERE id = $1`, traderID).Scan(&balanceAfterSell))
	require.Greater(t, balanceAfterSell, balanceAfterBuy, "Balance should increase after selling")

	// Get position after sell
	var yesSharesAfterSell float64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT yes_shares FROM positions WHERE user_id = $1 AND market_id = $2`, traderID, marketID).Scan(&yesSharesAfterSell))
	require.Less(t, yesSharesAfterSell, sharesBought, "Should have fewer shares after selling")
	require.InDelta(t, sharesBought-float64(sharesToSell), yesSharesAfterSell, 0.01, "Remaining shares should equal bought minus sold")

	t.Logf("Initial balance: %d, After buy: %d, After sell: %d", initialBalance, balanceAfterBuy, balanceAfterSell)
	t.Logf("Shares bought: %.2f, Sold: %d, Remaining: %.2f", sharesBought, sharesToSell, yesSharesAfterSell)
	t.Logf("Points spent: 200, Points received from sell: %d", balanceAfterSell-balanceAfterBuy)
}

func TestSellSharesInMultiOptionMarket(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	creatorID := createUser(t, pool, "creator@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	traderID := createUser(t, pool, "trader@example.com", false)

	marketRepo := market.NewRepository(pool)
	svc := market.NewService(marketRepo, trade.NewPayoutService(pool))
	m, err := svc.Create(context.Background(), market.CreateParams{
		Question:         "Which region wins?",
		Description:      "Three-way market",
		Category:         market.CategoryFinancials,
		Options:          []string{"LatAm", "EMEA", "NA"},
		CreatorID:        creatorID,
		ResolverID:       resolverID,
		InitialLiquidity: 900,
		ClosesAt:         time.Now().Add(24 * time.Hour),
		ResolvesAt:       time.Now().Add(48 * time.Hour),
	})
	require.NoError(t, err)
	require.Len(t, m.Options, 3)

	// Get initial balance
	var initialBalance int64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT balance FROM users WHERE id = $1`, traderID).Scan(&initialBalance))

	// Buy shares in LatAm option
	latAmOptionID := m.Options[0].ID
	tradeSvc := trade.NewService(trade.NewRepository(pool), nil)
	buyResult, err := tradeSvc.Execute(context.Background(), trade.TradeRequest{
		UserID:   traderID,
		MarketID: m.ID,
		OptionID: &latAmOptionID,
		Side:     trade.Side(strings.ToLower(m.Options[0].Label)),
		Cost:     300,
	})
	require.NoError(t, err)
	require.Equal(t, "LatAm", buyResult.OptionLabel)
	sharesBought := buyResult.Shares

	// Get balance after buy
	var balanceAfterBuy int64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT balance FROM users WHERE id = $1`, traderID).Scan(&balanceAfterBuy))
	require.Less(t, balanceAfterBuy, initialBalance)

	// Verify position was created
	var optionShares float64
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT shares FROM option_positions 
		WHERE user_id = $1 AND market_option_id = $2`, traderID, latAmOptionID).Scan(&optionShares))
	require.InDelta(t, sharesBought, optionShares, 0.01)

	// Sell half the shares back
	// For multi-option markets, SideSellYes/SideSellNo are used for any option sell
	// The option_id determines which option is being sold
	sharesToSell := int64(sharesBought / 2)
	sellResult, err := tradeSvc.Execute(context.Background(), trade.TradeRequest{
		UserID:   traderID,
		MarketID: m.ID,
		OptionID: &latAmOptionID,
		Side:     trade.SideSellYes,
		Cost:     sharesToSell,
	})
	require.NoError(t, err)
	require.Equal(t, "LatAm", sellResult.OptionLabel)
	require.Less(t, sellResult.Cost, int64(0), "Sell trade should have negative cost (proceeds credited separately)")

	// Get balance after sell
	var balanceAfterSell int64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT balance FROM users WHERE id = $1`, traderID).Scan(&balanceAfterSell))
	require.Greater(t, balanceAfterSell, balanceAfterBuy, "Balance should increase after selling")

	// Verify position was reduced
	var optionSharesAfterSell float64
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT shares FROM option_positions 
		WHERE user_id = $1 AND market_option_id = $2`, traderID, latAmOptionID).Scan(&optionSharesAfterSell))
	require.InDelta(t, sharesBought-float64(sharesToSell), optionSharesAfterSell, 0.01)

	t.Logf("Initial balance: %d, After buy: %d, After sell: %d", initialBalance, balanceAfterBuy, balanceAfterSell)
	t.Logf("Shares bought: %.2f, Sold: %d, Remaining: %.2f", sharesBought, sharesToSell, optionSharesAfterSell)
}

func TestSellFailsWithoutPosition(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	creatorID := createUser(t, pool, "creator@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	traderID := createUser(t, pool, "trader@example.com", false)
	marketID := createMarket(t, pool, creatorID, resolverID, time.Now().Add(1*time.Hour), time.Now().Add(2*time.Hour))

	service := trade.NewService(trade.NewRepository(pool), nil)
	_, err := service.Execute(context.Background(), trade.TradeRequest{
		UserID:   traderID,
		MarketID: marketID,
		Side:     trade.SideSellYes,
		Cost:     100,
	})

	require.Error(t, err)
	appErr := &httpx.Error{}
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, httpx.CodeInsufficientBalance, appErr.Code)
}

func TestSellFailsWithInsufficientShares(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	creatorID := createUser(t, pool, "creator@example.com", false)
	resolverID := createUser(t, pool, "resolver@example.com", false)
	traderID := createUser(t, pool, "trader@example.com", false)
	marketID := createMarket(t, pool, creatorID, resolverID, time.Now().Add(1*time.Hour), time.Now().Add(2*time.Hour))

	// Buy some YES shares
	require.NoError(t, makeTrade(t, pool, traderID, marketID, trade.SideYes, 200))

	// Get actual shares owned
	var yesShares float64
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT yes_shares FROM positions WHERE user_id = $1 AND market_id = $2`, traderID, marketID).Scan(&yesShares))

	// Try to sell more shares than owned
	service := trade.NewService(trade.NewRepository(pool), nil)
	_, err := service.Execute(context.Background(), trade.TradeRequest{
		UserID:   traderID,
		MarketID: marketID,
		Side:     trade.SideSellYes,
		Cost:     int64(yesShares + 100), // Try to sell more than owned
	})

	require.Error(t, err)
	appErr := &httpx.Error{}
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, httpx.CodeInsufficientBalance, appErr.Code)
}
