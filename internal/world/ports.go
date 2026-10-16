package world

// Sounds plays the world's sound effects.
type Sounds interface {
	// Chop sounds an axe landing on a tree.
	Chop()
}

// silence is the default when no sound effects are wired.
type silence struct{}

func (silence) Chop() {}
