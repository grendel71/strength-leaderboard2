package handler

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/blau/strength-leaderboard2/internal/auth"
	"github.com/blau/strength-leaderboard2/internal/db"
	"github.com/gomarkdown/markdown"
	"github.com/jackc/pgx/v5/pgtype"
)

var templates map[string]*template.Template

var funcMap = template.FuncMap{
	"decimal":   formatDecimal,
	"int4":     formatInt4,
	"rank":     func(i int) int { return i + 1 },
	"gender":   func(t pgtype.Text) string { return t.String },
	"avatar":   func(t pgtype.Text) string { return t.String },
	"has":      func(t pgtype.Text) bool { return t.Valid && t.String != "" },
	"isAdmin":  func(u *auth.SessionUser) bool { return u != nil && u.Role == "admin" },
	"md":       func(s string) template.HTML { return template.HTML(markdown.ToHTML([]byte(s), nil, nil)) },
	"add":      func(a, b int) int { return a + b },
	"boolToInt": func(b bool) int { if b { return 1 }; return 0 },
}

func InitTemplates(templateFS fs.FS) {
	templates = make(map[string]*template.Template)

	// Shared files included in every page
	shared := []string{
		"templates/layout.html",
		"templates/partials/nav.html",
		"templates/partials/leaderboard_table.html",
		"templates/partials/bonus_leaderboard_table.html",
		"templates/partials/profile_card.html",
	}

	pages := []string{
		"templates/leaderboard.html",
		"templates/bonus_leaderboard.html",
		"templates/login.html",
		"templates/register.html",
		"templates/profile.html",
		"templates/profile_edit.html",
	}

	for _, page := range pages {
		files := append([]string{page}, shared...)
		t := template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS, files...),
		)
		// Key by just the page name without path/extension
		name := pageName(page)
		templates[name] = t
	}
}

func pageName(path string) string {
	// "templates/leaderboard.html" -> "leaderboard"
	start := len("templates/")
	end := len(path) - len(".html")
	return path[start:end]
}

func formatDecimal(n pgtype.Numeric) string {
	if !n.Valid {
		return "-"
	}
	fl, err := n.Float64Value()
	if err != nil || !fl.Valid {
		return "-"
	}
	if fl.Float64 == float64(int64(fl.Float64)) {
		return fmt.Sprintf("%.0f", fl.Float64)
	}
	return fmt.Sprintf("%.1f", fl.Float64)
}

func formatInt4(n pgtype.Int4) string {
	if !n.Valid {
		return "-"
	}
	return fmt.Sprintf("%d", n.Int32)
}

type pageData struct {
	User                            *auth.SessionUser
	IsHTMX                          bool
	Athletes                        []db.Athlete
	Athlete                         *db.Athlete
	Sort                            string
	Gender                          string
	Error                           string
	Success                         string
	Dialog                          bool
	BonusLifts                      []db.GetAthleteBonusLiftsRow
	AllBonusLifts                   []db.BonusLiftDefinition
	BonusAthletes                   []db.ListAthletesByBonusLiftRow
	SelectedBonusLiftID             int32
	SelectedBonusLiftEnableDistance bool
	SelectedBonusLiftEnableReps     bool
	BonusLiftMetric                 string
	BonusLiftName                   string
}

func renderPage(w http.ResponseWriter, name string, data pageData) {
	t, ok := templates[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func renderPartial(w http.ResponseWriter, name string, data any) {
	// For partials, find any template that has it (they all share the same partials)
	for _, t := range templates {
		if p := t.Lookup(name); p != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := p.Execute(w, data); err != nil {
				http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	http.Error(w, "partial not found: "+name, http.StatusInternalServerError)
}
