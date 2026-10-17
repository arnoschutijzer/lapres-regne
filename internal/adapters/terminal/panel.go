package terminal

import (
	"fmt"
	"strings"

	"lapresregne/internal/world"
)

const (
	// panelWidth and panelHeight size the inventory panel in terminal cells.
	panelWidth  = 19
	panelHeight = 11
	// panelColumns lays the 28 slots out as Old School RuneScape does: 4 across, 7 down.
	panelColumns = 4
	emptyGlyph   = "··"
)

// shapeIcons draws each item shape in a slot, two cells wide, from the item's tint; empty slots
// stay faint.
var shapeIcons = map[world.Shape]func(tint [3]int) string{
	world.LogShape: logIcon,
}

// A log lies on its side: its bark runs along the first cell, and its cut end, a dot taller as
// seen at an angle, fills the second.
const (
	logBark = "⣶"
	logEnd  = "⣿"
)

// logIcon draws the bark in the log's tint and the cut end in the paler wood inside.
func logIcon(tint [3]int) string {
	return foreground(tint) + logBark + foreground(cutEnd(tint)) + logEnd
}

// endGrain is the pale wood inside every log, which a cut end shows tinted by its tree.
var endGrain = [3]int{226, 204, 150}

func cutEnd(bark [3]int) [3]int { return mix(bark, endGrain, 0.6) }

// icon draws an item in a slot: its shape in its color.
func icon(item world.Item) string {
	color := item.Color()
	return shapeIcons[item.Shape()]([3]int{int(color.R), int(color.G), int(color.B)})
}

var (
	panelText = foreground(textInk)
	emptySlot = foreground([3]int{58, 72, 62}) + emptyGlyph
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
				glyph = icon(slots[index].Item)
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

// remove drops or deposits the highlighted item, or the first item when no item is highlighted,
// then highlights the next item so repeated presses empty the inventory in order.
func (s *slotCursor) remove(w *world.World, remove func(slot int) bool) {
	slots := w.Hero().Inventory().Slots()
	slot := s.slot
	if !s.active || !slots[slot].Filled {
		first, filled := w.Hero().Inventory().FirstFilled()
		if !filled {
			return
		}
		slot = first
	}
	remove(slot)
	s.active, s.slot = true, slot
	slots = w.Hero().Inventory().Slots()
	for step := 1; step < world.InventorySize; step++ {
		if next := (slot + step) % world.InventorySize; slots[next].Filled {
			s.slot = next
			return
		}
	}
}
