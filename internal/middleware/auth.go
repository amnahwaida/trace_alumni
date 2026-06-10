package middleware

import (
	"context"
	"net/http"
	"time"

	"trace-alumni/internal/database"
	"trace-alumni/internal/models"
)

type contextKey string

const UserContextKey contextKey = "user"

// GetUser retrieves the authenticated user from request context
func GetUser(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// AuthMiddleware checks for valid session cookie and injects user into context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var user models.User
		err = database.DB.QueryRow(`
			SELECT u.id, u.username, u.role, u.created_at
			FROM sessions s
			JOIN users u ON s.user_id = u.id
			WHERE s.id = ? AND s.expires_at > ?
		`, cookie.Value, time.Now().UTC()).Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt)

		if err != nil {
			// Invalid or expired session - clear cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Inject user into context
		ctx := context.WithValue(r.Context(), UserContextKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole returns middleware that restricts access to specific roles
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r)
			if user == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "Akses ditolak", http.StatusForbidden)
		})
	}
}
