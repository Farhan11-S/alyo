package main

import (
	"alyo/internal/core/database"
	"encoding/json"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

type Application struct {
	Store     database.Store
	Templates map[string]*template.Template
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	port := os.Getenv("PORT")
	dbURL := os.Getenv("DATABASE_URL")
	if port == "" || dbURL == "" {
		log.Fatal("PORT and DATABASE_URL must be set")
	}

	store, err := database.NewDBStore(dbURL)
	if err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}

	app := &Application{Store: store}

	log.Printf("Starting API server on port %s", port)
	if err := app.serve(port); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}

// serve mengatur router dan memulai server HTTP.
func (app *Application) serve(port string) error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Handler untuk halaman
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/animes", app.apiListAnimesHandler)
		r.Get("/animes/{id}", app.apiDetailAnimeHandler)
		r.Get("/channels", app.apiChannelsHandler)
		r.Get("/top-weekly", app.apiTopWeeklyHandler)
	})

	imageServer := http.FileServer(http.Dir("./web/"))
	r.Handle("/img/*", http.StripPrefix("/", imageServer))

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return srv.ListenAndServe()
}

// homeHandler menangani permintaan ke halaman utama.
func (app *Application) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

// API
func (app *Application) apiListAnimesHandler(w http.ResponseWriter, r *http.Request) {
	const pageSize = 24
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}

	params := database.GetAnimesParams{
		Search: query.Get("search"),
		Sort:   query.Get("sort"),
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}

	animes, err := app.Store.GetAnimes(params)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch animes"})
		return
	}

	totalAnimes, _ := app.Store.CountAnimes(params)
	totalPages := int(math.Ceil(float64(totalAnimes) / float64(pageSize)))

	response := map[string]interface{}{
		"data": animes,
		"pagination": map[string]int{
			"currentPage": page,
			"totalPages":  totalPages,
		},
	}
	app.writeJSON(w, http.StatusOK, response)
}

func (app *Application) apiDetailAnimeHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid anime ID"})
		return
	}

	anime, err := app.Store.GetAnimeWithEpisodes(id)
	if err != nil {
		app.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Anime not found"})
		return
	}
	app.writeJSON(w, http.StatusOK, anime)
}

func (app *Application) apiChannelsHandler(w http.ResponseWriter, r *http.Request) {
	channels, err := app.Store.GetAllChannelsMap()
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch channels"})
		return
	}
	app.writeJSON(w, http.StatusOK, channels)
}

func (app *Application) apiTopWeeklyHandler(w http.ResponseWriter, r *http.Request) {
	animes, err := app.Store.GetTopWeeklyAnimes()
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch top weekly animes"})
		return
	}
	app.writeJSON(w, http.StatusOK, animes)
}
