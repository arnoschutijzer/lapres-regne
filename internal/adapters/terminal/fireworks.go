package terminal

import "math"

// The fireworks of a level-up, after Old School RuneScape's: bursts of sparks go off one after
// another around the hero's head, each in its own colour, flashing out, slowing, drooping and fading.
const (
	fireworkBursts = 7
	// burstEvery is how many seconds apart the bursts go off, and burstLife how long each lasts.
	burstEvery, burstLife = 0.38, 0.9
	sparksPerBurst        = 12
	// burstReach is how far a burst's sparks fly, in world units, before the view's minimum.
	burstReach = 0.55
	// burstMinimumPixels keeps a burst readable when the view is zoomed out to a small hero.
	burstMinimumPixels = 11.0
)

// fireworksDuration is how many seconds after a level-up the last burst dies down.
const fireworksDuration = burstEvery*(fireworkBursts-1) + burstLife

// fireworkInks are the bursts' colours in the order they go off: gold, green, white and ember.
var fireworkInks = [...]int{22, 23, 24, 25}

// drawFireworks sets off the bursts of the latest level-up around the hero.
func (r *renderer) drawFireworks(c *forestCanvas) {
	f := r.world
	levelUp, since, ok := f.LatestLevelUp()
	if !ok || since.Seconds() >= fireworksDuration {
		return
	}
	heroX, heroZ := f.Hero().Position()
	reach := max(burstReach, burstMinimumPixels/c.scale)
	// Positions around the head grow with the bursts, so zoomed out they still surround the hero.
	spread := reach / burstReach
	for i := range fireworkBursts {
		age := since.Seconds() - float64(i)*burstEvery
		if age < 0 || age >= burstLife {
			continue
		}
		// Each level-up places its bursts its own way, and every frame of it the same way.
		random := func(salt uint64) float64 { return f.Random(levelUp.Level, i, 140+salt) }
		// The bursts take turns to either side of the head and above it, as seen on the screen.
		side := float64(1 - 2*(i%2))
		across := side * spread * (0.25 + 0.45*random(1))
		toward := spread * 0.3 * (random(0) - 0.5)
		x := heroX + across*c.yawCos - toward*c.yawSin
		z := heroZ + across*c.yawSin + toward*c.yawCos
		y := 1.6 + spread*(0.1+0.6*random(2))
		drawBurst(c, x, y, z, reach, age/burstLife, random(3)*math.Pi, fireworkInks[i%len(fireworkInks)])
	}
}

// drawBurst draws one burst a fraction of the way through its life: a flash, then sparks that
// streak out, slow, droop and fade. turn spins the sparks' directions about the upright axis.
func drawBurst(c *forestCanvas, x, y, z, reach, life, turn float64, ink int) {
	if life < 0.12 {
		flash := reach * 0.15
		c.line(x-flash, y, z, x+flash, y, z, 1, ink)
		c.line(x, y-flash, z, x, y+flash, z, 1, ink)
	}
	// The sparks race out and slow down, their tails shortening as they go.
	out := reach * (1 - math.Pow(1-life, 3))
	tail := reach * 0.45 * (1 - life)
	droop := reach * 0.5 * life * life
	light := 1.0
	if life > 0.5 {
		light = 1 - 1.5*(life-0.5)
	}
	for i := range sparksPerBurst {
		dx, dy, dz := sparkDirection(i, turn)
		inner := max(0, out-tail)
		c.line(x+dx*inner, y+dy*inner-droop, z+dz*inner, x+dx*out, y+dy*out-droop, z+dz*out, light, ink)
	}
}

// sparkDirection spreads a burst's sparks evenly over a sphere, turned about the upright axis.
func sparkDirection(i int, turn float64) (dx, dy, dz float64) {
	dy = 1 - 2*(float64(i)+0.5)/sparksPerBurst
	ring := math.Sqrt(1 - dy*dy)
	angle := float64(i)*2.39996 + turn
	return math.Cos(angle) * ring, dy, math.Sin(angle) * ring
}
