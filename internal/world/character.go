package world

import (
	"math"
	"time"
)

type Facing byte

const (
	FacingDown Facing = iota
	FacingUp
	FacingLeft
	FacingRight
)

func (f Facing) String() string {
	return [...]string{"down", "up", "left", "right"}[f]
}

// stride is the distance covered by one counted step.
const stride = 0.55

// Character is the hero exploring the world.
type Character struct {
	x, z         float64
	steps        int
	stepDistance float64
	facing       Facing
	aimed        bool
	lastWalk     time.Time
	inventory    Inventory
	woodcutting  Skill
	// attempts counts every roll for a log, so a restarted chop rolls afresh.
	attempts int
}

func (c *Character) Position() (x, z float64) { return c.x, c.z }
func (c *Character) Steps() int               { return c.steps }
func (c *Character) Facing() Facing           { return c.facing }
func (c *Character) LastWalk() time.Time      { return c.lastWalk }
func (c *Character) Inventory() Inventory     { return c.inventory }
func (c *Character) Woodcutting() Skill       { return c.woodcutting }

// Face turns the character without moving it.
func (c *Character) Face(facing Facing) { c.facing = facing }

// AimAt faces a world position. Once aimed, walking no longer turns the character.
func (c *Character) AimAt(x, z float64) {
	dx, dz := x-c.x, z-c.z
	if dx == 0 && dz == 0 {
		return
	}
	c.Face(FacingToward(dx, dz))
	c.aimed = true
}

// FacingToward picks the facing closest to a direction; ties favour up and down.
func FacingToward(dx, dz float64) Facing {
	if math.Abs(dx) > math.Abs(dz) {
		if dx < 0 {
			return FacingLeft
		}
		return FacingRight
	}
	if dz < 0 {
		return FacingUp
	}
	return FacingDown
}

// walk takes one counted step, as triggered by a single key press.
func (c *Character) walk(dx, dz, distance float64) {
	c.move(dx, dz, distance)
	c.steps++
}

// advance moves continuously and counts a step for every stride covered.
func (c *Character) advance(dx, dz, distance float64) {
	c.stepDistance += c.move(dx, dz, distance)
	for c.stepDistance >= stride {
		c.steps++
		c.stepDistance -= stride
	}
}

func (c *Character) move(dx, dz, distance float64) float64 {
	length := math.Hypot(dx, dz)
	if length == 0 {
		return 0
	}
	dx, dz = dx/length, dz/length
	if !c.aimed {
		c.Face(FacingToward(dx, dz))
	}
	c.lastWalk = time.Now()
	c.x, c.z = c.x+dx*distance, c.z+dz*distance
	return distance
}
