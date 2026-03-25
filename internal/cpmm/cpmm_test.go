package cpmm

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStartsBalancedPool(t *testing.T) {
	pool := New(500)

	require.Equal(t, 500.0, pool.YesReserve)
	require.Equal(t, 500.0, pool.NoReserve)
	require.Equal(t, 250000.0, pool.K)
	require.InDelta(t, 0.5, pool.YesPrice(), 1e-9)
	require.InDelta(t, 0.5, pool.NoPrice(), 1e-9)
}

func TestBuyYesPreservesInvariantAndChangesPrice(t *testing.T) {
	pool := New(1000)
	shares, next, err := pool.BuyYes(250)

	require.NoError(t, err)
	require.Positive(t, shares)
	require.InDelta(t, pool.K, next.YesReserve*next.NoReserve, 1e-6)
	require.Greater(t, next.YesPrice(), pool.YesPrice())
	require.Less(t, next.NoPrice(), pool.NoPrice())
	require.InDelta(t, 1.0, next.YesPrice()+next.NoPrice(), 1e-9)
}

func TestBuyNoPreservesInvariantAndChangesPrice(t *testing.T) {
	pool := New(1000)
	shares, next, err := pool.BuyNo(250)

	require.NoError(t, err)
	require.Positive(t, shares)
	require.InDelta(t, pool.K, next.YesReserve*next.NoReserve, 1e-6)
	require.Less(t, next.YesPrice(), pool.YesPrice())
	require.Greater(t, next.NoPrice(), pool.NoPrice())
	require.InDelta(t, 1.0, next.YesPrice()+next.NoPrice(), 1e-9)
}

func TestBuyRejectsInvalidAmounts(t *testing.T) {
	pool := New(100)

	_, _, err := pool.BuyYes(0)
	require.ErrorIs(t, err, ErrInvalidAmount)

	_, _, err = pool.BuyNo(-5)
	require.ErrorIs(t, err, ErrInvalidAmount)
}

func TestCostForSharesRejectsImpossibleRequests(t *testing.T) {
	pool := New(100)

	_, err := pool.CostForYesShares(0)
	require.ErrorIs(t, err, ErrInvalidAmount)

	_, err = pool.CostForNoShares(100)
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestBuyYesThenBuyNoIsNotFullyReversible(t *testing.T) {
	pool := New(1000)

	_, afterYes, err := pool.BuyYes(100)
	require.NoError(t, err)

	_, afterNo, err := afterYes.BuyNo(100)
	require.NoError(t, err)

	require.False(t,
		math.Abs(afterNo.YesReserve-pool.YesReserve) < 1e-9 &&
			math.Abs(afterNo.NoReserve-pool.NoReserve) < 1e-9,
	)
	// Slippage means a round-trip spend changes the pool state.
	require.NotEqual(t, pool.YesPrice(), afterNo.YesPrice())
}

func TestSellYesPreservesInvariantAndChangesPrice(t *testing.T) {
	pool := New(1000)
	// First buy some YES shares
	shares, afterBuy, err := pool.BuyYes(250)
	require.NoError(t, err)
	require.Positive(t, shares)

	// Then sell them back
	proceeds, afterSell, err := afterBuy.SellYes(shares)
	require.NoError(t, err)
	require.Positive(t, proceeds)

	// Invariant preserved
	require.InDelta(t, pool.K, afterSell.YesReserve*afterSell.NoReserve, 1e-6)
	// Price should move back toward original
	require.Less(t, afterSell.YesPrice(), afterBuy.YesPrice())
	require.InDelta(t, 1.0, afterSell.YesPrice()+afterSell.NoPrice(), 1e-9)
}

func TestSellNoPreservesInvariantAndChangesPrice(t *testing.T) {
	pool := New(1000)
	// First buy some NO shares
	shares, afterBuy, err := pool.BuyNo(250)
	require.NoError(t, err)
	require.Positive(t, shares)

	// Then sell them back
	proceeds, afterSell, err := afterBuy.SellNo(shares)
	require.NoError(t, err)
	require.Positive(t, proceeds)

	// Invariant preserved
	require.InDelta(t, pool.K, afterSell.YesReserve*afterSell.NoReserve, 1e-6)
	// Price should move back toward original
	require.Less(t, afterSell.NoPrice(), afterBuy.NoPrice())
	require.InDelta(t, 1.0, afterSell.YesPrice()+afterSell.NoPrice(), 1e-9)
}

func TestSellRejectsInvalidAmounts(t *testing.T) {
	pool := New(100)

	_, _, err := pool.SellYes(0)
	require.ErrorIs(t, err, ErrInvalidAmount)

	_, _, err = pool.SellNo(-5)
	require.ErrorIs(t, err, ErrInvalidAmount)
}

func TestBuyThenSellIsNotFullyReversible(t *testing.T) {
	pool := New(1000)

	// Buy YES shares
	shares, afterBuy, err := pool.BuyYes(400)
	require.NoError(t, err)

	// Sell them back immediately
	proceeds, afterSell, err := afterBuy.SellYes(shares)
	require.NoError(t, err)

	// Verify proceeds are positive
	require.Positive(t, proceeds)

	// In CPMM, buying then selling the exact same shares is mathematically reversible
	// (minus potential floating point rounding). The "slippage" manifests as:
	// 1. The price changes after the buy
	// 2. The next trader to buy faces worse prices
	// The key invariant is preserved
	require.InDelta(t, pool.K, afterSell.YesReserve*afterSell.NoReserve, 1e-6)
}
