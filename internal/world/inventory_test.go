package world_test

import (
	"testing"
	"time"

	"lapresregne/internal/world"
)

func TestAnEmptyInventoryHas28Slots(t *testing.T) {
	inventory := world.New(7).Hero().Inventory()
	if inventory.Len() != 0 || inventory.Full() || world.InventorySize != 28 {
		t.Fatalf("len = %d, full = %v, size = %d; want an empty 28-slot inventory", inventory.Len(), inventory.Full(), world.InventorySize)
	}
}

func TestACutLogTakesASlotOfItsOwnKind(t *testing.T) {
	w := besideTree()
	w.Chop()
	chopUntilFelled(t, w)
	slots := w.Hero().Inventory().Slots()
	if len(slots) != world.InventorySize || slots[0] != (world.Slot{Kind: world.NormalTree, Filled: true}) || slots[1].Filled {
		t.Fatalf("slots = %v, want one normal log in the first of 28 slots", slots)
	}
}

func TestLogsAreNamedAfterTheirTree(t *testing.T) {
	names := map[world.TreeKind]string{world.NormalTree: "logs", world.Oak: "oak logs", world.Willow: "willow logs", world.Yew: "yew logs"}
	for kind, name := range names {
		if got := kind.Logs(); got != name {
			t.Errorf("%v logs = %q, want %q", kind, got, name)
		}
	}
}

func TestADroppedLogLiesUnderTheHeroForThreeMinutes(t *testing.T) {
	w := besideTree()
	w.Chop()
	chopUntilFelled(t, w)
	if w.Drop(1) {
		t.Fatal("an empty slot must not drop anything")
	}
	if !w.Drop(0) || w.Hero().Inventory().Len() != 0 {
		t.Fatal("dropping the first slot must empty it")
	}
	x, z := w.Hero().Position()
	ground := w.GroundLogs()
	if len(ground) != 1 || ground[0].Kind != world.NormalTree || ground[0].X != x || ground[0].Z != z {
		t.Fatalf("ground = %+v, want one normal log under the hero at %v, %v", ground, x, z)
	}
	w.Advance(0, 0, 3*time.Minute-time.Millisecond, false)
	if len(w.GroundLogs()) != 1 {
		t.Fatal("a dropped log must lie on the ground for three minutes")
	}
	w.Advance(0, 0, time.Millisecond, false)
	if len(w.GroundLogs()) != 0 {
		t.Fatal("a dropped log must disappear after three minutes")
	}
}

func TestADroppedLogSoundsAsItLands(t *testing.T) {
	w := besideTree()
	sounds := &recordedSounds{}
	w.SetSounds(sounds)
	w.Chop()
	chopUntilFelled(t, w)
	w.Drop(1)
	if sounds.drops != 0 {
		t.Fatalf("drops = %d, want silence when an empty slot drops nothing", sounds.drops)
	}
	w.Drop(0)
	if sounds.drops != 1 {
		t.Fatalf("drops = %d, want one log heard landing", sounds.drops)
	}
}
