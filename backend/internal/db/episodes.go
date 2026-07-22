package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// VersionRow is one release version's file state, stored as part of an
// episode row's versions_json column.
type VersionRow struct {
	CRC32          string `json:"crc32"`
	FilePath       string `json:"filePath"`
	DownloadStatus string `json:"downloadStatus"`
}

// EpisodeRow is one (arc, episode) row. Versions holds every release version
// tracked for that episode (e.g. "normal", "extended"), so importing one
// version never overwrites another.
type EpisodeRow struct {
	ArcNumber     int
	EpisodeNumber int
	Title         string
	Description   string
	Monitored     bool
	LastChecked   string
	Versions      map[string]VersionRow
}

func (d *DB) UpsertEpisode(tx *sql.Tx, r EpisodeRow) error {
	if r.Versions == nil {
		r.Versions = map[string]VersionRow{}
	}
	versionsJSON, err := json.Marshal(r.Versions)
	if err != nil {
		return fmt.Errorf("marshal versions: %w", err)
	}
	// crc32/version/file_path/download_status are legacy columns from the
	// pre-versions_json schema, kept only because older databases have them
	// as NOT NULL with no default — write empty placeholders so inserts
	// against those databases don't fail. versions_json is the source of truth.
	_, err = tx.Exec(`
		INSERT INTO episodes(arc_number,episode_number,crc32,version,file_path,download_status,title,description,monitored,last_checked,versions_json)
		VALUES(?,?,'','','','',?,?,?,?,?)
		ON CONFLICT(arc_number,episode_number) DO UPDATE SET
			title         = excluded.title,
			description   = excluded.description,
			monitored     = excluded.monitored,
			last_checked  = excluded.last_checked,
			versions_json = excluded.versions_json`,
		r.ArcNumber, r.EpisodeNumber, r.Title, r.Description, boolInt(r.Monitored), r.LastChecked, string(versionsJSON),
	)
	return err
}

// GetAllEpisodes returns all episode rows keyed by arcNumber → episodeKey (string of episode_number).
func (d *DB) GetAllEpisodes() (map[int]map[string]EpisodeRow, error) {
	rows, err := d.SQL.Query(`
		SELECT arc_number,episode_number,title,description,monitored,last_checked,versions_json
		FROM episodes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[int]map[string]EpisodeRow{}
	for rows.Next() {
		var r EpisodeRow
		var mon int
		var versionsJSON string
		if err := rows.Scan(&r.ArcNumber, &r.EpisodeNumber, &r.Title, &r.Description,
			&mon, &r.LastChecked, &versionsJSON); err != nil {
			return nil, err
		}
		r.Monitored = mon == 1
		r.Versions = map[string]VersionRow{}
		if versionsJSON != "" {
			if err := json.Unmarshal([]byte(versionsJSON), &r.Versions); err != nil {
				return nil, fmt.Errorf("unmarshal versions for arc %d ep %d: %w", r.ArcNumber, r.EpisodeNumber, err)
			}
		}
		if result[r.ArcNumber] == nil {
			result[r.ArcNumber] = map[string]EpisodeRow{}
		}
		result[r.ArcNumber][fmt.Sprintf("%d", r.EpisodeNumber)] = r
	}
	return result, rows.Err()
}
