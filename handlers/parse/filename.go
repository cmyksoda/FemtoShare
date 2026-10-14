package parse

import (
	"errors"
	"strings"

	"github.com/cmyksoda/femtoshare/femtoshare"
)

// MaxFilenameBytes is the maximum number of bytes allowed for uploaded files
// There's no technical reason on FemtoShare's side for this limitation, but it's
// useful to have some upper bound to limit malicious inputs, and 255 is a
// common filename limit (in single-byte characters) across most filesystems.
const MaxFilenameBytes = 255

var ErrFilenameEmpty = errors.New("filename must be non-empty")
var ErrFilenameTooLong = errors.New("filename too long")
var ErrFilenameHasDotPrefix = errors.New("filename cannot begin with dots")
var ErrFilenameIllegalCharacters = errors.New("illegal characters in filename")

func Filename(s string) (femtoshare.Filename, error) {
	if s == "" {
		return femtoshare.Filename(""), ErrFilenameEmpty
	}
	if len(s) > MaxFilenameBytes {
		return femtoshare.Filename(""), ErrFilenameTooLong
	}
	if s == "." || strings.HasPrefix(s, "..") {
		return femtoshare.Filename(""), ErrFilenameHasDotPrefix
	}
	if strings.ContainsAny(s, "\\/\a\b\t\n\v\f\r\n") {
		return femtoshare.Filename(""), ErrFilenameIllegalCharacters
	}
	return femtoshare.Filename(s), nil
}
