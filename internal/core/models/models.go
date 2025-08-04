package models

import "time"

// Channel merepresentasikan tabel 'channels'
type Channel struct {
	ID                string  `db:"channel_id"`
	Name              string  `db:"name"`
	URL               string  `db:"url"`
	ProfilePictureURL *string `db:"profile_picture_url"`
}

// Anime merepresentasikan tabel 'animes'
type Anime struct {
	ID                 int        `db:"anime_id"`
	Title              string     `db:"title"`
	Synopsis           *string    `db:"synopsis"`
	ThumbnailURL       *string    `db:"thumbnail_url"`
	ReleaseYear        *int       `db:"release_year"`
	LastUpdated        *time.Time `db:"last_updated"`
	TotalViewCount     int64      `db:"total_view_count"`
	WeeklyViewIncrease int64      `db:"weekly_view_increase"`
	ChannelID          string     `db:"channel_id"`
	Language           string     `db:"language"`
}

// Playlist merepresentasikan tabel 'playlists'
type Playlist struct {
	ID          string  `db:"playlist_id"`
	ChannelID   string  `db:"channel_id"`
	AnimeID     *int    `db:"anime_id"`
	Title       string  `db:"title"`
	Description *string `db:"description"`
	Language    string  `db:"language"`
}

// Episode merepresentasikan tabel 'episodes'
type Episode struct {
	VideoID       string     `db:"video_id"`
	PlaylistID    string     `db:"playlist_id"`
	Title         string     `db:"title"`
	EpisodeNumber *int       `db:"episode_number"`
	PublishedAt   *time.Time `db:"published_at"`
	ThumbnailURL  *string    `db:"thumbnail_url"`
	ViewCount     int64      `db:"view_count"` // Field baru
}

// AnimeWithEpisodes adalah struct gabungan untuk halaman detail.
type AnimeWithEpisodes struct {
	Anime
	Episodes []Episode
}
