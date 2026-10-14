package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/cmyksoda/femtoshare/femtoshare"
	"github.com/cmyksoda/femtoshare/handlers/parse"
	"github.com/cmyksoda/femtoshare/random"
)

const (
	GuestLinkIDLength         = 16
	GuestLinkByteLimitMinimum = 1024 * 1024
)

type GuestLinkPostResponse struct {
	ID string `json:"id"`
}

// Omit visually similar characters (I,l,1), (0,O)
var guestLinkIDCharacters = []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789")

func (s Server) guestLinksPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gl, err := s.guestLinkFromRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		gl.ID = generateGuestLinkID()
		gl.Created = s.now()

		if err := s.store.InsertGuestLink(gl); err != nil {
			log.Printf("failed to save guest link: %v", err)
			http.Error(w, fmt.Sprintf("Failed to save guest link: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, GuestLinkPostResponse{ID: gl.ID.String()})
	}
}

func (s Server) guestLinksDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseGuestLinkID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("failed to parse guest link ID %s: %v", mux.Vars(r)["id"], err)
			http.Error(w, fmt.Sprintf("Invalid guest link ID: %v", err), http.StatusBadRequest)
			return
		}

		if err := s.store.DeleteGuestLink(id); err != nil {
			log.Printf("failed to delete guest link: %v", err)
			http.Error(w, fmt.Sprintf("Failed to delete guest link: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) guestLinksEnableDisable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseGuestLinkID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("failed to parse guest link ID %s: %v", mux.Vars(r)["id"], err)
			http.Error(w, fmt.Sprintf("Invalid guest link ID: %v", err), http.StatusBadRequest)
			return
		}

		if _, err := s.store.GetGuestLink(id); err != nil {
			log.Printf("failed to get guest link ID %s: %v", mux.Vars(r)["id"], err)
			http.Error(w, fmt.Sprintf("Guest link with ID %s not found: %v", mux.Vars(r)["id"], err), http.StatusNotFound)
			return
		}

		// Determine if client is enabling or disabling link.
		var dbFn func(femtoshare.GuestLinkID) error
		if strings.HasSuffix(r.URL.Path, "/enable") {
			dbFn = s.store.EnableGuestLink
		} else {
			dbFn = s.store.DisableGuestLink
		}

		if err := dbFn(id); err != nil {
			log.Printf("failed to change guest link enabled state: %v", err)
			http.Error(w, fmt.Sprintf("Failed to change guest link enabled state: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (s Server) guestLinkFromRequest(r *http.Request) (femtoshare.GuestLink, error) {
	var payload struct {
		Label          string  `json:"label"`
		UrlExpiration  string  `json:"urlExpirationTime"`
		FileExpiration string  `json:"fileLifetime"`
		MaxFileBytes   *uint64 `json:"maxFileBytes"`
		MaxFileUploads *int    `json:"maxFileUploads"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Printf("failed to decode JSON request: %v", err)
		return femtoshare.GuestLink{}, err
	}

	label, err := parse.GuestLinkLabel(payload.Label)
	if err != nil {
		return femtoshare.GuestLink{}, err
	}

	urlExpiration, err := parse.Expiration(payload.UrlExpiration, s.now())
	if err != nil {
		return femtoshare.GuestLink{}, err
	}

	fileExpiration, err := parse.FileLifetimeFromString(payload.FileExpiration)
	if err != nil {
		return femtoshare.GuestLink{}, err
	}

	maxFileBytes, err := parseMaxFileBytes(payload.MaxFileBytes)
	if err != nil {
		return femtoshare.GuestLink{}, err
	}

	maxFileUploads, err := parseUploadCountLimit(payload.MaxFileUploads)
	if err != nil {
		return femtoshare.GuestLink{}, err
	}

	return femtoshare.GuestLink{
		Label:           label,
		UrlExpires:      urlExpiration,
		MaxFileLifetime: fileExpiration,
		MaxFileBytes:    maxFileBytes,
		MaxFileUploads:  maxFileUploads,
	}, nil
}

func parseMaxFileBytes(limitRaw *uint64) (femtoshare.GuestUploadMaxFileBytes, error) {
	if limitRaw == nil {
		return femtoshare.GuestUploadUnlimitedFileSize, nil
	}
	if *limitRaw < GuestLinkByteLimitMinimum {
		return nil, fmt.Errorf("guest upload size limit must be at at least %d bytes", GuestLinkByteLimitMinimum)
	}

	return femtoshare.GuestUploadMaxFileBytes(limitRaw), nil
}

func parseUploadCountLimit(limitRaw *int) (femtoshare.GuestUploadCountLimit, error) {
	if limitRaw == nil {
		return femtoshare.GuestUploadUnlimitedFileUploads, nil
	}
	if *limitRaw <= 0 {
		return nil, errors.New("guest upload count limit must be a positive number")
	}

	return femtoshare.GuestUploadCountLimit(limitRaw), nil
}

func generateGuestLinkID() femtoshare.GuestLinkID {
	return femtoshare.GuestLinkID(random.String(GuestLinkIDLength, guestLinkIDCharacters))
}

func parseGuestLinkID(s string) (femtoshare.GuestLinkID, error) {
	if len(s) != GuestLinkIDLength {
		return femtoshare.GuestLinkID(""), fmt.Errorf("guest link ID (%v) has invalid length: got %d, want %d", s, len(s), GuestLinkIDLength)
	}

	// We could do this outside the function and store the result.
	idCharsHash := map[rune]bool{}
	for _, c := range guestLinkIDCharacters {
		idCharsHash[c] = true
	}

	for _, c := range s {
		if _, ok := idCharsHash[c]; !ok {
			return femtoshare.GuestLinkID(""), fmt.Errorf("entry ID (%s) contains invalid character: %s", s, string(c))
		}
	}
	return femtoshare.GuestLinkID(s), nil
}
