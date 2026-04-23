package handler

import (
	"net/http"

	"github.com/blau/strength-leaderboard2/internal/auth"
	"github.com/blau/strength-leaderboard2/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type LeaderboardHandler struct {
	queries *db.Queries
}

func NewLeaderboardHandler(q *db.Queries) *LeaderboardHandler {
	return &LeaderboardHandler{queries: q}
}

func (h *LeaderboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	gender := r.URL.Query().Get("gender")
	if sort == "" {
		sort = "total"
	}

	athletes, err := h.fetchAthletes(r, sort, gender)
	if err != nil {
		http.Error(w, "failed to load leaderboard", http.StatusInternalServerError)
		return
	}

	data := pageData{
		User:     auth.UserFromContext(r.Context()),
		Athletes: athletes,
		Sort:     sort,
		Gender:   gender,
	}

	// If HTMX request, only return the table partial
	if r.Header.Get("HX-Request") == "true" {
		renderPartial(w, "leaderboard_table", data)
		return
	}

	renderPage(w, "leaderboard", data)
}

func (h *LeaderboardHandler) fetchAthletes(r *http.Request, sort, gender string) ([]db.Athlete, error) {
	ctx := r.Context()
	if gender != "" {
		return h.queries.ListAthletesSortedByGender(ctx, db.ListAthletesSortedByGenderParams{
			Gender:    pgtype.Text{String: gender, Valid: true},
			SortField: sort,
		})
	}
	return h.queries.ListAthletesSorted(ctx, sort)
}
