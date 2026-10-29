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
	originX, originY := c.patternOrigin()
	for dy := range 4 {
		for dx := range 2 {
			x, y := col*2+dx, row*4+dy
			p := c.pixels[y*c.width+x]
			if p.light > 0 && figureInk(p.ink) {
				return c.figureCell(col, row)
			}
			colors[dy*2+dx] = clearTone(c, palette, x, y, x-originX, y-originY)
		}
	}
	drawn := twoTone(colors)
	dim := c.vignette(col, row)
	drawn.color, drawn.paper = mix(drawn.color, background, dim), mix(drawn.paper, background, dim)
	return drawn
}

// vignette is how far the cell at col, row sinks toward the background, as the view darkens toward
// its edges and corners: not at all around the middle, where the camera looks, and most in the
// corners, drawing the eye toward the hero.
func (c *forestCanvas) vignette(col, row int) float64 {
	// The camera looks at the point 0.57 of the way down the canvas.
	across := (float64(col*2+1) - float64(c.width)/2) / (float64(c.width) / 2)
	down := (float64(row*4+2) - float64(c.height)*0.57) / (float64(c.height) * 0.57)
	d := math.Sqrt((across*across + down*down) / 2)
	t := min(1, max(0, (d-vignetteFrom)/(1-vignetteFrom)))
	return vignetteDepth * t * t * (3 - 2*t)
}

// vignetteFrom is how far from the middle toward a corner the vignette starts to darken the view,
// and vignetteDepth how far it sinks the corners toward the background.
const (
	vignetteFrom  = 0.4
	vignetteDepth = 0.55
)

// patternOrigin is the canvas pixel showing where the world's origin lies on level ground, so
// patterns laid out from it move with the world as the view pans.
func (c *forestCanvas) patternOrigin() (int, int) {
	across, toward := c.toView(0, 0)
	return int(math.Round(float64(c.width)/2 + across*c.scale)),
		int(math.Round(float64(c.height)*0.57 + (toward*c.tiltSin+c.eye*c.tiltCos)*c.scale))
}

// clearTone is the flat color the canvas pixel at x, y shows: its material's band of value, in the
// nearest shade of how much more or less light it catches than that material typically does, a
// shade lighter or darker where the material's pattern marks u, v, its place among the patterns.
func clearTone(c *forestCanvas, palette [][3]int, x, y, u, v int) [3]int {
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
	contrast, lift := clearContrast, 1.0
	if groundInk(p.ink) {
		lift += c.pool(x, y)
		// The ground is lit more faintly zoomed out to quiet it; here its band of value sets it
		// apart instead, so it keeps its strength. Its shading is softened, so the noise in its
		// light leaves it one calm tone over wide stretches.
		light /= c.quiet()
		contrast = clearGroundContrast
		if p.shaded || p.ink == shadowGrassInk || p.ink == shadowPathInk || p.ink == shadowFloorInk {
			lift *= clearShadow
		}
	}
	out := c.underlay() - 1
	step := clearStep + (clearStepFar-clearStep)*out
	shade := clearShade(light/typical, contrast, lift, step, clearBlend+(clearBlendFar-clearBlend)*out, x, y)
	shade += step * clearPattern(p.ink, p.face > 0, u, v, out)
	return withValue(palette[p.ink], value*max(step, shade))
}

// clearPattern is how many shades lighter, or darker for less than 0, a material is drawn at u, v
// among its pattern, so materials of near value still read apart: grass in upright blades, water in
// level dashes, paths speckled, roofs in faint courses of tiles and the faces of stone walls in faint
// bricks, which fade as the view zooms out, to 1 all the way out, where they would only be noise.
func clearPattern(ink int, face bool, u, v int, out float64) float64 {
	switch ink {
	case 1, shadowGrassInk:
		if blade(u, v) {
			return 1.5
		}
	case waterInk:
		// Every fourth row of pixels, dashes three long lie staggered from row to row.
		if floorMod(v, 4) == 0 && floorMod(u+5*floorDiv(v, 4), 9) < 3 {
			return 1
		}
	case pathInk, shadowPathInk:
		if grain(u*3+11, v*5+7) < 0.12 {
			return -1
		}
	case clayInk, shadedClayInk, slateInk, shadedSlateInk, thatchInk, shadedThatchInk:
		if floorMod(v, 4) == 0 {
			return -0.5
		}
	case stoneInk, shadedStoneInk:
		// Courses five pixels high, their joints eight apart and staggered by half a brick.
		if face && (floorMod(v, 5) == 0 || floorMod(u+4*floorMod(floorDiv(v, 5), 2), 8) == 0) {
			return -0.6 * (1 - out)
		}
	}
	return 0
}

// blade tells whether a blade of grass passes through u, v. The ground is parted into tiles three
// pixels wide and five high, most holding a blade three pixels tall rising from a scattered root
// and leaning one way or the other at its tip.
func blade(u, v int) bool {
	tileU, tileV := floorDiv(u, 3), floorDiv(v, 5)
	if grain(tileU*7+1, tileV*13+5) > 0.7 {
		return false
	}
	rootU := tileU*3 + int(grain(tileU*5+3, tileV*11+2)*2)
	rootV := tileV*5 + 2 + int(grain(tileU*3+9, tileV*7+4)*3)
	tip := rootU
	if grain(tileU*11+6, tileV*3+8) < 0.5 {
		tip++
	}
	return u == rootU && (v == rootV || v == rootV-1) || u == tip && v == rootV-2
}

// floorDiv divides a by b, rounding down, and floorMod is what it leaves over, from 0 up to b.
func floorDiv(a, b int) int {
	if a < 0 {
		return (a - b + 1) / b
	}
	return a / b
}

func floorMod(a, b int) int { return a - floorDiv(a, b)*b }

// pool is how much brighter the ground shows at the canvas pixel x, y from the pool of light
// around the hero's feet, which lets the hero be found at once however far the view is zoomed out:
// most at their feet, fading to none at the pool's edge, and stronger the farther out the view is.
func (c *forestCanvas) pool(x, y int) float64 {
	if !c.heroShown {
		return 0
	}
	radius := max(poolPixels, poolReach*c.scale)
	across := (float64(x) - c.heroX) / radius
	// The pool lies on the ground, so it is foreshortened as the ground is.
	down := (float64(y) - c.heroY) / (radius * c.tiltSin)
	if c.tiltSin == 0 {
		down = (float64(y) - c.heroY) / radius
	}
	d := across*across + down*down
	if d >= 1 {
		return 0
	}
	out := c.underlay() - 1
	return (poolNear + (poolFar-poolNear)*out) * (1 - d) * (1 - d)
}

// poolReach is how far the pool of light around the hero reaches, in world units, but never fewer
// than poolPixels canvas pixels; poolNear is how much brighter the ground shows at the hero's feet at
// the zoom where trees show their detail, and poolFar how much all the way out.
const (
	poolReach  = 2.5
	poolPixels = 12
	poolNear   = 0.25
	poolFar    = 0.55
)

// clearShadow is how bright ground in the shadow of a tree or a building shows, as a share of its
// shade in the open, so shadows fall in one clear direction whatever the noise in the ground's light.
const clearShadow = 0.65

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
// them. contrast is how strongly the shade follows the ratio, and lift brightens it by a factor
// before it is banded.
func clearShade(ratio, contrast, lift, step, blend float64, x, y int) float64 {
	// Light is felt more evenly than it falls: at a contrast of 0.6, halving it darkens a surface by
	// about a third.
	shade := math.Pow(min(1.8, max(0.3, ratio)), contrast) * lift
	// Counted from the typical light, so a surface lit as it typically is shows one flat shade.
	level := (shade - 1) / step
	lower := math.Floor(level)
	if min(1, max(0, (level-lower-(1-blend)/2)/blend)) > ordered(x, y) {
		lower++
	}
	return 1 + lower*step
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
		return 0.17, 0.24
	case 1, shadowGrassInk:
		return 0.19, 0.36
	case 7: // sunlit banks
		return 0.21, 0.54
	case bankInk:
		return 0.19, 0.34
	case reedInk:
		return 0.2, 0.6
	case pathInk, shadowPathInk:
		return 0.29, 0.32
	case floorInk, shadowFloorInk:
		return 0.23, 0.36
	case boardsInk:
		return 0.23, 0.26
	case yellowFlowerInk, whiteFlowerInk, violetFlowerInk:
		return 0.3, 0.38
	case pebbleInk:
		return 0.25, 0.64
	case bushInk:
		return 0.11, 0.64
	case shadedLeafInk, 11, 15:
		return 0.065, 0.45
	case leafInk, 12, 16:
		return 0.12, 0.8
	case 13: // shaded yew needles
		return 0.055, 0.45
	case 14: // sunlit yew needles
		return 0.1, 0.8
	case 4: // bark
		return 0.1, 0.62
	case stoneInk:
		return 0.46, 0.7
	case shadedStoneInk:
		return 0.34, 0.46
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
