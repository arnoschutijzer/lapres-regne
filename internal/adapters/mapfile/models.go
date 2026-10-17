package mapfile

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"lapresregne/internal/world"
)

// readModels reads a models file, which shapes fixtures as the game models them:
//
//	model table
//	point -1 0 -0.5
//	point 1 0 -0.5
//	point 0 0.8 0.5
//	face 0 1 2 785032
//
//	place table 1 0 2 1
//	hang banner 3 0
//
// Each model lists its points, in world units about the middle of the ground it stands on with y
// upward, then its triangles between three points by their index, each in a colour as six hex
// digits. A place line stands a model over a block of tiles: its north-west tile's column and
// row, then how many tiles it spans across and down. A hang line hangs a model on the walls of a
// tile, about its middle, shaping none of the fixtures standing there.
func readModels(path string) ([]world.PlacedModel, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	models := map[string]*world.Model{}
	var current *world.Model
	var placed []world.PlacedModel
	lines := bufio.NewScanner(file)
	for number := 1; lines.Scan(); number++ {
		fields := strings.Fields(lines.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "//") {
			continue
		}
		mistake := func(want string) error {
			return fmt.Errorf("%s line %d: want %q", path, number, want)
		}
		switch fields[0] {
		case "model":
			if len(fields) != 2 {
				return nil, mistake("model <name>")
			}
			current = &world.Model{}
			models[fields[1]] = current
		case "point":
			x, y, z, ok := three(fields[1:])
			if current == nil || !ok {
				return nil, mistake("point <x> <y> <z>, after a model")
			}
			current.Points = append(current.Points, world.Point{X: x, Y: y, Z: z})
		case "face":
			if current == nil {
				return nil, mistake("face <a> <b> <c> <rrggbb>, between points of the model")
			}
			face, ok := readFace(fields[1:], len(current.Points))
			if !ok {
				return nil, mistake("face <a> <b> <c> <rrggbb>, between points of the model")
			}
			current.Faces = append(current.Faces, face)
		case "place":
			ints, ok := integers(fields[2:])
			model, known := models[safe(fields, 1)]
			if !ok || len(ints) != 4 || !known {
				return nil, mistake("place <model> <col> <row> <cols> <rows>, after the model")
			}
			placed = append(placed, world.PlacedModel{Col: ints[0], Row: ints[1], Cols: ints[2], Rows: ints[3], Model: model})
		case "hang":
			ints, ok := integers(fields[2:])
			model, known := models[safe(fields, 1)]
			if !ok || len(ints) != 2 || !known {
				return nil, mistake("hang <model> <col> <row>, after the model")
			}
			placed = append(placed, world.PlacedModel{Col: ints[0], Row: ints[1], Cols: 1, Rows: 1, Model: model, Hangs: true})
		default:
			return nil, mistake("model, point, face, place or hang")
		}
	}
	return placed, lines.Err()
}

// safe is a field of a line, or "" past its end.
func safe(fields []string, i int) string {
	if i < len(fields) {
		return fields[i]
	}
	return ""
}

func three(fields []string) (x, y, z float64, ok bool) {
	if len(fields) != 3 {
		return 0, 0, 0, false
	}
	var values [3]float64
	for i, field := range fields {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return 0, 0, 0, false
		}
		values[i] = value
	}
	return values[0], values[1], values[2], true
}

func integers(fields []string) ([]int, bool) {
	values := make([]int, len(fields))
	for i, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil {
			return nil, false
		}
		values[i] = value
	}
	return values, true
}

// readFace reads a triangle between three of a model's points and its colour.
func readFace(fields []string, points int) (world.ModelFace, bool) {
	if len(fields) != 4 || len(fields[3]) != 6 {
		return world.ModelFace{}, false
	}
	corners, ok := integers(fields[:3])
	if !ok {
		return world.ModelFace{}, false
	}
	for _, corner := range corners {
		if corner < 0 || corner >= points {
			return world.ModelFace{}, false
		}
	}
	rgb, err := strconv.ParseUint(fields[3], 16, 32)
	if err != nil {
		return world.ModelFace{}, false
	}
	color := world.Color{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb)}
	return world.ModelFace{A: corners[0], B: corners[1], C: corners[2], Color: color}, true
}
