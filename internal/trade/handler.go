package trade

import (
	"encoding/json"
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
		Side string `json:"side"` // "yes" or "no"
		Cost int64  `json:"cost"` // play money points to spend
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	if req.Side != "yes" && req.Side != "no" {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeValidation, "Side must be yes or no.", nil))
		return
	}
	t, err := h.svc.Execute(r.Context(), TradeRequest{
		UserID:   userID,
		MarketID: marketID,
		Side:     Side(req.Side),
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
