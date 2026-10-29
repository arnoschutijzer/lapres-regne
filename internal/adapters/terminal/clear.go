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
	vignetteFrom  = 0.2
	vignetteDepth = 0.7
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
		lift *= 1 - contactDepth*c.contact(x, y)
	}
	ratio := light / typical
	switch {
	case p.face > 0:
		// The renderer lights a face 0.45 + 0.55 of the sunlight on it, lifting faces turned from
		// the sun so they show in braille; here they fall to the sky's light alone.
		ratio, contrast = relit((p.face-0.45)/0.55)/relit(faceSun), clearLitContrast
	case foliage(p.ink):
		ratio, contrast = relit(light)/relit(typical), clearLitContrast
	}
	out := c.underlay() - 1
	step := clearStep + (clearStepFar-clearStep)*out
	shade := clearShade(ratio, contrast, lift, step, clearBlend+(clearBlendFar-clearBlend)*out, x, y)
	// Sunlight is warm and the sky that lights the shade is cool, so a surface leans warm the more
	// light it catches than its material typically does, and cool the less.
	color := tinted(muted(palette[p.ink]), (shade-1)/temperatureSpan)
	shade += step * clearPattern(p.ink, p.face > 0, u, v, out)
	return withValue(color, dusk(value*max(step, shade)))
}

// tinted leans a color toward the sun's warm hue for warmth above 0, up to 1, and toward the sky's
// cool hue for warmth below 0, down to -1.
func tinted(color [3]int, warmth float64) [3]int {
	if warmth > 0 {
		return mix(color, sunHue, sunTint*min(1, warmth))
	}
	return mix(color, skyHue, skyTint*min(1, -warmth))
}

// sunHue and skyHue are the hues of sunlight and of the sky's light in the shade; sunTint and skyTint
// are how far the most sunlit and the most shaded surfaces lean toward them, and temperatureSpan how
// far from a material's typical shade a surface must be to lean all the way.
var (
	sunHue = [3]int{255, 190, 120}
	skyHue = [3]int{110, 140, 200}
)

const (
	sunTint         = 0.3
	skyTint         = 0.4
	temperatureSpan = 0.4
)

// dusk darkens a value as dusk does, the palest most, so the view stays dark and calm while every
// material keeps its place among the others.
func dusk(value float64) float64 {
	return value * max(0, duskDim-duskFall*value)
}

// duskDim is how far dusk keeps the darkest values, and duskFall how much further it darkens each
// value the paler it is.
const (
	duskDim  = 0.75
	duskFall = 0.45
)

// muted is a color drawn clearMute of the way toward the grey of its own value.
func muted(color [3]int) [3]int {
	grey := int(math.Round(lumaFraction(color) * 255))
	return mix(color, [3]int{grey, grey, grey}, clearMute)
}

// clearMute is how far the clear style mutes every color toward grey.
const clearMute = 0.25

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

// contact is how much of the sky the ground at the canvas pixel x, y is hidden from by whatever
// stands close behind it and rises above it, from 0 in the open to 1 hemmed in: walls, trunks and
// figures darken the ground at their feet, most right at them. It looks up the canvas, where what
// stands on the ground just behind a point rises, for pixels standing well nearer the viewer than
// the ground there, and weighs the nearest most.
func (c *forestCanvas) contact(x, y int) float64 {
	depth := c.pixels[y*c.width+x].depth
	reach := max(2, int(contactReach*c.scale))
	hidden, total := 0.0, 0.0
	for k := 1; k <= reach; k++ {
		weight := 1 - float64(k-1)/float64(reach)
		for _, dx := range [3]int{-k / 2, 0, k / 2} {
			nx, ny := x+dx, y-k
			if nx < 0 || ny < 0 || nx >= c.width {
				continue
			}
			total += weight
			q := c.pixels[ny*c.width+nx]
			if (q.light > 0 || q.face > 0) && !groundInk(q.ink) && depth-q.depth > contactGap {
				hidden += weight
			}
		}
	}
	if total == 0 {
		return 0
	}
	return hidden / total
}

// contactReach is how far, in world units, the ground looks for what hems it in; contactGap is how
// much nearer the viewer, in world units, a pixel must stand to hide the sky; contactDepth is how
// much of its light hemmed-in ground loses.
const (
	contactReach = 0.8
	contactGap   = 0.15
	contactDepth = 0.6
)

// relit is how brightly a surface shows under a low sun and a faint sky, for the sunlight the
// renderer lit it with: that sunlight is a third the sky's and the rest the sun's own as the
// surface turns toward it, and the sky's share here falls to skyLight, so surfaces turned from the
// sun sink into shade as they do outdoors.
func relit(sun float64) float64 {
	direct := min(1, max(0, (sun-0.34)/0.6))
	return skyLight + (1-skyLight)*direct
}

// skyLight is how brightly the sky alone lights a surface turned from the sun, as a share of full
// sunlight; faceSun is the sunlight on a face the clear style shows at its material's own value,
// a wall turned halfway toward the sun.
const (
	skyLight = 0.2
	faceSun  = 0.55
)

// foliage tells whether an ink belongs to a crown of leaves or needles, which the renderer lights by
// the sunlight on each leaf.
func foliage(ink int) bool {
	return ink == shadedLeafInk || ink == leafInk || ink >= 11 && ink <= 16
}

// pool is how much brighter the ground shows at the canvas pixel x, y from the pool of light
// around the hero's feet, which lets the hero be found at once however far the view is zoomed out:
// most at their feet, falling off as light does to none at the pool's edge, and stronger the
// farther out the view is.
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
	// Light falls off with the square of the distance from a source a little above the ground,
	// eased to nothing at the pool's edge so it ends without a seam.
	edge := 1 / (1 + 1/poolHeight)
	falloff := (1/(1+d/poolHeight) - edge) / (1 - edge)
	out := c.underlay() - 1
	return (poolNear + (poolFar-poolNear)*out) * falloff
}

// poolReach is how far the pool of light around the hero reaches, in world units, but never fewer
// than poolPixels canvas pixels; poolNear is how much brighter the ground shows at the hero's feet at
// the zoom where trees show their detail, and poolFar how much all the way out. poolHeight is how
// high the light hangs, squared, as a share of the pool's reach squared: the lower it hangs, the
// faster its light falls off.
const (
	poolReach  = 3.0
	poolPixels = 14
	poolNear   = 0.6
	poolFar    = 1.1
	poolHeight = 0.06
)

// clearShadow is how bright ground in the shadow of a tree or a building shows, as a share of its
// shade in the open, so shadows fall in one clear direction whatever the noise in the ground's light.
const clearShadow = 0.65

// clearContrast is how strongly a surface's shade follows the light it catches, as a power of how
// many times its typical light that is; clearGroundContrast is how strongly the ground's does, and
// clearLitContrast how strongly that of faces and leaves relit by the sun and sky does.
const (
	clearContrast       = 0.6
	clearGroundContrast = 0.3
	clearLitContrast    = 0.8
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
		return 0.18, 0.24
	case 1, shadowGrassInk:
		return 0.2, 0.36
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
		return 0.075, 0.45
	case leafInk, 12, 16:
		return 0.17, 0.8
	case 13: // shaded yew needles
		return 0.065, 0.45
	case 14: // sunlit yew needles
		return 0.14, 0.8
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
