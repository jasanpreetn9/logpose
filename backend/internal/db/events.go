package db

import (
	"fmt"
	"time"
)

type EventRow struct {
	ID        int64
	Type      string
	Message   string
	Payload   string
	Success   bool
	CreatedAt time.Time
}

func (d *DB) InsertEvent(typ, message, payload string, success bool) error {
	_, err := d.SQL.Exec(
		`INSERT INTO events(type,message,payload,success,created_at) VALUES(?,?,?,?,?)`,
		typ, message, payload, boolInt(success), time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (d *DB) ListEvents(limit int) ([]EventRow, error) {
	return d.queryEvents(`SELECT id,type,message,payload,success,created_at FROM events ORDER BY id DESC LIMIT ?`, limit)
}

func (d *DB) ListEventsByType(typ string, limit int) ([]EventRow, error) {
	return d.queryEvents(`SELECT id,type,message,payload,success,created_at FROM events WHERE type=? ORDER BY id DESC LIMIT ?`, typ, limit)
}

func (d *DB) queryEvents(query string, args ...any) ([]EventRow, error) {
	rows, err := d.SQL.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []EventRow
	for rows.Next() {
		var r EventRow
		var suc int
		var tsStr string
		if err := rows.Scan(&r.ID, &r.Type, &r.Message, &r.Payload, &suc, &tsStr); err != nil {
			return nil, err
		}
		r.Success = suc == 1
		if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			r.CreatedAt = t
		}
		events = append(events, r)
	}
	return events, rows.Err()
}

func EventRowID(r EventRow) string { return fmt.Sprintf("%d", r.ID) }
