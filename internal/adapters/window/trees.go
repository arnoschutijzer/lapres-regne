package window

import (
	"image"
	"image/color"
	"math"

	"lapresregne/internal/world"
)

// leafPattern is a hand-drawn tile of leaves: + marks a leaf turned up to the light, - its shaded
// underside, and dots leave the crown's shading as it is. Laid over a crown it only shows where
// one band of shade meets the next, breaking the edge between them into leaves.
var leafPattern = []string{
	"..++......",
	".++++-....",
	"++++--..+.",
	".++--..+++",
	"..--..++++",
	"......++--",
	"..+....--.",
	".+++......",
}

// leaf is the pattern's push on a crown's shading at a pixel, each row of tiles shifted along.
func leaf(x, y int) float64 {
	rows := len(leafPattern)
	tileRow := floorDiv(y, rows)
	line := leafPattern[mod(y, rows)]
	switch line[mod(x+tileRow*4, len(line))] {
	case '+':
		return 1
	case '-':
		return -1
	}
	return 0
}

// stumpRows draw a felled tree's stump: letters are the pale wood of the cut, digits its bark.
var stumpRows = []string{
	".aaaaa.",
	"abbcbba",
	"aabbbaa",
	"3322211",
	"3322211",
	"2.1.1.0",
}

// lay draws rows of a hand-drawn picture with its top left at left, top. Digits take colors from
// digit, and letters from legend; dots are clear.
func lay(sprite *image.RGBA, rows []string, left, top int, digit func(int) color.RGBA, legend map[byte]color.RGBA) {
	for y, line := range rows {
		for x := range line {
			at := image.Pt(left+x, top+y)
			if !at.In(sprite.Bounds()) {
				continue
			}
			switch mark := line[x]; {
			case mark == '.':
			case mark >= '0' && mark <= '9':
				sprite.SetRGBA(at.X, at.Y, digit(int(mark-'0')))
			default:
				sprite.SetRGBA(at.X, at.Y, legend[mark])
			}
		}
	}
}

// treeSprite draws a tree of a kind, crownWidth pixels across its crown and height pixels from the
// top of its crown to its foot, which is at the bottom centre.
func treeSprite(kind world.TreeKind, crownWidth, height int, variation float64) *image.RGBA {
	width := crownWidth + 4
	// A crown taller than the tree reaches above it, so the sprite grows to hold it.
	crownDown := float64(crownWidth) / 2 * 0.78
	if kind == world.Yew {
		crownDown = float64(crownWidth) / 2 * 0.8 * 1.3
	}
	if kind != world.DeadTree {
		grown := int(math.Ceil(trunkShows(height)+crownDown*2*1.07)) + 2
		height = max(height, grown)
	}
	sprite := image.NewRGBA(image.Rect(0, 0, width, height+1))
	centre := width / 2
	if kind == world.DeadTree {
		drawTrunk(sprite, centre, height/5, height, max(4, crownWidth/6), deadBarkTones)
		drawBranches(sprite, float64(centre), float64(height)/5+2, float64(height)*0.5, variation)
		return sprite
	}
	// The crown is a few round lobes over the trunk, wider than tall seen from above; a yew's
	// stands taller and narrower. Each lobe is shaded in flat bands lit from the upper left.
	across := float64(crownWidth) / 2
	down := across * 0.78
	if kind == world.Yew {
		across *= 0.8
		down = across * 1.3
	}
	middleY := float64(height) - trunkShows(height) - down
	drawTrunk(sprite, centre, int(middleY), height, max(4, crownWidth/8), barkTones)
	leaves := canopyTones[kind]
	// The lobes lie unevenly, some large and some straggling at the edge, so no two crowns share a
	// shape, and their edges are torn into ragged clumps of leaves.
	seed := uint64(variation*977) + uint64(kind)*31
	type lobe struct{ x, y, r, phase float64 }
	size := math.Min(across, down)
	lobes := []lobe{{float64(centre), middleY, size * 0.55, variation * 9}}
	count := 7 + int(hash(1, 2, seed)*4)
	for i := range count {
		angle := hash(i, 3, seed) * 2 * math.Pi
		reach := 0.35 + 0.4*hash(i, 4, seed)
		r := size * (0.26 + 0.26*hash(i, 5, seed))
		lobes = append(lobes, lobe{
			float64(centre) + math.Cos(angle)*(across-r)*reach/0.75,
			middleY + math.Sin(angle)*(down-r)*reach/0.75,
			r,
			hash(i, 6, seed) * 6,
		})
	}
	light := [3]float64{-0.5, -0.62, 0.6}
	bounds := sprite.Bounds()
	inCrown := make([]bool, bounds.Dx()*bounds.Dy())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nearest, shade, rim := math.Inf(1), 0.0, false
			for _, l := range lobes {
				dx, dy := float64(x)+0.5-l.x, float64(y)+0.5-l.y
				angle := math.Atan2(dy, dx)
				ragged := hash(int(math.Floor((angle+math.Pi)/(2*math.Pi)*22)), int(l.phase*10), seed)
				r := l.r * (1 + 0.08*math.Sin(angle*5+l.phase) + 0.2*(ragged-0.5))
				d := (dx*dx + dy*dy) / (r * r)
				if d > 1 {
					continue
				}
				// The lobe lowest on the screen stands nearest the viewer and covers the others.
				nz := math.Sqrt(1 - d)
				if order := -(l.y + nz*l.r*0.3); order < nearest {
					nearest = order
					shade = (dx/r)*light[0] + (dy/r)*light[1] + nz*light[2]
					rim = d > 0.8 && dy > 0
				}
			}
			if math.IsInf(nearest, 1) {
				continue
			}
			inCrown[y*bounds.Dx()+x] = true
			value := (shade+0.22+leaf(x, y)*0.2)*4.2 + 0.2
			if rim {
				// The underside of each lobe sinks into the dark between it and the next.
				value = math.Min(value, 1)
			}
			// Gaps in the leaves open onto the dark inside the crown.
			if shade < 0.35 && hash(x/3, y/2, seed+7) < 0.07 {
				value = 0
			}
			sprite.SetRGBA(x, y, leaves.dithered(value, x, y))
		}
	}
	// Boughs reach out from the trunk, seen through the gaps in the leaves and below them.
	drawBoughs(sprite, centre, middleY, across, down, seed, func(x, y int) bool {
		if !inCrown[y*bounds.Dx()+x] {
			return y > int(middleY)
		}
		c := sprite.RGBAAt(x, y)
		return c == leaves.at(0) || c == leaves.at(1)
	})
	if kind == world.Willow {
		drawStrands(sprite, inCrown, leaves)
	}
	return sprite
}

// trunkShows is how much of a tree's trunk shows below its crown, for a tree height pixels tall.
func trunkShows(height int) float64 { return float64(height) * 0.22 }

// drawTrunk draws a trunk from top down to its foot at the bottom, width pixels wide where it
// meets the ground and narrowing above, twisting as it rises, lit from the left and clawing into
// the ground with its roots.
func drawTrunk(sprite *image.RGBA, centre, top, foot, width int, bark tones) {
	twist := hash(centre, foot, 13) * 6
	for y := top; y <= foot; y++ {
		rise := float64(foot-y) / float64(max(1, foot-top))
		across := max(2, int(math.Round(float64(width)*(1-0.35*rise))))
		if y >= foot-1 {
			across += 2
		}
		shift := int(math.Round(math.Sin(float64(y)*0.13+twist) * 1.3 * rise))
		left := centre - across/2 + shift
		for x := left; x < left+across; x++ {
			// Light falls from the left, and furrows run up the bark.
			value := 3.2 - float64(x-left)/float64(across)*2.4
			if hash(x-shift, y/4, 17) > 0.72 {
				value -= 1.2
			}
			if x == left || x == left+across-1 {
				value = math.Min(value, 0.8)
			}
			sprite.SetRGBA(x, y, bark.dithered(value, x, y))
		}
	}
	// Roots spread from the foot, sinking into the ground as they reach out.
	for i, reach := range [...]float64{-1, 1, -0.6, 0.7} {
		length := float64(width) * (0.9 + hash(i, centre, 14)*0.8)
		for step := 0.0; step < length; step += 0.5 {
			x := centre + int(math.Round(reach*(float64(width)/2+step)))
			y := foot - 1 + int(step*0.25) + i/2
			if image.Pt(x, y).In(sprite.Bounds()) {
				sprite.SetRGBA(x, y, bark.dithered(2-step/length*1.5, x, y))
			}
		}
	}
}

// drawBoughs draws the boughs reaching up and out from the top of the trunk into the crown, where
// shows tells that a pixel is not covered by leaves.
func drawBoughs(sprite *image.RGBA, centre int, middleY, across, down float64, seed uint64, shows func(x, y int) bool) {
	var bough func(x, y, angle, length float64, depth int)
	bough = func(x, y, angle, length float64, depth int) {
		for step := 0.0; step < length; step += 0.5 {
			px, py := int(x+math.Cos(angle)*step), int(y+math.Sin(angle)*step)
			for thick := 0; thick < 3-depth; thick++ {
				at := image.Pt(px+thick, py)
				if at.In(sprite.Bounds()) && shows(at.X, at.Y) {
					sprite.SetRGBA(at.X, at.Y, barkTones.at(1+min(thick, 1)))
				}
			}
		}
		if depth == 2 {
			return
		}
		endX, endY := x+math.Cos(angle)*length, y+math.Sin(angle)*length
		bough(endX, endY, angle-0.5, length*0.6, depth+1)
		bough(endX, endY, angle+0.45, length*0.55, depth+1)
	}
	base := middleY + down*0.6
	for i := range 3 {
		angle := -math.Pi/2 + (float64(i)-1)*0.75 + (hash(i, 9, seed)-0.5)*0.4
		bough(float64(centre), base, angle, across*0.45, 0)
	}
}

// drawBranches draws a dead tree's bare limbs, forking as they reach up from its trunk.
func drawBranches(sprite *image.RGBA, centre, from, reach, variation float64) {
	var limb func(x, y, angle, length float64, depth int)
	limb = func(x, y, angle, length float64, depth int) {
		for step := 0.0; step < length; step += 0.5 {
			px, py := int(x+math.Cos(angle)*step), int(y+math.Sin(angle)*step)
			if !image.Pt(px, py).In(sprite.Bounds()) {
				continue
			}
			sprite.SetRGBA(px, py, deadBarkTones.at(2))
			if depth < 2 && image.Pt(px+1, py).In(sprite.Bounds()) {
				sprite.SetRGBA(px+1, py, deadBarkTones.at(1))
			}
			if image.Pt(px-1, py).In(sprite.Bounds()) && hash(px, py, 19) > 0.6 {
				sprite.SetRGBA(px-1, py, deadBarkTones.at(4))
			}
		}
		if depth == 3 {
			return
		}
		endX, endY := x+math.Cos(angle)*length, y+math.Sin(angle)*length
		spread := 0.45 + variation*0.2
		limb(endX, endY, angle-spread, length*0.62, depth+1)
		limb(endX, endY, angle+spread*0.8, length*0.58, depth+1)
	}
	limb(centre, from+reach*0.3, -math.Pi/2-0.5, reach*0.35, 1)
	limb(centre, from+reach*0.15, -math.Pi/2+0.45, reach*0.4, 1)
	limb(centre, from, -math.Pi/2+(variation-0.5)*0.3, reach*0.3, 0)
}

// drawStrands hangs a willow's long strands of leaves from the lower edge of its crown, where
// inCrown marks the crown's pixels row by row.
func drawStrands(sprite *image.RGBA, inCrown []bool, leaves tones) {
	bounds := sprite.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		if hash(x, 0, 29) < 0.35 {
			continue
		}
		edge := -1
		for y := bounds.Max.Y - 1; y >= bounds.Min.Y; y-- {
			if inCrown[y*bounds.Dx()+x] {
				edge = y
				break
			}
		}
		if edge < 0 {
			continue
		}
		length := 6 + int(hash(x, 1, 29)*12)
		for y := edge - 2; y < edge+length && y < bounds.Max.Y; y++ {
			value := 2.6 - float64(y-edge)/float64(length)*2
			sprite.SetRGBA(x, y, leaves.dithered(value, x, y))
		}
	}
}

// stumpSprite draws a felled tree's stump.
func stumpSprite() *image.RGBA {
	sprite := image.NewRGBA(image.Rect(0, 0, len(stumpRows[0]), len(stumpRows)))
	lay(sprite, stumpRows, 0, 0, barkTones.at, map[byte]color.RGBA{'a': woodTones[1], 'b': woodTones[2], 'c': woodTones[0]})
	return sprite
}

// shadowSprite is the flat shadow an ellipse across wide pixels and down tall casts on the ground.
func shadowSprite(across, down int) *image.RGBA {
	sprite := image.NewRGBA(image.Rect(0, 0, across, down))
	rx, ry := float64(across)/2, float64(down)/2
	for y := range down {
		for x := range across {
			dx, dy := (float64(x)+0.5-rx)/rx, (float64(y)+0.5-ry)/ry
			// The shadow's edge breaks up through the dither rather than ending on a clean line.
			if d := dx*dx + dy*dy; d <= 0.75 || d <= 1 && (bayer[y&3][x&3]+0.5)/16 > (d-0.75)/0.25 {
				sprite.SetRGBA(x, y, shadowColor)
			}
		}
	}
	return sprite
}
