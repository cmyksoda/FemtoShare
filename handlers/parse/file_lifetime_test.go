package parse_test

import (
	"testing"

	"github.com/cmyksoda/femtoshare/femtoshare"
	"github.com/cmyksoda/femtoshare/handlers/parse"
)

func TestFileLifetime(t *testing.T) {
	for _, tt := range []struct {
		description string
		input       uint16
		output      femtoshare.FileLifetime
		err         error
	}{
		{
			description: "valid lifetime",
			input:       7,
			output:      femtoshare.NewFileLifetimeInDays(7),
			err:         nil,
		},
		{
			description: "accepts the minimum valid lifetime",
			input:       1,
			output:      femtoshare.NewFileLifetimeInDays(1),
			err:         nil,
		},
		{
			description: "accepts the maximum valid lifetime",
			input:       365 * 10,
			output:      femtoshare.NewFileLifetimeInYears(10),
			err:         nil,
		},
		{
			description: "rejects too short a lifetime",
			input:       0,
			output:      femtoshare.FileLifetime{},
			err:         parse.ErrFileLifetimeTooShort,
		},
		{
			description: "rejects too long a lifetime",
			input:       3651,
			output:      femtoshare.FileLifetime{},
			err:         parse.ErrFileLifetimeTooLong,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			lt, err := parse.FileLifetime(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
			if got, want := lt, tt.output; got != want {
				t.Errorf("lifetime=%s, want=%s", got.FriendlyName(), want.FriendlyName())
			}
		})
	}
}
