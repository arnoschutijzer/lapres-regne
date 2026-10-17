package world

// Fixture is something standing on a tile of a map: furniture indoors, or a landmark outside.
type Fixture byte

const (
	Table Fixture = iota
)

// PlacedFixture is a fixture standing on a tile of a map.
type PlacedFixture struct {
	Col, Row int
	Kind     Fixture
}

// Blocks tells whether the fixture keeps the hero off its tile.
func (f Fixture) Blocks() bool {
	return true
}

// FixtureAt is the fixture standing on the tile at a world position, or false where none stands
// and in the endless forest.
func (w *World) FixtureAt(x, z float64) (Fixture, bool) {
	if w.terrain == nil {
		return 0, false
	}
	fixture, ok := w.terrain.fixtures[tileOf(x, z)]
	return fixture, ok
}
