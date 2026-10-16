package world

import (
	"math"
	"time"
)

// TileSize is how wide one tile of a map is, in world units.
const TileSize = 2.0

// Ground is what covers one tile of a map.
type Ground byte

const (
	Grass Ground = iota
	Water
	// Path is a beaten earth track.
	Path
	// Floor is flagstones, inside buildings and courtyards.
	Floor
	Wall
	Fence
	Gate
	Grave
)

// Blocks tells whether whatever stands on the ground keeps the hero off it.
func (g Ground) Blocks() bool {
	return g >= Wall
}

// Map is a hand-made world: rows of tiles from north to south, each listing its tiles from west to
// east, with trees planted on some of them. Nothing beyond its edges can be walked on.
type Map struct {
	Name   string
	Ground [][]Ground
	Trees  []PlantedTree
	// StartCol and StartRow are the tile the hero starts on.
	StartCol, StartRow int
	// Areas names the part of the map each tile belongs to, row by row as Ground, such as a castle
	// or a village; "" for none.
	Areas [][]string
}

// PlantedTree is a tree of a kind growing on a tile of a map.
type PlantedTree struct {
	Col, Row int
	Kind     TreeKind
}

// Canopy cover over a map: open over bare tiles, closed over trees and shaded beside them.
const (
	openCover    = 0.3
	besideCover  = 0.55
	treeTopCover = 0.75
)

// terrain answers questions about the tiles of a map.
type terrain struct {
	ground [][]Ground
	areas  [][]string
	trees  map[cell]TreeKind
	// cover is the canopy over each tile, row by row.
	cover [][]float64
}

func newTerrain(m Map) *terrain {
	t := &terrain{ground: m.Ground, areas: m.Areas, trees: make(map[cell]TreeKind), cover: make([][]float64, len(m.Ground))}
	for row, tiles := range m.Ground {
		t.cover[row] = make([]float64, len(tiles))
		for col := range tiles {
			t.cover[row][col] = openCover
		}
	}
	for _, tree := range m.Trees {
		t.trees[cell{tree.Col, tree.Row}] = tree.Kind
		for row := tree.Row - 1; row <= tree.Row+1; row++ {
			for col := tree.Col - 1; col <= tree.Col+1; col++ {
				if _, ok := t.tile(col, row); ok {
					t.cover[row][col] = math.Max(t.cover[row][col], besideCover)
				}
			}
		}
	}
	for _, tree := range m.Trees {
		if _, ok := t.tile(tree.Col, tree.Row); ok {
			t.cover[tree.Row][tree.Col] = treeTopCover
		}
	}
	return t
}

// tile is the ground of a tile, or false beyond the map's edges.
func (t *terrain) tile(col, row int) (Ground, bool) {
	if row < 0 || row >= len(t.ground) || col < 0 || col >= len(t.ground[row]) {
		return 0, false
	}
	return t.ground[row][col], true
}

// area names the part of the map a world position belongs to, or "" for none.
func (t *terrain) area(x, z float64) string {
	col, row := int(math.Floor(x/TileSize)), int(math.Floor(z/TileSize))
	if row < 0 || row >= len(t.areas) || col < 0 || col >= len(t.areas[row]) {
		return ""
	}
	return t.areas[row][col]
}

// at is the ground at a world position, or false beyond the map's edges.
func (t *terrain) at(x, z float64) (Ground, bool) {
	return t.tile(int(math.Floor(x/TileSize)), int(math.Floor(z/TileSize)))
}

// coverAt blends the cover of the four tiles around a world position, so the canopy's shade
// fades between tiles.
func (t *terrain) coverAt(x, z float64) float64 {
	u, v := x/TileSize-0.5, z/TileSize-0.5
	col, row := int(math.Floor(u)), int(math.Floor(v))
	u, v = u-float64(col), v-float64(row)
	north := t.tileCover(col, row)*(1-u) + t.tileCover(col+1, row)*u
	south := t.tileCover(col, row+1)*(1-u) + t.tileCover(col+1, row+1)*u
	return north*(1-v) + south*v
}

func (t *terrain) tileCover(col, row int) float64 {
	if _, ok := t.tile(col, row); !ok {
		return openCover
	}
	return t.cover[row][col]
}

// NewOnMap builds a world on a map, with the hero on its start tile.
func NewOnMap(seed uint64, m Map) *World {
	w := New(seed)
	w.mapName = m.Name
	w.terrain = newTerrain(m)
	w.hero.passable = w.passable
	w.hero.x, w.hero.z = (float64(m.StartCol)+0.5)*TileSize, (float64(m.StartRow)+0.5)*TileSize
	w.cameraX, w.cameraZ = w.hero.x, w.hero.z
	return w
}

// ResumeOn builds a world on a map from a seed and the saved progress, as Resume does. The hero
// returns to where they stood only when the progress was saved on the same map.
func ResumeOn(m Map, seed uint64, progress Progress, now time.Time) (*World, error) {
	return resume(NewOnMap(seed, m), progress, now)
}

// Mapped tells whether the world is a map rather than the endless forest.
func (w *World) Mapped() bool { return w.terrain != nil }

// MapName names the map the world is built on; empty for the endless forest.
func (w *World) MapName() string { return w.mapName }

// GroundAt is the ground of the map at a world position, or false beyond its edges and in the
// endless forest.
func (w *World) GroundAt(x, z float64) (Ground, bool) {
	if w.terrain == nil {
		return 0, false
	}
	return w.terrain.at(x, z)
}

// TreeGrid is how far apart the cells of the tree grid are: one tree per tile on a map.
func (w *World) TreeGrid() float64 {
	if w.terrain != nil {
		return TileSize
	}
	return TreeSpacing
}

// passable tells whether the hero may stand at a world position.
func (w *World) passable(x, z float64) bool {
	if w.terrain == nil {
		return true
	}
	ground, ok := w.terrain.at(x, z)
	return ok && !ground.Blocks()
}

// plantedTree is the tree planted on a tile of the map, placed a little off the tile's centre and
// grown smaller than the endless forest's trees, so its crown spreads over no more than the tiles
// around it.
func (w *World) plantedTree(gx, gz int) (Tree, bool) {
	kind, planted := w.terrain.trees[cell{gx, gz}]
	if !planted {
		return Tree{}, false
	}
	x := (float64(gx) + 0.35 + w.Random(gx, gz, 1)*0.3) * TileSize
	z := (float64(gz) + 0.35 + w.Random(gx, gz, 2)*0.3) * TileSize
	tree := w.grow(gx, gz, x, z, kind)
	tree.Height, tree.Radius, tree.Trunk = tree.Height*0.8, tree.Radius*0.6, tree.Trunk*0.8
	return tree, true
}
