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

// World coordinates are sampled directly: exploration never allocates a growing map.
type World struct {
	seed                   uint64
	x, z, cameraX, cameraZ float64
	steps                  int
	stepDistance           float64
	facing                 Facing
	lastWalk               time.Time
}

func New(seed uint64) *World {
	return &World{seed: seed, facing: FacingDown}
}

func (w *World) Seed() uint64             { return w.seed }
func (w *World) Position() (x, z float64) { return w.x, w.z }
func (w *World) Camera() (x, z float64)   { return w.cameraX, w.cameraZ }
func (w *World) Steps() int               { return w.steps }
func (w *World) Facing() Facing           { return w.facing }
func (w *World) LastWalk() time.Time      { return w.lastWalk }

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

// Walk takes one step in a direction, as triggered by a single key press.
func (w *World) Walk(dx, dz float64, sprinting bool) {
	if dx == 0 && dz == 0 {
		return
	}
	distance := 0.55
	if sprinting {
		distance *= 2
	}
	w.move(dx, dz, distance)
	w.steps++
}

// Advance by elapsed time, independently of terminal key-repeat timing.
func (w *World) Advance(dx, dz float64, dt time.Duration, sprinting bool) {
	speed := 9.0
	if sprinting {
		speed *= 2
	}
	w.stepDistance += w.move(dx, dz, speed*dt.Seconds())
	for w.stepDistance >= 0.55 {
		w.steps++
		w.stepDistance -= 0.55
	}
}

func (w *World) move(dx, dz, distance float64) float64 {
	length := math.Hypot(dx, dz)
	if length == 0 {
		return 0
	}
	dx, dz = dx/length, dz/length
	w.facing = FacingDown
	if math.Abs(dx) > math.Abs(dz) {
		w.facing = FacingRight
		if dx < 0 {
			w.facing = FacingLeft
		}
	} else if dz < 0 {
		w.facing = FacingUp
	}
	w.lastWalk = time.Now()
	riverDistance, width := w.River(w.x, w.z)
	if riverDistance < width {
		distance *= 0.5
	}
	w.x, w.z = w.x+dx*distance, w.z+dz*distance
	return distance
}

func (w *World) Follow(dt time.Duration) {
	blend := 1 - math.Exp(-32*dt.Seconds())
	w.cameraX += (w.x - w.cameraX) * blend
	w.cameraZ += (w.z - w.cameraZ) * blend
}
