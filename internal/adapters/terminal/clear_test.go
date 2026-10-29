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
