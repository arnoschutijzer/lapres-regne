package terminal

import (
	"fmt"
	"strings"
	"sync"

	"lapresregne/internal/world"
)

// Slots sit slotPitch cells apart from slotIndent on: a bracket or space, the icon, a bracket or
// space.
const (
	slotIndent = 2
	slotPitch  = iconCells + 2
	// slotLine is the panel line holding the first row of slots, just below the title.
	slotLine = 1
	// panelColumns lays the 28 slots out as Old School RuneScape does: 4 across, 7 down.
	panelColumns = 4
	// panelWidth and panelHeight size the inventory panel in terminal cells.
	panelWidth  = slotIndent + panelColumns*slotPitch + 1
	panelHeight = slotLine + world.InventorySize/panelColumns*iconLines
)

// shapeIcons shades each item shape in a slot from the item's tint.
var shapeIcons = map[world.Shape]iconShader{
	world.LogShape: logShader,
}

// endGrain is the pale wood inside every log, which a cut end shows tinted by its tree.
var endGrain = [3]int{226, 204, 150}

func cutEnd(bark [3]int) [3]int { return mix(bark, endGrain, 0.6) }

// icons keeps each item's rendered icon, since an icon never changes.
var icons sync.Map

// iconCellsOf draws an item's icon line by line and cell by cell: its shape shaded in its color.
func iconCellsOf(item world.Item) [][]string {
	if drawn, ok := icons.Load(item); ok {
		return drawn.([][]string)
	}
	color := item.Color()
	drawn := renderIcon(shapeIcons[item.Shape()], [3]int{int(color.R), int(color.G), int(color.B)}, iconCells, iconLines)
	icons.Store(item, drawn)
	return drawn
}

// icon draws an item in a slot a line at a time, each line ending on the background paper.
func icon(item world.Item) []string {
	cells := iconCellsOf(item)
	lines := make([]string, len(cells))
	for line, drawn := range cells {
		lines[line] = strings.Join(drawn, "") + paperOf(background)
	}
	return lines
}

var (
	panelText = foreground(textInk)
	// emptySlot marks an empty slot faintly across the middle of where an icon would be.
	emptySlot = foreground([3]int{58, 72, 62}) + strings.Repeat(" ", iconCells/2-1) + "··" + strings.Repeat(" ", iconCells/2-1)
)

// emptyIcon draws an empty slot a line at a time.
func emptyIcon() []string {
	lines := make([]string, iconLines)
	for line := range lines {
		lines[line] = strings.Repeat(" ", iconCells)
	}
	lines[(iconLines-1)/2] = emptySlot
	return lines
}

// slotLines lays out one row of slots, a line for each line of the icons, the highlighted slot
// bracketed.
func slotLines(icons [][]string, highlighted int) []string {
	lines := make([]string, iconLines)
	for line := range lines {
		lines[line] = strings.Repeat(" ", slotIndent)
		for column, drawn := range icons {
			if column == highlighted {
				lines[line] += "[" + drawn[line] + panelText + "]"
			} else {
				lines[line] += " " + drawn[line] + panelText + " "
			}
		}
	}
	return lines
}

// inventoryPanel lays out the panel's lines, each padded to panelWidth visible cells.
func inventoryPanel(inventory world.Inventory, cursor slotCursor) []string {
	pad := func(text string, visible int) string {
		return text + strings.Repeat(" ", max(0, panelWidth-visible))
	}
	title := fmt.Sprintf("   inventory %d/%d", inventory.Len(), world.InventorySize)
	lines := []string{pad(panelText+title, len(title))}
	slots := inventory.Slots()
	for row := range world.InventorySize / panelColumns {
		drawn := make([][]string, panelColumns)
		highlighted := -1
		for column := range panelColumns {
			index := row*panelColumns + column
			drawn[column] = emptyIcon()
			if slots[index].Filled {
				drawn[column] = icon(slots[index].Item)
			}
			if cursor.active && cursor.slot == index {
				highlighted = column
			}
		}
		for _, line := range slotLines(drawn, highlighted) {
			lines = append(lines, pad(line, slotIndent+panelColumns*slotPitch))
		}
	}
	return lines
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
