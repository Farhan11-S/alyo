package database

import (
	"alyo/internal/core/models"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// GetAnimesParams adalah struct untuk parameter pencarian, filter, dan sort.
type GetAnimesParams struct {
	Search string
	Sort   string
	Limit  int
	Offset int
}

// Store mendefinisikan semua fungsi untuk berinteraksi dengan database.
type Store interface {
	UpsertChannel(channel models.Channel) error
	FindAnimeByTitle(title string) (*models.Anime, error)
	UpsertAnime(anime models.Anime) (int, error)
	UpsertPlaylist(playlist models.Playlist) error
	UpsertEpisode(episode models.Episode) error
	GetAllAnimes() ([]models.Anime, error)
	GetAnimeWithEpisodes(animeID int) (*models.AnimeWithEpisodes, error)
	GetAnimes(params GetAnimesParams) ([]models.Anime, error)
	CountAnimes(params GetAnimesParams) (int, error)
	UpdateAnimeLastUpdated(animeID int, timestamp time.Time) error
	UpdateAnimeThumbnailURL(animeID int, url string) error
	GetAnimeViewData(animeID int) (totalViews int64, err error)
	UpdateAnimeViewData(animeID int, totalViews int64, weeklyIncrease int64) error
	GetTopWeeklyAnimes() ([]models.Anime, error)
	GetAllChannelsMap() (map[string]models.Channel, error)
}

// DBStore adalah implementasi dari Store menggunakan PostgreSQL.
type DBStore struct {
	db *sqlx.DB
}

// NewDBStore membuat instance baru dari DBStore.
func NewDBStore(databaseURL string) (Store, error) {
	db, err := sqlx.Connect("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Println("Successfully connected to the database")
	return &DBStore{db: db}, nil
}

// UpsertChannel menyisipkan channel baru atau memperbarui yang sudah ada.
func (s *DBStore) UpsertChannel(channel models.Channel) error {
	query := `INSERT INTO channels (channel_id, name, url, profile_picture_url) VALUES ($1, $2, $3, $4) ON CONFLICT (channel_id) DO UPDATE SET name = EXCLUDED.name, url = EXCLUDED.url, profile_picture_url = EXCLUDED.profile_picture_url;`
	_, err := s.db.Exec(query, channel.ID, channel.Name, channel.URL, channel.ProfilePictureURL)
	return err
}

// FindAnimeByTitle mencari anime berdasarkan judulnya.
func (s *DBStore) FindAnimeByTitle(title string) (*models.Anime, error) {
	var anime models.Anime
	query := `SELECT * FROM animes WHERE title = $1`
	err := s.db.Get(&anime, query, title)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &anime, err
}

// UpsertAnime menyisipkan anime baru atau memperbarui yang sudah ada.
func (s *DBStore) UpsertAnime(anime models.Anime) (int, error) {
	var animeID int
	query := `INSERT INTO animes (title, synopsis, thumbnail_url, release_year) VALUES ($1, $2, $3, $4) ON CONFLICT (title) DO UPDATE SET synopsis = EXCLUDED.synopsis, thumbnail_url = EXCLUDED.thumbnail_url, release_year = EXCLUDED.release_year RETURNING anime_id;`
	err := s.db.QueryRowx(query, anime.Title, anime.Synopsis, anime.ThumbnailURL, anime.ReleaseYear).Scan(&animeID)
	return animeID, err
}

// UpsertPlaylist menyisipkan playlist baru atau memperbarui yang sudah ada, termasuk bahasa.
func (s *DBStore) UpsertPlaylist(playlist models.Playlist) error {
	query := `INSERT INTO playlists (playlist_id, channel_id, anime_id, title, description, language) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (playlist_id) DO UPDATE SET channel_id = EXCLUDED.channel_id, anime_id = EXCLUDED.anime_id, title = EXCLUDED.title, description = EXCLUDED.description, language = EXCLUDED.language;`
	_, err := s.db.Exec(query, playlist.ID, playlist.ChannelID, playlist.AnimeID, playlist.Title, playlist.Description, playlist.Language)
	return err
}

// UpsertEpisode menyisipkan episode baru atau memperbarui yang sudah ada.
func (s *DBStore) UpsertEpisode(episode models.Episode) error {
	query := `INSERT INTO episodes (video_id, playlist_id, title, episode_number, published_at, thumbnail_url, view_count) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (video_id) DO UPDATE SET playlist_id = EXCLUDED.playlist_id, title = EXCLUDED.title, episode_number = EXCLUDED.episode_number, published_at = EXCLUDED.published_at, thumbnail_url = EXCLUDED.thumbnail_url, view_count = EXCLUDED.view_count;`
	_, err := s.db.Exec(query, episode.VideoID, episode.PlaylistID, episode.Title, episode.EpisodeNumber, episode.PublishedAt, episode.ThumbnailURL, episode.ViewCount)
	return err
}

func (s *DBStore) GetAllChannelsMap() (map[string]models.Channel, error) {
	channels := []models.Channel{}
	query := `SELECT * FROM channels`
	err := s.db.Select(&channels, query)
	if err != nil {
		return nil, err
	}

	channelMap := make(map[string]models.Channel)
	for _, ch := range channels {
		channelMap[ch.ID] = ch
	}
	return channelMap, nil
}

func (s *DBStore) CountAnimes(params GetAnimesParams) (int, error) {
	var count int
	// Query ini hanya menghitung, jadi lebih cepat.
	baseQuery := `
		SELECT COUNT(DISTINCT a.anime_id)
		FROM animes a
		JOIN playlists p ON a.anime_id = p.anime_id
	`
	conditions := []string{"a.thumbnail_url IS NOT NULL"}
	var args []interface{}
	argID := 1

	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("a.title ILIKE $%d", argID))
		args = append(args, "%"+params.Search+"%")
		argID++
	}

	whereClause := " WHERE " + strings.Join(conditions, " AND ")
	finalQuery := baseQuery + whereClause

	err := s.db.Get(&count, finalQuery, args...)
	return count, err
}

// GetAnimes diperbarui untuk memfilter berdasarkan bahasa.
func (s *DBStore) GetAnimes(params GetAnimesParams) ([]models.Anime, error) {
	var animes []models.Anime
	baseQuery := `
		SELECT
			a.anime_id, a.title, a.synopsis, a.thumbnail_url, a.release_year,
			a.last_updated, a.total_view_count, a.weekly_view_increase,
			(array_agg(p.channel_id))[1] as channel_id,
			string_agg(DISTINCT p.language, ',') as languages
		FROM animes a
		JOIN playlists p ON a.anime_id = p.anime_id
	`
	conditions := []string{"a.thumbnail_url IS NOT NULL"}
	var args []interface{}
	argID := 1

	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("a.title ILIKE $%d", argID))
		args = append(args, "%"+params.Search+"%")
		argID++
	}

	whereClause := " WHERE " + strings.Join(conditions, " AND ")
	groupByClause := " GROUP BY a.anime_id"
	orderBy := " ORDER BY last_updated DESC NULLS LAST"

	switch params.Sort {
	case "name_asc":
		orderBy = " ORDER BY title ASC"
	case "name_desc":
		orderBy = " ORDER BY title DESC"
	case "updated_asc":
		orderBy = " ORDER BY last_updated ASC NULLS LAST"
	case "updated_desc":
		orderBy = " ORDER BY last_updated DESC NULLS LAST"
	case "views_desc":
		orderBy = " ORDER BY total_view_count DESC NULLS LAST"
	}

	// Menambahkan LIMIT dan OFFSET untuk pagination
	paginationClause := fmt.Sprintf(" LIMIT %d OFFSET %d", params.Limit, params.Offset)

	finalQuery := baseQuery + whereClause + groupByClause + orderBy + paginationClause

	err := s.db.Select(&animes, finalQuery, args...)
	return animes, err
}

func (s *DBStore) GetAnimeViewData(animeID int) (totalViews int64, err error) {
	query := `SELECT total_view_count FROM animes WHERE anime_id = $1`
	err = s.db.Get(&totalViews, query, animeID)
	// Jika tidak ada baris (anime baru), kembalikan 0 sebagai nilai lama.
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return
}

func (s *DBStore) UpdateAnimeViewData(animeID int, newTotalViews int64, weeklyIncrease int64) error {
	query := `UPDATE animes SET total_view_count = $1, weekly_view_increase = $2 WHERE anime_id = $3`
	_, err := s.db.Exec(query, newTotalViews, weeklyIncrease, animeID)
	return err
}

func (s *DBStore) GetTopWeeklyAnimes() ([]models.Anime, error) {
	var animes []models.Anime
	query := `SELECT * FROM animes WHERE thumbnail_url IS NOT NULL AND weekly_view_increase > 0 ORDER BY weekly_view_increase DESC NULLS LAST LIMIT 10`
	err := s.db.Select(&animes, query)
	return animes, err
}

// UpdateAnimeLastUpdated memperbarui timestamp anime.
func (s *DBStore) UpdateAnimeLastUpdated(animeID int, timestamp time.Time) error {
	query := `UPDATE animes SET last_updated = $1 WHERE anime_id = $2`
	_, err := s.db.Exec(query, timestamp, animeID)
	return err
}

// UpdateAnimeThumbnailURL memperbarui thumbnail anime jika belum ada.
func (s *DBStore) UpdateAnimeThumbnailURL(animeID int, newURL string) error {
	if newURL == "" {
		return nil // Tidak melakukan apa-apa jika URL kosong.
	}

	// 2. Cek jika URL yang diberikan adalah URL yang valid.
	_, err := url.ParseRequestURI(newURL)
	if err != nil {
		// URL tidak valid, jadi kita tidak menjalankan query.
		// Kita bisa mencatat peringatan ini jika perlu.
		log.Printf("WARN: Invalid URL provided for thumbnail, skipping update: %s", newURL)
		return nil
	}

	query := `UPDATE animes SET thumbnail_url = $1 WHERE anime_id = $2 AND thumbnail_url IS NULL`
	_, err = s.db.Exec(query, newURL, animeID)
	return err
}

// GetAllAnimes mengambil semua anime dari database (versi sederhana).
func (s *DBStore) GetAllAnimes() ([]models.Anime, error) {
	var animes []models.Anime
	query := `SELECT * FROM animes WHERE thumbnail_url IS NOT NULL ORDER BY title ASC`
	err := s.db.Select(&animes, query)
	return animes, err
}

// GetAnimeWithEpisodes mengambil satu anime beserta semua episodenya.
func (s *DBStore) GetAnimeWithEpisodes(animeID int) (*models.AnimeWithEpisodes, error) {
	var anime models.Anime
	queryAnime := `SELECT * FROM animes WHERE anime_id = $1`
	err := s.db.Get(&anime, queryAnime, animeID)
	if err != nil {
		return nil, err
	}

	var episodes []models.Episode
	queryEpisodes := `SELECT e.* FROM episodes e JOIN playlists p ON e.playlist_id = p.playlist_id WHERE p.anime_id = $1 ORDER BY e.episode_number ASC, e.published_at ASC;`
	err = s.db.Select(&episodes, queryEpisodes, animeID)
	if err != nil {
		return nil, err
	}

	return &models.AnimeWithEpisodes{
		Anime:    anime,
		Episodes: episodes,
	}, nil
}
