package main

import (
	"alyo/internal/core/database"
	"alyo/internal/core/models"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/animes", app.apiListAnimesHandler)
		r.Get("/channels", app.apiChannelsHandler) // Endpoint baru
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%s", port),
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return srv.ListenAndServe()
}

// homeHandler menangani permintaan ke halaman utama.
func (app *Application) homeHandler(w http.ResponseWriter, r *http.Request) {
	const pageSize = 24

	// Ambil data untuk Top 10 Mingguan
	topWeekly, err := app.Store.GetTopWeeklyAnimes()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// Ambil data untuk halaman pertama
	getParams := database.GetAnimesParams{
		Search: "",
		Sort:   "updated_desc",
		Limit:  pageSize,
		Offset: 0,
	}
	animes, err := app.Store.GetAnimes(getParams)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// Hitung total anime untuk pagination awal
	countParams := database.GetAnimesParams{Search: ""}
	totalAnimes, _ := app.Store.CountAnimes(countParams)
	totalPages := int(math.Ceil(float64(totalAnimes) / float64(pageSize)))

	// Ambil data channel untuk mapping
	channelsMap, err := app.Store.GetAllChannelsMap()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Siapkan semua data untuk di-render oleh server
	data := map[string]interface{}{
		"Animes":      animes,
		"TopWeekly":   topWeekly,
		"ChannelsMap": channelsMap,
		"CurrentYear": time.Now().Year(),
		"Sort":        "updated_desc",
		"CurrentPage": 1,
		"TotalPages":  totalPages,
		"NextPage":    2,
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

	// Fungsi kustom untuk digunakan di dalam template
	funcMap := template.FuncMap{
		"split": strings.Split,
		"json": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				log.Println("Error marshalling to JSON:", err)
				return template.JS("{}")
			}
			return template.JS(b)
		},
	}

	pages, err := filepath.Glob("./web/templates/*.page.html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		// Tambahkan .Funcs(funcMap) saat mem-parsing template
		ts, err := template.New(name).Funcs(funcMap).ParseFiles(page)
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
	const pageSize = 24

	query := r.URL.Query()
	searchQuery := query.Get("search")
	sortOption := query.Get("sort")
	pageStr := query.Get("page")

	if sortOption == "" {
		sortOption = "updated_desc"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	// Hitung total hasil untuk pagination
	countParams := database.GetAnimesParams{Search: searchQuery}
	totalAnimes, err := app.Store.CountAnimes(countParams)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	totalPages := int(math.Ceil(float64(totalAnimes) / float64(pageSize)))

	// Ambil data anime untuk halaman saat ini
	offset := (page - 1) * pageSize
	getParams := database.GetAnimesParams{
		Search: searchQuery,
		Sort:   sortOption,
		Limit:  pageSize,
		Offset: offset,
	}
	animes, err := app.Store.GetAnimes(getParams)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// Siapkan respons JSON yang terstruktur
	response := struct {
		Animes     []models.Anime `json:"animes"`
		Pagination struct {
			CurrentPage int `json:"currentPage"`
			TotalPages  int `json:"totalPages"`
		} `json:"pagination"`
	}{
		Animes: animes,
		Pagination: struct {
			CurrentPage int `json:"currentPage"`
			TotalPages  int `json:"totalPages"`
		}{
			CurrentPage: page,
			TotalPages:  totalPages,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (app *Application) apiChannelsHandler(w http.ResponseWriter, r *http.Request) {
	channelsMap, err := app.Store.GetAllChannelsMap()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channelsMap)
}
