package terminal

import (
	"strings"
	"testing"

	"lapresregne/internal/world"
)

// standing is a canvas cols by rows cells wide and high, lit evenly by one ink, where the pixels
// for which front holds stand far in front of the rest.
func standing(cols, rows, ink int, front func(x, y int) bool) *forestCanvas {
	c := evenCanvas(cols, rows, ink, 0.6)
	c.scale = detailScale
	for y := range c.height {
		for x := range c.width {
			if front(x, y) {
				c.pixels[y*c.width+x].depth = -5
			}
		}
	}
	return c
}

func TestLinesMarksAnEdgeWithAGlyphAlongIt(t *testing.T) {
	cases := []struct {
		name  string
		front func(x, y int) bool
		want  rune
	}{
		{"a left side", func(x, y int) bool { return x < 4 }, '|'},
		{"an edge along the cell's foot", func(x, y int) bool { return y < 8 }, '_'},
		{"an edge across the cell's middle", func(x, y int) bool { return y < 6 }, '-'},
		{"an edge along the cell's head", func(x, y int) bool { return y >= 4 }, '‾'},
		{"a rising slope", func(x, y int) bool { return x+y < 8 }, '/'},
		{"a falling slope", func(x, y int) bool { return y < x+3 }, '\\'},
	}
	for _, tc := range cases {
		cell := Lines{}.cell(standing(3, 3, 4, tc.front), forestPalette[:], 1, 1)
		if cell.mark != tc.want {
			t.Errorf("%s draws %q; want %q", tc.name, cell.mark, tc.want)
		}
	}
}

func TestLinesDrawsTheEdgeApartFromThePaper(t *testing.T) {
	cell := Lines{}.cell(standing(3, 3, 4, func(x, y int) bool { return x < 4 }), forestPalette[:], 1, 1)
	if difference := lumaOf(cell.color) - lumaOf(cell.paper); difference > -0.04 && difference < 0.04 {
		t.Errorf("the edge's line shows %v over paper %v; want it clearly lighter or darker", cell.color, cell.paper)
	}
}

func TestLinesLeavesGrassBladesToTheDots(t *testing.T) {
	c := standing(3, 3, 1, func(x, y int) bool { return x < 4 })
	if cell := (Lines{}).cell(c, forestPalette[:], 1, 1); cell.mark != 0 {
		t.Errorf("grass standing in front of grass draws %q; want no mark", cell.mark)
	}
}

func TestLinesDrawsCellsAwayFromEdgesAsTheClearStyleDoes(t *testing.T) {
	c := standing(6, 6, stoneInk, func(x, y int) bool { return x < 4 })
	if lined, clear := (Lines{}).cell(c, forestPalette[:], 4, 3), (Clear{}).cell(c, forestPalette[:], 4, 3); lined != clear {
		t.Errorf("a cell away from any edge draws %+v; want it as the clear style draws it, %+v", lined, clear)
	}
}

func TestLinesFramesDrawGlyphsAmongTheDots(t *testing.T) {
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	r.style = Lines{}
	frame := r.frame(120, 30, 0)
	if !strings.ContainsFunc(frame, func(glyph rune) bool { return glyph > 0x2800 && glyph <= 0x28ff }) {
		t.Error("a lined frame draws no braille dots")
	}
	// The forest fills the rows between the title and the hints.
	lines := 0
	for _, painted := range terminalGlyphs(frame) {
		if painted.row >= 2 && painted.row <= 28 && strings.ContainsRune("/\\|-_‾", painted.glyph) {
			lines++
		}
	}
	if lines == 0 {
		t.Error("a lined frame draws no edge glyphs over the forest")
	}
	if !strings.Contains(frame, "V style: lines") {
		t.Error("the hint line does not name the lines style")
	}
}
