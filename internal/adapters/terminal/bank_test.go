package terminal

import (
	"fmt"
	"strings"
	"testing"

	"lapresregne/internal/world"
)

func TestBTogglesTheBank(t *testing.T) {
	for _, key := range []int{'b', 'B'} {
		if got := keyAction(key); got != toggleBank {
			t.Errorf("keyAction(%q) = %v, want toggleBank", key, got)
		}
	}
}

func TestOpeningTheBankShowsTheInventoryToDepositFrom(t *testing.T) {
	r := newRenderer(besideTree(), NewCamera(0, 0), "L’Après-Règne")
	r.toggleBank()
	if !r.bankOpen || !r.inventoryOpen {
		t.Fatalf("bank open = %v, inventory open = %v; want both open", r.bankOpen, r.inventoryOpen)
	}
	r.toggleBank()
	if r.bankOpen || !r.inventoryOpen {
		t.Fatalf("bank open = %v, inventory open = %v; want the bank closed and the inventory left open", r.bankOpen, r.inventoryOpen)
	}
}

func TestAnEmptyBankSaysSo(t *testing.T) {
	if lines := bankPanel(world.Bank{}, bankMaxColumns); !strings.Contains(strings.Join(lines, ""), "empty") || len(lines) != panelHeight {
		t.Fatalf("lines = %q, want %d lines saying the bank is empty", lines, panelHeight)
	}
}

// banking deposits every log the hero carries.
func banking(t *testing.T, logs int) *world.World {
	t.Helper()
	w := withLogs(t, logs)
	for slot := range logs {
		w.Deposit(slot)
	}
	return w
}

func TestTheBankShowsEachStackAsAnIconUnderItsCount(t *testing.T) {
	w := banking(t, 3)
	lines := bankPanel(w.Bank(), bankMaxColumns)
	// The first slot's count sits above its icon, both a cell into the slot.
	if want := strings.Repeat(" ", slotIndent+1) + countTint(3) + "3   "; !strings.HasPrefix(lines[slotLine], want) {
		t.Fatalf("count line = %q, want it to start %q", lines[slotLine], want)
	}
	if want := strings.Repeat(" ", slotIndent+1) + icon(world.Logs); !strings.HasPrefix(lines[slotLine+1], want) {
		t.Fatalf("icon line = %q, want it to start %q", lines[slotLine+1], want)
	}
}

func TestTheBankGridIsEightAcrossWhenThereIsRoom(t *testing.T) {
	cases := map[int]int{100: 8, 62: 8, 61: 7, 40: 3}
	for cols, want := range cases {
		if got := bankColumns(cols); got != want {
			t.Errorf("bankColumns(%d) = %d, want %d", cols, got, want)
		}
		if bankWidth(bankColumns(cols))+panelWidth >= cols {
			t.Errorf("at %d columns the bank must leave a gap before the inventory", cols)
		}
	}
}

func TestCountsAreTintedByTheirSize(t *testing.T) {
	if countTint(99_999) != countYellow || countTint(100_000) != countWhite || countTint(10_000_000) != countGreen {
		t.Fatal("counts must be yellow, white from 100K and green from 10M")
	}
}

func TestBankCountsShortenToFitThePanel(t *testing.T) {
	cases := map[int]string{1: "1", 9999: "9999", 10_000: "10K", 999_999: "999K", 1_000_000: "1M", 2_147_000_000: "2147M"}
	for count, want := range cases {
		if got := bankCount(count); got != want {
			t.Errorf("bankCount(%d) = %q, want %q", count, got, want)
		}
	}
}

func TestTheBankPanelOpensAtTheTopLeftAndClosesCleanly(t *testing.T) {
	r := newRenderer(besideTree(), NewCamera(0, 0), "L’Après-Règne")
	r.frame(100, 30, 0)
	r.toggleBank()
	open := r.frame(100, 30, 0)
	if !strings.Contains(open, "bank") || !strings.Contains(open, "inventory") {
		t.Fatal("opening the bank must show the bank beside the inventory")
	}
	r.toggleBank()
	closed := r.frame(100, 30, 0)
	// The forest must be repainted in the panel's first row, at the forest's left edge.
	if strings.Contains(closed, "bank") || !strings.Contains(closed, fmt.Sprintf("\x1b[2;%dH", 1)) {
		t.Fatal("closing the bank must repaint the forest beneath it")
	}
}

func TestWhileTheBankIsOpenClickingASlotDepositsItsItem(t *testing.T) {
	w := withLogs(t, 2)
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	r.clickSlot(1)
	if w.Hero().Inventory().Len() != 2 || r.cursor != (slotCursor{active: true, slot: 1}) {
		t.Fatal("with the bank closed, a click must only highlight the slot")
	}
	r.toggleBank()
	r.clickSlot(1)
	if slots := w.Hero().Inventory().Slots(); !slots[0].Filled || slots[1].Filled || len(w.Bank().Stacks()) != 1 {
		t.Fatalf("slots = %v, bank = %v; want the clicked log deposited", slots[:2], w.Bank().Stacks())
	}
}

func TestWhileTheBankIsOpenXDepositsInsteadOfDropping(t *testing.T) {
	w := withLogs(t, 2)
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	r.toggleBank()
	r.removeFromSlot()
	if len(w.GroundItems()) != 0 || w.Bank().Stacks()[0] != (world.Stack{Item: world.Logs, Count: 1}) {
		t.Fatalf("ground = %v, bank = %v; want the first log deposited, not dropped", w.GroundItems(), w.Bank().Stacks())
	}
	r.toggleBank()
	r.removeFromSlot()
	if len(w.GroundItems()) != 1 {
		t.Fatal("with the bank closed, X must drop the log")
	}
}

func TestStackAtFindsTheClickedStackInTheOpenBank(t *testing.T) {
	w := banking(t, 1)
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	r.toggleBank()
	r.frame(100, 30, 0)
	// The first stack's count sits on the panel's fourth line, the fifth row of the screen, and its
	// icon below; its slot spans five cells from the panel's indent.
	for _, cell := range [][2]int{{slotIndent + 1, 5}, {slotIndent + bankPitch, 6}} {
		if stack, ok := r.stackAt(cell[0], cell[1]); !ok || stack != 0 {
			t.Errorf("stackAt(%d, %d) = %d, %v; want the first stack", cell[0], cell[1], stack, ok)
		}
	}
	for _, cell := range [][2]int{{slotIndent, 5}, {slotIndent + bankPitch + 1, 5}, {slotIndent + 1, 7}, {slotIndent + 1, 4}} {
		if _, ok := r.stackAt(cell[0], cell[1]); ok {
			t.Errorf("stackAt(%d, %d) must not pick a stack", cell[0], cell[1])
		}
	}
	r.toggleBank()
	if _, ok := r.stackAt(slotIndent+1, 5); ok {
		t.Error("a closed bank has no stacks to click")
	}
}
