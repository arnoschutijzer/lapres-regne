package terminal

import "math"

// Lines draws the world as the clear style does and marks each edge where something stands well in
// front of what lies behind it with a line glyph along the edge, | / \ ‾ - or _, so shapes keep
// their outlines when the view takes in a lot.
type Lines struct{}

func (Lines) Name() string { return "lines" }

func (Lines) cell(c *forestCanvas, palette [][3]int, col, row int) forestCell {
	drawn := Clear{}.cell(c, palette, col, row)
	if drawn.solid {
		return drawn
	}
	mark, x, y := edgeMark(c, col, row)
	if mark == 0 {
		return drawn
	}
	// Grass blades stand in front of the ground everywhere: marking them would hatch every meadow.
	if front := c.pixels[y*c.width+x].ink; front == 1 || front == shadowGrassInk {
		return drawn
	}
	originX, originY := c.patternOrigin()
	front := mix(clearTone(c, palette, x, y, x-originX, y-originY), background, c.vignette(col, row))
	drawn.mark, drawn.color = mark, lineColor(front, drawn.paper)
	return drawn
}

// lineColor is the color of a line along an edge, drawn on paper, for what stands in front at it:
// its hue at a value pushed further from the paper's where the two already differ, and where they do
// not, as where it fills the cell, the paper's own color darkened, or lightened on dark paper where
// a darker line would vanish.
func lineColor(front, paper [3]int) [3]int {
	value := lumaFraction(paper)
	switch {
	case math.Abs(lumaFraction(front)-value) >= lineApart:
		pushed := lumaFraction(front) + linePush*(lumaFraction(front)-value)
		return withValue(front, min(lineBrightest, max(lineDarkest, pushed)))
	case value < lineDarkPaper:
		return withValue(paper, value+lineLift)
	}
	return mix(paper, background, lineDark)
}

// lineApart is how far apart in value what stands in front and the paper must lie for the line to
// take the front's color; linePush is how much further from the paper the line is pushed. Paper
// darker than lineDarkPaper takes a line lineLift lighter; any other a line lineDark of the way
// toward the background.
const (
	lineApart     = 0.04
	linePush      = 0.5
	lineDarkPaper = 0.15
	lineLift      = 0.08
	lineDark      = 0.6
)

// lineDarkest and lineBrightest bound the value a line pushed from its paper takes.
const (
	lineDarkest   = 0.02
	lineBrightest = 0.8
)

// edgeMark is the line glyph along the edge crossing the cell, if one does, and the canvas pixel of
// what stands in front at it: | or a slash where the edge runs upright or aslant, and where it runs
// level, ‾, - or _ as it crosses the cell's head, middle or foot.
func edgeMark(c *forestCanvas, col, row int) (mark rune, frontX, frontY int) {
	// across and down sum which way the canvas falls away behind the edge at each pixel it crosses.
	across, down := 0, 0
	crossed, level := 0, 0
	gap := c.lineGap()
	var fronts [8][2]int
	var counts [8]int
	var inks [8]int
	kinds := 0
	for dy := range 4 {
		for dx := range 2 {
			x, y := col*2+dx, row*4+dy
			sx, sy := c.stepBehind(x-1, y, x+1, y, gap), c.stepBehind(x, y-1, x, y+1, gap)
			if sx == 0 && sy == 0 {
				continue
			}
			across, down = across+sx, down+sy
			crossed++
			level += dy
			fx, fy := c.frontmost(x, y)
			ink := c.pixels[fy*c.width+fx].ink
			at := 0
			for at < kinds && inks[at] != ink {
				at++
			}
			if at == kinds {
				inks[at], fronts[at] = ink, [2]int{fx, fy}
				kinds++
			}
			counts[at]++
		}
	}
	// A mark needs an edge that runs on through the cell: scattered steps among leaves and grass
	// tufts that turn every way are left to the dots.
	sideways, upDown := abs(across), abs(down)
	if crossed == 0 || sideways+upDown < max(lineCoherence, crossed) {
		return 0, 0, 0
	}
	most := 0
	for at := range kinds {
		if counts[at] > counts[most] {
			most = at
		}
	}
	frontX, frontY = fronts[most][0], fronts[most][1]
	switch {
	// Steps sideways cross an upright edge, steps up or down a level one.
	case sideways > 2*upDown:
		return '|', frontX, frontY
	case upDown > 2*sideways:
		// level is the sum of the crossed pixels' rows; their mean picks the glyph's height.
		switch mean := float64(level) / float64(crossed); {
		case mean < 1:
			return '‾', frontX, frontY
		case mean > 2:
			return '_', frontX, frontY
		default:
			return '-', frontX, frontY
		}
	case across*down > 0:
		return '/', frontX, frontY
	default:
		return '\\', frontX, frontY
	}
}

// lineCoherence is how many steps of an edge through a cell must agree in their direction for the
// cell to be marked; and the agreeing steps must be at least as many as the pixels they cross.
const lineCoherence = 2

// linePixels is how many canvas pixels of depth the edge of a mark must step back at least, so
// zoomed out, where each pixel spans more of the world, only tall things standing well in front
// are marked rather than every bulge of a crown.
const linePixels = 8

// lineGap is how far, in world units, the canvas must step back for an edge to be marked.
func (c *forestCanvas) lineGap() float64 { return max(outlineGap, linePixels/c.scale) }

// stepBehind is 1 when the canvas at x2, y2 lies more than gap behind that at x1, y1, -1 when it
// lies more than gap in front, and 0 otherwise or past the canvas's sides.
func (c *forestCanvas) stepBehind(x1, y1, x2, y2 int, gap float64) int {
	if x1 < 0 || y1 < 0 || x2 >= c.width || y2 >= c.height {
		return 0
	}
	near, far := c.pixels[y1*c.width+x1], c.pixels[y2*c.width+x2]
	switch step := far.depth - near.depth; {
	case near.light <= 0 || far.light <= 0:
		return 0
	case step > gap:
		return 1
	case step < -gap:
		return -1
	}
	return 0
}

// frontmost is the canvas pixel nearest the viewer at or beside x, y.
func (c *forestCanvas) frontmost(x, y int) (int, int) {
	bestX, bestY := x, y
	best := c.pixels[y*c.width+x]
	for _, step := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		nx, ny := x+step[0], y+step[1]
		if nx < 0 || ny < 0 || nx >= c.width || ny >= c.height {
			continue
		}
		if p := c.pixels[ny*c.width+nx]; p.light > 0 && p.depth < best.depth {
			bestX, bestY, best = nx, ny, p
		}
	}
	return bestX, bestY
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
