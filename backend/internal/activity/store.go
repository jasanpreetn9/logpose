package activity

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"onepace-library/internal/db"
	"onepace-library/internal/sse"
)

const maxEvents = 500

type Store struct {
	mu     sync.RWMutex
	events []Event
	seq    atomic.Int64
	db     *db.DB
	hub    *sse.Hub // may be nil
}

func NewStore(d *db.DB, hub *sse.Hub) *Store {
	return &Store{
		events: make([]Event, 0, 64),
		db:     d,
		hub:    hub,
	}
}

// LoadFromDB pre-populates the in-memory slice from the DB.
// Call once at startup before serving requests.
func (s *Store) LoadFromDB(limit int) {
	rows, err := s.db.ListEvents(limit)
	if err != nil {
		log.Printf("activity.LoadFromDB: %v", err)
		return
	}
	events := make([]Event, 0, len(rows))
	for _, r := range rows {
		events = append(events, eventFromRow(r))
	}
	s.mu.Lock()
	s.events = events
	if len(events) > 0 {
		s.seq.Store(rows[len(rows)-1].ID) // set seq to highest known id
	}
	s.mu.Unlock()
}

func (s *Store) Add(t EventType, message, details string, success bool) {
	id := s.seq.Add(1)
	ev := Event{
		ID:        fmt.Sprintf("%d", id),
		Type:      t,
		Timestamp: time.Now().UTC(),
		Message:   message,
		Details:   details,
		Success:   success,
	}

	s.mu.Lock()
	s.events = append([]Event{ev}, s.events...)
	if len(s.events) > maxEvents {
		s.events = s.events[:maxEvents]
	}
	s.mu.Unlock()

	// Persist to DB (best-effort — don't block callers on DB errors).
	if err := s.db.InsertEvent(string(t), message, details, success); err != nil {
		log.Printf("activity: InsertEvent: %v", err)
	}

	// Broadcast via SSE.
	if s.hub != nil {
		if b, err := json.Marshal(ev); err == nil {
			s.hub.Broadcast(string(b))
		}
	}
}

func (s *Store) All() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Event, len(s.events))
	copy(result, s.events)
	return result
}

func (s *Store) ByType(t EventType) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Event
	for _, ev := range s.events {
		if ev.Type == t {
			result = append(result, ev)
		}
	}
	return result
}

func eventFromRow(r db.EventRow) Event {
	return Event{
		ID:        db.EventRowID(r),
		Type:      EventType(r.Type),
		Timestamp: r.CreatedAt,
		Message:   r.Message,
		Details:   r.Payload,
		Success:   r.Success,
	}
}
