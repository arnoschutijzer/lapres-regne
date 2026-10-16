package terminal

import (
	"math"

	"lapresregne/internal/world"
)

// gait follows the hero from frame to frame, so the detailed body turns smoothly toward where
// they walk, or toward their facing while they stand, and strides with the ground it covers.
type gait struct {
	started      bool
	lastX, lastZ float64
	// heading is the direction the body faces: 0 faces down the screen, π/2 faces right.
	heading float64
	// phase advances one full turn per stride cycle of ground covered; amount eases the stride
	// in from rest and back out, and speed is the eased pace in world units per second.
	phase, amount, speed float64
}

// cycle is the ground covered by one full stride, left and right foot, lengthening at speed.
func (g *gait) cycle() float64 {
	return 1.2 + 0.2*g.speed
}

// update follows the hero to x, z after dt seconds.
func (g *gait) update(x, z float64, facing world.Facing, dt float64) {
	dx, dz := x-g.lastX, z-g.lastZ
	g.lastX, g.lastZ = x, z
	// The first frame, and a jump too far for one frame, carry no motion.
	if !g.started || math.Hypot(dx, dz) > 1 {
		dx, dz = 0, 0
	}
	target := facingAngle(facing)
	if dx != 0 || dz != 0 {
		target = math.Atan2(dx, dz)
	}
	if !g.started {
		g.started, g.heading = true, target
		return
	}
	g.heading = wrapAngle(g.heading + angleBetween(g.heading, target)*(1-math.Exp(-14*dt)))
	covered := math.Hypot(dx, dz)
	if dt > 0 {
		g.speed += (covered/dt - g.speed) * (1 - math.Exp(-8*dt))
	}
	moving := 0.0
	if covered > 0 {
		moving = 1
	}
	g.amount += (moving - g.amount) * (1 - math.Exp(-8*dt))
	if g.amount < 0.001 {
		g.amount = 0
	}
	g.phase = math.Mod(g.phase+covered/g.cycle()*2*math.Pi, 2*math.Pi)
}

// facingAngle is the heading of a facing.
func facingAngle(facing world.Facing) float64 {
	switch facing {
	case world.FacingUp:
		return math.Pi
	case world.FacingLeft:
		return -math.Pi / 2
	case world.FacingRight:
		return math.Pi / 2
	}
	return 0
}

// angleBetween is the shortest turn from one heading to another.
func angleBetween(from, to float64) float64 {
	return wrapAngle(to - from)
}

// wrapAngle brings an angle into the range -π to π.
func wrapAngle(angle float64) float64 {
	return math.Remainder(angle, 2*math.Pi)
}
