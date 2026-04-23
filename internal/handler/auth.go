package handler

import (
	"net/http"
	"strings"

	iauth "github.com/blau/strength-leaderboard2/internal/auth"
	"github.com/blau/strength-leaderboard2/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuthHandler struct {
	queries *db.Queries
}

func NewAuthHandler(q *db.Queries) *AuthHandler {
	return &AuthHandler{queries: q}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if iauth.UserFromContext(r.Context()) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	renderPage(w, "login", pageData{})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderPage(w, "login", pageData{Error: "Invalid form"})
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		renderPage(w, "login", pageData{Error: "Username and password are required"})
		return
	}

	user, err := h.queries.GetUserByUsername(r.Context(), username)
	if err != nil || !iauth.CheckPassword(user.Password, password) {
		renderPage(w, "login", pageData{Error: "Invalid username or password"})
		return
	}

	sid, err := iauth.GenerateSessionID()
	if err != nil {
		renderPage(w, "login", pageData{Error: "Server error"})
		return
	}

	err = h.queries.CreateSession(r.Context(), db.CreateSessionParams{
		ID:        sid,
		UserID:    pgtype.Int4{Int32: user.ID, Valid: true},
		ExpiresAt: iauth.SessionExpiry(),
	})
	if err != nil {
		renderPage(w, "login", pageData{Error: "Server error"})
		return
	}

	iauth.SetSessionCookie(w, sid)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if iauth.UserFromContext(r.Context()) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	renderPage(w, "register", pageData{})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderPage(w, "register", pageData{Error: "Invalid form"})
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")
	displayName := strings.TrimSpace(r.FormValue("display_name"))
	gender := r.FormValue("gender")

	if username == "" || password == "" {
		renderPage(w, "register", pageData{Error: "Username and password are required"})
		return
	}
	if len(username) < 3 || len(username) > 50 {
		renderPage(w, "register", pageData{Error: "Username must be 3-50 characters"})
		return
	}
	if len(password) < 8 {
		renderPage(w, "register", pageData{Error: "Password must be at least 8 characters"})
		return
	}
	if password != confirm {
		renderPage(w, "register", pageData{Error: "Passwords do not match"})
		return
	}
	if displayName == "" {
		displayName = username
	}
	if gender == "" {
		gender = "male"
	}

	hash, err := iauth.HashPassword(password)
	if err != nil {
		renderPage(w, "register", pageData{Error: "Server error"})
		return
	}

	// Create athlete first
	athlete, err := h.queries.CreateAthlete(r.Context(), db.CreateAthleteParams{
		Name:   displayName,
		Gender: pgtype.Text{String: gender, Valid: true},
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			renderPage(w, "register", pageData{Error: "Display name already taken"})
			return
		}
		renderPage(w, "register", pageData{Error: "Failed to create profile"})
		return
	}

	// Create user linked to athlete
	user, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Username:  username,
		Password:  hash,
		AthleteID: pgtype.Int4{Int32: athlete.ID, Valid: true},
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			renderPage(w, "register", pageData{Error: "Username already taken"})
			return
		}
		renderPage(w, "register", pageData{Error: "Failed to create account"})
		return
	}

	// Auto-login
	sid, err := iauth.GenerateSessionID()
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_ = h.queries.CreateSession(r.Context(), db.CreateSessionParams{
		ID:        sid,
		UserID:    pgtype.Int4{Int32: user.ID, Valid: true},
		ExpiresAt: iauth.SessionExpiry(),
	})

	iauth.SetSessionCookie(w, sid)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sid := iauth.GetSessionCookie(r)
	if sid != "" {
		_ = h.queries.DeleteSession(r.Context(), sid)
	}
	iauth.ClearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
