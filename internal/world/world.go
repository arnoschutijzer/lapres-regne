package world

import (
	"math"
	"time"
)

// World coordinates are sampled directly: exploration never allocates a growing map.
type World struct {
	seed        uint64
	hero        Character
	woodcutting woodcutting
	// order is the player order the hero carries out; nil while they stand idle.
	order order
	// offeredClimb marks a staircase or ladder reached that leads both ways, until the offer is
	// taken.
	offeredClimb bool
	// ground holds the dropped items until they disappear.
	ground      []GroundItem
	message     string
	celebration celebration
	sounds      Sounds
	// terrain holds the map the world is built on; nil for the endless forest. floors holds the
	// map's floors above its ground floor, and storey is the floor the hero stands on.
	terrain *terrain
	floors  []*terrain
	storey  int
	mapName string
}

func New(seed uint64) *World {
	w := &World{seed: seed, sounds: silence{}}
	w.woodcutting = woodcutting{woods: w}
	return w
}

// SetSounds plays the world's sound effects through sounds.
func (w *World) SetSounds(sounds Sounds) { w.sounds = sounds }

func (w *World) effects() Sounds { return w.sounds }

// Message is the game's reply to the hero's latest action, or empty when there is none.
func (w *World) Message() string { return w.message }

func (w *World) say(message string) { w.message = message }

func (w *World) Seed() uint64     { return w.seed }
func (w *World) Hero() *Character { return &w.hero }

func (w *World) Random(x, z int, salt uint64) float64 {
	n := uint64(int64(x))*0x9e3779b185ebca87 ^ uint64(int64(z))*0xc2b2ae3d27d4eb4f ^ w.seed ^ salt
	n ^= n >> 30
	n *= 0xbf58476d1ce4e5b9
	n ^= n >> 27
	n *= 0x94d049bb133111eb
	n ^= n >> 31
	return float64(n>>11) / (1 << 53)
}

func (w *World) Noise(x, z float64, salt uint64) float64 {
	ix, iz := int(math.Floor(x)), int(math.Floor(z))
	u, v := x-float64(ix), z-float64(iz)
	u, v = u*u*(3-2*u), v*v*(3-2*v)
	a := w.Random(ix, iz, salt)*(1-u) + w.Random(ix+1, iz, salt)*u
	b := w.Random(ix, iz+1, salt)*(1-u) + w.Random(ix+1, iz+1, salt)*u
	return a*(1-v) + b*v
}

// walkingSpeed is how far the hero walks per second.
const walkingSpeed = 4.5

// Walk takes one step in a direction in the world, as triggered by a single key press.
func (w *World) Walk(dx, dz float64, sprinting bool) {
	if dx == 0 && dz == 0 {
		return
	}
	w.Stop()
	distance := stride
	if sprinting {
		distance *= 2
	}
	w.hero.walk(dx, dz, w.wading(distance))
}

// Advance by elapsed time, independently of terminal key-repeat timing, walking in a direction in
// the world.
func (w *World) Advance(dx, dz float64, dt time.Duration, sprinting bool) {
	if dx != 0 || dz != 0 {
		w.Stop()
	}
	speed := walkingSpeed
	if sprinting {
		speed *= 2
	}
	w.hero.advance(dx, dz, w.wading(speed*dt.Seconds()))
	w.ageCelebration(dt)
	chopping, _ := w.chopping()
	w.woodcutting.advance(dt, chopping)
	w.despawn(dt)
	w.carryOutOrder(dt, speed)
}

// WalkTo walks the hero to a point on the ground, running while sprinting, giving up whatever they
// were doing. On a map they go round whatever blocks the way, and stop as near as they can get to
// a point they cannot reach.
func (w *World) WalkTo(x, z float64) {
	w.walkTo(x, z, nil)
}

// walkTo walks the hero to a point on the ground and acts on arrival.
func (w *World) walkTo(x, z float64, arrive func() order) {
	w.Stop()
	w.order = &walk{world: w, waypoints: w.route(x, z), arrive: arrive}
}

// wading halves the distance while the hero stands in a river.
func (w *World) wading(distance float64) float64 {
	riverDistance, width := w.River(w.hero.x, w.hero.z)
	if riverDistance < width {
		return distance * 0.5
	}
	return distance
}
