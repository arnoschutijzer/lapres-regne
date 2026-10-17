package window

import (
	"image"
	"image/color"
	"math"
)

// The atmosphere lies over the whole scene: low fog rolls over the floor, pale shafts of light
// slant down from the upper left where the light comes from, ash drifts in the air, and the
// screen's edges sink into darkness.
// Everything moves slowly, so the forest stays calm.

// vignette is how far a pixel of the screen sinks into darkness, from 0 in the middle to
// vignetteDepth in the corners.
func vignette(x, y int) float64 {
	dx := (float64(x) + 0.5 - screenWidth/2) / (screenWidth / 2)
	dy := (float64(y) + 0.5 - screenHeight/2) / (screenHeight / 2)
	d := math.Sqrt(dx*dx*0.75 + dy*dy)
	return vignetteDepth * math.Pow(math.Min(1, math.Max(0, (d-0.45)/0.75)), 1.4)
}

const vignetteDepth = 0.88

// vignetteImage is the darkness laid over the screen, as premultiplied void.
func vignetteImage() *image.RGBA {
	dark := image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	for y := range screenHeight {
		for x := range screenWidth {
			a := vignette(x, y)
			dark.SetRGBA(x, y, color.RGBA{uint8(float64(voidColor.R) * a), uint8(float64(voidColor.G) * a), uint8(float64(voidColor.B) * a), uint8(255 * a)})
		}
	}
	return dark
}

// shaftPeriod is how far across the shafts repeat, in art pixels, and shaftSlant how far right a
// shaft leans for each pixel it falls.
const (
	shaftPeriod = 560
	shaftSlant  = 0.55
)

// shaftLightColor is the warm light of the shafts, added to whatever they fall on.
var shaftLightColor = [3]float64{196, 186, 150}

// shaft is how strong the light of the shafts is at a point along their period: a few soft bands
// of differing width and strength.
func shaft(u float64) float64 {
	bands := [...]struct{ middle, width, strength float64 }{
		{70, 26, 0.07}, {120, 9, 0.045}, {300, 40, 0.055}, {350, 12, 0.04}, {470, 18, 0.045},
	}
	light := 0.0
	for _, b := range bands {
		d := math.Abs(math.Mod(u-b.middle+shaftPeriod*1.5, shaftPeriod) - shaftPeriod/2)
		if d < b.width {
			// Each band fades softly toward its edges.
			light += b.strength * (0.5 + 0.5*math.Cos(d/b.width*math.Pi))
		}
	}
	return light
}

// shaftsImage is one period of the shafts across the screen's height, as premultiplied light to
// add to the scene. Light falls from the upper left, so shafts lean down to the right.
func shaftsImage() *image.RGBA {
	light := image.NewRGBA(image.Rect(0, 0, shaftPeriod, screenHeight))
	for y := range screenHeight {
		// The shafts fade as they reach the ground at the bottom of the screen.
		fade := 1 - 0.5*float64(y)/screenHeight
		for x := range shaftPeriod {
			a := shaft(float64(x)-float64(y)*shaftSlant) * fade
			r, g, b := uint8(shaftLightColor[0]*a), uint8(shaftLightColor[1]*a), uint8(shaftLightColor[2]*a)
			light.SetRGBA(x, y, color.RGBA{r, g, b, max(r, g, b)})
		}
	}
	return light
}

// shaftDrift is how many art pixels a second the shafts drift across the world, as the light
// shifts through the canopy.
const shaftDrift = 1.5

// shaftOffset is how far across the screen one period of the shafts starts, for a camera whose top
// left is the world's pixel originX, some seconds in. The shafts hold their place on the ground as
// the camera moves.
func shaftOffset(originX int, seconds float64) float64 {
	return -math.Mod(math.Mod(float64(originX)-seconds*shaftDrift, shaftPeriod)+shaftPeriod, shaftPeriod)
}

// mote is a speck drifting in the air at a point of the screen, glowing at a strength from 0 to 1:
// grey ash, or now and then an ember.
type mote struct {
	x, y, glow float64
	ember      bool
}

// moteCell is the size of the world's cells, in art pixels, that each may hold one mote.
const moteCell = 56

// motes are the specks the camera shows some seconds in, for a camera whose top left is the
// world's pixel originX, originY. Each drifts slowly about its own place in the world, rising and
// swaying, and glows softly on and off.
func motes(originX, originY int, seconds float64) []mote {
	var shown []mote
	for cy := floorDiv(originY, moteCell) - 1; cy <= floorDiv(originY+screenHeight, moteCell)+1; cy++ {
		for cx := floorDiv(originX, moteCell) - 1; cx <= floorDiv(originX+screenWidth, moteCell)+1; cx++ {
			if hash(cx, cy, 71) > 0.45 {
				continue
			}
			phase := hash(cx, cy, 72) * 2 * math.Pi
			rise := math.Mod(seconds*2.5+hash(cx, cy, 73)*moteCell, moteCell)
			x := float64(cx*moteCell) + hash(cx, cy, 74)*moteCell + 5*math.Sin(seconds*0.35+phase) - float64(originX)
			y := float64(cy*moteCell) + hash(cx, cy, 75)*moteCell - rise - float64(originY)
			glow := 0.5 + 0.5*math.Sin(seconds*0.8+phase*1.3)
			if x >= -2 && y >= -2 && x < screenWidth+2 && y < screenHeight+2 {
				shown = append(shown, mote{x, y, glow, hash(cx, cy, 76) < 0.15})
			}
		}
	}
	return shown
}

// moteImage is a mote's glow: a bright pixel in a faint halo.
func moteImage(t tones) *image.RGBA {
	glow := image.NewRGBA(image.Rect(0, 0, 3, 3))
	halo := t[1]
	faint := color.RGBA{halo.R / 3, halo.G / 3, halo.B / 3, 255 / 3}
	for _, at := range []image.Point{{1, 0}, {0, 1}, {2, 1}, {1, 2}} {
		glow.SetRGBA(at.X, at.Y, faint)
	}
	glow.SetRGBA(1, 1, t[2])
	return glow
}

// fogPeriod is the size of the fog's tile, in art pixels; the tile repeats seamlessly.
const fogPeriod = 256

// fogColor is the cold, pale mist drifting low over the forest floor.
var fogColor = rgb(0x5a6460)

// fogDensity is how thick the fog is at a pixel of its tile, from 0 to 1: soft drifts made of
// value noise on a grid that wraps round the tile.
func fogDensity(x, y int, salt uint64) float64 {
	octave := func(cell int) float64 {
		cells := fogPeriod / cell
		gx, gy := floorDiv(x, cell), floorDiv(y, cell)
		fx, fy := float64(mod(x, cell))/float64(cell), float64(mod(y, cell))/float64(cell)
		corner := func(cx, cy int) float64 { return hash(mod(cx, cells), mod(cy, cells), salt) }
		smooth := func(t float64) float64 { return t * t * (3 - 2*t) }
		sx, sy := smooth(fx), smooth(fy)
		top := corner(gx, gy) + (corner(gx+1, gy)-corner(gx, gy))*sx
		bottom := corner(gx, gy+1) + (corner(gx+1, gy+1)-corner(gx, gy+1))*sx
		return top + (bottom-top)*sy
	}
	density := octave(64)*0.65 + octave(32)*0.35
	return math.Min(1, math.Max(0, (density-0.45)*2.2))
}

// fogImage is one tile of fog, its density dithered into scattered pixels of mist.
func fogImage(salt uint64) *image.RGBA {
	fog := image.NewRGBA(image.Rect(0, 0, fogPeriod, fogPeriod))
	for y := range fogPeriod {
		for x := range fogPeriod {
			if fogDensity(x, y, salt) > (bayer[y&3][x&3]+0.5)/16 {
				fog.SetRGBA(x, y, fogColor)
			}
		}
	}
	return fog
}

// fogLayer is one layer of fog: how fast it drifts across the world, in art pixels a second, and
// how strongly it shows.
type fogLayer struct {
	drift, strength float64
	salt            uint64
}

// fogLayers drift at different speeds, so the mist seems to roll.
var fogLayers = [...]fogLayer{{drift: 2, strength: 0.16, salt: 81}, {drift: -1.2, strength: 0.1, salt: 82}}

// fogOffset is where a tile of a fog layer starts on the screen, across and down, for a camera
// whose top left is the world's pixel originX, originY, some seconds in. The fog holds its place
// on the ground as the camera moves, drifting slowly.
func fogOffset(layer fogLayer, originX, originY int, seconds float64) (x, y float64) {
	wrap := func(v float64) float64 { return -math.Mod(math.Mod(v, fogPeriod)+fogPeriod, fogPeriod) }
	return wrap(float64(originX) - seconds*layer.drift), wrap(float64(originY) - seconds*layer.drift*0.3)
}
