package db

import "database/sql"

type ArcRow struct {
	ArcNumber int
	Title     string
	Monitored bool
}

func (d *DB) SaveArc(tx *sql.Tx, arcNumber int, title string, monitored bool) error {
	_, err := tx.Exec(
		`INSERT OR REPLACE INTO arcs(arc_number,title,monitored) VALUES(?,?,?)`,
		arcNumber, title, boolInt(monitored),
	)
	return err
}

func (d *DB) GetAllArcs() ([]ArcRow, error) {
	rows, err := d.SQL.Query(`SELECT arc_number,title,monitored FROM arcs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var arcs []ArcRow
	for rows.Next() {
		var a ArcRow
		var mon int
		if err := rows.Scan(&a.ArcNumber, &a.Title, &mon); err != nil {
			return nil, err
		}
		a.Monitored = mon == 1
		arcs = append(arcs, a)
	}
	return arcs, rows.Err()
}
