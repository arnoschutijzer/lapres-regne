package terminal

import (
	"fmt"
	"strings"

	"dither-cli/internal/world"
)

const (
	// panelWidth and panelHeight size the inventory panel in terminal cells.
	panelWidth  = 19
	panelHeight = 11
	// panelColumns lays the 28 slots out as Old School RuneScape does: 4 across, 7 down.
	panelColumns = 4
	logGlyph     = "⢴⡦"
	emptyGlyph   = "··"
)

// logTint colors each kind of log; empty slots stay faint.
var logTint = map[world.TreeKind]string{
	world.NormalTree: "\x1b[38;2;196;170;120m",
	world.Oak:        "\x1b[38;2;178;128;74m",
	world.Willow:     "\x1b[38;2;150;158;112m",
	world.Yew:        "\x1b[38;2;132;84;60m",
}

const (
	panelText = "\x1b[38;2;147;166;130m"
	emptySlot = "\x1b[38;2;58;72;62m" + emptyGlyph
)

// Slots sit four cells apart from slotIndent on: a bracket or space, the glyph, a bracket or space.
const (
	slotIndent = 2
	slotPitch  = 4
	// slotLine is the panel line holding the first row of slots.
	slotLine = 3
)

// inventoryPanel lays out the panel's lines, each padded to panelWidth visible cells.
func inventoryPanel(inventory world.Inventory, cursor slotCursor) []string {
	pad := func(text string, visible int) string {
		return text + strings.Repeat(" ", max(0, panelWidth-visible))
	}
	title := fmt.Sprintf("   inventory %d/%d", inventory.Len(), world.InventorySize)
	lines := []string{pad("", 0), pad(panelText+title, len(title)), pad("", 0)}
	slots := inventory.Slots()
	for row := range world.InventorySize / panelColumns {
		line := strings.Repeat(" ", slotIndent)
		for column := range panelColumns {
			index := row*panelColumns + column
			glyph := emptySlot
			if slots[index].Filled {
				glyph = logTint[slots[index].Kind] + logGlyph
			}
			if cursor.active && cursor.slot == index {
				line += "[" + glyph + panelText + "]"
			} else {
				line += " " + glyph + " "
			}
		}
		lines = append(lines, pad(line, slotIndent+panelColumns*slotPitch))
	}
	return append(lines, pad("", 0))
}

// slotCursor is the highlighted inventory slot, chosen with [ and ] or a click.
type slotCursor struct {
	active bool
	slot   int
}

// move steps the highlight through the slots in reading order, wrapping at the ends.
func (s *slotCursor) move(step int) {
	if !s.active {
		s.active = true
		s.slot = 0
		if step < 0 {
			s.slot = world.InventorySize - 1
		}
		return
	}
	s.slot = (s.slot + step + world.InventorySize) % world.InventorySize
}

// drop drops the highlighted log, or the first log when no log is highlighted, then
// highlights the next log so repeated presses empty the inventory in order.
func (s *slotCursor) drop(w *world.World) {
	slots := w.Hero().Inventory().Slots()
	slot := s.slot
	if !s.active || !slots[slot].Filled {
		first, filled := w.Hero().Inventory().FirstFilled()
		if !filled {
			return
		}
		slot = first
	}
	w.Drop(slot)
	s.active, s.slot = true, slot
	slots = w.Hero().Inventory().Slots()
	for step := 1; step < world.InventorySize; step++ {
		if next := (slot + step) % world.InventorySize; slots[next].Filled {
			s.slot = next
			return
		}
	}
}
