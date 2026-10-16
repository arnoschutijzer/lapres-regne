package terminal

import "math"

// detailScale is the zoom from which the hero is drawn in detail.
var detailScale = forestScale * math.Pow(zoomStep, 6)

// heroLook holds the palette entries of the hero's base body: their skin and hair, and the plain
// undershirt and shorts that gear will later cover.
type heroLook struct{ skin, hair, undershirt, shorts int }

var defaultLook = heroLook{skin: 17, hair: 18, undershirt: 19, shorts: 20}

// eyeInk marks the eyes, drawn as unlit gaps in the face.
const eyeInk = 21

// heroBody places the hero's body parts around their feet, turned to their heading.
type heroBody struct {
	c       *forestCanvas
	x, z    float64
	heading float64
	// side and forward are unit directions across the body to its right and out of its front.
	sideX, sideZ, forwardX, forwardZ float64
}

func newHeroBody(c *forestCanvas, x, z, heading float64) heroBody {
	cos, sin := math.Cos(heading), math.Sin(heading)
	return heroBody{c: c, x: x, z: z, heading: heading, sideX: cos, sideZ: -sin, forwardX: sin, forwardZ: cos}
}

// at turns an offset to the hero's side and front into a world position.
func (b heroBody) at(side, forward float64) (x, z float64) {
	return b.x + b.sideX*side + b.forwardX*forward, b.z + b.sideZ*side + b.forwardZ*forward
}

// part draws one rounded, sunlit body part with its own width, height and depth.
func (b heroBody) part(side, y, forward, width, height, depth float64, ink int) {
	x, z := b.at(side, forward)
	ellipsoid(b.c, x, y, z, width, height, depth, b.heading, func(px, py int, depth float64, _, normal vec) {
		// Shading flows smoothly across parts; only the outline behind the figure marks its edge.
		light := 0.6 + 0.4*math.Max(0, normal.dot(vec{-0.52, 0.80, -0.29}))
		b.c.plot(px, py, depth, light, ink)
	})
}

// outline darkens a thin band around a body part, set just behind the hero, so the figure
// stands out from the dithered ground.
func (b heroBody) outline(side, y, forward, width, height, depth float64) {
	x, z := b.at(side, forward)
	band := 1.5 / b.c.scale
	ellipsoid(b.c, x, y, z, width+band, height+band, depth+band, b.heading, func(px, py int, depth float64, _, _ vec) {
		b.c.plot(px, py, depth+0.4, 0, 0)
	})
}

// eye leaves an unlit gap two dots tall on the face, when that side of the face turns toward the view.
func (b heroBody) eye(side float64) {
	const forward = 0.09
	// The head's surface normal at the eye, turned into the world, must face the camera.
	normalSide, normalForward := side/(0.085*0.085), forward/(0.095*0.095)
	length := math.Hypot(normalSide, normalForward)
	if (normalSide*b.sideZ+normalForward*b.forwardZ)/length < 0.15 {
		return
	}
	x, z := b.at(side, forward)
	px, py, depth := b.c.project(x, 1.54, z)
	for dy := range 2 {
		b.c.plot(int(math.Round(px)), int(math.Round(py))+dy, depth-0.05, 0, eyeInk)
	}
}

// bodyPart is one rounded piece of the hero, placed beside, above and in front of their feet.
type bodyPart struct {
	side, y, forward, width, height, depth float64
	ink                                    int
}

// joint is a point on the hero's body, beside, above and in front of their feet.
type joint struct{ side, y, forward float64 }

// limb tapers a chain of small spheres from one joint to the next, so arms and legs stay slender.
func limb(parts []bodyPart, from, to joint, fromRadius, toRadius float64, ink int) []bodyPart {
	length := math.Sqrt(math.Pow(to.side-from.side, 2) + math.Pow(to.y-from.y, 2) + math.Pow(to.forward-from.forward, 2))
	steps := max(2, int(math.Ceil(length/(math.Min(fromRadius, toRadius)*0.7))))
	for i := range steps + 1 {
		u := float64(i) / float64(steps)
		r := fromRadius + (toRadius-fromRadius)*u
		parts = append(parts, bodyPart{
			from.side + (to.side-from.side)*u, from.y + (to.y-from.y)*u, from.forward + (to.forward-from.forward)*u,
			r, r, r, ink,
		})
	}
	return parts
}

// along finds the point a fraction u of the way from one joint to the next.
func along(from, to joint, u float64) joint {
	return joint{from.side + (to.side-from.side)*u, from.y + (to.y-from.y)*u, from.forward + (to.forward-from.forward)*u}
}

// bend reaches length from a joint at angle from straight down, leaning forward for positive angles.
func bend(from joint, side, angle, length float64) joint {
	return joint{from.side + side, from.y - length*math.Cos(angle), from.forward + length*math.Sin(angle)}
}

// drawDetailedHero draws the hero's base body with realistic, slender proportions: skin, short
// hair, a plain undershirt and shorts, and bare feet. Legs swing from the hips with the knee
// bending on the back swing, and arms swing against them.
func drawDetailedHero(c *forestCanvas, x, z, heading, stride float64, look heroLook) {
	b := newHeroBody(c, x, z, heading)
	swing := stride / 0.24 * 0.22
	var parts []bodyPart
	for _, side := range []float64{-1, 1} {
		angle := swing * side
		hip := joint{side * 0.075, 0.86, 0}
		knee := bend(hip, side*0.005, angle, 0.39)
		ankle := bend(knee, 0, angle-math.Max(0, -angle)*0.9, 0.38)
		hem := along(hip, knee, 0.36)
		parts = limb(parts, hip, hem, 0.07, 0.066, look.shorts)
		parts = limb(parts, hem, knee, 0.06, 0.046, look.skin)
		parts = limb(parts, knee, ankle, 0.045, 0.032, look.skin)
		parts = append(parts, bodyPart{ankle.side, ankle.y - 0.02, ankle.forward + 0.04, 0.035, 0.028, 0.075, look.skin})

		shoulder := joint{side * 0.18, 1.34, 0}
		elbow := bend(shoulder, side*0.025, -angle*0.8, 0.28)
		wrist := bend(elbow, 0, -angle*0.8+0.15, 0.25)
		sleeve := along(shoulder, elbow, 0.32)
		parts = limb(parts, shoulder, sleeve, 0.05, 0.045, look.undershirt)
		parts = limb(parts, sleeve, elbow, 0.04, 0.034, look.skin)
		parts = limb(parts, elbow, wrist, 0.033, 0.026, look.skin)
		parts = append(parts, bodyPart{wrist.side, wrist.y - 0.05, wrist.forward, 0.024, 0.05, 0.03, look.skin})
	}
	parts = append(parts,
		// Hips, waist and chest taper and widen like a real torso.
		bodyPart{0, 0.9, 0, 0.14, 0.09, 0.085, look.shorts},
		bodyPart{0, 1.04, 0, 0.12, 0.1, 0.075, look.undershirt},
		bodyPart{0, 1.2, 0, 0.13, 0.12, 0.085, look.undershirt},
		bodyPart{0, 1.31, 0, 0.16, 0.045, 0.065, look.undershirt},
		bodyPart{0, 1.41, 0, 0.035, 0.05, 0.035, look.skin},
		bodyPart{0, 1.53, 0.005, 0.085, 0.105, 0.095, look.skin},
		// Short hair caps the top and back of the head, leaving the face bare.
		bodyPart{0, 1.59, -0.03, 0.088, 0.06, 0.09, look.hair},
	)
	for _, p := range parts {
		b.outline(p.side, p.y, p.forward, p.width, p.height, p.depth)
	}
	for _, p := range parts {
		b.part(p.side, p.y, p.forward, p.width, p.height, p.depth, p.ink)
	}
	b.eye(-0.032)
	b.eye(0.032)
}
