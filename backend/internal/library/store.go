package library

import (
	"database/sql"
	"errors"
	"log"
	"sync"

	"onepace-library/internal/db"
)

// Store is a thread-safe in-memory library backed by SQLite.
// The in-memory lib is the fast read path; every Write is persisted to the DB.
type Store struct {
	mu     sync.RWMutex
	scanMu sync.Mutex
	lib    *Library
	path   string // kept so Save() can still write a JSON export
	db     *db.DB
}

func NewStore(jsonPath string, d *db.DB) *Store {
	lib, err := loadFromDB(d)
	if err != nil {
		log.Printf("Failed to load library from DB: %v — starting empty", err)
		lib = New()
	}
	return &Store{lib: lib, path: jsonPath, db: d}
}

// Read calls fn with a read lock held. Do not retain the *Library pointer outside fn.
func (s *Store) Read(fn func(*Library)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s.lib)
}

// Write calls fn with a write lock held, then persists to SQLite.
func (s *Store) Write(fn func(*Library) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(s.lib); err != nil {
		return err
	}
	return s.persist()
}

// TryScan acquires the scan mutex (non-blocking) then runs fn under a write lock.
func (s *Store) TryScan(fn func(*Library) error) error {
	if !s.scanMu.TryLock() {
		return errors.New("scan already in progress")
	}
	defer s.scanMu.Unlock()
	return s.Write(fn)
}

// persist writes the full in-memory library to SQLite in one transaction.
// Must be called with s.mu held.
func (s *Store) persist() error {
	return s.db.Tx(func(tx *sql.Tx) error {
		for _, arc := range s.lib.Arcs {
			if err := s.db.SaveArc(tx, arc.ArcNumber, arc.Title, arc.Monitored); err != nil {
				return err
			}
			for _, ep := range arc.Episodes {
				if err := s.db.UpsertEpisode(tx, episodeToRow(arc.ArcNumber, ep)); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ── DB ↔ domain conversions ──────────────────────────────────────────────────

func loadFromDB(d *db.DB) (*Library, error) {
	arcRows, err := d.GetAllArcs()
	if err != nil {
		return nil, err
	}
	epRows, err := d.GetAllEpisodes()
	if err != nil {
		return nil, err
	}

	lib := New()
	for _, a := range arcRows {
		lib.Arcs[a.ArcNumber] = &Arc{
			ArcNumber: a.ArcNumber,
			Title:     a.Title,
			Monitored: a.Monitored,
			Episodes:  map[string]Episode{},
		}
	}
	for arcNum, eps := range epRows {
		if _, ok := lib.Arcs[arcNum]; !ok {
			lib.Arcs[arcNum] = &Arc{
				ArcNumber: arcNum,
				Monitored: true,
				Episodes:  map[string]Episode{},
			}
		}
		for key, r := range eps {
			lib.Arcs[arcNum].Episodes[key] = episodeFromRow(r)
		}
	}
	return lib, nil
}

func episodeFromRow(r db.EpisodeRow) Episode {
	return Episode{
		EpisodeNumber:  r.EpisodeNumber,
		CRC32:          r.CRC32,
		Version:        r.Version,
		FilePath:       r.FilePath,
		Title:          r.Title,
		Description:    r.Description,
		DownloadStatus: r.DownloadStatus,
		Monitored:      r.Monitored,
		LastChecked:    r.LastChecked,
	}
}

func episodeToRow(arcNum int, ep Episode) db.EpisodeRow {
	return db.EpisodeRow{
		ArcNumber:      arcNum,
		EpisodeNumber:  ep.EpisodeNumber,
		CRC32:          ep.CRC32,
		Version:        ep.Version,
		FilePath:       ep.FilePath,
		Title:          ep.Title,
		Description:    ep.Description,
		DownloadStatus: ep.DownloadStatus,
		Monitored:      ep.Monitored,
		LastChecked:    ep.LastChecked,
	}
}
