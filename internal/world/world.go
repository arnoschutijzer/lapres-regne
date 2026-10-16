package world

import (
	"math"
	"time"
)

// World coordinates are sampled directly: exploration never allocates a growing map.
type World struct {
	seed             uint64
	hero             Character
	cameraX, cameraZ float64
	// felled holds the cells whose tree is a stump.
	felled   map[cell]bool
	approach approach
	chop     chop
	sounds   Sounds
}

func New(seed uint64) *World {
	return &World{seed: seed, sounds: silence{}}
}

// SetSounds plays the world's sound effects through sounds.
func (w *World) SetSounds(sounds Sounds) { w.sounds = sounds }

func (w *World) Seed() uint64           { return w.seed }
func (w *World) Hero() *Character       { return &w.hero }
func (w *World) Camera() (x, z float64) { return w.cameraX, w.cameraZ }

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
const walkingSpeed = 9.0

// Walk takes one step in a direction, as triggered by a single key press.
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

// Advance by elapsed time, independently of terminal key-repeat timing.
func (w *World) Advance(dx, dz float64, dt time.Duration, sprinting bool) {
	if dx != 0 || dz != 0 {
		w.Stop()
	}
	speed := walkingSpeed
	if sprinting {
		speed *= 2
	}
	w.hero.advance(dx, dz, w.wading(speed*dt.Seconds()))
	w.approachFor(dt)
	w.chopFor(dt)
}

// wading halves the distance while the hero stands in a river.
func (w *World) wading(distance float64) float64 {
	riverDistance, width := w.River(w.hero.x, w.hero.z)
	if riverDistance < width {
		return distance * 0.5
	}
	return distance
}

func (w *World) Follow(dt time.Duration) {
	blend := 1 - math.Exp(-32*dt.Seconds())
	w.cameraX += (w.hero.x - w.cameraX) * blend
	w.cameraZ += (w.hero.z - w.cameraZ) * blend
}
