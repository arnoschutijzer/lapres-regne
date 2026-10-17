package world

// Stack is a pile of one kind of item kept in the bank.
type Stack struct {
	Item  Item
	Count int
}

// Bank keeps the items the hero deposits, wherever they are. As in Old School RuneScape, every
// kind of item stacks in the bank, so each kind takes one place, in the order first deposited.
type Bank struct {
	stacks []Stack
}

// Stacks lists the piles in the bank in order.
func (b Bank) Stacks() []Stack { return append([]Stack(nil), b.stacks...) }

// deposit adds an item to its kind's pile, starting a new pile after the others for a new kind.
func (b *Bank) deposit(item Item) {
	for index := range b.stacks {
		if b.stacks[index].Item == item {
			b.stacks[index].Count++
			return
		}
	}
	b.stacks = append(b.stacks, Stack{Item: item, Count: 1})
}

// Bank is what the hero keeps in the bank.
func (w *World) Bank() Bank { return w.bank }

// Deposit moves the item in an inventory slot into the bank, reporting whether the slot held one.
func (w *World) Deposit(slot int) bool {
	item, ok := w.hero.inventory.drop(slot)
	if !ok {
		return false
	}
	w.bank.deposit(item)
	return true
}

// withdraw takes one item from the pile in a place, clearing the place once the pile is gone.
func (b *Bank) withdraw(index int) Item {
	item := b.stacks[index].Item
	b.stacks[index].Count--
	if b.stacks[index].Count == 0 {
		b.stacks = append(b.stacks[:index], b.stacks[index+1:]...)
	}
	return item
}

// Withdraw takes one item from a pile in the bank into the first empty inventory slot, reporting
// whether it did; a full inventory leaves the item in the bank.
func (w *World) Withdraw(stack int) bool {
	if stack < 0 || stack >= len(w.bank.stacks) {
		return false
	}
	if w.hero.inventory.Full() {
		w.say("You don't have enough inventory space.")
		return false
	}
	w.hero.inventory.add(w.bank.withdraw(stack))
	return true
}

// DepositAll moves every item of the kind in an inventory slot into the bank, reporting whether
// the slot held one.
func (w *World) DepositAll(slot int) bool {
	slots := w.hero.inventory.Slots()
	if slot < 0 || slot >= InventorySize || !slots[slot].Filled {
		return false
	}
	for index, carried := range slots {
		if carried.Filled && carried.Item == slots[slot].Item {
			w.Deposit(index)
		}
	}
	return true
}

// WithdrawAll takes every item of a pile in the bank into the inventory, or as many as fit,
// reporting whether it took any.
func (w *World) WithdrawAll(stack int) bool {
	if stack < 0 || stack >= len(w.bank.stacks) {
		return false
	}
	item := w.bank.stacks[stack].Item
	took := false
	// The pile's place empties once its last item is taken.
	for stack < len(w.bank.stacks) && w.bank.stacks[stack].Item == item && w.Withdraw(stack) {
		took = true
	}
	return took
}
