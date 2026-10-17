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

func TestShadowsSinkTowardVioletAndHighlightsWarm(t *testing.T) {
	shadow := ramp(barkMaterial, 0.25)
	if shadow.B <= shadow.G {
		t.Errorf("bark in shadow = %v; want it bluer than green", shadow)
	}
	stone := ramp(stoneMaterial, 1)
	if int(stone.R)-int(stone.B) <= int(stoneMaterial[0])-int(stoneMaterial[2]) {
		t.Errorf("stone in full light = %v; want it warmer than its own %v", stone, stoneMaterial)
	}
}

func TestShadingFallsInFlatShades(t *testing.T) {
	if ramp(grassMaterial, 0.5) != ramp(grassMaterial, 0.55) {
		t.Error("nearly equal light gives two colors; want one flat shade")
	}
	if ramp(grassMaterial, 0.2) == ramp(grassMaterial, 0.8) {
		t.Error("light 0.2 and 0.8 give one color; want different shades")
	}
}

func TestMaterialsAreRicherThanTheirOwnColor(t *testing.T) {
	full := ramp(grassMaterial, 0.75)
	spread := func(r, g, b float64) float64 { return math.Max(r, math.Max(g, b)) - math.Min(r, math.Min(g, b)) }
	if spread(float64(full.R), float64(full.G), float64(full.B)) <= spread(grassMaterial[0], grassMaterial[1], grassMaterial[2]) {
		t.Errorf("grass at full shade = %v; want more saturated than %v", full, grassMaterial)
	}
}

func TestSpritesHaveADarkOutline(t *testing.T) {
	sprite := image.NewRGBA(image.Rect(0, 0, 3, 3))
	lit := ramp(grassMaterial, 1)
	sprite.SetRGBA(1, 1, lit)
	drawn := outlined(sprite)
	for _, edge := range []image.Point{{0, 1}, {2, 1}, {1, 0}, {1, 2}} {
		c := drawn.RGBAAt(edge.X, edge.Y)
		if c.A == 0 || int(c.R)+int(c.G)+int(c.B) >= int(lit.R)+int(lit.G)+int(lit.B) {
			t.Errorf("pixel %v beside the sprite = %v; want a dark outline", edge, c)
		}
	}
	if corner := drawn.RGBAAt(0, 0); corner.A != 0 {
		t.Errorf("the diagonal corner = %v; want it clear", corner)
	}
}

func TestATreeStandsOnItsTrunk(t *testing.T) {
	for _, kind := range []world.TreeKind{world.NormalTree, world.Oak, world.Willow, world.Yew, world.DeadTree} {
		sprite := treeSprite(kind, 40, 80, 0.5)
		bounds := sprite.Bounds()
		if sprite.RGBAAt(bounds.Dx()/2, 79).A == 0 {
			t.Errorf("%v has nothing at its foot; want its trunk there", kind)
		}
	}
}

func TestTheHeroLooksTheWayTheyFace(t *testing.T) {
	left, right := heroSprite(world.FacingLeft, unit), heroSprite(world.FacingRight, unit)
	if string(left.Pix) == string(right.Pix) {
		t.Error("the hero facing left and right look the same")
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
