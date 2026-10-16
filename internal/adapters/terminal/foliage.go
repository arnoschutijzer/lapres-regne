package terminal

import (
	"math"

	"lapresregne/internal/world"
)

type vec struct{ x, y, z float64 }

func (a vec) mul(s float64) vec { return vec{a.x * s, a.y * s, a.z * s} }
func (a vec) dot(b vec) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }
func (a vec) unit() vec         { return a.mul(1 / math.Sqrt(a.dot(a))) }

func (c *forestCanvas) shadow(tree world.Tree) {
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

// Ray/ellipsoid intersections provide rounded crowns with real depth and lighting.
func (r *renderer) crown(c *forestCanvas, x, y, z, rx, ry, rz float64, leaves foliageInks) {
	f := r.world
	heroWorldX, heroWorldZ := f.Hero().Position()
	sx, sy, depth := c.project(x, y, z)
	boundX := rx * c.scale
	boundY := math.Sqrt(ry*ry*0.69*0.69+rz*rz*0.72*0.72) * c.scale
	heroX, heroY, _ := c.project(heroWorldX, 0.9, heroWorldZ)
	a := 0.72*0.72/(ry*ry) + 0.69*0.69/(rz*rz)
	for py := max(0, int(sy-boundY)-1); py <= min(c.height-1, int(sy+boundY)+1); py++ {
		for px := max(0, int(sx-boundX)-1); px <= min(c.width-1, int(sx+boundX)+1); px++ {
			u, v := (float64(px)-sx)/c.scale, (float64(py)-sy)/c.scale
			b := 2 * v * (-0.69*0.72/(ry*ry) + 0.72*0.69/(rz*rz))
			d := u*u/(rx*rx) + v*v*(0.69*0.69/(ry*ry)+0.72*0.72/(rz*rz)) - 1
			discriminant := b*b - 4*a*d
			if discriminant < 0 {
				continue
			}
			t := (-b + math.Sqrt(discriminant)) / (2 * a)
			if depth-t > c.pixels[py*c.width+px].depth {
				continue
			}
			localY, localZ := -v*0.69+t*0.72, v*0.72+t*0.69
			normal := vec{u / (rx * rx), localY / (ry * ry), localZ / (rz * rz)}.unit()
			light := 0.20 + 0.68*math.Max(0, normal.dot(vec{-0.52, 0.80, -0.29}))
			light += 0.04 * math.Sin(u*11+localY*8+localZ*7)
			ink := leaves.shaded
			if light > 0.58 {
				ink = leaves.sunlit
			}
			// A small cutaway keeps the traveler readable under overlapping crowns.
			hx, hy := (float64(px)-heroX)/7, (float64(py)-heroY)/9
			if distance := hx*hx + hy*hy; distance < 1 {
				if distance < 0.13 || f.Random(int(math.Floor(x+u*3)), int(math.Floor(z+localZ*3)), 83) > 0.18 {
					continue
				}
			}
			c.plot(px, py, depth-t, light, ink)
		}
	}
}

// foliageInks are the palette entries a crown uses in shade and in sunlight.
type foliageInks struct{ shaded, sunlit int }

// leavesOf gives willows pale, soft leaves and yews dark needles; oaks share the common green.
func leavesOf(kind world.TreeKind) foliageInks {
	switch kind {
	case world.Willow:
		return foliageInks{11, 12}
	case world.Yew:
		return foliageInks{13, 14}
	}
	return foliageInks{2, 3}
}

// drawTree draws a tree; shake leans its top sideways as an axe strikes it.
func (r *renderer) drawTree(c *forestCanvas, tree world.Tree, shake float64) {
	x, z, h, radius := tree.X, tree.Z, tree.Height, tree.Radius
	if tree.Stump {
		drawStump(c, tree)
		return
	}
	leaves := leavesOf(tree.Kind)
	for _, offset := range []float64{-tree.Trunk, 0, tree.Trunk} {
		c.line(x+offset, 0, z, x+offset*0.6+shake, h, z, 0.62, 4)
	}
	a := tree.Variation * 6.28
	for i := range 3 {
		angle := a + float64(i)*2.094
		dx, dz := math.Cos(angle)*radius*0.62+shake, math.Sin(angle)*radius*0.62
		c.line(x+shake*0.45, h*0.45, z, x+dx, h*0.92, z+dz, 0.7, 4)
		r.crown(c, x+dx, h*(0.84+float64(i)*0.05), z+dz, radius*0.72, radius*0.57, radius*0.72, leaves)
	}
	r.crown(c, x+shake, h+0.3, z, radius*0.83, radius*0.70, radius*0.83, leaves)
	if tree.Kind == world.Willow {
		drawWillowStrands(c, tree, shake)
	}
}

// drawWillowStrands hangs thin strands of leaves from the rim of a willow's crown.
func drawWillowStrands(c *forestCanvas, tree world.Tree, shake float64) {
	x, z, h, radius := tree.X, tree.Z, tree.Height, tree.Radius
	for i := range 9 {
		angle := tree.Variation*6.28 + float64(i)*0.698
		rimX, rimZ := x+shake+math.Cos(angle)*radius*0.95, z+math.Sin(angle)*radius*0.95
		c.line(rimX, h*0.8, rimZ, rimX+shake*0.5, h*0.3, rimZ, 0.45, 11)
	}
}

// stumpHeight keeps a stump a few braille dots tall.
const stumpHeight = 1.1

// drawStump leaves a short trunk with a pale, freshly cut top.
func drawStump(c *forestCanvas, tree world.Tree) {
	x, z, trunk := tree.X, tree.Z, tree.Trunk*1.3
	for _, offset := range []float64{-trunk, 0, trunk} {
		c.line(x+offset, 0, z, x+offset, stumpHeight, z, 0.62, 4)
	}
	c.line(x-trunk, stumpHeight, z, x+trunk, stumpHeight, z, 0.9, 7)
}

// drawGroundLog lays a dropped log on the ground, its pale cut end facing the viewer's right.
func drawGroundLog(c *forestCanvas, log world.GroundLog) {
	c.line(log.X-0.35, 0.1, log.Z, log.X+0.25, 0.1, log.Z, 0.62, 4)
	c.line(log.X-0.35, 0.2, log.Z, log.X+0.25, 0.2, log.Z, 0.5, 4)
	c.line(log.X+0.3, 0.08, log.Z, log.X+0.3, 0.22, log.Z, 0.9, 7)
}

func (r *renderer) drawGrass(c *forestCanvas, left, right, back, front, seconds float64) {
	f := r.world
	for gz := int(math.Floor(back)); gz <= int(math.Ceil(front)); gz++ {
		for gx := int(math.Floor(left)); gx <= int(math.Ceil(right)); gx++ {
			chance := f.Random(gx, gz, 31)
			x := float64(gx) + f.Random(gx, gz, 32)
			z := float64(gz) + f.Random(gx, gz, 33)
			// Skip offscreen roots before evaluating the more expensive biome fields.
			sx, sy, _ := c.project(x, 0, z)
			if sx < -3 || sx > float64(c.width)+3 || sy < 0 || sy > float64(c.height)+12 {
				continue
			}
			distance, width := f.River(x, z)
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
