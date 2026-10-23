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

func TestPixelArtDrawsTwoSquarePixelsInEachCell(t *testing.T) {
	palette := forestPalette[:]
	cell := PixelArt{}.cell(oneCell(2, 1, 3, 1), palette, 0, 0)
	if !cell.solid || quadrants[cell.bits] != '▀' {
		t.Fatalf("cell draws %q, solid %v; want an upper half block", quadrants[cell.bits], cell.solid)
	}
	if cell.color == cell.paper {
		t.Errorf("both pixels show %v; want each its own ink", cell.color)
	}
	unlit := PixelArt{}.cell(oneCell(2, 0, 3, 0), palette, 0, 0)
	if unlit.color != background || unlit.paper != background {
		t.Errorf("unlit pixels show %v and %v; want the background %v", unlit.color, unlit.paper, background)
	}
}

func TestPixelArtShadowsTurnColdAndHighlightsWarm(t *testing.T) {
	shadow := PixelArt{}.cell(oneCell(4, 0.2, 4, 0.2), forestPalette[:], 0, 0).color
	if shadow[2] <= shadow[1] {
		t.Errorf("bark in shadow = %v; want it sunk toward violet, bluer than green", shadow)
	}
	iron := forestPalette[32]
	highlight := PixelArt{}.cell(oneCell(32, 1, 32, 1), forestPalette[:], 0, 0).color
	if highlight[0]-highlight[2] <= iron[0]-iron[2] {
		t.Errorf("iron in full light = %v; want it warmer than its own %v", highlight, iron)
	}
}

func TestPixelArtOutlinesWhatStandsInFrontOfTheGround(t *testing.T) {
	// Two cells side by side: the left one's ink stands far in front of the right one's.
	c := &forestCanvas{width: 4, height: 4, pixels: make([]forestPixel, 16)}
	for i := range c.pixels {
		c.pixels[i] = forestPixel{ink: 4, light: 1, depth: -5}
		if i%4 >= 2 {
			c.pixels[i] = forestPixel{ink: 0, light: 1, depth: 0}
		}
	}
	outlined := PixelArt{}.cell(c, forestPalette[:], 0, 0)
	inside := PixelArt{}.cell(oneCell(4, 1, 4, 1), forestPalette[:], 0, 0)
	if brightnessOf(outlined.color) >= brightnessOf(inside.color) {
		t.Errorf("bark at the silhouette = %v; want darker than bark within it, %v", outlined.color, inside.color)
	}
	ground := PixelArt{}.cell(c, forestPalette[:], 1, 0)
	if open := (PixelArt{}).cell(oneCell(0, 1, 0, 1), forestPalette[:], 0, 0); ground.color != open.color {
		t.Errorf("ground behind the bark = %v; want it drawn without an outline", ground.color)
	}
}

func brightnessOf(color [3]int) int { return color[0] + color[1] + color[2] }

func TestPixelArtShadesInFlatBands(t *testing.T) {
	palette := forestPalette[:]
	darker := PixelArt{}.cell(oneCell(2, 0.50, 2, 0.20), palette, 0, 0)
	lighter := PixelArt{}.cell(oneCell(2, 0.53, 2, 0.80), palette, 0, 0)
	if darker.color != lighter.color {
		t.Errorf("nearly equal light gives %v and %v; want one flat shade", darker.color, lighter.color)
	}
	if darker.paper == lighter.paper {
		t.Errorf("light 0.2 and 0.8 both give %v; want different shades", darker.paper)
	}
}

func TestPixelArtFramesDrawNoBraille(t *testing.T) {
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	r.style = PixelArt{}
	frame := r.frame(100, 30, 0)
	if strings.ContainsFunc(frame, func(glyph rune) bool { return glyph >= 0x2800 && glyph <= 0x28ff }) {
		t.Error("a pixel art frame draws braille dots")
	}
	if !strings.ContainsRune(frame, '▀') {
		t.Error("a pixel art frame draws no half blocks")
	}
}

func TestDitheredFramesDrawBraille(t *testing.T) {
	frame := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne").frame(100, 30, 0)
	if !strings.ContainsFunc(frame, func(glyph rune) bool { return glyph > 0x2800 && glyph <= 0x28ff }) {
		t.Error("a dithered frame draws no braille dots")
	}
}

func TestLooksStartOnTheNamedStyle(t *testing.T) {
	looks := NewLooks("dither", PixelArt{}, Dithered{})
	if looks.Name() != "dither" {
		t.Fatalf("looks open on %q; want the named dither", looks.Name())
	}
	if _, ok := looks.style().(Dithered); !ok {
		t.Fatalf("looks draw in %T; want Dithered", looks.style())
	}
}

func TestLooksStartOnTheFirstStyleForAnUnknownName(t *testing.T) {
	if looks := NewLooks("", PixelArt{}, Dithered{}); looks.Name() != "pixel" {
		t.Fatalf("looks open on %q; want the first style, pixel", looks.Name())
	}
}

func TestLooksCycleThroughEveryStyle(t *testing.T) {
	looks := NewLooks("pixel", PixelArt{}, Dithered{})
	looks.next()
	if looks.Name() != "dither" {
		t.Fatalf("after one switch looks show %q; want dither", looks.Name())
	}
	looks.next()
	if looks.Name() != "pixel" {
		t.Fatalf("after two switches looks show %q; want pixel again", looks.Name())
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

// stippledDots counts the dots stippled over every cell of a canvas.
func stippledDots(c *forestCanvas) int {
	count := 0
	for row := range c.height / 4 {
		for col := range c.width / 2 {
			count += dots(Stippled{}.cell(c, forestPalette[:], col, row))
		}
	}
	return count
}

func TestStippleScattersMoreInkInShadowThanInLight(t *testing.T) {
	shadow, light := stippledDots(evenCanvas(8, 8, 1, 0.2)), stippledDots(evenCanvas(8, 8, 1, 0.9))
	if shadow <= light {
		t.Errorf("grass in shadow takes %d dots and in light %d; want more in shadow", shadow, light)
	}
}

func TestStippleScattersMoreInkOverDarkMaterials(t *testing.T) {
	ground, leaves := stippledDots(evenCanvas(8, 8, 0, 0.8)), stippledDots(evenCanvas(8, 8, 12, 0.8))
	if ground <= leaves {
		t.Errorf("dark ground takes %d dots and pale willow leaves %d in the same light; want more over the ground", ground, leaves)
	}
}

func TestStippleScattersDotsUnevenly(t *testing.T) {
	c := evenCanvas(8, 8, 1, 0.5)
	first := Stippled{}.cell(c, forestPalette[:], 0, 0).bits
	for col := 1; col < 8; col++ {
		if (Stippled{}).cell(c, forestPalette[:], col, 0).bits != first {
			return
		}
	}
	t.Errorf("every cell of an even surface draws the dots %08b; want them scattered", first)
}

func TestStippleDrawsDarkInkOnParchment(t *testing.T) {
	cell := Stippled{}.cell(oneCell(1, 0.4, 1, 0.4), forestPalette[:], 0, 0)
	if !cell.inked {
		t.Fatal("stippled dots take the palette's ink; want their own")
	}
	if brightnessOf(cell.color) >= brightnessOf(cell.paper) {
		t.Errorf("dots %v on paper %v; want dark ink on lighter paper", cell.color, cell.paper)
	}
	if cell.paper == background {
		t.Errorf("stippled paper is the terminal background; want parchment")
	}
}

func TestStippleLeavesBareParchmentWhereNothingIsDrawn(t *testing.T) {
	if cell := (Stippled{}).cell(oneCell(1, 0, 1, 0), forestPalette[:], 0, 0); cell.bits != 0 || cell.paper != parchment {
		t.Errorf("an empty cell draws dots %08b on %v; want bare parchment", cell.bits, cell.paper)
	}
}

func TestStippleInksTheEdgeOfWhatStandsInFront(t *testing.T) {
	// Two cells side by side, lit brightly: the left one's ink stands far in front of the right one's.
	c := evenCanvas(2, 1, 1, 1)
	for i := range c.pixels {
		if i%4 < 2 {
			c.pixels[i].depth = -5
		}
	}
	edge, open := Stippled{}.cell(c, forestPalette[:], 0, 0), Stippled{}.cell(evenCanvas(1, 1, 1, 1), forestPalette[:], 0, 0)
	if dots(edge) <= dots(open) {
		t.Errorf("the edge takes %d dots and open grass %d; want a line of ink along the edge", dots(edge), dots(open))
	}
}

func TestStippleDrawsTheHeroInDenseInk(t *testing.T) {
	if hero := stippledDots(evenCanvas(4, 4, defaultLook.skin, 1)); hero < 4*4*6 {
		t.Errorf("the hero in full light takes %d of %d dots; want them nearly solid ink", hero, 4*4*8)
	}
}

func TestStippledFramesDrawBrailleOnParchment(t *testing.T) {
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	r.style = Stippled{}
	frame := r.frame(100, 30, 0)
	if !strings.ContainsFunc(frame, func(glyph rune) bool { return glyph > 0x2800 && glyph <= 0x28ff }) {
		t.Error("a stippled frame draws no braille dots")
	}
	for _, cell := range r.cells {
		if brightnessOf(cell.paper) < brightnessOf(parchment)*3/4 {
			t.Fatalf("a stippled frame lays paper %v; want parchment near %v", cell.paper, parchment)
		}
	}
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

func TestBoldKeepsTheGroundDotted(t *testing.T) {
	c := evenCanvas(3, 3, 1, 0.8)
	if bold, dither := (Bold{}).cell(c, forestPalette[:], 1, 1), (Dithered{}).cell(c, forestPalette[:], 1, 1); bold != dither {
		t.Errorf("bold grass draws %+v; want it as the dither draws it, %+v", bold, dither)
	}
}

func TestBoldShadesBlocksInFlatBands(t *testing.T) {
	darker, lighter := Bold{}.cell(oneCell(4, 0.62, 4, 0.62), forestPalette[:], 0, 0), Bold{}.cell(oneCell(4, 0.66, 4, 0.66), forestPalette[:], 0, 0)
	if darker.paper != lighter.paper {
		t.Errorf("nearly equal light lays %v and %v; want one flat shade", darker.paper, lighter.paper)
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
