package youtube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
type ChannelListResponse struct {
	Items []ChannelItem `json:"items"`
}

type ChannelItem struct {
	Snippet ChannelSnippet `json:"snippet"`
}

type ChannelSnippet struct {
	Thumbnails Thumbnails `json:"thumbnails"`
}

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
	Default Quality `json:"default"`
	High    Quality `json:"high"`
}

type Quality struct {
	URL string `json:"url"`
}

type VideoListResponse struct {
	Items []VideoDetailItem `json:"items"`
}

type VideoDetailItem struct {
	ID         string     `json:"id"`
	Statistics Statistics `json:"statistics"`
}

type Statistics struct {
	ViewCount string `json:"viewCount"`
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

func (c *Client) GetVideoDetails(videoIDs []string) ([]VideoDetailItem, error) {
	if len(videoIDs) == 0 {
		return nil, nil
	}

	// API YouTube memperbolehkan hingga 50 ID per permintaan.
	// Kita akan memprosesnya dalam batch jika lebih dari 50.
	var allVideoDetails []VideoDetailItem
	chunkSize := 50

	for i := 0; i < len(videoIDs); i += chunkSize {
		end := i + chunkSize
		if end > len(videoIDs) {
			end = len(videoIDs)
		}
		chunk := videoIDs[i:end]
		ids := strings.Join(chunk, ",")

		url := fmt.Sprintf("%s/videos?part=statistics&id=%s&key=%s", apiBaseURL, ids, c.apiKey)

		resp, err := c.httpClient.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch video details: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("youtube api returned non-200 status for video details: %s", resp.Status)
		}

		var response VideoListResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode video details response: %w", err)
		}

		allVideoDetails = append(allVideoDetails, response.Items...)
	}

	return allVideoDetails, nil
}

func (c *Client) GetChannelProfilePicture(channelID string) (string, error) {
	url := fmt.Sprintf("%s/channels?part=snippet&id=%s&key=%s", apiBaseURL, channelID, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch channel details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("youtube api returned non-200 status for channel details: %s", resp.Status)
	}

	var response ChannelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode channel details response: %w", err)
	}

	if len(response.Items) > 0 {
		return response.Items[0].Snippet.Thumbnails.Default.URL, nil
	}

	return "", nil
}
