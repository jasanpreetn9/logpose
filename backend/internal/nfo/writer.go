package nfo

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
)

type EpisodeNFO struct {
	XMLName xml.Name `xml:"episodedetails"`

	Title       string `xml:"title"`
	Plot        string `xml:"plot"`
	Season      int    `xml:"season"`
	Episode     int    `xml:"episode"`
	Aired       string `xml:"aired,omitempty"`
	UniqueID    string `xml:"uniqueid,omitempty"`
	ShowTitle   string `xml:"showtitle,omitempty"`
	OriginalURL string `xml:"originalurl,omitempty"`
}

func GenerateEpisodeNFO(ep library.Episode, meta metadata.Episode, arcTitle string, outputPath string) error {

	nfo := EpisodeNFO{
		Title:       ep.Title,
		Plot:        ep.Description,
		Season:      meta.Arc,     // One Pace arcs = seasons
		Episode:     meta.Episode, // Part number
		Aired:       meta.Released,
		UniqueID:    ep.CRC32, // identifies specific version
		ShowTitle:   arcTitle,
		OriginalURL: "", // optional: source URL
	}

	return writeNFOFile(outputPath, nfo)
}

func writeNFOFile(path string, data EpisodeNFO) error {
	b, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal xml: %w", err)
	}

	// Add XML header
	xmlOutput := []byte(xml.Header + string(b))

	return os.WriteFile(path, xmlOutput, 0644)
}

// Create NFO filename from video file
func NFOPathForVideo(videoPath string) string {
	base := videoPath[:len(videoPath)-len(filepath.Ext(videoPath))]
	return base + ".nfo"
}
