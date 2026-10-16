package world

// InventorySize is how many items the hero carries, as in Old School RuneScape.
const InventorySize = 28

// Inventory holds what the hero carries. Logs do not stack: each takes a slot of its own.
type Inventory struct {
	slots []TreeKind
}

// Slots lists the logs carried, by the tree they came from, in the order they were cut.
func (i Inventory) Slots() []TreeKind { return append([]TreeKind(nil), i.slots...) }
func (i Inventory) Len() int          { return len(i.slots) }
func (i Inventory) Full() bool        { return len(i.slots) >= InventorySize }

// add puts a log in the next free slot, reporting whether there was room.
func (i *Inventory) add(kind TreeKind) bool {
	if i.Full() {
		return false
	}
	i.slots = append(i.slots, kind)
	return true
}
