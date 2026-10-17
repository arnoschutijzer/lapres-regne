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
	cell := PixelArt{}.cell(oneCell(2, 1, 3, 0), palette, 0, 0)
	if !cell.solid || quadrants[cell.bits] != '▀' {
		t.Fatalf("cell draws %q, solid %v; want an upper half block", quadrants[cell.bits], cell.solid)
	}
	if cell.color != forestPalette[2] {
		t.Errorf("upper pixel = %v; want the fully lit ink %v", cell.color, forestPalette[2])
	}
	if cell.paper != background {
		t.Errorf("lower pixel = %v; want the background %v where nothing is lit", cell.paper, background)
	}
}

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
