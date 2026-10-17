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

// surface is what kind of ground a pixel shows, for deciding what may grow on it.
type surface byte

const (
	bareSurface surface = iota
	grassSurface
)

// groundPixel is the color of the ground at one of the world's art pixels, and its surface. Grass
// falls into flat fields of tone, darker under the canopy, with clean edges between them.
func groundPixel(w *world.World, px, py int) (color.RGBA, surface, int) {
	x, z := pixelToWorld(px, py)
	if w.Mapped() {
		ground, ok := w.GroundAt(x, z)
		if !ok {
			return voidColor, bareSurface, 0
		}
		switch ground {
		case world.Water:
			return water(px, py, 0.5), bareSurface, 0
		case world.Path:
			return pathTones.at(1 + int(w.Noise(x/3, z/3, 12)*3)), bareSurface, 0
		case world.Floor, world.Door:
			return flagstone(px, py), bareSurface, 0
		case world.Boards:
			return board(px, py), bareSurface, 0
		}
	} else if distance, width := w.River(x, z); distance < width {
		return water(px, py, 1-distance/width), bareSurface, 0
	} else if distance < width+0.6 {
		return bankTones.at(int(w.Noise(x/2, z/2, 14) * 3)), bareSurface, 0
	}
	shade := 0.62 - w.Cover(x, z)*0.6 + (w.Noise(x/7, z/7, 3)-0.5)*0.55 + (w.Noise(x/2.5, z/2.5, 8)-0.5)*0.18
	tone := int(math.Floor(shade * float64(len(grassTones))))
	return grassTones.at(tone), grassSurface, min(len(grassTones)-1, max(0, tone))
}

// water is the color of water at a pixel, deeper toward the middle, from 0 at the bank to 1 in the
// middle, with thin ripples catching the light.
func water(px, py int, depth float64) color.RGBA {
	tone := 2
	if depth > 0.6 {
		tone = 0
	} else if depth > 0.3 {
		tone = 1
	}
	if py%4 == 0 {
		if h := hash(px/7, py/4, 31); h > 0.97 {
			return waterTones.at(4)
		} else if h > 0.8 {
			return waterTones.at(tone + 1)
		}
	}
	return waterTones.at(tone)
}

// flagstone is a floor of staggered flagstones, lit along their upper edges.
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
	return stoneTones.at(2 - int(hash(floorDiv(px, across), row, 41)*1.5))
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
	return boardTones.at(2 + y/4)
}

func floorDiv(a, b int) int { return int(math.Floor(float64(a) / float64(b))) }
func mod(a, b int) int      { return a - floorDiv(a, b)*b }

// stamp is a small hand-drawn picture laid onto the ground. Digits in its rows are grass tones
// relative to the grass it grows on, 3 being that grass; letters take their colors from legend;
// dots are clear.
type stamp struct {
	rows   []string
	legend map[byte]color.RGBA
}

var (
	tallTuft = stamp{rows: []string{
		"...4...",
		".4.4.4.",
		".4.3.4.",
		"4.343.4",
		"3434343",
	}}
	lowTuft = stamp{rows: []string{
		".4..4.",
		"4.44.4",
		"343343",
	}}
	crimsonFlower = stamp{rows: []string{
		".a.",
		"aba",
		".a.",
		".4.",
	}, legend: map[byte]color.RGBA{'a': crimsonTones[1], 'b': crimsonTones[2]}}
	goldFlower = stamp{rows: []string{
		"a.a",
		".b.",
		"a4a",
	}, legend: map[byte]color.RGBA{'a': goldTones[1], 'b': goldTones[2]}}
	paleFlower = stamp{rows: []string{
		".b.",
		"bab",
		".4.",
	}, legend: map[byte]color.RGBA{'a': paleTones[1], 'b': paleTones[2]}}
	pebble = stamp{rows: []string{
		".cb.",
		"cbba",
		".aa.",
	}, legend: map[byte]color.RGBA{'a': stoneTones[1], 'b': stoneTones[2], 'c': stoneTones[3]}}
)

// width and height are a stamp's size in pixels.
func (s stamp) width() int  { return len(s.rows[0]) }
func (s stamp) height() int { return len(s.rows) }

// decorationCell is the size of the cells, in art pixels, that each hold at most one stamp.
const decorationAcross, decorationDown = 7, 5

// decoration picks the stamp, if any, growing in one cell of the ground: tufts in thick patches,
// wildflowers in drifts of one kind across open clearings, and the odd pebble.
func decoration(w *world.World, cx, cy int) (stamp, bool) {
	x, z := pixelToWorld(cx*decorationAcross, cy*decorationDown)
	roll := hash(cx, cy, 51)
	open := 1 - w.Cover(x, z)
	tufts := 0.02
	if w.Noise(x/3, z/3, 61) > 0.58 {
		tufts = 0.45
	}
	flowers := 0.0
	if drift := w.Noise(x/5, z/5, 62); drift > 0.66 {
		flowers = 0.25 * open
	}
	switch {
	case roll < flowers:
		// One drift holds one kind of flower.
		return [...]stamp{crimsonFlower, goldFlower, paleFlower}[int(w.Noise(x/12, z/12, 63)*2.99)], true
	case roll < flowers+0.004:
		return pebble, true
	case roll < flowers+0.004+tufts*open:
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
	tones := make([]int, chunkSize*chunkSize)
	surfaces := make([]surface, chunkSize*chunkSize)
	for y := range chunkSize {
		for x := range chunkSize {
			c, kind, tone := groundPixel(w, left+x, top+y)
			chunk.SetRGBA(x, y, c)
			tones[y*chunkSize+x], surfaces[y*chunkSize+x] = tone, kind
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
			// The stamp stands on the pixel at its bottom middle, somewhere in its cell.
			footX := cellX*decorationAcross + int(hash(cellX, cellY, 54)*decorationAcross)
			footY := cellY*decorationDown + int(hash(cellX, cellY, 55)*decorationDown)
			_, kind, tone := groundPixel(w, footX, footY)
			if kind != grassSurface {
				continue
			}
			for row, line := range s.rows {
				for col := range line {
					x := footX - s.width()/2 + col - left
					y := footY - s.height() + 1 + row - top
					if x < 0 || y < 0 || x >= chunkSize || y >= chunkSize || surfaces[y*chunkSize+x] != grassSurface {
						continue
					}
					switch mark := line[col]; {
					case mark == '.':
					case mark >= '0' && mark <= '9':
						chunk.SetRGBA(x, y, grassTones.at(tone+int(mark-'0')-3))
					default:
						chunk.SetRGBA(x, y, s.legend[mark])
					}
				}
			}
		}
	}
	return chunk
}
