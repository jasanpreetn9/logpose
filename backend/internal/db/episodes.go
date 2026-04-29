package db

import (
	"database/sql"
	"fmt"
)

type EpisodeRow struct {
	ArcNumber      int
	EpisodeNumber  int
	CRC32          string
	Version        string
	FilePath       string
	Title          string
	Description    string
	DownloadStatus string
	Monitored      bool
	LastChecked    string
}

func (d *DB) UpsertEpisode(tx *sql.Tx, r EpisodeRow) error {
	return upsertEpTx(tx, r.ArcNumber, r.EpisodeNumber, r.CRC32, r.Version,
		r.FilePath, r.Title, r.Description, r.DownloadStatus, r.Monitored, r.LastChecked)
}

func upsertEpTx(tx *sql.Tx, arcNum, epNum int, crc32, version, filePath, title, description, downloadStatus string, monitored bool, lastChecked string) error {
	_, err := tx.Exec(`
		INSERT INTO episodes(arc_number,episode_number,crc32,version,file_path,title,description,download_status,monitored,last_checked)
		VALUES(?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(arc_number,episode_number) DO UPDATE SET
			crc32           = excluded.crc32,
			version         = excluded.version,
			file_path       = excluded.file_path,
			title           = excluded.title,
			description     = excluded.description,
			download_status = excluded.download_status,
			monitored       = excluded.monitored,
			last_checked    = excluded.last_checked`,
		arcNum, epNum, crc32, version, filePath, title, description, downloadStatus, boolInt(monitored), lastChecked,
	)
	return err
}

// GetAllEpisodes returns all episode rows keyed by arcNumber → episodeKey (string of episode_number).
func (d *DB) GetAllEpisodes() (map[int]map[string]EpisodeRow, error) {
	rows, err := d.SQL.Query(`
		SELECT arc_number,episode_number,crc32,version,file_path,title,description,download_status,monitored,last_checked
		FROM episodes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[int]map[string]EpisodeRow{}
	for rows.Next() {
		var r EpisodeRow
		var mon int
		if err := rows.Scan(&r.ArcNumber, &r.EpisodeNumber, &r.CRC32, &r.Version,
			&r.FilePath, &r.Title, &r.Description, &r.DownloadStatus, &mon, &r.LastChecked); err != nil {
			return nil, err
		}
		r.Monitored = mon == 1
		if result[r.ArcNumber] == nil {
			result[r.ArcNumber] = map[string]EpisodeRow{}
		}
		result[r.ArcNumber][fmt.Sprintf("%d", r.EpisodeNumber)] = r
	}
	return result, rows.Err()
}
