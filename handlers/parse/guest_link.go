package parse

import (
	"fmt"

	"github.com/cmyksoda/femtoshare/femtoshare"
)

// Arbitrary limit to prevent too-long labels in the UI.
const MaxGuestLinkLabelLength = 200

var ErrGuestLinkLabelTooLong = fmt.Errorf("label too long - limit %d characters", MaxGuestLinkLabelLength)

func GuestLinkLabel(label string) (femtoshare.GuestLinkLabel, error) {
	if len(label) > MaxGuestLinkLabelLength {
		return femtoshare.GuestLinkLabel(""), ErrGuestLinkLabelTooLong
	}

	return femtoshare.GuestLinkLabel(label), nil
}
