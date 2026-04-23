package handler

import (
	"fmt"
	"math/big"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
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

	bonusLifts, _ := h.queries.GetAthleteBonusLifts(r.Context(), athlete.ID)

	// HTMX requests get just the profile card for dialog
	if r.Header.Get("HX-Request") == "true" {
		renderPartial(w, "profile_card", pageData{
			User:       auth.UserFromContext(r.Context()),
			Athlete:    &athlete,
			Dialog:     true,
			BonusLifts: bonusLifts,
		})
		return
	}

	renderPage(w, "profile", pageData{
		User:       auth.UserFromContext(r.Context()),
		Athlete:    &athlete,
		BonusLifts: bonusLifts,
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

	bonusLifts, _ := h.queries.GetAthleteBonusLifts(r.Context(), athlete.ID)
	allBonusLifts, _ := h.queries.ListBonusLiftDefinitions(r.Context())

	renderPage(w, "profile_edit", pageData{
		User:          user,
		Athlete:       &athlete,
		BonusLifts:    bonusLifts,
		AllBonusLifts: allBonusLifts,
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

	// Handle Bonus Lifts (Other Lifts)
	// Inputs:
	// - bonus_lift_val_{definition_id}
	// - bonus_lift_distance_{definition_id}
	// - bonus_lift_reps_{definition_id}
	// - bonus_lift_remove_{definition_id}
	type bonusLiftUpdate struct {
		Remove   bool
		Value    pgtype.Numeric
		Distance pgtype.Numeric
		Reps     pgtype.Int4
	}

	updates := make(map[int32]*bonusLiftUpdate)
	getUpdate := func(defID int32) *bonusLiftUpdate {
		if u, ok := updates[defID]; ok {
			return u
		}
		u := &bonusLiftUpdate{}
		updates[defID] = u
		return u
	}

	for key, values := range r.MultipartForm.Value {
		if len(values) == 0 {
			continue
		}

		switch {
		case strings.HasPrefix(key, "bonus_lift_remove_"):
			defID, _ := strconv.Atoi(strings.TrimPrefix(key, "bonus_lift_remove_"))
			getUpdate(int32(defID)).Remove = values[0] != ""
		case strings.HasPrefix(key, "bonus_lift_val_"):
			defID, _ := strconv.Atoi(strings.TrimPrefix(key, "bonus_lift_val_"))
			getUpdate(int32(defID)).Value = parseDecimal(values[0])
		case strings.HasPrefix(key, "bonus_lift_distance_"):
			defID, _ := strconv.Atoi(strings.TrimPrefix(key, "bonus_lift_distance_"))
			getUpdate(int32(defID)).Distance = parseDecimal(values[0])
		case strings.HasPrefix(key, "bonus_lift_reps_"):
			defID, _ := strconv.Atoi(strings.TrimPrefix(key, "bonus_lift_reps_"))
			getUpdate(int32(defID)).Reps = parseInt4(values[0])
		}
	}

	for defID, u := range updates {
		if u.Remove {
			_ = h.queries.DeleteAthleteBonusLift(r.Context(), db.DeleteAthleteBonusLiftParams{
				AthleteID:        athlete.ID,
				LiftDefinitionID: defID,
			})
			continue
		}

		if !u.Value.Valid && !u.Distance.Valid && !u.Reps.Valid {
			_ = h.queries.DeleteAthleteBonusLift(r.Context(), db.DeleteAthleteBonusLiftParams{
				AthleteID:        athlete.ID,
				LiftDefinitionID: defID,
			})
			continue
		}

		_ = h.queries.UpsertAthleteBonusLift(r.Context(), db.UpsertAthleteBonusLiftParams{
			AthleteID:        athlete.ID,
			LiftDefinitionID: defID,
			Value:            u.Value,
			Distance:         u.Distance,
			Reps:             u.Reps,
		})
	}

	// Handle New Bonus Lift Definition (admin-only)
	newLiftName := r.FormValue("new_bonus_lift_name")
	if newLiftName != "" {
		if user.Role != "admin" {
			athlete, _ := h.queries.GetAthleteByID(r.Context(), user.AthleteID.Int32)
			bonusLifts, _ := h.queries.GetAthleteBonusLifts(r.Context(), athlete.ID)
			allBonusLifts, _ := h.queries.ListBonusLiftDefinitions(r.Context())
			renderPage(w, "profile_edit", pageData{
				User:          user,
				Athlete:       &athlete,
				BonusLifts:    bonusLifts,
				AllBonusLifts: allBonusLifts,
				Error:         "Only admins can create new lifts",
			})
			return
		}

		// Find or create definition
		def, err := h.queries.GetBonusLiftDefinitionByName(r.Context(), newLiftName)
		if err != nil {
			enableDistance := r.FormValue("new_bonus_lift_enable_distance") != ""
			enableReps := r.FormValue("new_bonus_lift_enable_reps") != ""
			def, err = h.queries.CreateBonusLiftDefinition(r.Context(), db.CreateBonusLiftDefinitionParams{
				Name:           newLiftName,
				EnableDistance: enableDistance,
				EnableReps:     enableReps,
			})
			if err != nil {
				athlete, _ := h.queries.GetAthleteByID(r.Context(), user.AthleteID.Int32)
				bonusLifts, _ := h.queries.GetAthleteBonusLifts(r.Context(), athlete.ID)
				allBonusLifts, _ := h.queries.ListBonusLiftDefinitions(r.Context())
				renderPage(w, "profile_edit", pageData{
					User:          user,
					Athlete:       &athlete,
					BonusLifts:    bonusLifts,
					AllBonusLifts: allBonusLifts,
					Error:         "Failed to create lift: " + err.Error(),
				})
				return
			}
		}

		val := parseDecimal(r.FormValue("new_bonus_lift_val"))
		distance := parseDecimal(r.FormValue("new_bonus_lift_distance"))
		reps := parseInt4(r.FormValue("new_bonus_lift_reps"))
		if val.Valid || distance.Valid || reps.Valid {
			_ = h.queries.UpsertAthleteBonusLift(r.Context(), db.UpsertAthleteBonusLiftParams{
				AthleteID:        athlete.ID,
				LiftDefinitionID: def.ID,
				Value:            val,
				Distance:         distance,
				Reps:             reps,
			})
		}
	}

	bonusLifts, _ := h.queries.GetAthleteBonusLifts(r.Context(), athlete.ID)

	if r.Header.Get("HX-Request") == "true" {
		renderPartial(w, "profile_card", pageData{
			User:       user,
			Athlete:    &athlete,
			BonusLifts: bonusLifts,
			Success:    "Profile updated",
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

func parseInt4(s string) pgtype.Int4 {
	if s == "" {
		return pgtype.Int4{Valid: false}
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(i), Valid: true}
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
