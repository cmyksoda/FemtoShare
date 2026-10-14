package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cmyksoda/femtoshare/femtoshare"
	"github.com/cmyksoda/femtoshare/store"
	"github.com/cmyksoda/femtoshare/transcode"
)

type (
	ConversionStateFunc func(femtoshare.EntryID) transcode.State

	entryConversionResponse struct {
		State    transcode.State `json:"state"`
		Filename string          `json:"filename,omitempty"`
	}
)

// entryConversionGet is public so guest uploaders can watch their video
// convert, since knowing the entry ID already grants download access.
func (s Server) entryConversionGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := femtoshare.EntryIDFromString(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		entry, err := s.store.GetEntryMetadata(id)
		if _, ok := errors.AsType[store.EntryNotFoundError](err); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", id, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		res := entryConversionResponse{State: transcode.StateReady}
		if s.conversionState != nil {
			res.State = s.conversionState(id)
		}
		// Passphrase-protected filenames stay private, matching downloads.
		if entry.DownloadPassphrase.Empty() || isAuthenticated(r.Context()) {
			res.Filename = string(entry.Filename)
		}
		respondJSON(w, res)
	}
}
