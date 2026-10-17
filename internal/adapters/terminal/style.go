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
// in one, the paper under it in the other.
type PixelArt struct{}

func (PixelArt) Name() string { return "pixel" }

// pixelShades is how many flat shades of each material pixel art shows.
const pixelShades = 6

func (PixelArt) cell(c *forestCanvas, palette [][3]int, col, row int) forestCell {
	top := artPixel(c, palette, col*2, row*4)
	bottom := artPixel(c, palette, col*2, row*4+2)
	return forestCell{bits: 3, solid: true, color: top, paper: bottom}
}

// artPixel is the color of one square art pixel, covering the 2 x 2 canvas pixels from left, top: the
// ink covering most of them, at their mean light rounded to a flat shade.
func artPixel(c *forestCanvas, palette [][3]int, left, top int) [3]int {
	var tally [4]struct {
		ink, coverage int
		light         float64
	}
	inks := 0
	for y := top; y < top+2; y++ {
		for x := left; x < left+2; x++ {
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
			tally[at].light += p.light
		}
	}
	if inks == 0 {
		return background
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
	light := min(1, chosen.light/float64(chosen.coverage))
	shade := float64(int(light*pixelShades+0.5)) / pixelShades
	return mix(background, palette[chosen.ink], shade)
}
