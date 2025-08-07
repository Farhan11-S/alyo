package main

import (
	"alyo/internal/core/database"
	"alyo/internal/core/models"
	"alyo/internal/youtube"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	"Ani-One Asia":   "UC0wNSTMWIL3qaorLx0jie6A",
	"Muse Asia":      "UCGbshtvS9t-8CW11W7TooQg",
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

		profilePicURL, err := app.YouTubeClient.GetChannelProfilePicture(id)
		if err != nil {
			log.Printf("ERROR: Could not get profile picture for channel %s: %v", name, err)
		}

		var localImagePath string
		if profilePicURL != "" {
			localImagePath = fmt.Sprintf("/img/channels/%s.jpg", id)
			fullPath := filepath.Join("web", strings.TrimPrefix(localImagePath, "/"))

			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				errDownload := downloadAndSaveImage(profilePicURL, fullPath)
				if errDownload != nil {
					log.Printf("ERROR: Could not download image for channel %s: %v", name, errDownload)
					localImagePath = ""
				}
			}
		}

		channelURL := "https://www.youtube.com/channel/" + id
		err = app.Store.UpsertChannel(models.Channel{ID: id, Name: name, URL: channelURL, ProfilePictureURL: &localImagePath})
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
			// Perbaikan: Memanggil findOrCreateAnime sebagai method dari app
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
			if len(videos) == 0 {
				continue
			}

			var videoIDs []string
			for _, v := range videos {
				videoIDs = append(videoIDs, v.Snippet.ResourceID.VideoID)
			}

			videoDetails, err := app.YouTubeClient.GetVideoDetails(videoIDs)
			if err != nil {
				log.Printf("    ERROR: Could not get video details for playlist '%s': %v", p.Snippet.Title, err)
				continue
			}

			viewCounts := make(map[string]int64)
			for _, detail := range videoDetails {
				vc, _ := strconv.ParseInt(detail.Statistics.ViewCount, 10, 64)
				viewCounts[detail.ID] = vc
			}

			var currentTotalViews int64 = 0
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
					ViewCount:     viewCounts[v.Snippet.ResourceID.VideoID],
				}
				err := app.Store.UpsertEpisode(episodeModel)
				if err != nil {
					log.Printf("      ERROR: Could not upsert episode '%s': %v", v.Snippet.Title, err)
				}

				currentTotalViews += episodeModel.ViewCount

				if latestEpisodeTime == nil || v.Snippet.PublishedAt.After(*latestEpisodeTime) {
					latestEpisodeTime = &v.Snippet.PublishedAt
				}

				if earliestDate == nil || v.Snippet.PublishedAt.Before(*earliestDate) {
					earliestDate = &v.Snippet.PublishedAt
					firstEpisodeThumbnailURL = &thumbURL
				}
			}

			oldTotalViews, err := app.Store.GetAnimeViewData(animeID)
			if err != nil {
				log.Printf("    WARN: Could not get old view data for anime ID %d: %v", animeID, err)
			}

			weeklyIncrease := currentTotalViews - oldTotalViews

			err = app.Store.UpdateAnimeViewData(animeID, currentTotalViews, weeklyIncrease)
			if err != nil {
				log.Printf("    ERROR: Could not update view data for anime ID %d: %v", animeID, err)
			}

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

func downloadAndSaveImage(url string, filePath string) error {
	dir := filepath.Dir(filePath)
	// Buat semua direktori perantara jika belum ada
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("received non-200 response code: %d", response.StatusCode)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	log.Printf("Successfully downloaded and saved image to %s", filePath)
	return nil
}

// Perbaikan: Mengubah findOrCreateAnime menjadi method dari *AppConfig
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
	indonesianKeywords := []string{"sub indo", "indonesia", "[id]"}
	for _, keyword := range indonesianKeywords {
		if strings.Contains(lowerTitle, keyword) {
			return "id"
		}
	}
	return "en"
}
