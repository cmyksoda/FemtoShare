package femtoshare_test

import (
	"testing"

	"github.com/cmyksoda/femtoshare/femtoshare"
)

func TestFileLifetime(t *testing.T) {
	for _, tt := range []struct {
		lifetime         femtoshare.FileLifetime
		days             uint16
		years            uint16
		isOnYearBoundary bool
		friendlyName     string
	}{
		{
			lifetime:         femtoshare.NewFileLifetimeInDays(1),
			days:             1,
			years:            0,
			isOnYearBoundary: false,
			friendlyName:     "1 day",
		},
		{
			lifetime:         femtoshare.NewFileLifetimeInDays(7),
			days:             7,
			years:            0,
			isOnYearBoundary: false,
			friendlyName:     "7 days",
		},
		{
			lifetime:         femtoshare.NewFileLifetimeInDays(30),
			days:             30,
			years:            0,
			isOnYearBoundary: false,
			friendlyName:     "30 days",
		},
		{
			lifetime:         femtoshare.NewFileLifetimeInYears(1),
			days:             365,
			years:            1,
			isOnYearBoundary: true,
			friendlyName:     "1 year",
		},
		{
			lifetime:         femtoshare.NewFileLifetimeInDays(366),
			days:             366,
			years:            0,
			isOnYearBoundary: false,
			friendlyName:     "366 days",
		},
		{
			lifetime:         femtoshare.NewFileLifetimeInYears(10),
			days:             3650,
			years:            10,
			isOnYearBoundary: false,
			friendlyName:     "10 years",
		},
		{
			lifetime:         femtoshare.FileLifetimeInfinite,
			isOnYearBoundary: false,
			friendlyName:     "Never",
		},
	} {
		t.Run(tt.friendlyName, func(t *testing.T) {
			if got, want := tt.lifetime.FriendlyName(), tt.friendlyName; got != want {
				t.Errorf("friendlyName=%v, want=%v", got, want)
			}
		})
	}
}
