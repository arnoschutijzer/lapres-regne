package terminal

import (
	"math"

	"dither-cli/internal/world"
)

// strokePeriod is one unhurried axe stroke, in seconds.
var strokePeriod = world.StrikeInterval.Seconds()

// axeAngle eases the axe up overhead after it lands, then brings it down quickly onto the trunk.
func axeAngle(sinceHit float64) float64 {
	const low, high, raising = 0.2, 1.3, 0.75
	progress := sinceHit / strokePeriod
	if progress < raising {
		u := progress / raising
		return low + (high-low)*u*u*(3-2*u)
	}
	return high - (high-low)*(progress-raising)/(1-raising)
}

// jolt nudges a struck tree once, then lets it settle before the next stroke.
func jolt(sinceHit float64) float64 {
	return 0.35 * math.Exp(-14*sinceHit) * math.Cos(sinceHit*30)
}

// drawAxe swings an axe from the hero's hands toward the faced tree.
func drawAxe(c *forestCanvas, x, z float64, facing world.Facing, sinceHit float64) {
	dx, dz := 0.0, 1.0
	switch facing {
	case world.FacingLeft:
		dx, dz = -1, 0
	case world.FacingRight:
		dx, dz = 1, 0
	case world.FacingUp:
		dx, dz = 0, -1
	}
	angle := axeAngle(sinceHit)
	reachOut, lift := math.Cos(angle)*0.75, math.Sin(angle)*0.75
	headX, headY, headZ := x+dx*reachOut, 1.0+lift, z+dz*reachOut
	c.line(x, 1.0, z, headX, headY, headZ, 0.62, 4)
	c.line(headX, headY-0.18, headZ, headX+dx*0.08, headY+0.18, headZ+dz*0.08, 1, 9)
}

// chipFlight is how long wood chips stay in the air after a strike.
const chipFlight = 0.3

// drawChips throws a few pale chips from the struck trunk, each tree and stroke along its own arcs.
func drawChips(c *forestCanvas, f *world.World, tree world.Tree, index int, t float64) {
	if t > chipFlight {
		return
	}
	seed := index + int(math.Floor(tree.X*31))
	for chip := range 4 {
		chip += int(math.Floor(tree.Z * 31))
		angle := f.Random(seed, chip, 61) * 2 * math.Pi
		speed := 1.2 + f.Random(seed, chip, 62)*1.4
		lift := 1.5 + f.Random(seed, chip, 63)*1.5
		x := tree.X + math.Cos(angle)*speed*t
		z := tree.Z + math.Sin(angle)*speed*t*0.5
		y := 1.0 + lift*t - 4.9*t*t
		px, py, depth := c.project(x, y, z)
		c.plot(int(math.Round(px)), int(math.Round(py)), depth-0.5, 0.9, 10)
	}
}
