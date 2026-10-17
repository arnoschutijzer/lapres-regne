package terminal

import (
	"slices"
	"testing"
	"time"

	"lapresregne/internal/world"
)

// atTheLodgeDoor is the edged lodge with the hero walked up to just outside its doorway, which
// spans x 40 to 42 along z 28.
func atTheLodgeDoor(barrier world.Ground) *world.World {
	w := edgedLodge(barrier)
	w.WalkTo(41, 29.4)
	for range 60 {
		w.Advance(0, 0, 50*time.Millisecond, false)
	}
	return w
}

func TestAShutDoorFillsItsDoorwayWithPlanks(t *testing.T) {
	w := atTheLodgeDoor(world.Wall)
	w.ToggleDoor()
	_, c := closeView(w)
	if found := inkNear(c, 41.3, doorHeight/2, 28.1, plankInk); found < 10 {
		t.Fatalf("%d of 25 pixels of planks in the doorway, want the door shut across it", found)
	}
}

func TestAShutGateClosesTheGapInItsFence(t *testing.T) {
	w := atTheLodgeDoor(world.Fence)
	_, open := closeView(w)
	w.ToggleDoor()
	_, shut := closeView(w)
	if before, after := inkNear(open, 41.3, 0.7, 28, plankInk), inkNear(shut, 41.3, 0.7, 28, plankInk); after <= before {
		t.Fatalf("%d pixels of planks across the gateway shut, %d open; want the gate swung across it", after, before)
	}
}

func TestEOffersToCloseTheOpenDoorInReach(t *testing.T) {
	r, _ := closeView(atTheLodgeDoor(world.Wall))
	r.toggleMenu()
	if r.menuTitle != "Door" || !slices.Equal(r.menu, []string{"Close"}) {
		t.Fatalf("menu = %q %q, want the door offered to close", r.menuTitle, r.menu)
	}
}

func TestChoosingCloseShutsTheDoorAndOffersToOpenIt(t *testing.T) {
	r, _ := closeView(atTheLodgeDoor(world.Wall))
	r.toggleMenu()
	r.chooseFirst()
	if r.menu != nil {
		t.Fatal("choosing an action must close the menu")
	}
	if !r.world.EdgesAt(41, 29).NorthShut {
		t.Fatal("choosing Close must shut the door")
	}
	r.toggleMenu()
	if !slices.Equal(r.menu, []string{"Open"}) {
		t.Fatalf("menu = %q, want the shut door offered to open", r.menu)
	}
}

func TestAGateInAFenceIsOfferedAsAGate(t *testing.T) {
	r, _ := closeView(atTheLodgeDoor(world.Fence))
	r.toggleMenu()
	if r.menuTitle != "Gate" {
		t.Fatalf("menu title = %q, want Gate", r.menuTitle)
	}
}

func TestEClosesAnOpenMenu(t *testing.T) {
	r, _ := closeView(atTheLodgeDoor(world.Wall))
	r.toggleMenu()
	r.toggleMenu()
	if r.menu != nil {
		t.Fatalf("menu = %q, want it closed", r.menu)
	}
}
