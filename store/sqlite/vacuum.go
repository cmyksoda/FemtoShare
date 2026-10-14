package sqlite

import (
	"database/sql"
	"log"
)

const autoVacuumIncremental = 2

// enableIncrementalVacuum lets the database file shrink after deletions. An
// existing database only switches modes after one full VACUUM.
func enableIncrementalVacuum(db *sql.DB) {
	var mode int
	if err := db.QueryRow(`PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		log.Fatalf("failed to read auto_vacuum mode: %v", err)
	}
	if mode == autoVacuumIncremental {
		return
	}

	log.Printf("compacting database to enable incremental vacuuming, which may take a while")
	// Both statements must run on the same connection for the mode to stick.
	if _, err := db.Exec(`
		PRAGMA auto_vacuum = INCREMENTAL;
		VACUUM;
		`); err != nil {
		log.Fatalf("failed to enable incremental vacuuming: %v", err)
	}
}

// reclaimFreeSpace returns pages freed by deleted file data to the filesystem.
func (s Store) reclaimFreeSpace() {
	// incremental_vacuum frees one page per step, so drain every row.
	rows, err := s.db.Query(`PRAGMA incremental_vacuum`)
	if err != nil {
		log.Printf("failed to vacuum database: %v", err)
		return
	}
	for rows.Next() {
	}
	if err := rows.Err(); err != nil {
		log.Printf("failed to vacuum database: %v", err)
	}
	rows.Close()

	// In WAL mode, the file only shrinks at a checkpoint, but Litestream must
	// own checkpoints.
	if s.optimizeForLitestream {
		return
	}
	if _, err := s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		log.Printf("failed to checkpoint database: %v", err)
	}
}
