package world_test

import (
	"math"
	"testing"
	"time"

	"lapresregne/internal/world"
)

func TestTheCameraZoomsBetweenTheForestAndACloseUp(t *testing.T) {
	w := world.New(7)
	if w.Zoom() != 0 {
		t.Fatalf("zoom = %d in a new world, want the forest view", w.Zoom())
	}
	w.ZoomBy(3)
	if w.Zoom() != 3 {
		t.Fatalf("zoom = %d after three steps in, want 3", w.Zoom())
	}
	w.ZoomBy(100)
	if w.Zoom() != world.CloseUp {
		t.Fatalf("zoom = %d after scrolling far in, want the close-up %d", w.Zoom(), world.CloseUp)
	}
	w.ZoomBy(-100)
	if w.Zoom() != 0 {
		t.Fatalf("zoom = %d after scrolling far out, want the forest view", w.Zoom())
	}
}

func TestTheZoomIsKeptAndResumedInAnyWorld(t *testing.T) {
	kept := world.New(7)
	kept.ZoomBy(5)
	progress := &memoryProgress{}
	if err := kept.Keep(progress, quitAt); err != nil {
		t.Fatal(err)
	}
	for _, seed := range []uint64{7, 8} {
		w, err := world.Resume(seed, progress, quitAt.Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if w.Zoom() != 5 {
			t.Fatalf("zoom = %d resuming in world %d, want the kept 5", w.Zoom(), seed)
		}
	}
}

func settle(w *world.World) {
	for range 100 {
		w.Follow(50 * time.Millisecond)
	}
}

func TestPanningMovesTheCameraAwayFromTheHero(t *testing.T) {
	w := world.New(7)
	w.PanBy(-3, 2)
	settle(w)
	heroX, heroZ := w.Hero().Position()
	x, z := w.Camera()
	if math.Abs(x-(heroX-3)) > 1e-6 || math.Abs(z-(heroZ+2)) > 1e-6 {
		t.Fatalf("camera = (%v, %v), want 3 units left of and 2 below the hero at (%v, %v)", x, z, heroX, heroZ)
	}
	w.PanBy(5, -6)
	settle(w)
	if x, z := w.Camera(); math.Abs(x-(heroX+2)) > 1e-6 || math.Abs(z-(heroZ-4)) > 1e-6 {
		t.Fatalf("camera = (%v, %v), want 2 units right of and 4 above the hero at (%v, %v)", x, z, heroX, heroZ)
	}
}

func TestWalkingBringsThePannedCameraBackToTheHero(t *testing.T) {
	w := world.New(7)
	w.PanBy(4, 3)
	w.Advance(1, 0, 100*time.Millisecond, false)
	settle(w)
	heroX, heroZ := w.Hero().Position()
	if x, z := w.Camera(); math.Abs(x-heroX) > 1e-6 || math.Abs(z-heroZ) > 1e-6 {
		t.Fatalf("camera = (%v, %v) after walking, want it back on the hero at (%v, %v)", x, z, heroX, heroZ)
	}
}

func TestTheCameraTurnsInStepsAroundTheView(t *testing.T) {
	w := world.New(7)
	if w.Yaw() != 0 {
		t.Fatalf("yaw = %v in a new world, want the camera facing north", w.Yaw())
	}
	w.TurnBy(world.QuarterTurn)
	if math.Abs(w.Yaw()-math.Pi/2) > 1e-9 {
		t.Fatalf("yaw = %v after a quarter turn of steps, want π/2", w.Yaw())
	}
	w.TurnBy(-2 * world.QuarterTurn)
	if math.Abs(w.Yaw()+math.Pi/2) > 1e-9 {
		t.Fatalf("yaw = %v after turning back half a turn, want -π/2", w.Yaw())
	}
}

func TestWalkingFollowsTheTurnedView(t *testing.T) {
	w := world.New(7)
	w.TurnBy(world.QuarterTurn)
	// A quarter turn shows the world's south on the screen's right.
	w.Advance(1, 0, time.Second, false)
	if x, z := w.Hero().Position(); math.Abs(x) > 1e-9 || z <= 0 {
		t.Fatalf("hero at (%v, %v) after walking right, want them south of the start", x, z)
	}
}

func TestPanningFollowsTheTurnedView(t *testing.T) {
	w := world.New(7)
	w.TurnBy(world.QuarterTurn)
	w.PanBy(0, -2)
	settle(w)
	if x, z := w.Camera(); math.Abs(x-2) > 1e-6 || math.Abs(z) > 1e-6 {
		t.Fatalf("camera = (%v, %v) after panning up, want 2 units east of the hero", x, z)
	}
}
