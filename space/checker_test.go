package space_test

import (
	"errors"
	"testing"

	"github.com/cmyksoda/femtoshare/space"
	"github.com/cmyksoda/femtoshare/space/checkers"
)

type mockFileSystemChecker struct {
	usage checkers.FemtoShareUsage
	err   error
}

func (c mockFileSystemChecker) MeasureUsage() (checkers.FemtoShareUsage, error) {
	return c.usage, c.err
}

type mockDatabaseChecker struct {
	totalSize uint64
	err       error
}

func (c mockDatabaseChecker) TotalSize() (uint64, error) {
	return c.totalSize, c.err
}

func TestCheck(t *testing.T) {
	dummyFileSystemErr := errors.New("dummy filesystem checker error")
	dummyDatabaseErr := errors.New("dummy database checker error")
	for _, tt := range []struct {
		description   string
		fsUsage       checkers.FemtoShareUsage
		fsErr         error
		dbUsage       uint64
		dbErr         error
		usageExpected space.Usage
		errExpected   error
	}{
		{
			description: "aggregates checker results correctly",
			fsUsage: checkers.FemtoShareUsage{
				FileSystemUsage: checkers.FileSystemUsage{
					UsedBytes:  70,
					TotalBytes: 100,
				},
				FemtoShareDbFileSize: 65,
			},
			fsErr:   nil,
			dbUsage: 60,
			dbErr:   nil,
			usageExpected: space.Usage{
				TotalServingBytes:    60,
				DatabaseFileSize:     65,
				FileSystemUsedBytes:  70,
				FileSystemTotalBytes: 100,
			},
			errExpected: nil,
		},
		{
			description:   "returns error when filesystem checker fails",
			fsUsage:       checkers.FemtoShareUsage{},
			fsErr:         dummyFileSystemErr,
			dbUsage:       5,
			dbErr:         nil,
			usageExpected: space.Usage{},
			errExpected:   dummyFileSystemErr,
		},
		{
			description: "returns error when database checker fails",
			fsUsage: checkers.FemtoShareUsage{
				FemtoShareDbFileSize: 3,
				FileSystemUsage: checkers.FileSystemUsage{
					UsedBytes:  5,
					TotalBytes: 7,
				},
			},
			fsErr:         nil,
			dbUsage:       0,
			dbErr:         dummyDatabaseErr,
			usageExpected: space.Usage{},
			errExpected:   dummyDatabaseErr,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			fsc := mockFileSystemChecker{
				usage: tt.fsUsage,
				err:   tt.fsErr,
			}
			dbc := mockDatabaseChecker{
				totalSize: tt.dbUsage,
				err:       tt.dbErr,
			}

			usage, err := space.NewCheckerFromCheckers(fsc, dbc).Check()
			if got, want := err, tt.errExpected; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if got, want := usage, tt.usageExpected; got != want {
				t.Errorf("usage=%+v, want=%+v", got, want)
			}
		})
	}
}
