package terminal

import (
	"math"
	"testing"

	"lapresregne/internal/world"
)

const frame = 1.0 / 60

// walk moves a gait along a direction for a number of frames at walking speed.
func walk(g *gait, x, z *float64, dx, dz float64, frames int) {
	length := math.Hypot(dx, dz)
	for range frames {
		*x += dx / length * 4.5 * frame
		*z += dz / length * 4.5 * frame
		g.update(*x, *z, world.FacingDown, frame)
	}
}

func TestWalkingDiagonallyTurnsTheBodyDiagonally(t *testing.T) {
	var g gait
	x, z := 0.0, 0.0
	walk(&g, &x, &z, 1, 1, 60)
	if want := math.Pi / 4; math.Abs(g.heading-want) > 0.01 {
		t.Fatalf("heading = %.2f walking down and right, want %.2f", g.heading, want)
	}
}

func TestStandingStillTheBodyTurnsToItsFacing(t *testing.T) {
	var g gait
	for range 60 {
		g.update(0, 0, world.FacingRight, frame)
	}
	if want := math.Pi / 2; math.Abs(g.heading-want) > 0.01 {
		t.Fatalf("heading = %.2f facing right, want %.2f", g.heading, want)
	}
}

func TestTheBodyTurnsGraduallyTheShortWayRound(t *testing.T) {
	var g gait
	x, z := 0.0, 0.0
	walk(&g, &x, &z, 1, 0, 60)
	walk(&g, &x, &z, 0, -1, 1)
	if g.heading <= math.Pi/2 || g.heading >= math.Pi {
		t.Fatalf("heading = %.2f a frame after turning from right to up, want it partway between", g.heading)
	}
	walk(&g, &x, &z, -1, 0, 60)
	if want := -math.Pi / 2; math.Abs(angleBetween(g.heading, want)) > 0.01 {
		t.Fatalf("heading = %.2f after turning on to the left, want %.2f", g.heading, want)
	}
}

func TestStandingStillTheLegsRest(t *testing.T) {
	var g gait
	for range 120 {
		g.update(0, 0, world.FacingDown, frame)
	}
	if g.phase != 0 || g.amount != 0 {
		t.Fatalf("phase = %.2f, amount = %.2f standing still, want the legs at rest", g.phase, g.amount)
	}
}

func TestTheStrideEasesInThenFollowsTheGroundCovered(t *testing.T) {
	var g gait
	x, z := 0.0, 0.0
	g.update(x, z, world.FacingDown, frame)
	walk(&g, &x, &z, 1, 0, 1)
	if g.amount <= 0 || g.amount >= 0.5 {
		t.Fatalf("stride amount = %.2f a frame into walking, want it just starting", g.amount)
	}
	walk(&g, &x, &z, 1, 0, 59)
	if g.amount < 0.95 {
		t.Fatalf("stride amount = %.2f after a second of walking, want a full stride", g.amount)
	}
	before := g.phase
	walk(&g, &x, &z, 1, 0, 30)
	covered := 30 * 4.5 * frame
	if want := before + covered/g.cycle()*2*math.Pi; math.Abs(angleBetween(g.phase, want)) > 0.05 {
		t.Fatalf("phase = %.2f after covering %.1f, want %.2f, one cycle per %.1f covered", g.phase, covered, want, g.cycle())
	}
}

func TestStoppingEasesTheStrideOut(t *testing.T) {
	var g gait
	x, z := 0.0, 0.0
	g.update(x, z, world.FacingDown, frame)
	walk(&g, &x, &z, 1, 0, 60)
	g.update(x, z, world.FacingDown, frame)
	if g.amount < 0.5 {
		t.Fatalf("stride amount = %.2f a frame after stopping, want it settling gradually", g.amount)
	}
	for range 60 {
		g.update(x, z, world.FacingDown, frame)
	}
	if g.amount > 0.02 {
		t.Fatalf("stride amount = %.2f a second after stopping, want the legs at rest", g.amount)
	}
}
