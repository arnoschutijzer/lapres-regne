package terminal

import (
	"math"
	"time"
)

// closeUpZoom is the zoom of the closest view of the hero.
const closeUpZoom = 12

// quarterTurn is how many of the camera's turning steps make a quarter turn.
const quarterTurn = 24

// The camera tilts as Old School RuneScape's does, from lowestTilt, 128 of the 2048 steps in a full
// turn above the horizon, up to highestTilt, 383 of them, in tiltSteps steps. It opens at
// openingTilt until the player tilts it.
const (
	lowestTilt  = 128 * 2 * math.Pi / 2048
	highestTilt = 383 * 2 * math.Pi / 2048
	openingTilt = 0.8067
	tiltSteps   = 12
)

// Camera is the view onto the world: it follows the hero, looks away from them while panned until
// they walk, turns about the upright axis and zooms between the forest view and a close-up.
type Camera struct {
	x, z float64
	// panX and panZ are how far the camera looks away from the hero, until the hero walks.
	panX, panZ float64
	// yaw is how far the camera has turned about the upright axis, in radians.
	yaw float64
	// tilt is how steeply the camera looks down on the ground, in radians above the horizon.
	tilt float64
	// zoom is how many steps the camera has moved in from the forest view toward a close-up.
	zoom int
}

// NewCamera opens the view at a zoom and yaw, as kept from an earlier session.
func NewCamera(zoom int, yaw float64) *Camera {
	c := &Camera{yaw: yaw, tilt: openingTilt}
	c.zoomBy(zoom)
	return c
}

// Zoom is how many steps the camera is in from the forest view, from 0 up to the close-up.
func (c *Camera) Zoom() int { return c.zoom }

// Yaw is how far the camera has turned about the upright axis, in radians. At 0 north is up the
// screen; at π/2 east is.
func (c *Camera) Yaw() float64 { return c.yaw }

// Tilt is how steeply the camera looks down on the ground, in radians above the horizon: 0 would look
// along it, π/2 straight down.
func (c *Camera) Tilt() float64 { return c.tilt }

// TiltTo tilts the camera to a tilt, as kept from an earlier session, kept within the range Old
// School RuneScape allows.
func (c *Camera) TiltTo(tilt float64) {
	c.tilt = min(highestTilt, max(lowestTilt, tilt))
}

// tiltBy raises the camera by steps, or lowers it for negative steps.
func (c *Camera) tiltBy(steps int) {
	c.TiltTo(c.tilt + float64(steps)*(highestTilt-lowestTilt)/tiltSteps)
}

// zoomBy moves the camera steps closer, or farther for negative steps, between the forest view
// and the close-up.
func (c *Camera) zoomBy(steps int) {
	c.zoom = min(closeUpZoom, max(0, c.zoom+steps))
}

// turnBy turns the camera by steps, each a small part of a quarter turn.
func (c *Camera) turnBy(steps int) {
	c.yaw += float64(steps) * math.Pi / 2 / quarterTurn
}

// panBy moves the camera away from the hero by dx, dz world units along the screen.
func (c *Camera) panBy(dx, dz float64) {
	dx, dz = c.fromView(dx, dz)
	c.panX += dx
	c.panZ += dz
}

// walk turns a walking direction on the screen into the world, bringing a panned camera back to
// the hero once they walk.
func (c *Camera) walk(dx, dz float64) (float64, float64) {
	if dx != 0 || dz != 0 {
		c.stopPanning()
	}
	return c.fromView(dx, dz)
}

// stopPanning brings a panned camera back to the hero.
func (c *Camera) stopPanning() { c.panX, c.panZ = 0, 0 }

// fromView turns a direction on the screen, with dz pointing down it, into the world.
func (c *Camera) fromView(dx, dz float64) (float64, float64) {
	sin, cos := math.Sincos(c.yaw)
	return dx*cos - dz*sin, dx*sin + dz*cos
}

// lookAt puts the camera straight onto a point, without easing toward it.
func (c *Camera) lookAt(x, z float64) { c.x, c.z = x, z }

// follow eases the camera toward the hero at x, z, or as far from them as it is panned.
func (c *Camera) follow(x, z float64, dt time.Duration) {
	blend := 1 - math.Exp(-32*dt.Seconds())
	c.x += (x + c.panX - c.x) * blend
	c.z += (z + c.panZ - c.z) * blend
}

// shownCamera is where the view is drawn from: the camera's place, but whole canvas pixels across
// and rows down from where it eases toward, so the view holds still while the camera eases the
// last fractions of a pixel onto the hero.
func (r *renderer) shownCamera(c *forestCanvas) (float64, float64) {
	x, z := r.world.Hero().Position()
	x, z = x+r.camera.panX, z+r.camera.panZ
	sin, cos := c.yawSin, c.yawCos
	dx, dz := r.camera.x-x, r.camera.z-z
	acrossStep, towardStep := 1/c.scale, 1/(c.scale*c.tiltSin)
	across := math.Round((dx*cos+dz*sin)/acrossStep) * acrossStep
	toward := math.Round((-dx*sin+dz*cos)/towardStep) * towardStep
	return x + across*cos - toward*sin, z + across*sin + toward*cos
}

// follow eases the camera toward the hero.
func (r *renderer) follow(dt time.Duration) {
	x, z := r.world.Hero().Position()
	r.camera.follow(x, z, dt)
}
