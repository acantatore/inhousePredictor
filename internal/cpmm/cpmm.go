// Package cpmm implements a Constant Product Market Maker for binary prediction markets.
//
// Pool invariant: yes_reserve * no_reserve = K
// YES price = no_reserve / (yes_reserve + no_reserve)
//
// Initialization: yes_reserve = no_reserve = initial_liquidity → K = L²
// This gives a 50/50 starting price and keeps slippage proportional to trade size.
package cpmm

import (
	"errors"
	"math"
)

var (
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrInvalidAmount         = errors.New("amount must be positive")
)

type Pool struct {
	YesReserve float64
	NoReserve  float64
	K          float64 // invariant: YesReserve * NoReserve
}

// New initialises a 50/50 pool from a liquidity amount (in play money points).
// yes_reserve = no_reserve = L  →  K = L²
func New(liquidity int64) Pool {
	L := float64(liquidity)
	return Pool{YesReserve: L, NoReserve: L, K: L * L}
}

// YesPrice returns the implied probability of YES resolving (0–1).
func (p Pool) YesPrice() float64 {
	return p.NoReserve / (p.YesReserve + p.NoReserve)
}

// NoPrice returns the implied probability of NO resolving (0–1).
func (p Pool) NoPrice() float64 {
	return p.YesReserve / (p.YesReserve + p.NoReserve)
}

// BuyYes computes shares received and the next pool state for spending cost points on YES.
// Derivation: add cost to NO side, solve for new YES reserve via K invariant.
func (p Pool) BuyYes(cost int64) (shares float64, next Pool, err error) {
	if cost <= 0 {
		return 0, p, ErrInvalidAmount
	}
	c := float64(cost)
	newNo := p.NoReserve + c
	newYes := p.K / newNo
	shares = p.YesReserve - newYes
	if shares <= 0 {
		return 0, p, ErrInsufficientLiquidity
	}
	return shares, Pool{YesReserve: newYes, NoReserve: newNo, K: p.K}, nil
}

// BuyNo computes shares received and the next pool state for spending cost points on NO.
func (p Pool) BuyNo(cost int64) (shares float64, next Pool, err error) {
	if cost <= 0 {
		return 0, p, ErrInvalidAmount
	}
	c := float64(cost)
	newYes := p.YesReserve + c
	newNo := p.K / newYes
	shares = p.NoReserve - newNo
	if shares <= 0 {
		return 0, p, ErrInsufficientLiquidity
	}
	return shares, Pool{YesReserve: newYes, NoReserve: newNo, K: p.K}, nil
}

// CostForYesShares returns the cost (points) to buy an exact number of YES shares.
func (p Pool) CostForYesShares(shares float64) (int64, error) {
	if shares <= 0 {
		return 0, ErrInvalidAmount
	}
	if shares >= p.YesReserve {
		return 0, ErrInsufficientLiquidity
	}
	newYes := p.YesReserve - shares
	cost := p.K/newYes - p.NoReserve
	return int64(math.Ceil(cost)), nil
}

// CostForNoShares returns the cost (points) to buy an exact number of NO shares.
func (p Pool) CostForNoShares(shares float64) (int64, error) {
	if shares <= 0 {
		return 0, ErrInvalidAmount
	}
	if shares >= p.NoReserve {
		return 0, ErrInsufficientLiquidity
	}
	newNo := p.NoReserve - shares
	cost := p.K/newNo - p.YesReserve
	return int64(math.Ceil(cost)), nil
}

// SellYes computes proceeds and next pool state for selling YES shares back to the pool.
// User gives shares back to YES reserve, receives collateral from NO reserve.
func (p Pool) SellYes(shares float64) (proceeds int64, next Pool, err error) {
	if shares <= 0 {
		return 0, p, ErrInvalidAmount
	}
	// Adding shares back to YES reserve
	newYes := p.YesReserve + shares
	newNo := p.K / newYes
	// Calculate proceeds: what we remove from NO reserve
	proceedsFloat := p.NoReserve - newNo
	if proceedsFloat <= 0 {
		return 0, p, ErrInsufficientLiquidity
	}
	return int64(proceedsFloat), Pool{YesReserve: newYes, NoReserve: newNo, K: p.K}, nil
}

// SellNo computes proceeds and next pool state for selling NO shares back to the pool.
// User gives shares back to NO reserve, receives collateral from YES reserve.
func (p Pool) SellNo(shares float64) (proceeds int64, next Pool, err error) {
	if shares <= 0 {
		return 0, p, ErrInvalidAmount
	}
	// Adding shares back to NO reserve
	newNo := p.NoReserve + shares
	newYes := p.K / newNo
	// Calculate proceeds: what we remove from YES reserve
	proceedsFloat := p.YesReserve - newYes
	if proceedsFloat <= 0 {
		return 0, p, ErrInsufficientLiquidity
	}
	return int64(proceedsFloat), Pool{YesReserve: newYes, NoReserve: newNo, K: p.K}, nil
}
