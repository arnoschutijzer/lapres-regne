package terminal

import (
	"slices"
	"testing"

	"lapresregne/internal/world"
)

// tower is a meadow with a stone-floored room on tiles 18, 10 to 22, 13, walled along the edges
// round it, and floors of boards above it, as many as upstairs asks for, each walled the same. A
// staircase on tile 22, 11 rises through every floor, and the hero stands beside it on the ground
// floor.
func tower(upstairs int) *world.World {
	return world.NewOnMap(7, towerMap(upstairs))
}

// towerMap is the map of the tower.
func towerMap(upstairs int) world.Map {
	const cols, rows = 40, 30
	floor := func(ground world.Ground) ([][]world.Ground, world.Edges) {
		tiles := make([][]world.Ground, rows)
		edges := world.Edges{
			West: make([][]world.Ground, rows), North: make([][]world.Ground, rows),
			Rising: make([][]world.Ground, rows), Falling: make([][]world.Ground, rows),
		}
		for row := range rows {
			tiles[row] = make([]world.Ground, cols)
			edges.West[row], edges.North[row] = make([]world.Ground, cols), make([]world.Ground, cols)
			edges.Rising[row], edges.Falling[row] = make([]world.Ground, cols), make([]world.Ground, cols)
		}
		for row := 10; row <= 13; row++ {
			for col := 18; col <= 22; col++ {
				tiles[row][col] = ground
			}
			edges.West[row][18], edges.West[row][23] = world.Wall, world.Wall
		}
		for col := 18; col <= 22; col++ {
			edges.North[10][col], edges.North[14][col] = world.Wall, world.Wall
		}
		return tiles, edges
	}
	stairs := []world.PlacedFixture{{Col: 22, Row: 11, Kind: world.Staircase}}
	ground, edges := floor(world.Floor)
	m := world.Map{Name: "Tower", Ground: ground, Edges: edges, Fixtures: stairs, StartCol: 21, StartRow: 11}
	for range upstairs {
		boards, walls := floor(world.Boards)
		m.Upstairs = append(m.Upstairs, world.UpperFloor{Ground: boards, Edges: walls, Fixtures: stairs})
	}
	return m
}

func TestTheFloorTheHeroClimbedToIsDrawnAStoreyUp(t *testing.T) {
	w := tower(1)
	w.Climb(true)
	_, c := closeView(w)
	c.lift = 0
	// The middle of tile 19, 10, on the first floor, clear of the walls in front of it.
	if found := inkNear(c, 39, world.StoreyHeight, 21, boardsInk, inShadow(boardsInk)); found < 10 {
		t.Fatalf("%d of 25 pixels of boards a storey up, want the first floor drawn there", found)
	}
}

func TestTheGroundFloorIsDrawnWhileTheHeroStandsOnIt(t *testing.T) {
	_, c := closeView(tower(1))
	c.lift = 0
	if found := inkNear(c, 39, 0, 21, floorInk, inShadow(floorInk)); found < 10 {
		t.Fatalf("%d of 25 pixels of flagstones on the ground floor, want the floors above it hidden", found)
	}
}

func TestEOffersToClimbUpTheStaircaseInReach(t *testing.T) {
	r, _ := closeView(tower(1))
	r.toggleMenu()
	if r.menuTitle != "Staircase" || !slices.Equal(r.menu, []string{"Climb-up"}) {
		t.Fatalf("menu = %q %q, want the staircase offered to climb up", r.menuTitle, r.menu)
	}
}

func TestChoosingClimbUpTakesTheHeroUpstairs(t *testing.T) {
	r, _ := closeView(tower(1))
	r.toggleMenu()
	r.choose(0)
	if r.world.Storey() != 1 {
		t.Fatalf("storey = %d, want the hero up the staircase", r.world.Storey())
	}
}

func TestAMiddleFloorsStaircaseOffersToClimbUpOrDown(t *testing.T) {
	w := tower(2)
	w.Climb(true)
	r, _ := closeView(w)
	r.toggleMenu()
	if !slices.Equal(r.menu, []string{"Climb-up", "Climb-down"}) {
		t.Fatalf("menu = %q, want the staircase offered to climb up or down", r.menu)
	}
	r.choose(1)
	if w.Storey() != 0 {
		t.Fatalf("storey = %d after choosing to climb down, want the ground floor", w.Storey())
	}
}

func TestFlightAtPicksTheStaircaseUnderAClick(t *testing.T) {
	r := newRenderer(tower(1), "L’Après-Règne")
	r.zoom = zoom{current: 6}
	r.frame(160, 60, 0)
	r.canvas.lift = 0
	column, row := cellOf(r, 45, 0, 23)
	if flight, ok := r.flightAt(column, row); !ok || flight.Col != 22 || flight.Row != 11 {
		t.Fatalf("flightAt(%d, %d) = %+v, %v; want the staircase on tile 22, 11", column, row, flight, ok)
	}
}

func TestAModelOnTheFloorTheHeroClimbedToIsDrawnAStoreyUp(t *testing.T) {
	m := towerMap(1)
	m.Upstairs[0].Fixtures = append(m.Upstairs[0].Fixtures, world.PlacedFixture{Col: 19, Row: 10, Kind: world.Table})
	m.Upstairs[0].Models = []world.PlacedModel{{Col: 19, Row: 10, Cols: 1, Rows: 1, Model: tableModel()}}
	w := world.NewOnMap(7, m)
	w.Climb(true)
	r, c := closeView(w)
	c.lift = 0
	top := r.inkOf(world.Color{R: 140, G: 30, B: 40})
	if found := inkNear(c, 39, world.StoreyHeight+0.8, 21, top); found < 10 {
		t.Fatalf("%d of 25 pixels of the table's top a storey up, want its model drawn on the first floor", found)
	}
}
