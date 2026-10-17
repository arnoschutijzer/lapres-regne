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
	// Boards is a floor of wooden boards, inside houses.
	Boards
	// Door is an open doorway through a wall, on a flagstone floor.
	Door
	// Bridge is a wooden deck carrying the way over water.
	Bridge
	Wall
	// Battlement is a castle wall, crowned with merlons.
	Battlement
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
	// Fixtures lists the furniture and landmarks standing on the map's tiles.
	Fixtures []PlacedFixture
	// StartCol and StartRow are the tile the hero starts on.
	StartCol, StartRow int
	// Areas names the part of the map each tile belongs to, row by row as Ground, such as a castle
	// or a village; "" for none.
	Areas [][]string
	// Heights is the level of each tile, row by row as Ground, each level LevelHeight above the
	// last; tiles left out lie at level 0.
	Heights [][]int
}

// LevelHeight is how much higher each level of a map's heights lies than the last, in world units.
const LevelHeight = 0.75

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

// Building is a roofed building on a map: a room closed in by its walls and doors, spanning the
// tiles from Col, Row across Cols and Rows, walls included.
type Building struct {
	Col, Row, Cols, Rows int
	// Thatched marks a building floored mostly with boards; one floored with stone is slated.
	Thatched bool
}

// terrain answers questions about the tiles of a map.
type terrain struct {
	ground [][]Ground
	areas  [][]string
	// buildings lists the map's roofed buildings, and rooms the tiles inside each of them.
	buildings []Building
	rooms     map[cell]int
	trees     map[cell]TreeKind
	fixtures  map[cell]Fixture
	// cover is the canopy over each tile, row by row.
	cover [][]float64
	// heights is the level of each tile, row by row, and highest the highest level.
	heights [][]int
	highest int
}

func newTerrain(m Map) *terrain {
	t := &terrain{ground: m.Ground, areas: m.Areas, heights: m.Heights, trees: make(map[cell]TreeKind), fixtures: make(map[cell]Fixture), cover: make([][]float64, len(m.Ground))}
	for _, levels := range m.Heights {
		for _, level := range levels {
			t.highest = max(t.highest, level)
		}
	}
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
	for _, fixture := range m.Fixtures {
		t.fixtures[cell{fixture.Col, fixture.Row}] = fixture.Kind
	}
	t.findBuildings()
	return t
}

// indoor tells whether a tile is floored inside, as rooms are.
func indoor(g Ground) bool { return g == Floor || g == Boards }

// findBuildings finds every room of flagstones or boards closed in by walls and doors alone, and
// roofs it as a building.
func (t *terrain) findBuildings() {
	t.rooms = make(map[cell]int)
	seen := make(map[cell]bool)
	steps := [4]cell{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	for row := range t.ground {
		for col := range t.ground[row] {
			start := cell{col, row}
			if seen[start] || !indoor(t.ground[row][col]) {
				continue
			}
			// Gather the room, noting whether it opens onto anything but its walls and doors.
			room, open, boards := []cell{start}, false, 0
			// The room's doorways count as indoors too, so its roof stays lifted while the hero
			// stands in one.
			var doorways []cell
			seen[start] = true
			for i := 0; i < len(room); i++ {
				here := room[i]
				if t.ground[here.gz][here.gx] == Boards {
					boards++
				}
				for _, step := range steps {
					next := cell{here.gx + step.gx, here.gz + step.gz}
					ground, ok := t.tile(next.gx, next.gz)
					switch {
					case !ok:
						open = true
					case indoor(ground):
						if !seen[next] {
							seen[next] = true
							room = append(room, next)
						}
					case ground == Door:
						doorways = append(doorways, next)
					case ground != Wall && ground != Battlement:
						open = true
					}
				}
			}
			if open {
				continue
			}
			left, top, right, bottom := start.gx, start.gz, start.gx, start.gz
			for _, c := range room {
				left, right = min(left, c.gx), max(right, c.gx)
				top, bottom = min(top, c.gz), max(bottom, c.gz)
				t.rooms[c] = len(t.buildings)
			}
			for _, doorway := range doorways {
				t.rooms[doorway] = len(t.buildings)
			}
			t.buildings = append(t.buildings, Building{
				Col: left - 1, Row: top - 1, Cols: right - left + 3, Rows: bottom - top + 3,
				Thatched: boards*2 > len(room),
			})
		}
	}
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

// elevationAt is the height of the ground at a world position. Water and bridge decks lie flat on
// their own level. Other ground blends the heights of the four tiles around the position, so it
// slopes smoothly from tile to tile; its banks fall to meet the water at the water's edge, except
// beside a bridge, whose deck holds them up on its level.
func (t *terrain) elevationAt(x, z float64) float64 {
	here := tileOf(x, z)
	if ground, ok := t.tile(here.gx, here.gz); ok && (ground == Water || ground == Bridge) {
		return t.tileHeight(here.gx, here.gz)
	}
	u, v := x/TileSize-0.5, z/TileSize-0.5
	col, row := int(math.Floor(u)), int(math.Floor(v))
	u, v = u-float64(col), v-float64(row)
	around := [4]cell{{col, row}, {col + 1, row}, {col, row + 1}, {col + 1, row + 1}}
	own := t.tileHeight(here.gx, here.gz)
	deck, bridged := 0.0, false
	for _, tile := range around {
		if ground, _ := t.clampedTile(tile.gx, tile.gz); ground == Bridge {
			deck, bridged = t.tileHeight(tile.gx, tile.gz), true
		}
	}
	var heights [4]float64
	for i, tile := range around {
		heights[i] = t.tileHeight(tile.gx, tile.gz)
		if ground, _ := t.clampedTile(tile.gx, tile.gz); ground == Water {
			// Mirrored about the water's level, the blend reaches it halfway between the tiles.
			heights[i] = 2*heights[i] - own
			if bridged {
				heights[i] = deck
			}
		}
	}
	north := heights[0]*(1-u) + heights[1]*u
	south := heights[2]*(1-u) + heights[3]*u
	return north*(1-v) + south*v
}

// clampedTile is the ground of a tile; beyond the map's edges, of the nearest tile on it.
func (t *terrain) clampedTile(col, row int) (Ground, bool) {
	row = min(max(row, 0), len(t.ground)-1)
	if row < 0 {
		return 0, false
	}
	return t.tile(min(max(col, 0), len(t.ground[row])-1), row)
}

// tileHeight is the height of a tile; beyond the map's edges the ground keeps the height of the
// nearest tile on it.
func (t *terrain) tileHeight(col, row int) float64 {
	row = min(max(row, 0), len(t.ground)-1)
	if row < 0 || row >= len(t.heights) {
		return 0
	}
	col = min(max(col, 0), len(t.ground[row])-1)
	if col < 0 || col >= len(t.heights[row]) {
		return 0
	}
	return float64(t.heights[row][col]) * LevelHeight
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

// Buildings lists the roofed buildings of the map; none in the endless forest.
func (w *World) Buildings() []Building {
	if w.terrain == nil {
		return nil
	}
	return w.terrain.buildings
}

// Indoors tells whether the hero stands inside a roofed building.
func (w *World) Indoors() bool {
	if w.terrain == nil {
		return false
	}
	_, inside := w.terrain.rooms[tileOf(w.hero.x, w.hero.z)]
	return inside
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
	return w.terrain.passable(x, z)
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
