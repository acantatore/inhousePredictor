package user

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/naranjax/inhousepredictor/internal/auth"
	"github.com/naranjax/inhousepredictor/internal/httpx"
)

type Handler struct {
	svc       *Service
	jwtSecret string
}

func NewHandler(svc *Service, jwtSecret string) *Handler {
	return &Handler{svc: svc, jwtSecret: jwtSecret}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	u, err := h.svc.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, u)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Bad request.", err))
		return
	}
	u, err := h.svc.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Invalid credentials.", err))
		return
	}
	token, err := auth.NewToken(u.ID, u.IsAdmin, h.jwtSecret)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, httpx.CodeInternal, "Internal server error.", err))
		return
	}
	claims, err := auth.ParseToken(token, h.jwtSecret)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, httpx.CodeInternal, "Internal server error.", err))
		return
	}
	auth.SetTokenCookie(w, token, claims.ExpiresAt.Time)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
		return
	}
	u, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusNotFound, httpx.CodeNotFound, "Not found.", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, u)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, httpx.CodeBadRequest, "Invalid limit.", err))
			return
		}
		limit = parsed
	}

	users, err := h.svc.List(r.Context(), r.URL.Query().Get("q"), limit)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, httpx.CodeInternal, "Internal server error.", err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, users)
}
