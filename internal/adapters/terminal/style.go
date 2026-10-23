package terminal

import "math"

// Style turns the canvas pixels under one terminal cell into the cell drawn there, so the same
// forest can be drawn in different looks.
type Style interface {
	// Name is what the style is called on the command line and in the kept view.
	Name() string
	cell(c *forestCanvas, palette [][3]int, col, row int) forestCell
}

// Looks are the styles V switches between, and the one the world is drawn in now.
type Looks struct {
	styles  []Style
	current int
}

// NewLooks opens on the style with a name, as kept from an earlier session, or on the first style
// when none has that name.
func NewLooks(name string, styles ...Style) *Looks {
	l := &Looks{styles: styles}
	for i, style := range styles {
		if style.Name() == name {
			l.current = i
		}
	}
	return l
}

// Name is the name of the style the world is drawn in now.
func (l *Looks) Name() string { return l.style().Name() }

func (l *Looks) style() Style { return l.styles[l.current] }

// next switches to the following style, and from the last back to the first.
func (l *Looks) next() { l.current = (l.current + 1) % len(l.styles) }

// Dithered draws the scenery as braille dots in an ordered dither over a faint material paper, and
// the hero in solid quadrants.
type Dithered struct{}

func (Dithered) Name() string { return "dither" }

func (Dithered) cell(c *forestCanvas, palette [][3]int, col, row int) forestCell {
	return dottedCell(c, palette, col, row, ordered)
}

// ordered is the dither's threshold for the canvas pixel at x, y, from 0 to 1.
func ordered(x, y int) float64 { return (bayer[y%4][x%4] + 0.5) / 16 }

// dottedCell draws the cell as braille dots wherever a pixel's light passes the threshold at it,
// in the ink with most dots, over paper faintly tinted by the material that covers most of it.
func dottedCell(c *forestCanvas, palette [][3]int, col, row int, threshold func(x, y int) float64) forestCell {
	bits, ink := 0, 0
	// The cell's eight pixels hold at most eight inks: tally each one's dots, the pixels it
	// covers and their light.
	var tally [8]struct {
		ink, votes, coverage int
		light                float64
	}
	inks := 0
	figure := false
	for dy := range 4 {
		for dx := range 2 {
			x, y := col*2+dx, row*4+dy
			p := c.pixels[y*c.width+x]
			lit, dotted := p.light > 0, p.light > threshold(x, y)
			if !lit && !dotted {
				continue
			}
			at := 0
			for at < inks && tally[at].ink != p.ink {
				at++
			}
			if at == inks {
				tally[at].ink = p.ink
				inks++
			}
			if lit {
				tally[at].coverage++
				tally[at].light += p.light
				figure = figure || figureInk(p.ink)
			}
			if dotted {
				bits |= 1 << brailleDot[dy][dx]
				tally[at].votes++
			}
		}
	}
	if figure {
		return c.figureCell(col, row)
	}
	best := 0
	for _, counted := range tally[:inks] {
		// The hero, their axe and flying chips stay visible over anything else in the cell.
		score := counted.votes
		if foremost(counted.ink) && score > 0 {
			score += 64
		}
		if prominent(counted.ink) {
			score *= 2
		}
		// Ties go to the lower ink, as when every ink was weighed in turn.
		if score > best || score == best && score > 0 && counted.ink < ink {
			ink, best = counted.ink, score
		}
	}
	drawn := forestCell{bits: uint8(bits), ink: uint8(ink), paper: background}
	// A faint material color reinforces broad surfaces while the ordered dots carry their shading.
	for _, counted := range tally[:inks] {
		if material := counted.ink; counted.coverage >= 6 {
			strength := 0.18
			if prominent(material) || material == 2 || material == 3 || material == 4 || material >= 11 && material <= 16 {
				strength = 0.35
			}
			drawn.paper = mix(background, palette[material], min(1, strength*c.underlay()*counted.light/float64(counted.coverage)))
			break
		}
	}
	return drawn
}

// PixelArt draws the forest as square pixels of flat color, two to a cell: the upper half block
// in one, the paper under it in the other. Its shading follows gothic pixel art such as
// Blasphemous: a few flat shades per material, shadows sinking toward a cold violet and highlights
// lifting toward warm gold, and a dark outline where something stands in front of what lies
// behind it.
type PixelArt struct{}

func (PixelArt) Name() string { return "pixel" }

// pixelShades is how many flat shades of each material pixel art shows.
const pixelShades = 5

// shadowHue is the cold violet every material's darkest shade sinks toward, and lightHue the warm
// gold its brightest shade lifts toward.
var (
	shadowHue = [3]int{22, 14, 34}
	lightHue  = [3]int{255, 214, 150}
)

// fullShade is the shade at which a material shows its own color; brighter shades lift toward
// lightHue by up to highlightLift.
const (
	fullShade     = 0.8
	highlightLift = 0.25
)

// outlineGap is how far, in world units, an art pixel must stand in front of a neighbour to be
// outlined.
const outlineGap = 1.0

func (PixelArt) cell(c *forestCanvas, palette [][3]int, col, row int) forestCell {
	left, top := col*2, row*4
	return forestCell{bits: 3, solid: true, color: outlinedPixel(c, palette, left, top), paper: outlinedPixel(c, palette, left, top+2)}
}

// outlinedPixel is the color of the art pixel covering the 2 x 2 canvas pixels from left, top:
// its material's darkest shade where it stands well in front of a neighbour, and its banded shade
// anywhere else.
func outlinedPixel(c *forestCanvas, palette [][3]int, left, top int) [3]int {
	pixel := artPixel(c, left, top)
	if !pixel.lit {
		return background
	}
	shade := float64(int(pixel.light*pixelShades+0.5)) / pixelShades
	for _, step := range [4][2]int{{-2, 0}, {2, 0}, {0, -2}, {0, 2}} {
		x, y := left+step[0], top+step[1]
		if x < 0 || y < 0 || x+1 >= c.width || y+1 >= c.height {
			continue
		}
		if artPixel(c, x, y).depth-pixel.depth > outlineGap {
			shade = 1.0 / pixelShades
			break
		}
	}
	return ramp(palette[pixel.ink], shade)
}

// ramp is a material's color at a shade from 0 to 1: sinking from its own color toward shadowHue,
// faster in the darks, below fullShade, and lifting toward lightHue above it.
func ramp(color [3]int, shade float64) [3]int {
	if shade >= fullShade {
		return mix(color, lightHue, highlightLift*(shade-fullShade)/(1-fullShade))
	}
	return mix(shadowHue, color, math.Pow(shade/fullShade, 1.5))
}

// art is what one art pixel shows: the ink covering most of its canvas pixels, at their mean light,
// and the depth of its frontmost canvas pixel, lit or not.
type art struct {
	ink          int
	light, depth float64
	lit          bool
}

func artPixel(c *forestCanvas, left, top int) art {
	var tally [4]struct {
		ink, coverage int
		light         float64
	}
	inks := 0
	front := math.Inf(1)
	for y := top; y < top+2; y++ {
		for x := left; x < left+2; x++ {
			p := c.pixels[y*c.width+x]
			front = min(front, p.depth)
			if p.light <= 0 {
				continue
			}
			at := 0
			for at < inks && tally[at].ink != p.ink {
				at++
			}
			if at == inks {
				tally[at].ink = p.ink
				inks++
			}
			tally[at].coverage++
			tally[at].light += p.light
		}
	}
	if inks == 0 {
		return art{depth: front}
	}
	best, score := 0, 0
	for i, counted := range tally[:inks] {
		// The hero, their axe and flying chips stay visible over anything else, as in the dither.
		weight := counted.coverage
		if foremost(counted.ink) {
			weight += 64
		}
		if prominent(counted.ink) {
			weight *= 2
		}
		if weight > score {
			best, score = i, weight
		}
	}
	chosen := tally[best]
	return art{ink: chosen.ink, light: min(1, chosen.light/float64(chosen.coverage)), depth: front, lit: true}
}

// Stippled draws the world as a pointillist engraving: dark ink dots scattered over parchment,
// thickest where a material is dark or lies in shadow, an unbroken line of ink where something
// stands in front of what lies behind it, and the hero in dense ink.
type Stippled struct{}

func (Stippled) Name() string { return "stipple" }

// parchment is the paper the stipple is drawn on, and stippleInk the color of its dots.
var (
	parchment  = [3]int{150, 140, 116}
	stippleInk = [3]int{30, 25, 22}
)

// stippleWash is how far the paper under a cell darkens toward the ink as its dots thicken, so
// broad dark surfaces read as shade between the dots.
const stippleWash = 0.12

func (Stippled) cell(c *forestCanvas, palette [][3]int, col, row int) forestCell {
	bits := 0
	shade := 0.0
	for dy := range 4 {
		for dx := range 2 {
			x, y := col*2+dx, row*4+dy
			darkness := stippleDarkness(c, palette, x, y)
			shade += darkness
			if darkness > grain(x, y) {
				bits |= 1 << brailleDot[dy][dx]
			}
		}
	}
	return forestCell{bits: uint8(bits), inked: true, color: stippleInk, paper: mix(parchment, stippleInk, stippleWash*shade/8)}
}

// stippleDarkness is how likely the canvas pixel at x, y is to take a dot: none where nothing is
// drawn, all but certain on the hero and along an edge, and elsewhere the darkness of its
// material's color in its light.
func stippleDarkness(c *forestCanvas, palette [][3]int, x, y int) float64 {
	p := c.pixels[y*c.width+x]
	if p.light <= 0 {
		return 0
	}
	if figureInk(p.ink) {
		return 0.92
	}
	for _, step := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		nx, ny := x+step[0], y+step[1]
		if nx < 0 || ny < 0 || nx >= c.width || ny >= c.height {
			continue
		}
		if c.pixels[ny*c.width+nx].depth-p.depth > outlineGap {
			return 1
		}
	}
	color := palette[p.ink]
	value := p.light * float64(color[0]*3+color[1]*6+color[2]) / (10 * 255)
	// The eye reads stipple density on a log scale: spread values from the darkest shade to sunlit
	// leaves between the faintest grain and the thickest, then steepen the middle as an engraving
	// does, so shade gathers into dark masses and light opens onto paper.
	dark := 1 - min(1, math.Log(max(value, darkestValue)/darkestValue)/math.Log(palestValue/darkestValue))
	dark = dark * dark * (3 - 2*dark)
	return faintestStipple + (thickestStipple-faintestStipple)*dark
}

// darkestValue and palestValue bound the material values the stipple spreads its density over, and
// faintestStipple and thickestStipple are the densities at either end.
const (
	darkestValue    = 0.015
	palestValue     = 0.8
	faintestStipple = 0.04
	thickestStipple = 0.95
)

// grain is a scattered threshold from 0 to 1 for the canvas pixel at x, y, so dots fall unevenly
// as stippling does rather than in the dither's ordered pattern.
func grain(x, y int) float64 {
	h := uint32(x)*0x9e3779b1 ^ uint32(y)*0x85ebca77
	h ^= h >> 15
	h *= 0x2c1b3c6d
	h ^= h >> 12
	h *= 0x297a2d39
	h ^= h >> 15
	return float64(h) / (1 << 32)
}

// Etched draws the dither's colored dots on its tinted paper, jittered as stippling scatters them,
// and marks each edge where something stands in front of what lies behind it with a line glyph
// along the edge, in the color of what stands in front.
type Etched struct{}

func (Etched) Name() string { return "etch" }

// etchJitter is how much of the etch's dot threshold is scattered grain rather than ordered dither.
const etchJitter = 0.5

func (Etched) cell(c *forestCanvas, palette [][3]int, col, row int) forestCell {
	drawn := dottedCell(c, palette, col, row, jittered)
	if drawn.solid {
		return drawn
	}
	// Grass blades stand in front of the ground everywhere: marking them would hatch every meadow.
	if mark, ink := edgeMark(c, col, row); mark != 0 && ink != 1 && ink != shadowGrassInk {
		drawn.mark, drawn.ink = mark, uint8(ink)
	}
	return drawn
}

// jittered blends the ordered dither's threshold with scattered grain.
func jittered(x, y int) float64 { return (1-etchJitter)*ordered(x, y) + etchJitter*grain(x, y) }

// edgeMark is the line glyph along the edge crossing the cell, if one does, and the ink of what
// stands in front at it: | or a slash where the edge runs upright or aslant, and where it runs
// level, ‾, - or _ as it crosses the cell's head, middle or foot.
func edgeMark(c *forestCanvas, col, row int) (rune, int) {
	// across and down sum which way the canvas falls away behind the edge at each pixel it crosses.
	across, down := 0, 0
	crossed, level := 0, 0
	gap := c.etchGap()
	var inks [8]int
	var counts [8]int
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
			front := c.frontmost(x, y)
			at := 0
			for at < kinds && inks[at] != front {
				at++
			}
			if at == kinds {
				inks[at] = front
				kinds++
			}
			counts[at]++
		}
	}
	// A mark needs an edge that runs on through the cell: scattered steps among leaves and grass
	// tufts that turn every way are left to the dots.
	sideways, upDown := abs(across), abs(down)
	if sideways+upDown < max(etchCoherence, crossed) {
		return 0, 0
	}
	most := 0
	for at := range kinds {
		if counts[at] > counts[most] {
			most = at
		}
	}
	ink := inks[most]
	switch {
	// Steps sideways cross an upright edge, steps up or down a level one.
	case sideways > 2*upDown:
		return '|', ink
	case upDown > 2*sideways:
		// level is the sum of the crossed pixels' rows; their mean picks the glyph's height.
		switch mean := float64(level) / float64(crossed); {
		case mean < 1:
			return '‾', ink
		case mean > 2:
			return '_', ink
		default:
			return '-', ink
		}
	case across*down > 0:
		return '/', ink
	default:
		return '\\', ink
	}
}

// etchCoherence is how many steps of an edge through a cell must agree in their direction for the
// cell to be marked; and the agreeing steps must be at least as many as the pixels they cross.
const etchCoherence = 2

// etchPixels is how many canvas pixels of depth the edge of a mark must step back at least, so
// zoomed out, where each pixel spans more of the world, only tall things standing well in front
// are marked rather than every bulge of a crown.
const etchPixels = 8

// etchGap is how far, in world units, the canvas must step back for an edge to be marked.
func (c *forestCanvas) etchGap() float64 { return max(outlineGap, etchPixels/c.scale) }

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

// frontmost is the ink of the nearest pixel at or beside x, y.
func (c *forestCanvas) frontmost(x, y int) int {
	best := c.pixels[y*c.width+x]
	for _, step := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		nx, ny := x+step[0], y+step[1]
		if nx < 0 || ny < 0 || nx >= c.width || ny >= c.height {
			continue
		}
		if p := c.pixels[ny*c.width+nx]; p.light > 0 && p.depth < best.depth {
			best = p
		}
	}
	return best.ink
}

func abs(n int) int { return max(n, -n) }
