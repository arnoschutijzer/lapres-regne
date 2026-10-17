package terminal

// Style turns the canvas pixels under one terminal cell into the cell drawn there, so the same
// forest can be drawn in different looks.
type Style interface {
	cell(c *forestCanvas, palette [][3]int, col, row int) forestCell
}

// Dithered draws the scenery as braille dots in an ordered dither over a faint material paper, and
// the hero in solid quadrants.
type Dithered struct{}

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
