package library

import "time"

type Library struct {
	Arcs map[int]*Arc `json:"arcs"`
}

type Arc struct {
	ArcNumber int                `json:"arcNumber"`
	Title     string             `json:"title"`
	Episodes  map[string]Episode `json:"episodes"`
	Monitored bool               `json:"monitored"`
}

// Episode tracks one metadata episode (arc + episode number), which may have
// more than one release version (e.g. "normal" and "extended") imported
// simultaneously. Versions is keyed by that version label.
type Episode struct {
	EpisodeNumber int                       `json:"episodeNumber"`
	Title         string                    `json:"title"`
	Description   string                    `json:"description"`
	Monitored     bool                      `json:"monitored"`
	LastChecked   string                    `json:"lastChecked"`
	Versions      map[string]EpisodeVersion `json:"versions"`
}

// EpisodeVersion tracks a single release version's file, independent of any
// other version of the same episode.
type EpisodeVersion struct {
	CRC32          string `json:"crc32"`
	FilePath       string `json:"filePath"`
	DownloadStatus string `json:"downloadStatus"`
}

// Creates a new empty library
func New() *Library {
	return &Library{
		Arcs: make(map[int]*Arc), // FIXED
	}
}

// Find or create arc
func (l *Library) GetOrCreateArc(arcNumber int, title string) *Arc {

	if arc, ok := l.Arcs[arcNumber]; ok {
		return arc
	}

	newArc := &Arc{
		ArcNumber: arcNumber,
		Title:     title,
		Monitored: true,
		Episodes:  make(map[string]Episode),
	}

	l.Arcs[arcNumber] = newArc
	return newArc
}

func (e *Episode) UpdateLastChecked() {
	e.LastChecked = time.Now().UTC().Format(time.RFC3339)
}
