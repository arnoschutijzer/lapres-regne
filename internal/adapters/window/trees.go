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
		drawTrunk(sprite, centre, height/5, height, max(3, crownWidth/9), deadBarkTones)
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
	type lobe struct{ x, y, r, phase float64 }
	lobes := []lobe{{float64(centre), middleY, math.Min(across, down) * 0.62, variation * 9}}
	count := 6
	for i := range count {
		angle := float64(i)/float64(count)*2*math.Pi + variation*5
		r := math.Min(across, down) * (0.42 + 0.1*hash(i, 1, uint64(variation*977)))
		lobes = append(lobes, lobe{
			float64(centre) + math.Cos(angle)*(across-r),
			middleY + math.Sin(angle)*(down-r),
			r,
			float64(i) * 1.7,
		})
	}
	light := [3]float64{-0.5, -0.62, 0.6}
	bounds := sprite.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nearest, shade, rim := math.Inf(1), 0.0, false
			for _, l := range lobes {
				dx, dy := float64(x)+0.5-l.x, float64(y)+0.5-l.y
				// A scalloped edge of leaves runs round each lobe.
				r := l.r * (1 + 0.07*math.Sin(math.Atan2(dy, dx)*7+l.phase))
				d := (dx*dx + dy*dy) / (r * r)
				if d > 1 {
					continue
				}
				// The lobe lowest on the screen stands nearest the viewer and covers the others.
				nz := math.Sqrt(1 - d)
				if order := -(l.y + nz*l.r*0.3); order < nearest {
					nearest = order
					shade = (dx/r)*light[0] + (dy/r)*light[1] + nz*light[2]
					rim = d > 0.86 && dy > 0
				}
			}
			if math.IsInf(nearest, 1) {
				continue
			}
			value := (shade+0.42+leaf(x, y)*0.16)*3.4 + 0.5
			if rim {
				// The underside of each lobe sinks into the dark between it and the next.
				value = math.Min(value, 1.2)
			}
			sprite.SetRGBA(x, y, leaves.dithered(value, x, y))
		}
	}
	if kind == world.Willow {
		drawStrands(sprite, centre, middleY, across, down, leaves)
	}
	return sprite
}

// trunkShows is how much of a tree's trunk shows below its crown, for a tree height pixels tall.
func trunkShows(height int) float64 { return float64(height) * 0.22 }

// drawTrunk draws a trunk from top down to its foot at the bottom, width pixels wide and lit from
// the left, flaring into roots at the foot.
func drawTrunk(sprite *image.RGBA, centre, top, foot, width int, bark tones) {
	left := centre - width/2
	for y := top; y <= foot; y++ {
		flare := 0
		if y >= foot-1 {
			flare = 1
		}
		for x := left - flare; x < left+width+flare; x++ {
			tone := 2
			switch {
			case x <= left:
				tone = 3
			case x >= left+width-2:
				tone = 1
			case hash(x, y/3, 17) > 0.8:
				// Furrows in the bark run up the trunk.
				tone = 1
			}
			if x < left || x >= left+width {
				tone = 0
			}
			sprite.SetRGBA(x, y, bark.at(tone))
		}
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

// drawStrands hangs a willow's long strands of leaves down from the lower half of its crown.
func drawStrands(sprite *image.RGBA, centre int, middleY, across, down float64, leaves tones) {
	for x := centre - int(across) + 3; x <= centre+int(across)-3; x += 2 {
		dx := float64(x-centre) / across
		edge := middleY + down*math.Sqrt(math.Max(0, 1-dx*dx))*0.85
		length := 5 + int(hash(x, 0, 29)*9)
		for y := int(edge); y < int(edge)+length; y++ {
			if !image.Pt(x, y).In(sprite.Bounds()) {
				break
			}
			tone := 3 - (y-int(edge))*3/length
			if dx < -0.3 {
				tone++
			}
			sprite.SetRGBA(x, y, leaves.at(tone))
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
