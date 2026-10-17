package terminal

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

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
	if lines := bankPanel(world.Bank{}, bankMaxColumns, bankCursor{}, 0); !strings.Contains(strings.Join(lines, ""), "empty") || len(lines) != panelHeight {
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
	lines := bankPanel(w.Bank(), bankMaxColumns, bankCursor{}, 0)
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

func TestShiftTurnsTheInventoryKeysIntoTheBankKeys(t *testing.T) {
	cases := map[string]action{
		"{": previousStack, "}": nextStack, "X": withdrawStack, "x": dropLog, "[": previousSlot,
		// Kitty's protocol reports the unshifted key with Shift among the modifiers.
		"\x1b[91;2u": previousStack, "\x1b[93;2u": nextStack, "\x1b[120;2u": withdrawStack, "\x1b[120u": dropLog,
	}
	for input, want := range cases {
		var d keyDecoder
		events := feed(&d, input)
		if got := events[len(events)-1].action; got != want {
			t.Errorf("%q gives %v, want %v", input, got, want)
		}
	}
}

func TestTheBankCursorWrapsAroundTheStacks(t *testing.T) {
	cursor := bankCursor{}
	cursor.move(1, 0)
	if cursor.active {
		t.Fatal("an empty bank has nothing to highlight")
	}
	cursor.move(-1, 3)
	if !cursor.active || cursor.stack != 2 {
		t.Fatalf("cursor = %+v, want Shift-[ from nothing to highlight the last stack", cursor)
	}
	cursor.move(1, 3)
	if cursor.stack != 0 {
		t.Fatalf("cursor = %+v, want Shift-] past the last stack to wrap to the first", cursor)
	}
}

func TestMovingTheBankHighlightOpensTheBank(t *testing.T) {
	r := newRenderer(banking(t, 1), NewCamera(0, 0), "L’Après-Règne")
	r.moveBankCursor(1)
	if !r.bankOpen || r.bankCursor != (bankCursor{active: true, stack: 0}) {
		t.Fatalf("bank open = %v, cursor = %+v; want the bank open on the first stack", r.bankOpen, r.bankCursor)
	}
}

func TestShiftXWithdrawsFromTheHighlightedStackAndDropsWhileTheBankIsClosed(t *testing.T) {
	w := banking(t, 2)
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	r.toggleBank()
	r.withdraw()
	if w.Hero().Inventory().Len() != 1 || w.Bank().Stacks()[0].Count != 1 {
		t.Fatal("with nothing highlighted, Shift-X must withdraw from the first stack")
	}
	r.withdraw()
	if w.Hero().Inventory().Len() != 2 || len(w.Bank().Stacks()) != 0 || r.bankCursor.active {
		t.Fatalf("bank = %v, cursor = %+v; want the last log withdrawn and nothing left to highlight", w.Bank().Stacks(), r.bankCursor)
	}
	r.toggleBank()
	r.withdraw()
	if len(w.GroundItems()) != 1 {
		t.Fatal("with the bank closed, Shift-X must drop like X")
	}
}

func TestClickingAStackHighlightsItAndWithdrawsOne(t *testing.T) {
	w := banking(t, 2)
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	r.toggleBank()
	r.clickStack(0)
	if w.Hero().Inventory().Len() != 1 || r.bankCursor != (bankCursor{active: true, stack: 0}) {
		t.Fatalf("logs = %d, cursor = %+v; want one log withdrawn from the highlighted stack", w.Hero().Inventory().Len(), r.bankCursor)
	}
}

func TestTheHighlightedStackIsBracketed(t *testing.T) {
	lines := bankPanel(banking(t, 1).Bank(), bankMaxColumns, bankCursor{active: true, stack: 0}, 0)
	if want := strings.Repeat(" ", slotIndent) + "[" + icon(world.Logs) + panelText + "]"; !strings.HasPrefix(lines[slotLine+1], want) {
		t.Fatalf("icon line = %q, want it to start %q", lines[slotLine+1], want)
	}
}

func TestTheBankScrollsToKeepTheHighlightInView(t *testing.T) {
	// 50 stacks fill 7 rows of 8; 4 rows show at once.
	cases := []struct{ top, stack, want int }{
		{0, 31, 0}, {0, 32, 1}, {0, 49, 3}, {3, 8, 1}, {2, 20, 2},
	}
	for _, c := range cases {
		if got := scrolled(c.top, bankCursor{active: true, stack: c.stack}, 8, 50); got != c.want {
			t.Errorf("scrolled(top %d, stack %d) = %d, want %d", c.top, c.stack, got, c.want)
		}
	}
	if got := scrolled(5, bankCursor{}, 8, 9); got != 0 {
		t.Errorf("a bank that fits must show from the top, got row %d", got)
	}
}

func TestRightClicksReportTheCell(t *testing.T) {
	var d keyDecoder
	events := feed(&d, "\x1b[<2;12;7M")
	last := events[len(events)-1]
	if !last.rightClick || last.click || last.column != 12 || last.row != 7 {
		t.Fatalf("event = %+v, want a right click at column 12, row 7", last)
	}
}

func TestRightClickingAnInventorySlotOffersToDepositOneOrAll(t *testing.T) {
	w := withLogs(t, 3)
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	r.frame(100, 30, 0)
	// The inventory's first slot sits four columns into the panel, on the screen's fifth row.
	inventoryColumn := 100 - panelWidth + 1 + slotIndent + 1
	if r.rightClick(inventoryColumn, 5) {
		t.Fatal("with the bank closed, a right-click on the inventory offers nothing")
	}
	r.toggleBank()
	r.frame(100, 30, 0)
	if !r.rightClick(inventoryColumn, 5) || r.menuTitle != "Logs" || !reflect.DeepEqual(r.menu, []string{"Deposit-1", "Deposit-All"}) {
		t.Fatalf("menu = %q %v, want Deposit-1 and Deposit-All for the logs", r.menuTitle, r.menu)
	}
	r.choose(1)
	if w.Hero().Inventory().Len() != 0 || w.Bank().Stacks()[0].Count != 3 || r.menu != nil {
		t.Fatalf("carried = %d, bank = %v; want every log deposited and the menu closed", w.Hero().Inventory().Len(), w.Bank().Stacks())
	}
}

func TestRightClickingABankStackOffersToWithdrawOneOrAll(t *testing.T) {
	w := banking(t, 3)
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	r.toggleBank()
	r.frame(100, 30, 0)
	if !r.rightClick(slotIndent+2, 6) || !reflect.DeepEqual(r.menu, []string{"Withdraw-1", "Withdraw-All"}) {
		t.Fatalf("menu = %q %v, want Withdraw-1 and Withdraw-All", r.menuTitle, r.menu)
	}
	r.choose(0)
	if w.Hero().Inventory().Len() != 1 {
		t.Fatal("Withdraw-1 must withdraw one log")
	}
	r.rightClick(slotIndent+2, 6)
	r.choose(1)
	if w.Hero().Inventory().Len() != 3 || len(w.Bank().Stacks()) != 0 || r.bankCursor.active {
		t.Fatalf("carried = %d, bank = %v, cursor = %+v; want the rest withdrawn", w.Hero().Inventory().Len(), w.Bank().Stacks(), r.bankCursor)
	}
}

func TestRightClickingElsewhereOffersNothing(t *testing.T) {
	r := newRenderer(banking(t, 1), NewCamera(0, 0), "L’Après-Règne")
	r.toggleBank()
	r.frame(100, 30, 0)
	if r.rightClick(50, 20) || r.menu != nil {
		t.Fatal("a right-click on the forest must not open a menu")
	}
}

func TestTheWheelReportsTheCellItScrollsOver(t *testing.T) {
	var d keyDecoder
	events := feed(&d, "\x1b[<65;12;7M")
	if last := events[len(events)-1]; last.zoom != -1 || last.column != 12 || last.row != 7 {
		t.Fatalf("event = %+v, want the wheel scrolling down over column 12, row 7", last)
	}
}

// bankOfFour opens a renderer on a bank of every kind of log in a forest 40 columns wide: 3 stacks
// a row, so 2 rows, which fit.
func bankOfFour(t *testing.T) *renderer {
	t.Helper()
	carried := []world.Item{world.Logs, world.OakLogs, world.WillowLogs, world.YewLogs}
	slots := make([]world.Slot, len(carried))
	for index, item := range carried {
		slots[index] = world.Slot{Item: item, Filled: true}
	}
	w, err := world.Resume(7, &fixedProgress{world.Snapshot{Seed: 1, Inventory: slots}}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for slot := range carried {
		w.Deposit(slot)
	}
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	r.toggleBank()
	r.frame(40, 30, 0)
	return r
}

type fixedProgress struct{ snapshot world.Snapshot }

func (p *fixedProgress) Load() (world.Snapshot, bool, error) { return p.snapshot, true, nil }
func (p *fixedProgress) Save(world.Snapshot) error           { return nil }

func TestTheWheelScrollsTheBankUnderThePointer(t *testing.T) {
	r := bankOfFour(t)
	// Two rows fit in four, so nothing scrolls.
	if !r.scrollBank(slotIndent+1, 5, 1) || r.bankTop != 0 {
		t.Fatalf("top = %d, want a bank that fits to stay at its first row", r.bankTop)
	}
	r.bankTop = 0
	if r.scrollBank(30, 20, 1) {
		t.Fatal("the wheel away from the bank must not scroll it")
	}
	r.toggleBank()
	if r.scrollBank(slotIndent+1, 5, 1) {
		t.Fatal("a closed bank must not scroll")
	}
}

func TestTheBankScrollsByRowsWithinItsStacks(t *testing.T) {
	// 50 stacks fill 7 rows of 8; 4 rows show at once.
	cases := []struct{ top, by, want int }{{0, 1, 1}, {0, -1, 0}, {2, 5, 3}, {3, -2, 1}}
	for _, c := range cases {
		if got := scrolled(c.top+c.by, bankCursor{}, 8, 50); got != c.want {
			t.Errorf("scrolling %d rows from row %d = %d, want %d", c.by, c.top, got, c.want)
		}
	}
}

func TestTheBankTitleShowsWhichWaysItScrolls(t *testing.T) {
	bank := banking(t, 1).Bank()
	cases := []struct {
		top, rows int
		want      string
	}{{0, 4, "bank"}, {0, 7, "bank ↓"}, {2, 7, "bank ↑↓"}, {3, 7, "bank ↑"}}
	for _, c := range cases {
		if got := strings.TrimSpace(bankTitle(c.top, c.rows)); got != c.want {
			t.Errorf("bankTitle(%d, %d) = %q, want %q", c.top, c.rows, got, c.want)
		}
	}
	if !strings.Contains(bankPanel(bank, bankMaxColumns, bankCursor{}, 0)[1], "bank") {
		t.Fatal("the panel must carry the title")
	}
}

func TestMovingTheHighlightScrollsItBackIntoView(t *testing.T) {
	r := bankOfFour(t)
	r.bankTop = 5
	r.moveBankCursor(1)
	if r.bankTop != 0 {
		t.Fatalf("top = %d, want the highlighted first stack's row shown", r.bankTop)
	}
}
