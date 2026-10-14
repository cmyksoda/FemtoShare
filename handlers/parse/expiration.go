package parse

import (
	"errors"
	"fmt"
	"time"

	"github.com/cmyksoda/femtoshare/femtoshare"
)

const expirationTimeFormat = time.RFC3339

var ErrExpirationUnrecognizedFormat = fmt.Errorf("unrecognized format for expiration time, must be in %s format", expirationTimeFormat)
var ErrExpirationTooSoon = errors.New("expire time must be at least one hour in the future")

func Expiration(expirationRaw string, now time.Time) (femtoshare.ExpirationTime, error) {
	expiration, err := time.Parse(expirationTimeFormat, expirationRaw)
	if err != nil {
		return femtoshare.ExpirationTime{}, ErrExpirationUnrecognizedFormat
	}

	if expiration.Sub(now) < (time.Hour * 1) {
		return femtoshare.ExpirationTime{}, ErrExpirationTooSoon
	}

	return femtoshare.ExpirationTime(expiration), nil
}
