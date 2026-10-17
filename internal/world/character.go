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
	// skills holds the experience gained in each skill by name; a skill never trained is absent.
	skills map[string]Skill
	// passable tells where the character may stand; nil lets them stand anywhere.
	passable func(x, z float64) bool
}

func (c *Character) Position() (x, z float64) { return c.x, c.z }
func (c *Character) Steps() int               { return c.steps }
func (c *Character) Facing() Facing           { return c.facing }
func (c *Character) LastWalk() time.Time      { return c.lastWalk }
func (c *Character) Inventory() Inventory     { return c.inventory }

// Skill is the experience gained in the skill of that name.
func (c *Character) Skill(name string) Skill { return c.skills[name] }

// train gains experience in the skill of that name and reports its level before and after.
func (c *Character) train(name string, xp float64) (from, to int) {
	if c.skills == nil {
		c.skills = make(map[string]Skill)
	}
	skill := c.skills[name]
	from = skill.Level()
	skill.gain(xp)
	c.skills[name] = skill
	return from, skill.Level()
}

// experience lists the experience gained in each skill trained, by name.
func (c *Character) experience() map[string]float64 {
	experience := make(map[string]float64, len(c.skills))
	for name, skill := range c.skills {
		experience[name] = skill.XP()
	}
	return experience
}

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

// advance moves continuously, counts a step for every stride covered and reports the distance
// covered.
func (c *Character) advance(dx, dz, distance float64) float64 {
	moved := c.move(dx, dz, distance)
	c.stepDistance += moved
	for c.stepDistance >= stride {
		c.steps++
		c.stepDistance -= stride
	}
	return moved
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
	if c.passable == nil {
		c.lastWalk = time.Now()
		c.x, c.z = c.x+dx*distance, c.z+dz*distance
		return distance
	}
	moved := 0.0
	for moved < distance {
		step := math.Min(collisionStep, distance-moved)
		covered := c.slide(dx, dz, step)
		// Sliding along a wall almost straight into it covers next to nothing, and would take
		// endless steps to cover the distance, so the hero stops against the wall instead.
		if covered < step*0.01 {
			break
		}
		moved += covered
	}
	if moved > 0 {
		c.lastWalk = time.Now()
	}
	return moved
}

// collisionStep is the longest distance moved at once where the character may be blocked, so they
// stop close against whatever blocks them.
const collisionStep = 0.05

// slide moves a short distance, sliding along whatever blocks the way, and reports the distance
// covered.
func (c *Character) slide(dx, dz, distance float64) float64 {
	x, z := c.x+dx*distance, c.z+dz*distance
	switch {
	case c.passable(x, z):
	case dx != 0 && c.passable(x, c.z):
		z = c.z
	case dz != 0 && c.passable(c.x, z):
		x = c.x
	default:
		return 0
	}
	covered := math.Hypot(x-c.x, z-c.z)
	c.x, c.z = x, z
	return covered
}
