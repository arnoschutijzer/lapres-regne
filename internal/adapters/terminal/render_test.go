package terminal

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

// besideTree stands the hero facing the seed 7 tree at x 6.0, z 1.7 with the camera on the hero.
func besideTree() *world.World {
	w := world.New(7)
	for range 3 {
		w.Walk(0, 1, false)
	}
	for range 9 {
		w.Walk(1, 0, false)
	}
	w.Follow(time.Second)
	return w
}

func countInk(c *forestCanvas, inks ...int) int {
	count := 0
	for _, p := range c.pixels {
		for _, ink := range inks {
			if p.ink == ink {
				count++
			}
		}
	}
	return count
}

func TestAFelledTreeDrawsAStumpWithoutACrown(t *testing.T) {
	w := besideTree()
	r := newRenderer(w, "L’Après-Règne")
	leaves := countInk(r.render(100, 27, 0), 2, 3)
	w.Chop()
	w.Advance(0, 0, time.Second, false)
	if felled := countInk(r.render(100, 27, 0), 2, 3); felled >= leaves {
		t.Fatalf("leaf pixels = %d after felling, want fewer than %d", felled, leaves)
	}
	tree, _ := w.TreeAt(1, 0)
	_, top, _ := r.canvas.project(tree.X, stumpHeight*0.8, tree.Z)
	x, base, _ := r.canvas.project(tree.X, 0, tree.Z)
	stump := 0
	for y := int(top); y <= int(base); y++ {
		if r.canvas.pixels[y*r.canvas.width+int(x)].ink == 4 {
			stump++
		}
	}
	if stump == 0 {
		t.Fatal("a felled tree must leave a bark-colored stump")
	}
}

func TestChoppingSwingsAnAxe(t *testing.T) {
	w := besideTree()
	r := newRenderer(w, "L’Après-Règne")
	w.Chop()
	raised := countInk(r.render(100, 27, 0), 9)
	if raised == 0 {
		t.Fatal("chopping must draw an axe head")
	}
	before := append([]forestPixel(nil), r.canvas.pixels...)
	r.render(100, 27, 0.2)
	moved := false
	for i, p := range r.canvas.pixels {
		if (p.ink == 9) != (before[i].ink == 9) {
			moved = true
			break
		}
	}
	if !moved {
		t.Fatal("the axe must swing while chopping")
	}
}

func TestTheActionMenuAndLogsShowOnScreen(t *testing.T) {
	w := besideTree()
	r := newRenderer(w, "L’Après-Règne")
	if frame := r.frame(100, 30, 0); !strings.Contains(frame, "logs 0") || strings.Contains(frame, "Chop down") {
		t.Fatal("the status line must count logs and the menu must start closed")
	}
	r.menu = []string{"Chop down"}
	if frame := r.frame(100, 30, 0); !strings.Contains(frame, "1 Chop down") {
		t.Fatal("an open menu must list numbered actions")
	}
	r.menu = nil
	w.Chop()
	if frame := r.frame(100, 30, 0); !strings.Contains(frame, "chopping") || !strings.Contains(frame, "arrows/WASD") {
		t.Fatal("chopping must show in the status line and a closed menu must restore the controls")
	}
}
