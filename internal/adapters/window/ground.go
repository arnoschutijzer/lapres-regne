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

// soil is what a pixel of ground shows, for deciding what may lie on it: its material, and which
// of the material's tones it shows. Only the forest floor and its moss take litter.
type soil struct {
	material tones
	tone     int
	open     bool
}

// sunward points toward the light, which falls from the upper left and a little from behind.
var sunward = func() [3]float64 {
	x, y, z := -0.5, 0.8, -0.35
	length := math.Sqrt(x*x + y*y + z*z)
	return [3]float64{x / length, y / length, z / length}
}()

// relief is how much brighter a slope is than flat ground, from its rise across and toward the
// viewer for each unit along: slopes turned to the light catch it, those turned away fall dark.
func relief(riseX, riseZ float64) float64 {
	length := math.Sqrt(riseX*riseX + 1 + riseZ*riseZ)
	lit := (-riseX*sunward[0] + sunward[1] - riseZ*sunward[2]) / length
	return lit/sunward[1] - 1
}

// groundColor is the color of the ground at a world point, shown at the art pixel px, py and
// lifted by light tones brighter, and its soil. The floor is dark earth patched with moss, darker under the
// canopy, its tones dithered into each other.
func groundColor(w *world.World, x, z float64, px, py int, light float64) (color.RGBA, soil) {
	paint := func(material tones, value float64, open bool) (color.RGBA, soil) {
		value += light
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
// leaves rotting in drifts, snapped twigs, the odd stone, and rarely a blood-red flower. Cells
// are counted across and toward the viewer over the ground, as if it were flat.
func decoration(w *world.World, cx, cy int) (stamp, bool) {
	x, z := cellPoint(cx, cy)
	roll := hash(cx, cy, 51)
	open := 1 - w.Cover(x, z)
	tufts := 0.02
	if w.Noise(x/3, z/3, 61) > 0.58 {
		tufts = 0.4
	}
	leaves := 0.006
	if w.Noise(x/5, z/5, 62) > 0.66 {
		leaves = 0.2
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

// cellPoint is the world point where a cell's stamp lies, somewhere within the cell.
func cellPoint(cx, cy int) (x, z float64) {
	return (float64(cx*decorationAcross) + hash(cx, cy, 54)*decorationAcross) / unit,
		(float64(cy*decorationDown) + hash(cx, cy, 55)*decorationDown) / (unit * depthSquash)
}

// groundChunk is one painted chunk of ground, and the world point each of its pixels shows: x
// follows from the pixel's column, z is kept, NaN where no ground shows.
type groundChunk struct {
	image *image.RGBA
	z     []float32
}

// pointAt is the world point of the ground a pixel of the chunk shows, counted from its top left.
func (c *groundChunk) pointAt(left, x, y int) (wx, wz float64, ok bool) {
	z := c.z[y*chunkSize+x]
	if math.IsNaN(float64(z)) {
		return 0, 0, false
	}
	return (float64(left+x) + 0.5) / unit, float64(z), true
}

// ledgeFace is the color of a ledge's face at the art pixel px, py, down from its lip at 0 to its
// foot at 1: a lip of moss hanging over bare earth laced with roots, sinking into the dark at the
// foot.
func ledgeFace(px, py int, down float64) color.RGBA {
	if down < 0.08 {
		return mossTones.dithered(3.6, px, py)
	}
	value := 4.2 - down*3.6
	if hash(px, py/3, 91) > 0.82 {
		// Roots hang down the face.
		value -= 1.4
	}
	if hash(px/2, py/2, 92) > 0.94 {
		// Stones jut from the earth and catch the light.
		return stoneTones.dithered(3.5-down*2, px, py)
	}
	return faceTones.dithered(value, px, py)
}

// faceLevel marks a pixel showing a ledge's face rather than the top of a ledge.
const faceLevel = math.MinInt16

// shadowReach is how many pixels across a ledge's shadow falls onto the lower ground beside it.
const shadowReach = 5

// drawLedgeEdges draws where one ledge meets another across the screen or beyond it, as the light
// from the left shows them: a ledge stepping down to the right casts a shadow onto the ground
// below it, one stepping up catches the light along its lip, and one dropping away from the viewer
// catches it along its crest.
func drawLedgeEdges(image *image.RGBA, levels []int16, left, top int) {
	at := func(x, y int) int16 { return levels[y*chunkSize+x] }
	darken := func(x, y int, strength float64) {
		if (bayer[(top+y)&3][(left+x)&3]+0.5)/16 < strength {
			c := image.RGBAAt(x, y)
			image.SetRGBA(x, y, color.RGBA{c.R / 3, c.G / 3, c.B / 3, 255})
		}
	}
	brighten := func(x, y int) {
		c := image.RGBAAt(x, y)
		image.SetRGBA(x, y, color.RGBA{uint8(min(255, int(c.R)*5/3+6)), uint8(min(255, int(c.G)*5/3+6)), uint8(min(255, int(c.B)*5/3+4)), 255})
	}
	for y := range chunkSize {
		for x := range chunkSize - 1 {
			here, right := at(x, y), at(x+1, y)
			if here == faceLevel || right == faceLevel || here == right {
				continue
			}
			if here > right {
				// The ground steps down to the right: the higher ledge shades the lower.
				for reach := 1; reach <= shadowReach && x+reach < chunkSize && at(x+reach, y) == right; reach++ {
					darken(x+reach, y, 1-float64(reach-1)/shadowReach)
				}
			} else {
				// The ground steps up to the right: its lit face shows as a bright lip.
				brighten(x+1, y)
			}
		}
	}
	for y := 1; y < chunkSize; y++ {
		for x := range chunkSize {
			here, beyond := at(x, y), at(x, y-1)
			if here != faceLevel && beyond != faceLevel && here > beyond {
				// The ledge drops away from the viewer: its crest catches the light.
				brighten(x, y)
			}
		}
	}
}

// crestSteps is how many steps of ground a rise must hide before its crest is drawn as one: half
// a pixel each, so grazing slopes do not stripe with crests.
const crestSteps = 8

// paintChunk paints the ground of one chunk, counted in chunks from the world's origin. Each
// column is marched from the nearest ground that could show in it toward the farthest, and each
// step paints only the ground that rises on the screen above what nearer steps painted, so nearer
// hills hide whatever lies behind them.
func paintChunk(w *world.World, cx, cy int) *groundChunk {
	chunk := &groundChunk{image.NewRGBA(image.Rect(0, 0, chunkSize, chunkSize)), make([]float32, chunkSize*chunkSize)}
	for i := range chunk.z {
		chunk.z[i] = float32(math.NaN())
	}
	left, top := cx*chunkSize, cy*chunkSize
	open := make([]bool, chunkSize*chunkSize)
	// levels holds the ledge each pixel's ground lies on, so the edges between ledges can be drawn.
	levels := make([]int16, chunkSize*chunkSize)
	toward := unit * depthSquash
	step := 0.5 / toward
	near := (float64(top+chunkSize)+w.Highest()*lift)/toward + step
	far := float64(top)/toward - step
	highest := math.Max(1, w.Highest())
	for col := range chunkSize {
		x := (float64(left+col) + 0.5) / unit
		painted := top + chunkSize
		lastRaw := w.Elevation(x, near+step)
		// hidden counts the steps since the last that showed: ground behind a rise. ridgeTop is the
		// topmost pixel of the last stretch shown, where a rise crests.
		hidden, ridgeTop := 0, -1
		var ridgeX, ridgeZ, ridgeLight float64
		for z := near; z >= far && painted > top; z -= step {
			raw := w.Elevation(x, z)
			h := terraced(raw)
			_, py := project(x, z, h)
			rise := int(math.Ceil(py - 0.5))
			if rise >= painted {
				// Ground below the chunk's bottom edge is not hidden, only not yet reached.
				if ridgeTop >= 0 {
					hidden++
				}
				lastRaw = raw
				continue
			}
			if level(raw) > level(lastRaw) {
				// The ground steps up to the next ledge here: its face rises toward the viewer.
				for y := max(top, rise); y < painted; y++ {
					chunk.image.SetRGBA(col, y-top, ledgeFace(left+col, y, float64(y-rise)/float64(max(1, painted-rise))))
					chunk.z[(y-top)*chunkSize+col] = float32(z)
					levels[(y-top)*chunkSize+col] = faceLevel
				}
				painted = max(top, rise)
				ridgeTop, hidden, lastRaw = -1, 0, raw
				continue
			}
			riseX := (w.Elevation(x+0.25, z) - raw) / 0.25 * heightScale
			riseZ := (lastRaw - raw) / step * heightScale
			// Slopes turned to the light catch it, and high ground stands lighter than the
			// valleys, which sink into the dark.
			light := relief(riseX, riseZ)*2 + (raw/highest-0.5)*1.4
			crest := hidden >= crestSteps
			if crest && ridgeTop >= top {
				// A rise crests here and hides the ground behind it: its crest catches the light.
				c, _ := groundColor(w, ridgeX, ridgeZ, left+col, ridgeTop, ridgeLight+1.2)
				chunk.image.SetRGBA(col, ridgeTop-top, c)
			}
			for y := max(top, rise); y < painted; y++ {
				shade := light
				if crest && y >= painted-2 {
					// Just behind a crest the ground lies in its shadow.
					shade -= 1.5
				}
				c, under := groundColor(w, x, z, left+col, y, shade)
				chunk.image.SetRGBA(col, y-top, c)
				chunk.z[(y-top)*chunkSize+col] = float32(z)
				open[(y-top)*chunkSize+col] = under.open
				levels[(y-top)*chunkSize+col] = int16(level(raw))
			}
			painted = max(top, rise)
			ridgeTop, ridgeX, ridgeZ, ridgeLight = painted, x, z, light
			hidden = 0
			lastRaw = raw
		}
	}
	drawLedgeEdges(chunk.image, levels, left, top)
	// Stamps reaching in from cells beside the chunk are laid too, so none is cut at its edge;
	// each lies where its point of the ground shows, if a nearer rise does not hide it.
	const reach = 8
	nearCell := floorDiv(top+chunkSize+int(w.Highest()*lift)+reach, decorationDown)
	for cellY := floorDiv(top-reach, decorationDown); cellY <= nearCell; cellY++ {
		for cellX := floorDiv(left-reach, decorationAcross); cellX <= floorDiv(left+chunkSize, decorationAcross); cellX++ {
			s, ok := decoration(w, cellX, cellY)
			if !ok {
				continue
			}
			x, z := cellPoint(cellX, cellY)
			px, py := project(x, z, terraced(w.Elevation(x, z)))
			footX, footY := int(math.Floor(px))-left, int(math.Floor(py))-top
			if footX < -reach || footY < 0 || footX >= chunkSize+reach || footY >= chunkSize+reach {
				continue
			}
			if footX >= 0 && footX < chunkSize && footY < chunkSize {
				if shown := chunk.z[footY*chunkSize+footX]; math.IsNaN(float64(shown)) || math.Abs(float64(shown)-z) > 0.4 {
					continue
				}
			}
			_, under := groundColor(w, x, z, left+footX, top+footY, 0)
			if !under.open {
				continue
			}
			for row, line := range s.rows {
				for col := range line {
					x := footX - s.width()/2 + col
					y := footY - s.height() + 1 + row
					if x < 0 || y < 0 || x >= chunkSize || y >= chunkSize || !open[y*chunkSize+x] {
						continue
					}
					switch mark := line[col]; {
					case mark == '.':
					case mark >= '0' && mark <= '9':
						chunk.image.SetRGBA(x, y, under.material.at(under.tone+int(mark-'0')-3))
					default:
						chunk.image.SetRGBA(x, y, s.legend[mark])
					}
				}
			}
		}
	}
	return chunk
}
