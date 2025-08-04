-- File: 000001_create_initial_tables.up.sql
-- Perintah untuk membuat semua tabel (migrasi NAIK KONSOLIDASI)

CREATE TABLE IF NOT EXISTS channels (
    channel_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(255) NOT NULL,
    profile_picture_url VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS animes (
    anime_id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL UNIQUE,
    synopsis TEXT,
    thumbnail_url VARCHAR(255),
    release_year INT,
    last_updated TIMESTAMPTZ,
    -- Kolom baru untuk data views
    total_view_count BIGINT DEFAULT 0,
    weekly_view_increase BIGINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS playlists (
    playlist_id VARCHAR(255) PRIMARY KEY,
    channel_id VARCHAR(255) NOT NULL,
    anime_id INT,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    FOREIGN KEY (channel_id) REFERENCES channels(channel_id) ON DELETE CASCADE,
    FOREIGN KEY (anime_id) REFERENCES animes(anime_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS episodes (
    video_id VARCHAR(255) PRIMARY KEY,
    playlist_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    episode_number INT,
    published_at TIMESTAMptZ,
    thumbnail_url VARCHAR(255),
    -- Kolom baru untuk data views
    view_count BIGINT DEFAULT 0,
    FOREIGN KEY (playlist_id) REFERENCES playlists(playlist_id) ON DELETE CASCADE
);

-- Membuat index untuk mempercepat query
CREATE INDEX IF NOT EXISTS idx_playlists_channel_id ON playlists(channel_id);
CREATE INDEX IF NOT EXISTS idx_playlists_anime_id ON playlists(anime_id);
CREATE INDEX IF NOT EXISTS idx_playlists_language ON playlists(language);
CREATE INDEX IF NOT EXISTS idx_episodes_playlist_id ON episodes(playlist_id);
CREATE INDEX IF NOT EXISTS idx_animes_last_updated ON animes(last_updated);
-- Index baru untuk sorting berdasarkan views
CREATE INDEX IF NOT EXISTS idx_animes_total_view_count ON animes(total_view_count);
CREATE INDEX IF NOT EXISTS idx_animes_weekly_view_increase ON animes(weekly_view_increase);
