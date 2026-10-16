package world_test

import (
	"reflect"
	"testing"

	"dither-cli/internal/world"
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
	if slots := w.Hero().Inventory().Slots(); !reflect.DeepEqual(slots, []world.TreeKind{world.NormalTree}) {
		t.Fatalf("slots = %v, want one normal log", slots)
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
