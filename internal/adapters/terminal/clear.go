package terminal

import "math"

// Clear draws every material in a band of value of its own, from dark water through trees, grass,
// paths and roofs up to pale stone, so the ground's materials and what stands on them read apart
// when the view takes in a lot. Each surface is filled flat in a few shades, dithered only where
// one shade gives way to the next or two materials meet within a cell, and fewer shades the farther
// out the view is zoomed.
type Clear struct{}

func (Clear) Name() string { return "clear" }

func (Clear) cell(c *forestCanvas, palette [][3]int, col, row int) forestCell {
	var colors [8][3]int
	for dy := range 4 {
		for dx := range 2 {
			x, y := col*2+dx, row*4+dy
			p := c.pixels[y*c.width+x]
			if p.light > 0 && figureInk(p.ink) {
				return c.figureCell(col, row)
			}
			colors[dy*2+dx] = clearTone(c, palette, x, y)
		}
	}
	return twoTone(colors)
}

// clearTone is the flat color the canvas pixel at x, y shows: its material's band of value, in the
// nearest shade of how much more or less light it catches than that material typically does.
func clearTone(c *forestCanvas, palette [][3]int, x, y int) [3]int {
	p := c.pixels[y*c.width+x]
	light := p.light
	// A wall's face is shaded by the light falling on it rather than its stonework, so it reads as
	// one flat plane, its mortar and darkest stones included.
	if p.face > 0 {
		light = p.face
	}
	if light <= 0 {
		return background
	}
	value, typical := clearBand(palette, p.ink)
	contrast := clearContrast
	if groundInk(p.ink) {
		// The ground is lit more faintly zoomed out to quiet it; here its band of value sets it
		// apart instead, so it keeps its strength. Its shading is softened, so the noise in its
		// light leaves it one calm tone over wide stretches.
		light /= c.quiet()
		contrast = clearGroundContrast
	}
	return withValue(palette[p.ink], value*clearShade(light/typical, contrast, x, y, c.underlay()-1))
}

// clearContrast is how strongly a surface's shade follows the light it catches, as a power of how
// many times its typical light that is; clearGroundContrast is how strongly the ground's does.
const (
	clearContrast       = 0.6
	clearGroundContrast = 0.3
)

// clearStep is how far apart the clear style's shades lie, as a share of a material's band, at the
// zoom where trees show their detail, widening to clearStepFar all the way out; clearBlend is how
// much of the step between two shades is dithered, narrowing to clearBlendFar all the way out.
const (
	clearStep     = 0.14
	clearStepFar  = 0.22
	clearBlend    = 0.4
	clearBlendFar = 0.25
)

// clearShade is the share of its band a pixel at x, y shows for how many times its material's
// typical light it catches: the nearest of the style's shades, which lie a step apart either side
// of the material's typical light, dithered with the next only within the blend of the step between
// them. contrast is how strongly the shade follows the ratio, and out is how far the view is zoomed
// out, from 0 at the zoom where trees show their detail to 1 all the way out.
func clearShade(ratio, contrast float64, x, y int, out float64) float64 {
	// Light is felt more evenly than it falls: at a contrast of 0.6, halving it darkens a surface by
	// about a third.
	shade := math.Pow(min(1.8, max(0.3, ratio)), contrast)
	step := clearStep + (clearStepFar-clearStep)*out
	blend := clearBlend + (clearBlendFar-clearBlend)*out
	// Counted from the typical light, so a surface lit as it typically is shows one flat shade.
	level := (shade - 1) / step
	lower := math.Floor(level)
	if min(1, max(0, (level-lower-(1-blend)/2)/blend)) > ordered(x, y) {
		lower++
	}
	return max(step, 1+lower*step)
}

// clearBand is the value, from 0 to 1, a material shows in the clear style at the light it typically
// catches, and that light. Water lies darkest, then trees, so they stand dark on the grass, then
// floors and paths, with roofs between and stone palest; anything else keeps its own color's value.
func clearBand(palette [][3]int, ink int) (value, typical float64) {
	switch ink {
	case waterInk:
		return 0.075, 0.44
	case 6: // ripples
		return 0.16, 0.8
	case 0: // bare ground
		return 0.15, 0.24
	case 1, shadowGrassInk:
		return 0.17, 0.36
	case 7: // sunlit banks
		return 0.19, 0.54
	case bankInk:
		return 0.17, 0.34
	case reedInk:
		return 0.2, 0.6
	case pathInk, shadowPathInk:
		return 0.27, 0.32
	case floorInk, shadowFloorInk:
		return 0.23, 0.36
	case boardsInk:
		return 0.23, 0.26
	case yellowFlowerInk, whiteFlowerInk, violetFlowerInk:
		return 0.3, 0.38
	case pebbleInk:
		return 0.25, 0.64
	case bushInk:
		return 0.14, 0.64
	case shadedLeafInk, 11, 15:
		return 0.08, 0.45
	case leafInk, 12, 16:
		return 0.14, 0.8
	case 13: // shaded yew needles
		return 0.07, 0.45
	case 14: // sunlit yew needles
		return 0.12, 0.8
	case 4: // bark
		return 0.12, 0.62
	case stoneInk:
		return 0.44, 0.7
	case shadedStoneInk:
		return 0.32, 0.46
	case mossInk:
		return 0.3, 0.49
	case glassInk:
		return 0.14, 0.36
	case ironInk:
		return 0.36, 0.6
	case plankInk:
		return 0.25, 0.8
	case thatchInk:
		return 0.3, 0.7
	case shadedThatchInk:
		return 0.21, 0.7
	case slateInk:
		return 0.21, 0.7
	case shadedSlateInk:
		return 0.15, 0.7
	case clayInk:
		return 0.24, 0.62
	case shadedClayInk:
		return 0.17, 0.62
	}
	return min(0.5, max(0.1, lumaFraction(palette[ink])*0.5)), 0.75
}

// lumaFraction is how bright a color looks, from 0 to 1.
func lumaFraction(color [3]int) float64 {
	return (0.299*float64(color[0]) + 0.587*float64(color[1]) + 0.114*float64(color[2])) / 255
}

// withValue is a color of the same hue as color that looks as bright as value, from 0 to 1, or as
// near as its hue allows.
func withValue(color [3]int, value float64) [3]int {
	scale := value / max(lumaFraction(color), 1.0/255)
	for _, channel := range color {
		if channel > 0 {
			scale = min(scale, 255/float64(channel))
		}
	}
	var valued [3]int
	for i, channel := range color {
		valued[i] = int(math.Round(float64(channel) * scale))
	}
	return valued
}
