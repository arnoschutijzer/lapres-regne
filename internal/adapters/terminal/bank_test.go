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
	if lines := bankPanel(world.Bank{}); !strings.Contains(strings.Join(lines, ""), "empty") || len(lines) != panelHeight {
		t.Fatalf("lines = %q, want %d lines saying the bank is empty", lines, panelHeight)
	}
}

func TestTheBankPanelListsEachStackWithItsCount(t *testing.T) {
	w := withLogs(t, 3)
	for slot := range 3 {
		w.Deposit(slot)
	}
	lines := bankPanel(w.Bank())
	if !strings.Contains(lines[slotLine], icon(world.Logs)) || !strings.Contains(lines[slotLine], "logs") || !strings.HasSuffix(lines[slotLine], "   3") {
		t.Fatalf("first stack line = %q, want the logs icon, name and a count of 3", lines[slotLine])
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
	w := withLogs(t, 1)
	w.Deposit(0)
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	r.toggleBank()
	r.frame(100, 30, 0)
	// The first stack sits on the panel's fourth line, the fifth row of the screen.
	if stack, ok := r.stackAt(1, 5); !ok || stack != 0 {
		t.Fatalf("stackAt(1, 5) = %d, %v; want the first stack", stack, ok)
	}
	if _, ok := r.stackAt(bankWidth, 5); !ok {
		t.Error("the stack's line must be clickable to the panel's right edge")
	}
	if _, ok := r.stackAt(bankWidth+1, 5); ok {
		t.Error("a click beside the panel must not pick a stack")
	}
	if _, ok := r.stackAt(1, 6); ok {
		t.Error("a line with no stack must not pick one")
	}
	r.toggleBank()
	if _, ok := r.stackAt(1, 5); ok {
		t.Error("a closed bank has no stacks to click")
	}
}
