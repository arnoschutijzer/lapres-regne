package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// World coordinates are sampled directly: exploration never allocates a growing map.
type forest struct {
	seed                   uint64
	x, z, cameraX, cameraZ float64
	steps                  int
	facing                 action
	started, lastWalk      time.Time
}

func newForest(seed uint64) *forest {
	return &forest{seed: seed, facing: down, started: time.Now()}
}

func (f *forest) random(x, z int, salt uint64) float64 {
	n := uint64(int64(x))*0x9e3779b185ebca87 ^ uint64(int64(z))*0xc2b2ae3d27d4eb4f ^ f.seed ^ salt
	n ^= n >> 30
	n *= 0xbf58476d1ce4e5b9
	n ^= n >> 27
	n *= 0x94d049bb133111eb
	n ^= n >> 31
	return float64(n>>11) / (1 << 53)
}

func (f *forest) noise(x, z float64, salt uint64) float64 {
	ix, iz := int(math.Floor(x)), int(math.Floor(z))
	u, v := x-float64(ix), z-float64(iz)
	u, v = u*u*(3-2*u), v*v*(3-2*v)
	a := f.random(ix, iz, salt)*(1-u) + f.random(ix+1, iz, salt)*u
	b := f.random(ix, iz+1, salt)*(1-u) + f.random(ix+1, iz+1, salt)*u
	return a*(1-v) + b*v
}

func (f *forest) walk(direction action) {
	dx, dz := 0.0, 0.0
	switch direction {
	case left:
		dx = -0.55
	case right:
		dx = 0.55
	case up:
		dz = -0.55
	case down:
		dz = 0.55
	default:
		return
	}
	f.x, f.z = f.x+dx, f.z+dz
	f.facing, f.lastWalk = direction, time.Now()
	f.steps++
}

func (f *forest) follow() {
	f.cameraX += (f.x - f.cameraX) * 0.28
	f.cameraZ += (f.z - f.cameraZ) * 0.28
}

type forestPixel struct {
	depth, light float64
	ink          int
}

type forestCanvas struct {
	width, height int
	pixels        []forestPixel
	x, z, scale   float64
}

var forestPalette = [][3]int{
	{59, 78, 58},    // ground
	{102, 131, 71},  // grass
	{76, 100, 62},   // shaded leaves
	{147, 166, 91},  // sunlit leaves
	{143, 118, 80},  // bark
	{69, 130, 144},  // river
	{156, 195, 179}, // ripples
	{198, 187, 120}, // sunlight
	{236, 185, 105}, // character
}

func (c *forestCanvas) project(x, y, z float64) (float64, float64, float64) {
	return float64(c.width)/2 + (x-c.x)*c.scale,
		float64(c.height)*0.57 + ((z-c.z)*0.72-y*0.69)*c.scale,
		-(z-c.z)*0.69 - y*0.72
}

func (c *forestCanvas) plot(x, y int, depth, light float64, ink int) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return
	}
	i := y*c.width + x
	if depth <= c.pixels[i].depth {
		c.pixels[i] = forestPixel{depth, light, ink}
	}
}

func (c *forestCanvas) line(x1, y1, z1, x2, y2, z2, light float64, ink int) {
	ax, ay, ad := c.project(x1, y1, z1)
	bx, by, bd := c.project(x2, y2, z2)
	steps := max(1, int(math.Ceil(math.Max(math.Abs(bx-ax), math.Abs(by-ay)))))
	for i := 0; i <= steps; i++ {
		u := float64(i) / float64(steps)
		c.plot(int(math.Round(ax+(bx-ax)*u)), int(math.Round(ay+(by-ay)*u)), ad+(bd-ad)*u-0.02, light, ink)
	}
}

func (f *forest) render(cols, rows int, seconds float64) *forestCanvas {
	c := &forestCanvas{width: cols * 2, height: rows * 4, x: f.cameraX, z: f.cameraZ, scale: 3.6}
	c.pixels = make([]forestPixel, c.width*c.height)
	for y := range c.height {
		for x := range c.width {
			wx := (float64(x)-float64(c.width)/2)/c.scale + c.x
			wz := (float64(y)-float64(c.height)*0.57)/(c.scale*0.72) + c.z
			n := f.noise(wx*0.65, wz*0.65, 19)
			c.pixels[y*c.width+x] = forestPixel{-(wz - c.z) * 0.69, 0.12 + n*0.26, 0}
		}
	}
	f.drawCharacter(c, seconds)
	return c
}

func (f *forest) drawCharacter(c *forestCanvas, seconds float64) {
	stride := 0.0
	if time.Since(f.lastWalk) < 200*time.Millisecond {
		stride = math.Sin(seconds*18) * 0.24
	}
	// A small warm-colored traveler: boots, coat, arms, and a round hat.
	c.line(f.x-0.16, 0.05, f.z+stride, f.x-0.12, 0.7, f.z, 1, 8)
	c.line(f.x+0.16, 0.05, f.z-stride, f.x+0.12, 0.7, f.z, 1, 8)
	c.line(f.x, 0.55, f.z, f.x, 1.15, f.z, 1, 8)
	c.line(f.x-0.34, 0.65, f.z-stride, f.x+0.34, 1.0, f.z+stride, 0.95, 8)
	for i := range 8 {
		a := float64(i) * math.Pi / 4
		c.line(f.x, 1.48, f.z, f.x+math.Cos(a)*0.28, 1.48, f.z+math.Sin(a)*0.28, 1, 8)
	}
}

func (f *forest) frame(width, height int, seconds float64) string {
	var out strings.Builder
	out.WriteString("\x1b[H\x1b[0m\x1b[48;2;9;15;14m\x1b[38;2;147;166;130m")
	if width < 40 || height < 14 {
		message := "Resize to 40 x 14. Q quits."
		out.WriteString("\x1b[2J" + message[:min(width, len(message))])
		return out.String()
	}
	cols, rows := min(width, 220), min(height-3, 90)
	c := f.render(cols, rows, seconds)
	margin := (width - cols) / 2
	out.WriteString("\x1b[1;1H\x1b[2K  D I T H E R  /  W I L D S")
	for row := range height - 2 {
		fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K", row+2)
		if row >= rows {
			continue
		}
		fmt.Fprintf(&out, "\x1b[%dG", margin+1)
		lastInk := -1
		for col := range cols {
			bits, ink := 0, 0
			var votes [9]int
			mapping := [4][2]int{{0, 3}, {1, 4}, {2, 5}, {6, 7}}
			for dy := range 4 {
				for dx := range 2 {
					x, y := col*2+dx, row*4+dy
					p := c.pixels[y*c.width+x]
					if p.light > (bayer[y%4][x%4]+0.5)/16 {
						bits |= 1 << mapping[dy][dx]
						votes[p.ink]++
					}
				}
			}
			for i := range votes {
				if votes[i] > votes[ink] || i == 8 && votes[i] > 0 {
					ink = i
				}
			}
			if bits == 0 {
				out.WriteByte(' ')
				continue
			}
			if ink != lastInk {
				rgb := forestPalette[ink]
				fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%dm", rgb[0], rgb[1], rgb[2])
				lastInk = ink
			}
			out.WriteRune(rune(0x2800 + bits))
		}
	}
	status := fmt.Sprintf("  x %.1f  z %.1f  |  steps %d", f.x, f.z, f.steps)
	controls := "  arrows / WASD / IJKL walk · Q quit"
	fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K\x1b[38;2;147;166;130m%s", height-1, string([]rune(controls)[:min(width-1, len([]rune(controls)))]))
	fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K%s", height, status[:min(width-1, len(status))])
	return out.String()
}
