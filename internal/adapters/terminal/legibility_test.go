package terminal

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"lapresregne/internal/world"
)

type paintedGlyph struct {
	glyph                  rune
	row, column            int
	foreground, background [3]int
}

// terminalGlyphs reads the actual text, cursor positions and colors sent to the terminal.
func terminalGlyphs(frame string) []paintedGlyph {
	var result []paintedGlyph
	var foreground, background [3]int
	row, column := 1, 1
	for len(frame) > 0 {
		if strings.HasPrefix(frame, "\x1b[") {
			end := strings.IndexAny(frame[2:], "mHJK")
			if end < 0 {
				break
			}
			end += 2
			params := strings.Split(frame[2:end], ";")
			switch frame[end] {
			case 'H':
				row, column = 1, 1
				if len(params) > 0 && params[0] != "" {
					row, _ = strconv.Atoi(params[0])
				}
				if len(params) > 1 {
					column, _ = strconv.Atoi(params[1])
				}
			case 'm':
				var mode, r, g, b int
				if n, _ := fmt.Sscanf(frame[2:end], "%d;2;%d;%d;%d", &mode, &r, &g, &b); n == 4 {
					if mode == 38 {
						foreground = [3]int{r, g, b}
					}
					if mode == 48 {
						background = [3]int{r, g, b}
					}
				}
			}
			frame = frame[end+1:]
			continue
		}
		glyph, size := utf8.DecodeRuneInString(frame)
		frame = frame[size:]
		result = append(result, paintedGlyph{glyph, row, column, foreground, background})
		column++
	}
	return result
}

func isSolid(glyph rune) bool {
	return strings.ContainsRune("▘▝▀▖▌▞▛▗▚▐▜▄▙▟█", glyph)
}

func TestTheSceneryKeepsHeavyDitheringWithSolidColorUnderneath(t *testing.T) {
	glyphs := terminalGlyphs(newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne").frame(100, 38, 0))
	dots, solid, tinted := 0, 0, 0
	for _, p := range glyphs {
		if p.glyph > 0x2800 && p.glyph <= 0x28ff {
			dots++
			if p.background != background {
				tinted++
			}
		}
		if isSolid(p.glyph) {
			solid++
		}
	}
	if solid == 0 || dots < solid*20 || tinted < dots/4 {
		t.Fatalf("dithered cells %d, tinted cells %d, solid cells %d; want heavy dithering over colored scenery and a solid hero", dots, tinted, solid)
	}
}

func TestTheHeroKeepsTheirWorldScaleWhenZoomingOut(t *testing.T) {
	heights := make([]int, closeUpZoom+1)
	for step := range heights {
		w := meadow()
		top, bottom := 100, 0
		for _, p := range terminalGlyphs(newRenderer(w, NewCamera(step, 0), "L’Après-Règne").frame(100, 38, 0)) {
			if isSolid(p.glyph) {
				top, bottom = min(top, p.row), max(bottom, p.row)
			}
		}
		heights[step] = bottom - top + 1
		if heights[step] <= 0 {
			t.Fatalf("zoom %d shows no hero", step)
		}
	}
	if heights[0] > 2 || heights[closeUpZoom] < heights[0]*6 {
		t.Fatalf("hero heights %v; want the distant hero at natural scale and growing with the world", heights)
	}
}

func TestClothingKeepsDistinctColorsInTheRenderedView(t *testing.T) {
	for _, step := range []int{3, 6, closeUpZoom} {
		w := meadow()
		shirt, shorts, skin := false, false, false
		for _, p := range terminalGlyphs(newRenderer(w, NewCamera(step, 0), "L’Après-Règne").frame(100, 38, 0)) {
			if !isSolid(p.glyph) {
				continue
			}
			colors := [][3]int{p.foreground, p.background}
			if p.glyph == '█' {
				colors = colors[:1]
			}
			for _, color := range colors {
				r, g, b := color[0], color[1], color[2]
				shirt = shirt || r > 100 && r >= g && r-g < 12 && g > b && g-b < 25
				shorts = shorts || r > 50 && b > g && g > r
				skin = skin || r > 100 && g*10 > r*7 && g*10 < r*9 && b*10 < g*9
			}
		}
		if !shirt || !shorts || !skin {
			t.Fatalf("zoom %d: visible shirt %v, shorts %v, skin %v; want distinct clothing colors", step, shirt, shorts, skin)
		}
	}
}

func TestTheInterfaceRestoresItsBackgroundAfterDrawingColoredScenery(t *testing.T) {
	r := newRenderer(meadow(), NewCamera(0, 0), "L’Après-Règne")
	r.inventoryOpen = true
	for _, p := range terminalGlyphs(r.frame(100, 38, 0)) {
		if (p.row == 37 || p.row == 38 || p.row == 2 && p.column >= 100-panelWidth+1) && p.background != background {
			t.Fatalf("interface glyph %q at %d,%d inherits scenery background %v", p.glyph, p.row, p.column, p.background)
		}
	}
}

func TestTheAxeKeepsItsSteelColorWhenZoomedOut(t *testing.T) {
	w := besideTree()
	w.Chop()
	for _, step := range []int{0, 3, 6, closeUpZoom} {
		r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
		r.zoom.current = float64(step)
		metal := false
		for _, p := range terminalGlyphs(r.frame(100, 38, 0)) {
			if !isSolid(p.glyph) {
				continue
			}
			for _, color := range [][3]int{p.foreground, p.background} {
				red, green, blue := color[0], color[1], color[2]
				metal = metal || red > 150 && green > red && blue > green
			}
		}
		if !metal {
			t.Fatalf("zoom %d shows no steel-colored axe head", step)
		}
	}
}

func TestTheHeroRemainsVisibleWhileTurningTheDistantView(t *testing.T) {
	for _, facing := range []world.Facing{world.FacingDown, world.FacingUp, world.FacingLeft, world.FacingRight} {
		for _, turn := range []int{0, quarterTurn / 2, quarterTurn} {
			w := meadow()
			w.Hero().Face(facing)
			visible := false
			for _, p := range terminalGlyphs(newRenderer(w, cameraAt(0, turn), "L’Après-Règne").frame(100, 38, 0)) {
				visible = visible || isSolid(p.glyph)
			}
			if !visible {
				t.Fatalf("hero facing %v at camera turn %d is invisible", facing, turn)
			}
		}
	}
}
