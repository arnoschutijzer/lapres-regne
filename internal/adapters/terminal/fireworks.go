package terminal

import (
	"math"

	"lapresregne/internal/world"
)

// The fireworks' palette entries.
const (
	goldFireworks = 22 + iota
	greenFireworks
	whiteFireworks
	emberFireworks
	cyanFireworks
	magentaFireworks
	redFireworks
)

// fireworkInks are every colour the fireworks burst in.
var fireworkInks = [...]int{goldFireworks, greenFireworks, whiteFireworks, emberFireworks, cyanFireworks, magentaFireworks, redFireworks}

// fireworksDisplay is how a level-up's fireworks go off, after Old School RuneScape's: bursts of
// sparks one after another around the hero's head, each flashing out, slowing, drooping and fading.
type fireworksDisplay struct {
	bursts int
	// every is how many seconds apart the bursts go off, and life how long each lasts.
	every, life float64
	sparks      int
	// reach is how far a burst's sparks fly, in world units, but never less than minimumPixels
	// on the canvas, so a zoomed-out hero still gets readable bursts.
	reach, minimumPixels float64
	// beside and above are how far from the head the bursts go off, to either side on the screen
	// and upward, from the nearest to the farthest, in reaches; depth is how far they scatter
	// toward and away from the viewer.
	beside, above [2]float64
	depth         float64
	// inks are the bursts' colours in the order they go off.
	inks []int
	// core lights a smaller burst of gold inside each burst of another colour.
	core bool
}

// displayFor picks a level-up's fireworks: the usual four colours, bigger and more colourful ones
// for a 99, and red and gold ones that last longer for the max total level.
func displayFor(milestone world.Milestone) fireworksDisplay {
	switch milestone {
	case world.SkillMastered:
		return fireworksDisplay{
			bursts: 16, every: 0.2, life: 1, sparks: 16, reach: 0.9, minimumPixels: 18,
			beside: [2]float64{0.3, 1.8}, above: [2]float64{0.3, 1.9}, depth: 0.6,
			inks: []int{cyanFireworks, goldFireworks, magentaFireworks, greenFireworks, redFireworks, whiteFireworks},
		}
	case world.TotalMaxed:
		return fireworksDisplay{
			bursts: 30, every: 0.22, life: 1.2, sparks: 20, reach: 1, minimumPixels: 20,
			beside: [2]float64{0.3, 2}, above: [2]float64{0.3, 2}, depth: 0.6,
			inks: []int{redFireworks, goldFireworks}, core: true,
		}
	}
	return fireworksDisplay{
		bursts: 7, every: 0.38, life: 0.9, sparks: 12, reach: 0.55, minimumPixels: 11,
		beside: [2]float64{0.45, 1.27}, above: [2]float64{0.18, 1.27}, depth: 0.55,
		inks: []int{goldFireworks, greenFireworks, whiteFireworks, emberFireworks},
	}
}

// duration is how many seconds after the level-up the last burst dies down.
func (d fireworksDisplay) duration() float64 {
	return d.every*float64(d.bursts-1) + d.life
}

// activeBursts counts the bursts in the air some seconds after the level-up.
func (d fireworksDisplay) activeBursts(since float64) int {
	active := 0
	for i := range d.bursts {
		if age := since - float64(i)*d.every; age >= 0 && age < d.life {
			active++
		}
	}
	return active
}

// drawFireworks sets off the latest level-up's fireworks around the hero.
func (r *renderer) drawFireworks(c *forestCanvas) {
	f := r.world
	levelUp, since, ok := f.LatestLevelUp()
	if !ok {
		return
	}
	heroX, heroZ := f.Hero().Position()
	c.standOn(heroX, heroZ)
	drawDisplay(c, displayFor(levelUp.Milestone), levelUp.Level, since.Seconds(), heroX, heroZ, f.Random)
}

// drawDisplay draws a display some seconds after reaching a level, around a hero standing at x, z.
// random places each level's bursts its own way, and every frame of them the same way.
func drawDisplay(c *forestCanvas, d fireworksDisplay, level int, since, heroX, heroZ float64, random func(x, z int, salt uint64) float64) {
	reach := max(d.reach, d.minimumPixels/c.scale)
	// The highest sparks reach a burst's own reach above the highest burst, shifted by its scatter
	// toward or away from the viewer; the bursts shrink to fit above the head, though never below
	// a few pixels.
	_, headY, _ := c.project(heroX, 1.6, heroZ)
	tallest := ((d.above[1]+1)*c.tiltCos + d.depth/2*c.tiltSin) * c.scale
	reach = max(6/c.scale, min(reach, 0.9*headY/tallest))
	between := func(span [2]float64, u float64) float64 { return reach * (span[0] + (span[1]-span[0])*u) }
	for i := range d.bursts {
		age := since - float64(i)*d.every
		if age < 0 || age >= d.life {
			continue
		}
		burstRandom := func(salt uint64) float64 { return random(level, i, 140+salt) }
		// The bursts take turns to either side of the head and above it, as seen on the screen.
		across := float64(1-2*(i%2)) * between(d.beside, burstRandom(1))
		toward := reach * d.depth * (burstRandom(0) - 0.5)
		x := heroX + across*c.yawCos - toward*c.yawSin
		z := heroZ + across*c.yawSin + toward*c.yawCos
		y := 1.6 + between(d.above, burstRandom(2))
		ink := d.inks[i%len(d.inks)]
		life, turn := age/d.life, burstRandom(3)*math.Pi
		drawBurst(c, x, y, z, reach, life, turn, d.sparks, ink)
		if d.core && ink != goldFireworks {
			drawBurst(c, x, y, z, reach*0.5, life, turn+0.3, d.sparks/2, goldFireworks)
		}
	}
}

// drawBurst draws one burst a fraction of the way through its life: a flash, then sparks that
// streak out, slow, droop and fade. turn spins the sparks' directions about the upright axis.
func drawBurst(c *forestCanvas, x, y, z, reach, life, turn float64, sparks, ink int) {
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
	for i := range sparks {
		dx, dy, dz := sparkDirection(i, sparks, turn)
		inner := max(0, out-tail)
		c.line(x+dx*inner, y+dy*inner-droop, z+dz*inner, x+dx*out, y+dy*out-droop, z+dz*out, light, ink)
	}
}

// sparkDirection spreads a burst's sparks evenly over a sphere, turned about the upright axis.
func sparkDirection(i, sparks int, turn float64) (dx, dy, dz float64) {
	dy = 1 - 2*(float64(i)+0.5)/float64(sparks)
	ring := math.Sqrt(1 - dy*dy)
	angle := float64(i)*2.39996 + turn
	return math.Cos(angle) * ring, dy, math.Sin(angle) * ring
}
