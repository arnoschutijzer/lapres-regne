package terminal

import (
	"fmt"
	"testing"

	"dither-cli/internal/world"
)

func BenchmarkForestFrame(b *testing.B) {
	for _, size := range [][2]int{{100, 30}, {180, 50}, {220, 90}} {
		b.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(b *testing.B) {
			r := newRenderer(world.New(7), "L’Après-Règne")
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				r.frame(size[0], size[1], float64(i%120)/30)
			}
		})
	}
}

func TestGroundAtFollowsTheScreenAroundTheHero(t *testing.T) {
	r := newRenderer(world.New(7), "L’Après-Règne")
	r.frame(100, 30, 0)
	// The hero stands at the canvas point (width/2, height*0.57): column 51, row 17 on screen.
	leftX, _, ok := r.groundAt(10, 17)
	if !ok || leftX >= 0 {
		t.Fatalf("groundAt(10, 17) = %v, %v; want a point left of the hero", leftX, ok)
	}
	_, aboveZ, ok := r.groundAt(51, 3)
	if !ok || aboveZ >= 0 {
		t.Fatalf("groundAt(51, 3) = %v, %v; want a point above the hero", aboveZ, ok)
	}
	if _, _, ok := r.groundAt(51, 30); ok {
		t.Fatal("the status line must not map to the ground")
	}
}
