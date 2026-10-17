package terminal

import (
	"cmp"
	"math"
	"slices"

	"lapresregne/internal/world"
)

type vec struct{ x, y, z float64 }

func (a vec) mul(s float64) vec { return vec{a.x * s, a.y * s, a.z * s} }
func (a vec) dot(b vec) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }
func (a vec) unit() vec         { return a.mul(1 / math.Sqrt(a.dot(a))) }
func (a vec) sub(b vec) vec     { return vec{a.x - b.x, a.y - b.y, a.z - b.z} }
func (a vec) cross(b vec) vec {
	return vec{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x}
}

func (c *forestCanvas) shadow(tree world.Tree) {
	c.standOn(tree.X+tree.Height*0.55, tree.Z+tree.Height*0.3)
	x, y, _ := c.project(tree.X+tree.Height*0.55, 0, tree.Z+tree.Height*0.3)
	rx, ry := tree.Radius*c.scale*1.3, tree.Radius*c.scale*0.72
	for py := max(0, int(y-ry)); py <= min(c.height-1, int(y+ry)); py++ {
		for px := max(0, int(x-rx)); px <= min(c.width-1, int(x+rx)); px++ {
			u, v := (float64(px)-x)/rx, (float64(py)-y)/ry
			if d := u*u + v*v; d < 1 {
				p := &c.pixels[py*c.width+px]
				p.light *= 0.42 + d*0.36
				if p.ink == 7 {
					p.ink = 0
				}
			}
		}
	}
}

// ellipsoid ray-casts an ellipsoid, turned by yaw about its upright axis, onto the canvas and visits
// each pixel where its front surface is nearer than what is already drawn, with that surface point
// relative to the centre and its normal, both in the world. At yaw 0, rx lies along x and rz along z.
func ellipsoid(c *forestCanvas, x, y, z, rx, ry, rz, yaw float64, visit func(px, py int, depth float64, point, normal vec)) {
	sx, sy, depth := c.project(x, y, z)
	// The ray cast works in the view's frame, where the ellipsoid is turned by the camera too.
	turned := yaw + c.yaw
	sin, cos := math.Sincos(turned)
	ownSin, ownCos := math.Sincos(yaw)
	viewSin, viewCos := c.yawSin, c.yawCos
	reach := rx
	if turned != 0 {
		reach = math.Max(rx, rz)
	}
	boundX := reach * c.scale
	boundY := math.Sqrt(ry*ry*0.69*0.69+reach*reach*0.72*0.72) * c.scale
	// The view ray (0, 0.72, 0.69) in the ellipsoid's own frame, scaled by its radii.
	dx, dy, dz := -0.69*sin/rx, 0.72/ry, 0.69*cos/rz
	a := dx*dx + dy*dy + dz*dz
	// No point of the surface is nearer than this, so a pixel already holding something nearer is
	// skipped before any ray casting.
	nearest := depth - max(rx, ry, rz)
	for py := max(0, int(sy-boundY)-1); py <= min(c.height-1, int(sy+boundY)+1); py++ {
		for px := max(0, int(sx-boundX)-1); px <= min(c.width-1, int(sx+boundX)+1); px++ {
			if c.pixels[py*c.width+px].depth < nearest {
				continue
			}
			u, v := (float64(px)-sx)/c.scale, (float64(py)-sy)/c.scale
			ox, oy, oz := (u*cos-0.72*v*sin)/rx, -0.69*v/ry, (u*sin+0.72*v*cos)/rz
			b := 2 * (ox*dx + oy*dy + oz*dz)
			d := ox*ox + oy*oy + oz*oz - 1
			discriminant := b*b - 4*a*d
			if discriminant < 0 {
				continue
			}
			t := (-b + math.Sqrt(discriminant)) / (2 * a)
			if depth-t > c.pixels[py*c.width+px].depth {
				continue
			}
			// The gradient in the ellipsoid's frame, turned back into the world.
			lx, ly, lz := (ox+t*dx)/rx, (oy+t*dy)/ry, (oz+t*dz)/rz
			normal := vec{lx*ownCos + lz*ownSin, ly, -lx*ownSin + lz*ownCos}.unit()
			// The surface point across and toward the viewer, turned back into the world.
			across, toward := u, v*0.72+t*0.69
			point := vec{across*viewCos - toward*viewSin, -v*0.69 + t*0.72, across*viewSin + toward*viewCos}
			visit(px, py, depth-t, point, normal)
		}
	}
}

// leafClumps is a 0 to 1 pattern of rounded clumps of leaves about a world unit across, with
// finer leaves within each clump.
func leafClumps(x, y, z float64) float64 {
	broad := wave(3.7*x+1.3*y) * wave(2.9*y+3.3*z) * wave(3.1*z+2.3*x)
	fine := wave(11*x+5*y) * wave(9*y+12*z) * wave(10*z+7*x)
	return min(1, max(0, 0.5+0.75*broad+0.3*fine))
}

// sunlight is how brightly the sun lights a surface facing along normal.
func sunlight(normal vec) float64 {
	return 0.34 + 0.6*max(0, normal.dot(vec{-0.52, 0.80, -0.29}))
}

// rim fades a surface toward its silhouette, leaving a dotless outline so each shape reads in braille.
func (c *forestCanvas) rim(normal vec) float64 {
	facing := math.Abs(normal.dot(c.viewer()))
	return min(1, max(0, (facing-0.12)/0.3))
}

// crown draws a rounded crown of leaves with real depth and lighting.
func (r *renderer) crown(c *forestCanvas, x, y, z, rx, ry, rz float64, leaves foliageInks) {
	f := r.world
	heroWorldX, heroWorldZ := f.Hero().Position()
	heroX, heroY, _ := c.project(heroWorldX, 0.9, heroWorldZ)
	ellipsoid(c, x, y, z, rx, ry, rz, 0, func(px, py int, depth float64, point, normal vec) {
		light := (sunlight(normal) + 0.04*wave(point.x*11+point.y*8+point.z*7)) * c.rim(normal)
		// Up close the crown breaks into clumps of leaves, with gaps showing what lies behind.
		if c.scale >= detailScale {
			clump := leafClumps(x+point.x, y+point.y, z+point.z)
			if clump < 0.2 {
				return
			}
			light *= 0.5 + 0.65*clump
		}
		ink := leaves.shaded
		if light > 0.58 {
			ink = leaves.sunlit
		}
		// Clear the traveler's silhouette so leaves cannot speckle their clothing.
		hx, hy := (float64(px)-heroX)/(7*c.scale/forestScale), (float64(py)-heroY)/(9*c.scale/forestScale)
		if hx*hx+hy*hy < 1 {
			return
		}
		c.plot(px, py, depth, light, ink)
	})
}

// crownShape is the centre and radii of one rounded crown of leaves.
type crownShape struct{ x, y, z, rx, ry, rz float64 }

// crowns draws a tree's crowns nearest first, so each farther crown skips the leaves in front of it.
func (r *renderer) crowns(c *forestCanvas, shapes []crownShape, leaves foliageInks) {
	slices.SortFunc(shapes, func(a, b crownShape) int {
		_, _, aDepth := c.project(a.x, a.y, a.z)
		_, _, bDepth := c.project(b.x, b.y, b.z)
		return cmp.Compare(aDepth, bDepth)
	})
	for _, s := range shapes {
		r.crown(c, s.x, s.y, s.z, s.rx, s.ry, s.rz, leaves)
	}
}

// foliageInks are the palette entries a crown uses in shade and in sunlight.
type foliageInks struct{ shaded, sunlit int }

// leavesOf gives willows pale, soft leaves, yews blue-green needles and oaks bronzed leaves.
func leavesOf(kind world.TreeKind) foliageInks {
	switch kind {
	case world.Willow:
		return foliageInks{11, 12}
	case world.Yew:
		return foliageInks{13, 14}
	case world.Oak:
		return foliageInks{15, 16}
	}
	return foliageInks{2, 3}
}

// drawTree draws a tree in its species' silhouette; shake leans its top sideways as an axe strikes it.
func (r *renderer) drawTree(c *forestCanvas, tree world.Tree, shake float64) {
	if tree.Stump {
		drawStump(c, tree)
		return
	}
	switch tree.Kind {
	case world.Oak:
		r.drawOak(c, tree, shake)
	case world.Willow:
		r.drawWillow(c, tree, shake)
	case world.Yew:
		r.drawYew(c, tree, shake)
	case world.DeadTree:
		drawDeadTree(c, tree, shake)
	default:
		r.drawCommonTree(c, tree, shake)
	}
}

// drawCommonTree raises three rounded crowns around a taller top crown.
func (r *renderer) drawCommonTree(c *forestCanvas, tree world.Tree, shake float64) {
	x, z, h, radius := tree.X, tree.Z, tree.Height, tree.Radius
	leaves := leavesOf(tree.Kind)
	a := tree.Variation * 6.28
	var crowns [4]crownShape
	var branches [3]limbShape
	for i := range 3 {
		angle := a + float64(i)*2.094
		dx, dz := math.Cos(angle)*radius*0.62+shake, math.Sin(angle)*radius*0.62
		branches[i] = limbShape{x + shake*0.45, h * 0.45, z, x + dx, h * 0.92, z + dz, 0.7, tree.Trunk * 0.55}
		crowns[i] = crownShape{x + dx, h * (0.84 + float64(i)*0.05), z + dz, radius * 0.72, radius * 0.57, radius * 0.72}
	}
	crowns[3] = crownShape{x + shake, h + 0.3, z, radius * 0.83, radius * 0.70, radius * 0.83}
	// The leaves go first and the wood they mostly hide after, so hidden wood is never shaded.
	r.crowns(c, crowns[:], leaves)
	drawBranches(c, branches[:])
	drawTrunk(c, tree, tree.Trunk, h, shake)
}

// drawOak spreads a thick, short trunk into a broad, flat-topped canopy of many lobes.
func (r *renderer) drawOak(c *forestCanvas, tree world.Tree, shake float64) {
	x, z, h, radius := tree.X, tree.Z, tree.Height, tree.Radius
	leaves := leavesOf(tree.Kind)
	a := tree.Variation * 6.28
	var crowns [6]crownShape
	var branches [5]limbShape
	for i := range 5 {
		angle := a + float64(i)*1.257
		reach := radius * (0.9 + 0.15*math.Sin(angle*3))
		dx, dz := math.Cos(angle)*reach+shake*0.8, math.Sin(angle)*reach*0.45
		branches[i] = limbShape{x + shake*0.4, h * 0.6, z, x + dx*0.85, h * 0.88, z + dz*0.85, 0.66, tree.Trunk * 0.7}
		crowns[i] = crownShape{x + dx, h * (0.92 + float64(i%2)*0.06), z + dz, radius * 0.62, radius * 0.42, radius * 0.62}
	}
	crowns[5] = crownShape{x + shake, h * 1.08, z, radius * 1.05, radius * 0.5, radius * 0.95}
	r.crowns(c, crowns[:], leaves)
	drawBranches(c, branches[:])
	drawTrunk(c, tree, tree.Trunk*1.5, h*0.85, shake)
}

// drawWillow forks a trunk into a few branches whose tips spill long strands that arch over
// briefly, then hang straight down nearly to the ground.
func (r *renderer) drawWillow(c *forestCanvas, tree world.Tree, shake float64) {
	x, z, h, radius := tree.X, tree.Z, tree.Height, tree.Radius
	leaves := leavesOf(tree.Kind)
	a := tree.Variation * 6.28
	const strands = 34
	for i := range strands {
		angle := a + float64(i)*2.399
		cos, sin := math.Cos(angle), math.Sin(angle)
		vary := math.Abs(math.Sin(float64(i)*1.7 + tree.Variation*5))
		start := radius * (0.1 + 0.45*vary)
		// Strands rooted farther out start lower, so the crest slopes away from the trunk.
		top := h * (1.1 - 0.35*vary + 0.05*math.Abs(math.Sin(float64(i)*0.9)))
		end := h * (0.06 + 0.22*math.Abs(math.Sin(float64(i)*2.3+tree.Variation*9)))
		ink := leaves.shaded
		if cos < 0.3 && sin < 0.4 && i%3 != 0 {
			ink = leaves.sunlit
		}
		// A short arch over the strand's crest, then a long fall that drifts only slightly outward.
		fromX, fromZ := x+shake+cos*start, z+sin*start
		crestX, crestZ := x+shake+cos*(start+radius*0.18), z+sin*(start+radius*0.18)
		bendX, bendZ := x+shake*0.8+cos*(start+radius*0.3), z+sin*(start+radius*0.3)
		footX, footZ := x+shake*0.3+cos*(start+radius*0.38), z+sin*(start+radius*0.38)
		vine(c, fromX, top-h*0.03, fromZ, crestX, top+h*0.03, crestZ, ink)
		vine(c, crestX, top+h*0.03, crestZ, bendX, top-h*0.1, bendZ, ink)
		vine(c, bendX, top-h*0.1, bendZ, footX, end, footZ, ink)
	}
	// The strands hang in front of the wood, so it goes after them.
	var branches [4]limbShape
	for i := range 4 {
		angle := a + float64(i)*1.571
		tipX, tipZ := x+shake+math.Cos(angle)*radius*0.4, z+math.Sin(angle)*radius*0.4
		branches[i] = limbShape{x + shake*0.6, h * 0.6, z, tipX, h, tipZ, 0.7, tree.Trunk * 0.6}
	}
	drawBranches(c, branches[:])
	drawTrunk(c, tree, tree.Trunk, h*0.6, shake*0.6)
}

// vine hangs a willow strand between two points: a line of leaves in the forest view, and up close
// a string of small, sunlit leaves that alternate from side to side.
func vine(c *forestCanvas, x1, y1, z1, x2, y2, z2 float64, ink int) {
	if c.scale < detailScale {
		c.line(x1, y1, z1, x2, y2, z2, 0.95, ink)
		return
	}
	length := math.Sqrt((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1) + (z2-z1)*(z2-z1))
	steps := max(1, int(math.Ceil(length/0.08)))
	for i := range steps + 1 {
		u := float64(i) / float64(steps)
		sway := 0.035 * float64(1-2*(i%2))
		x, y, z := x1+(x2-x1)*u+sway, y1+(y2-y1)*u, z1+(z2-z1)*u
		ellipsoid(c, x, y, z, 0.035, 0.06, 0.035, 0, func(px, py int, depth float64, _, normal vec) {
			c.plot(px, py, depth, 0.45+0.55*max(0, normal.dot(vec{-0.52, 0.80, -0.29})), ink)
		})
	}
}

// drawYew stacks dark tiers of needles that narrow toward a tall, pointed top.
func (r *renderer) drawYew(c *forestCanvas, tree world.Tree, shake float64) {
	x, z, h, radius := tree.X, tree.Z, tree.Height, tree.Radius
	leaves := leavesOf(tree.Kind)
	const tiers = 5
	var crowns [tiers]crownShape
	for i := range tiers {
		rise := float64(i) / (tiers - 1)
		width := radius * (1.05 - 0.75*rise)
		lean := shake * (0.4 + 0.6*rise)
		crowns[i] = crownShape{x + lean, h * (0.55 + 0.75*rise), z, width, width * 0.5, width}
	}
	r.crowns(c, crowns[:], leaves)
	drawTrunk(c, tree, tree.Trunk*1.2, h, shake)
}

// drawDeadTree raises a bare, gnarled trunk that forks into crooked limbs, each splitting again
// into thin twigs, with no leaves at all.
func drawDeadTree(c *forestCanvas, tree world.Tree, shake float64) {
	x, z, h, radius := tree.X, tree.Z, tree.Height, tree.Radius
	a := tree.Variation * 6.28
	var limbs [12]limbShape
	for i := range 4 {
		angle := a + float64(i)*1.571 + 0.3*math.Sin(float64(i)*2.7)
		lean := shake * (0.5 + 0.1*float64(i))
		startY := h * (0.45 + 0.12*float64(i%2))
		reach := radius * (0.55 + 0.25*math.Abs(math.Sin(float64(i)*1.9+tree.Variation*4)))
		// Each limb bends once, then forks into two twigs.
		elbowX, elbowZ := x+lean+math.Cos(angle)*reach*0.6, z+math.Sin(angle)*reach*0.6
		elbowY := startY + h*0.25
		limbs[i*3] = limbShape{x + lean*0.5, startY, z, elbowX, elbowY, elbowZ, 0.62, tree.Trunk * 0.5}
		for twig := range 2 {
			fork := angle + 0.6*float64(2*twig-1)
			tipX, tipZ := elbowX+math.Cos(fork)*reach*0.45, elbowZ+math.Sin(fork)*reach*0.45
			limbs[i*3+1+twig] = limbShape{elbowX, elbowY, elbowZ, tipX, elbowY + h*(0.18+0.1*float64(twig)), tipZ, 0.55, tree.Trunk * 0.25}
		}
	}
	drawBranches(c, limbs[:])
	drawTrunk(c, tree, tree.Trunk*0.9, h*0.8, shake)
}

// drawTrunk raises a trunk, its top leaning with shake: three bark strands in the forest view,
// and up close solid, furrowed wood flaring into roots at the ground.
func drawTrunk(c *forestCanvas, tree world.Tree, thickness, height, shake float64) {
	if c.scale < detailScale {
		for _, offset := range []float64{-thickness, 0, thickness} {
			c.line(tree.X+offset, 0, tree.Z, tree.X+offset*0.6+shake, height, tree.Z, 0.62, 4)
		}
		return
	}
	x, z := tree.X, tree.Z
	wood(c, x, 0, z, x+shake, height, z, thickness*1.15, thickness*0.6)
	wood(c, x, 0.12, z, x, 0.45, z, thickness*1.6, thickness*1.15)
	for i := range 4 {
		angle := tree.Variation*6.28 + float64(i)*1.571 + 0.4
		reach := thickness * (2.6 + 0.8*math.Sin(float64(i)*2.1+tree.Variation*7))
		wood(c, x, 0.3, z, x+math.Cos(angle)*reach, 0.02, z+math.Sin(angle)*reach, thickness*0.55, thickness*0.2)
	}
}

// limbShape is a branch from one point to another, with its light and thickness.
type limbShape struct{ x1, y1, z1, x2, y2, z2, light, thickness float64 }

func drawBranches(c *forestCanvas, limbs []limbShape) {
	for _, l := range limbs {
		branch(c, l.x1, l.y1, l.z1, l.x2, l.y2, l.z2, l.light, l.thickness)
	}
}

// branch reaches from one point to another: a bark stroke in the forest view, and up close a
// limb of wood tapering from thickness to half of it.
func branch(c *forestCanvas, x1, y1, z1, x2, y2, z2, light, thickness float64) {
	if c.scale < detailScale {
		c.line(x1, y1, z1, x2, y2, z2, light, 4)
		return
	}
	wood(c, x1, y1, z1, x2, y2, z2, thickness, thickness*0.5)
}

// wood tapers a chain of overlapping spheres between two points into a solid, sunlit limb, its
// bark furrowed lengthwise.
func wood(c *forestCanvas, x1, y1, z1, x2, y2, z2, fromRadius, toRadius float64) {
	length := math.Sqrt((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1) + (z2-z1)*(z2-z1))
	axis := vec{x2 - x1, y2 - y1, z2 - z1}.mul(1 / length)
	steps := max(1, int(math.Ceil(length/(math.Min(fromRadius, toRadius)*0.5))))
	// The spheres go nearest first, so each farther one skips the pixels already covered.
	_, _, fromDepth := c.project(x1, y1, z1)
	_, _, toDepth := c.project(x2, y2, z2)
	for i := range steps + 1 {
		u := float64(i) / float64(steps)
		if toDepth < fromDepth {
			u = 1 - u
		}
		r := fromRadius + (toRadius-fromRadius)*u
		x, y, z := x1+(x2-x1)*u, y1+(y2-y1)*u, z1+(z2-z1)*u
		ellipsoid(c, x, y, z, r, r, r, 0, func(px, py int, depth float64, point, sphere vec) {
			// Shade the chain as one smooth limb: its normal points straight out from the axis.
			normal := sphere
			if along := sphere.dot(axis); along < 0.98 && along > -0.98 {
				normal = vec{sphere.x - axis.x*along, sphere.y - axis.y*along, sphere.z - axis.z*along}.unit()
			}
			// Deep, slightly wandering furrows run along the bark between raised ridges.
			ridge := wave(math.Atan2(normal.x, normal.z)*11 + wave((y+point.y)*2.3)*1.2)
			furrows := 0.45 + 0.55*math.Sqrt(max(0, ridge*0.5+0.5))
			// Light wraps around the bark from the sun's side, so upright wood is never left unlit.
			side := 0.5 + 0.5*normal.dot(vec{-0.87, 0.2, -0.45})
			c.plot(px, py, depth, (0.25+0.75*side)*furrows, 4)
		})
	}
}

// stumpHeight keeps a stump a few braille dots tall.
const stumpHeight = 1.1

// drawStump leaves a short trunk with a pale, freshly cut top.
func drawStump(c *forestCanvas, tree world.Tree) {
	x, z, trunk := tree.X, tree.Z, tree.Trunk*1.3
	if c.scale >= detailScale {
		drawTrunk(c, tree, tree.Trunk, stumpHeight, 0)
		// Growth rings show faintly across the cut.
		ellipsoid(c, x, stumpHeight, z, tree.Trunk*0.66, 0.03, tree.Trunk*0.66, 0, func(px, py int, depth float64, point, _ vec) {
			rings := 0.8 + 0.15*math.Sin(math.Hypot(point.x, point.z)/tree.Trunk*28)
			c.plot(px, py, depth, rings, 7)
		})
		return
	}
	for _, offset := range []float64{-trunk, 0, trunk} {
		c.line(x+offset, 0, z, x+offset, stumpHeight, z, 0.62, 4)
	}
	c.line(x-trunk, stumpHeight, z, x+trunk, stumpHeight, z, 0.9, 7)
}

// drawGroundItem lays a dropped item on the ground in its shape.
func drawGroundItem(c *forestCanvas, item world.GroundItem) {
	switch item.Item.Shape() {
	case world.LogShape:
		drawGroundLog(c, item.X, item.Z)
	}
}

// drawGroundLog lays a log on the ground, its pale cut end facing the viewer's right.
func drawGroundLog(c *forestCanvas, x, z float64) {
	c.line(x-0.35, 0.1, z, x+0.25, 0.1, z, 0.62, 4)
	c.line(x-0.35, 0.2, z, x+0.25, 0.2, z, 0.5, 4)
	c.line(x+0.3, 0.08, z, x+0.3, 0.22, z, 0.9, 7)
}

func (r *renderer) drawGrass(c *forestCanvas, left, right, back, front, seconds float64) {
	f := r.world
	for gz := int(math.Floor(back)); gz <= int(math.Ceil(front)); gz++ {
		for gx := int(math.Floor(left)); gx <= int(math.Ceil(right)); gx++ {
			// Zoomed out, fewer blades sway, keeping the ground calm.
			if f.Random(gx, gz, 35) > c.quiet() {
				continue
			}
			chance := f.Random(gx, gz, 31)
			x := float64(gx) + f.Random(gx, gz, 32)
			z := float64(gz) + f.Random(gx, gz, 33)
			// Skip offscreen roots before evaluating the more expensive biome fields.
			c.standOn(x, z)
			sx, sy, _ := c.project(x, 0, z)
			if sx < -3 || sx > float64(c.width)+3 || sy < 0 || sy > float64(c.height)+12 {
				continue
			}
			// Grass grows on a map's grass tiles, and beside the endless forest's rivers.
			distance, width := math.Inf(1), 0.0
			if f.Mapped() {
				if ground, onMap := f.GroundAt(x, z); !onMap || ground != world.Grass {
					continue
				}
			} else {
				distance, width = world.NearestRiver(r.rivers.at(z), x)
			}
			if distance < width {
				continue
			}
			cover := f.Cover(x, z)
			tall := cover < 0.38 || distance < width+1.5
			if !tall && chance > 0.10 || tall && chance > 0.78 {
				continue
			}
			height := 0.35 + chance*0.65
			if tall {
				height = 1.2 + chance*1.5
			}
			sway := math.Sin(seconds*1.5+x*0.4+z*0.3) * 0.16
			light := 0.46 + f.Random(gx, gz, 34)*0.4
			c.line(x, 0, z, x+sway*0.5, height*0.6, z, light*0.8, 1)
			c.line(x+sway*0.5, height*0.6, z, x+sway+0.16, height, z+0.08, light, 1)
			c.line(x+sway*0.5, height*0.55, z, x-0.22+sway, height*0.85, z-0.04, light*0.8, 1)
		}
	}
}
