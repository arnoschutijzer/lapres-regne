package terminal

import "math"

// detailScale is the zoom from which trees are drawn in detail.
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
	// bob lifts the whole body, and lean tips everything above the hips forward.
	bob, lean float64
	// side and forward are unit directions across the body to its right and out of its front.
	sideX, sideZ, forwardX, forwardZ float64
}

func newHeroBody(c *forestCanvas, x, z, heading float64) heroBody {
	cos, sin := math.Cos(heading), math.Sin(heading)
	return heroBody{c: c, x: x, z: z, heading: heading, sideX: cos, sideZ: -sin, forwardX: sin, forwardZ: cos}
}

// carry applies the body's bob and lean to a point at height y and forward offset.
func (b heroBody) carry(y, forward float64) (float64, float64) {
	if y > 0.86 {
		forward += b.lean * (y - 0.86)
	}
	return y + b.bob, forward
}

// at turns an offset to the hero's side and front into a world position.
func (b heroBody) at(side, forward float64) (x, z float64) {
	return b.x + b.sideX*side + b.forwardX*forward, b.z + b.sideZ*side + b.forwardZ*forward
}

// part draws one rounded, sunlit body part with its own width, height and depth.
func (b heroBody) part(side, y, forward, width, height, depth float64, ink int) {
	x, z := b.at(side, forward)
	width, height, depth = b.visible(width), b.visible(height), b.visible(depth)
	ellipsoid(b.c, x, y, z, width, height, depth, b.heading, func(px, py int, depth float64, _, normal vec) {
		// Shading flows smoothly across parts; only the outline behind the figure marks its edge.
		light := 0.6 + 0.4*max(0, normal.dot(vec{-0.52, 0.80, -0.29}))
		// A small, distant hero is lit more fully, so their few dots form a solid figure.
		light += 0.3 * math.Max(0, 1-b.c.scale/detailScale)
		b.c.plot(px, py, depth, light, ink)
	})
}

// outline darkens a thin band around a body part, set just behind the hero, so the figure
// stands out from the dithered ground.
func (b heroBody) outline(side, y, forward, width, height, depth float64) {
	x, z := b.at(side, forward)
	band := 1.5 / b.c.scale
	width, height, depth = b.visible(width), b.visible(height), b.visible(depth)
	ellipsoid(b.c, x, y, z, width+band, height+band, depth+band, b.heading, func(px, py int, depth float64, _, _ vec) {
		b.c.plot(px, py, depth+0.4+band, 0, 0)
	})
}

// visible keeps a part's radius at least about a canvas pixel, so slender limbs stay unbroken
// when the view zooms out.
func (b heroBody) visible(radius float64) float64 {
	return math.Max(radius, 0.6/b.c.scale)
}

// eye leaves an unlit gap two dots tall on the face, when that side of the face turns toward the view.
func (b heroBody) eye(side float64) {
	const forward = 0.09
	// The head's surface normal at the eye, turned into the world, must face the camera.
	normalSide, normalForward := side/(0.085*0.085), forward/(0.095*0.095)
	length := math.Hypot(normalSide, normalForward)
	normalX, normalZ := normalSide*b.sideX+normalForward*b.forwardX, normalSide*b.sideZ+normalForward*b.forwardZ
	viewer := b.c.viewer()
	if (normalX*viewer.x+normalZ*viewer.z)/(0.69*length) < 0.15 {
		return
	}
	y, carried := b.carry(1.54, forward)
	x, z := b.at(side, carried)
	px, py, depth := b.c.project(x, y, z)
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
// hair, a plain undershirt and shorts, and bare feet, posed by their gait. Legs swing from the
// hips with the knee lifting as each passes through, and arms swing against them.
func drawDetailedHero(c *forestCanvas, x, z float64, g gait, look heroLook) {
	b := newHeroBody(c, x, z, g.heading)
	// Wider strides at speed; the knee lifts as a leg swings through, and running arms bend.
	reach := g.amount * math.Min(0.45, 0.22+0.02*g.speed)
	elbowBend := 0.15 + 0.5*g.amount
	var parts []bodyPart
	for i, side := range []float64{-1, 1} {
		phase := g.phase + float64(i)*math.Pi
		angle := reach * math.Sin(phase)
		knee := g.amount * (0.1 + 0.65*math.Max(0, math.Cos(phase)))
		hip := joint{side * 0.075, 0.86, 0}
		kneeJoint := bend(hip, side*0.005, angle, 0.39)
		ankle := bend(kneeJoint, 0, angle-knee, 0.38)
		hem := along(hip, kneeJoint, 0.36)
		parts = limb(parts, hip, hem, 0.07, 0.066, look.shorts)
		parts = limb(parts, hem, kneeJoint, 0.06, 0.046, look.skin)
		parts = limb(parts, kneeJoint, ankle, 0.045, 0.032, look.skin)
		parts = append(parts, bodyPart{ankle.side, ankle.y - 0.02, ankle.forward + 0.04, 0.035, 0.028, 0.075, look.skin})

		armAngle := -angle * 0.6
		shoulder := joint{side * 0.18, 1.34, 0}
		elbow := bend(shoulder, side*0.025, armAngle, 0.28)
		wrist := bend(elbow, 0, armAngle+elbowBend, 0.25)
		sleeve := along(shoulder, elbow, 0.32)
		parts = limb(parts, shoulder, sleeve, 0.05, 0.045, look.undershirt)
		parts = limb(parts, sleeve, elbow, 0.04, 0.034, look.skin)
		parts = limb(parts, elbow, wrist, 0.033, 0.026, look.skin)
		hand := bend(wrist, 0, armAngle+elbowBend, 0.05)
		parts = append(parts, bodyPart{hand.side, hand.y, hand.forward, 0.024, 0.05, 0.03, look.skin})
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
	// The body rises and falls twice a cycle, once per footfall, and leans into its stride.
	b.bob = g.amount * 0.025 * math.Cos(2*g.phase)
	b.lean = g.amount * 0.1
	for i := range parts {
		parts[i].y, parts[i].forward = b.carry(parts[i].y, parts[i].forward)
	}
	for _, p := range parts {
		b.outline(p.side, p.y, p.forward, p.width, p.height, p.depth)
	}
	for _, p := range parts {
		b.part(p.side, p.y, p.forward, p.width, p.height, p.depth, p.ink)
	}
	// Eyes need a head a few dots across, or their gaps would erase the face.
	if 0.085*c.scale >= 2 {
		b.eye(-0.032)
		b.eye(0.032)
	}
}
