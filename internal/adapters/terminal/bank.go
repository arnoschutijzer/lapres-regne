package terminal

import (
	"fmt"
	"strings"

	"lapresregne/internal/world"
)

// bankWidth sizes the bank panel in terminal cells; it is as tall as the inventory panel, and the
// two fit side by side in the narrowest forest.
const bankWidth = 21

// bankRows is how many stacks the bank panel lists, one a line below its title.
const bankRows = panelHeight - slotLine - 1

// bankPanel lays out the bank panel's lines, each padded to bankWidth visible cells.
func bankPanel(bank world.Bank) []string {
	pad := func(text string, visible int) string {
		return text + strings.Repeat(" ", max(0, bankWidth-visible))
	}
	title := "   bank"
	lines := []string{pad("", 0), pad(panelText+title, len(title)), pad("", 0)}
	stacks := bank.Stacks()
	if len(stacks) == 0 {
		empty := "   empty"
		lines = append(lines, pad(foreground([3]int{58, 72, 62})+empty, len(empty)))
	}
	for _, stack := range stacks[:min(len(stacks), bankRows)] {
		// Indent, icon, name and count, the count right-aligned at the panel's edge.
		text := fmt.Sprintf(" %-11.11s %4s", stack.Item, bankCount(stack.Count))
		lines = append(lines, "  "+icon(stack.Item)+panelText+text)
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

// stackAt finds the bank stack listed under a 1-based terminal cell of the open bank panel.
func (r *renderer) stackAt(column, row int) (int, bool) {
	if !r.bankOpen || r.screenWidth == 0 || r.canvas.width == 0 {
		return 0, false
	}
	cols := r.canvas.width / 2
	panelColumn := column - 1 - (r.screenWidth-cols)/2
	stack := row - 2 - slotLine
	if panelColumn < 0 || panelColumn >= bankWidth || stack < 0 || stack >= min(bankRows, len(r.world.Bank().Stacks())) {
		return 0, false
	}
	return stack, true
}
