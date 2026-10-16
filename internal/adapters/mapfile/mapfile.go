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
//	@  where the hero starts, on grass
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
		case line == "terrain":
			grid, hasTerrain = line, true
		default:
			setting, value, _ := strings.Cut(line, " ")
			if setting != "name" {
				return world.Map{}, fmt.Errorf("line %d: unknown setting %q", number, setting)
			}
			m.Name = strings.TrimSpace(value)
		}
	}
	if err := lines.Err(); err != nil {
		return world.Map{}, err
	}
	if !hasTerrain {
		return world.Map{}, fmt.Errorf("no terrain grid")
	}
	if !hasStart {
		return world.Map{}, fmt.Errorf("no start tile '@'")
	}
	return m, nil
}
