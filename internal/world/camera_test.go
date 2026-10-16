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

func TestPanningMovesTheCameraSidewaysFromTheHero(t *testing.T) {
	w := world.New(7)
	w.PanBy(-3)
	settle(w)
	heroX, heroZ := w.Hero().Position()
	x, z := w.Camera()
	if math.Abs(x-(heroX-3)) > 1e-6 || math.Abs(z-heroZ) > 1e-6 {
		t.Fatalf("camera = (%v, %v), want 3 units left of the hero at (%v, %v)", x, z, heroX, heroZ)
	}
	w.PanBy(5)
	settle(w)
	if x, _ := w.Camera(); math.Abs(x-(heroX+2)) > 1e-6 {
		t.Fatalf("camera x = %v, want 2 units right of the hero at %v", x, heroX)
	}
}

func TestWalkingBringsThePannedCameraBackToTheHero(t *testing.T) {
	w := world.New(7)
	w.PanBy(4)
	w.Advance(1, 0, 100*time.Millisecond, false)
	settle(w)
	heroX, _ := w.Hero().Position()
	if x, _ := w.Camera(); math.Abs(x-heroX) > 1e-6 {
		t.Fatalf("camera x = %v after walking, want it back on the hero at %v", x, heroX)
	}
}
