package models

import "time"

// Channel merepresentasikan tabel 'channels'
type Channel struct {
	ID                string  `db:"channel_id" json:"channel_id"`
	Name              string  `db:"name" json:"name"`
	URL               string  `db:"url" json:"url"`
	ProfilePictureURL *string `db:"profile_picture_url" json:"profile_picture_url"`
}

// Anime merepresentasikan tabel 'animes'
type Anime struct {
	ID                 int        `db:"anime_id" json:"anime_id"`
	Title              string     `db:"title" json:"title"`
	Synopsis           *string    `db:"synopsis" json:"synopsis"`
	ThumbnailURL       *string    `db:"thumbnail_url" json:"thumbnail_url"`
	ReleaseYear        *int       `db:"release_year" json:"release_year"`
	LastUpdated        *time.Time `db:"last_updated" json:"last_updated"`
	TotalViewCount     int64      `db:"total_view_count" json:"total_view_count"`
	WeeklyViewIncrease int64      `db:"weekly_view_increase" json:"weekly_view_increase"`
	ChannelID          string     `db:"channel_id" json:"channel_id"`
	Languages          string     `db:"languages" json:"languages"`
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
