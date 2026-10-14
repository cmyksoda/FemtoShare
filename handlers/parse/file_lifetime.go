package parse

import (
	"fmt"
	"time"

	"github.com/cmyksoda/femtoshare/femtoshare"
)

const minFileLifetimeInDays = 1
const maxFileLifetimeInYears = 10

// This is imprecise, but it's okay because file lifetimes are not exact
// measures of time.
const daysPerYear = 365

var (
	ErrFileLifetimeTooShort           = fmt.Errorf("file lifetime must be at least %d days", minFileLifetimeInDays)
	ErrFileLifetimeTooLong            = fmt.Errorf("file lifetime must be at most %d years", maxFileLifetimeInYears)
	ErrFileLifetimeUnrecognizedFormat = fmt.Errorf("unrecognized format for file life time, must be in 1ns, 1us (or 1µs), 1ms, 1s, 1m, 1h format")
)

func FileLifetime(lifetimeInDays uint16) (femtoshare.FileLifetime, error) {
	if lifetimeInDays < minFileLifetimeInDays {
		return femtoshare.FileLifetime{}, ErrFileLifetimeTooShort
	}
	if lifetimeInDays > (maxFileLifetimeInYears * daysPerYear) {
		return femtoshare.FileLifetime{}, ErrFileLifetimeTooLong
	}
	return femtoshare.NewFileLifetimeInDays(lifetimeInDays), nil
}

func FileLifetimeFromString(fileLifetimeRaw string) (femtoshare.FileLifetime, error) {
	dur, err := time.ParseDuration(fileLifetimeRaw)
	if err != nil {
		return femtoshare.FileLifetime{}, ErrFileLifetimeUnrecognizedFormat
	}

	if flt, _ := femtoshare.NewFileLifetimeFromDuration(dur); flt == femtoshare.FileLifetimeInfinite {
		return femtoshare.FileLifetimeInfinite, nil
	}

	return femtoshare.NewFileLifetimeFromDuration(dur)
}
