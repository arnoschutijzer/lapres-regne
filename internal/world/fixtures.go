package world

// Fixture is something standing on a tile of a map: furniture indoors, or a landmark outside.
type Fixture byte

const (
	Table Fixture = iota
	Chair
	Throne
	// Armour is a suit of armour on its stand.
	Armour
	// Range is a cooking range or hearth with a fire in it.
	Range
	Barrel
	Crate
	Sacks
	// Shelves are shelves or a cabinet against a wall.
	Shelves
	Staircase
	Ladder
	// Torch is a standing torch.
	Torch
	// Candles are candles on a stand.
	Candles
	// Standard is a banner on a pole.
	Standard
	Altar
	// Pew is a church pew.
	Pew
	// Organ is a church organ.
	Organ
	Coffin
	Fountain
	Statue
	Well
	Haystack
	// Signpost is a signpost or a noticeboard.
	Signpost
	Cart
	Trough
	Scarecrow
	// Crop is a farm's vegetables growing in a row, low enough to walk among.
	Crop
	Anvil
	// Counter is a shop's counter.
	Counter
	Bed
)

// PlacedFixture is a fixture standing on a tile of a map.
type PlacedFixture struct {
	Col, Row int
	Kind     Fixture
}

// Blocks tells whether the fixture keeps the hero off its tile: everything does but crops.
func (f Fixture) Blocks() bool {
	return f != Crop
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
