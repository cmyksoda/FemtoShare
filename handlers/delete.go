package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cmyksoda/femtoshare/femtoshare"
)

func (s Server) entryDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := femtoshare.EntryIDFromString(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		err = s.store.DeleteEntry(id)
		if err != nil {
			log.Printf("failed to delete entry %v: %v", id, err)
			http.Error(w, "failed to delete entry", http.StatusInternalServerError)
			return
		}
	}
}
