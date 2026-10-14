package space

import "github.com/cmyksoda/femtoshare/space/checkers"

type (
	FileSystemChecker interface {
		MeasureUsage() (checkers.FemtoShareUsage, error)
	}

	DatabaseChecker interface {
		TotalSize() (uint64, error)
	}

	Checker struct {
		fsChecker FileSystemChecker
		dbChecker DatabaseChecker
	}

	Usage struct {
		// TotalServingBytes represents the sum total of the bytes of file data that
		// FemtoShare has of file uploads in the database. This is just file bytes
		// and does not include FemtoShare-specific metadata about the files.
		TotalServingBytes uint64
		// DatabaseFileSize represents the total number of bytes on the filesystem
		// dedicated to storing FemtoShare's SQLite database files.
		DatabaseFileSize uint64
		// FileSystemUsedBytes represents total bytes in use on the filesystem where
		// FemtoShare's database files are located. This represents the total of all
		// used bytes on the filesystem, not just FemtoShare.
		FileSystemUsedBytes uint64
		// FileSystemTotalBytes represents the total bytes available on the
		// filesystem where FemtoShare's database files are located.
		FileSystemTotalBytes uint64
	}
)

func NewChecker(dbPath string, dbReader checkers.DatabaseMetadataReader) Checker {
	return NewCheckerFromCheckers(checkers.NewFileSystemChecker(dbPath), checkers.NewDatabaseChecker(dbReader))
}

func NewCheckerFromCheckers(fsChecker FileSystemChecker, dbChecker DatabaseChecker) Checker {
	return Checker{
		fsChecker,
		dbChecker,
	}
}

func (c Checker) Check() (Usage, error) {
	fsUsage, err := c.fsChecker.MeasureUsage()
	if err != nil {
		return Usage{}, err
	}

	dbTotalSize, err := c.dbChecker.TotalSize()
	if err != nil {
		return Usage{}, err
	}

	return Usage{
		TotalServingBytes:    dbTotalSize,
		DatabaseFileSize:     fsUsage.FemtoShareDbFileSize,
		FileSystemUsedBytes:  fsUsage.UsedBytes,
		FileSystemTotalBytes: fsUsage.TotalBytes,
	}, nil
}
