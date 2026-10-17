package world

import "time"

// Sounds plays the world's sound effects.
type Sounds interface {
	Play(Effect)
}

// Effect is a sound effect the world plays.
type Effect byte

const (
	// AxeStrike is an axe landing on a tree, and TreeFall a chopped tree coming down.
	AxeStrike Effect = iota
	TreeFall
	// WoodenThud is a dropped wooden item landing on the ground.
	WoodenThud
	// LevelUpJingle plays for a new level, and UnlockingJingle for one that lets the hero do
	// something new.
	LevelUpJingle
	UnlockingJingle
	// DoorOpening is a door or gate swinging open, and DoorClosing one swinging shut.
	DoorOpening
	DoorClosing
	// StairsClimb is the hero climbing a staircase, and LadderClimb a ladder.
	StairsClimb
	LadderClimb
)

// silence is the default when no sound effects are wired.
type silence struct{}

func (silence) Play(Effect) {}

// Progress keeps the hero's progress between sessions.
type Progress interface {
	// Load returns the last saved snapshot, or false when nothing was saved yet.
	Load() (Snapshot, bool, error)
	Save(Snapshot) error
}

// Snapshot is the progress kept between sessions.
type Snapshot struct {
	// Seed names the world the position and stumps belong to.
	Seed uint64
	// Map names the map the position and stumps belong to; empty for the endless forest.
	Map  string
	X, Z float64
	// Storey is the floor the hero stood on: 0 for the ground floor.
	Storey int
	// Skills holds the experience gained in each skill trained, by name.
	Skills map[string]float64
	// Inventory lists the inventory slot by slot, gaps included.
	Inventory []Slot
	Stumps    []Stump
	SavedAt   time.Time
}

// Stump is a felled tree on the tree grid and how long it still takes to regrow.
type Stump struct {
	GX, GZ int
	Regrow time.Duration
}
