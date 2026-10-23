package terminal

import (
	"math"
	"testing"
	"time"

	"lapresregne/internal/world"
)

func settle(c *Camera, x, z float64) {
	for range 100 {
		c.follow(x, z, 50*time.Millisecond)
	}
}

func TestTheCameraZoomsBetweenTheForestAndACloseUp(t *testing.T) {
	c := NewCamera(0, 0)
	if c.Zoom() != 0 {
		t.Fatalf("zoom = %d for a new camera, want the forest view", c.Zoom())
	}
	c.zoomBy(3)
	if c.Zoom() != 3 {
		t.Fatalf("zoom = %d after three steps in, want 3", c.Zoom())
	}
	c.zoomBy(100)
	if c.Zoom() != closeUpZoom {
		t.Fatalf("zoom = %d after scrolling far in, want the close-up %d", c.Zoom(), closeUpZoom)
	}
	c.zoomBy(-100)
	if c.Zoom() != 0 {
		t.Fatalf("zoom = %d after scrolling far out, want the forest view", c.Zoom())
	}
}

func TestAKeptZoomOpensWithinTheCloseUp(t *testing.T) {
	if zoom := NewCamera(100, 0).Zoom(); zoom != closeUpZoom {
		t.Fatalf("zoom = %d opening at 100, want the close-up %d", zoom, closeUpZoom)
	}
}

func TestPanningMovesTheCameraAwayFromTheHero(t *testing.T) {
	c := NewCamera(0, 0)
	heroX, heroZ := 1.0, 2.0
	c.panBy(-3, 2)
	settle(c, heroX, heroZ)
	if math.Abs(c.x-(heroX-3)) > 1e-6 || math.Abs(c.z-(heroZ+2)) > 1e-6 {
		t.Fatalf("camera = (%v, %v), want 3 units left of and 2 below the hero at (%v, %v)", c.x, c.z, heroX, heroZ)
	}
	c.panBy(5, -6)
	settle(c, heroX, heroZ)
	if math.Abs(c.x-(heroX+2)) > 1e-6 || math.Abs(c.z-(heroZ-4)) > 1e-6 {
		t.Fatalf("camera = (%v, %v), want 2 units right of and 4 above the hero at (%v, %v)", c.x, c.z, heroX, heroZ)
	}
}

func TestWalkingBringsThePannedCameraBackToTheHero(t *testing.T) {
	c := NewCamera(0, 0)
	c.panBy(4, 3)
	c.walk(1, 0)
	settle(c, 1, 2)
	if math.Abs(c.x-1) > 1e-6 || math.Abs(c.z-2) > 1e-6 {
		t.Fatalf("camera = (%v, %v) after walking, want it back on the hero at (1, 2)", c.x, c.z)
	}
}

func TestStandingStillKeepsTheCameraPanned(t *testing.T) {
	c := NewCamera(0, 0)
	c.panBy(4, 3)
	c.walk(0, 0)
	settle(c, 0, 0)
	if math.Abs(c.x-4) > 1e-6 || math.Abs(c.z-3) > 1e-6 {
		t.Fatalf("camera = (%v, %v) with no key held, want it still panned to (4, 3)", c.x, c.z)
	}
}

func TestTheCameraTurnsInStepsAroundTheView(t *testing.T) {
	c := NewCamera(0, 0)
	if c.Yaw() != 0 {
		t.Fatalf("yaw = %v for a new camera, want it facing north", c.Yaw())
	}
	c.turnBy(quarterTurn)
	if math.Abs(c.Yaw()-math.Pi/2) > 1e-9 {
		t.Fatalf("yaw = %v after a quarter turn of steps, want π/2", c.Yaw())
	}
	c.turnBy(-2 * quarterTurn)
	if math.Abs(c.Yaw()+math.Pi/2) > 1e-9 {
		t.Fatalf("yaw = %v after turning back half a turn, want -π/2", c.Yaw())
	}
}

func TestWalkingFollowsTheTurnedView(t *testing.T) {
	c := NewCamera(0, 0)
	c.turnBy(quarterTurn)
	// A quarter turn shows the world's south on the screen's right.
	if x, z := c.walk(1, 0); math.Abs(x) > 1e-9 || z <= 0 {
		t.Fatalf("walking right heads (%v, %v), want south", x, z)
	}
}

func TestPanningFollowsTheTurnedView(t *testing.T) {
	c := NewCamera(0, 0)
	c.turnBy(quarterTurn)
	c.panBy(0, -2)
	settle(c, 0, 0)
	if math.Abs(c.x-2) > 1e-6 || math.Abs(c.z) > 1e-6 {
		t.Fatalf("camera = (%v, %v) after panning up, want 2 units east of the hero", c.x, c.z)
	}
}

func TestTheCameraTiltsWithinRuneScapesRange(t *testing.T) {
	c := NewCamera(0, 0)
	if c.Tilt() != openingTilt {
		t.Fatalf("tilt = %v for a new camera, want the opening %v", c.Tilt(), openingTilt)
	}
	c.tiltBy(tiltSteps * 2)
	if math.Abs(c.Tilt()-383*2*math.Pi/2048) > 1e-9 {
		t.Fatalf("tilt = %v raised all the way, want Old School RuneScape's highest, 383 of 2048 steps", c.Tilt())
	}
	c.tiltBy(-tiltSteps * 2)
	if math.Abs(c.Tilt()-128*2*math.Pi/2048) > 1e-9 {
		t.Fatalf("tilt = %v lowered all the way, want Old School RuneScape's lowest, 128 of 2048 steps", c.Tilt())
	}
	c.TiltTo(1)
	c.tiltBy(1)
	if c.Tilt() <= 1 {
		t.Fatalf("tilt = %v raised one step from 1, want it steeper", c.Tilt())
	}
}

func TestTheViewLooksDownAtTheCamerasTilt(t *testing.T) {
	camera := NewCamera(0, 0)
	camera.TiltTo(highestTilt)
	c := newRenderer(world.New(7), camera, "L’Après-Règne").render(100, 27, 0)
	if math.Abs(c.viewer().y-math.Sin(highestTilt)) > 1e-9 {
		t.Fatalf("the view looks down along %+v, want the camera's tilt %v", c.viewer(), highestTilt)
	}
}
