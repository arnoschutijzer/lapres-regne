package world

import "time"

// Sounds plays the world's sound effects.
type Sounds interface {
	// Chop sounds an axe landing on a tree.
	Chop()
	// Fall sounds a chopped tree coming down.
	Fall()
	// Drop sounds a dropped item landing on the ground.
	Drop(Landing)
	// LevelUp plays the jingle for a new level; unlocked picks the variant for a level that lets
	// the hero chop a new kind of tree.
	LevelUp(unlocked bool)
	// OpenDoor sounds a door or gate swinging open, and CloseDoor one swinging shut.
	OpenDoor()
	CloseDoor()
}

// silence is the default when no sound effects are wired.
type silence struct{}

func (silence) Chop()        {}
func (silence) Fall()        {}
func (silence) Drop(Landing) {}
func (silence) LevelUp(bool) {}
func (silence) OpenDoor()    {}
func (silence) CloseDoor()   {}

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
	Map           string
	X, Z          float64
	WoodcuttingXP float64
	// Zoom is the camera's zoom, in steps in from the forest view.
	Zoom int
	// Yaw is how far the camera has turned about the upright axis, in radians.
	Yaw float64
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
