package terminal

import "lapresregne/internal/world"

// toggleMenu closes an open menu, or opens the menu of actions for the tree in reach, else for
// the doorway in reach.
func (r *renderer) toggleMenu() {
	if r.menu != nil {
		r.menu = nil
		return
	}
	if tree, ok := r.world.TreeInReach(); ok {
		r.menuTitle, r.menu, r.menuDoorway = tree.Kind.String(), []string{"Chop down"}, false
		return
	}
	if doorway, ok := r.world.DoorwayInReach(); ok {
		action := "Close"
		if r.shut(doorway) {
			action = "Open"
		}
		r.menuTitle, r.menu, r.menuDoorway = r.doorwayName(doorway), []string{action}, true
	}
}

// chooseFirst carries out the first action on the open menu and closes it.
func (r *renderer) chooseFirst() {
	if r.menu == nil {
		return
	}
	if r.menuDoorway {
		r.world.ToggleDoor()
	} else {
		r.world.Chop()
	}
	r.menu = nil
}

// shut tells whether the door in a doorway is shut.
func (r *renderer) shut(doorway world.Doorway) bool {
	edges := r.world.EdgesAt((float64(doorway.Col)+0.5)*world.TileSize, (float64(doorway.Row)+0.5)*world.TileSize)
	if doorway.North {
		return edges.NorthShut
	}
	return edges.WestShut
}

// doorwayName names a doorway as drawn: a door where walls meet it, else a gate in a fence.
func (r *renderer) doorwayName(doorway world.Doorway) string {
	endCol, endRow := doorway.Col, doorway.Row+1
	if doorway.North {
		endCol, endRow = doorway.Col+1, doorway.Row
	}
	if r.built.walledDoorway(doorway.Col, doorway.Row, endCol, endRow) {
		return "Door"
	}
	return "Gate"
}
