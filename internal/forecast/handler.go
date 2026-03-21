package forecast

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

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	status := QuestionStatus(r.URL.Query().Get("status"))
	questions, err := h.svc.ListQuestions(r.Context(), r.URL.Query().Get("program"), status)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, questions)
}

func (h *Handler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
		return
	}
	var req struct {
		Title          string   `json:"title"`
		Description    string   `json:"description"`
		Program        string   `json:"program"`
		ResolverID     string   `json:"resolver_id"`
		ResolutionRule string   `json:"resolution_rule"`
		ClosesAt       string   `json:"closes_at"`
		ResolvesAt     string   `json:"resolves_at"`
		ContributorIDs []string `json:"contributor_ids"`
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
	closesAt, err := parseRFC3339(req.ClosesAt)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid closes_at.", err))
		return
	}
	resolvesAt, err := parseRFC3339(req.ResolvesAt)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid resolves_at.", err))
		return
	}
	contributorIDs, err := parseUUIDs(req.ContributorIDs)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid contributor_ids.", err))
		return
	}
	question, err := h.svc.CreateQuestion(r.Context(), CreateQuestionParams{
		Title:          req.Title,
		Description:    req.Description,
		Program:        req.Program,
		OwnerID:        ownerID,
		ResolverID:     resolverID,
		ResolutionRule: req.ResolutionRule,
		ClosesAt:       closesAt,
		ResolvesAt:     resolvesAt,
		ContributorIDs: contributorIDs,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, question)
}

func (h *Handler) GetQuestion(w http.ResponseWriter, r *http.Request) {
	questionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid forecast question id.", err))
		return
	}
	question, err := h.svc.GetQuestion(r.Context(), questionID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, question)
}

func (h *Handler) SubmitForecast(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
		return
	}
	questionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid forecast question id.", err))
		return
	}
	var req struct {
		ProbabilityBps int    `json:"probability_bps"`
		Rationale      string `json:"rationale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	question, err := h.svc.SubmitForecast(r.Context(), SubmitForecastParams{QuestionID: questionID, UserID: userID, ProbabilityBps: req.ProbabilityBps, Rationale: req.Rationale})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, question)
}

func (h *Handler) ResolveQuestion(w http.ResponseWriter, r *http.Request) {
	resolverID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
		return
	}
	questionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid forecast question id.", err))
		return
	}
	var req struct {
		Outcome     QuestionOutcome `json:"outcome"`
		EvidenceURL string          `json:"evidence_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	question, err := h.svc.ResolveQuestion(r.Context(), questionID, resolverID, req.Outcome, req.EvidenceURL)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, question)
}

func (h *Handler) ProgramView(w http.ResponseWriter, r *http.Request) {
	view, err := h.svc.ProgramRiskView(r.Context(), chi.URLParam(r, "program"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}

func (h *Handler) CreateExternalSignal(w http.ResponseWriter, r *http.Request) {
	if !auth.IsAdminFromContext(r.Context()) {
		httpx.WriteError(w, httpx.NewError(http.StatusForbidden, httpx.CodeForbidden, "Forbidden.", nil))
		return
	}
	questionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid forecast question id.", err))
		return
	}
	var req struct {
		Source         string `json:"source"`
		ProbabilityBps int    `json:"probability_bps"`
		Note           string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	signal, err := h.svc.CreateExternalSignal(r.Context(), questionID, req.Source, req.ProbabilityBps, req.Note)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, signal)
}
