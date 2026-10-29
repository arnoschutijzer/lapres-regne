package terminal

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

// Etched draws the dither's colored dots on its tinted paper, jittered as stippling scatters them.
type Etched struct{}

func (Etched) Name() string { return "etch" }

// etchJitter is how much of the etch's dot threshold is scattered grain rather than ordered dither.
const etchJitter = 0.5

func (Etched) cell(c *forestCanvas, palette [][3]int, col, row int) forestCell {
	return dottedCell(c, palette, col, row, jittered)
}

// jittered blends the ordered dither's threshold with scattered grain.
func jittered(x, y int) float64 { return (1-etchJitter)*ordered(x, y) + etchJitter*grain(x, y) }

// Bold keeps the dither's ordered colored dots but lays whatever stands on the ground over flat
// blocks of its material's color in a few shades, and rims it in a darker shade where it stands in
// front of what lies behind it, so trees, walls and figures read as solid shapes with clear edges
// over the dotted ground.
type Bold struct{}

func (Bold) Name() string { return "bold" }

// boldPaper is how strongly the flat block under the dots shows its material's color in full light
// at the zoom where trees show their detail, strengthened farther out as the dither's paper is, up
// to boldStrongest; boldRim is how strongly its rim does, and boldCoverage how many of a cell's
// eight pixels a material must cover to lay its block.
const (
	boldPaper     = 0.55
	boldStrongest = 0.85
	boldRim       = 0.15
	boldCoverage  = 1
)

// boldFaintestGround is how strongly the ground's dots and paper show when zoomed all the way out,
// easing to full strength from the zoom where trees show their detail, so blocks stand out of a
// quiet ground when the view takes in a lot; boldDimmestShade is the dimmest shade a block takes
// there, so trees in deep shade still read as shapes.
const (
	boldFaintestGround = 0.4
	boldDimmestShade   = 0.5
)

// boldShades is how many flat shades of each material a block shows.
const boldShades = 5

// outlineGap is how far, in world units, a canvas pixel must stand in front of a neighbour for its
// block to be rimmed.
const outlineGap = 1.0

func (Bold) cell(c *forestCanvas, palette [][3]int, col, row int) forestCell {
	drawn := dottedCell(c, palette, col, row, ordered)
	if drawn.solid {
		return drawn
	}
	// How far the view is zoomed out, from 0 at the zoom where trees show their detail to 1 all the
	// way out.
	out := c.underlay() - 1
	block := boldBlock(c, col, row)
	if block.coverage < boldCoverage || groundInk(block.ink) {
		dot := palette[drawn.ink]
		if drawn.paper != background {
			dot = standOut(dot, drawn.paper)
		}
		quiet := 1 - (1-boldFaintestGround)*out
		drawn.inked = true
		drawn.color = mix(background, dot, quiet)
		drawn.paper = mix(background, drawn.paper, quiet)
		return drawn
	}
	// A few flat shades keep the block solid, never so dim that it sinks into the ground.
	shade := max(1, float64(int(block.light*boldShades+0.5))) / boldShades
	shade = max(shade, boldDimmestShade*out)
	strength := min(boldStrongest, boldPaper*c.underlay()) * shade
	if block.rim {
		strength = boldRim
	}
	drawn.paper = mix(background, palette[block.ink], strength)
	return drawn
}

// groundInk marks the inks of the ground and what lies flat on it: grass, water, paths, floors and
// their shadows, which stay dotted so what stands on them reads as a shape.
func groundInk(ink int) bool {
	switch ink {
	case 0, 1, 5, 6, 7, pathInk, floorInk, shadowGrassInk, shadowPathInk, shadowFloorInk, bankInk, boardsInk:
		return true
	}
	return false
}

// block is the material laid flat under a cell's dots: the one covering most of its pixels,
// prominent ones counting double, at their mean light, and whether any of those pixels stands well
// in front of a neighbour. A face's pixels count the light falling on the face, so the stonework's
// mortar and darker stones leave the shade to the dots.
type block struct {
	ink, coverage int
	light         float64
	rim           bool
}

func boldBlock(c *forestCanvas, col, row int) block {
	var tally [8]block
	inks := 0
	for dy := range 4 {
		for dx := range 2 {
			x, y := col*2+dx, row*4+dy
			p := c.pixels[y*c.width+x]
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
			if p.face > 0 {
				tally[at].light += p.face
			} else {
				tally[at].light += p.light
			}
			tally[at].rim = tally[at].rim || standsInFront(c, x, y)
		}
	}
	best, score := block{}, 0
	for _, counted := range tally[:inks] {
		weight := counted.coverage
		if prominent(counted.ink) {
			weight *= 2
		}
		if weight > score {
			best, score = counted, weight
		}
	}
	if best.coverage > 0 {
		best.light = min(1, best.light/float64(best.coverage))
	}
	return best
}

// standsInFront tells whether the canvas pixel at x, y stands well in front of a neighbour.
func standsInFront(c *forestCanvas, x, y int) bool {
	depth := c.pixels[y*c.width+x].depth
	for _, step := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		nx, ny := x+step[0], y+step[1]
		if nx < 0 || ny < 0 || nx >= c.width || ny >= c.height {
			continue
		}
		if c.pixels[ny*c.width+nx].depth-depth > outlineGap {
			return true
		}
	}
	return false
}

// solidSpeck is how many unlit pixels within a cell its surface fills rather than dotting them.
const solidSpeck = 2

// twoTone draws a cell whose eight pixels show colors, listed row by row, in flat color: the most
// common color fills the cell and the next is dotted over it.
func twoTone(colors [8][3]int) forestCell {
	var tally [8]struct {
		color [3]int
		count int
	}
	kinds := 0
	for _, color := range colors {
		at := 0
		for at < kinds && tally[at].color != color {
			at++
		}
		if at == kinds {
			tally[at].color = color
			kinds++
		}
		tally[at].count++
	}
	// The most common color fills the cell and the next is dotted over it; any other color takes
	// whichever of the two lies nearer.
	first, second := 0, -1
	for i := 1; i < kinds; i++ {
		if tally[i].count > tally[first].count {
			first = i
		}
	}
	for i := range kinds {
		if i != first && (second < 0 || tally[i].count > tally[second].count) {
			second = i
		}
	}
	drawn := forestCell{inked: true, paper: tally[first].color, color: tally[first].color}
	// A pixel or two left unlit within a surface would show as black specks, so the surface fills
	// them.
	if second < 0 || tally[second].color == background && tally[second].count <= solidSpeck {
		return drawn
	}
	drawn.color = tally[second].color
	for dy := range 4 {
		for dx := range 2 {
			color := colors[dy*2+dx]
			if colorDistance(color, drawn.color) < colorDistance(color, drawn.paper) {
				drawn.bits |= 1 << brailleDot[dy][dx]
			}
		}
	}
	return drawn
}
