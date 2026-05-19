package user

import (
	"context"
	"errors"
	"net/http"
)

const SessionCookieName = "session_id"

type contextKey string

const userIDContextKey contextKey = "user_id"

func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	v := ctx.Value(userIDContextKey)
	userID, ok := v.(int64)
	return userID, ok
}

func AuthMiddleware(service Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			userID, err := service.GetSessionUser(r.Context(), cookie.Value)
			if err != nil {
				if errors.Is(err, ErrSessionNotFound) {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithUserID(r.Context(), userID)))
		})
	}
}
