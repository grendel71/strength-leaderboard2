package handler

import (
	"fmt"
	"math/big"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/blau/strength-leaderboard2/internal/auth"
	"github.com/blau/strength-leaderboard2/internal/db"
	"github.com/blau/strength-leaderboard2/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AthleteHandler struct {
	queries *db.Queries
	storage *storage.S3Storage
}

func NewAthleteHandler(q *db.Queries, s *storage.S3Storage) *AthleteHandler {
	return &AthleteHandler{queries: q, storage: s}
}

func (h *AthleteHandler) View(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	athlete, err := h.queries.GetAthleteByID(r.Context(), int32(id))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// HTMX requests get just the profile card for dialog
	if r.Header.Get("HX-Request") == "true" {
		renderPartial(w, "profile_card", pageData{
			User:     auth.UserFromContext(r.Context()),
			Athlete:  &athlete,
			Dialog:   true,
		})
		return
	}

	renderPage(w, "profile", pageData{
		User:    auth.UserFromContext(r.Context()),
		Athlete: &athlete,
	})
}

func (h *AthleteHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !user.AthleteID.Valid {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	athlete, err := h.queries.GetAthleteByID(r.Context(), user.AthleteID.Int32)
	if err != nil {
		http.Error(w, "athlete not found", http.StatusNotFound)
		return
	}

	renderPage(w, "profile_edit", pageData{
		User:    user,
		Athlete: &athlete,
	})
}

func (h *AthleteHandler) EditSave(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !user.AthleteID.Valid {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Increase max memory for file uploads
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	// Handle Avatar Upload
	avatarURL := r.FormValue("avatar_url")
	file, header, err := r.FormFile("avatar_file")
	if err == nil {
		defer file.Close()
		ext := filepath.Ext(header.Filename)
		key := fmt.Sprintf("avatars/%d_%d%s", user.AthleteID.Int32, time.Now().Unix(), ext)
		newURL, uploadErr := h.storage.Upload(r.Context(), key, file, header.Header.Get("Content-Type"))
		if uploadErr != nil {
			http.Error(w, "upload failed: "+uploadErr.Error(), http.StatusInternalServerError)
			return
		}
		avatarURL = newURL
	}

	params := db.UpdateAthleteParams{
		ID:         user.AthleteID.Int32,
		Name:       r.FormValue("name"),
		Gender:     pgtype.Text{String: r.FormValue("gender"), Valid: r.FormValue("gender") != ""},
		BodyWeight: parseDecimal(r.FormValue("body_weight")),
		AvatarUrl:  pgtype.Text{String: avatarURL, Valid: avatarURL != ""},
		Bio:        pgtype.Text{String: r.FormValue("bio"), Valid: r.FormValue("bio") != ""},
		Squat:      parseDecimal(r.FormValue("squat")),
		Bench:      parseDecimal(r.FormValue("bench")),
		Deadlift:   parseDecimal(r.FormValue("deadlift")),
		Ohp:        parseDecimal(r.FormValue("ohp")),
	}

	// Auto-calculate total
	params.Total = calcTotal(params.Squat, params.Bench, params.Deadlift)

	athlete, err := h.queries.UpdateAthlete(r.Context(), params)
	if err != nil {
		renderPage(w, "profile_edit", pageData{
			User:    user,
			Athlete: &athlete,
			Error:   "Failed to save: " + err.Error(),
		})
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		renderPartial(w, "profile_card", pageData{
			User:    user,
			Athlete: &athlete,
			Success: "Profile updated",
		})
		return
	}

	http.Redirect(w, r, "/athlete/"+strconv.Itoa(int(athlete.ID)), http.StatusSeeOther)
}

func parseDecimal(s string) pgtype.Numeric {
	if s == "" {
		return pgtype.Numeric{Valid: false}
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return pgtype.Numeric{Valid: false}
	}
	rat := new(big.Rat).SetFloat64(f)
	return pgtype.Numeric{
		Int:   rat.Num(),
		Exp:   0,
		Valid: true,
	}
}

func calcTotal(squat, bench, deadlift pgtype.Numeric) pgtype.Numeric {
	if !squat.Valid || !bench.Valid || !deadlift.Valid {
		return pgtype.Numeric{Valid: false}
	}
	s, _ := numericToFloat(squat)
	b, _ := numericToFloat(bench)
	d, _ := numericToFloat(deadlift)
	total := s + b + d
	rat := new(big.Rat).SetFloat64(total)
	return pgtype.Numeric{
		Int:   rat.Num(),
		Exp:   0,
		Valid: true,
	}
}

func numericToFloat(n pgtype.Numeric) (float64, bool) {
	if !n.Valid {
		return 0, false
	}
	fl, err := n.Float64Value()
	if err != nil || !fl.Valid {
		return 0, false
	}
	return fl.Float64, true
}
