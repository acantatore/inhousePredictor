package trade

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/naranjax/inhousepredictor/internal/auth"
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
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	marketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid market id", http.StatusBadRequest)
		return
	}
	var req struct {
		Side string `json:"side"` // "yes" or "no"
		Cost int64  `json:"cost"` // play money points to spend
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Side != "yes" && req.Side != "no" {
		http.Error(w, "side must be 'yes' or 'no'", http.StatusBadRequest)
		return
	}
	t, err := h.svc.Execute(r.Context(), TradeRequest{
		UserID:   userID,
		MarketID: marketID,
		Side:     Side(req.Side),
		Cost:     req.Cost,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respond(w, http.StatusCreated, t)
}

func (h *Handler) MyPositions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	positions, err := h.svc.GetPositions(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, positions)
}

func (h *Handler) MarketTrades(w http.ResponseWriter, r *http.Request) {
	marketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid market id", http.StatusBadRequest)
		return
	}
	trades, err := h.svc.GetMarketTrades(r.Context(), marketID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, trades)
}

func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
