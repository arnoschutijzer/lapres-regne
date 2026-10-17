package terminal

import (
	"math"

	"lapresregne/internal/world"
)

// The inks of what is scattered over the ground: wildflowers, stones, bushes, and the mud and reeds
// along the water.
const (
	yellowFlowerInk = shadowFloorInk + 1 + iota
	whiteFlowerInk
	violetFlowerInk
	pebbleInk
	bushInk
	bankInk
	reedInk
)

var flowerInks = [...]int{yellowFlowerInk, whiteFlowerInk, violetFlowerInk}

// scatterScale is the zoom from which stones, bushes and flowers are drawn one by one; farther out
// they would be single dots lost in the ground, and the ground's own specks show the flowers.
var scatterScale = forestScale * math.Pow(zoomStep, 3)

// meadowAt tells how much a patch of open grass blooms, from 0 for none upward.
func meadowAt(f *world.World, x, z float64) float64 {
	return max(0, f.Noise(x/7, z/7, 97)-0.58)
}

// bareAt tells whether the grass has worn away to bare earth in a patch.
func bareAt(f *world.World, x, z float64) bool {
	return f.Noise(x/5, z/5, 98) > 0.76
}

// groundSpeck colors a pixel of open grass: a wildflower where meadows bloom, or bare earth in a
// worn patch. It reports false to leave the grass as it is.
func groundSpeck(f *world.World, x, z, n float64, scale float64) (ink int, light float64, ok bool) {
	if bareAt(f, x, z) {
		return pathInk, 0.2 + n*0.2, true
	}
	// Up close the flowers stand on their stems instead.
	if scale >= detailScale {
		return 0, 0, false
	}
	bloom := meadowAt(f, x, z)
	if bloom == 0 {
		return 0, 0, false
	}
	gx, gz := int(math.Floor(x*3)), int(math.Floor(z*3))
	if f.Random(gx, gz, 99) > 0.08+bloom*0.6 {
		return 0, 0, false
	}
	return flowerInks[int(f.Random(gx, gz, 100)*3)], 0.75, true
}

// banked tells whether a point on a grass tile lies on the muddy edge of a neighbouring water tile.
func (a *builtArea) banked(x, z float64) bool {
	col, row, alongX, alongZ := tileAt(x, z)
	const reach = 0.5
	water := func(step [2]int) bool { return a.at(col+step[0], row+step[1]).ground == world.Water }
	return alongX < reach && water(neighbours[3]) || world.TileSize-alongX < reach && water(neighbours[1]) ||
		alongZ < reach && water(neighbours[0]) || world.TileSize-alongZ < reach && water(neighbours[2])
}

// besideWater tells whether a point on a map's grass lies on a tile next to water.
func (a *builtArea) besideWater(x, z float64) bool {
	col, row, _, _ := tileAt(x, z)
	for _, step := range neighbours {
		if a.at(col+step[0], row+step[1]).ground == world.Water {
			return true
		}
	}
	return false
}

// scatterSpacing is the size of the cells of the grid that places scattered things, one at most in
// each.
const scatterSpacing = 1.3

// drawScatter dots the grass within the drawn area with reeds along the water, and up close with
// stones, bushes and wildflowers.
func (r *renderer) drawScatter(c *forestCanvas, left, right, back, front float64) {
	f := r.world
	for gz := int(math.Floor(back / scatterSpacing)); gz <= int(math.Ceil(front/scatterSpacing)); gz++ {
		for gx := int(math.Floor(left / scatterSpacing)); gx <= int(math.Ceil(right/scatterSpacing)); gx++ {
			x := (float64(gx) + f.Random(gx, gz, 110)) * scatterSpacing
			z := (float64(gz) + f.Random(gx, gz, 111)) * scatterSpacing
			c.standOn(x, z)
			sx, sy, _ := c.project(x, 0, z)
			if sx < -10 || sx > float64(c.width)+10 || sy < -10 || sy > float64(c.height)+10 {
				continue
			}
			grass, waterside := r.grassAt(x, z)
			if !grass {
				continue
			}
			roll := f.Random(gx, gz, 112)
			variety := f.Random(gx, gz, 113)
			switch {
			case waterside && roll < 0.7:
				drawReeds(c, x, z, variety)
			case c.scale < scatterScale:
			case roll < 0.03:
				drawStone(c, x, z, variety)
			case roll < 0.05:
				drawBush(c, x, z, variety)
			case roll < 0.08+meadowAt(f, x, z)*1.5:
				drawFlower(c, x, z, variety)
			}
		}
	}
}

// grassAt tells whether open grass grows at a point, and whether it grows beside water.
func (r *renderer) grassAt(x, z float64) (grass, waterside bool) {
	f := r.world
	if f.Mapped() {
		ground, onMap := f.GroundAt(x, z)
		if !onMap || ground != world.Grass {
			return false, false
		}
		return true, r.built.besideWater(x, z)
	}
	distance, width := world.NearestRiver(r.rivers.at(z), x)
	if distance < width {
		return false, false
	}
	return true, distance < width+1.2
}

// drawReeds raises a clump of slender reeds, leaning a little each their own way.
func drawReeds(c *forestCanvas, x, z, variety float64) {
	for i := range 3 + int(variety*3) {
		angle := float64(i)*2.4 + variety*6
		offsetX, offsetZ := math.Cos(angle)*0.18, math.Sin(angle)*0.12
		height := 0.9 + 0.5*math.Abs(math.Sin(angle*1.7))
		lean := math.Sin(angle*2.3) * 0.15
		c.line(x+offsetX, 0, z+offsetZ, x+offsetX+lean, height, z+offsetZ, 0.6+0.3*math.Abs(math.Cos(angle)), reedInk)
	}
}

// drawStone sets a rounded stone half sunk in the grass, sized by variety from 0 to 1.
func drawStone(c *forestCanvas, x, z, variety float64) {
	radius := 0.12 + variety*0.2
	ellipsoid(c, x, radius*0.3, z, radius, radius*0.6, radius*0.8, variety*3, func(px, py int, depth float64, _, normal vec) {
		c.plot(px, py, depth, 0.2+0.8*sunlight(normal)*c.rim(normal), pebbleInk)
	})
}

// drawBush rounds a low bush of leaves on the grass, sized by variety from 0 to 1.
func drawBush(c *forestCanvas, x, z, variety float64) {
	radius, height := 0.35+variety*0.3, 0.3+variety*0.15
	ellipsoid(c, x, height*0.8, z, radius, height, radius*0.85, 0, func(px, py int, depth float64, point, normal vec) {
		clumps := leafClumps(x+point.x*2, height+point.y*2, z+point.z*2)
		c.plot(px, py, depth, (0.2+0.8*sunlight(normal))*(0.6+0.5*clumps)*c.rim(normal), bushInk)
	})
}

// drawFlower stands a wildflower on its stem, its bloom one of the flowers' colors.
func drawFlower(c *forestCanvas, x, z, variety float64) {
	height := 0.25 + variety*0.2
	c.line(x, 0, z, x, height, z, 0.5, 1)
	head := c.thin(0.06)
	ellipsoid(c, x, height, z, head, head, head, 0, func(px, py int, depth float64, _, normal vec) {
		c.plot(px, py, depth, 0.6+0.4*sunlight(normal), flowerInks[int(variety*7)%3])
	})
}
