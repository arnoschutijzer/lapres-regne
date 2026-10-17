package terminal

import (
	"testing"

	"lapresregne/internal/world"
)

// edgedLodge is a meadow with a stone-floored room on tiles 18, 10 to 22, 13, walled along the
// edges round it in a barrier of a kind, with a doorway in the middle of its south side and the
// hero outside, two tiles south of the doorway.
func edgedLodge(barrier world.Ground) *world.World {
	const cols, rows = 40, 30
	ground := make([][]world.Ground, rows)
	edges := world.Edges{
		West: make([][]world.Ground, rows), North: make([][]world.Ground, rows),
		Rising: make([][]world.Ground, rows), Falling: make([][]world.Ground, rows),
	}
	for row := range rows {
		ground[row] = make([]world.Ground, cols)
		edges.West[row], edges.North[row] = make([]world.Ground, cols), make([]world.Ground, cols)
		edges.Rising[row], edges.Falling[row] = make([]world.Ground, cols), make([]world.Ground, cols)
	}
	for row := 10; row <= 13; row++ {
		for col := 18; col <= 22; col++ {
			ground[row][col] = world.Floor
		}
		edges.West[row][18], edges.West[row][23] = barrier, barrier
	}
	for col := 18; col <= 22; col++ {
		edges.North[10][col], edges.North[14][col] = barrier, barrier
	}
	edges.North[14][20] = world.Door
	return world.NewOnMap(7, world.Map{Name: "Lodge", Ground: ground, Edges: edges, StartCol: 20, StartRow: 16})
}

// closeView renders a world zoomed in.
func closeView(w *world.World) (*renderer, *forestCanvas) {
	r := newRenderer(w, "L’Après-Règne")
	r.zoom = zoom{current: 6}
	return r, r.render(160, 90, 0)
}

var stoneInks = []int{stoneInk, shadedStoneInk, mossInk, glassInk, plankInk}

func TestAWallStandsAlongTheEdgeBetweenTiles(t *testing.T) {
	_, c := closeView(edgedLodge(world.Wall))
	// The south wall runs along z 28, from x 36 to 46; its face rises at x 39.
	if found := inkNear(c, 39, 1.5, 28.36, stoneInks...); found < 10 {
		t.Fatalf("%d of 25 pixels of stone up the south wall's face, want it standing along the edge", found)
	}
}

func TestADoorwayAlongAnEdgeOpensUnderItsLintel(t *testing.T) {
	_, c := closeView(edgedLodge(world.Wall))
	// The doorway spans x 40 to 42 along z 28.
	if found := inkNear(c, 41, (doorHeight+wallHeight)/2, 28.36, stoneInk, shadedStoneInk); found < 10 {
		t.Fatalf("%d of 25 pixels of stone above the doorway, want its lintel", found)
	}
	if found := inkNear(c, 41.3, doorHeight/2, 28.36, stoneInk, shadedStoneInk); found > 5 {
		t.Fatalf("%d of 25 pixels of stone in the doorway, want it open", found)
	}
}

func TestABattlementAlongAnEdgeIsCrownedWithMerlons(t *testing.T) {
	_, c := closeView(edgedLodge(world.Battlement))
	merlons := 0
	for x := 36.25; x < 40; x += 0.5 {
		merlons += inkNear(c, x, wallHeight+copingHeight+merlonHeight*0.6, 28.36, stoneInk, shadedStoneInk)
	}
	if merlons < 10 {
		t.Fatalf("%d pixels of merlons along the south wall's top, want it crowned", merlons)
	}
}

func TestAFenceAlongAnEdgeIsRailed(t *testing.T) {
	_, c := closeView(edgedLodge(world.Fence))
	if found := inkNear(c, 38, 0.9, 28, plankInk); found < 3 {
		t.Fatalf("%d of 25 pixels of planks at the fence's top rail, want it railed", found)
	}
}

func TestADiagonalWallRunsAcrossItsTile(t *testing.T) {
	w := edgedLodge(world.Wall)
	edges := world.Edges{Rising: make([][]world.Ground, 30)}
	edges.Rising[16] = make([]world.Ground, 40)
	edges.Rising[16][17] = world.Wall
	w = world.NewOnMap(7, world.Map{Name: "Diagonal", Ground: make2D(30, 40), Edges: edges, StartCol: 20, StartRow: 16})
	_, c := closeView(w)
	if found := inkNear(c, 35, 1.5, 33, stoneInks...); found < 10 {
		t.Fatalf("%d of 25 pixels of stone in the middle of the tile, want the wall running across it", found)
	}
}

// make2D is a meadow of rows by cols tiles.
func make2D(rows, cols int) [][]world.Ground {
	ground := make([][]world.Ground, rows)
	for row := range ground {
		ground[row] = make([]world.Ground, cols)
	}
	return ground
}

func TestARoomWalledAlongItsEdgesIsRoofedFromOutside(t *testing.T) {
	_, c := closeView(edgedLodge(world.Wall))
	if found := inkNear(c, 41, wallHeight+copingHeight+1.0, 24, roofInks...); found < 10 {
		t.Fatalf("%d of 25 pixels of roof over the room's middle, want it roofed", found)
	}
}

func TestFurnitureBacksOntoAWallAlongTheEdgeBesideIt(t *testing.T) {
	r, _ := closeView(edgedLodge(world.Wall))
	tile := r.built.at(18, 11)
	tile.furnished, tile.fixture = true, world.Shelves
	if back := backOf(tile); back != 3 {
		t.Fatalf("shelves beside the west wall back onto side %d, want west", back)
	}
}
