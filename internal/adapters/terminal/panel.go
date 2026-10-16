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

// inventoryPanel lays out the panel's lines, each padded to panelWidth visible cells.
func inventoryPanel(inventory world.Inventory) []string {
	pad := func(text string, visible int) string {
		return text + strings.Repeat(" ", max(0, panelWidth-visible))
	}
	title := fmt.Sprintf("   inventory %d/%d", inventory.Len(), world.InventorySize)
	lines := []string{pad("", 0), pad(panelText+title, len(title)), pad("", 0)}
	slots := inventory.Slots()
	for row := range world.InventorySize / panelColumns {
		line := "   "
		for column := range panelColumns {
			if slot := row*panelColumns + column; slot < len(slots) {
				line += logTint[slots[slot]] + logGlyph + "  "
			} else {
				line += emptySlot + "  "
			}
		}
		lines = append(lines, pad(line, 3+panelColumns*4))
	}
	return append(lines, pad("", 0))
}
