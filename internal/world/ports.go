package world

import "time"

// Sounds plays the world's sound effects.
type Sounds interface {
	// Chop sounds an axe landing on a tree.
	Chop()
}

// silence is the default when no sound effects are wired.
type silence struct{}

func (silence) Chop() {}

// Progress keeps the hero's progress between sessions.
type Progress interface {
	// Load returns the last saved snapshot, or false when nothing was saved yet.
	Load() (Snapshot, bool, error)
	Save(Snapshot) error
}

// Snapshot is the progress kept between sessions.
type Snapshot struct {
	// Seed names the world the position and stumps belong to.
	Seed          uint64
	X, Z          float64
	WoodcuttingXP float64
	Logs          int
	Stumps        []Stump
	SavedAt       time.Time
}

// Stump is a felled tree on the tree grid and how long it still takes to regrow.
type Stump struct {
	GX, GZ int
	Regrow time.Duration
}
