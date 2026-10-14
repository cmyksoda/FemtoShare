package parse_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cmyksoda/femtoshare/femtoshare"
	"github.com/cmyksoda/femtoshare/handlers/parse"
)

func TestGuestLinkLabel(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       string
		output      femtoshare.GuestLinkLabel
		err         error
	}{
		{
			description: "accept valid label",
			input:       "For my good pals",
			output:      femtoshare.GuestLinkLabel("For my good pals"),
			err:         nil,
		},
		{
			description: "allow empty label",
			input:       "",
			output:      femtoshare.GuestLinkLabel(""),
			err:         nil,
		},
		{
			description: "reject labels that are too long",
			input:       strings.Repeat("A", parse.MaxGuestLinkLabelLength+1),
			output:      femtoshare.GuestLinkLabel(""),
			err:         parse.ErrGuestLinkLabelTooLong,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.description, tt.input), func(t *testing.T) {
			label, err := parse.GuestLinkLabel(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", err, want)
			}
			if got, want := label, tt.output; got != want {
				t.Errorf("label=%v, want=%v", label, want)
			}
		})
	}
}
