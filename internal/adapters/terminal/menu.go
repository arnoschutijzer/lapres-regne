package terminal

import "lapresregne/internal/world"

// menuFor is what the open menu offers actions for.
type menuFor byte

const (
	menuTree menuFor = iota
	menuDoorway
	menuFlight
	// menuSlot offers to deposit the item in an inventory slot, and menuStack to withdraw from a
	// bank stack; menuTarget is which slot or stack.
	menuSlot
	menuStack
)

// The actions offered for a staircase or ladder.
const (
	climbUp   = "Climb-up"
	climbDown = "Climb-down"
)

// toggleMenu closes an open menu, or opens the menu of actions for what is in reach.
func (r *renderer) toggleMenu() {
	if r.menu != nil {
		r.menu = nil
		return
	}
	r.openMenu()
}

// openMenu opens the menu of actions for the tree in reach, else for the doorway in reach, else
// for the staircase or ladder in reach.
func (r *renderer) openMenu() {
	if tree, ok := r.world.TreeInReach(); ok {
		r.menuTitle, r.menu, r.menuFor = tree.Kind.String(), []string{"Chop down"}, menuTree
		return
	}
	if doorway, ok := r.world.DoorwayInReach(); ok {
		action := "Close"
		if r.shut(doorway) {
			action = "Open"
		}
		r.menuTitle, r.menu, r.menuFor = r.doorwayName(doorway), []string{action}, menuDoorway
		return
	}
	if flight, ok := r.world.FlightInReach(); ok && (flight.Up || flight.Down) {
		var actions []string
		if flight.Up {
			actions = append(actions, climbUp)
		}
		if flight.Down {
			actions = append(actions, climbDown)
		}
		title := "Staircase"
		if flight.Kind == world.Ladder {
			title = "Ladder"
		}
		r.menuTitle, r.menu, r.menuFor = title, actions, menuFlight
	}
}

// choose carries out the action at an index on the open menu, counting from 0, and closes it.
func (r *renderer) choose(index int) {
	if index >= len(r.menu) {
		return
	}
	switch r.menuFor {
	case menuTree:
		r.world.Chop()
	case menuDoorway:
		r.world.ToggleDoor()
	case menuFlight:
		r.world.Climb(r.menu[index] == climbUp)
	case menuSlot:
		r.deposit(r.menuTarget, index == 1)
	case menuStack:
		r.withdrawFrom(r.menuTarget, index == 1)
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
