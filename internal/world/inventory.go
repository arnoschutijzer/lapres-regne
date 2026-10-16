package world

import "time"

// InventorySize is how many items the hero carries, as in Old School RuneScape.
const InventorySize = 28

// Slot is one place in the inventory, holding an item when filled.
type Slot struct {
	Item   Item
	Filled bool
}

// Inventory holds what the hero carries. Items do not stack: each takes a slot of its own, and
// slots keep their place, so dropping an item leaves a gap that the next item fills.
type Inventory struct {
	slots [InventorySize]Slot
}

// Slots lists all 28 slots in reading order.
func (i Inventory) Slots() []Slot { return append([]Slot(nil), i.slots[:]...) }

// Len counts the filled slots.
func (i Inventory) Len() int {
	filled := 0
	for _, slot := range i.slots {
		if slot.Filled {
			filled++
		}
	}
	return filled
}

func (i Inventory) Full() bool { return i.Len() == InventorySize }

// FirstFilled finds the first slot holding an item.
func (i Inventory) FirstFilled() (int, bool) {
	for index, slot := range i.slots {
		if slot.Filled {
			return index, true
		}
	}
	return 0, false
}

// add puts an item in the first empty slot, reporting whether there was room.
func (i *Inventory) add(item Item) bool {
	for index, slot := range i.slots {
		if !slot.Filled {
			i.slots[index] = Slot{Item: item, Filled: true}
			return true
		}
	}
	return false
}

// drop empties a slot, returning the item it held.
func (i *Inventory) drop(index int) (Item, bool) {
	if index < 0 || index >= InventorySize || !i.slots[index].Filled {
		return 0, false
	}
	item := i.slots[index].Item
	i.slots[index] = Slot{}
	return item, true
}

// groundTime is how long a dropped item lies on the ground: Old School RuneScape's 3 minutes
// before a dropped tradeable item disappears.
const groundTime = 3 * time.Minute

// GroundItem is a dropped item lying in the world.
type GroundItem struct {
	X, Z float64
	Item Item
	left time.Duration
}

// Drop puts the item in an inventory slot on the ground under the hero, reporting whether
// the slot held one.
func (w *World) Drop(slot int) bool {
	item, ok := w.hero.inventory.drop(slot)
	if !ok {
		return false
	}
	w.ground = append(w.ground, GroundItem{X: w.hero.x, Z: w.hero.z, Item: item, left: groundTime})
	w.sounds.Drop(item.Landing())
	return true
}

// GroundItems lists the dropped items still lying on the ground.
func (w *World) GroundItems() []GroundItem { return append([]GroundItem(nil), w.ground...) }

// despawn clears dropped items once their time on the ground is up.
func (w *World) despawn(dt time.Duration) {
	kept := w.ground[:0]
	for _, item := range w.ground {
		if item.left > dt {
			item.left -= dt
			kept = append(kept, item)
		}
	}
	w.ground = kept
}
