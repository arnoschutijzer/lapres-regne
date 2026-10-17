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
// chopping the tree the hero walked up to, starts work in the next frame, since the walk there
// took this frame's time.
func (w *World) carryOutOrder(dt time.Duration, speed float64) {
	if w.order != nil {
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

// closeInOn walks the hero up to a target, round whatever blocks the way, giving up whatever they
// were doing, and acts once they are within a distance of it.
func (w *World) closeInOn(target point, within float64, arrive func() order) {
	w.Stop()
	route := w.route(target.x, target.z)
	w.order = &closeIn{mover: w, target: target, within: within, waypoints: route[:len(route)-1], arrive: arrive}
}

// aimHolder is an order that holds the hero facing their work, such as a chop facing its tree.
type aimHolder interface {
	holdsAim()
}

// AimAt faces the hero toward a world position, except while an order holds them facing their work.
func (w *World) AimAt(x, z float64) {
	if _, held := w.order.(aimHolder); held {
		return
	}
	w.hero.AimAt(x, z)
}
