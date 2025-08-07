# Dokumentasi API ALYŌ v1

Hai! Ini dokumentasi API ALYŌ. Pake ini buat ngambil daftar anime gratis & legal dari YouTube, lengkap sama fitur cari, urut, dan data popularitas.

**Base URL**: `http://localhost:8080`

---

## Struktur Data

### Objek `Anime`
```json
{
    "anime_id": 1,
    "title": "Mushoku Tensei: Jobless Reincarnation",
    "thumbnail_url": "/img/animes/1.jpg",
    "last_updated": "2025-08-07T12:00:00Z",
    "total_view_count": 15000000,
    "weekly_view_increase": 250000,
    "languages": "id,en"
}
```

- thumbnail_url: Path ke gambar cache di server kita.

- weekly_view_increase: Jumlah penonton baru dalam seminggu, buat nentuin anime ngetren.

- languages: Subtitle yang tersedia (id untuk Indonesia, en untuk Inggris).

Objek Episode
```json
{
    "video_id": "some_video_id",
    "title": "[Sub Indo] Mushoku Tensei Season 2 - Episode 01",
    "episode_number": 1,
    "thumbnail_url": "[https://i.ytimg.com/vi/some_video_id/hqdefault.jpg](https://i.ytimg.com/vi/some_video_id/hqdefault.jpg)",
    "view_count": 1200000
}
```

- video_id: ID video YouTube buat bikin link nonton.

- thumbnail_url: Link langsung ke gambar sampul di server YouTube.

Endpoint API
1. Ambil Daftar Anime
Endpoint utama buat dapetin semua anime.

- Endpoint: GET /api/v1/animes

- Parameter:

    - search (string): Cari judul anime.

    - sort (string): Urutkan hasil. Pilihan: updated_desc (default), views_desc, name_asc, name_desc, updated_asc.

    - page (integer): Halaman pagination (default: 1).

Contoh Hasilnya:
```json
{
    "data": [ /* ... daftar anime ... */ ],
    "pagination": { "currentPage": 1, "totalPages": 5 }
}
```

2. Cek Detail Anime
Ambil info lengkap satu anime plus semua episodenya.

- Endpoint: GET /api/v1/animes/{id}

Contoh Hasilnya:
```json
{
    "Anime": { /* ... detail anime ... */ },
    "Episodes": [ /* ... daftar episode ... */ ]
}
```

3. Lihat Daftar Channel
Ambil data semua channel buat dicocokin sama channel_id di data anime.

- Endpoint: GET /api/v1/channels

Contoh Hasilnya:
```json
{
    "UC0wNSTMWIL3qaorLx0abI7A": { /* ... data Ani-One Asia ... */ },
    "UCxxnxya_32jcKj4yN1_kD7A": { /* ... data Muse Indonesia ... */ }
}
```
