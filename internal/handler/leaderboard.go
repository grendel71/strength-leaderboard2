package handler

import (
	"net/http"
	"strconv"

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
	if r.Header.Get("HX-Request") == "true" && (r.URL.Path == "/leaderboard" || r.URL.Path == "/") {
		renderPartial(w, "leaderboard_table", data)
		return
	}

	renderPage(w, "leaderboard", data)
}

func (h *LeaderboardHandler) BonusIndex(w http.ResponseWriter, r *http.Request) {
	liftIDStr := r.URL.Query().Get("lift_id")
	gender := r.URL.Query().Get("gender")

	allBonusLifts, _ := h.queries.ListBonusLiftDefinitions(r.Context())

	var selectedLiftID int32
	if liftIDStr != "" {
		id, _ := strconv.Atoi(liftIDStr)
		selectedLiftID = int32(id)
	} else if len(allBonusLifts) > 0 {
		selectedLiftID = allBonusLifts[0].ID
	}

	var bonusAthletes []db.ListAthletesByBonusLiftRow
	var liftName string
	if selectedLiftID > 0 {
		var err error
		bonusAthletes, err = h.queries.ListAthletesByBonusLift(r.Context(), db.ListAthletesByBonusLiftParams{
			ID:     selectedLiftID,
			Gender: pgtype.Text{String: gender, Valid: true},
		})
		if err == nil && len(bonusAthletes) > 0 {
			liftName = bonusAthletes[0].LiftName
		} else {
			// Fallback to definition list if query fails or empty
			for _, l := range allBonusLifts {
				if l.ID == selectedLiftID {
					liftName = l.Name
					break
				}
			}
		}
	}

	data := pageData{
		User:                auth.UserFromContext(r.Context()),
		BonusAthletes:       bonusAthletes,
		AllBonusLifts:       allBonusLifts,
		SelectedBonusLiftID: selectedLiftID,
		BonusLiftName:       liftName,
		Gender:              gender,
	}

	if r.Header.Get("HX-Request") == "true" && r.URL.Path == "/other" {
		renderPartial(w, "bonus_leaderboard_table", data)
		return
	}

	renderPage(w, "bonus_leaderboard", data)
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
