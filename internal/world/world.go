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
	// panX and panZ are how far the camera looks away from the hero, until the hero walks.
	panX, panZ float64
	// yaw is how far the camera has turned about the upright axis, in radians.
	yaw float64
	// zoom is how many steps the camera has moved in from the forest view toward a close-up.
	zoom int
	// felled holds the cells whose tree is a stump, with the time left until it regrows.
	felled map[cell]time.Duration
	// depletion is how long each tree that gives several logs has been worn down by chopping.
	depletion map[cell]time.Duration
	approach  approach
	chop      chop
	// destination is the point on the ground the hero walks to after a click.
	destination destination
	// doorApproach is the doorway whose door the hero opens or closes on reaching the destination.
	doorApproach doorApproach
	// ground holds the dropped items until they disappear.
	ground      []GroundItem
	message     string
	celebration celebration
	sounds      Sounds
	// terrain holds the map the world is built on; nil for the endless forest.
	terrain *terrain
	mapName string
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
const walkingSpeed = 4.5

// Walk takes one step in a direction on the screen, as triggered by a single key press.
func (w *World) Walk(dx, dz float64, sprinting bool) {
	if dx == 0 && dz == 0 {
		return
	}
	dx, dz = w.fromView(dx, dz)
	w.Stop()
	w.panX, w.panZ = 0, 0
	distance := stride
	if sprinting {
		distance *= 2
	}
	w.hero.walk(dx, dz, w.wading(distance))
}

// Advance by elapsed time, independently of terminal key-repeat timing, walking in a direction on
// the screen.
func (w *World) Advance(dx, dz float64, dt time.Duration, sprinting bool) {
	if dx != 0 || dz != 0 {
		w.Stop()
		w.panX, w.panZ = 0, 0
	}
	dx, dz = w.fromView(dx, dz)
	speed := walkingSpeed
	if sprinting {
		speed *= 2
	}
	w.hero.advance(dx, dz, w.wading(speed*dt.Seconds()))
	w.ageCelebration(dt)
	w.walkToFor(dt, speed)
	w.approachFor(dt, speed)
	w.regrow(dt)
	w.despawn(dt)
	w.recoverTimers(dt)
	w.chopFor(dt)
}

type destination struct {
	active bool
	// waypoints lead round whatever blocks the way, ending on the destination.
	waypoints []point
}

// WalkTo walks the hero to a point on the ground, running while sprinting, giving up whatever they
// were doing. On a map they go round whatever blocks the way, and stop as near as they can get to
// a point they cannot reach.
func (w *World) WalkTo(x, z float64) {
	w.Stop()
	w.panX, w.panZ = 0, 0
	w.destination = destination{active: true, waypoints: w.route(x, z)}
}

// walkToFor walks toward the destination at speed and stops on it.
func (w *World) walkToFor(dt time.Duration, speed float64) {
	if !w.destination.active {
		return
	}
	waypoints, moving := w.follow(w.destination.waypoints, w.wading(speed*dt.Seconds()))
	w.destination.waypoints = waypoints
	// A hero who cannot get any closer gives up on the destination.
	if len(waypoints) == 0 || !moving {
		w.destination = destination{}
		w.arriveAtDoor()
	}
}

// wading halves the distance while the hero stands in a river.
func (w *World) wading(distance float64) float64 {
	riverDistance, width := w.River(w.hero.x, w.hero.z)
	if riverDistance < width {
		return distance * 0.5
	}
	return distance
}

// CloseUp is the zoom of the closest view of the hero.
const CloseUp = 12

// Zoom is how many steps the camera is in from the forest view, from 0 up to CloseUp.
func (w *World) Zoom() int { return w.zoom }

// ZoomBy moves the camera steps closer, or farther for negative steps, between the forest view
// and the close-up.
func (w *World) ZoomBy(steps int) {
	w.zoom = min(CloseUp, max(0, w.zoom+steps))
}

// PanBy moves the camera away from the hero by dx, dz world units along the screen.
func (w *World) PanBy(dx, dz float64) {
	dx, dz = w.fromView(dx, dz)
	w.panX += dx
	w.panZ += dz
}

// QuarterTurn is how many of the camera's turning steps make a quarter turn.
const QuarterTurn = 24

// Yaw is how far the camera has turned about the upright axis, in radians. At 0 north is up the
// screen; at π/2 east is.
func (w *World) Yaw() float64 { return w.yaw }

// TurnBy turns the camera by steps, each a small part of a quarter turn.
func (w *World) TurnBy(steps int) {
	w.yaw += float64(steps) * math.Pi / 2 / QuarterTurn
}

// fromView turns a direction on the screen, with dz pointing down it, into the world.
func (w *World) fromView(dx, dz float64) (float64, float64) {
	sin, cos := math.Sincos(w.yaw)
	return dx*cos - dz*sin, dx*sin + dz*cos
}

func (w *World) Follow(dt time.Duration) {
	blend := 1 - math.Exp(-32*dt.Seconds())
	w.cameraX += (w.hero.x + w.panX - w.cameraX) * blend
	w.cameraZ += (w.hero.z + w.panZ - w.cameraZ) * blend
}
