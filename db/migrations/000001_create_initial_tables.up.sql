-- File: 000001_create_initial_tables.up.sql
-- Perintah untuk membuat semua tabel (migrasi NAIK KONSOLIDASI)

CREATE TABLE IF NOT EXISTS channels (
    channel_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS animes (
    anime_id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL UNIQUE,
    synopsis TEXT,
    thumbnail_url VARCHAR(255),
    release_year INT,
    last_updated TIMESTAMPTZ -- Kolom dari migrasi sebelumnya
);

CREATE TABLE IF NOT EXISTS playlists (
    playlist_id VARCHAR(255) PRIMARY KEY,
    channel_id VARCHAR(255) NOT NULL,
    anime_id INT,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    language VARCHAR(10) NOT NULL DEFAULT 'en', -- Kolom baru untuk bahasa
    FOREIGN KEY (channel_id) REFERENCES channels(channel_id) ON DELETE CASCADE,
    FOREIGN KEY (anime_id) REFERENCES animes(anime_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS episodes (
    video_id VARCHAR(255) PRIMARY KEY,
    playlist_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    episode_number INT,
    published_at TIMESTAMPTZ,
    thumbnail_url VARCHAR(255),
    FOREIGN KEY (playlist_id) REFERENCES playlists(playlist_id) ON DELETE CASCADE
);

-- Membuat index untuk mempercepat query
CREATE INDEX IF NOT EXISTS idx_playlists_channel_id ON playlists(channel_id);
CREATE INDEX IF NOT EXISTS idx_playlists_anime_id ON playlists(anime_id);
CREATE INDEX IF NOT EXISTS idx_playlists_language ON playlists(language); -- Index baru
CREATE INDEX IF NOT EXISTS idx_episodes_playlist_id ON episodes(playlist_id);
CREATE INDEX IF NOT EXISTS idx_animes_last_updated ON animes(last_updated); -- Index dari migrasi sebelumnya
