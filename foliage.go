package main

import "math"

type vec struct{ x, y, z float64 }

func (a vec) mul(s float64) vec { return vec{a.x * s, a.y * s, a.z * s} }
func (a vec) dot(b vec) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }
func (a vec) unit() vec         { return a.mul(1 / math.Sqrt(a.dot(a))) }

func (c *forestCanvas) shadow(tree woodlandTree) {
	x, y, _ := c.project(tree.x+tree.height*0.55, 0, tree.z+tree.height*0.3)
	rx, ry := tree.radius*c.scale*1.3, tree.radius*c.scale*0.72
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
func (f *forest) crown(c *forestCanvas, x, y, z, rx, ry, rz float64) {
	sx, sy, depth := c.project(x, y, z)
	boundX := rx * c.scale
	boundY := math.Sqrt(ry*ry*0.69*0.69+rz*rz*0.72*0.72) * c.scale
	heroX, heroY, _ := c.project(f.x, 0.9, f.z)
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
			ink := 2
			if light > 0.58 {
				ink = 3
			}
			// A small cutaway keeps the traveler readable under overlapping crowns.
			hx, hy := (float64(px)-heroX)/7, (float64(py)-heroY)/9
			if distance := hx*hx + hy*hy; distance < 1 {
				if distance < 0.13 || f.random(int(math.Floor(x+u*3)), int(math.Floor(z+localZ*3)), 83) > 0.18 {
					continue
				}
			}
			c.plot(px, py, depth-t, light, ink)
		}
	}
}

func (f *forest) drawTree(c *forestCanvas, tree woodlandTree) {
	x, z, h, r := tree.x, tree.z, tree.height, tree.radius
	for _, offset := range []float64{-tree.trunk, 0, tree.trunk} {
		c.line(x+offset, 0, z, x+offset*0.6, h, z, 0.62, 4)
	}
	a := tree.variation * 6.28
	for i := range 3 {
		angle := a + float64(i)*2.094
		dx, dz := math.Cos(angle)*r*0.62, math.Sin(angle)*r*0.62
		c.line(x, h*0.45, z, x+dx, h*0.92, z+dz, 0.7, 4)
		f.crown(c, x+dx, h*(0.84+float64(i)*0.05), z+dz, r*0.72, r*0.57, r*0.72)
	}
	f.crown(c, x, h+0.3, z, r*0.83, r*0.70, r*0.83)
}

func (f *forest) drawGrass(c *forestCanvas, left, right, back, front, seconds float64) {
	for gz := int(math.Floor(back)); gz <= int(math.Ceil(front)); gz++ {
		for gx := int(math.Floor(left)); gx <= int(math.Ceil(right)); gx++ {
			chance := f.random(gx, gz, 31)
			x := float64(gx) + f.random(gx, gz, 32)
			z := float64(gz) + f.random(gx, gz, 33)
			// Skip offscreen roots before evaluating the more expensive biome fields.
			sx, sy, _ := c.project(x, 0, z)
			if sx < -3 || sx > float64(c.width)+3 || sy < 0 || sy > float64(c.height)+12 {
				continue
			}
			distance, width := f.river(x, z)
			if distance < width {
				continue
			}
			cover := f.cover(x, z)
			tall := cover < 0.38 || distance < width+1.5
			if !tall && chance > 0.10 || tall && chance > 0.78 {
				continue
			}
			height := 0.35 + chance*0.65
			if tall {
				height = 1.2 + chance*1.5
			}
			sway := math.Sin(seconds*1.5+x*0.4+z*0.3) * 0.16
			light := 0.46 + f.random(gx, gz, 34)*0.4
			c.line(x, 0, z, x+sway*0.5, height*0.6, z, light*0.8, 1)
			c.line(x+sway*0.5, height*0.6, z, x+sway+0.16, height, z+0.08, light, 1)
			c.line(x+sway*0.5, height*0.55, z, x-0.22+sway, height*0.85, z-0.04, light*0.8, 1)
		}
	}
}
