// Package watcher monitors the downloads directory with fsnotify and triggers
// an import scan after a 2-second debounce when new video files appear.
package watcher

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounce = 2 * time.Second

// Start watches downloadPath and calls scanFn after each debounced burst of
// Create events for .mkv/.mp4 files. Runs until ctx is cancelled.
func Start(ctx context.Context, downloadPath string, scanFn func()) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	if err := w.Add(downloadPath); err != nil {
		return err
	}

	log.Printf("watcher: watching %s", downloadPath)

	var timer *time.Timer

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return nil

		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Create == 0 {
				continue
			}
			name := strings.ToLower(event.Name)
			if !strings.HasSuffix(name, ".mkv") && !strings.HasSuffix(name, ".mp4") {
				continue
			}
			// Reset the debounce timer on each relevant event.
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounce, func() {
				log.Printf("watcher: new file detected, triggering scan")
				scanFn()
			})

		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher: %v", err)
		}
	}
}
