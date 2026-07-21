package metadata

// Episodes.json entry
type Episode struct {
	Arc         int         `json:"arc"`
	Episode     int         `json:"episode"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Chapters    string      `json:"chapters"`
	EpisodesRef string      `json:"episodes"`
	Released    string      `json:"released"`
	File        EpisodeFile `json:"file"`
}

// File inside episodes.json
type EpisodeFile struct {
	Version    string `json:"version"`
	CRC32      string `json:"crc32"`
	Length     string `json:"length"`
	URL        string `json:"url"`
	MagnetURI  string `json:"magnet_uri"`
	TorrentURL string `json:"torrent_url"`
}

// DownloadURL returns the best available link for handing to qBittorrent:
// a magnet URI first (no tracker/host dependency), then a direct .torrent
// URL, falling back to the legacy Nyaa page/search URL.
func (f EpisodeFile) DownloadURL() string {
	if f.MagnetURI != "" {
		return f.MagnetURI
	}
	if f.TorrentURL != "" {
		return f.TorrentURL
	}
	return f.URL
}

type Arc struct {
	ArcNumber         int    `json:"arc"`
	Title             string `json:"title"`
	AudioLanguages    string `json:"audio_languages"`
	SubtitleLanguages string `json:"subtitle_languages"`
	Resolution        string `json:"resolution"`
	MangaChapters     string `json:"manga_chapters"`
	NumberOfChapters  string `json:"number_of_chapters"`
	AnimeEpisodes     string `json:"anime_episodes"`
	EpisodesAdapted   string `json:"episodes_adapted"`
	FillerEpisodes    string `json:"filler_episodes"`
	TimeSavedMins     string `json:"time_saved_mins"`
	TimeSavedPercent  string `json:"time_saved_percent"`
	Status            string `json:"status"`
}
