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
	Version string `json:"version"`
	CRC32   string `json:"crc32"`
	Length  string `json:"length"`
	URL     string `json:"url"`
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
