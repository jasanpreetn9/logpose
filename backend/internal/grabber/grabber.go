// Package grabber queues monitored missing episodes for download, giving
// Logpose a Sonarr-style automatic grab after each metadata refresh.
package grabber

import (
	"fmt"
	"log"

	"onepace-library/internal/activity"
	"onepace-library/internal/downloads"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/qbittorrent"
)

type candidate struct {
	crc32 string
	title string
	url   string
}

// GrabWanted queues every monitored episode that is missing from the library
// and not already queued or downloading. Returns the number queued.
func GrabWanted(
	meta *metadata.Client,
	store *library.Store,
	qb *qbittorrent.Client,
	acts *activity.Store,
	tracker *downloads.Tracker,
) int {
	// One candidate per (arc, episode): prefer the normal version, fall
	// back to any version with a download URL.
	type epKey struct{ arc, episode int }
	best := map[epKey]candidate{}
	preferred := map[epKey]bool{}

	store.Read(func(lib *library.Library) {
		for crc, ep := range meta.Episodes() {
			downloadURL := ep.File.DownloadURL()
			if downloadURL == "" {
				continue
			}

			arc := lib.Arcs[ep.Arc] // may be nil
			monitored := arc != nil && arc.Monitored
			imported := false
			if arc != nil {
				if libEp, ok := arc.Episodes[fmt.Sprintf("%d", ep.Episode)]; ok {
					monitored = libEp.Monitored
					if v, ok := libEp.Versions[ep.File.Version]; ok {
						imported = v.DownloadStatus == "imported"
					}
				}
			}
			if !monitored || imported {
				continue
			}
			if tracker.IsActive(crc) {
				continue
			}

			key := epKey{ep.Arc, ep.Episode}
			if preferred[key] {
				continue
			}
			isNormal := ep.File.Version == "normal"
			if _, exists := best[key]; !exists || isNormal {
				best[key] = candidate{crc32: crc, title: ep.Title, url: downloadURL}
				preferred[key] = isNormal
			}
		}
	})

	queued := 0
	for _, c := range best {
		if err := qb.AddTorrent(c.url); err != nil {
			acts.Add(activity.EventDownloadFailed, "Auto-grab failed: "+c.title, err.Error(), false)
			log.Printf("grabber: failed to queue %s: %v", c.crc32, err)
			continue
		}
		tracker.MarkQueued(c.crc32)
		acts.Add(activity.EventDownloadQueued, "Auto-grabbed: "+c.title, c.url, true)
		log.Printf("grabber: queued %s (%s)", c.title, c.crc32)
		queued++
	}
	return queued
}
