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

// BenchmarkScenes renders the largest view of the places and camera angles a traveler sees most.
func BenchmarkScenes(b *testing.B) {
	scenes := []struct {
		name  string
		setUp func(w *world.World)
	}{
		{"forest", func(*world.World) {}},
		{"midway", func(w *world.World) { w.ZoomBy(world.CloseUp / 2) }},
		{"close-up", func(w *world.World) { w.ZoomBy(world.CloseUp) }},
		{"turned", func(w *world.World) { w.TurnBy(world.QuarterTurn / 2) }},
		{"river", func(w *world.World) { w.Advance(1, 0, 6*time.Second, false) }},
	}
	for _, scene := range scenes {
		b.Run(scene.name, func(b *testing.B) {
			w := world.New(7)
			scene.setUp(w)
			w.Follow(time.Second)
			r := newRenderer(w, "L’Après-Règne")
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				r.frame(220, 93, float64(i%120)/30)
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

func TestTreeAtNeedsAClickOnTheTreeItself(t *testing.T) {
	r := newRenderer(world.New(7), "L’Après-Règne")
	r.frame(100, 30, 0)
	// Column 58, row 18 is open ground below the crown of the tree at x 6.0, z 1.7.
	if tree, ok := r.treeAt(58, 18); ok {
		t.Fatalf("treeAt(58, 18) = %+v; want the ground below the crown", tree)
	}
}

func TestUpCloseOnlyTheTrunkOfACutAwayCanopyIsClickable(t *testing.T) {
	w := besideTree()
	w.ZoomBy(world.CloseUp)
	r := newRenderer(w, "L’Après-Règne")
	r.frame(100, 30, 0)
	// The trunk of the tree at x 6.0, z 1.7 fills column 80, row 10, right of the hero.
	if tree, ok := r.treeAt(80, 10); !ok || tree.X < 6 || tree.X > 6.1 {
		t.Fatalf("treeAt(80, 10) = %+v, %v; want the trunk of the tree at x 6.0", tree, ok)
	}
	// Column 60, row 10 lies under the canopy, which is cut away around the hero.
	if tree, ok := r.treeAt(60, 10); ok {
		t.Fatalf("treeAt(60, 10) = %+v; want the cut-away canopy left unclickable", tree)
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
					// Walking speed is 4.5 per second, so one long frame carries the hero there.
					distance := math.Hypot(tree.X-1, tree.Z)
					w.Advance(tree.X-1, tree.Z, time.Duration(distance/4.5*float64(time.Second)), false)
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

// leafBounds draws one tree of a kind alone and measures the canvas box its leaves cover.
func leafBounds(kind world.TreeKind) (width, height, lowest int) {
	r := newRenderer(world.New(7), "L’Après-Règne")
	c := &forestCanvas{width: 120, height: 120, x: 1000, scale: 3.6}
	c.turnTo(0)
	c.pixels = make([]forestPixel, c.width*c.height)
	for i := range c.pixels {
		c.pixels[i].depth = math.Inf(1)
	}
	r.drawTree(c, world.Tree{X: 1000, Height: 6, Radius: 2.4, Trunk: 0.25, Kind: kind}, 0)
	left, right, top := c.width, -1, c.height
	leaves := leavesOf(kind)
	for y := range c.height {
		for x := range c.width {
			if ink := c.pixels[y*c.width+x].ink; ink == leaves.shaded || ink == leaves.sunlit {
				left, right, top, lowest = min(left, x), max(right, x), min(top, y), max(lowest, y)
			}
		}
	}
	return right - left, lowest - top, lowest
}

func TestEachKindOfTreeHasItsOwnSilhouette(t *testing.T) {
	oakWidth, oakHeight, oakLowest := leafBounds(world.Oak)
	if oakWidth <= oakHeight*3/2 {
		t.Errorf("an oak's canopy is %d wide and %d tall, want it spreading wide and flat", oakWidth, oakHeight)
	}
	yewWidth, yewHeight, _ := leafBounds(world.Yew)
	if yewHeight <= yewWidth {
		t.Errorf("a yew is %d wide and %d tall, want it taller than wide", yewWidth, yewHeight)
	}
	_, _, willowLowest := leafBounds(world.Willow)
	if willowLowest <= oakLowest+10 {
		t.Errorf("a willow's leaves reach row %d, want them hanging well below an oak's at row %d", willowLowest, oakLowest)
	}
}

func TestZoomingInDrawsTheHeroLarger(t *testing.T) {
	r := newRenderer(world.New(7), "L’Après-Règne")
	hero := []int{8, defaultLook.skin, defaultLook.hair, defaultLook.undershirt, defaultLook.shorts}
	far := countInk(r.render(100, 27, 0), hero...)
	r.world.ZoomBy(world.CloseUp)
	r.zoom.step(r.world.Zoom(), 5*time.Second)
	if near := countInk(r.render(100, 27, 0), hero...); near <= far*4 {
		t.Fatalf("hero pixels = %d zoomed in, want far more than %d in the forest view", near, far)
	}
}

// closeUp renders the hero at full zoom.
func closeUp(w *world.World) *forestCanvas {
	r := newRenderer(w, "L’Après-Règne")
	w.ZoomBy(world.CloseUp)
	r.zoom.step(w.Zoom(), 5*time.Second)
	return r.render(100, 27, 0)
}

func TestUpCloseTheHeroIsDrawnInDetail(t *testing.T) {
	near := closeUp(world.New(7))
	for part, ink := range map[string]int{"skin": defaultLook.skin, "hair": defaultLook.hair, "undershirt": defaultLook.undershirt, "shorts": defaultLook.shorts} {
		if countInk(near, ink) == 0 {
			t.Errorf("the close-up draws no %s", part)
		}
	}
	far := newRenderer(world.New(7), "L’Après-Règne").render(100, 27, 0)
	if countInk(far, defaultLook.skin, defaultLook.undershirt) != 0 {
		t.Error("the forest view must keep the small traveler")
	}
}

func TestTheHeroShowsTheirFaceOnlyWhenFacingTheViewer(t *testing.T) {
	w := world.New(7)
	w.Hero().Face(world.FacingDown)
	if countInk(closeUp(w), eyeInk) == 0 {
		t.Error("a hero facing down must show their eyes")
	}
	w.Hero().Face(world.FacingUp)
	if countInk(closeUp(w), eyeInk) != 0 {
		t.Error("a hero facing away must show only the back of their head")
	}
}

// ellipsoidWidth measures how many canvas columns a long, thin ellipsoid covers when turned by yaw.
func ellipsoidWidth(yaw float64) int {
	c := &forestCanvas{width: 120, height: 120, scale: 10}
	c.turnTo(0)
	c.pixels = make([]forestPixel, c.width*c.height)
	for i := range c.pixels {
		c.pixels[i].depth = math.Inf(1)
	}
	left, right := c.width, -1
	ellipsoid(c, 0, 0, 0, 2, 0.5, 0.2, yaw, func(px, _ int, _ float64, _, _ vec) {
		left, right = min(left, px), max(right, px)
	})
	return right - left
}

func TestAnEllipsoidTurnsAboutItsUprightAxis(t *testing.T) {
	across, along := ellipsoidWidth(0), ellipsoidWidth(math.Pi/2)
	if along*4 >= across {
		t.Fatalf("a long ellipsoid covers %d columns across the view and %d turned toward it, want it much narrower turned", across, along)
	}
}

// barkAcross draws one tree of a kind alone at a scale and counts the bark pixels across its trunk
// just above the ground.
func barkAcross(kind world.TreeKind, scale float64) int {
	r := newRenderer(world.New(7), "L’Après-Règne")
	c := &forestCanvas{width: 400, height: 400, x: 1000, scale: scale}
	c.turnTo(0)
	c.pixels = make([]forestPixel, c.width*c.height)
	for i := range c.pixels {
		c.pixels[i].depth = math.Inf(1)
	}
	tree := world.Tree{X: 1000, Height: 6, Radius: 2.4, Trunk: 0.25, Kind: kind}
	r.drawTree(c, tree, 0)
	_, y, _ := c.project(tree.X, 0.25, tree.Z)
	bark := 0
	for x := range c.width {
		if c.pixels[int(y)*c.width+x].ink == 4 {
			bark++
		}
	}
	return bark
}

func TestUpCloseTrunksAreSolidWood(t *testing.T) {
	for _, kind := range []world.TreeKind{world.NormalTree, world.Oak, world.Willow, world.Yew} {
		// A trunk at least as thick as its strands are spread, across at the close-up scale.
		want := int(2 * 0.25 * closeUpScale * 0.8)
		if got := barkAcross(kind, closeUpScale); got < want {
			t.Errorf("a %v trunk is %d bark pixels across up close, want a solid trunk of at least %d", kind, got, want)
		}
	}
}

func TestTheDetailedHeroStaysAFewStepsFromTheForestView(t *testing.T) {
	r := newRenderer(world.New(7), "L’Après-Règne")
	r.world.ZoomBy(3)
	r.zoom.step(r.world.Zoom(), 5*time.Second)
	if countInk(r.render(100, 27, 0), defaultLook.skin) == 0 {
		t.Fatal("three wheel steps in, the hero must already be drawn in detail")
	}
}

func TestOnlyTreesThatReachTheCanvasAreDrawn(t *testing.T) {
	c := &forestCanvas{width: 200, height: 120, scale: forestScale}
	c.turnTo(0)
	tree := world.Tree{Height: 6, Radius: 2.4, Trunk: 0.25}
	inView, farLeft, belowWithCrownInView := tree, tree, tree
	farLeft.X = -60
	// The ground at the bottom edge lies about 20 units in front of the camera; the crown rises into view.
	belowWithCrownInView.Z = 22
	if !c.mayShow(inView) || !c.mayShow(belowWithCrownInView) {
		t.Fatal("a tree reaching into the canvas must be drawn")
	}
	if c.mayShow(farLeft) {
		t.Fatal("a tree wholly beside the canvas must be skipped")
	}
}
