// Package transcode converts uploaded videos into a format that every major
// browser can play.
package transcode

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
)

type (
	State string

	Store interface {
		GetEntriesMetadata() ([]picoshare.UploadMetadata, error)
		GetEntryMetadata(picoshare.EntryID) (picoshare.UploadMetadata, error)
		ReadEntryFile(picoshare.EntryID) (io.ReadSeeker, error)
		ReplaceEntryFile(id picoshare.EntryID, reader io.Reader, contentType picoshare.ContentType, filename picoshare.Filename) error
	}

	Queue struct {
		store   Store
		pending chan picoshare.EntryID
		mu      sync.Mutex
		states  map[picoshare.EntryID]State
	}
)

const (
	StatePending    State = "pending"
	StateConverting State = "converting"
	StateReady      State = "ready"
	StateFailed     State = "failed"
)

const outputContentType = picoshare.ContentType("video/mp4")

// NewQueue starts a background worker that converts videos one at a time so
// a burst of uploads can't starve the server of CPU.
func NewQueue(store Store) *Queue {
	q := &Queue{
		store:   store,
		pending: make(chan picoshare.EntryID, 1000),
		states:  map[picoshare.EntryID]State{},
	}
	go q.run()
	return q
}

func IsVideo(contentType picoshare.ContentType) bool {
	return strings.HasPrefix(string(contentType), "video/")
}

func (q *Queue) Enqueue(id picoshare.EntryID) {
	q.setState(id, StatePending)
	select {
	case q.pending <- id:
	default:
		log.Printf("transcode queue is full, leaving entry %s as uploaded", id)
		q.setState(id, StateReady)
	}
}

func (q *Queue) EnqueueExistingVideos() error {
	entries, err := q.store.GetEntriesMetadata()
	if err != nil {
		return fmt.Errorf("failed to list entries: %w", err)
	}
	for _, entry := range entries {
		if IsVideo(entry.ContentType) {
			q.Enqueue(entry.ID)
		}
	}
	return nil
}

// State reports the conversion progress of an entry. Entries that were never
// queued are ready as uploaded.
func (q *Queue) State(id picoshare.EntryID) State {
	q.mu.Lock()
	defer q.mu.Unlock()
	if state, ok := q.states[id]; ok {
		return state
	}
	return StateReady
}

func (q *Queue) setState(id picoshare.EntryID, state State) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if state == StateReady {
		delete(q.states, id)
	} else {
		q.states[id] = state
	}
}

func (q *Queue) run() {
	for id := range q.pending {
		if err := q.convertEntry(id); err != nil {
			log.Printf("failed to convert entry %s: %v", id, err)
			q.setState(id, StateFailed)
			continue
		}
		q.setState(id, StateReady)
	}
}

func (q *Queue) convertEntry(id picoshare.EntryID) error {
	metadata, err := q.store.GetEntryMetadata(id)
	if _, ok := errors.AsType[store.EntryNotFoundError](err); ok {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to read entry metadata: %w", err)
	}

	dir, err := os.MkdirTemp("", "picoshare-transcode-")
	if err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("failed to remove transcode working directory: %v", err)
		}
	}()

	inputPath := filepath.Join(dir, "input")
	if err := q.copyEntryToFile(id, inputPath); err != nil {
		return err
	}

	probe, err := probeVideo(inputPath)
	if err != nil {
		return err
	}
	if !probe.needsConversion(metadata.ContentType) {
		return nil
	}

	log.Printf("converting %s (%s) into a browser-friendly video", id, metadata.Filename)
	q.setState(id, StateConverting)

	outputPath := filepath.Join(dir, "output.mp4")
	if output, err := exec.Command("ffmpeg", probe.ffmpegArgs(inputPath, outputPath)...).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run ffmpeg: %w: %s", err, output)
	}

	output, err := os.Open(outputPath)
	if err != nil {
		return fmt.Errorf("failed to open converted video: %w", err)
	}
	defer output.Close()

	err = q.store.ReplaceEntryFile(id, output, outputContentType, mp4Filename(metadata.Filename))
	if _, ok := errors.AsType[store.EntryNotFoundError](err); ok {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to store converted video: %w", err)
	}

	log.Printf("finished converting %s", id)
	return nil
}

func (q *Queue) copyEntryToFile(id picoshare.EntryID, path string) error {
	reader, err := q.store.ReadEntryFile(id)
	if err != nil {
		return fmt.Errorf("failed to read entry file: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create input file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("failed to copy entry to disk: %w", err)
	}
	return f.Close()
}

func mp4Filename(filename picoshare.Filename) picoshare.Filename {
	name := string(filename)
	return picoshare.Filename(strings.TrimSuffix(name, filepath.Ext(name)) + ".mp4")
}
