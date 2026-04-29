package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
)

type UnifiedEpisode struct {
	Arc         int              `json:"arc"`
	Episode     int              `json:"episode"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Released    string           `json:"released"`
	Downloaded  bool             `json:"downloaded"`
	Monitored   bool             `json:"monitored"`
	Versions    []EpisodeVersion `json:"versions"`
}

type EpisodeVersion struct {
	CRC32    string `json:"crc32"`
	Version  string `json:"version"`
	FilePath string `json:"file_path"`
	Status   string `json:"status"`
}

type UnifiedArc struct {
	Arc               int    `json:"arc"`
	Title             string `json:"title"`
	AudioLanguages    string `json:"audio_languages"`
	SubtitleLanguages string `json:"subtitle_languages"`
	Resolution        string `json:"resolution"`

	MangaChapters    string `json:"manga_chapters"`
	NumberOfChapters string `json:"number_of_chapters"`
	AnimeEpisodes    string `json:"anime_episodes"`
	EpisodesAdapted  string `json:"episodes_adapted"`
	FillerEpisodes   string `json:"filler_episodes"`
	TimeSavedMins    string `json:"time_saved_mins"`
	TimeSavedPercent string `json:"time_saved_percent"`

	Status            string `json:"status"`
	EpisodeCount      int    `json:"episode_count"`
	EpisodeDownloaded int    `json:"episode_downloaded"`

	Episodes []UnifiedEpisode `json:"episodes"`
}

func HandleGetAllEpisodes(meta *metadata.Client, store *library.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		type arcBuild struct {
			arc          *UnifiedArc
			episodeIndex map[int]int
		}
		arcMap := map[int]*arcBuild{}

		episodes := meta.Episodes()

		store.Read(func(lib *library.Library) {
			for crc, ep := range episodes {
				if _, ok := arcMap[ep.Arc]; !ok {
					arcMeta, _ := meta.GetArcByNumber(ep.Arc)
					arcMap[ep.Arc] = &arcBuild{
						arc: &UnifiedArc{
							Arc:               ep.Arc,
							Title:             arcMeta.Title,
							AudioLanguages:    arcMeta.AudioLanguages,
							SubtitleLanguages: arcMeta.SubtitleLanguages,
							Resolution:        arcMeta.Resolution,
							Status:            arcMeta.Status,
							MangaChapters:     arcMeta.MangaChapters,
							NumberOfChapters:  arcMeta.NumberOfChapters,
							AnimeEpisodes:     arcMeta.AnimeEpisodes,
							EpisodesAdapted:   arcMeta.EpisodesAdapted,
							FillerEpisodes:    arcMeta.FillerEpisodes,
							TimeSavedMins:     arcMeta.TimeSavedMins,
							TimeSavedPercent:  arcMeta.TimeSavedPercent,
							Episodes:          []UnifiedEpisode{},
						},
						episodeIndex: map[int]int{},
					}
				}

				build := arcMap[ep.Arc]

				idx, exists := build.episodeIndex[ep.Episode]
				if !exists {
					build.arc.Episodes = append(build.arc.Episodes, UnifiedEpisode{
						Arc:         ep.Arc,
						Episode:     ep.Episode,
						Title:       ep.Title,
						Description: ep.Description,
						Released:    ep.Released,
						Versions:    []EpisodeVersion{},
					})
					idx = len(build.arc.Episodes) - 1
					build.episodeIndex[ep.Episode] = idx
				}

				existing := &build.arc.Episodes[idx]

				version := EpisodeVersion{
					CRC32:   crc,
					Version: ep.File.Version,
					Status:  "missing",
				}

				if arcLib, ok := lib.Arcs[ep.Arc]; ok {
					key := fmt.Sprintf("%d", ep.Episode)
					if libEp, ok := arcLib.Episodes[key]; ok {
						// Monitored is an episode-level flag — set it regardless of version.
						existing.Monitored = libEp.Monitored
						if libEp.CRC32 == crc {
							// This is the version currently in the library.
							version.FilePath = libEp.FilePath
							version.Status = libEp.DownloadStatus
						} else if libEp.DownloadStatus == "imported" {
							// A different version of an already-imported episode — offer as upgrade.
							version.Status = "upgradable"
						}
					}
				}

				existing.Versions = append(existing.Versions, version)
			}
		})

		result := make([]UnifiedArc, 0, len(arcMap))
		for _, build := range arcMap {
			arc := build.arc
			arc.EpisodeCount = len(arc.Episodes)
			for i := range arc.Episodes {
				ep := &arc.Episodes[i]
				for _, v := range ep.Versions {
					if v.Status != "missing" {
						ep.Downloaded = true
						break
					}
				}
				if ep.Downloaded {
					arc.EpisodeDownloaded++
				}
			}
			sort.Slice(arc.Episodes, func(i, j int) bool {
				return arc.Episodes[i].Episode < arc.Episodes[j].Episode
			})
			result = append(result, *arc)
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].Arc < result[j].Arc
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
