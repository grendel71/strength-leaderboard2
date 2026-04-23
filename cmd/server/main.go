package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blau/strength-leaderboard2/internal/auth"
	"github.com/blau/strength-leaderboard2/internal/config"
	"github.com/blau/strength-leaderboard2/internal/db"
	"github.com/blau/strength-leaderboard2/internal/handler"
	"github.com/blau/strength-leaderboard2/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed all:templates
var templateFS embed.FS

//go:embed all:static
var staticFS embed.FS

func main() {
	cfg := config.Load()
	if cfg.DBHost == "" || cfg.DBUser == "" || cfg.DBName == "" {
		log.Fatal("DB_HOST, DB_USER, and DB_NAME are required")
	}

	// Database
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	if err := db.ApplyMigrations(context.Background(), pool, "migrations"); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	log.Println("connected to database")

	queries := db.New(pool)

	// Templates
	handler.InitTemplates(templateFS)

	// Storage
	s3Storage, err := storage.NewS3Storage(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to initialize S3 storage: %v", err)
	}

	// Handlers
	leaderboardH := handler.NewLeaderboardHandler(queries)
	athleteH := handler.NewAthleteHandler(queries, s3Storage)
	authH := handler.NewAuthHandler(queries)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(auth.Middleware(queries))

	// Static files
	r.Handle("/static/*", http.FileServerFS(staticFS))

	// Public routes
	r.Get("/", leaderboardH.Index)
	r.Get("/leaderboard", leaderboardH.Index)
	r.Get("/other", leaderboardH.BonusIndex)
	r.Get("/athlete/{id}", athleteH.View)
	r.Get("/login", authH.LoginPage)
	r.Post("/login", authH.Login)
	r.Get("/register", authH.RegisterPage)
	r.Post("/register", authH.Register)
	r.Post("/logout", authH.Logout)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth)
		r.Get("/profile/edit", athleteH.EditForm)
		r.Post("/profile/edit", athleteH.EditSave)
	})

	// Session cleanup (every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			_ = queries.DeleteExpiredSessions(context.Background())
		}
	}()

	// Server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server starting on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	fmt.Println("server stopped")
}
