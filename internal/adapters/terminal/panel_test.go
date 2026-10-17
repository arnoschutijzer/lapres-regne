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
	cursor.remove(w, w.Drop)
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
	cursor.remove(w, w.Drop)
	slots := w.Hero().Inventory().Slots()
	if slots[0].Filled || !slots[1].Filled || cursor.slot != 1 {
		t.Fatalf("slots = %v, cursor = %+v; want the first log dropped and the cursor on the next", slots[:2], cursor)
	}
	none := slotCursor{}
	none.remove(w, w.Drop)
	if w.Hero().Inventory().Len() != 0 {
		t.Fatal("with nothing highlighted, X must drop the first log")
	}
}

func TestAnIconIsLinesOfHalfBlockPixels(t *testing.T) {
	drawn := icon(world.OakLogs)
	if len(drawn) != iconLines {
		t.Fatalf("lines = %d, want %d", len(drawn), iconLines)
	}
	for _, line := range drawn {
		if strings.Count(line, "▀") != iconCells || !strings.HasSuffix(line, paperOf(background)) {
			t.Fatalf("line = %q, want %d half blocks ending on the background paper", line, iconCells)
		}
	}
}

func TestALogIconShowsAPaleCutEndAtItsUpperRightAndBarkBelowIt(t *testing.T) {
	oak := [3]int{178, 128, 74}
	pixels := logPixels(oak)
	if len(pixels) != iconLines*2 || len(pixels[0]) != iconCells {
		t.Fatalf("pixels = %d by %d, want %d by %d", len(pixels[0]), len(pixels), iconCells, iconLines*2)
	}
	end, bark := pixels[0][3], pixels[2][1]
	if end != cutEnd(oak) || bark != oak {
		t.Fatalf("end %v, bark %v; want the cut end's pale wood at the upper right and the oak's bark below", end, bark)
	}
	if pixels[0][0] != background || pixels[3][3] != background {
		t.Fatal("the corners away from the log must show the background")
	}
}

func TestTheInventoryIsCompact(t *testing.T) {
	if panelWidth != 23 || panelHeight != 15 {
		t.Fatalf("inventory = %d by %d, want 23 by 15", panelWidth, panelHeight)
	}
}

func TestEachLogsIconKeepsItsTreesTint(t *testing.T) {
	seen := map[string]world.Item{}
	for _, item := range []world.Item{world.Logs, world.OakLogs, world.WillowLogs, world.YewLogs} {
		drawn := strings.Join(icon(item), "")
		if other, taken := seen[drawn]; taken {
			t.Errorf("%v draw the same icon as %v", item, other)
		}
		seen[drawn] = item
	}
}

func TestTheHighlightedSlotIsBracketed(t *testing.T) {
	w := withLogs(t, 1)
	lines := inventoryPanel(w.Hero().Inventory(), slotCursor{active: true, slot: 0})
	for line, drawn := range icon(world.Logs) {
		if !strings.Contains(lines[slotLine+line], "["+drawn+panelText+"]") {
			t.Fatalf("first slot line %d = %q, want the first slot bracketed", line, lines[slotLine+line])
		}
	}
	// Color escapes contain "[", so count the closing brackets: one a line of the icon.
	if strings.Count(strings.Join(lines, ""), "]") != iconLines {
		t.Fatal("only the highlighted slot must be bracketed")
	}
}

func TestSlotAtFindsTheClickedSlotInTheOpenPanel(t *testing.T) {
	r := newRenderer(besideTree(), NewCamera(0, 0), "L’Après-Règne")
	r.inventoryOpen = true
	r.frame(100, 40, 0)
	// The panel's slots start past its indent, from its second line, the screen's third row; each
	// is slotPitch cells wide and iconLines rows tall.
	left := 100 - panelWidth + 1
	first := left + slotIndent
	cases := []struct {
		column, row, slot int
	}{
		{first, 3, 0}, {first + slotPitch - 1, 3 + iconLines - 1, 0}, {first + slotPitch, 3, 1},
		{first, 3 + iconLines, 4}, {first + 4*slotPitch - 1, 3 + 7*iconLines - 1, 27},
	}
	for _, c := range cases {
		if slot, ok := r.slotAt(c.column, c.row); !ok || slot != c.slot {
			t.Errorf("slotAt(%d, %d) = %d, %v; want slot %d", c.column, c.row, slot, ok, c.slot)
		}
	}
	if _, ok := r.slotAt(first, 2); ok {
		t.Error("the panel title must not be a slot")
	}
	r.inventoryOpen = false
	if _, ok := r.slotAt(first, 3); ok {
		t.Error("a closed panel has no slots to click")
	}
}

func TestALogsCutEndIsPalerThanItsBarkAndKeepsItsTreesTint(t *testing.T) {
	ends := map[[3]int]bool{}
	for _, item := range []world.Item{world.Logs, world.OakLogs, world.WillowLogs, world.YewLogs} {
		color := item.Color()
		bark := [3]int{int(color.R), int(color.G), int(color.B)}
		end := cutEnd(bark)
		if end[0]+end[1]+end[2] <= bark[0]+bark[1]+bark[2] {
			t.Errorf("%v cut end %v must be paler than its bark %v", item, end, bark)
		}
		ends[end] = true
	}
	if len(ends) != 4 {
		t.Fatal("each tree's logs must keep a cut end of their own")
	}
}

func TestSlotsCutOffByAShortTerminalCannotBeClicked(t *testing.T) {
	r := newRenderer(besideTree(), NewCamera(0, 0), "L’Après-Règne")
	r.inventoryOpen = true
	// A terminal 14 rows tall shows the forest on rows 2 to 12, below the title row.
	r.frame(100, 14, 0)
	first := 100 - panelWidth + 1 + slotIndent
	if _, ok := r.slotAt(first, 12); !ok {
		t.Fatal("a slot on the forest's last row must still be clickable")
	}
	if _, ok := r.slotAt(first, 13); ok {
		t.Fatal("a click on the hint line must not pick a slot hidden below it")
	}
}
