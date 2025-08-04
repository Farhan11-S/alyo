package main

import (
	"alyo/internal/core/database"
	"alyo/internal/core/models"
	"alyo/internal/youtube"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

type AppConfig struct {
	Store         database.Store
	YouTubeClient *youtube.Client
}

var targetChannels = map[string]string{
	"Muse Indonesia": "UCxxnxya_32jcKj4yN1_kD7A",
	"Ani-One Asia":   "UC0wNSTMWIL3qaorLx0abI7A",
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if dbURL == "" || apiKey == "" {
		log.Fatal("DATABASE_URL and YOUTUBE_API_KEY must be set")
	}

	store, err := database.NewDBStore(dbURL)
	if err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}

	ytClient := youtube.NewClient(apiKey)

	app := AppConfig{
		Store:         store,
		YouTubeClient: ytClient,
	}

	log.Println("Starting cron job scheduler...")
	c := cron.New(cron.WithSeconds())

	_, err = c.AddFunc("0 0 */12 * * *", func() {
		log.Println("--- Running Worker ---")
		app.runWorker()
		log.Println("--- Worker Finished ---")
	})
	if err != nil {
		log.Fatalf("Could not add cron job: %v", err)
	}

	log.Println("--- Running initial worker job ---")
	app.runWorker()
	log.Println("--- Initial worker job finished ---")

	c.Start()
	select {}
}

func (app *AppConfig) runWorker() {
	for name, id := range targetChannels {
		log.Printf("Processing channel: %s", name)

		channelURL := "https://www.youtube.com/channel/" + id
		err := app.Store.UpsertChannel(models.Channel{ID: id, Name: name, URL: channelURL})
		if err != nil {
			log.Printf("ERROR: Could not upsert channel %s: %v", name, err)
			continue
		}

		playlists, err := app.YouTubeClient.GetPlaylistsForChannel(id)
		if err != nil {
			log.Printf("ERROR: Could not get playlists for channel %s: %v", name, err)
			continue
		}

		log.Printf("Found %d playlists for channel %s", len(playlists), name)

		for _, p := range playlists {
			if !isRelevantPlaylist(p.Snippet.Title) {
				continue
			}
			log.Printf("  -> Processing relevant playlist: %s", p.Snippet.Title)

			animeTitle := extractAnimeTitle(p.Snippet.Title)
			animeID, err := app.findOrCreateAnime(animeTitle, p.Snippet.Description)
			if err != nil {
				log.Printf("    ERROR: Could not find or create anime '%s': %v", animeTitle, err)
				continue
			}

			playlistModel := models.Playlist{
				ID:          p.ID,
				ChannelID:   p.Snippet.ChannelID,
				AnimeID:     &animeID,
				Title:       p.Snippet.Title,
				Description: &p.Snippet.Description,
				Language:    extractLanguage(p.Snippet.Title),
			}
			err = app.Store.UpsertPlaylist(playlistModel)
			if err != nil {
				log.Printf("    ERROR: Could not upsert playlist '%s': %v", p.Snippet.Title, err)
				continue
			}

			videos, err := app.YouTubeClient.GetVideosForPlaylist(p.ID)
			if err != nil {
				log.Printf("    ERROR: Could not get videos for playlist '%s': %v", p.Snippet.Title, err)
				continue
			}

			var latestEpisodeTime *time.Time
			var firstEpisodeThumbnailURL *string
			var earliestDate *time.Time

			for _, v := range videos {
				epNum := extractEpisodeNumber(v.Snippet.Title)
				thumbURL := v.Snippet.Thumbnails.High.URL

				episodeModel := models.Episode{
					VideoID:       v.Snippet.ResourceID.VideoID,
					PlaylistID:    p.ID,
					Title:         v.Snippet.Title,
					EpisodeNumber: epNum,
					PublishedAt:   &v.Snippet.PublishedAt,
					ThumbnailURL:  &thumbURL,
				}
				err := app.Store.UpsertEpisode(episodeModel)
				if err != nil {
					log.Printf("      ERROR: Could not upsert episode '%s': %v", v.Snippet.Title, err)
				}

				// Perbarui waktu episode terbaru
				if latestEpisodeTime == nil || v.Snippet.PublishedAt.After(*latestEpisodeTime) {
					latestEpisodeTime = &v.Snippet.PublishedAt
				}

				// Cari thumbnail dari episode paling awal (dianggap episode 1)
				if earliestDate == nil || v.Snippet.PublishedAt.Before(*earliestDate) {
					earliestDate = &v.Snippet.PublishedAt
					firstEpisodeThumbnailURL = &thumbURL
				}
			}

			// Setelah semua episode diproses, perbarui timestamp dan thumbnail anime
			if latestEpisodeTime != nil {
				err := app.Store.UpdateAnimeLastUpdated(animeID, *latestEpisodeTime)
				if err != nil {
					log.Printf("    ERROR: Could not update last_updated for anime ID %d: %v", animeID, err)
				}
			}
			if firstEpisodeThumbnailURL != nil {
				err := app.Store.UpdateAnimeThumbnailURL(animeID, *firstEpisodeThumbnailURL)
				if err != nil {
					log.Printf("    WARN: Could not update thumbnail for anime ID %d: %v", animeID, err)
				}
			}

			time.Sleep(2 * time.Second)
		}
	}
}

func (app *AppConfig) findOrCreateAnime(title, synopsis string) (int, error) {
	existingAnime, err := app.Store.FindAnimeByTitle(title)
	if err != nil {
		return 0, err
	}
	if existingAnime != nil {
		return existingAnime.ID, nil
	}
	log.Printf("    INFO: Anime '%s' not found, creating new entry.", title)
	newAnime := models.Anime{
		Title:    title,
		Synopsis: &synopsis,
	}
	return app.Store.UpsertAnime(newAnime)
}

func isRelevantPlaylist(title string) bool {
	lowerTitle := strings.ToLower(title)
	irrelevantKeywords := []string{"trailer", "pv", "ost", "theme song", "clip", "teaser"}
	for _, keyword := range irrelevantKeywords {
		if strings.Contains(lowerTitle, keyword) {
			return false
		}
	}
	relevantKeywords := []string{"episode", "full", "season", "s1", "s2", "s3", "s4"}
	for _, keyword := range relevantKeywords {
		if strings.Contains(lowerTitle, keyword) {
			return true
		}
	}
	return true
}

func extractAnimeTitle(playlistTitle string) string {
	re := regexp.MustCompile(`(?i)\[.*?\]|\(.*?\)|season \d|s\d|cour \d|part \d|full episode|sub indo`)
	title := re.ReplaceAllString(playlistTitle, "")
	return strings.TrimSpace(title)
}

func extractEpisodeNumber(videoTitle string) *int {
	re := regexp.MustCompile(`(?i)(?:episode|ep|#)\s*(\d{1,3})`)
	matches := re.FindStringSubmatch(videoTitle)
	if len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return &num
		}
	}
	return nil
}

func extractLanguage(title string) string {
	lowerTitle := strings.ToLower(title)
	// Kata kunci untuk Bahasa Indonesia
	indonesianKeywords := []string{"sub indo", "indonesia", "[id]"}
	for _, keyword := range indonesianKeywords {
		if strings.Contains(lowerTitle, keyword) {
			return "id"
		}
	}
	// Default ke Bahasa Inggris
	return "en"
}
