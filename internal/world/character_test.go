package world_test

import (
	"math"
	"testing"
	"time"

	"lapresregne/internal/world"
)

func TestWalkingFacesTheWalkingDirectionUntilAimed(t *testing.T) {
	w := world.New(7)
	w.Walk(-1, 0, false)
	if facing := w.Hero().Facing(); facing != world.FacingLeft {
		t.Fatalf("facing = %v, want left after walking left", facing)
	}
	w.Walk(0, -1, false)
	if facing := w.Hero().Facing(); facing != world.FacingUp {
		t.Fatalf("facing = %v, want up after walking up", facing)
	}
}

func TestAimingFacesTheTargetInFourDirections(t *testing.T) {
	cases := []struct {
		name string
		x, z float64
		want world.Facing
	}{
		{"left", -3, 1, world.FacingLeft},
		{"right", 3, -1, world.FacingRight},
		{"up", 1, -3, world.FacingUp},
		{"down", -1, 3, world.FacingDown},
		{"diagonal tie", 2, -2, world.FacingUp},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := world.New(7)
			w.Hero().AimAt(c.x, c.z)
			if facing := w.Hero().Facing(); facing != c.want {
				t.Fatalf("facing = %v, want %v", facing, c.want)
			}
		})
	}
}

func TestAimingWinsOverWalking(t *testing.T) {
	w := world.New(7)
	w.Hero().AimAt(-5, 0)
	w.Walk(1, 0, false)
	w.Advance(1, 0, 0, false)
	if facing := w.Hero().Facing(); facing != world.FacingLeft {
		t.Fatalf("facing = %v, want left while walking right after aiming left", facing)
	}
	if x, _ := w.Hero().Position(); x <= 0 {
		t.Fatalf("x = %v, aiming must not stop the hero from walking", x)
	}
}

func TestAimingAtTheHeroKeepsTheLastFacing(t *testing.T) {
	w := world.New(7)
	w.Walk(1, 0, false)
	x, z := w.Hero().Position()
	w.Hero().AimAt(x, z)
	if facing := w.Hero().Facing(); facing != world.FacingRight {
		t.Fatalf("facing = %v, want right to be kept", facing)
	}
}

func TestTheHeroWalksAtAnUnhurriedPaceAndSprintsTwiceAsFast(t *testing.T) {
	w := world.New(7)
	w.Advance(1, 0, time.Second, false)
	if x, _ := w.Hero().Position(); math.Abs(x-4.5) > 1e-9 {
		t.Fatalf("x = %v after walking a second, want 4.5", x)
	}
	w.Advance(1, 0, time.Second, true)
	if x, _ := w.Hero().Position(); math.Abs(x-13.5) > 1e-9 {
		t.Fatalf("x = %v after sprinting a second more, want 13.5", x)
	}
}

func TestWalkingToAPointCrossesTheGroundAtWalkingPaceAndStopsThere(t *testing.T) {
	w := world.New(7)
	w.WalkTo(3, 4)
	w.Advance(0, 0, 500*time.Millisecond, false)
	if x, z := w.Hero().Position(); math.Abs(x-1.35) > 1e-9 || math.Abs(z-1.8) > 1e-9 {
		t.Fatalf("hero at (%v, %v) after half a second, want 2.25 units along the way at (1.35, 1.8)", x, z)
	}
	w.Advance(0, 0, 2*time.Second, false)
	w.Advance(0, 0, time.Second, false)
	if x, z := w.Hero().Position(); math.Abs(x-3) > 1e-9 || math.Abs(z-4) > 1e-9 {
		t.Fatalf("hero at (%v, %v), want them stopped on the point (3, 4)", x, z)
	}
}

func TestMovingCancelsWalkingToAPoint(t *testing.T) {
	cases := []struct {
		name string
		move func(w *world.World)
	}{
		{"key press", func(w *world.World) { w.Walk(0, 1, false) }},
		{"held key", func(w *world.World) { w.Advance(0, 1, 10*time.Millisecond, false) }},
		{"stop", func(w *world.World) { w.Stop() }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := world.New(7)
			w.WalkTo(-10, 0)
			c.move(w)
			x, _ := w.Hero().Position()
			w.Advance(0, 0, time.Second, false)
			if after, _ := w.Hero().Position(); after != x {
				t.Fatalf("x = %v, then %v; want the walk to the point cancelled", x, after)
			}
		})
	}
}

func TestApproachingATreeCancelsWalkingToAPoint(t *testing.T) {
	w := world.New(7)
	w.WalkTo(-10, 0)
	tree, _ := w.TreeAt(1, 0)
	w.Approach(tree)
	for range 50 {
		w.Advance(0, 0, 100*time.Millisecond, false)
	}
	if !w.Chopping() {
		t.Fatal("the hero must reach the tree and chop it")
	}
}
