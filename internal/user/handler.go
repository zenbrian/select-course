package user

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
)

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{
		service: service,
	}
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

const sessionCookieMaxAgeSeconds = 60 * 60 * 24

func (h *handler) Register(w http.ResponseWriter, r *http.Request) {
	var body AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	if body.Username == "" || body.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.service.Register(r.Context(), body.Username, body.Password)
	if err != nil {
		status, msg := mapUserError(err)
		slog.Warn("user register failed", "username", body.Username, "status", status, "error", err)
		http.Error(w, msg, status)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *handler) Login(w http.ResponseWriter, r *http.Request) {
	var body AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	if body.Username == "" || body.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.service.Login(r.Context(), body.Username, body.Password)
	if err != nil {
		status, msg := mapUserError(err)
		slog.Warn("user login failed", "username", body.Username, "status", status, "error", err)
		http.Error(w, msg, status)
		return
	}

	sessionID, err := h.service.CreateSession(r.Context(), user.ID)
	if err != nil {
		status, msg := mapUserError(err)
		http.Error(w, msg, status)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   sessionCookieMaxAgeSeconds,
		Expires:  time.Now().Add(sessionCookieMaxAgeSeconds * time.Second),
	})

	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		if err := h.service.DestroySession(r.Context(), cookie.Value); err != nil {
			status, msg := mapUserError(err)
			http.Error(w, msg, status)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "logged out successfully"}`))
}

func (h *handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		status, msg := mapUserError(err)
		http.Error(w, msg, status)
		return
	}

	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *handler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		status, msg := mapUserError(err)
		http.Error(w, msg, status)
		return
	}

	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func mapUserError(err error) (int, string) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound, "user not found"
	case errors.Is(err, ErrWrongPassword):
		return http.StatusUnauthorized, "invalid credentials"
	case errors.Is(err, ErrUsernameTaken):
		return http.StatusConflict, "username already exists"
	case errors.Is(err, ErrSessionNotFound):
		return http.StatusUnauthorized, "invalid session"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
