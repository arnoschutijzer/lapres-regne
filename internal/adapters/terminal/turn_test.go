package terminal

import (
	"math"
	"testing"
	"time"

	"lapresregne/internal/world"
)

func TestTheViewEasesTowardTheCameraYaw(t *testing.T) {
	var turn turning
	turn.step(math.Pi/2, 50*time.Millisecond)
	if turn.current <= 0 || turn.current >= math.Pi/2 {
		t.Fatalf("yaw = %v after a moment, want it partway to π/2", turn.current)
	}
	turn.step(math.Pi/2, 2*time.Second)
	if turn.current != math.Pi/2 {
		t.Fatalf("yaw = %v once settled, want π/2", turn.current)
	}
}

func TestAResumedViewOpensAtTheCameraYaw(t *testing.T) {
	camera := cameraAt(0, quarterTurn)
	if got := newRenderer(world.New(7), camera, "L’Après-Règne").turn.current; got != camera.Yaw() {
		t.Fatalf("yaw = %v on the first frame, want the camera's %v without easing in", got, camera.Yaw())
	}
}

func TestGroundAtFollowsTheTurnedView(t *testing.T) {
	w := world.New(7)
	r := newRenderer(w, cameraAt(0, quarterTurn), "L’Après-Règne")
	r.frame(100, 30, 0)
	// A quarter turn shows the world's north on the screen's left and its east up the screen. The
	// hills shift the ground a little nearer or farther than on flat ground.
	x, z, ok := r.groundAt(10, 17)
	if !ok || math.Abs(x) > -z/8 || z >= 0 {
		t.Fatalf("groundAt(10, 17) = %v, %v, %v; want a point north of the hero", x, z, ok)
	}
	x, z, ok = r.groundAt(51, 3)
	if !ok || x <= 0 || math.Abs(z) > 0.5 {
		t.Fatalf("groundAt(51, 3) = %v, %v, %v; want a point east of the hero", x, z, ok)
	}
}

func TestTheSunStaysPutAsTheCameraTurns(t *testing.T) {
	c := &forestCanvas{width: 120, height: 120, scale: 10}
	c.turnTo(math.Pi / 2)
	c.tiltTo(tilt)
	c.pixels = make([]forestPixel, c.width*c.height)
	for i := range c.pixels {
		c.pixels[i].depth = math.Inf(1)
	}
	var centre vec
	sx, sy, _ := c.project(0, 0, 0)
	ellipsoid(c, 0, 0, 0, 1, 1, 1, 0, func(px, py int, _ float64, _, normal vec) {
		if px == int(sx) && py == int(sy) {
			centre = normal
		}
	})
	// Turned a quarter, the camera looks at the sphere from the west, so the facing surface points west.
	if centre.x > -0.5 || math.Abs(centre.z) > 0.1 {
		t.Fatalf("the normal facing the camera is %+v in the world, want it pointing west", centre)
	}
}

func TestTheHeroShowsTheirFaceToTheTurnedCamera(t *testing.T) {
	w := world.New(7)
	w.Hero().Face(world.FacingLeft)
	if countInk(closeUp(w, quarterTurn), eyeInk) == 0 {
		t.Error("a hero facing the turned camera must show their eyes")
	}
	w.Hero().Face(world.FacingRight)
	if countInk(closeUp(w, quarterTurn), eyeInk) != 0 {
		t.Error("a hero facing away from the turned camera must show only the back of their head")
	}
}
