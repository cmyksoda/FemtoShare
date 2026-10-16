package handlers

import (
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/cmyksoda/femtoshare/femtoshare"
	"github.com/cmyksoda/femtoshare/store"
)

// maxEntryUnlockRequestBytes bounds the body of a request to unlock an entry.
// Even a passphrase of MaxPassphraseLength code points that form encoding
// percent-escapes entirely into 4-byte sequences fits well within it.
const maxEntryUnlockRequestBytes = 4096

type entryUnlockRequest struct {
	EntryID    femtoshare.EntryID
	Passphrase femtoshare.DownloadPassphrase
}

func (s Server) entryGet() http.HandlerFunc {
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
		if !entry.DownloadPassphrase.Empty() {
			// Prevent caches from serving a password-bypassing download after an
			// administrator downloads this entry with their session cookie.
			w.Header().Set("Cache-Control", "no-store")
			if !isAuthenticated(r.Context()) {
				http.Redirect(w, r, entryUnlockPath(entry.ID), http.StatusFound)
				return
			}
		}
		s.serveEntryContent(w, r, entry)
	}
}

func (s Server) entryUnlockGet() http.HandlerFunc {
	t := parseTemplates("templates/pages/entry-unlock.html")
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

		// Prevent caches from serving a password-bypassing redirect after an
		// administrator accesses this page with their session cookie.
		w.Header().Set("Cache-Control", "no-store")
		if entry.DownloadPassphrase.Empty() || isAuthenticated(r.Context()) {
			http.Redirect(w, r, entryDownloadPath(entry.ID), http.StatusFound)
			return
		}

		renderTemplate(w, t, struct {
			commonProps
			IncorrectPassphrase bool
		}{
			commonProps:         makeCommonProps("FemtoShare - Download", r.Context()),
			IncorrectPassphrase: false,
		})
	}
}

func (s Server) entryUnlockPost() http.HandlerFunc {
	t := parseTemplates("templates/pages/entry-unlock.html")
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, maxEntryUnlockRequestBytes)
		unlockRequest, err := parseEntryUnlockRequest(r)
		if err != nil && !errors.Is(err, femtoshare.ErrInvalidPassphrase) {
			http.Error(w, "invalid passphrase form", http.StatusBadRequest)
			return
		}

		entry, err := s.store.GetEntryMetadata(unlockRequest.EntryID)
		if _, ok := errors.AsType[store.EntryNotFoundError](err); ok {
			http.Error(w, "entry not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry with id %v: %v", unlockRequest.EntryID, err)
			http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
			return
		}

		if entry.DownloadPassphrase.Empty() || isAuthenticated(r.Context()) {
			http.Redirect(w, r, entryDownloadPath(entry.ID), http.StatusFound)
			return
		}

		// If password is incorrect, serve unlock page with error message.
		if !entry.DownloadPassphrase.Equal(unlockRequest.Passphrase) {
			w.WriteHeader(http.StatusUnauthorized)
			renderTemplate(w, t, struct {
				commonProps
				IncorrectPassphrase bool
			}{
				commonProps:         makeCommonProps("FemtoShare - Download", r.Context()),
				IncorrectPassphrase: true,
			})
			return
		}

		// If password is correct, serve the content.
		s.serveEntryContent(w, r, entry)
	}
}

func parseEntryUnlockRequest(r *http.Request) (entryUnlockRequest, error) {
	id, err := femtoshare.EntryIDFromString(mux.Vars(r)["id"])
	if err != nil {
		return entryUnlockRequest{}, err
	}
	if err := r.ParseForm(); err != nil {
		return entryUnlockRequest{}, err
	}
	passphrase, err := femtoshare.NewDownloadPassphrase(r.PostForm.Get("passphrase"))
	if err != nil {
		return entryUnlockRequest{EntryID: id}, err
	}
	return entryUnlockRequest{
		EntryID:    id,
		Passphrase: passphrase,
	}, nil
}

func entryDownloadPath(id femtoshare.EntryID) string {
	return "/-" + id.String()
}

func entryUnlockPath(id femtoshare.EntryID) string {
	return entryDownloadPath(id) + "/unlock"
}

func (s Server) serveEntryContent(w http.ResponseWriter, r *http.Request, entry femtoshare.UploadMetadata) {
	if entry.Filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, entry.Filename))
	}

	contentType := entry.ContentType
	if contentType == "" || contentType == "application/octet-stream" {
		if inferred, err := inferContentTypeFromFilename(entry.Filename); err == nil {
			contentType = inferred
		}
	}
	w.Header().Set("Content-Type", contentType.String())

	// Serve the response in a sandbox so that an uploaded HTML or SVG file
	// cannot run scripts in the server's origin.
	policy := func() string {
		// Parse the media type instead of matching a prefix of the raw header so
		// that a malformed Content-Type, which browsers would ignore and sniff,
		// keeps the strict policy.
		mediaType, _, err := mime.ParseMediaType(contentType.String())
		if err == nil && (strings.HasPrefix(mediaType, "audio/") || strings.HasPrefix(mediaType, "video/")) {
			// Browser-generated media players need their original origin for media
			// requests. The sandbox still blocks scripts from running.
			return "sandbox allow-same-origin"
		}
		return "sandbox"
	}()
	w.Header().Set("Content-Security-Policy", policy)

	entryFile, err := s.store.ReadEntryFile(entry.ID)
	if err != nil {
		log.Printf("error retrieving entry data with id %v: %v", entry.ID, err)
		http.Error(w, "failed to retrieve entry", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, entry.Filename.String(), entry.Uploaded, entryFile)

	if err := recordDownload(s.store, entry.ID, s.now(), r.RemoteAddr, r.Header.Get("User-Agent")); err != nil {
		log.Printf("failed to record download of file %s: %v", entry.ID.String(), err)
	}
}

func inferContentTypeFromFilename(f femtoshare.Filename) (femtoshare.ContentType, error) {
	// For files that modern browser can play natively, infer the content type if
	// none was specified at upload time.
	if mimetype := mime.TypeByExtension(filepath.Ext(f.String())); mimetype != "" {
		return femtoshare.ContentType(mimetype), nil
	}
	return femtoshare.ContentType(""), errors.New("could not infer content type from filename")
}

func recordDownload(db Store, id femtoshare.EntryID, t time.Time, remoteAddr, userAgent string) error {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}

	return db.InsertEntryDownload(id, femtoshare.DownloadRecord{
		Time:      t,
		ClientIP:  ip,
		UserAgent: userAgent,
	})
}
