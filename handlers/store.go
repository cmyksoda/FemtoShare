package handlers

import (
	"io"

	"github.com/cmyksoda/femtoshare/femtoshare"
)

type Store interface {
	GetEntriesMetadata() ([]femtoshare.UploadMetadata, error)
	ReadEntryFile(femtoshare.EntryID) (io.ReadSeeker, error)
	GetEntryMetadata(id femtoshare.EntryID) (femtoshare.UploadMetadata, error)
	InsertEntry(reader io.Reader, metadata femtoshare.UploadMetadata) error
	UpdateEntryMetadata(id femtoshare.EntryID, metadata femtoshare.UploadMetadata) error
	DeleteEntry(id femtoshare.EntryID) error
	GetGuestLink(femtoshare.GuestLinkID) (femtoshare.GuestLink, error)
	GetGuestLinks() ([]femtoshare.GuestLink, error)
	InsertGuestLink(femtoshare.GuestLink) error
	DeleteGuestLink(femtoshare.GuestLinkID) error
	DisableGuestLink(femtoshare.GuestLinkID) error
	EnableGuestLink(femtoshare.GuestLinkID) error
	InsertEntryDownload(femtoshare.EntryID, femtoshare.DownloadRecord) error
	GetEntryDownloads(id femtoshare.EntryID) ([]femtoshare.DownloadRecord, error)
	ReadSettings() (femtoshare.Settings, error)
	UpdateSettings(femtoshare.Settings) error
}
