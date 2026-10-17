package world_test

import (
	"reflect"
	"testing"
	"time"

	"lapresregne/internal/world"
)

// lodge is a meadow with a stone-floored room on tiles 2, 1 to 4, 3, walled along the edges
// round it with a doorway in the middle of its south side, and the hero outside to the south.
func lodge(barrier world.Ground) world.Map {
	const cols, rows = 9, 7
	ground := make([][]world.Ground, rows)
	edges := world.Edges{West: make([][]world.Ground, rows), North: make([][]world.Ground, rows)}
	for row := range rows {
		ground[row] = make([]world.Ground, cols)
		edges.West[row] = make([]world.Ground, cols)
		edges.North[row] = make([]world.Ground, cols)
	}
	for row := 1; row <= 3; row++ {
		for col := 2; col <= 4; col++ {
			ground[row][col] = world.Floor
		}
		edges.West[row][2], edges.West[row][5] = barrier, barrier
	}
	for col := 2; col <= 4; col++ {
		edges.North[1][col], edges.North[4][col] = barrier, barrier
	}
	edges.North[4][3] = world.Door
	return world.Map{Name: "lodge", Ground: ground, Edges: edges, StartCol: 3, StartRow: 5}
}

func TestAnEdgeWallStopsTheHeroCrossingIt(t *testing.T) {
	m := lodge(world.Wall)
	m.StartCol, m.StartRow = 1, 2
	w := world.NewOnMap(7, m)
	w.Advance(1, 0, 2*time.Second, false)
	if x, _ := w.Hero().Position(); x > 2*world.TileSize {
		t.Fatalf("x = %v; want the hero stopped outside the wall along x %v", x, 2*world.TileSize)
	}
}

func TestTheHeroWalksInThroughADoorwayInAnEdgeWall(t *testing.T) {
	w := world.NewOnMap(7, lodge(world.Wall))
	w.Advance(0, -1, 2*time.Second, false)
	if _, z := w.Hero().Position(); z > 4*world.TileSize {
		t.Fatalf("z = %v; want the hero in through the doorway", z)
	}
}

func TestAFenceAlongAnEdgeStopsTheHero(t *testing.T) {
	m := lodge(world.Fence)
	m.StartCol, m.StartRow = 1, 2
	w := world.NewOnMap(7, m)
	w.Advance(1, 0, 2*time.Second, false)
	if x, _ := w.Hero().Position(); x > 2*world.TileSize {
		t.Fatalf("x = %v; want the hero stopped outside the fence", x)
	}
}

func TestADiagonalWallBarsTheTileItCrosses(t *testing.T) {
	m := lodge(world.Wall)
	m.Edges.Rising = make([][]world.Ground, 7)
	m.Edges.Rising[5] = make([]world.Ground, 9)
	m.Edges.Rising[5][5] = world.Wall
	w := world.NewOnMap(7, m)
	w.Advance(1, 0, 2*time.Second, false)
	if x, _ := w.Hero().Position(); x > 5*world.TileSize {
		t.Fatalf("x = %v; want the hero kept off the tile the diagonal wall crosses", x)
	}
}

func TestWalkingIntoAWalledRoomGoesRoundToItsDoorway(t *testing.T) {
	m := lodge(world.Wall)
	m.StartCol, m.StartRow = 0, 0
	w := world.NewOnMap(7, m)
	w.WalkTo(3.5*world.TileSize, 2.5*world.TileSize)
	walkFor(w, 15*time.Second)
	if x, z := w.Hero().Position(); x != 3.5*world.TileSize || z != 2.5*world.TileSize {
		t.Fatalf("hero at %v, %v; want them in the middle of the room through its doorway", x, z)
	}
}

func TestARoomWalledAlongItsEdgesIsARoofedBuilding(t *testing.T) {
	w := world.NewOnMap(7, lodge(world.Wall))
	want := []world.Building{{Col: 2, Row: 1, Cols: 3, Rows: 3, Edged: true}}
	if buildings := w.Buildings(); !reflect.DeepEqual(buildings, want) {
		t.Fatalf("buildings = %+v, want %+v", buildings, want)
	}
	if w.Indoors() {
		t.Fatal("the hero starts outside the room")
	}
	w.Advance(0, -1, 2*time.Second, false)
	if !w.Indoors() {
		t.Fatal("the hero has walked into the room")
	}
}

func TestARoomFencedInHasNoRoof(t *testing.T) {
	if buildings := world.NewOnMap(7, lodge(world.Fence)).Buildings(); len(buildings) != 0 {
		t.Fatalf("buildings = %+v, want none, as a fence closes in no building", buildings)
	}
}

func TestARoomWithDiagonalWallsAcrossItsCornersIsARoofedBuilding(t *testing.T) {
	m := lodge(world.Wall)
	m.Edges.Rising, m.Edges.Falling = make([][]world.Ground, 7), make([][]world.Ground, 7)
	for row := range 7 {
		m.Edges.Rising[row], m.Edges.Falling[row] = make([]world.Ground, 9), make([]world.Ground, 9)
	}
	// The room's corners are cut off by walls running diagonally across the corner tiles, the
	// tiles still floored beneath them, as the game's octagonal towers are.
	m.Edges.West[1][2], m.Edges.North[1][2], m.Edges.Rising[1][2] = world.Grass, world.Grass, world.Wall
	m.Edges.West[1][5], m.Edges.North[1][4], m.Edges.Falling[1][4] = world.Grass, world.Grass, world.Wall
	m.Edges.West[3][2], m.Edges.North[4][2], m.Edges.Falling[3][2] = world.Grass, world.Grass, world.Wall
	m.Edges.West[3][5], m.Edges.North[4][4], m.Edges.Rising[3][4] = world.Grass, world.Grass, world.Wall
	if buildings := world.NewOnMap(7, m).Buildings(); len(buildings) != 1 {
		t.Fatalf("buildings = %+v, want the room roofed, its cut corners closing it in", buildings)
	}
}
