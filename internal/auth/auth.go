package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/blau/strength-leaderboard2/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookie = "session_id"
	sessionMaxAge = 30 * 24 * time.Hour // 30 days
	bcryptCost    = 12
)

type contextKey string

const userContextKey contextKey = "user"

type SessionUser struct {
	UserID    int32
	Username  string
	Role      string
	AthleteID pgtype.Int4
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionMaxAge.Seconds()),
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func GetSessionCookie(r *http.Request) string {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

func SessionExpiry() pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  time.Now().Add(sessionMaxAge),
		Valid: true,
	}
}

func WithUser(ctx context.Context, u *SessionUser) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

func UserFromContext(ctx context.Context) *SessionUser {
	u, _ := ctx.Value(userContextKey).(*SessionUser)
	return u
}

// Middleware loads the session user into context if a valid session cookie exists.
func Middleware(queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sid := GetSessionCookie(r)
			if sid != "" {
				sess, err := queries.GetSession(r.Context(), sid)
				if err == nil {
					r = r.WithContext(WithUser(r.Context(), &SessionUser{
						UserID:    sess.UserID.Int32,
						Username:  sess.Username,
						Role:      sess.Role,
						AthleteID: sess.AthleteID,
					}))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth redirects to /login if no user is in context.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
