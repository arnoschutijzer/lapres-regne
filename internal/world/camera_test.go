package world_test

import (
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
