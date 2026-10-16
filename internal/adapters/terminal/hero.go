package terminal

import (
	"math"

	"lapresregne/internal/world"
)

// detailScale is the zoom from which the hero is drawn in detail.
var detailScale = forestScale * math.Pow(zoomStep, 6)

// heroLook holds the palette entries the detailed hero is painted in.
type heroLook struct{ skin, hair, tunic, trousers, boots int }

var defaultLook = heroLook{skin: 17, hair: 18, tunic: 19, trousers: 20, boots: 21}

// eyeInk marks the eyes, drawn as unlit gaps in the face.
const eyeInk = 22

// heroBody places the hero's body parts around their feet, turned the way they face.
type heroBody struct {
	c                  *forestCanvas
	x, z               float64
	forwardX, forwardZ float64
	sideX, sideZ       float64
}

func newHeroBody(c *forestCanvas, x, z float64, facing world.Facing) heroBody {
	b := heroBody{c: c, x: x, z: z, forwardZ: 1, sideX: 1}
	switch facing {
	case world.FacingUp:
		b.forwardZ, b.sideX = -1, -1
	case world.FacingLeft:
		b.forwardX, b.forwardZ, b.sideX, b.sideZ = -1, 0, 0, 1
	case world.FacingRight:
		b.forwardX, b.forwardZ, b.sideX, b.sideZ = 1, 0, 0, -1
	}
	return b
}

// at turns an offset to the hero's side and front into a world position.
func (b heroBody) at(side, forward float64) (x, z float64) {
	return b.x + b.sideX*side + b.forwardX*forward, b.z + b.sideZ*side + b.forwardZ*forward
}

// part draws one rounded, sunlit body part with its own width, height and depth.
func (b heroBody) part(side, y, forward, width, height, depth float64, ink int) {
	x, z := b.at(side, forward)
	rx, rz := b.radii(width, depth)
	ellipsoid(b.c, x, y, z, rx, height, rz, func(px, py int, depth float64, _, normal vec) {
		light := 0.5 + 0.5*math.Max(0, normal.dot(vec{-0.52, 0.80, -0.29}))
		b.c.plot(px, py, depth, light*math.Min(1, rim(normal)*1.6), ink)
	})
}

// outline darkens a thin band around a body part, set just behind the hero, so the figure
// stands out from the dithered ground.
func (b heroBody) outline(side, y, forward, width, height, depth float64) {
	x, z := b.at(side, forward)
	band := 1.5 / b.c.scale
	rx, rz := b.radii(width+band, depth+band)
	ellipsoid(b.c, x, y, z, rx, height+band, rz, func(px, py int, depth float64, _, _ vec) {
		b.c.plot(px, py, depth+0.4, 0, 0)
	})
}

// radii turns a part's width across the body and depth front to back into world radii.
func (b heroBody) radii(width, depth float64) (rx, rz float64) {
	return math.Abs(b.sideX)*width + math.Abs(b.forwardX)*depth, math.Abs(b.sideZ)*width + math.Abs(b.forwardZ)*depth
}

// eye leaves an unlit gap two dots tall on the front of the face.
func (b heroBody) eye(side float64) {
	x, z := b.at(side, 0.15)
	px, py, depth := b.c.project(x, 1.29, z)
	for dy := range 2 {
		b.c.plot(int(math.Round(px)), int(math.Round(py))+dy, depth-0.05, 0, eyeInk)
	}
}

// drawDetailedHero builds the hero from rounded parts, so dithered light shades each one.
func drawDetailedHero(c *forestCanvas, x, z float64, facing world.Facing, stride float64, look heroLook) {
	b := newHeroBody(c, x, z, facing)
	type part struct {
		side, y, forward, width, height, depth float64
		ink                                    int
	}
	parts := []part{
		{0, 0.85, 0, 0.19, 0.27, 0.12, look.tunic},
		{0, 1.27, 0, 0.15, 0.16, 0.15, look.skin},
		// Hair caps the top and back of the head, leaving the face bare from the front.
		{0, 1.36, -0.06, 0.16, 0.13, 0.15, look.hair},
	}
	for _, side := range []float64{-1, 1} {
		swing := stride * side
		parts = append(parts,
			part{side * 0.1, 0.07, 0.03 + swing, 0.075, 0.07, 0.11, look.boots},
			part{side * 0.1, 0.38, swing * 0.5, 0.075, 0.28, 0.08, look.trousers},
			part{side * 0.25, 0.83, -swing * 0.6, 0.06, 0.24, 0.06, look.tunic},
			part{side * 0.26, 0.58, -swing, 0.05, 0.05, 0.05, look.skin},
		)
	}
	for _, p := range parts {
		b.outline(p.side, p.y, p.forward, p.width, p.height, p.depth)
	}
	for _, p := range parts {
		b.part(p.side, p.y, p.forward, p.width, p.height, p.depth, p.ink)
	}
	switch facing {
	case world.FacingDown:
		b.eye(-0.06)
		b.eye(0.06)
	case world.FacingLeft, world.FacingRight:
		b.eye(0.03)
	}
}
