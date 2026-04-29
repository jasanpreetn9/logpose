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

type Episode struct {
	EpisodeNumber  int    `json:"episodeNumber"`
	CRC32          string `json:"crc32"`
	Version        string `json:"version"`
	FilePath       string `json:"filePath"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	DownloadStatus string `json:"downloadStatus"`
	Monitored      bool   `json:"monitored"`
	LastChecked    string `json:"lastChecked"`
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
