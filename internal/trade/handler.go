package trade

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/naranjax/inhousepredictor/internal/auth"
	"github.com/naranjax/inhousepredictor/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Trade(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
		return
	}
	marketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid market id.", err))
		return
	}
	var req struct {
		Side     string  `json:"side"`
		OptionID *string `json:"option_id,omitempty"`
		Cost     int64   `json:"cost"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	// Debug logging
	fmt.Printf("[DEBUG] Trade request - Side: %q, Cost: %d, OptionID: %v\n", req.Side, req.Cost, req.OptionID)
	// Validate side - for binary markets (no option_id), allow: yes, no, sell_yes, sell_no
	// For multi-option markets (with option_id), any side is valid
	if req.OptionID == nil {
		validSides := map[string]bool{"yes": true, "no": true, "sell_yes": true, "sell_no": true}
		if !validSides[req.Side] {
			httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeValidation, "Side must be yes, no, sell_yes, or sell_no.", nil))
			return
		}
	}
	var optionID *uuid.UUID
	if req.OptionID != nil && *req.OptionID != "" {
		parsed, err := uuid.Parse(*req.OptionID)
		if err != nil {
			httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid option id.", err))
			return
		}
		optionID = &parsed
	}
	t, err := h.svc.Execute(r.Context(), TradeRequest{
		UserID:   userID,
		MarketID: marketID,
		Side:     Side(req.Side),
		OptionID: optionID,
		Cost:     req.Cost,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, t)
}

func (h *Handler) MyPositions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
		return
	}
	positions, err := h.svc.GetPositions(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, httpx.CodeInternal, "Internal server error.", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, positions)
}

func (h *Handler) MarketTrades(w http.ResponseWriter, r *http.Request) {
	marketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid market id.", err))
		return
	}
	trades, err := h.svc.GetMarketTrades(r.Context(), marketID)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, httpx.CodeInternal, "Internal server error.", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, trades)
}
