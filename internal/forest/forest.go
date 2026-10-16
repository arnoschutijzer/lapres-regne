package forest

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
type Forest struct {
	seed                   uint64
	x, z, cameraX, cameraZ float64
	steps                  int
	stepDistance           float64
	facing                 Facing
	lastWalk               time.Time
}

func New(seed uint64) *Forest {
	return &Forest{seed: seed, facing: FacingDown}
}

func (f *Forest) Seed() uint64             { return f.seed }
func (f *Forest) Position() (x, z float64) { return f.x, f.z }
func (f *Forest) Camera() (x, z float64)   { return f.cameraX, f.cameraZ }
func (f *Forest) Steps() int               { return f.steps }
func (f *Forest) Facing() Facing           { return f.facing }
func (f *Forest) LastWalk() time.Time      { return f.lastWalk }

func (f *Forest) Random(x, z int, salt uint64) float64 {
	n := uint64(int64(x))*0x9e3779b185ebca87 ^ uint64(int64(z))*0xc2b2ae3d27d4eb4f ^ f.seed ^ salt
	n ^= n >> 30
	n *= 0xbf58476d1ce4e5b9
	n ^= n >> 27
	n *= 0x94d049bb133111eb
	n ^= n >> 31
	return float64(n>>11) / (1 << 53)
}

func (f *Forest) Noise(x, z float64, salt uint64) float64 {
	ix, iz := int(math.Floor(x)), int(math.Floor(z))
	u, v := x-float64(ix), z-float64(iz)
	u, v = u*u*(3-2*u), v*v*(3-2*v)
	a := f.Random(ix, iz, salt)*(1-u) + f.Random(ix+1, iz, salt)*u
	b := f.Random(ix, iz+1, salt)*(1-u) + f.Random(ix+1, iz+1, salt)*u
	return a*(1-v) + b*v
}

// Walk takes one step in a direction, as triggered by a single key press.
func (f *Forest) Walk(dx, dz float64, sprinting bool) {
	if dx == 0 && dz == 0 {
		return
	}
	distance := 0.55
	if sprinting {
		distance *= 2
	}
	f.move(dx, dz, distance)
	f.steps++
}

// Advance by elapsed time, independently of terminal key-repeat timing.
func (f *Forest) Advance(dx, dz float64, dt time.Duration, sprinting bool) {
	speed := 9.0
	if sprinting {
		speed *= 2
	}
	f.stepDistance += f.move(dx, dz, speed*dt.Seconds())
	for f.stepDistance >= 0.55 {
		f.steps++
		f.stepDistance -= 0.55
	}
}

func (f *Forest) move(dx, dz, distance float64) float64 {
	length := math.Hypot(dx, dz)
	if length == 0 {
		return 0
	}
	dx, dz = dx/length, dz/length
	f.facing = FacingDown
	if math.Abs(dx) > math.Abs(dz) {
		f.facing = FacingRight
		if dx < 0 {
			f.facing = FacingLeft
		}
	} else if dz < 0 {
		f.facing = FacingUp
	}
	f.lastWalk = time.Now()
	riverDistance, width := f.River(f.x, f.z)
	if riverDistance < width {
		distance *= 0.5
	}
	f.x, f.z = f.x+dx*distance, f.z+dz*distance
	return distance
}

func (f *Forest) Follow(dt time.Duration) {
	blend := 1 - math.Exp(-32*dt.Seconds())
	f.cameraX += (f.x - f.cameraX) * blend
	f.cameraZ += (f.z - f.cameraZ) * blend
}
