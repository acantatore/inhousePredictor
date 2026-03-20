package market

import (
	"encoding/json"
	"net/http"
	"time"

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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		Question         string    `json:"question"`
		Description      string    `json:"description"`
		Category         string    `json:"category"`
		ResolverID       string    `json:"resolver_id"`
		InitialLiquidity int64     `json:"initial_liquidity"`
		ClosesAt         time.Time `json:"closes_at"`
		ResolvesAt       time.Time `json:"resolves_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resolverID, err := uuid.Parse(req.ResolverID)
	if err != nil {
		http.Error(w, "invalid resolver_id", http.StatusBadRequest)
		return
	}
	m, err := h.svc.Create(r.Context(), CreateParams{
		Question:         req.Question,
		Description:      req.Description,
		Category:         Category(req.Category),
		CreatorID:        userID,
		ResolverID:       resolverID,
		InitialLiquidity: req.InitialLiquidity,
		ClosesAt:         req.ClosesAt,
		ResolvesAt:       req.ResolvesAt,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respond(w, http.StatusCreated, m)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	m, err := h.svc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	respond(w, http.StatusOK, m)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	category := Category(r.URL.Query().Get("category"))
	status := Status(r.URL.Query().Get("status"))
	markets, err := h.svc.List(r.Context(), category, status)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, markets)
}

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	marketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Outcome     string `json:"outcome"`
		EvidenceURL string `json:"evidence_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.svc.Resolve(r.Context(), marketID, userID, Outcome(req.Outcome), req.EvidenceURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "resolved"})
}

func (h *Handler) Dispute(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	marketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.svc.Dispute(r.Context(), marketID, userID, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "dispute filed"})
}

func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
