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
