package window

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"lapresregne/internal/world"
)

const tick = time.Second / ticksPerSecond

func TestATreeStandsOnItsTrunk(t *testing.T) {
	for _, kind := range []world.TreeKind{world.NormalTree, world.Oak, world.Willow, world.Yew, world.DeadTree} {
		sprite := treeSprite(kind, 40, 80, 0.5)
		bounds := sprite.Bounds()
		if sprite.RGBAAt(bounds.Dx()/2, 79).A == 0 {
			t.Errorf("%v has nothing at its foot; want its trunk there", kind)
		}
	}
}

func TestTheCameraMapsTheScreenBackOntoTheGround(t *testing.T) {
	view := camera{3, -2}
	sx, sy := view.toScreen(5.5, 1.25)
	if x, z := view.toWorld(sx, sy); math.Abs(x-5.5) > 1e-9 || math.Abs(z-1.25) > 1e-9 {
		t.Fatalf("toWorld(toScreen(5.5, 1.25)) = %v, %v", x, z)
	}
	if sx, sy := view.toScreen(3, -2); sx != screenWidth/2 || sy != screenHeight/2 {
		t.Fatalf("the camera's own point shows at %v, %v; want the screen's centre", sx, sy)
	}
}

func TestNearerSpritesAreDrawnOverFartherOnes(t *testing.T) {
	g := newGame(world.New(7), nil)
	g.step(controls{}, tick)
	if len(g.scene) < 2 {
		t.Fatalf("the scene holds %d sprites; want the hero among trees", len(g.scene))
	}
	for i := 1; i < len(g.scene); i++ {
		if g.scene[i].depth < g.scene[i-1].depth {
			t.Fatalf("sprite %d at depth %v drawn after one at %v", i, g.scene[i].depth, g.scene[i-1].depth)
		}
	}
}

func TestHeldKeysWalkTheHero(t *testing.T) {
	g := newGame(world.New(7), nil)
	x, _ := g.world.Hero().Position()
	for range ticksPerSecond {
		g.step(controls{dx: 1}, tick)
	}
	if moved, _ := g.world.Hero().Position(); moved <= x {
		t.Fatalf("holding right for a second moves the hero from x %v to %v; want east", x, moved)
	}
}

func TestClickingTheGroundWalksThere(t *testing.T) {
	g := newGame(world.New(7), nil)
	g.step(controls{}, tick)
	// Find open ground a little to the hero's left, clear of any tree.
	for sx := screenWidth/2 - 40; sx > 0; sx -= 4 {
		if _, onTree := treeAt(g.scene, sx, screenHeight/2); onTree {
			continue
		}
		targetX, _ := g.view.snapped().toWorld(float64(sx)+0.5, screenHeight/2+0.5)
		g.step(controls{click: true, pointing: true, cursorX: sx, cursorY: screenHeight / 2}, tick)
		for range 5 * ticksPerSecond {
			g.step(controls{}, tick)
		}
		if x, _ := g.world.Hero().Position(); math.Abs(x-targetX) > 1 {
			t.Fatalf("after clicking the ground at x %v the hero stands at x %v", targetX, x)
		}
		return
	}
	t.Fatal("no open ground to click")
}

func TestClickingATreeWalksThereToChopIt(t *testing.T) {
	g := newGame(world.New(7), nil)
	g.step(controls{}, tick)
	for _, p := range g.scene {
		// A hero at level 1 can chop only normal and dead trees.
		if p.tree == nil || p.tree.Stump || p.tree.Kind != world.NormalTree {
			continue
		}
		bounds := p.art.Bounds()
		foot := image.Pt(p.x+bounds.Dx()/2, p.y+bounds.Dy()-3)
		if tree, ok := treeAt(g.scene, foot.X, foot.Y); !ok || tree != *p.tree {
			continue
		}
		g.step(controls{click: true, pointing: true, cursorX: foot.X, cursorY: foot.Y}, tick)
		for range 20 * ticksPerSecond {
			g.step(controls{}, tick)
			if chopped, chopping := g.world.ChoppedTree(); chopping && chopped.X == p.tree.X && chopped.Z == p.tree.Z {
				return
			}
		}
		t.Fatalf("clicking the tree at %v, %v never set the hero chopping it", p.tree.X, p.tree.Z)
	}
	t.Fatal("no tree's trunk to click")
}

func TestTheGroundShowsTheRiver(t *testing.T) {
	w := world.New(7)
	// Walk south along x 0 until the ground is a river.
	for z := 0.0; z < 400; z += 0.5 {
		if distance, width := w.River(0, z); distance < width/2 {
			c, _, _ := groundPixel(w, 0, int(z*unit*depthSquash))
			for _, blue := range waterTones {
				if c == blue {
					return
				}
			}
			t.Fatalf("the river shows as %v; want one of the water's tones", c)
		}
	}
	t.Skip("no river within 400 units south")
}

func TestTheGroundIsPaintedInCleanShapesWithoutStatic(t *testing.T) {
	chunk := paintChunk(world.New(7), 0, 0)
	lone := 0
	for y := range chunkSize {
		for x := 1; x < chunkSize-1; x++ {
			c := chunk.RGBAAt(x, y)
			if c != chunk.RGBAAt(x-1, y) && c != chunk.RGBAAt(x+1, y) {
				lone++
			}
		}
	}
	if share := float64(lone) / (chunkSize * chunkSize); share > 0.04 {
		t.Errorf("%.0f%% of the ground's pixels stand alone in their row; want clean shapes", share*100)
	}
}

func TestClearingsGrowWildflowers(t *testing.T) {
	w := world.New(7)
	flowers := map[color.RGBA]bool{crimsonTones[1]: true, goldTones[1]: true, paleTones[1]: true}
	for cy := -3; cy <= 3; cy++ {
		for cx := -3; cx <= 3; cx++ {
			chunk := paintChunk(w, cx, cy)
			for i := 0; i < len(chunk.Pix); i += 4 {
				if flowers[color.RGBA{chunk.Pix[i], chunk.Pix[i+1], chunk.Pix[i+2], 255}] {
					return
				}
			}
		}
	}
	t.Fatal("no wildflower grows anywhere near the start")
}

func TestTheSameGroundIsPaintedEveryTime(t *testing.T) {
	w := world.New(7)
	if string(paintChunk(w, 2, -1).Pix) != string(paintChunk(w, 2, -1).Pix) {
		t.Fatal("painting one chunk twice gives two pictures")
	}
}

func brightness(c color.RGBA) int { return int(c.R) + int(c.G) + int(c.B) }

func TestACrownIsLitFromTheUpperLeft(t *testing.T) {
	for _, kind := range []world.TreeKind{world.NormalTree, world.Oak, world.Willow, world.Yew} {
		sprite := treeSprite(kind, 60, 60, 0.4)
		bounds := sprite.Bounds()
		// Compare the crown's upper left quarter with its lower right one, above the trunk.
		var lit, shaded, litCount, shadedCount int
		for y := 0; y < bounds.Dy()*2/3; y++ {
			for x := 0; x < bounds.Dx(); x++ {
				c := sprite.RGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				if x < bounds.Dx()/2 && y < bounds.Dy()/3 {
					lit, litCount = lit+brightness(c), litCount+1
				} else if x >= bounds.Dx()/2 && y >= bounds.Dy()/3 {
					shaded, shadedCount = shaded+brightness(c), shadedCount+1
				}
			}
		}
		if litCount == 0 || shadedCount == 0 || lit/litCount <= shaded/shadedCount {
			t.Errorf("%v's crown is no brighter at its upper left than its lower right", kind)
		}
	}
}

func TestStandingTreesCastShadowsAwayFromTheLight(t *testing.T) {
	g := newGame(world.New(7), nil)
	g.step(controls{}, tick)
	if len(g.shadows) < 2 {
		t.Fatalf("%d shadows lie on the ground; want the trees' and the hero's", len(g.shadows))
	}
	x, z := g.world.Hero().Position()
	footX, _ := g.view.snapped().toScreen(x, z)
	for _, shadow := range g.shadows {
		if shadow.depth != z {
			continue
		}
		if middle := float64(shadow.x) + float64(shadow.art.Bounds().Dx())/2; middle <= footX {
			t.Errorf("the hero's shadow centres at x %v, not right of their foot at %v", middle, footX)
		}
	}
}

func TestEveryPictureOfTheHeroLinesUp(t *testing.T) {
	for facing, pose := range heroPoses {
		pictures := [][]string{pose.upper, pose.upperWorn}
		pictures = append(pictures, pose.legs[:]...)
		pictures = append(pictures, pose.legsWorn[:]...)
		for _, rows := range pictures {
			for i, row := range rows {
				if len(row) != heroWidth {
					t.Errorf("%v: row %d %q is %d wide; want %d", facing, i, row, len(row), heroWidth)
				}
			}
		}
		if len(pose.upperWorn) != len(pose.upper) {
			t.Errorf("%v: the worn layer is %d rows tall over a body of %d", facing, len(pose.upperWorn), len(pose.upper))
		}
	}
}

func TestTheHeroFacesTheWayTheyWalk(t *testing.T) {
	left, right := heroSprite(world.FacingLeft, 0), heroSprite(world.FacingRight, 0)
	if string(left.Pix) == string(right.Pix) || string(left.Pix) != string(mirrored(right).Pix) {
		t.Error("the hero facing left is not the hero facing right, mirrored")
	}
	if string(heroSprite(world.FacingDown, 0).Pix) == string(heroSprite(world.FacingUp, 0).Pix) {
		t.Error("the hero looks the same from the front and the back")
	}
}

func TestTheHeroWearsTheirClothesOverTheirBody(t *testing.T) {
	sprite := heroSprite(world.FacingDown, 0)
	if c := sprite.RGBAAt(6, 12); c != heroLegend['W'] {
		t.Errorf("the hero's chest shows %v; want the undershirt", c)
	}
	if c := sprite.RGBAAt(4, 22); c == heroLegend['W'] || c == heroLegend['P'] || c.A == 0 {
		t.Errorf("the hero's shin shows %v; want bare skin", c)
	}
}

func TestTheHeroStridesOnlyWhileWalking(t *testing.T) {
	g := newGame(world.New(7), nil)
	g.step(controls{}, tick)
	g.step(controls{}, tick)
	if frame := g.gait.frame(); frame != 0 {
		t.Fatalf("standing still shows walk frame %d; want 0", frame)
	}
	frames := map[int]bool{}
	for range 2 * ticksPerSecond {
		g.step(controls{dx: 1}, tick)
		frames[g.gait.frame()] = true
	}
	if !frames[1] || !frames[2] {
		t.Fatalf("two seconds of walking showed frames %v; want both strides", frames)
	}
}

func TestTheScreensEdgesSinkIntoDarkness(t *testing.T) {
	if middle, corner := vignette(screenWidth/2, screenHeight/2), vignette(0, 0); middle != 0 || corner <= 0.5 {
		t.Errorf("the vignette is %v in the middle and %v in the corner; want none and deep", middle, corner)
	}
}

func TestLightShaftsLeanDownToTheRightAndDriftSlowly(t *testing.T) {
	// The brightest point of the top row lies left of the brightest of a row further down.
	brightest := func(y int) int {
		best, at := 0.0, 0
		for x := range shaftPeriod {
			if light := shaft(float64(x) - float64(y)*shaftSlant); light > best {
				best, at = light, x
			}
		}
		return at
	}
	if top, lower := brightest(0), brightest(100); lower <= top {
		t.Errorf("the brightest shaft is at x %d up top and %d further down; want it leaning right", top, lower)
	}
	// The offset wraps round the period, so the drift is the shorter way round.
	moved := math.Mod(math.Abs(shaftOffset(0, 10)-shaftOffset(0, 0)), shaftPeriod)
	moved = math.Min(moved, shaftPeriod-moved)
	if moved == 0 || moved > 30 {
		t.Errorf("the shafts drift %v pixels in ten seconds; want a slow drift", moved)
	}
}

func TestMotesDriftSlowlyInTheAir(t *testing.T) {
	before, after := motes(0, 0, 3), motes(0, 0, 3.5)
	if len(before) == 0 || len(before) != len(after) {
		t.Fatalf("%d motes, then %d half a second later; want the same few", len(before), len(after))
	}
	for i := range before {
		if moved := math.Hypot(after[i].x-before[i].x, after[i].y-before[i].y); moved > 4 {
			t.Errorf("mote %d moves %v pixels in half a second; want a slow drift", i, moved)
		}
	}
}
