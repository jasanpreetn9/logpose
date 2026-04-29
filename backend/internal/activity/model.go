package activity

import "time"

type EventType string

const (
	EventDownloadQueued    EventType = "download_queued"
	EventDownloadFailed    EventType = "download_failed"
	EventLibraryScan       EventType = "library_scan"
	EventDownloadsScan     EventType = "downloads_scan"
	EventImport            EventType = "import"
)

type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Success   bool      `json:"success"`
}
