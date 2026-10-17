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

// walk is an order to walk along waypoints round whatever blocks the way, and act on arrival.
type walk struct {
	world     *World
	waypoints []point
	// arrive acts once the hero stops, on the destination or as near as they could get to it,
	// and returns the order that follows; nil acts on nothing.
	arrive func() order
}

func (o *walk) carryOut(dt time.Duration, speed float64) order {
	waypoints, moving := o.world.follow(o.waypoints, o.world.wading(speed*dt.Seconds()))
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
	world  *World
	target point
	within float64
	// waypoints lead round whatever blocks the way, before the last stretch straight to the target.
	waypoints []point
	// arrive acts once the hero is within reach and returns the order that follows.
	arrive func() order
}

func (o *closeIn) carryOut(dt time.Duration, speed float64) order {
	w := o.world
	if len(o.waypoints) > 0 {
		waypoints, moving := w.follow(o.waypoints, w.wading(speed*dt.Seconds()))
		o.waypoints = waypoints
		if !moving {
			return nil
		}
		return o
	}
	x, z := w.hero.Position()
	dx, dz := o.target.x-x, o.target.z-z
	remaining := math.Hypot(dx, dz) - o.within
	if remaining > 0 {
		if w.hero.advance(dx, dz, math.Min(remaining, w.wading(speed*dt.Seconds()))) == 0 {
			return nil
		}
		remaining = math.Hypot(o.target.x-w.hero.x, o.target.z-w.hero.z) - o.within
	}
	if remaining > 1e-9 {
		return o
	}
	return o.arrive()
}
