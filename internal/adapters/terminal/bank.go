package terminal

import (
	"fmt"
	"strings"

	"lapresregne/internal/world"
)

// The bank lays its stacks out in a grid as Old School RuneScape does, 8 across, each stack's
// count above its icon. A slot is five cells wide: a bracket or space, the glyph, a bracket or
// space, then a space; the count starts above the glyph.
const (
	bankPitch      = 5
	bankMaxColumns = 8
	// bankRows is how many rows of stacks the panel shows below its title, two lines each.
	bankRows = (panelHeight - slotLine) / 2
)

// bankColumns fits the grid beside the inventory panel: 8 across when there is room, fewer in a
// narrow forest.
func bankColumns(cols int) int {
	return max(1, min(bankMaxColumns, (cols-panelWidth-1-slotIndent)/bankPitch))
}

// bankWidth is the bank panel's width in terminal cells for a number of columns.
func bankWidth(columns int) int { return slotIndent + columns*bankPitch }

// Counts are tinted as in Old School RuneScape: yellow, white from 100K, green from 10M.
var (
	countYellow = foreground([3]int{232, 212, 96})
	countWhite  = foreground([3]int{236, 236, 228})
	countGreen  = foreground([3]int{120, 220, 140})
)

func countTint(count int) string {
	switch {
	case count < 100_000:
		return countYellow
	case count < 10_000_000:
		return countWhite
	}
	return countGreen
}

// bankPanel lays out the bank panel's lines from a first row of stacks on, each padded to the
// panel's width in visible cells, with the highlighted stack bracketed.
func bankPanel(bank world.Bank, columns int, cursor bankCursor, top int) []string {
	width := bankWidth(columns)
	pad := func(text string, visible int) string {
		return text + strings.Repeat(" ", max(0, width-visible))
	}
	title := "   bank"
	lines := []string{pad("", 0), pad(panelText+title, len(title)), pad("", 0)}
	stacks := bank.Stacks()
	if len(stacks) == 0 {
		empty := "   empty"
		lines = append(lines, pad(foreground([3]int{58, 72, 62})+empty, len(empty)))
	}
	for row := top; row < top+bankRows; row++ {
		if row*columns >= len(stacks) {
			break
		}
		counts, icons := strings.Repeat(" ", slotIndent), strings.Repeat(" ", slotIndent)
		visible := slotIndent
		for column := range columns {
			index := row*columns + column
			if index >= len(stacks) {
				break
			}
			stack := stacks[index]
			counts += " " + countTint(stack.Count) + fmt.Sprintf("%-4s", bankCount(stack.Count))
			if cursor.active && cursor.stack == index {
				icons += "[" + icon(stack.Item) + panelText + "] "
			} else {
				icons += " " + icon(stack.Item) + panelText + "  "
			}
			visible += bankPitch
		}
		lines = append(lines, pad(counts, visible), pad(icons, visible))
	}
	for len(lines) < panelHeight {
		lines = append(lines, pad("", 0))
	}
	return lines
}

// bankCount shortens a stack's count to four characters, in thousands or millions when large.
func bankCount(count int) string {
	switch {
	case count < 10_000:
		return fmt.Sprint(count)
	case count < 1_000_000:
		return fmt.Sprintf("%dK", count/1_000)
	}
	return fmt.Sprintf("%dM", count/1_000_000)
}

// toggleBank opens or closes the bank; opening it shows the inventory to deposit from.
func (r *renderer) toggleBank() {
	r.bankOpen = !r.bankOpen
	if r.bankOpen {
		r.inventoryOpen = true
	}
}

// clickSlot highlights a clicked inventory slot, and deposits its item while the bank is open.
func (r *renderer) clickSlot(slot int) {
	r.cursor = slotCursor{active: true, slot: slot}
	if r.bankOpen {
		r.world.Deposit(slot)
	}
}

// removeFromSlot deposits the highlighted item while the bank is open, and drops it otherwise.
func (r *renderer) removeFromSlot() {
	remove := r.world.Drop
	if r.bankOpen {
		remove = r.world.Deposit
	}
	r.cursor.remove(r.world, remove)
}

// stackAt finds the bank stack under a 1-based terminal cell of the open bank panel, on its count
// or its icon.
func (r *renderer) stackAt(column, row int) (int, bool) {
	if !r.bankOpen || r.screenWidth == 0 || r.canvas.width == 0 {
		return 0, false
	}
	cols := r.canvas.width / 2
	columns := bankColumns(cols)
	panelColumn := column - 1 - (r.screenWidth-cols)/2 - slotIndent
	gridRow := (row - 2 - slotLine) / 2
	if panelColumn < 0 || panelColumn >= columns*bankPitch || row-2-slotLine < 0 || gridRow >= bankRows {
		return 0, false
	}
	stack := (r.bankTop+gridRow)*columns + panelColumn/bankPitch
	if stack >= len(r.world.Bank().Stacks()) {
		return 0, false
	}
	return stack, true
}

// bankCursor is the highlighted bank stack, chosen with Shift-[ and Shift-] or a click.
type bankCursor struct {
	active bool
	stack  int
}

// move steps the highlight through a number of stacks in reading order, wrapping at the ends.
func (c *bankCursor) move(step, stacks int) {
	if stacks == 0 {
		return
	}
	if !c.active {
		c.active, c.stack = true, 0
		if step < 0 {
			c.stack = stacks - 1
		}
		return
	}
	c.stack = (c.stack + step + stacks) % stacks
}

// keep holds the highlight on a stack that is still there after the bank shrinks.
func (c *bankCursor) keep(stacks int) {
	if stacks == 0 {
		*c = bankCursor{}
		return
	}
	c.stack = min(c.stack, stacks-1)
}

// scrolled is the first row of stacks to show so the highlighted stack stays in view, moving as
// little as it can from the row shown before.
func scrolled(top int, cursor bankCursor, columns, stacks int) int {
	rows := (stacks + columns - 1) / columns
	if cursor.active {
		row := cursor.stack / columns
		top = min(top, row)
		top = max(top, row-bankRows+1)
	}
	return max(0, min(top, rows-bankRows))
}

// moveBankCursor steps the bank highlight, opening the bank so the highlight is never hidden.
func (r *renderer) moveBankCursor(step int) {
	if !r.bankOpen {
		r.toggleBank()
	}
	r.bankCursor.move(step, len(r.world.Bank().Stacks()))
}

// clickStack highlights a clicked bank stack and withdraws one item from it.
func (r *renderer) clickStack(stack int) {
	r.bankCursor = bankCursor{active: true, stack: stack}
	r.world.Withdraw(stack)
	r.bankCursor.keep(len(r.world.Bank().Stacks()))
}

// withdraw withdraws one item from the highlighted stack, or the first, while the bank is open,
// and drops the highlighted inventory item otherwise.
func (r *renderer) withdraw() {
	if !r.bankOpen {
		r.removeFromSlot()
		return
	}
	stacks := len(r.world.Bank().Stacks())
	if stacks == 0 {
		return
	}
	if !r.bankCursor.active {
		r.bankCursor = bankCursor{active: true}
	}
	r.bankCursor.keep(stacks)
	r.clickStack(r.bankCursor.stack)
}
