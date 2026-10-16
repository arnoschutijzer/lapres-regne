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
		*x += dx / length * 9 * frame
		*z += dz / length * 9 * frame
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
