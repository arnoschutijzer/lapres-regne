package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
)

type vec struct{ x, y, z float64 }

func (a vec) add(b vec) vec     { return vec{a.x + b.x, a.y + b.y, a.z + b.z} }
func (a vec) sub(b vec) vec     { return vec{a.x - b.x, a.y - b.y, a.z - b.z} }
func (a vec) mul(s float64) vec { return vec{a.x * s, a.y * s, a.z * s} }
func (a vec) dot(b vec) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }
func (a vec) unit() vec         { return a.mul(1 / math.Sqrt(a.dot(a))) }
func (a vec) cross(b vec) vec {
	return vec{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x}
}

type vertex struct {
	point, normal vec
}

type triangle struct {
	vertices [3]vertex
	wood     bool
}

type tree struct{ mesh []triangle }

func newTree() *tree {
	t := &tree{}
	rng := rand.New(rand.NewPCG(12, 34))
	// A fixed seed keeps the same tree as the camera moves or the terminal resizes.
	t.branch(rng, vec{0, 0, 0}, vec{0.05, 1, 0.02}, 1.65, 0.23, 4)
	for i := range 7 {
		a := float64(i) * 2 * math.Pi / 7
		t.limb(vec{0, 0.13, 0}, vec{math.Cos(a) * 0.65, -0.03, math.Sin(a) * 0.65}, 0.14, 0.025)
	}
	return t
}

func (t *tree) branch(rng *rand.Rand, start, direction vec, length, radius float64, depth int) {
	direction = direction.unit()
	end := start.add(direction.mul(length))
	t.limb(start, end, radius, radius*0.58)
	if depth <= 1 {
		// Overlapping, irregular crowns expose both a silhouette and lit surfaces.
		for range 3 {
			center := end.add(vec{(rng.Float64() - 0.5) * 0.8, rng.Float64() * 0.4, (rng.Float64() - 0.5) * 0.8})
			size := 0.42 + rng.Float64()*0.23
			t.crown(rng, center, vec{size, size * 0.83, size})
		}
	}
	if depth == 0 {
		return
	}
	phase := rng.Float64() * 2 * math.Pi
	for i := range 3 {
		a := phase + float64(i)*2*math.Pi/3 + rng.Float64()*0.4
		spread := 0.65 + rng.Float64()*0.5
		next := direction.mul(0.45).add(vec{math.Cos(a) * spread, 0.65 + rng.Float64()*0.25, math.Sin(a) * spread})
		t.branch(rng, end, next, length*(0.64+rng.Float64()*0.1), radius*0.56, depth-1)
	}
}

func (t *tree) limb(start, end vec, r1, r2 float64) {
	direction := end.sub(start).unit()
	reference := vec{0, 1, 0}
	if math.Abs(direction.y) > 0.9 {
		reference = vec{1, 0, 0}
	}
	u := direction.cross(reference).unit()
	v := direction.cross(u).unit()
	const sides = 9
	for i := range sides {
		a, b := float64(i)*2*math.Pi/sides, float64(i+1)*2*math.Pi/sides
		n1 := u.mul(math.Cos(a)).add(v.mul(math.Sin(a)))
		n2 := u.mul(math.Cos(b)).add(v.mul(math.Sin(b)))
		p := vertex{start.add(n1.mul(r1)), n1}
		q := vertex{start.add(n2.mul(r1)), n2}
		r := vertex{end.add(n2.mul(r2)), n2}
		s := vertex{end.add(n1.mul(r2)), n1}
		t.mesh = append(t.mesh, triangle{[3]vertex{p, q, r}, true}, triangle{[3]vertex{p, r, s}, true})
	}
}

func (t *tree) crown(rng *rand.Rand, center, size vec) {
	const rings, sides = 7, 11
	var points [rings + 1][sides]vertex
	for y := range rings + 1 {
		latitude := float64(y) * math.Pi / rings
		for x := range sides {
			longitude := float64(x) * 2 * math.Pi / sides
			normal := vec{math.Sin(latitude) * math.Cos(longitude), math.Cos(latitude), math.Sin(latitude) * math.Sin(longitude)}
			jitter := 0.87 + rng.Float64()*0.26
			offset := vec{normal.x * size.x, normal.y * size.y, normal.z * size.z}.mul(jitter)
			points[y][x] = vertex{center.add(offset), normal}
		}
	}
	for y := range rings {
		for x := range sides {
			next := (x + 1) % sides
			a, b, c, d := points[y][x], points[y][next], points[y+1][next], points[y+1][x]
			t.mesh = append(t.mesh, triangle{vertices: [3]vertex{a, b, c}}, triangle{vertices: [3]vertex{a, c, d}})
		}
	}
}

type screenVertex struct{ x, y, depth, light float64 }

type canvas struct {
	width, height int
	depth, light  []float64
	wood          []bool
}

func newCanvas(width, height int) *canvas {
	c := &canvas{width: width, height: height, depth: make([]float64, width*height), light: make([]float64, width*height), wood: make([]bool, width*height)}
	for i := range c.depth {
		c.depth[i] = math.Inf(1)
	}
	return c
}

func edge(a, b, p screenVertex) float64 {
	return (b.x-a.x)*(p.y-a.y) - (b.y-a.y)*(p.x-a.x)
}

func (c *canvas) draw(points [3]screenVertex, wood bool) {
	a, b, d := points[0], points[1], points[2]
	area := edge(a, b, d)
	if math.Abs(area) < 0.001 {
		return
	}
	left := max(0, int(math.Floor(min(a.x, b.x, d.x))))
	right := min(c.width-1, int(math.Ceil(max(a.x, b.x, d.x))))
	top := max(0, int(math.Floor(min(a.y, b.y, d.y))))
	bottom := min(c.height-1, int(math.Ceil(max(a.y, b.y, d.y))))
	for y := top; y <= bottom; y++ {
		for x := left; x <= right; x++ {
			p := screenVertex{x: float64(x) + 0.5, y: float64(y) + 0.5}
			wa, wb, wd := edge(b, d, p)/area, edge(d, a, p)/area, edge(a, b, p)/area
			if wa < 0 || wb < 0 || wd < 0 {
				continue
			}
			depth := wa*a.depth + wb*b.depth + wd*d.depth
			i := y*c.width + x
			if depth < c.depth[i] {
				c.depth[i], c.light[i], c.wood[i] = depth, wa*a.light+wb*b.light+wd*d.light, wood
			}
		}
	}
}

// Bayer thresholds turn smooth lighting into a stable, ordered dot pattern.
var bayer = [4][4]float64{
	{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5},
}

func (t *tree) render(cols, rows int, angle float64) *canvas {
	c := newCanvas(cols*2, rows*4)
	scale := math.Min(float64(c.width)/7.2, float64(c.height)/6.0)
	sin, cos := math.Sincos(angle + 0.35)
	light := vec{-0.6, 0.75, -0.55}.unit()
	project := func(v vertex, wood bool) screenVertex {
		p := v.point.sub(vec{0, 2.35, 0})
		x, z := p.x*cos+p.z*sin, -p.x*sin+p.z*cos
		// A slightly elevated orthographic camera keeps the whole tree in frame.
		y, depth := p.y*0.976-z*0.218, p.y*0.218+z*0.976
		n := v.normal
		n = vec{n.x*cos + n.z*sin, n.y, -n.x*sin + n.z*cos}
		brightness := 0.20 + 0.68*math.Max(0, n.dot(light))
		if wood {
			brightness = 0.48 + 0.50*math.Max(0, n.dot(light))
		}
		return screenVertex{float64(c.width)/2 + x*scale, float64(c.height)/2 - y*scale, depth, brightness}
	}
	for _, face := range t.mesh {
		c.draw([3]screenVertex{project(face.vertices[0], face.wood), project(face.vertices[1], face.wood), project(face.vertices[2], face.wood)}, face.wood)
	}
	return c
}

func (t *tree) frame(width, height int, position, angle float64) string {
	var out strings.Builder
	out.WriteString("\x1b[H\x1b[0m\x1b[38;2;183;213;191m\x1b[48;2;9;15;14m")
	if width < 40 || height < 14 {
		message := "Resize to 40 x 14. Q quits."
		out.WriteString("\x1b[2J" + message[:min(width, len(message))])
		return out.String()
	}
	// Render caps bound work on unusually large terminals; center the viewport.
	cols, rows := min(width, 220), min(height-4, 90)
	c := t.render(cols, rows, angle)
	margin := (width - cols) / 2
	for y := range height - 1 {
		fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K", y+1)
		if y == 1 {
			out.WriteString("  \x1b[38;2;103;132;116m" + appTitle + " / TREE")
			if width >= 66 {
				fmt.Fprintf(&out, "\x1b[%dG← → / J L   ·   Q quit", width-22)
			}
			out.WriteString("\x1b[38;2;183;213;191m")
		}
		row := y - 3
		if row < 0 || row >= rows {
			continue
		}
		fmt.Fprintf(&out, "\x1b[%dG", margin+1)
		for x := range cols {
			bits, woodDots, dots := 0, 0, 0
			// Unicode braille packs eight independently dithered pixels per cell.
			mapping := [4][2]int{{0, 3}, {1, 4}, {2, 5}, {6, 7}}
			for dy := range 4 {
				for dx := range 2 {
					px, py := x*2+dx, row*4+dy
					i := py*c.width + px
					if c.light[i] > (bayer[py%4][px%4]+0.5)/16 {
						bits |= 1 << mapping[dy][dx]
						dots++
						if c.wood[i] {
							woodDots++
						}
					}
				}
			}
			if bits == 0 {
				out.WriteByte(' ')
			} else if woodDots > dots/2 {
				out.WriteString("\x1b[38;2;193;171;132m")
				out.WriteRune(rune(0x2800 + bits))
				out.WriteString("\x1b[38;2;183;213;191m")
			} else {
				out.WriteRune(rune(0x2800 + bits))
			}
		}
	}
	fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K\x1b[%dG\x1b[38;2;225;239;208m_", height, 1+int(math.Round(position*float64(width-1))))
	return out.String()
}
