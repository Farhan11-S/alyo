package main

import (
	"alyo/internal/core/database"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

type Application struct {
	Store     database.Store
	Templates map[string]*template.Template
}

func main() {
	// Memuat konfigurasi dari .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	port := os.Getenv("PORT")
	dbURL := os.Getenv("DATABASE_URL")
	if port == "" || dbURL == "" {
		log.Fatal("PORT and DATABASE_URL must be set")
	}

	// Inisialisasi koneksi database
	store, err := database.NewDBStore(dbURL)
	if err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}

	// Muat template HTML ke dalam cache
	templates, err := newTemplateCache()
	if err != nil {
		log.Fatalf("Could not create template cache: %v", err)
	}

	app := &Application{
		Store:     store,
		Templates: templates,
	}

	// Jalankan server
	log.Printf("Starting web server on port %s", port)
	err = app.serve(port)
	if err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}

// serve mengatur router dan memulai server HTTP.
func (app *Application) serve(port string) error {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Handler untuk file statis (CSS, JS)
	fileServer := http.FileServer(http.Dir("./web/static/"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Handler untuk halaman
	r.Get("/", app.homeHandler)
	r.Get("/anime/{id}", app.animeDetailHandler)

	// API routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AllowContentType("application/json"))
		r.Get("/api/animes", app.apiListAnimesHandler)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return srv.ListenAndServe()
}

// homeHandler menangani permintaan ke halaman utama.
func (app *Application) homeHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	searchQuery := query.Get("search")
	sortOption := query.Get("sort")
	languageOption := query.Get("language") // <-- Parameter baru

	if sortOption == "" {
		sortOption = "updated_desc"
	}

	params := database.GetAnimesParams{
		Search:   searchQuery,
		Sort:     sortOption,
		Language: languageOption, // <-- Teruskan ke parameter DB
	}

	animes, err := app.Store.GetAnimes(params)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	data := map[string]interface{}{
		"Animes":      animes,
		"CurrentYear": time.Now().Year(),
		"Search":      searchQuery,
		"Sort":        sortOption,
		"Language":    languageOption, // <-- Kirim kembali ke template
	}

	app.render(w, r, "index.page.html", data)
}

// animeDetailHandler menangani permintaan ke halaman detail anime.
func (app *Application) animeDetailHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	animeWithEpisodes, err := app.Store.GetAnimeWithEpisodes(id)
	if err != nil {
		http.NotFound(w, r)
		log.Println(err)
		return
	}

	data := map[string]interface{}{
		"Anime":       animeWithEpisodes,
		"CurrentYear": time.Now().Year(),
	}

	app.render(w, r, "anime_detail.page.html", data)
}

// render merender template HTML.
func (app *Application) render(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	ts, ok := app.Templates[name]
	if !ok {
		http.Error(w, fmt.Sprintf("The template %s does not exist.", name), http.StatusInternalServerError)
		return
	}

	err := ts.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
	}
}

// newTemplateCache mem-parsing semua template HTML dan menyimpannya dalam map.
func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := filepath.Glob("./web/templates/*.page.html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		ts, err := template.ParseFiles(page)
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob("./web/templates/*.layout.html")
		if err != nil {
			return nil, err
		}
		cache[name] = ts
	}
	return cache, nil
}

// API
func (app *Application) apiListAnimesHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	params := database.GetAnimesParams{
		Search: query.Get("search"),
		Sort:   query.Get("sort"), // "name_asc", "name_desc", "updated_asc", "updated_desc"
	}

	animes, err := app.Store.GetAnimes(params)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("API Error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(animes)
}
