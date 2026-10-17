package world

import "math"

// Edges are the walls, battlements, fences and doorways standing along the edges between a map's
// tiles and diagonally across them, each row by row as Ground: West along each tile's west edge,
// North along its north edge, Rising from its south-west corner to its north-east, Falling from
// its north-west corner to its south-east. Grass, the zero Ground, stands for nothing; a Door is a
// doorway through a wall, or an open gate in a fence.
type Edges struct {
	West, North, Rising, Falling [][]Ground
}

// TileEdges is what stands along a tile's west and north edges and diagonally across it.
type TileEdges struct {
	West, North, Rising, Falling Ground
}

// How far either side of an edge its wall or fence keeps the hero.
const (
	wallReach  = 0.35
	fenceReach = 0.12
	// jambReach is how far into a doorway its stone jambs stand from either end.
	jambReach = 0.4
)

// along is what stands in one of the grids of edges at a tile; nothing beyond them.
func along(grid [][]Ground, col, row int) Ground {
	if row < 0 || row >= len(grid) || col < 0 || col >= len(grid[row]) {
		return Grass
	}
	return grid[row][col]
}

// edgesOf is what stands along a tile's edges and across it.
func (t *terrain) edgesOf(c cell) TileEdges {
	return TileEdges{
		West: along(t.edges.West, c.gx, c.gz), North: along(t.edges.North, c.gx, c.gz),
		Rising: along(t.edges.Rising, c.gx, c.gz), Falling: along(t.edges.Falling, c.gx, c.gz),
	}
}

// bars tells whether a wall or fence stands in an edge, keeping the hero from crossing it.
func bars(g Ground) bool {
	return g == Wall || g == Battlement || g == Fence
}

// keptBy is how far either side of an edge what stands in it keeps the hero.
func keptBy(g Ground) float64 {
	if g == Fence {
		return fenceReach
	}
	return wallReach
}

// between is what stands in the edge between two tiles side by side.
func (t *terrain) between(a, b cell) Ground {
	switch {
	case b.gx == a.gx+1:
		return t.edgesOf(b).West
	case b.gx == a.gx-1:
		return t.edgesOf(a).West
	case b.gz == a.gz+1:
		return t.edgesOf(b).North
	}
	return t.edgesOf(a).North
}

// crossed tells whether a wall or fence diagonally across a tile bars it.
func (t *terrain) crossed(c cell) bool {
	edges := t.edgesOf(c)
	return bars(edges.Rising) || bars(edges.Falling)
}

// segment is a stretch of an edge, from one point on the ground to another.
type segment struct{ x0, z0, x1, z1 float64 }

// barred tells whether a point lies within reach of a wall or fence standing along the edges of
// its tile or of the tiles round it, or of the stone jambs either side of a doorway.
func (t *terrain) barred(x, z float64) bool {
	here := tileOf(x, z)
	for row := here.gz - 1; row <= here.gz+1; row++ {
		for col := here.gx - 1; col <= here.gx+1; col++ {
			edges := t.edgesOf(cell{col, row})
			left, top := float64(col)*TileSize, float64(row)*TileSize
			for _, edge := range [2]struct {
				ground Ground
				along  segment
			}{
				{edges.West, segment{left, top, left, top + TileSize}},
				{edges.North, segment{left, top, left + TileSize, top}},
			} {
				if bars(edge.ground) && edge.along.distance(x, z) < keptBy(edge.ground) {
					return true
				}
				if edge.ground == Door && t.jambed(edge.along, x, z) {
					return true
				}
			}
		}
	}
	return false
}

// jambed tells whether a point lies against a jamb of a doorway along an edge set in a wall.
func (t *terrain) jambed(doorway segment, x, z float64) bool {
	length := math.Hypot(doorway.x1-doorway.x0, doorway.z1-doorway.z0)
	u := jambReach / length
	for _, jamb := range [2]segment{
		{doorway.x0, doorway.z0, doorway.x0 + (doorway.x1-doorway.x0)*u, doorway.z0 + (doorway.z1-doorway.z0)*u},
		{doorway.x1 - (doorway.x1-doorway.x0)*u, doorway.z1 - (doorway.z1-doorway.z0)*u, doorway.x1, doorway.z1},
	} {
		if jamb.distance(x, z) < wallReach {
			return true
		}
	}
	return false
}

// distance is how far a point lies from the segment.
func (s segment) distance(x, z float64) float64 {
	dx, dz := s.x1-s.x0, s.z1-s.z0
	u := math.Max(0, math.Min(1, ((x-s.x0)*dx+(z-s.z0)*dz)/(dx*dx+dz*dz)))
	return math.Hypot(x-s.x0-u*dx, z-s.z0-u*dz)
}

// EdgesAt is what stands along the edges of the tile at a world position and across it; nothing
// in the endless forest.
func (w *World) EdgesAt(x, z float64) TileEdges {
	if w.terrain == nil {
		return TileEdges{}
	}
	return w.terrain.edgesOf(tileOf(x, z))
}
