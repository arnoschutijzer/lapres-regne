package world

import (
	"math"
	"testing"
	"time"
)

// openGround lets the hero walk anywhere at full pace.
type openGround struct{ hero Character }

func (g *openGround) Hero() *Character { return &g.hero }

// follow is never needed on open ground, where the hero heads straight for the target.
func (g *openGround) follow(waypoints []point, _ float64) ([]point, bool) { return waypoints, false }

func (g *openGround) wading(distance float64) float64 { return distance }

func TestClosingInStopsWithinReachAndActsThere(t *testing.T) {
	ground := &openGround{}
	acted := false
	var o order = &closeIn{mover: ground, target: point{3, 0}, within: 1, arrive: func() order { acted = true; return nil }}
	for o != nil && !acted {
		o = o.carryOut(100*time.Millisecond, walkingSpeed)
	}
	if x, z := ground.hero.Position(); !acted || math.Abs(x-2) > 1e-9 || z != 0 {
		t.Fatalf("hero at %.2f, %.2f, acted = %v; want them acting 1 short of the target", x, z, acted)
	}
}
