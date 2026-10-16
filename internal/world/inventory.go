package world

import "time"

// InventorySize is how many items the hero carries, as in Old School RuneScape.
const InventorySize = 28

// Slot is one place in the inventory, holding a log from a kind of tree when filled.
type Slot struct {
	Kind   TreeKind
	Filled bool
}

// Inventory holds what the hero carries. Logs do not stack: each takes a slot of its own, and
// slots keep their place, so dropping a log leaves a gap that the next log fills.
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

// FirstFilled finds the first slot holding a log.
func (i Inventory) FirstFilled() (int, bool) {
	for index, slot := range i.slots {
		if slot.Filled {
			return index, true
		}
	}
	return 0, false
}

// add puts a log in the first empty slot, reporting whether there was room.
func (i *Inventory) add(kind TreeKind) bool {
	for index, slot := range i.slots {
		if !slot.Filled {
			i.slots[index] = Slot{Kind: kind, Filled: true}
			return true
		}
	}
	return false
}

// drop empties a slot, returning the log it held.
func (i *Inventory) drop(index int) (TreeKind, bool) {
	if index < 0 || index >= InventorySize || !i.slots[index].Filled {
		return 0, false
	}
	kind := i.slots[index].Kind
	i.slots[index] = Slot{}
	return kind, true
}

// groundTime is how long a dropped log lies on the ground: Old School RuneScape's 3 minutes
// before a dropped tradeable item disappears.
const groundTime = 3 * time.Minute

// GroundLog is a dropped log lying in the world.
type GroundLog struct {
	X, Z float64
	Kind TreeKind
	left time.Duration
}

// Drop puts the log in an inventory slot on the ground under the hero, reporting whether
// the slot held one.
func (w *World) Drop(slot int) bool {
	kind, ok := w.hero.inventory.drop(slot)
	if !ok {
		return false
	}
	w.ground = append(w.ground, GroundLog{X: w.hero.x, Z: w.hero.z, Kind: kind, left: groundTime})
	return true
}

// GroundLogs lists the dropped logs still lying on the ground.
func (w *World) GroundLogs() []GroundLog { return append([]GroundLog(nil), w.ground...) }

// despawn clears dropped logs once their time on the ground is up.
func (w *World) despawn(dt time.Duration) {
	kept := w.ground[:0]
	for _, log := range w.ground {
		if log.left > dt {
			log.left -= dt
			kept = append(kept, log)
		}
	}
	w.ground = kept
}
