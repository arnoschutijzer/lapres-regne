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
			lit, dotted := p.light > 0, p.light > (bayer[y%4][x%4]+0.5)/16
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
