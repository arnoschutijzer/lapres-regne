package terminal

import "lapresregne/internal/world"

// The proportions of furniture, in world units.
const (
	tableHeight = 0.8
	// tableTop is how thick a table's top is.
	tableTop = 0.12
)

// drawFixture draws the furniture or landmark standing on a tile whose north-west corner is at
// x, z.
func drawFixture(c *forestCanvas, x, z float64, tile builtTile) {
	switch tile.fixture {
	case world.Table:
		drawTable(c, x, z, tile.alike)
	}
}

// footprint is the part of a tile a fixture covers, inset from the tile's edges by inset but
// reaching to each edge where the same kind stands beside it.
func footprint(x, z, inset float64, alike [4]bool) (x0, z0, x1, z1 float64) {
	x0, z0, x1, z1 = x+inset, z+inset, x+world.TileSize-inset, z+world.TileSize-inset
	if alike[0] {
		z0 = z
	}
	if alike[1] {
		x1 = x + world.TileSize
	}
	if alike[2] {
		z1 = z + world.TileSize
	}
	if alike[3] {
		x0 = x
	}
	return x0, z0, x1, z1
}

// drawTable stands a wooden table in the middle of a tile, its top running on over the tiles
// beside it where the table goes on, with legs only at its outer corners.
func drawTable(c *forestCanvas, x, z float64, alike [4]bool) {
	x0, z0, x1, z1 := footprint(x, z, 0.3, alike)
	leg := c.thin(0.08)
	corners := [4]struct {
		x, z  float64
		outer bool
	}{
		{x0 + 0.12, z0 + 0.12, !alike[0] && !alike[3]},
		{x1 - 0.12, z0 + 0.12, !alike[0] && !alike[1]},
		{x0 + 0.12, z1 - 0.12, !alike[2] && !alike[3]},
		{x1 - 0.12, z1 - 0.12, !alike[2] && !alike[1]},
	}
	for _, corner := range corners {
		if corner.outer {
			c.box(corner.x-leg, corner.z-leg, corner.x+leg, corner.z+leg, 0, tableHeight-tableTop, [4]bool{}, plankInk, plain)
		}
	}
	hidden := [4]bool{alike[0], alike[1], alike[2], alike[3]}
	c.box(x0, z0, x1, z1, tableHeight-tableTop, tableHeight, hidden, plankInk, planks)
}
