package user

import (
	"encoding/json"
	"net/http"

	"github.com/naranjax/inhousepredictor/internal/auth"
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
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	u, err := h.svc.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respond(w, http.StatusCreated, u)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	u, err := h.svc.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token, err := auth.NewToken(u.ID, u.IsAdmin, h.jwtSecret)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	u, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	respond(w, http.StatusOK, u)
}

func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
