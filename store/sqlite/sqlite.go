package sqlite

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/cmyksoda/femtoshare/femtoshare"
)

const (
	timeFormat = time.RFC3339
	// I think Chrome reads in 32768 chunks, but I haven't checked rigorously.
	defaultChunkSize = uint64(32768 * 10)
)

type (
	Params struct {
		Path                  string
		OptimizeForLitestream bool
		Now                   func() time.Time
		FemtoShareChunkSize   uint64
	}

	Store struct {
		db                    *sql.DB
		chunkSize             uint64
		optimizeForLitestream bool
		now                   func() time.Time
	}

	rowScanner interface {
		Scan(...any) error
	}
)

func New(params Params) Store {
	log.Printf("reading DB from %s", params.Path)
	db, err := sql.Open("sqlite3", params.Path)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := db.Exec(`
		PRAGMA temp_store = FILE;
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = 1;
		`); err != nil {
		log.Fatalf("failed to set pragmas: %v", err)
	}

	if params.OptimizeForLitestream {
		if _, err := db.Exec(`
			-- Apply Litestream recommendations: https://litestream.io/tips/
			PRAGMA busy_timeout = 5000;
			PRAGMA synchronous = NORMAL;
			PRAGMA wal_autocheckpoint = 0;
				`); err != nil {
			log.Fatalf("failed to set Litestream compatibility pragmas: %v", err)
		}
	}

	applyMigrations(db)

	chunkSize := params.FemtoShareChunkSize
	if chunkSize == 0 {
		chunkSize = defaultChunkSize
	}

	s := Store{
		db:                    db,
		chunkSize:             chunkSize,
		optimizeForLitestream: params.OptimizeForLitestream,
		now:                   params.Now,
	}
	s.enableIncrementalVacuum()
	return s
}

func formatExpirationTime(et femtoshare.ExpirationTime) string {
	return formatTime(time.Time(et))
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func formatFileLifetime(lt femtoshare.FileLifetime) string {
	return lt.String()
}

func parseDatetime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}

func parseFileLifetime(s string) (femtoshare.FileLifetime, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return femtoshare.FileLifetime{}, err
	}
	return femtoshare.NewFileLifetimeFromDuration(d)
}
