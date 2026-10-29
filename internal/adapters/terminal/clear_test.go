package terminal

import (
	"strings"
	"testing"

	"lapresregne/internal/world"
)

// lumaOf is how bright a color looks, from 0 to 1.
func lumaOf(color [3]int) float64 {
	return (0.299*float64(color[0]) + 0.587*float64(color[1]) + 0.114*float64(color[2])) / 255
}

// clearPaper is the paper the clear style lays under the middle of a canvas lit evenly by one ink
// at a light, seen at a scale.
func clearPaper(ink int, light, scale float64) [3]int {
	c := evenCanvas(3, 3, ink, light)
	c.scale = scale
	return Clear{}.cell(c, forestPalette[:], 1, 1).paper
}

func TestClearSetsTheGroundsMaterialsApartByValue(t *testing.T) {
	// Each at the light it typically shows when zoomed all the way out, where all three looked alike.
	water := lumaOf(clearPaper(waterInk, 0.22, forestScale))
	grass := lumaOf(clearPaper(1, 0.18, forestScale))
	path := lumaOf(clearPaper(pathInk, 0.16, forestScale))
	stone := lumaOf(clearPaper(stoneInk, 0.7, forestScale))
	if !(water+0.08 < grass && grass+0.08 < path && path+0.15 < stone) {
		t.Errorf("water %.2f, grass %.2f, path %.2f, stone %.2f; want each clearly brighter than the last", water, grass, path, stone)
	}
}

func TestClearKeepsTheGroundAsBrightZoomedOutAsCloseUp(t *testing.T) {
	// The ground is lit at half strength when zoomed all the way out.
	far := clearPaper(1, 0.18, forestScale)
	near := clearPaper(1, 0.36, detailScale)
	if far != near {
		t.Errorf("grass zoomed out shows %v; want it as bright as close up, %v", far, near)
	}
}

func TestClearFillsAnEvenlyLitWallFlat(t *testing.T) {
	c := evenCanvas(3, 3, stoneInk, 0.7)
	c.scale = detailScale
	cell := Clear{}.cell(c, forestPalette[:], 1, 1)
	if cell.bits != 0 && cell.color != cell.paper {
		t.Errorf("an evenly lit wall dots %08b in %v over %v; want one flat color", cell.bits, cell.color, cell.paper)
	}
}

func TestClearShowsShadeAsADarkerFlatTone(t *testing.T) {
	open := lumaOf(clearPaper(1, 0.36, detailScale))
	shaded := lumaOf(clearPaper(1, 0.36*0.42, detailScale))
	if shaded > open*0.8 {
		t.Errorf("grass in a tree's shadow shows %.2f; want clearly darker than open grass, %.2f", shaded, open)
	}
}

func TestClearDithersOnlyWhereOneShadeGivesWayToTheNext(t *testing.T) {
	// A wall growing lighter from left to right across twenty cells.
	c := evenCanvas(20, 1, stoneInk, 0)
	c.scale = detailScale
	for i := range c.pixels {
		c.pixels[i].light = 0.3 + 0.7*float64(i%c.width)/float64(c.width)
	}
	flat, dithered := 0, 0
	for col := range 20 {
		if cell := (Clear{}).cell(c, forestPalette[:], col, 0); cell.bits != 0 && cell.color != cell.paper {
			dithered++
		} else {
			flat++
		}
	}
	if flat == 0 || dithered == 0 {
		t.Errorf("a lightening wall draws %d flat and %d dithered cells; want both", flat, dithered)
	}
}

func TestClearDrawsTheHeroInSolidQuadrants(t *testing.T) {
	c := evenCanvas(3, 3, defaultLook.skin, 0.8)
	if cell := (Clear{}).cell(c, forestPalette[:], 1, 1); !cell.solid {
		t.Error("the hero draws braille; want solid quadrants")
	}
}

func TestClearFramesDrawBraille(t *testing.T) {
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	r.style = Clear{}
	frame := r.frame(120, 30, 0)
	if !strings.ContainsFunc(frame, func(glyph rune) bool { return glyph > 0x2800 && glyph <= 0x28ff }) {
		t.Error("a clear frame draws no braille dots")
	}
	if !strings.Contains(frame, "V style: clear") {
		t.Error("the hint line does not name the clear style")
	}
}

// viewedCanvas is a canvas cols by rows cells lit evenly by one ink at a light, seen from the
// opening camera at the zoom where trees show their detail, centred on x, z.
func viewedCanvas(cols, rows, ink int, light, x, z float64) *forestCanvas {
	c := evenCanvas(cols, rows, ink, light)
	c.scale = detailScale
	c.turnTo(0)
	c.tiltTo(openingTilt)
	c.x, c.z = x, z
	return c
}

// clearDots is which dots of a canvas the clear style draws lighter, or darker, than the paper under
// them, row by row of dots.
func clearDots(c *forestCanvas, lighter bool) [][]bool {
	dotted := make([][]bool, c.height)
	for y := range dotted {
		dotted[y] = make([]bool, c.width)
	}
	for row := range c.height / 4 {
		for col := range c.width / 2 {
			cell := Clear{}.cell(c, forestPalette[:], col, row)
			if cell.color == cell.paper || (lumaOf(cell.color) > lumaOf(cell.paper)) != lighter {
				continue
			}
			for dy := range 4 {
				for dx := range 2 {
					if cell.bits&(1<<brailleDot[dy][dx]) != 0 {
						dotted[row*4+dy][col*2+dx] = true
					}
				}
			}
		}
	}
	return dotted
}

// dotRuns counts the dots with another dot beside them, and those with one above or below them.
func dotRuns(dotted [][]bool) (total, beside, upright int) {
	at := func(x, y int) bool { return y >= 0 && y < len(dotted) && x >= 0 && x < len(dotted[y]) && dotted[y][x] }
	for y := range dotted {
		for x := range dotted[y] {
			if !dotted[y][x] {
				continue
			}
			total++
			if at(x-1, y) || at(x+1, y) {
				beside++
			}
			if at(x, y-1) || at(x, y+1) {
				upright++
			}
		}
	}
	return total, beside, upright
}

func TestClearDrawsGrassAsUprightBlades(t *testing.T) {
	total, _, upright := dotRuns(clearDots(viewedCanvas(30, 12, 1, 0.36, 0, 0), true))
	if total == 0 || float64(upright) < 0.8*float64(total) {
		t.Errorf("grass draws %d light dots, %d of them in upright runs; want blades of upright dots", total, upright)
	}
}

func TestClearDrawsWaterAsLevelDashes(t *testing.T) {
	total, beside, upright := dotRuns(clearDots(viewedCanvas(30, 12, waterInk, 0.44, 0, 0), true))
	if total == 0 || float64(beside) < 0.8*float64(total) || upright > 0 {
		t.Errorf("water draws %d light dots, %d beside another and %d above or below one; want level dashes", total, beside, upright)
	}
}

func TestClearSpecklesPathsDarker(t *testing.T) {
	total, _, _ := dotRuns(clearDots(viewedCanvas(30, 12, pathInk, 0.32, 0, 0), false))
	if total == 0 {
		t.Error("a path draws no darker specks")
	}
}

func TestClearLaysRoofTilesInCourses(t *testing.T) {
	total, beside, _ := dotRuns(clearDots(viewedCanvas(30, 12, clayInk, 0.62, 0, 0), false))
	if total == 0 || float64(beside) < 0.9*float64(total) {
		t.Errorf("a roof draws %d darker dots, %d beside another; want level courses", total, beside)
	}
}

func TestClearKeepsPatternsOnTheWorldAsTheViewPans(t *testing.T) {
	// Panning two canvas pixels east slides everything one cell west.
	here := viewedCanvas(30, 12, 1, 0.36, 0, 0)
	panned := viewedCanvas(30, 12, 1, 0.36, 2/detailScale, 0)
	for row := range 12 {
		for col := 1; col < 30; col++ {
			before := Clear{}.cell(here, forestPalette[:], col, row)
			after := Clear{}.cell(panned, forestPalette[:], col-1, row)
			if before != after {
				t.Fatalf("grass at cell %d, %d draws %08b, but %08b one cell west once panned; want the blades to move with the world", col, row, before.bits, after.bits)
			}
		}
	}
}

func TestClearLightsTheGroundAroundTheHero(t *testing.T) {
	// Bare ground, which has no pattern of its own.
	c := evenCanvas(60, 20, 0, 0.12)
	c.scale = forestScale
	c.heroX, c.heroY, c.heroShown = 60, 40, true
	near := lumaOf(Clear{}.cell(c, forestPalette[:], 30, 10).paper)
	far := lumaOf(Clear{}.cell(c, forestPalette[:], 2, 2).paper)
	if near < far*1.3 {
		t.Errorf("ground beside the hero shows %.2f; want clearly brighter than ground far off, %.2f", near, far)
	}
}

func TestClearLeavesTheGroundUnlitWithoutAHero(t *testing.T) {
	// Bare ground, which has no pattern of its own.
	c := evenCanvas(60, 20, 0, 0.12)
	c.scale = forestScale
	c.heroX, c.heroY = 60, 40
	near := Clear{}.cell(c, forestPalette[:], 30, 10).paper
	if far := (Clear{}).cell(c, forestPalette[:], 2, 2).paper; near != far {
		t.Errorf("with no hero drawn, ground in the middle shows %v; want it as ground far off, %v", near, far)
	}
}

func TestRenderingMarksWhereTheHeroStands(t *testing.T) {
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	c := r.render(100, 30, 0)
	if !c.heroShown {
		t.Fatal("a rendered frame does not mark where the hero stands")
	}
	feet := c.pixels[int(c.heroY-1)*c.width+int(c.heroX)]
	if !figureInk(feet.ink) {
		t.Errorf("just above the marked spot lies ink %d; want the hero's feet", feet.ink)
	}
}
