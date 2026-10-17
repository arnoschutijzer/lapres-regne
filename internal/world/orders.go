package world

import (
	"math"
	"time"
)

// order is the one player order the hero carries out at a time: walking somewhere, closing in on
// something to act on it, or working at it. A new order replaces the current one. Background
// effects, such as regrowth and the level-up celebration, run alongside orders and are no orders.
type order interface {
	// carryOut works on the order for dt, moving the hero at speed, and returns the order that
	// follows: the same one while unfinished, the one it hands over to, or nil once done.
	carryOut(dt time.Duration, speed float64) order
}

// carryOutOrder works on the current order for dt. An order handed over on arrival, such as
// chopping the tree the hero walked up to, starts work in the same frame.
func (w *World) carryOutOrder(dt time.Duration, speed float64) {
	if w.order == nil {
		return
	}
	current := w.order
	w.order = current.carryOut(dt, speed)
	if w.order != nil && w.order != current {
		w.order = w.order.carryOut(dt, speed)
	}
}

// Stop cancels the current order and clears the last message.
func (w *World) Stop() {
	w.order = nil
	w.message = ""
}

// mover is what walking orders need from the world around the hero.
type mover interface {
	Hero() *Character
	// follow walks up to step toward the waypoints in turn, returning the waypoints still ahead,
	// and false when the hero cannot get any closer to the next one.
	follow(waypoints []point, step float64) ([]point, bool)
	// wading shortens a distance walked while the hero stands in a river.
	wading(distance float64) float64
}

// walk is an order to walk along waypoints round whatever blocks the way, and act on arrival.
type walk struct {
	mover     mover
	waypoints []point
	// arrive acts once the hero stops, on the destination or as near as they could get to it,
	// and returns the order that follows; nil acts on nothing.
	arrive func() order
}

func (o *walk) carryOut(dt time.Duration, speed float64) order {
	waypoints, moving := o.mover.follow(o.waypoints, o.mover.wading(speed*dt.Seconds()))
	o.waypoints = waypoints
	// A hero who cannot get any closer gives up on the destination.
	if len(waypoints) > 0 && moving {
		return o
	}
	if o.arrive == nil {
		return nil
	}
	return o.arrive()
}

// closeIn is an order to walk round whatever blocks the way up to a target, stopping within a
// distance of it, and act there. A hero who cannot get any closer gives up without acting.
type closeIn struct {
	mover  mover
	target point
	within float64
	// waypoints lead round whatever blocks the way, before the last stretch straight to the target.
	waypoints []point
	// arrive acts once the hero is within reach and returns the order that follows.
	arrive func() order
}

func (o *closeIn) carryOut(dt time.Duration, speed float64) order {
	m, hero := o.mover, o.mover.Hero()
	if len(o.waypoints) > 0 {
		waypoints, moving := m.follow(o.waypoints, m.wading(speed*dt.Seconds()))
		o.waypoints = waypoints
		if !moving {
			return nil
		}
		return o
	}
	x, z := hero.Position()
	dx, dz := o.target.x-x, o.target.z-z
	remaining := math.Hypot(dx, dz) - o.within
	if remaining > 0 {
		if hero.advance(dx, dz, math.Min(remaining, m.wading(speed*dt.Seconds()))) == 0 {
			return nil
		}
		x, z = hero.Position()
		remaining = math.Hypot(o.target.x-x, o.target.z-z) - o.within
	}
	if remaining > 1e-9 {
		return o
	}
	return o.arrive()
}

// TreeInReach finds the standing tree the hero acts on: the nearest one in front within reach,
// else the nearest one within reach in any direction, so overshooting a tree still reaches it.
func (w *World) TreeInReach() (Tree, bool) { return w.woodcutting.treeInReach() }

// Chop orders the hero to chop the tree in reach, reporting whether chopping started. A tree above
// the hero's level leaves a message instead.
func (w *World) Chop() bool {
	tree, ok := w.woodcutting.treeInReach()
	if !ok {
		return false
	}
	chop := w.woodcutting.start(tree)
	if chop == nil {
		return false
	}
	w.order = chop
	return true
}

// Approach walks the hero to a standing tree, round whatever blocks the way, and chops it on arrival,
// reporting whether the tree still stands.
func (w *World) Approach(tree Tree) bool {
	standing, exists := w.TreeAt(tree.cell.gx, tree.cell.gz)
	if !exists || standing.Stump {
		return false
	}
	w.Stop()
	route := w.route(standing.X, standing.Z)
	w.order = &closeIn{
		mover:     w,
		target:    point{standing.X, standing.Z},
		within:    arrival,
		waypoints: route[:len(route)-1],
		arrive:    func() order { return w.woodcutting.start(standing) },
	}
	return true
}

// chopping is the chop in progress, if any.
func (w *World) chopping() (*chop, bool) {
	chop, ok := w.order.(*chop)
	return chop, ok
}

func (w *World) Chopping() bool {
	_, chopping := w.chopping()
	return chopping
}

// AimAt faces the hero toward a world position, except while a chop holds them toward the tree.
func (w *World) AimAt(x, z float64) {
	if w.Chopping() {
		return
	}
	w.hero.AimAt(x, z)
}

// LastStrike tells which strike of the chop landed last, counting from zero, and how long ago.
func (w *World) LastStrike() (index int, since time.Duration, ok bool) {
	chop, chopping := w.chopping()
	if !chopping {
		return 0, 0, false
	}
	index, since = chop.lastStrike()
	return index, since, true
}

// ChoppedTree is the tree being chopped, if any.
func (w *World) ChoppedTree() (Tree, bool) {
	chop, chopping := w.chopping()
	if !chopping {
		return Tree{}, false
	}
	return w.TreeAt(chop.tree.gx, chop.tree.gz)
}
