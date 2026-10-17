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
