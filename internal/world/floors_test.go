package world_test

import (
	"slices"
	"testing"
	"time"

	"lapresregne/internal/world"
)

// tower is a walled room of flagstones with two floors of boards above it, a staircase at column
// 5, row 2 rising through all three, and the hero beside it on the ground floor.
func tower() world.Map {
	const cols, rows = 9, 7
	grid := func() [][]world.Ground {
		ground := make([][]world.Ground, rows)
		for row := range ground {
			ground[row] = make([]world.Ground, cols)
		}
		return ground
	}
	ground, first, second := grid(), grid(), grid()
	for row := 1; row <= 4; row++ {
		for col := 2; col <= 6; col++ {
			ground[row][col], first[row][col], second[row][col] = world.Floor, world.Boards, world.Boards
		}
	}
	stairs := []world.PlacedFixture{{Col: 5, Row: 2, Kind: world.Staircase}}
	return world.Map{
		Name: "tower", Ground: ground, Fixtures: stairs, StartCol: 4, StartRow: 2,
		Upstairs: []world.UpperFloor{{Ground: first, Fixtures: stairs}, {Ground: second, Fixtures: stairs}},
	}
}

func TestAStaircaseBesideTheHeroIsInReach(t *testing.T) {
	flight, ok := world.NewOnMap(7, tower()).FlightInReach()
	if want := (world.Flight{Col: 5, Row: 2, Kind: world.Staircase, Up: true}); !ok || flight != want {
		t.Fatalf("flight in reach = %+v, %v; want %+v", flight, ok, want)
	}
}

func TestNoFlightIsInReachFarFromOne(t *testing.T) {
	m := tower()
	m.StartCol, m.StartRow = 2, 4
	if flight, ok := world.NewOnMap(7, m).FlightInReach(); ok {
		t.Fatalf("flight in reach = %+v; want none", flight)
	}
}

func TestTheHeroStartsOnTheGroundFloor(t *testing.T) {
	if storey := world.NewOnMap(7, tower()).Storey(); storey != 0 {
		t.Fatalf("storey = %d, want the ground floor", storey)
	}
}

func TestClimbingUpAStaircaseTakesTheHeroToTheFloorAbove(t *testing.T) {
	w := world.NewOnMap(7, tower())
	if !w.Climb(true) {
		t.Fatal("climbing up the staircase failed")
	}
	if storey := w.Storey(); storey != 1 {
		t.Fatalf("storey = %d, want the first floor", storey)
	}
	x, z := w.Hero().Position()
	if ground, ok := w.GroundOn(1, x, z); !ok || ground != world.Boards {
		t.Fatalf("the hero stands on %v, %v; want the first floor's boards", ground, ok)
	}
}

func TestAMiddleFloorsStaircaseLeadsUpAndDown(t *testing.T) {
	w := world.NewOnMap(7, tower())
	w.Climb(true)
	flight, ok := w.FlightInReach()
	if !ok || !flight.Up || !flight.Down {
		t.Fatalf("flight in reach = %+v, %v; want one leading up and down", flight, ok)
	}
}

func TestClimbingDownReturnsToTheFloorBelow(t *testing.T) {
	w := world.NewOnMap(7, tower())
	w.Climb(true)
	w.Climb(true)
	if !w.Climb(false) || w.Storey() != 1 {
		t.Fatalf("storey = %d after climbing down from the second floor, want the first", w.Storey())
	}
}

func TestTheTopFloorsStaircaseLeadsOnlyDown(t *testing.T) {
	w := world.NewOnMap(7, tower())
	w.Climb(true)
	w.Climb(true)
	if w.Climb(true) || w.Storey() != 2 {
		t.Fatalf("storey = %d after climbing up from the top floor, want it kept at 2", w.Storey())
	}
}

func TestTheGroundFloorsStaircaseLeadsOnlyUp(t *testing.T) {
	w := world.NewOnMap(7, tower())
	if w.Climb(false) || w.Storey() != 0 {
		t.Fatalf("storey = %d after climbing down from the ground floor, want it kept at 0", w.Storey())
	}
}

func TestUpstairsTheHeroCannotWalkOffTheFloorIntoTheAir(t *testing.T) {
	w := world.NewOnMap(7, tower())
	w.Climb(true)
	w.Advance(-1, 0, 3*time.Second, false)
	if x, _ := w.Hero().Position(); x < 2*world.TileSize {
		t.Fatalf("x = %v; want the hero kept on the first floor, which starts at x %v", x, 2*world.TileSize)
	}
}

func TestEachFloorLiesAStoreyAboveTheOneBelow(t *testing.T) {
	w := world.NewOnMap(7, tower())
	x, z := 4.5*world.TileSize, 2.5*world.TileSize
	if got, want := w.ElevationOn(2, x, z), w.Elevation(x, z)+2*world.StoreyHeight; got != want {
		t.Fatalf("elevation of the second floor = %v, want %v", got, want)
	}
}

func TestAFloorHasNoGroundWhereItIsOpenToTheAir(t *testing.T) {
	w := world.NewOnMap(7, tower())
	if ground, ok := w.GroundOn(1, 0.5*world.TileSize, 0.5*world.TileSize); ok {
		t.Fatalf("ground on the first floor beyond its boards = %v; want none", ground)
	}
}

// cornered is the tower with a wall on its first floor rising across the floor's north-west corner
// tile, column 2, row 1, from that tile's south-west corner to its north-east.
func cornered() world.Map {
	m := tower()
	rising := [][]world.Ground{make([]world.Ground, 3), {world.Grass, world.Grass, world.Wall}}
	m.Upstairs[0].Edges = world.Edges{Rising: rising}
	return m
}

func TestAFloorEndsAtAWallAcrossItsCornerTile(t *testing.T) {
	w := world.NewOnMap(7, cornered())
	if ground, ok := w.GroundOn(1, 2.2*world.TileSize, 1.2*world.TileSize); ok {
		t.Fatalf("ground on the first floor outside the diagonal wall = %v; want none", ground)
	}
}

func TestAFloorRunsUpToAWallAcrossItsCornerTile(t *testing.T) {
	w := world.NewOnMap(7, cornered())
	if ground, ok := w.GroundOn(1, 2.8*world.TileSize, 1.8*world.TileSize); !ok || ground != world.Boards {
		t.Fatalf("ground on the first floor inside the diagonal wall = %v, %v; want boards", ground, ok)
	}
}

func TestApproachingAStaircaseClimbsItOnArrival(t *testing.T) {
	m := tower()
	m.StartCol, m.StartRow = 2, 4
	w := world.NewOnMap(7, m)
	w.ApproachFlight(world.Flight{Col: 5, Row: 2, Kind: world.Staircase, Up: true})
	walkFor(w, 3*time.Second)
	if w.Storey() != 1 {
		t.Fatalf("storey = %d, want the hero up the staircase they walked to", w.Storey())
	}
}

func TestApproachingAStaircaseLeadingBothWaysOffersTheChoice(t *testing.T) {
	w := world.NewOnMap(7, tower())
	w.Climb(true)
	w.WalkTo(2.5*world.TileSize, 4.5*world.TileSize)
	walkFor(w, 3*time.Second)
	w.ApproachFlight(world.Flight{Col: 5, Row: 2, Kind: world.Staircase, Up: true, Down: true})
	walkFor(w, 3*time.Second)
	if w.Storey() != 1 {
		t.Fatalf("storey = %d, want the hero kept on the first floor until they choose", w.Storey())
	}
	if !w.OfferedClimb() {
		t.Fatal("arriving at a staircase leading both ways must offer the choice")
	}
	if w.OfferedClimb() {
		t.Fatal("the choice must be offered once")
	}
}

func TestResumingReturnsTheHeroToTheFloorTheyStoodOn(t *testing.T) {
	kept := world.NewOnMap(7, tower())
	kept.Climb(true)
	progress := &memoryProgress{}
	kept.Keep(progress, quitAt)
	w, err := world.ResumeOn(tower(), 7, progress, quitAt)
	if err != nil {
		t.Fatal(err)
	}
	if w.Storey() != 1 {
		t.Fatalf("storey = %d, want the first floor the hero was saved on", w.Storey())
	}
	x, z := w.Hero().Position()
	if keptX, keptZ := kept.Hero().Position(); x != keptX || z != keptZ {
		t.Fatalf("position = %v, %v; want %v, %v", x, z, keptX, keptZ)
	}
}

func TestResumingOnAFloorTheMapNoLongerHasStartsAfresh(t *testing.T) {
	kept := world.NewOnMap(7, tower())
	kept.Climb(true)
	kept.Climb(true)
	progress := &memoryProgress{}
	kept.Keep(progress, quitAt)
	shorter := tower()
	shorter.Upstairs = shorter.Upstairs[:1]
	w, err := world.ResumeOn(shorter, 7, progress, quitAt)
	if err != nil {
		t.Fatal(err)
	}
	if w.Storey() != 0 {
		t.Fatalf("storey = %d, want the hero back on the ground floor", w.Storey())
	}
	if x, z := w.Hero().Position(); x != 4.5*world.TileSize || z != 2.5*world.TileSize {
		t.Fatalf("position = %v, %v; want the start tile", x, z)
	}
}

func TestClimbingSoundsTheStaircaseClimbed(t *testing.T) {
	w := world.NewOnMap(7, tower())
	sounds := &recordedSounds{}
	w.SetSounds(sounds)
	w.Climb(false)
	w.Climb(true)
	if want := []world.Fixture{world.Staircase}; !slices.Equal(sounds.climbs, want) {
		t.Fatalf("climbs = %v, want only the climb up the staircase sounded", sounds.climbs)
	}
}

func TestABuildingRisesAsManyFloorsAsStandAboveItsRooms(t *testing.T) {
	m := lodge(world.Wall)
	first := make([][]world.Ground, len(m.Ground))
	for row := range first {
		first[row] = make([]world.Ground, len(m.Ground[row]))
	}
	first[2][3] = world.Boards
	m.Upstairs = []world.UpperFloor{{Ground: first}}
	buildings := world.NewOnMap(7, m).Buildings()
	if len(buildings) != 1 || buildings[0].Above != 1 {
		t.Fatalf("buildings = %+v, want the lodge rising a floor above its ground floor", buildings)
	}
}

func TestABuildingWithNoFloorAboveRisesNoHigher(t *testing.T) {
	buildings := world.NewOnMap(7, lodge(world.Wall)).Buildings()
	if len(buildings) != 1 || buildings[0].Above != 0 {
		t.Fatalf("buildings = %+v, want the lodge with no floor above its ground floor", buildings)
	}
}

// stairwell is the tower with a staircase three tiles long, on column 5, rows 1 to 3, rising
// through every floor, its north end hemmed in by crates and a wall, and the hero in the room's
// far corner.
func stairwell() world.Map {
	m := tower()
	stairs := []world.PlacedFixture{{Col: 5, Row: 1, Kind: world.Staircase}, {Col: 5, Row: 2, Kind: world.Staircase}, {Col: 5, Row: 3, Kind: world.Staircase}}
	m.Fixtures = append(slices.Clone(stairs),
		world.PlacedFixture{Col: 4, Row: 1, Kind: world.Crate}, world.PlacedFixture{Col: 6, Row: 1, Kind: world.Crate},
		world.PlacedFixture{Col: 4, Row: 2, Kind: world.Crate}, world.PlacedFixture{Col: 6, Row: 2, Kind: world.Crate})
	m.Ground[0][4], m.Ground[0][5], m.Ground[0][6] = world.Wall, world.Wall, world.Wall
	for i := range m.Upstairs {
		m.Upstairs[i].Fixtures = stairs
	}
	m.StartCol, m.StartRow = 2, 4
	return m
}

func TestApproachingAStaircasesFarTileClimbsItFromBesideItsNearOne(t *testing.T) {
	w := world.NewOnMap(7, stairwell())
	w.ApproachFlight(world.Flight{Col: 5, Row: 1, Kind: world.Staircase})
	walkFor(w, 5*time.Second)
	if w.Storey() != 1 {
		x, z := w.Hero().Position()
		t.Fatalf("storey = %d with the hero at %.1f, %.1f; want them up the staircase", w.Storey(), x, z)
	}
}

// walledStairs is the lodge with a staircase inside against its west wall, on tile 2, 2, and a
// floor above with its top, and the hero outside the wall, beside the staircase.
func walledStairs() world.Map {
	m := lodge(world.Wall)
	stairs := []world.PlacedFixture{{Col: 2, Row: 2, Kind: world.Staircase}}
	first := make([][]world.Ground, len(m.Ground))
	for row := range first {
		first[row] = make([]world.Ground, len(m.Ground[row]))
		copy(first[row], m.Ground[row])
	}
	m.Fixtures = stairs
	m.Upstairs = []world.UpperFloor{{Ground: first, Edges: m.Edges, Fixtures: stairs}}
	m.StartCol, m.StartRow = 1, 2
	return m
}

func TestAStaircaseBehindAWallIsOutOfReach(t *testing.T) {
	w := world.NewOnMap(7, walledStairs())
	if flight, ok := w.FlightInReach(); ok {
		t.Fatalf("flight in reach = %+v; want none through the wall", flight)
	}
	if w.Climb(true) {
		t.Fatal("the hero climbed a staircase through a wall")
	}
}

func TestApproachingAStaircaseBehindAWallGoesRoundThroughTheDoor(t *testing.T) {
	w := world.NewOnMap(7, walledStairs())
	w.ApproachFlight(world.Flight{Col: 2, Row: 2, Kind: world.Staircase})
	walkFor(w, 10*time.Second)
	if w.Storey() != 1 {
		x, z := w.Hero().Position()
		t.Fatalf("storey = %d with the hero at %.1f, %.1f; want them in through the door and up the staircase", w.Storey(), x, z)
	}
	if x, _ := w.Hero().Position(); x < 2*world.TileSize {
		t.Fatalf("x = %v; want the hero to have climbed from inside the lodge", x)
	}
}
