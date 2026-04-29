// Package db provides SQLite persistence for the library and activity stores.
// It deliberately imports no domain packages (library, activity) to avoid
// import cycles — callers are responsible for converting between row types
// and their domain models.
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite connection.
type DB struct {
	SQL *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS arcs (
  arc_number INTEGER PRIMARY KEY,
  title      TEXT    NOT NULL,
  monitored  INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS episodes (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  arc_number      INTEGER NOT NULL,
  episode_number  INTEGER NOT NULL,
  crc32           TEXT    NOT NULL,
  version         TEXT    NOT NULL DEFAULT '',
  file_path       TEXT    NOT NULL DEFAULT '',
  title           TEXT    NOT NULL DEFAULT '',
  description     TEXT    NOT NULL DEFAULT '',
  download_status TEXT    NOT NULL DEFAULT 'missing',
  monitored       INTEGER NOT NULL DEFAULT 1,
  last_checked    TEXT    NOT NULL DEFAULT '',
  UNIQUE(arc_number, episode_number)
);

CREATE TABLE IF NOT EXISTS events (
  id         INTEGER  PRIMARY KEY AUTOINCREMENT,
  type       TEXT     NOT NULL,
  message    TEXT     NOT NULL,
  payload    TEXT     NOT NULL DEFAULT '',
  success    INTEGER  NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite with WAL mode needs a single writer connection.
	sqlDB.SetMaxOpenConns(1)

	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := sqlDB.Exec(schema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	d := &DB{SQL: sqlDB}

	if err := d.migrateFromJSON(jsonSiblingPath(path)); err != nil {
		log.Printf("library.json migration skipped: %v", err)
	}

	return d, nil
}

func (d *DB) Close() error { return d.SQL.Close() }

// Tx runs fn inside a BEGIN IMMEDIATE transaction.
func (d *DB) Tx(fn func(tx *sql.Tx) error) error {
	tx, err := d.SQL.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// jsonSiblingPath derives a library.json path sitting next to the db file.
func jsonSiblingPath(dbPath string) string {
	if dbPath == ":memory:" || dbPath == "" {
		return ""
	}
	idx := strings.LastIndex(dbPath, "/")
	if idx < 0 {
		return "library.json"
	}
	return dbPath[:idx+1] + "library.json"
}

// ── migration ────────────────────────────────────────────────────────────────

// localLibrary mirrors library.Library JSON without importing the library package.
type localLibrary struct {
	Arcs map[int]*localArc `json:"arcs"`
}

type localArc struct {
	ArcNumber int                    `json:"arcNumber"`
	Title     string                 `json:"title"`
	Monitored bool                   `json:"monitored"`
	Episodes  map[string]localEpisode `json:"episodes"`
}

type localEpisode struct {
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

func (d *DB) migrateFromJSON(jsonPath string) error {
	if jsonPath == "" {
		return nil
	}
	if _, err := os.Stat(jsonPath); err != nil {
		return nil
	}

	var count int
	if err := d.SQL.QueryRow("SELECT COUNT(*) FROM episodes").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // already migrated
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}

	var lib localLibrary
	if err := json.Unmarshal(data, &lib); err != nil {
		return fmt.Errorf("parse library.json: %w", err)
	}

	return d.Tx(func(tx *sql.Tx) error {
		for arcNum, arc := range lib.Arcs {
			mon := boolInt(arc.Monitored)
			if _, err := tx.Exec(
				`INSERT OR REPLACE INTO arcs(arc_number,title,monitored) VALUES(?,?,?)`,
				arcNum, arc.Title, mon,
			); err != nil {
				return err
			}
			for _, ep := range arc.Episodes {
				if err := upsertEpTx(tx, arcNum, ep.EpisodeNumber, ep.CRC32, ep.Version,
					ep.FilePath, ep.Title, ep.Description, ep.DownloadStatus, ep.Monitored, ep.LastChecked); err != nil {
					return err
				}
			}
		}
		log.Printf("Migrated library.json → SQLite (%d arcs)", len(lib.Arcs))
		return nil
	})
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
