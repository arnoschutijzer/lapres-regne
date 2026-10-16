package terminal

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"dither-cli/internal/forest"
)

// Bayer thresholds turn smooth lighting into a stable, ordered dot pattern.
var bayer = [4][4]float64{
	{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5},
}

// renderer draws the forest as braille cells and sends only changed cells.
type renderer struct {
	world                     *forest.Forest
	title                     string
	canvas                    forestCanvas
	trees                     []forest.Tree
	cells                     []forestCell
	screenWidth, screenHeight int
	lastStatus                string
}

func newRenderer(world *forest.Forest, title string) *renderer {
	return &renderer{world: world, title: title}
}

type forestPixel struct {
	depth, light float64
	ink          int
}

type forestCell struct {
	bits, ink uint8
}

const maxForestColumns = 220

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

func (r *renderer) render(cols, rows int, seconds float64) *forestCanvas {
	f := r.world
	c := &r.canvas
	cameraX, cameraZ := f.Camera()
	c.width, c.height, c.x, c.z, c.scale = cols*2, rows*4, cameraX, cameraZ, 3.6
	if len(c.pixels) != c.width*c.height {
		c.pixels = make([]forestPixel, c.width*c.height)
	}
	left := c.x - float64(c.width)/c.scale/2 - 5
	right := c.x + float64(c.width)/c.scale/2 + 5
	back := c.z - float64(c.height)/c.scale - 5
	front := c.z + float64(c.height)/c.scale + 12
	trees := r.trees[:0]
	for gz := int(math.Floor(back / forest.TreeSpacing)); gz <= int(math.Ceil(front/forest.TreeSpacing)); gz++ {
		for gx := int(math.Floor(left / forest.TreeSpacing)); gx <= int(math.Ceil(right/forest.TreeSpacing)); gx++ {
			if tree, exists := f.TreeAt(gx, gz); exists {
				trees = append(trees, tree)
			}
		}
	}
	r.trees = trees
	// Front-to-back drawing lets hidden crown pixels skip lighting calculations.
	sort.Slice(trees, func(i, j int) bool {
		return trees[i].Z*0.69+trees[i].Height*0.72 > trees[j].Z*0.69+trees[j].Height*0.72
	})
	channels := make([]forest.RiverChannel, 0, 6)
	for y := range c.height {
		wz := (float64(y)-float64(c.height)*0.57)/(c.scale*0.72) + c.z
		channels = f.RiverChannels(channels[:0], wz, left, right)
		for x := range c.width {
			wx := (float64(x)-float64(c.width)/2)/c.scale + c.x
			n := f.Noise(wx*0.65, wz*0.65, 19)
			cover := f.Cover(wx, wz)
			ink, light := 0, 0.13+n*0.23
			if cover < 0.38 {
				ink, light = 1, 0.21+n*0.24
			}
			// Diagonal pools of sunlight survive only where the canopy opens.
			beam := math.Max(0, math.Sin(wx*0.28-wz*0.19+f.Noise(wx/9, wz/9, 81)*2)-0.45)
			sun := beam * math.Max(0, 1-cover*1.3)
			light += sun * 0.8
			if sun > 0.24 {
				ink = 7
			}
			distance, width := forest.NearestRiver(channels, wx)
			if distance < width {
				ink, light = 5, 0.28+n*0.3
				if math.Sin(wz*2.6+wx*0.8-seconds*1.8+n*1.3) > 0.91 {
					ink, light = 6, 0.8
				}
			} else if distance < width+0.65 {
				ink, light = 7, 0.32+n*0.25
			}
			c.pixels[y*c.width+x] = forestPixel{-(wz - c.z) * 0.69, light, ink}
		}
	}
	for _, tree := range trees {
		c.shadow(tree)
	}
	for _, tree := range trees {
		r.drawTree(c, tree)
	}
	r.drawCharacter(c, seconds)
	r.drawGrass(c, left, right, back, front, seconds)
	return c
}

func (r *renderer) drawCharacter(c *forestCanvas, seconds float64) {
	x, z := r.world.Position()
	stride := 0.0
	if time.Since(r.world.LastWalk()) < 200*time.Millisecond {
		stride = math.Sin(seconds*18) * 0.24
	}
	strideX, strideZ := 0.0, stride
	if facing := r.world.Facing(); facing == forest.FacingLeft || facing == forest.FacingRight {
		strideX, strideZ = stride, 0
	}
	// A small warm-colored traveler: boots, coat, arms, and a round hat.
	c.line(x-0.16+strideX, 0.05, z+strideZ, x-0.12, 0.7, z, 1, 8)
	c.line(x+0.16-strideX, 0.05, z-strideZ, x+0.12, 0.7, z, 1, 8)
	c.line(x, 0.55, z, x, 1.15, z, 1, 8)
	c.line(x-0.34, 0.65, z-stride, x+0.34, 1.0, z+stride, 0.95, 8)
	for i := range 8 {
		a := float64(i) * math.Pi / 4
		c.line(x, 1.48, z, x+math.Cos(a)*0.28, 1.48, z+math.Sin(a)*0.28, 1, 8)
	}
}

func (r *renderer) frame(width, height int, seconds float64) string {
	var out strings.Builder
	out.WriteString("\x1b[H\x1b[0m\x1b[48;2;9;15;14m\x1b[38;2;147;166;130m")
	if width < 40 || height < 14 {
		r.screenWidth = 0
		message := "Resize to 40 x 14. Q quits."
		out.WriteString("\x1b[2J" + message[:min(width, len(message))])
		return out.String()
	}
	cols, rows := min(width, maxForestColumns), min(height-3, 90)
	c := r.render(cols, rows, seconds)
	margin := (width - cols) / 2
	full := width != r.screenWidth || height != r.screenHeight
	if full {
		r.screenWidth, r.screenHeight = width, height
		if len(r.cells) != cols*rows {
			r.cells = make([]forestCell, cols*rows)
		}
		out.WriteString("\x1b[2J\x1b[1;1H  " + r.title)
		hint := "  arrows/WASD/IJKL · shift run · Q quit"
		if width >= 54 {
			hint += " · N/P tracks"
		}
		controls := []rune(hint)
		fmt.Fprintf(&out, "\x1b[%d;1H%s", height-1, string(controls[:min(width-1, len(controls))]))
	}
	lastInk := -1
	for row := range rows {
		var current [maxForestColumns]forestCell
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
			current[col] = forestCell{uint8(bits), uint8(ink)}
		}
		previous := r.cells[row*cols : (row+1)*cols]
		for col := 0; col < cols; {
			if !full && current[col] == previous[col] {
				col++
				continue
			}
			end := col + 1
			// Bridge tiny gaps when that costs less than another cursor escape.
			for scan := end; scan < cols; scan++ {
				if full || current[scan] != previous[scan] {
					if scan-end > 3 {
						break
					}
					end = scan + 1
				}
			}
			fmt.Fprintf(&out, "\x1b[%d;%dH", row+2, margin+col+1)
			for ; col < end; col++ {
				cell := current[col]
				previous[col] = cell
				if cell.bits == 0 {
					out.WriteByte(' ')
					continue
				}
				if int(cell.ink) != lastInk {
					rgb := forestPalette[cell.ink]
					fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%dm", rgb[0], rgb[1], rgb[2])
					lastInk = int(cell.ink)
				}
				out.WriteRune(rune(0x2800 + int(cell.bits)))
			}
		}
	}
	x, z := r.world.Position()
	status := fmt.Sprintf("  x %.1f  z %.1f  |  steps %d  |  %s  |  seed %d", x, z, r.world.Steps(), r.world.Region(), r.world.Seed())
	if full || status != r.lastStatus {
		fmt.Fprintf(&out, "\x1b[%d;1H\x1b[2K\x1b[38;2;147;166;130m%s", height, status[:min(width-1, len(status))])
		r.lastStatus = status
	}
	return out.String()
}
