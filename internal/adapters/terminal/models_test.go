package terminal

import (
	"strings"
	"testing"

	"lapresregne/internal/world"
)

// tableModel is a model of a table: a slab on the floor 1.6 by 1.2 across and 0.8 tall, its top
// a dark red the palette has no ink for, its sides brown.
func tableModel() *world.Model {
	top, side := world.Color{R: 140, G: 30, B: 40}, world.Color{R: 120, G: 80, B: 50}
	points := []world.Point{
		{X: -0.8, Y: 0, Z: -0.6}, {X: 0.8, Y: 0, Z: -0.6}, {X: 0.8, Y: 0, Z: 0.6}, {X: -0.8, Y: 0, Z: 0.6},
		{X: -0.8, Y: 0.8, Z: -0.6}, {X: 0.8, Y: 0.8, Z: -0.6}, {X: 0.8, Y: 0.8, Z: 0.6}, {X: -0.8, Y: 0.8, Z: 0.6},
	}
	faces := []world.ModelFace{
		{A: 4, B: 7, C: 6, Color: top}, {A: 4, B: 6, C: 5, Color: top},
		{A: 3, B: 2, C: 6, Color: side}, {A: 3, B: 6, C: 7, Color: side},
		{A: 0, B: 3, C: 7, Color: side}, {A: 0, B: 7, C: 4, Color: side},
		{A: 2, B: 1, C: 5, Color: side}, {A: 2, B: 5, C: 6, Color: side},
		{A: 1, B: 0, C: 4, Color: side}, {A: 1, B: 4, C: 5, Color: side},
	}
	return &world.Model{Points: points, Faces: faces}
}

// modelledHall is the hall with a table on the tile two north of the hero, shaped by its model.
func modelledHall() *world.World {
	ground := make([][]world.Ground, 30)
	for row := range ground {
		ground[row] = make([]world.Ground, 40)
		for col := range ground[row] {
			ground[row][col] = world.Floor
		}
	}
	return world.NewOnMap(7, world.Map{
		Name: "Hall", Ground: ground, StartCol: 20, StartRow: 15,
		Fixtures: []world.PlacedFixture{{Col: 20, Row: 13, Kind: world.Table}},
		Models:   []world.PlacedModel{{Col: 20, Row: 13, Cols: 1, Rows: 1, Model: tableModel()}},
	})
}

func TestAModelledFixtureIsDrawnAsItsModelInItsOwnColours(t *testing.T) {
	r := newRenderer(modelledHall(), NewCamera(0, 0), "L’Après-Règne")
	r.zoom = zoom{current: 6}
	c := r.render(160, 90, 0)
	top := r.inkOf(world.Color{R: 140, G: 30, B: 40})
	if found := inkNear(c, 20.5*world.TileSize, 0.8, 13.5*world.TileSize, top); found < 10 {
		t.Fatalf("%d of 25 pixels of the model's top on the table's tile, want it drawn", found)
	}
	if countInk(c, plankInk) != 0 {
		t.Fatal("the table drew its box model too, want only the game's model")
	}
	if frame := r.frame(160, 90, 0); !strings.Contains(frame, foreground([3]int{140, 30, 40})) {
		t.Fatal("the model's top must show in its own colour")
	}
}

func TestAModelHungOnAWallLeavesTheFixturesOnItsTileAsTheyAre(t *testing.T) {
	ground := make2D(30, 40)
	w := world.NewOnMap(7, world.Map{
		Name: "Hall", Ground: ground, StartCol: 20, StartRow: 15,
		Fixtures: []world.PlacedFixture{{Col: 20, Row: 13, Kind: world.Table}},
		Models:   []world.PlacedModel{{Col: 20, Row: 13, Cols: 1, Rows: 1, Model: tableModel(), Hangs: true}},
	})
	_, c := closeView(w)
	if countInk(c, plankInk) == 0 {
		t.Fatal("the table must still stand under the model hung on the wall")
	}
}
