package market

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
		return
	}
	var req struct {
		Question         string   `json:"question"`
		Description      string   `json:"description"`
		Category         string   `json:"category"`
		Options          []string `json:"options"`
		ResolverID       string   `json:"resolver_id"`
		InitialLiquidity int64    `json:"initial_liquidity"`
		ClosesAt         string   `json:"closes_at"`
		ResolvesAt       string   `json:"resolves_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	resolverID, err := uuid.Parse(req.ResolverID)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid resolver_id.", err))
		return
	}
	closesAt, err := time.Parse(time.RFC3339, req.ClosesAt)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid closes_at.", err))
		return
	}
	resolvesAt, err := time.Parse(time.RFC3339, req.ResolvesAt)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid resolves_at.", err))
		return
	}
	m, err := h.svc.Create(r.Context(), CreateParams{
		Question:         req.Question,
		Description:      req.Description,
		Category:         Category(req.Category),
		Options:          req.Options,
		CreatorID:        userID,
		ResolverID:       resolverID,
		InitialLiquidity: req.InitialLiquidity,
		ClosesAt:         closesAt,
		ResolvesAt:       resolvesAt,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, m)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid id.", err))
		return
	}
	m, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusNotFound, httpx.CodeNotFound, "Not found.", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	category := Category(r.URL.Query().Get("category"))
	status := Status(r.URL.Query().Get("status"))
	markets, err := h.svc.List(r.Context(), category, status)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, httpx.CodeInternal, "Internal server error.", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, markets)
}

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
		return
	}
	marketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid id.", err))
		return
	}
	var req struct {
		Outcome     string  `json:"outcome,omitempty"`
		OptionID    *string `json:"option_id,omitempty"`
		EvidenceURL string  `json:"evidence_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	outcome := Outcome(req.Outcome)
	var optionID *uuid.UUID
	if req.OptionID != nil && *req.OptionID != "" {
		parsed := strings.TrimSpace(*req.OptionID)
		optionUUID, err := uuid.Parse(parsed)
		if err != nil {
			httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid option_id.", err))
			return
		}
		if marketItem, err := h.svc.Get(r.Context(), marketID); err == nil {
			optionID = &optionUUID
			for _, option := range marketItem.Options {
				if option.ID == optionUUID {
					if strings.EqualFold(option.Label, "YES") {
						outcome = OutcomeYes
					} else if strings.EqualFold(option.Label, "NO") {
						outcome = OutcomeNo
					} else {
						outcome = OutcomeCancelled
					}
					break
				}
			}
		}
	}
	if err := h.svc.Resolve(r.Context(), ResolveRequest{MarketID: marketID, ResolverID: userID, Outcome: outcome, WinningOptionID: optionID, EvidenceURL: req.EvidenceURL}); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

func (h *Handler) Dispute(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
		return
	}
	marketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid id.", err))
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	if err := h.svc.Dispute(r.Context(), marketID, userID, req.Reason); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "dispute filed"})
}

func (h *Handler) ReviewDispute(w http.ResponseWriter, r *http.Request) {
	adminID, ok := auth.UserIDFromContext(r.Context())
	if !ok || !auth.IsAdminFromContext(r.Context()) {
		httpx.WriteError(w, httpx.NewError(http.StatusForbidden, httpx.CodeForbidden, "Forbidden.", nil))
		return
	}
	marketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid id.", err))
		return
	}
	var req struct {
		Action      string  `json:"action"`
		Outcome     *string `json:"outcome,omitempty"`
		EvidenceURL *string `json:"evidence_url,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	action := DisputeAction(strings.TrimSpace(req.Action))
	var outcome *Outcome
	if req.Outcome != nil {
		parsed := Outcome(strings.TrimSpace(*req.Outcome))
		outcome = &parsed
	}
	if err := h.svc.ReviewDispute(r.Context(), marketID, adminID, action, outcome, req.EvidenceURL); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "dispute reviewed"})
}

func (h *Handler) ListDisputes(w http.ResponseWriter, r *http.Request) {
	if !auth.IsAdminFromContext(r.Context()) {
		httpx.WriteError(w, httpx.NewError(http.StatusForbidden, httpx.CodeForbidden, "Forbidden.", nil))
		return
	}
	disputes, err := h.svc.ListDisputes(r.Context())
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, httpx.CodeInternal, "Internal server error.", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, disputes)
}
