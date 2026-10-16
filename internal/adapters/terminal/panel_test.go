package terminal

import (
	"strings"
	"testing"
	"time"

	"lapresregne/internal/world"
)

// withLogs chops the normal tree beside the hero again and again until it carries a number of logs.
func withLogs(t *testing.T, logs int) *world.World {
	t.Helper()
	w := besideTree()
	for w.Hero().Inventory().Len() < logs {
		w.Chop()
		chopUntilFelled(t, w)
		// Wait for the stump to regrow.
		w.Advance(0, 0, time.Minute, false)
	}
	return w
}

func TestTheCursorWrapsAroundTheSlots(t *testing.T) {
	cursor := slotCursor{}
	cursor.move(-1)
	if !cursor.active || cursor.slot != world.InventorySize-1 {
		t.Fatalf("cursor = %+v, want [ from nothing to select the last slot", cursor)
	}
	cursor.move(1)
	if cursor.slot != 0 {
		t.Fatalf("cursor = %+v, want ] past the last slot to wrap to the first", cursor)
	}
}

func TestXDropsTheHighlightedLogThenMovesToTheNextOne(t *testing.T) {
	w := withLogs(t, 3)
	cursor := slotCursor{active: true, slot: 1}
	cursor.drop(w)
	slots := w.Hero().Inventory().Slots()
	if !slots[0].Filled || slots[1].Filled || !slots[2].Filled {
		t.Fatalf("slots = %v, want only the highlighted second slot emptied", slots[:3])
	}
	if cursor.slot != 2 {
		t.Fatalf("cursor = %+v, want it on the next filled slot", cursor)
	}
}

func TestXWithoutAHighlightedLogDropsTheFirstOne(t *testing.T) {
	w := withLogs(t, 2)
	cursor := slotCursor{active: true, slot: 9}
	cursor.drop(w)
	slots := w.Hero().Inventory().Slots()
	if slots[0].Filled || !slots[1].Filled || cursor.slot != 1 {
		t.Fatalf("slots = %v, cursor = %+v; want the first log dropped and the cursor on the next", slots[:2], cursor)
	}
	none := slotCursor{}
	none.drop(w)
	if w.Hero().Inventory().Len() != 0 {
		t.Fatal("with nothing highlighted, X must drop the first log")
	}
}

func TestAnItemIconIsItsShapeInItsColor(t *testing.T) {
	if got, want := icon(world.OakLogs), "\x1b[38;2;178;128;74m"+logGlyph; got != want {
		t.Fatalf("oak logs icon = %q, want %q", got, want)
	}
}

func TestTheHighlightedSlotIsBracketed(t *testing.T) {
	w := withLogs(t, 1)
	lines := inventoryPanel(w.Hero().Inventory(), slotCursor{active: true, slot: 0})
	if !strings.Contains(lines[3], "["+icon(world.Logs)+panelText+"]") {
		t.Fatalf("first slot row = %q, want the first slot bracketed", lines[3])
	}
	// Color escapes contain "[", so count the closing brackets.
	if strings.Count(strings.Join(lines, ""), "]") != 1 {
		t.Fatal("only the highlighted slot must be bracketed")
	}
}

func TestSlotAtFindsTheClickedSlotInTheOpenPanel(t *testing.T) {
	r := newRenderer(besideTree(), "L’Après-Règne")
	r.inventoryOpen = true
	r.frame(100, 30, 0)
	// The panel's slots start four columns into the panel, on its fourth line.
	left := 100 - panelWidth + 1
	cases := []struct {
		column, row, slot int
	}{
		{left + 3, 5, 0}, {left + 4, 5, 0}, {left + 7, 5, 1}, {left + 3, 6, 4}, {left + 15, 11, 27},
	}
	for _, c := range cases {
		if slot, ok := r.slotAt(c.column, c.row); !ok || slot != c.slot {
			t.Errorf("slotAt(%d, %d) = %d, %v; want slot %d", c.column, c.row, slot, ok, c.slot)
		}
	}
	if _, ok := r.slotAt(left+3, 3); ok {
		t.Error("the panel title must not be a slot")
	}
	r.inventoryOpen = false
	if _, ok := r.slotAt(left+3, 5); ok {
		t.Error("a closed panel has no slots to click")
	}
}
