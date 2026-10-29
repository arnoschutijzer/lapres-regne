package terminal

import (
	"strings"
	"testing"

	"lapresregne/internal/world"
)

// oneCell is a canvas a single cell wide and high, its top and bottom halves lit by an ink each.
func oneCell(topInk int, topLight float64, bottomInk int, bottomLight float64) *forestCanvas {
	c := &forestCanvas{width: 2, height: 4, pixels: make([]forestPixel, 8)}
	for i := range c.pixels {
		if i < 4 {
			c.pixels[i] = forestPixel{ink: topInk, light: topLight}
		} else {
			c.pixels[i] = forestPixel{ink: bottomInk, light: bottomLight}
		}
	}
	return c
}

func TestDitheredFramesDrawBraille(t *testing.T) {
	frame := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne").frame(100, 30, 0)
	if !strings.ContainsFunc(frame, func(glyph rune) bool { return glyph > 0x2800 && glyph <= 0x28ff }) {
		t.Error("a dithered frame draws no braille dots")
	}
}

func TestLooksStartOnTheNamedStyle(t *testing.T) {
	looks := NewLooks("dither", Etched{}, Dithered{})
	if looks.Name() != "dither" {
		t.Fatalf("looks open on %q; want the named dither", looks.Name())
	}
	if _, ok := looks.style().(Dithered); !ok {
		t.Fatalf("looks draw in %T; want Dithered", looks.style())
	}
}

func TestLooksStartOnTheFirstStyleForAnUnknownName(t *testing.T) {
	if looks := NewLooks("", Etched{}, Dithered{}); looks.Name() != "etch" {
		t.Fatalf("looks open on %q; want the first style, etch", looks.Name())
	}
}

func TestLooksCycleThroughEveryStyle(t *testing.T) {
	looks := NewLooks("etch", Etched{}, Dithered{})
	looks.next()
	if looks.Name() != "dither" {
		t.Fatalf("after one switch looks show %q; want dither", looks.Name())
	}
	looks.next()
	if looks.Name() != "etch" {
		t.Fatalf("after two switches looks show %q; want etch again", looks.Name())
	}
}

// dots counts the braille dots a cell draws.
func dots(cell forestCell) int {
	count := 0
	for bits := cell.bits; bits != 0; bits &= bits - 1 {
		count++
	}
	return count
}

// evenCanvas is a canvas cols by rows cells wide and high, lit evenly by one ink.
func evenCanvas(cols, rows, ink int, light float64) *forestCanvas {
	c := &forestCanvas{width: cols * 2, height: rows * 4, pixels: make([]forestPixel, cols*rows*8)}
	for i := range c.pixels {
		c.pixels[i] = forestPixel{ink: ink, light: light}
	}
	return c
}

func TestEtchStipplesOpenGroundInItsOwnColor(t *testing.T) {
	c := evenCanvas(3, 3, 1, 0.6)
	cell := Etched{}.cell(c, forestPalette[:], 1, 1)
	if cell.bits == 0 {
		t.Fatal("open grass draws no dots")
	}
	if cell.ink != 1 {
		t.Errorf("open grass dots take ink %d; want the grass's own 1", cell.ink)
	}
}

func TestEtchScattersDotsUnlikeTheOrderedDither(t *testing.T) {
	c := evenCanvas(8, 8, 1, 0.5)
	for col := range 8 {
		for row := range 8 {
			if (Etched{}).cell(c, forestPalette[:], col, row).bits != (Dithered{}).cell(c, forestPalette[:], col, row).bits {
				return
			}
		}
	}
	t.Error("etched dots fall exactly where the ordered dither's do; want them jittered")
}

func brightnessOf(color [3]int) int { return color[0] + color[1] + color[2] }

func TestBoldLaysStandingThingsOnASolidBlockOfTheirColor(t *testing.T) {
	c := evenCanvas(3, 3, 4, 0.8)
	bold, dither := Bold{}.cell(c, forestPalette[:], 1, 1), Dithered{}.cell(c, forestPalette[:], 1, 1)
	if bold.bits != dither.bits || bold.ink != dither.ink {
		t.Errorf("bold bark draws dots %08b in ink %d; want the dither's %08b in %d", bold.bits, bold.ink, dither.bits, dither.ink)
	}
	if brightnessOf(bold.paper) <= brightnessOf(dither.paper) {
		t.Errorf("bold bark lies on %v; want a stronger block than the dither's %v", bold.paper, dither.paper)
	}
}

func TestBoldKeepsTheGroundDottedCloseUp(t *testing.T) {
	c := evenCanvas(3, 3, 1, 0.8)
	c.scale = detailScale
	bold, dither := Bold{}.cell(c, forestPalette[:], 1, 1), Dithered{}.cell(c, forestPalette[:], 1, 1)
	if bold.bits != dither.bits || bold.paper != dither.paper {
		t.Errorf("bold grass draws dots %08b on %v; want the dither's %08b on %v", bold.bits, bold.paper, dither.bits, dither.paper)
	}
}

func TestBoldQuietsTheGroundWhenZoomedOut(t *testing.T) {
	out, close := evenCanvas(3, 3, 1, 0.8), evenCanvas(3, 3, 1, 0.8)
	out.scale, close.scale = forestScale, detailScale
	far, near := Bold{}.cell(out, forestPalette[:], 1, 1), Bold{}.cell(close, forestPalette[:], 1, 1)
	if brightnessOf(far.color) >= brightnessOf(near.color) {
		t.Errorf("grass dots zoomed out show %v; want fainter than close up, %v", far.color, near.color)
	}
	if brightnessOf(far.paper) >= brightnessOf((Dithered{}).cell(out, forestPalette[:], 1, 1).paper) {
		t.Errorf("grass paper zoomed out is %v; want fainter than the dither's", far.paper)
	}
	if want := standOut(forestPalette[1], near.paper); near.color != want {
		t.Errorf("grass dots close up show %v; want the dither's %v", near.color, want)
	}
}

func TestBoldShadesBlocksInFlatBands(t *testing.T) {
	darker, lighter := Bold{}.cell(oneCell(4, 0.62, 4, 0.62), forestPalette[:], 0, 0), Bold{}.cell(oneCell(4, 0.66, 4, 0.66), forestPalette[:], 0, 0)
	if darker.paper != lighter.paper {
		t.Errorf("nearly equal light lays %v and %v; want one flat shade", darker.paper, lighter.paper)
	}
}

func TestBoldShadesAWallByItsLightNotItsStonework(t *testing.T) {
	// A stone wall in full light, its middle cell lying wholly on a darker stone.
	c := evenCanvas(3, 3, stoneInk, 0.9)
	for i := range c.pixels {
		c.pixels[i].face = 0.9
	}
	for y := 4; y < 8; y++ {
		for x := 2; x < 4; x++ {
			c.pixels[y*c.width+x].light = 0.9 * 0.72
		}
	}
	darker, lighter := Bold{}.cell(c, forestPalette[:], 1, 1), Bold{}.cell(c, forestPalette[:], 0, 1)
	if darker.paper != lighter.paper {
		t.Errorf("a darker stone lies on %v; want the wall's own shade, %v", darker.paper, lighter.paper)
	}
}

func TestBoldRimsWhatStandsInFront(t *testing.T) {
	// Two cells side by side: the left one's bark stands far in front of the right one's.
	c := evenCanvas(2, 1, 4, 1)
	for i := range c.pixels {
		if i%4 < 2 {
			c.pixels[i].depth = -5
		}
	}
	rim, inside := Bold{}.cell(c, forestPalette[:], 0, 0), Bold{}.cell(evenCanvas(1, 1, 4, 1), forestPalette[:], 0, 0)
	if brightnessOf(rim.paper) >= brightnessOf(inside.paper) {
		t.Errorf("bark at the silhouette lies on %v; want darker than bark within it, %v", rim.paper, inside.paper)
	}
}

func TestBoldFramesDrawBraille(t *testing.T) {
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	r.style = Bold{}
	frame := r.frame(100, 30, 0)
	if !strings.ContainsFunc(frame, func(glyph rune) bool { return glyph > 0x2800 && glyph <= 0x28ff }) {
		t.Error("a bold frame draws no braille dots")
	}
}

func TestClearDotsTheSecondMaterialInAMixedCell(t *testing.T) {
	cell := Clear{}.cell(oneCell(4, 1, 33, 1), forestPalette[:], 0, 0)
	if dots(cell) != 4 || cell.paper == cell.color {
		t.Errorf("bark over planks draws %d dots in %v on %v; want the half in one material dotted over the other", dots(cell), cell.color, cell.paper)
	}
}

func TestBoldLaysAWallFaceInOneFlatShade(t *testing.T) {
	_, c := closeView(cottage())
	papers := map[int]map[[3]int]bool{stoneInk: {}, shadedStoneInk: {}}
	for row := range c.height / 4 {
		for col := range c.width / 2 {
			block := boldBlock(c, col, row)
			if shades, stone := papers[block.ink]; stone && block.coverage == 8 && !block.rim {
				shades[Bold{}.cell(c, forestPalette[:], col, row).paper] = true
			}
		}
	}
	for ink, shades := range papers {
		t.Logf("ink %d lies on %d shades", ink, len(shades))
	}
	if len(papers[shadedStoneInk]) == 0 || len(papers[shadedStoneInk]) > 2 {
		t.Fatalf("the walls' shaded stone lies on %d shades; want each face flat in one", len(papers[shadedStoneInk]))
	}
}
