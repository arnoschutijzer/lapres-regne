package window

import (
	"image"
	"image/color"
	"math"

	"lapresregne/internal/world"
)

// chunkSize is how many art pixels across and down one chunk of painted ground spans. The ground
// never changes, so each chunk is painted once and kept while it is near the camera.
const chunkSize = 128

// The ground is laid out in the world's own art pixels: a world point x, z lies at the pixel
// x*unit across and z*unit*depthSquash down, whatever the camera shows.
func pixelToWorld(px, py int) (x, z float64) {
	return (float64(px) + 0.5) / unit, (float64(py) + 0.5) / (unit * depthSquash)
}

// soil is what a pixel of ground shows, for deciding what may lie on it: its material, and which
// of the material's tones it shows. Only the forest floor and its moss take litter.
type soil struct {
	material tones
	tone     int
	open     bool
}

// groundPixel is the color of the ground at one of the world's art pixels, and its soil. The floor
// is dark earth patched with moss, darker under the canopy, its tones dithered into each other.
func groundPixel(w *world.World, px, py int) (color.RGBA, soil) {
	x, z := pixelToWorld(px, py)
	paint := func(material tones, value float64, open bool) (color.RGBA, soil) {
		c := material.dithered(value, px, py)
		tone := min(len(material)-1, max(0, int(math.Round(value))))
		return c, soil{material, tone, open}
	}
	if w.Mapped() {
		ground, ok := w.GroundAt(x, z)
		if !ok {
			return voidColor, soil{}
		}
		switch ground {
		case world.Water:
			return water(px, py, 0.5), soil{}
		case world.Path:
			return paint(pathTones, 1.2+w.Noise(x/3, z/3, 12)*2.2, false)
		case world.Floor, world.Door:
			return flagstone(px, py), soil{}
		case world.Boards:
			return board(px, py), soil{}
		}
	} else if distance, width := w.River(x, z); distance < width {
		return water(px, py, 1-distance/width), soil{}
	} else if distance < width+0.7 {
		return paint(bankTones, 1+w.Noise(x/2, z/2, 14)*2*(distance-width)/0.7+0.8, false)
	}
	shade := 0.58 - w.Cover(x, z)*0.55 + (w.Noise(x/7, z/7, 3)-0.5)*0.5 + (w.Noise(x/2, z/2, 8)-0.5)*0.25
	// Moss spreads in patches where the canopy lets some light through; between moss and bare
	// earth the dither mixes the two.
	moss := (w.Noise(x/9, z/9, 21) - 0.5) * 8
	if threshold := (bayer[py&3][px&3] + 0.5) / 16; moss > threshold*2-1 {
		return paint(mossTones, shade*float64(len(mossTones)), true)
	}
	return paint(floorTones, shade*float64(len(floorTones)), true)
}

// water is the color of murky water at a pixel, deeper toward the middle, from 0 at the bank to 1
// in the middle, with thin ripples catching what little light there is.
func water(px, py int, depth float64) color.RGBA {
	if py%4 == 0 {
		if h := hash(px/7, py/4, 31); h > 0.985 {
			return waterTones.at(5)
		} else if h > 0.85 {
			return waterTones.at(4)
		}
	}
	return waterTones.dithered(3.4-depth*2.6, px, py)
}

// flagstone is a floor of staggered, worn flagstones, lit along their upper edges.
func flagstone(px, py int) color.RGBA {
	const across, down = 12, 7
	row := floorDiv(py, down)
	px += row % 2 * across / 2
	x, y := mod(px, across), mod(py, down)
	switch {
	case x == 0 || y == 0:
		return stoneTones.at(0)
	case y == 1:
		return stoneTones.at(3)
	}
	// Each stone is worn unevenly, and cracks run through a few.
	if hash(px, py, 42) > 0.96 {
		return stoneTones.at(1)
	}
	return stoneTones.dithered(1.6+hash(floorDiv(px, across), row, 41)*1.2, px, py)
}

// board is a floor of wooden boards running across, with a dark seam between each.
func board(px, py int) color.RGBA {
	const down = 5
	y := mod(py, down)
	if y == 0 {
		return boardTones.at(0)
	}
	if hash(px/9, floorDiv(py, down), 43) > 0.85 {
		return boardTones.at(1)
	}
	return boardTones.dithered(1.5+float64(y)/4, px, py)
}

func floorDiv(a, b int) int { return int(math.Floor(float64(a) / float64(b))) }
func mod(a, b int) int      { return a - floorDiv(a, b)*b }

// stamp is a small hand-drawn picture laid onto the ground. Digits in its rows are tones of the
// soil it lies on, relative to the soil's own, 3 being that tone; letters take their colors from
// legend; dots are clear.
type stamp struct {
	rows   []string
	legend map[byte]color.RGBA
}

var (
	// Withered tufts of grass stand up pale from the moss.
	tallTuft = stamp{rows: []string{
		"..4....",
		".45.4..",
		".4.44.5",
		"4.343.4",
		"2323232",
	}}
	lowTuft = stamp{rows: []string{
		"4..4.",
		".44.4",
		"23432",
	}}
	// Fallen leaves rot in drifts, and twigs lie snapped across the floor.
	litter = stamp{rows: []string{
		".a..b.",
		"abb.aa",
		".a.ba.",
	}, legend: map[byte]color.RGBA{'a': litterTones[0], 'b': litterTones[1]}}
	twig = stamp{rows: []string{
		"......aa",
		"...aab..",
		".aab....",
		"ab......",
	}, legend: map[byte]color.RGBA{'a': barkTones[1], 'b': barkTones[3]}}
	pebble = stamp{rows: []string{
		".cb.",
		"cbba",
		".aa.",
	}, legend: map[byte]color.RGBA{'a': stoneTones[1], 'b': stoneTones[2], 'c': stoneTones[4]}}
	crimsonFlower = stamp{rows: []string{
		".a.",
		"aba",
		".a.",
		".2.",
	}, legend: map[byte]color.RGBA{'a': crimsonTones[1], 'b': crimsonTones[2]}}
)

// width and height are a stamp's size in pixels.
func (s stamp) width() int  { return len(s.rows[0]) }
func (s stamp) height() int { return len(s.rows) }

// decorationCell is the size of the cells, in art pixels, that each hold at most one stamp.
const decorationAcross, decorationDown = 7, 5

// decoration picks the stamp, if any, lying in one cell of the ground: withered tufts in patches,
// leaves rotting in drifts, snapped twigs, the odd stone, and rarely a blood-red flower.
func decoration(w *world.World, cx, cy int) (stamp, bool) {
	x, z := pixelToWorld(cx*decorationAcross, cy*decorationDown)
	roll := hash(cx, cy, 51)
	open := 1 - w.Cover(x, z)
	tufts := 0.02
	if w.Noise(x/3, z/3, 61) > 0.58 {
		tufts = 0.4
	}
	leaves := 0.02
	if w.Noise(x/5, z/5, 62) > 0.6 {
		leaves = 0.35
	}
	switch {
	case roll < 0.003*open:
		return crimsonFlower, true
	case roll < 0.012:
		return pebble, true
	case roll < 0.03:
		return twig, true
	case roll < 0.03+leaves:
		return litter, true
	case roll < 0.03+leaves+tufts*open:
		if hash(cx, cy, 53) < 0.4 {
			return tallTuft, true
		}
		return lowTuft, true
	}
	return stamp{}, false
}

// paintChunk paints the ground of one chunk, counted in chunks from the world's origin.
func paintChunk(w *world.World, cx, cy int) *image.RGBA {
	chunk := image.NewRGBA(image.Rect(0, 0, chunkSize, chunkSize))
	left, top := cx*chunkSize, cy*chunkSize
	open := make([]bool, chunkSize*chunkSize)
	for y := range chunkSize {
		for x := range chunkSize {
			c, under := groundPixel(w, left+x, top+y)
			chunk.SetRGBA(x, y, c)
			open[y*chunkSize+x] = under.open
		}
	}
	// Stamps reaching in from cells beside the chunk are laid too, so none is cut at its edge.
	const reach = 8
	for cellY := floorDiv(top-reach, decorationDown); cellY <= floorDiv(top+chunkSize, decorationDown); cellY++ {
		for cellX := floorDiv(left-reach, decorationAcross); cellX <= floorDiv(left+chunkSize, decorationAcross); cellX++ {
			s, ok := decoration(w, cellX, cellY)
			if !ok {
				continue
			}
			// The stamp lies on the pixel at its bottom middle, somewhere in its cell.
			footX := cellX*decorationAcross + int(hash(cellX, cellY, 54)*decorationAcross)
			footY := cellY*decorationDown + int(hash(cellX, cellY, 55)*decorationDown)
			_, under := groundPixel(w, footX, footY)
			if !under.open {
				continue
			}
			for row, line := range s.rows {
				for col := range line {
					x := footX - s.width()/2 + col - left
					y := footY - s.height() + 1 + row - top
					if x < 0 || y < 0 || x >= chunkSize || y >= chunkSize || !open[y*chunkSize+x] {
						continue
					}
					switch mark := line[col]; {
					case mark == '.':
					case mark >= '0' && mark <= '9':
						chunk.SetRGBA(x, y, under.material.at(under.tone+int(mark-'0')-3))
					default:
						chunk.SetRGBA(x, y, s.legend[mark])
					}
				}
			}
		}
	}
	return chunk
}
