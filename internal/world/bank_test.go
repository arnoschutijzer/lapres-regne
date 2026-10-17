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

func TestWithdrawingTakesOneItemBackIntoTheFirstEmptySlot(t *testing.T) {
	w := carrying(t, world.OakLogs, world.OakLogs, world.YewLogs)
	for slot := range 3 {
		w.Deposit(slot)
	}
	if !w.Withdraw(0) {
		t.Fatal("a stack must withdraw an item")
	}
	if slots := w.Hero().Inventory().Slots(); slots[0] != (world.Slot{Item: world.OakLogs, Filled: true}) || w.Hero().Inventory().Len() != 1 {
		t.Fatalf("slots = %v, want one oak log back in the first slot", slots[:3])
	}
	want := []world.Stack{{Item: world.OakLogs, Count: 1}, {Item: world.YewLogs, Count: 1}}
	if stacks := w.Bank().Stacks(); !reflect.DeepEqual(stacks, want) {
		t.Fatalf("stacks = %v, want %v", stacks, want)
	}
}

func TestWithdrawingTheLastItemOfAStackClearsItsPlace(t *testing.T) {
	w := carrying(t, world.OakLogs, world.YewLogs)
	w.Deposit(0)
	w.Deposit(1)
	w.Withdraw(0)
	want := []world.Stack{{Item: world.YewLogs, Count: 1}}
	if stacks := w.Bank().Stacks(); !reflect.DeepEqual(stacks, want) {
		t.Fatalf("stacks = %v, want %v", stacks, want)
	}
	if w.Withdraw(1) {
		t.Fatal("a place with no stack must withdraw nothing")
	}
}

func TestAFullInventoryWithdrawsNothing(t *testing.T) {
	w := carrying(t, make([]world.Item, world.InventorySize)...)
	w.Deposit(0)
	// Walk to the tree beside the spawn and chop a log to fill the gap again.
	for range 3 {
		w.Walk(0, 1, false)
	}
	for range 9 {
		w.Walk(1, 0, false)
	}
	w.Chop()
	chopUntilFelled(t, w)
	if !w.Hero().Inventory().Full() {
		t.Fatal("the chopped log must fill the inventory")
	}
	if w.Withdraw(0) || w.Bank().Stacks()[0].Count != 1 {
		t.Fatal("a full inventory must leave the item in the bank")
	}
	if w.Message() != "You don't have enough inventory space." {
		t.Fatalf("message = %q, want the hero told there is no room", w.Message())
	}
}
