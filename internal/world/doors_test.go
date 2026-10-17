package world_test

import (
	"testing"
	"time"

	"lapresregne/internal/world"
)

// atTheDoor is the lodge with the hero just outside its doorway, facing it.
func atTheDoor() *world.World {
	w := world.NewOnMap(7, lodge(world.Wall))
	w.WalkTo(3.5*world.TileSize, 4.7*world.TileSize)
	walkFor(w, 3*time.Second)
	return w
}

func TestTheDoorwayBesideTheHeroIsInReach(t *testing.T) {
	doorway, ok := atTheDoor().DoorwayInReach()
	if want := (world.Doorway{Col: 3, Row: 4, North: true}); !ok || doorway != want {
		t.Fatalf("doorway in reach = %+v, %v; want %+v", doorway, ok, want)
	}
}

func TestNoDoorwayIsInReachFarFromOne(t *testing.T) {
	m := lodge(world.Wall)
	m.StartCol, m.StartRow = 7, 6
	if doorway, ok := world.NewOnMap(7, m).DoorwayInReach(); ok {
		t.Fatalf("doorway in reach = %+v; want none", doorway)
	}
}

func TestADoorStandsOpenUntilClosed(t *testing.T) {
	w := atTheDoor()
	if w.EdgesAt(3.5*world.TileSize, 4.5*world.TileSize).NorthShut {
		t.Fatal("the door starts shut; want it open")
	}
	if !w.ToggleDoor() {
		t.Fatal("closing the door in reach failed")
	}
	if !w.EdgesAt(3.5*world.TileSize, 4.5*world.TileSize).NorthShut {
		t.Fatal("the door stands open after closing it")
	}
}

func TestAShutDoorKeepsTheHeroOut(t *testing.T) {
	w := atTheDoor()
	w.ToggleDoor()
	w.Advance(0, -1, 2*time.Second, false)
	if _, z := w.Hero().Position(); z < 4*world.TileSize {
		t.Fatalf("z = %v; want the hero kept outside the shut door", z)
	}
}

func TestOpeningAShutDoorLetsTheHeroIn(t *testing.T) {
	w := atTheDoor()
	w.ToggleDoor()
	w.ToggleDoor()
	w.Advance(0, -1, 2*time.Second, false)
	if _, z := w.Hero().Position(); z > 4*world.TileSize {
		t.Fatalf("z = %v; want the hero in through the opened door", z)
	}
}

func TestWalkingToARoomBehindAShutDoorStopsOutside(t *testing.T) {
	w := atTheDoor()
	w.ToggleDoor()
	w.WalkTo(3.5*world.TileSize, 2.5*world.TileSize)
	walkFor(w, 15*time.Second)
	if _, z := w.Hero().Position(); z < 4*world.TileSize {
		t.Fatalf("z = %v; want the hero stopped outside the shut door", z)
	}
}

func TestTheHeroCannotCloseADoorTheyStandIn(t *testing.T) {
	w := world.NewOnMap(7, lodge(world.Wall))
	w.WalkTo(3.5*world.TileSize, 4*world.TileSize)
	walkFor(w, 3*time.Second)
	if w.ToggleDoor() {
		t.Fatal("the door closed on the hero standing in its doorway")
	}
	if w.Message() == "" {
		t.Fatal("want a message saying the hero is in the way")
	}
}

func TestApproachingADoorwayWalksThereAndClosesIt(t *testing.T) {
	w := world.NewOnMap(7, lodge(world.Wall))
	w.ApproachDoor(world.Doorway{Col: 3, Row: 4, North: true})
	walkFor(w, 3*time.Second)
	if !w.EdgesAt(3.5*world.TileSize, 4.5*world.TileSize).NorthShut {
		t.Fatal("the door stands open after approaching it")
	}
	if _, z := w.Hero().Position(); z < 4*world.TileSize {
		t.Fatalf("z = %v; want the hero still outside, on their side of the door", z)
	}
}

func TestApproachingAShutDoorOpensIt(t *testing.T) {
	w := atTheDoor()
	w.ToggleDoor()
	w.ApproachDoor(world.Doorway{Col: 3, Row: 4, North: true})
	walkFor(w, 3*time.Second)
	if w.EdgesAt(3.5*world.TileSize, 4.5*world.TileSize).NorthShut {
		t.Fatal("the door stays shut after approaching it")
	}
}

func TestWalkingElsewhereGivesUpOnTheDoor(t *testing.T) {
	w := world.NewOnMap(7, lodge(world.Wall))
	w.ApproachDoor(world.Doorway{Col: 3, Row: 4, North: true})
	w.WalkTo(7.5*world.TileSize, 5.5*world.TileSize)
	walkFor(w, 3*time.Second)
	if w.EdgesAt(3.5*world.TileSize, 4.5*world.TileSize).NorthShut {
		t.Fatal("the door closed after the hero walked elsewhere")
	}
}
