package world_test

import (
	"reflect"
	"testing"

	"lapresregne/internal/world"
)

// carrying resumes a world whose hero carries items in the first slots.
func carrying(t *testing.T, carried ...world.Item) *world.World {
	t.Helper()
	slots := make([]world.Slot, len(carried))
	for index, item := range carried {
		slots[index] = world.Slot{Item: item, Filled: true}
	}
	w, err := world.Resume(7, &memoryProgress{snapshot: world.Snapshot{Seed: 7, Inventory: slots}, saved: true}, quitAt)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestTheBankStartsEmpty(t *testing.T) {
	if stacks := world.New(7).Bank().Stacks(); len(stacks) != 0 {
		t.Fatalf("stacks = %v, want an empty bank", stacks)
	}
}

func TestDepositedItemsStackByKindInTheOrderFirstDeposited(t *testing.T) {
	w := carrying(t, world.OakLogs, world.Logs, world.OakLogs)
	for slot := range 3 {
		if !w.Deposit(slot) {
			t.Fatalf("slot %d must deposit its item", slot)
		}
	}
	want := []world.Stack{{Item: world.OakLogs, Count: 2}, {Item: world.Logs, Count: 1}}
	if stacks := w.Bank().Stacks(); !reflect.DeepEqual(stacks, want) {
		t.Fatalf("stacks = %v, want %v", stacks, want)
	}
	if w.Hero().Inventory().Len() != 0 {
		t.Fatal("deposited items must leave the inventory")
	}
}

func TestAnEmptySlotDepositsNothing(t *testing.T) {
	w := carrying(t, world.Logs)
	if w.Deposit(1) || len(w.Bank().Stacks()) != 0 {
		t.Fatal("an empty slot must not deposit anything")
	}
}
