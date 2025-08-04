package youtube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	apiBaseURL = "https://www.googleapis.com/youtube/v3"
)

// Client adalah klien untuk berinteraksi dengan YouTube Data API v3.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient membuat instance baru dari YouTube Client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// --- Structs untuk Parsing JSON Response ---

type PlaylistListResponse struct {
	NextPageToken string         `json:"nextPageToken"`
	Items         []PlaylistItem `json:"items"`
}

type PlaylistItem struct {
	ID      string          `json:"id"`
	Snippet PlaylistSnippet `json:"snippet"`
}

type PlaylistSnippet struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ChannelID   string `json:"channelId"`
}

type PlaylistItemListResponse struct {
	NextPageToken string      `json:"nextPageToken"`
	Items         []VideoItem `json:"items"`
}

type VideoItem struct {
	Snippet VideoSnippet `json:"snippet"`
}

type VideoSnippet struct {
	Title       string     `json:"title"`
	PublishedAt time.Time  `json:"publishedAt"`
	ResourceID  ResourceID `json:"resourceId"`
	Thumbnails  Thumbnails `json:"thumbnails"`
}

type ResourceID struct {
	VideoID string `json:"videoId"`
}

type Thumbnails struct {
	High Quality `json:"high"`
}

type Quality struct {
	URL string `json:"url"`
}

// GetPlaylistsForChannel mengambil semua playlist dari sebuah channel.
func (c *Client) GetPlaylistsForChannel(channelID string) ([]PlaylistItem, error) {
	var allPlaylists []PlaylistItem
	pageToken := ""

	for {
		url := fmt.Sprintf("%s/playlists?part=snippet&channelId=%s&maxResults=50&key=%s&pageToken=%s",
			apiBaseURL, channelID, c.apiKey, pageToken)

		resp, err := c.httpClient.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch playlists: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("youtube api returned non-200 status: %s", resp.Status)
		}

		var response PlaylistListResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode playlist response: %w", err)
		}

		allPlaylists = append(allPlaylists, response.Items...)

		if response.NextPageToken == "" {
			break // Keluar dari loop jika tidak ada halaman berikutnya
		}
		pageToken = response.NextPageToken
	}

	return allPlaylists, nil
}

// GetVideosForPlaylist mengambil semua video dari sebuah playlist.
func (c *Client) GetVideosForPlaylist(playlistID string) ([]VideoItem, error) {
	var allVideos []VideoItem
	pageToken := ""

	for {
		url := fmt.Sprintf("%s/playlistItems?part=snippet&playlistId=%s&maxResults=50&key=%s&pageToken=%s",
			apiBaseURL, playlistID, c.apiKey, pageToken)

		resp, err := c.httpClient.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch videos: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("youtube api returned non-200 status: %s", resp.Status)
		}

		var response PlaylistItemListResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode video response: %w", err)
		}

		allVideos = append(allVideos, response.Items...)

		if response.NextPageToken == "" {
			break
		}
		pageToken = response.NextPageToken
	}

	return allVideos, nil
}
