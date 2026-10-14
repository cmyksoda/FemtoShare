package sqlite_test

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cmyksoda/femtoshare/femtoshare"
	"github.com/cmyksoda/femtoshare/store/sqlite"
)

const largeFileBytes = 8 << 20

func TestReplaceEntryFileShrinksDatabaseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	dataStore := sqlite.New(sqlite.Params{Path: path, Now: time.Now})
	mustInsertLargeEntry(t, dataStore, path)

	if err := dataStore.ReplaceEntryFile(femtoshare.MustCreateEntryID("abcdefghij"), strings.NewReader("small"), "video/mp4", "clip.mp4"); err != nil {
		t.Fatalf("failed to replace entry file: %v", err)
	}

	if got := databaseBytesOnDisk(t, path); got > largeFileBytes/4 {
		t.Errorf("database uses %d bytes on disk after replacing a large file, want < %d", got, largeFileBytes/4)
	}
}

func TestDeleteEntryShrinksDatabaseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	dataStore := sqlite.New(sqlite.Params{Path: path, Now: time.Now})
	mustInsertLargeEntry(t, dataStore, path)

	if err := dataStore.DeleteEntry(femtoshare.MustCreateEntryID("abcdefghij")); err != nil {
		t.Fatalf("failed to delete entry: %v", err)
	}

	if got := databaseBytesOnDisk(t, path); got > largeFileBytes/4 {
		t.Errorf("database uses %d bytes on disk after deleting a large file, want < %d", got, largeFileBytes/4)
	}
}

func TestNewEnablesIncrementalVacuumOnExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE legacy (value TEXT)`); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}

	sqlite.New(sqlite.Params{Path: path, Now: time.Now})

	if info, err := os.Stat(path + "-wal"); err == nil && info.Size() != 0 {
		t.Errorf("WAL is %d bytes after compacting, want 0", info.Size())
	}

	db, err = sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}
	defer db.Close()

	var mode int
	if err := db.QueryRow(`PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		t.Fatalf("failed to read auto_vacuum mode: %v", err)
	}
	if got, want := mode, 2; got != want {
		t.Errorf("auto_vacuum=%d, want=%d (incremental)", got, want)
	}
}

func mustInsertLargeEntry(t *testing.T, dataStore sqlite.Store, path string) {
	t.Helper()
	data := bytes.Repeat([]byte("x"), largeFileBytes)
	if err := dataStore.InsertEntry(bytes.NewReader(data), femtoshare.UploadMetadata{
		ID:          femtoshare.MustCreateEntryID("abcdefghij"),
		Filename:    "clip.mov",
		ContentType: "video/quicktime",
		Uploaded:    mustParseTime("2025-05-25T00:00:00Z"),
		Expires:     mustParseExpirationTime("2040-01-01T00:00:00Z"),
		Size:        mustParseFileSize(len(data)),
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}
	if got := databaseBytesOnDisk(t, path); got < largeFileBytes {
		t.Fatalf("database uses %d bytes on disk after inserting a large file, want >= %d", got, largeFileBytes)
	}
}

// databaseBytesOnDisk includes the WAL because recent writes may not have been
// checkpointed into the main file yet.
func databaseBytesOnDisk(t *testing.T, path string) int64 {
	t.Helper()
	var total int64
	for _, p := range []string{path, path + "-wal"} {
		info, err := os.Stat(p)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Fatalf("failed to stat %s: %v", p, err)
		}
		total += info.Size()
	}
	return total
}
