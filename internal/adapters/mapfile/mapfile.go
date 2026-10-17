// Package mapfile reads hand-made maps from text files.
//
// A map file starts with settings, one per line, then grids of characters, each introduced by
// its name on a line of its own and ending at a blank line. Lines starting with // are comments.
//
//	name Lumbridge
//
//	terrain
//	..T..~~
//	.@...~~
//
// The terrain grid has one character per tile, rows running north to south:
//
//	.  grass            ~  water
//	T  tree             O  oak
//	W  willow           Y  yew
//	:  earth path       _  stone floor
//	#  wall             =  fence
//	G  gate             +  grave
//	D  open door        b  wooden floor
//	B  battlements      H  bridge
//	@  where the hero starts, on grass
//
// A "start <col> <row>" setting starts the hero on a tile of any ground instead.
//
// Walls, fences, gates and graves block the way, and a bridge is railed along the water.
//
// An areas grid, laid over the terrain, names the parts of the map. Each letter stands for the area
// named by an "area <letter> <name>" setting, and . for none:
//
//	area C Lumbridge Castle
//
//	areas
//	CCC..
//
// A heights grid, laid over the terrain, raises each tile to a level from 0 to 9; the ground slopes
// smoothly from tile to tile, and tiles the grid leaves out lie at level 0:
//
//	heights
//	00122
//
// A walls grid stands walls, battlements, fences and doorways along the edges between tiles, at
// twice the terrain's resolution: even lines run along the north edges of a row of tiles, odd
// lines along their west edges with each tile's middle between. Along an edge, # is a wall, B
// battlements, = a fence and D a doorway, or an open gate in a fence; anything else is nothing.
// A tile's middle may hold / for a wall rising from its south-west corner to its north-east, or \
// falling from its north-west corner to its south-east, battlements or a fence when both corners
// it joins are marked B or =. Corners are free to mark, for the grid to read as a plan:
//
//	walls
//	+#D#+
//	#   #
//	+###+
//
// A roofs grid, laid over the terrain, lays each building's roof: s slate, t thatch or c clay
// tiles, whichever most of its rooms lie under. Tiles the grid leaves out, or marks ., lie under
// slate:
//
//	roofs
//	..ccc
//
// An objects grid, laid over the terrain, stands furniture and landmarks on its tiles, each blocking
// the way but crops, with . for none:
//
//	t  table            c  chair
//	T  throne           a  suit of armour
//	r  cooking range    o  barrel
//	x  crate            q  sacks
//	s  shelves          S  staircase
//	L  ladder           i  standing torch
//	l  candles          f  standard
//	A  altar            p  pew
//	O  organ            k  coffin
//	F  fountain         u  statue
//	w  well             h  haystack
//	n  signpost         C  cart
//	g  trough           y  scarecrow
//	v  crops            N  anvil
//	K  counter          b  bed
//
//	objects
//	..t..
package mapfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lapresregne/internal/world"
)

// tile is what one character of the terrain grid stands for.
type tile struct {
	ground world.Ground
	tree   bool
	kind   world.TreeKind
}

var tiles = map[rune]tile{
	'.': {ground: world.Grass},
	'@': {ground: world.Grass},
	'~': {ground: world.Water},
	'T': {ground: world.Grass, tree: true, kind: world.NormalTree},
	'O': {ground: world.Grass, tree: true, kind: world.Oak},
	'W': {ground: world.Grass, tree: true, kind: world.Willow},
	'Y': {ground: world.Grass, tree: true, kind: world.Yew},
	':': {ground: world.Path},
	'_': {ground: world.Floor},
	'#': {ground: world.Wall},
	'=': {ground: world.Fence},
	'G': {ground: world.Gate},
	'+': {ground: world.Grave},
	'D': {ground: world.Door},
	'b': {ground: world.Boards},
	'B': {ground: world.Battlement},
	'H': {ground: world.Bridge},
}

// fixtures are what the characters of the objects grid stand for.
var fixtures = map[rune]world.Fixture{
	't': world.Table, 'c': world.Chair, 'T': world.Throne, 'a': world.Armour, 'r': world.Range,
	'o': world.Barrel, 'x': world.Crate, 'q': world.Sacks, 's': world.Shelves, 'S': world.Staircase,
	'L': world.Ladder, 'i': world.Torch, 'l': world.Candles, 'f': world.Standard,
	'A': world.Altar, 'p': world.Pew, 'O': world.Organ, 'k': world.Coffin,
	'F': world.Fountain, 'u': world.Statue, 'w': world.Well, 'h': world.Haystack, 'n': world.Signpost,
	'C': world.Cart, 'g': world.Trough, 'y': world.Scarecrow, 'v': world.Crop, 'N': world.Anvil,
	'K': world.Counter, 'b': world.Bed,
}

// roofings are what the characters of the roofs grid stand for.
var roofings = map[rune]world.Roofing{'.': world.Slate, 's': world.Slate, 't': world.Thatch, 'c': world.ClayTiles}

// barriers are what the characters along the edges of the walls grid stand for.
var barriers = map[rune]world.Ground{
	' ': world.Grass, '.': world.Grass, '+': world.Grass,
	'#': world.Wall, 'B': world.Battlement, '=': world.Fence, 'D': world.Door,
}

// Load reads the map in the file at path. A map without a name is named after its file.
func Load(path string) (world.Map, error) {
	file, err := os.Open(path)
	if err != nil {
		return world.Map{}, err
	}
	defer file.Close()
	loaded, err := read(bufio.NewScanner(file), strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if err != nil {
		return world.Map{}, fmt.Errorf("read map %s: %w", path, err)
	}
	return loaded, nil
}

func read(lines *bufio.Scanner, name string) (world.Map, error) {
	m := world.Map{Name: name}
	// grid is the grid being read, or empty between grids.
	grid, hasTerrain, hasStart := "", false, false
	// areas names each letter of the areas grid.
	areas := map[rune]string{}
	// objectRows counts the rows of the objects grid read so far.
	objectRows := 0
	// walls are the lines of the walls grid read so far.
	var walls []string
	for number := 1; lines.Scan(); number++ {
		line := strings.TrimRight(lines.Text(), " \t\r")
		switch {
		case strings.HasPrefix(line, "//"):
		case line == "":
			grid = ""
		case grid == "terrain":
			row := len(m.Ground)
			ground := make([]world.Ground, 0, len(line))
			for col, char := range []rune(line) {
				meaning, known := tiles[char]
				if !known {
					return world.Map{}, fmt.Errorf("line %d: unknown tile %q", number, char)
				}
				ground = append(ground, meaning.ground)
				if meaning.tree {
					m.Trees = append(m.Trees, world.PlantedTree{Col: col, Row: row, Kind: meaning.kind})
				}
				if char == '@' {
					if hasStart {
						return world.Map{}, fmt.Errorf("line %d: a second start tile '@'", number)
					}
					m.StartCol, m.StartRow, hasStart = col, row, true
				}
			}
			m.Ground = append(m.Ground, ground)
		case grid == "heights":
			row := make([]int, 0, len(line))
			for _, char := range line {
				if char < '0' || char > '9' {
					return world.Map{}, fmt.Errorf("line %d: height %q is not a digit", number, char)
				}
				row = append(row, int(char-'0'))
			}
			m.Heights = append(m.Heights, row)
		case grid == "objects":
			for col, char := range []rune(line) {
				if char == '.' {
					continue
				}
				fixture, known := fixtures[char]
				if !known {
					return world.Map{}, fmt.Errorf("line %d: unknown object %q", number, char)
				}
				m.Fixtures = append(m.Fixtures, world.PlacedFixture{Col: col, Row: objectRows, Kind: fixture})
			}
			objectRows++
		case grid == "roofs":
			row := make([]world.Roofing, 0, len(line))
			for _, char := range line {
				roofing, known := roofings[char]
				if !known {
					return world.Map{}, fmt.Errorf("line %d: unknown roofing %q", number, char)
				}
				row = append(row, roofing)
			}
			m.Roofs = append(m.Roofs, row)
		case grid == "walls":
			if err := readWalls(&m.Edges, len(walls), line); err != nil {
				return world.Map{}, fmt.Errorf("line %d: %w", number, err)
			}
			walls = append(walls, line)
		case grid == "areas":
			row := make([]string, 0, len(line))
			for _, char := range []rune(line) {
				name, named := areas[char]
				if !named && char != '.' {
					return world.Map{}, fmt.Errorf("line %d: unnamed area %q", number, char)
				}
				row = append(row, name)
			}
			m.Areas = append(m.Areas, row)
		case line == "terrain":
			grid, hasTerrain = line, true
		case line == "areas" || line == "heights" || line == "objects" || line == "roofs" || line == "walls":
			grid = line
		default:
			setting, value, _ := strings.Cut(line, " ")
			switch setting {
			case "name":
				m.Name = strings.TrimSpace(value)
			case "start":
				if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d %d", &m.StartCol, &m.StartRow); err != nil {
					return world.Map{}, fmt.Errorf("line %d: want \"start <col> <row>\"", number)
				}
				if hasStart {
					return world.Map{}, fmt.Errorf("line %d: a second start", number)
				}
				hasStart = true
			case "area":
				letter, name, _ := strings.Cut(strings.TrimSpace(value), " ")
				if len([]rune(letter)) != 1 || letter == "." || strings.TrimSpace(name) == "" {
					return world.Map{}, fmt.Errorf("line %d: want \"area <letter> <name>\"", number)
				}
				areas[[]rune(letter)[0]] = strings.TrimSpace(name)
			default:
				return world.Map{}, fmt.Errorf("line %d: unknown setting %q", number, setting)
			}
		}
	}
	if err := lines.Err(); err != nil {
		return world.Map{}, err
	}
	if !hasTerrain {
		return world.Map{}, fmt.Errorf("no terrain grid")
	}
	if !hasStart {
		return world.Map{}, fmt.Errorf("no start tile '@' or start setting")
	}
	if err := diagonals(&m.Edges, walls); err != nil {
		return world.Map{}, err
	}
	trim(&m.Edges, m.Ground)
	return m, nil
}

// readWalls reads the edges along one line of the walls grid, the index-th: an even line runs
// along the north edges of a row of tiles, and an odd line along their west edges, the tiles'
// middles between them.
func readWalls(edges *world.Edges, index int, line string) error {
	row := index / 2
	grid := &edges.North
	if index%2 == 1 {
		grid = &edges.West
	}
	for len(*grid) <= row {
		*grid = append(*grid, nil)
	}
	for i, char := range []rune(line) {
		if i%2 == index%2 {
			// A corner on a north line, or a tile's middle on a west line.
			if index%2 == 1 && char != '/' && char != '\\' && char != ' ' && char != '.' {
				return fmt.Errorf("unknown wall %q across a tile", char)
			}
			continue
		}
		barrier, known := barriers[char]
		if !known {
			return fmt.Errorf("unknown wall %q", char)
		}
		col := i / 2
		for len((*grid)[row]) <= col {
			(*grid)[row] = append((*grid)[row], world.Grass)
		}
		(*grid)[row][col] = barrier
	}
	return nil
}

// diagonals reads the walls running diagonally across tiles: / rising from a tile's south-west
// corner to its north-east, \ falling from its north-west corner to its south-east, each a wall,
// battlements or a fence as the corners it joins are marked.
func diagonals(edges *world.Edges, walls []string) error {
	corner := func(line, i int) world.Ground {
		if line < 0 || line >= len(walls) || i >= len([]rune(walls[line])) {
			return world.Grass
		}
		return barriers[[]rune(walls[line])[i]]
	}
	for index := 1; index < len(walls); index += 2 {
		row := index / 2
		edges.Rising = append(edges.Rising, make([]world.Ground, len([]rune(walls[index]))/2+1))
		edges.Falling = append(edges.Falling, make([]world.Ground, len([]rune(walls[index]))/2+1))
		for i, char := range []rune(walls[index]) {
			if i%2 == 0 || char != '/' && char != '\\' {
				continue
			}
			col := i / 2
			// The kind of wall is marked at both corners the diagonal joins.
			from, to := corner(index+1, i-1), corner(index-1, i+1)
			grid := edges.Rising
			if char == '\\' {
				from, to = corner(index-1, i-1), corner(index+1, i+1)
				grid = edges.Falling
			}
			kind := world.Wall
			if from == to && (from == world.Battlement || from == world.Fence) {
				kind = from
			}
			grid[row][col] = kind
		}
	}
	return nil
}

// trim fits each grid of edges to the terrain's tiles, dropping what lies beyond them and filling
// what the grid leaves out with nothing.
func trim(edges *world.Edges, ground [][]world.Ground) {
	for _, grid := range []*[][]world.Ground{&edges.West, &edges.North, &edges.Rising, &edges.Falling} {
		if len(*grid) == 0 {
			continue
		}
		fitted := make([][]world.Ground, len(ground))
		for row := range ground {
			fitted[row] = make([]world.Ground, len(ground[row]))
			if row < len(*grid) {
				copy(fitted[row], (*grid)[row])
			}
		}
		*grid = fitted
	}
}
