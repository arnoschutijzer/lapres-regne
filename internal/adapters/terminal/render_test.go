package terminal

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lapresregne/internal/adapters/mapfile"
	"lapresregne/internal/world"
)

func BenchmarkForestFrame(b *testing.B) {
	for _, size := range [][2]int{{100, 30}, {180, 50}, {220, 90}} {
		b.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(b *testing.B) {
			r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				r.frame(size[0], size[1], float64(i%120)/30)
			}
		})
	}
}

// BenchmarkScenes renders the largest view of the places and camera angles a traveler sees most.
func BenchmarkScenes(b *testing.B) {
	lumbridge, err := mapfile.Load(filepath.Join("..", "..", "..", "maps", "lumbridge.map"))
	if err != nil {
		b.Fatal(err)
	}
	forest := func() *world.World { return world.New(7) }
	castle := func() *world.World { return world.NewOnMap(7, lumbridge) }
	scenes := []struct {
		name   string
		place  func() *world.World
		walk   time.Duration
		camera func() *Camera
	}{
		{"forest", forest, 0, func() *Camera { return cameraAt(0, 0) }},
		{"midway", forest, 0, func() *Camera { return cameraAt(closeUpZoom/2, 0) }},
		{"close-up", forest, 0, func() *Camera { return cameraAt(closeUpZoom, 0) }},
		{"turned", forest, 0, func() *Camera { return cameraAt(0, quarterTurn/2) }},
		{"river", forest, 6 * time.Second, func() *Camera { return cameraAt(0, 0) }},
		{"lumbridge", castle, 0, func() *Camera { return cameraAt(0, 0) }},
		{"lumbridge-midway", castle, 0, func() *Camera { return cameraAt(closeUpZoom/2, 0) }},
		{"lumbridge-close-up", castle, 0, func() *Camera { return cameraAt(closeUpZoom, 0) }},
		{"lumbridge-turned", castle, 0, func() *Camera { return cameraAt(0, quarterTurn/2) }},
	}
	for _, scene := range scenes {
		b.Run(scene.name, func(b *testing.B) {
			w := scene.place()
			w.Advance(1, 0, scene.walk, false)
			r := newRenderer(w, scene.camera(), "L’Après-Règne")
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				r.frame(220, 93, float64(i%120)/30)
			}
		})
	}
}

func TestGroundAtFollowsTheScreenAroundTheHero(t *testing.T) {
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
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
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
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
	// Up close the axe head spans several pixels, so its swing shows wherever the hero stands.
	r := newRenderer(w, NewCamera(4, 0), "L’Après-Règne")
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
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
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
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
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
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
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
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
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
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	r.frame(100, 30, 0)
	// Column 58, row 18 is open ground below the crown of the tree at x 6.0, z 1.7.
	if tree, ok := r.treeAt(58, 18); ok {
		t.Fatalf("treeAt(58, 18) = %+v; want the ground below the crown", tree)
	}
}

func TestUpCloseOnlyTheTrunkOfACutAwayCanopyIsClickable(t *testing.T) {
	w := besideTree()
	r := newRenderer(w, NewCamera(closeUpZoom, 0), "L’Après-Règne")
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
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	if frame := r.frame(100, 30, 0); !strings.Contains(frame, "\x1b[1;") || !strings.Contains(frame, "woodcutting 1 · 0 xp") {
		t.Fatal("the title row must show the woodcutting level and xp")
	}
	w.Chop()
	chopUntilFelled(t, w)
	if frame := r.frame(100, 30, 0); !strings.Contains(frame, "woodcutting 1 · 25 xp") {
		t.Fatal("the title row must update with gained xp")
	}
}

// titleRow plays frames onto an empty screen of a width and reads back the text of its first row.
func titleRow(width int, frames ...string) string {
	row := []rune(strings.Repeat(" ", width))
	line, column := 1, 1
	for _, frame := range frames {
		input := []rune(frame)
		for i := 0; i < len(input); i++ {
			if input[i] != '\x1b' {
				if line == 1 && column <= width {
					row[column-1] = input[i]
				}
				column++
				continue
			}
			// A control sequence runs from ESC [ to its final letter.
			end := i + 2
			for end < len(input) && (input[end] < 0x40 || input[end] > 0x7e) {
				end++
			}
			params, final := string(input[i+2:end]), input[end]
			i = end
			switch {
			case final == 'H':
				line, column = 1, 1
				if params != "" {
					fmt.Sscanf(params, "%d;%d", &line, &column)
				}
			case final == 'J' && params == "2":
				row = []rune(strings.Repeat(" ", width))
			case final == 'K' && line == 1 && params == "2":
				row = []rune(strings.Repeat(" ", width))
			case final == 'K' && line == 1:
				for c := column; c <= width; c++ {
					row[c-1] = ' '
				}
			}
		}
	}
	return string(row)
}

func TestAShorterSkillLeavesNothingOfTheLastOneOnTheTitleRow(t *testing.T) {
	resumed := func(xp float64) *world.World {
		w, err := world.Resume(7, savedProgress{world.Snapshot{Seed: 1, Skills: map[string]float64{world.Woodcutting: xp}}}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	r := newRenderer(resumed(2462.5), NewCamera(0, 0), "L’Après-Règne")
	before := r.frame(100, 30, 0)
	// An oak's 37.5 xp makes the right-aligned skill one character shorter.
	r.world = resumed(2500)
	after := r.frame(100, 30, 0)
	if row := titleRow(100, before, after); !strings.HasSuffix(row, " woodcutting 15 · 2500 xp  ") {
		t.Fatalf("title row = %q, want it to end in just the new skill", row)
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
					return w
				}
			}
		}
	}
	t.Fatalf("no %v within 40 cells", kind)
	return nil
}

func TestWillowsAndYewsHaveTheirOwnLeaves(t *testing.T) {
	willow := newRenderer(besideWillow(), NewCamera(0, 0), "L’Après-Règne")
	if countInk(willow.render(100, 27, 0), 11, 12) == 0 {
		t.Fatal("a willow must draw pale willow leaves")
	}
	yew := newRenderer(nearFirst(t, world.Yew), NewCamera(0, 0), "L’Après-Règne")
	if countInk(yew.render(100, 27, 0), 13, 14) == 0 {
		t.Fatal("a yew must draw dark yew needles")
	}
}

func TestTheMenuNamesTheTreeAndMessagesShowOnTheHintLine(t *testing.T) {
	w := besideWillow()
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
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
	r := newRenderer(besideTree(), NewCamera(0, 0), "L’Après-Règne")
	if frame := r.frame(100, 30, 0); strings.Contains(frame, "logs") {
		t.Fatal("the status line must not mention logs")
	}
}

func TestTheInventoryPanelOpensWithTintedLogsAndClosesCleanly(t *testing.T) {
	w := besideTree()
	w.Chop()
	chopUntilFelled(t, w)
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
	if frame := r.frame(100, 30, 0); strings.Contains(frame, "inventory") {
		t.Fatal("the inventory panel must start closed")
	}
	r.inventoryOpen = true
	open := r.frame(100, 30, 0)
	if !strings.Contains(open, "inventory 1/28") || !strings.Contains(open, icon(world.Logs)[0]) {
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
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
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
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
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
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	hero := []int{8, defaultLook.skin, defaultLook.hair, defaultLook.undershirt, defaultLook.shorts}
	far := countInk(r.render(100, 27, 0), hero...)
	r.camera.zoomBy(closeUpZoom)
	r.zoom.step(r.camera.Zoom(), 5*time.Second)
	if near := countInk(r.render(100, 27, 0), hero...); near <= far*4 {
		t.Fatalf("hero pixels = %d zoomed in, want far more than %d in the forest view", near, far)
	}
}

// closeUp renders the hero at full zoom, with the camera turned by steps.
func closeUp(w *world.World, turn int) *forestCanvas {
	return newRenderer(w, cameraAt(closeUpZoom, turn), "L’Après-Règne").render(100, 27, 0)
}

// cameraAt is a camera zoomed in and turned by steps.
func cameraAt(zoom, turn int) *Camera {
	c := NewCamera(zoom, 0)
	c.turnBy(turn)
	return c
}

func TestUpCloseTheHeroIsDrawnInDetail(t *testing.T) {
	near := closeUp(world.New(7), 0)
	for part, ink := range map[string]int{"skin": defaultLook.skin, "hair": defaultLook.hair, "undershirt": defaultLook.undershirt, "shorts": defaultLook.shorts} {
		if countInk(near, ink) == 0 {
			t.Errorf("the close-up draws no %s", part)
		}
	}
}

func TestTheHeroShowsTheirFaceOnlyWhenFacingTheViewer(t *testing.T) {
	w := world.New(7)
	w.Hero().Face(world.FacingDown)
	if countInk(closeUp(w, 0), eyeInk) == 0 {
		t.Error("a hero facing down must show their eyes")
	}
	w.Hero().Face(world.FacingUp)
	if countInk(closeUp(w, 0), eyeInk) != 0 {
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
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
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
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	r.camera.zoomBy(3)
	r.zoom.step(r.camera.Zoom(), 5*time.Second)
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

func TestADeadTreeStandsBareOfLeaves(t *testing.T) {
	ground := make([][]world.Ground, 30)
	for row := range ground {
		ground[row] = make([]world.Ground, 40)
	}
	swamp := func(trees []world.PlantedTree) *world.World {
		return world.NewOnMap(7, world.Map{Name: "Swamp", Ground: ground, StartCol: 20, StartRow: 16, Trees: trees})
	}
	leaves := leavesOf(world.NormalTree)
	for _, scale := range []float64{0, 6} {
		render := func(w *world.World) *forestCanvas {
			r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
			r.zoom = zoom{current: scale}
			return r.render(160, 90, 0)
		}
		bare := render(swamp(nil))
		dead := render(swamp([]world.PlantedTree{{Col: 20, Row: 13, Kind: world.DeadTree}}))
		if countInk(dead, 4) <= countInk(bare, 4) {
			t.Errorf("zoom %v: the dead tree must stand in bark", scale)
		}
		if countInk(dead, leaves.shaded, leaves.sunlit) > countInk(bare, leaves.shaded, leaves.sunlit) {
			t.Errorf("zoom %v: the dead tree must bear no leaves", scale)
		}
	}
}

func TestEveryColourShowsBrighterWithItsHueKept(t *testing.T) {
	if got, want := foreground([3]int{100, 60, 20}), "\x1b[38;2;140;84;28m"; got != want {
		t.Fatalf("foreground = %q, want %q", got, want)
	}
	if got, want := paperOf([3]int{240, 120, 0}), "\x1b[48;2;255;128;0m"; got != want {
		t.Fatalf("paper = %q, want %q: a bright color lifts only as far as full strength", got, want)
	}
}

func TestTheColorUnderTheDotsStrengthensAsTheViewZoomsOut(t *testing.T) {
	far := forestCanvas{scale: forestScale}
	near := forestCanvas{scale: detailScale}
	if far.underlay() != 2 || near.underlay() != 1 {
		t.Fatalf("underlay zoomed out %v, at tree detail %v; want 2 and 1", far.underlay(), near.underlay())
	}
}

func TestDotsStandOutFromPaperTintedInTheirColor(t *testing.T) {
	dot, paper := [3]int{100, 130, 70}, [3]int{40, 50, 35}
	pushed := standOut(dot, paper)
	if colorDistance(pushed, paper) <= colorDistance(dot, paper) {
		t.Fatalf("dot %v on paper %v, want it pushed further from the paper than %v", pushed, paper, dot)
	}
}

func TestFramesArriveWhole(t *testing.T) {
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	for _, size := range [][2]int{{100, 30}, {100, 30}, {20, 5}} {
		frame := r.frame(size[0], size[1], 0)
		if !strings.HasPrefix(frame, synchronizedStart) || !strings.HasSuffix(frame, synchronizedEnd) {
			t.Errorf("a %d x %d frame is sent as it is drawn; want it held until whole, so it never shows half drawn", size[0], size[1])
		}
	}
}
