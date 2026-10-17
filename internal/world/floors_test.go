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
