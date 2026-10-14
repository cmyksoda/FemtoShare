package handlers

import "github.com/cmyksoda/femtoshare/space"

// SpaceCheckFunc measures FemtoShare storage usage.
type SpaceCheckFunc func() (space.Usage, error)
