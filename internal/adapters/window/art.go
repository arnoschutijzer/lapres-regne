package window

import (
	"image"
	"image/color"
	"math"

	"lapresregne/internal/world"
)

// The window's pixel art follows gothic pixel art such as Blasphemous: deep, saturated materials
// whose shadows sink toward a cold violet-black and whose highlights lift toward warm gold, a few
// flat shades each, and a dark outline around every sprite.
var (
	shadowHue = [3]float64{10, 5, 20}
	lightHue  = [3]float64{255, 196, 120}
)

// shades is how many flat shades each material shows.
const shades = 5

// saturation is how far each material's color is pushed away from its own grey.
const saturation = 1.35

// ramp is a material's color at a shade from 0, darkest, to 1, brightest. The shade is first
// rounded to one of the flat shades.
func ramp(material [3]float64, shade float64) color.RGBA {
	shade = math.Round(math.Min(1, math.Max(0, shade))*(shades-1)) / (shades - 1)
	grey := (material[0] + material[1] + material[2]) / 3
	var rich [3]float64
	for i, channel := range material {
		rich[i] = math.Min(255, math.Max(0, grey+(channel-grey)*saturation))
	}
	const full = 0.75
	var c [3]float64
	for i := range c {
		if shade >= full {
			c[i] = rich[i] + (lightHue[i]-rich[i])*0.3*(shade-full)/(1-full)
		} else {
			c[i] = shadowHue[i] + (rich[i]-shadowHue[i])*math.Pow(shade/full, 1.6)
		}
	}
	return color.RGBA{uint8(math.Round(c[0])), uint8(math.Round(c[1])), uint8(math.Round(c[2])), 255}
}

// The materials the window paints with, before shading.
var (
	grassMaterial    = [3]float64{70, 96, 52}
	waterMaterial    = [3]float64{40, 86, 110}
	pathMaterial     = [3]float64{120, 96, 66}
	stoneMaterial    = [3]float64{104, 100, 112}
	boardsMaterial   = [3]float64{128, 86, 52}
	barkMaterial     = [3]float64{96, 70, 48}
	deadBarkMaterial = [3]float64{110, 100, 90}
	skinMaterial     = [3]float64{212, 160, 120}
	hairMaterial     = [3]float64{70, 46, 32}
	shirtMaterial    = [3]float64{168, 160, 146}
	shortsMaterial   = [3]float64{70, 74, 96}
	leafMaterials    = map[world.TreeKind][3]float64{
		world.NormalTree: {62, 104, 46},
		world.Oak:        {98, 96, 36},
		world.Willow:     {108, 124, 66},
		world.Yew:        {30, 82, 66},
	}
)

// outlined draws a dark outline around every opaque pixel of a sprite: each clear pixel touching an
// opaque one takes the darkest shade of what it touches.
func outlined(sprite *image.RGBA) *image.RGBA {
	bounds := sprite.Bounds()
	out := image.NewRGBA(bounds)
	copy(out.Pix, sprite.Pix)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if sprite.RGBAAt(x, y).A != 0 {
				continue
			}
			for _, step := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				neighbour := image.Pt(x+step[0], y+step[1])
				if neighbour.In(bounds) && sprite.RGBAAt(neighbour.X, neighbour.Y).A != 0 {
					c := sprite.RGBAAt(neighbour.X, neighbour.Y)
					out.SetRGBA(x, y, color.RGBA{c.R / 4, c.G / 4, c.B / 4, 255})
					break
				}
			}
		}
	}
	return out
}

// hash is a stable pseudo-random value from 0 to 1 for a pixel of a sprite.
func hash(x, y int, salt uint64) float64 {
	h := uint64(x)*0x9E3779B97F4A7C15 ^ uint64(y)*0xC2B2AE3D27D4EB4F ^ salt*0x165667B19E3779F9
	h ^= h >> 29
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 32
	return float64(h&0xFFFFFF) / 0xFFFFFF
}

// treeSprite draws a tree of a kind, crownWidth pixels across its crown and height pixels tall,
// lit from the upper left. Its foot, where the trunk meets the ground, is at the bottom centre.
func treeSprite(kind world.TreeKind, crownWidth, height int, variation float64) *image.RGBA {
	width := crownWidth + 2
	sprite := image.NewRGBA(image.Rect(0, 0, width, height+2))
	centre := float64(width) / 2
	trunkWidth := max(3, crownWidth/7)
	bark := barkMaterial
	if kind == world.DeadTree {
		bark = deadBarkMaterial
	}
	trunkTop := height / 3
	if kind == world.DeadTree {
		trunkTop = height / 6
	}
	for y := trunkTop; y <= height; y++ {
		for x := int(centre) - trunkWidth/2; x < int(centre)-trunkWidth/2+trunkWidth; x++ {
			across := (float64(x) - (centre - float64(trunkWidth)/2)) / float64(trunkWidth)
			shade := 0.85 - across*0.6 + (hash(x, y/3, 11)-0.5)*0.2
			sprite.SetRGBA(x, y, ramp(bark, shade))
		}
	}
	if kind == world.DeadTree {
		drawBranches(sprite, centre, float64(trunkTop), float64(height)*0.55, variation, bark)
		return outlined(sprite)
	}
	leaves := leafMaterials[kind]
	// The crown is a cluster of round lumps inside an ellipse over the trunk, each shaded as a
	// sphere lit from the upper left; the trunk shows below it.
	crownBottom := float64(height) * 0.68
	if kind == world.Yew {
		crownBottom = float64(height) * 0.8
	}
	across, down := float64(crownWidth)/2, (crownBottom-1)/2
	middle := 1 + down
	lumpRadius := math.Min(across, down) * 0.5
	type lump struct{ x, y, r float64 }
	lumps := []lump{{centre, middle, lumpRadius * 1.2}}
	for i := range 8 {
		angle := float64(i)/8*2*math.Pi + variation*6
		r := lumpRadius * (0.85 + 0.25*hash(i, 0, uint64(variation*1000)))
		lumps = append(lumps, lump{
			centre + math.Cos(angle)*(across-r),
			middle + math.Sin(angle)*(down-r),
			r,
		})
	}
	light := [3]float64{-0.55, -0.6, 0.58}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			best, nearest := -1.0, math.Inf(1)
			for _, l := range lumps {
				dx, dy := (float64(x)+0.5-l.x)/l.r, (float64(y)+0.5-l.y)/l.r
				d := dx*dx + dy*dy
				if d > 1 {
					continue
				}
				// The lump nearest the viewer, lowest on the screen, covers the others.
				dz := math.Sqrt(1 - d)
				if order := -l.y - dz*l.r; order < nearest {
					nearest = order
					best = dx*light[0] + dy*light[1] + dz*light[2]
				}
			}
			if math.IsInf(nearest, 1) {
				continue
			}
			// Leaf clusters break up the sphere's shading into a crisp, hand-placed texture.
			texture := (hash(x/3, y/2, 23) - 0.5) * 0.3
			sprite.SetRGBA(x, y, ramp(leaves, 0.2+best*0.75+texture))
		}
	}
	return outlined(sprite)
}

// drawBranches draws a dead tree's bare limbs reaching up from its trunk.
func drawBranches(sprite *image.RGBA, centre, from, reach, variation float64, bark [3]float64) {
	for i := range 5 {
		angle := -math.Pi/2 + (float64(i)-2)*0.45 + (variation-0.5)*0.3
		startY := from + float64(i%3)*reach*0.15
		for step := 0.0; step < reach*(0.5+0.1*float64(i%2)); step += 0.5 {
			x := int(centre + math.Cos(angle)*step)
			y := int(startY + math.Sin(angle)*step)
			if image.Pt(x, y).In(sprite.Bounds()) {
				sprite.SetRGBA(x, y, ramp(bark, 0.7-step/reach*0.3))
			}
		}
	}
}

// stumpSprite draws a felled tree's stump, width pixels across.
func stumpSprite(width int) *image.RGBA {
	height := max(4, width*2/3)
	sprite := image.NewRGBA(image.Rect(0, 0, width+2, height+2))
	for y := 1; y <= height; y++ {
		for x := 1; x <= width; x++ {
			shade := 0.8 - float64(x)/float64(width)*0.5
			material := barkMaterial
			if y <= 2 {
				// The cut top shows the pale wood inside.
				material, shade = [3]float64{196, 160, 110}, 0.8
			}
			sprite.SetRGBA(x, y, ramp(material, shade))
		}
	}
	return outlined(sprite)
}

// heroSprite draws the hero facing one way, unit pixels to a world unit; their feet are at the
// bottom centre.
func heroSprite(facing world.Facing, unit float64) *image.RGBA {
	height := int(math.Round(1.8 * unit))
	width := max(8, int(math.Round(0.6*unit)))
	sprite := image.NewRGBA(image.Rect(0, 0, width+2, height+2))
	at := func(fraction float64) int { return 1 + int(math.Round(fraction*float64(height))) }
	headTop, headBottom := at(0), at(0.14)
	shirtBottom := at(0.48)
	shortsBottom := at(0.62)
	left, right := 1, width
	fill := func(x0, y0, x1, y1 int, material [3]float64) {
		for y := y0; y < y1; y++ {
			for x := x0; x <= x1; x++ {
				// Light falls from the upper left, so the right of each part sinks into shadow.
				shade := 0.9 - float64(x-x0)/float64(max(1, x1-x0))*0.45
				sprite.SetRGBA(x, y, ramp(material, shade))
			}
		}
	}
	headLeft, headRight := left+width/4, right-width/4
	fill(headLeft, headTop, headRight, headBottom, skinMaterial)
	// Hair covers the top of the head, and the whole head seen from behind.
	hairBottom := headTop + 2
	if facing == world.FacingUp {
		hairBottom = headBottom
	}
	fill(headLeft, headTop, headRight, hairBottom, hairMaterial)
	if facing != world.FacingUp {
		eye := color.RGBA{20, 12, 16, 255}
		eyeY := headTop + (headBottom-headTop)*3/5
		switch facing {
		case world.FacingDown:
			sprite.SetRGBA(headLeft+1, eyeY, eye)
			sprite.SetRGBA(headRight-1, eyeY, eye)
		case world.FacingLeft:
			sprite.SetRGBA(headLeft, eyeY, eye)
		case world.FacingRight:
			sprite.SetRGBA(headRight, eyeY, eye)
		}
	}
	fill(left+1, headBottom, right-1, shirtBottom, shirtMaterial)
	// Arms hang beside the shirt.
	fill(left, headBottom+1, left, shirtBottom, skinMaterial)
	fill(right, headBottom+1, right, shirtBottom, skinMaterial)
	fill(left+1, shirtBottom, right-1, shortsBottom, shortsMaterial)
	legGap := width / 2
	fill(left+1, shortsBottom, legGap, height+1, skinMaterial)
	fill(legGap+1, shortsBottom, right-1, height+1, skinMaterial)
	return outlined(sprite)
}
