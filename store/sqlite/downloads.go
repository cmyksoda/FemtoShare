package sqlite

import (
	"database/sql"
	"log"

	"github.com/cmyksoda/femtoshare/femtoshare"
)

func (s Store) InsertEntryDownload(id femtoshare.EntryID, r femtoshare.DownloadRecord) error {
	log.Printf("recording download of file ID %s from client %s", id.String(), r.ClientIP)
	if _, err := s.db.Exec(`
	INSERT INTO
		downloads
	(
		entry_id,
		download_timestamp,
		client_ip,
		user_agent
	)
	VALUES(:entry_id, :download_timestamp, :client_ip, :user_agent)`,
		sql.Named("entry_id", id.String()),
		sql.Named("download_timestamp", formatTime(r.Time)),
		sql.Named("client_ip", r.ClientIP),
		sql.Named("user_agent", r.UserAgent),
	); err != nil {
		log.Printf("insert into downloads table failed: %v", err)
		return err
	}
	return nil
}

func (s Store) GetEntryDownloads(id femtoshare.EntryID) ([]femtoshare.DownloadRecord, error) {
	rows, err := s.db.Query(`
	SELECT
		download_timestamp,
		client_ip,
		user_agent
	FROM
		downloads
	WHERE
		entry_id=:entry_id
	ORDER BY
		download_timestamp DESC`, sql.Named("entry_id", id.String()))
	if err == sql.ErrNoRows {
		return []femtoshare.DownloadRecord{}, nil
	} else if err != nil {
		return []femtoshare.DownloadRecord{}, err
	}

	downloads := []femtoshare.DownloadRecord{}
	for rows.Next() {
		var downloadTimeRaw string
		var clientIP string
		var userAgent string

		if err := rows.Scan(&downloadTimeRaw, &clientIP, &userAgent); err != nil {
			return []femtoshare.DownloadRecord{}, err
		}

		dt, err := parseDatetime(downloadTimeRaw)
		if err != nil {
			return []femtoshare.DownloadRecord{}, err
		}

		downloads = append(downloads, femtoshare.DownloadRecord{
			Time:      dt,
			ClientIP:  clientIP,
			UserAgent: userAgent,
		})
	}

	return downloads, nil
}
