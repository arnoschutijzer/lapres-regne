package terminal

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"lapresregne/internal/world"
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

// chopUntilFelled advances tick by tick until the chopped tree falls.
func chopUntilFelled(t *testing.T, w *world.World) {
	t.Helper()
	for range 200 {
		w.Advance(0, 0, 600*time.Millisecond, false)
		if !w.Chopping() {
			return
		}
	}
	t.Fatal("the tree did not fall within 200 ticks")
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
	chopUntilFelled(t, w)
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
	w.Advance(0, 0, 200*time.Millisecond, false)
	r.render(100, 27, 0)
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

func TestTheActionMenuShowsOnScreen(t *testing.T) {
	w := besideTree()
	r := newRenderer(w, "L’Après-Règne")
	if frame := r.frame(100, 30, 0); strings.Contains(frame, "Chop down") {
		t.Fatal("the menu must start closed")
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

// leavesMovedAcross reports whether leaf pixels differ before and after the world advances, ignoring the axe.
func leavesMovedAcross(r *renderer, w *world.World, elapsed time.Duration) bool {
	before := append([]forestPixel(nil), r.render(100, 27, 0).pixels...)
	w.Advance(0, 0, elapsed, false)
	after := r.render(100, 27, 0).pixels
	for i, p := range after {
		if p.ink == 9 || before[i].ink == 9 {
			continue
		}
		if (p.ink == 2 || p.ink == 3) != (before[i].ink == 2 || before[i].ink == 3) {
			return true
		}
	}
	return false
}

func TestTheChoppedTreeJoltsWhenTheAxeLandsThenSettles(t *testing.T) {
	w := besideTree()
	r := newRenderer(w, "L’Après-Règne")
	if leavesMovedAcross(r, w, 80*time.Millisecond) {
		t.Fatal("crowns must stand still while nobody chops")
	}
	w.Chop()
	w.Advance(0, 0, 20*time.Millisecond, false)
	if !leavesMovedAcross(r, w, 430*time.Millisecond) {
		t.Fatal("the chopped tree must jolt as the axe lands")
	}
	if leavesMovedAcross(r, w, 100*time.Millisecond) {
		t.Fatal("the chopped tree must settle between strokes")
	}
}

func TestWoodChipsFlyOnlyRightAfterAStrike(t *testing.T) {
	w := besideTree()
	r := newRenderer(w, "L’Après-Règne")
	w.Chop()
	w.Advance(0, 0, 80*time.Millisecond, false)
	if countInk(r.render(100, 27, 0), 10) == 0 {
		t.Fatal("wood chips must fly right after the axe lands")
	}
	w.Advance(0, 0, 420*time.Millisecond, false)
	if chips := countInk(r.render(100, 27, 0), 10); chips != 0 {
		t.Fatalf("chips = %d, want none once the stroke is raised again", chips)
	}
}

func TestTreeAtPicksTheTreeUnderAClick(t *testing.T) {
	r := newRenderer(world.New(7), "L’Après-Règne")
	r.frame(100, 30, 0)
	// The crown of the tree at x 6.0, z 1.7 fills column 61, row 15 seen from the spawn point.
	if tree, ok := r.treeAt(61, 15); !ok || tree.X < 6 || tree.X > 6.1 {
		t.Fatalf("treeAt(61, 15) = %+v, %v; want the tree at x 6.0", tree, ok)
	}
	if tree, ok := r.treeAt(10, 25); ok {
		t.Fatalf("treeAt(10, 25) = %+v; want open ground", tree)
	}
	if _, ok := r.treeAt(61, 30); ok {
		t.Fatal("the status line must not pick a tree")
	}
}

func TestTheWoodcuttingSkillShowsOnTheTitleRow(t *testing.T) {
	w := besideTree()
	r := newRenderer(w, "L’Après-Règne")
	if frame := r.frame(100, 30, 0); !strings.Contains(frame, "\x1b[1;") || !strings.Contains(frame, "woodcutting 1 · 0 xp") {
		t.Fatal("the title row must show the woodcutting level and xp")
	}
	w.Chop()
	chopUntilFelled(t, w)
	if frame := r.frame(100, 30, 0); !strings.Contains(frame, "woodcutting 1 · 25 xp") {
		t.Fatal("the title row must update with gained xp")
	}
}

// besideWillow stands the hero facing seed 7's riverside willow at x 11.0, z -3.6 with the camera on the hero.
func besideWillow() *world.World {
	w := world.New(7)
	for range 6 {
		w.Walk(0, -1, false)
	}
	for range 18 {
		w.Walk(1, 0, false)
	}
	w.Follow(time.Second)
	return w
}

// nearFirst moves the hero and camera beside the nearest tree of a kind.
func nearFirst(t *testing.T, kind world.TreeKind) *world.World {
	t.Helper()
	w := world.New(7)
	for radius := 0; radius < 40; radius++ {
		for gz := -radius; gz <= radius; gz++ {
			for gx := -radius; gx <= radius; gx++ {
				if tree, ok := w.TreeAt(gx, gz); ok && tree.Kind == kind {
					// Walking speed is 9 per second, so one long frame carries the hero there.
					distance := math.Hypot(tree.X-1, tree.Z)
					w.Advance(tree.X-1, tree.Z, time.Duration(distance/9*float64(time.Second)), false)
					w.Follow(time.Second)
					return w
				}
			}
		}
	}
	t.Fatalf("no %v within 40 cells", kind)
	return nil
}

func TestWillowsAndYewsHaveTheirOwnLeaves(t *testing.T) {
	willow := newRenderer(besideWillow(), "L’Après-Règne")
	if countInk(willow.render(100, 27, 0), 11, 12) == 0 {
		t.Fatal("a willow must draw pale willow leaves")
	}
	yew := newRenderer(nearFirst(t, world.Yew), "L’Après-Règne")
	if countInk(yew.render(100, 27, 0), 13, 14) == 0 {
		t.Fatal("a yew must draw dark yew needles")
	}
}

func TestTheMenuNamesTheTreeAndMessagesShowOnTheHintLine(t *testing.T) {
	w := besideWillow()
	r := newRenderer(w, "L’Après-Règne")
	r.menuTitle, r.menu = "willow", []string{"Chop down"}
	if frame := r.frame(100, 30, 0); !strings.Contains(frame, "willow:  1 Chop down") {
		t.Fatal("the menu must name the tree it acts on")
	}
	r.menuTitle, r.menu = "", nil
	w.Chop()
	if frame := r.frame(100, 30, 0); !strings.Contains(frame, "You need woodcutting level 30 to chop this willow.") {
		t.Fatal("the game's message must show on the hint line")
	}
}

func TestTheStatusLineNoLongerCountsLogs(t *testing.T) {
	r := newRenderer(besideTree(), "L’Après-Règne")
	if frame := r.frame(100, 30, 0); strings.Contains(frame, "logs") {
		t.Fatal("the status line must not mention logs")
	}
}

func TestTheInventoryPanelOpensWithTintedLogsAndClosesCleanly(t *testing.T) {
	w := besideTree()
	w.Chop()
	chopUntilFelled(t, w)
	r := newRenderer(w, "L’Après-Règne")
	if frame := r.frame(100, 30, 0); strings.Contains(frame, "inventory") {
		t.Fatal("the inventory panel must start closed")
	}
	r.inventoryOpen = true
	open := r.frame(100, 30, 0)
	if !strings.Contains(open, "inventory 1/28") || !strings.Contains(open, icon(world.Logs)) {
		t.Fatal("the open panel must count the slots and show the normal log in its tint")
	}
	if strings.Count(open, emptySlot) != world.InventorySize-1 {
		t.Fatalf("empty slots = %d, want %d", strings.Count(open, emptySlot), world.InventorySize-1)
	}
	if again := r.frame(100, 30, 0); strings.Contains(again, "inventory") {
		t.Fatal("an unchanged open panel must not be redrawn")
	}
	r.inventoryOpen = false
	closed := r.frame(100, 30, 0)
	// The forest must be repainted in the panel's first row, at its left edge.
	if strings.Contains(closed, "inventory") || !strings.Contains(closed, fmt.Sprintf("\x1b[2;%dH", 100-panelWidth+1)) {
		t.Fatal("closing the panel must repaint the forest beneath it")
	}
}

func TestADroppedLogLiesOnTheGround(t *testing.T) {
	w := besideTree()
	w.Chop()
	chopUntilFelled(t, w)
	x, z := w.Hero().Position()
	r := newRenderer(w, "L’Après-Règne")
	before := r.render(100, 27, 0)
	px, py, _ := before.project(x, 0.1, z+3*0.55)
	inkBefore := before.pixels[int(py)*before.width+int(px)].ink
	// Drop the log three steps down, then walk back so the hero does not cover it.
	for range 3 {
		w.Walk(0, 1, false)
	}
	w.Drop(0)
	for range 3 {
		w.Walk(0, -1, false)
	}
	after := r.render(100, 27, 0)
	if ink := after.pixels[int(py)*after.width+int(px)].ink; ink != 4 && ink != 7 || ink == inkBefore {
		t.Fatalf("ink at the dropped log = %d (was %d), want bark or its pale cut end", ink, inkBefore)
	}
}
