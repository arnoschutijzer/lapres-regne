package world

import "math"

// Doorway is a doorway along the west edge of the tile at Col, Row, or along its north edge.
type Doorway struct {
	Col, Row int
	North    bool
}

// doorReach is how far the middle of a doorway may lie from the hero for them to open or close
// its door: about a tile.
const doorReach = 2.0

// standingInTheWay is the message for closing a door on the hero standing in its doorway.
const standingInTheWay = "You can't close the door while you're standing in the way."

// middle is the middle of the doorway's edge.
func (d Doorway) middle() point {
	left, top := float64(d.Col)*TileSize, float64(d.Row)*TileSize
	if d.North {
		return point{left + TileSize/2, top}
	}
	return point{left, top + TileSize/2}
}

// edgeBetween is the edge between two tiles side by side.
func edgeBetween(a, b cell) Doorway {
	switch {
	case b.gx == a.gx+1:
		return Doorway{Col: b.gx, Row: b.gz}
	case b.gx == a.gx-1:
		return Doorway{Col: a.gx, Row: a.gz}
	case b.gz == a.gz+1:
		return Doorway{Col: b.gx, Row: b.gz, North: true}
	}
	return Doorway{Col: a.gx, Row: a.gz, North: true}
}

// DoorwayInReach finds the doorway nearest the hero within reach, or false where none is and in
// the endless forest.
func (w *World) DoorwayInReach() (Doorway, bool) {
	if w.terrain == nil {
		return Doorway{}, false
	}
	here := tileOf(w.hero.x, w.hero.z)
	var nearest Doorway
	nearestDistance, found := doorReach, false
	for row := here.gz - 1; row <= here.gz+1; row++ {
		for col := here.gx - 1; col <= here.gx+1; col++ {
			edges := w.here().edgesOf(cell{col, row})
			for _, doorway := range [2]struct {
				ground Ground
				north  bool
			}{{edges.West, false}, {edges.North, true}} {
				if doorway.ground != Door {
					continue
				}
				candidate := Doorway{Col: col, Row: row, North: doorway.north}
				middle := candidate.middle()
				if distance := math.Hypot(middle.x-w.hero.x, middle.z-w.hero.z); distance <= nearestDistance {
					nearest, nearestDistance, found = candidate, distance, true
				}
			}
		}
	}
	return nearest, found
}

// ToggleDoor opens the shut door in the doorway in reach, or closes the open one, reporting
// whether it moved. A door never closes on the hero standing in its doorway.
func (w *World) ToggleDoor() bool {
	doorway, ok := w.DoorwayInReach()
	if !ok {
		return false
	}
	return w.toggle(doorway)
}

// ApproachDoor walks the hero to their side of a doorway, round whatever blocks the way, and
// opens or closes its door on arrival.
func (w *World) ApproachDoor(doorway Doorway) {
	if w.terrain == nil {
		return
	}
	near := doorway.side(w.hero.x, w.hero.z).middle()
	w.WalkTo(near.x, near.z)
	w.doorApproach = doorApproach{active: true, doorway: doorway}
}

// doorApproach is a doorway the hero walks to, to open or close its door on arrival.
type doorApproach struct {
	active  bool
	doorway Doorway
}

// arriveAtDoor opens or closes the door the hero walked to, once they stop within reach of it.
func (w *World) arriveAtDoor() {
	if !w.doorApproach.active {
		return
	}
	doorway := w.doorApproach.doorway
	w.doorApproach = doorApproach{}
	middle := doorway.middle()
	if math.Hypot(middle.x-w.hero.x, middle.z-w.hero.z) <= doorReach {
		w.toggle(doorway)
	}
}

// side is the tile beside the doorway on the same side of it as a point.
func (d Doorway) side(x, z float64) cell {
	middle := d.middle()
	switch {
	case d.North && z < middle.z:
		return cell{d.Col, d.Row - 1}
	case !d.North && x < middle.x:
		return cell{d.Col - 1, d.Row}
	}
	return cell{d.Col, d.Row}
}

// toggle opens a shut door or closes an open one, reporting whether it moved.
func (w *World) toggle(doorway Doorway) bool {
	w.Stop()
	middle := doorway.middle()
	w.hero.Face(FacingToward(middle.x-w.hero.x, middle.z-w.hero.z))
	if w.here().shut[doorway] {
		delete(w.here().shut, doorway)
		w.sounds.OpenDoor()
		return true
	}
	w.here().shut[doorway] = true
	if !w.here().passable(w.hero.x, w.hero.z) {
		delete(w.here().shut, doorway)
		w.message = standingInTheWay
		return false
	}
	w.sounds.CloseDoor()
	return true
}
