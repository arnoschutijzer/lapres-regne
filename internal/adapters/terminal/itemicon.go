package terminal

import (
	"math"
	"strings"
)

// iconCells and iconLines size an item's icon in a slot: 8 cells across and 4 lines down. Each cell
// shows two square pixels, one above the other, so an icon is 8 by 8 pixels of any colour.
const (
	iconCells = 8
	iconLines = 4
)

// iconPixel is what an icon shows at a point: an ink lit from 0 to 1, or nothing.
type iconPixel struct {
	ink    [3]int
	light  float64
	filled bool
}

// iconShader shades an item's icon at a point across it, u rightward and v downward, both from 0
// to 1, for the item's tint.
type iconShader func(u, v float64, tint [3]int) iconPixel

// iconSamples is how many points a side each pixel averages, so edges shade smoothly.
const iconSamples = 3

// iconShades is how many shades of its ink a pixel can take, from shadow to highlight.
const iconShades = 4

// renderIcon draws an icon as lines of upper half blocks, the upper pixel in the foreground and
// the lower in the paper, each line ending on the background paper.
func renderIcon(shade iconShader, tint [3]int, cells, lines int) []string {
	pixels := ditherIcon(shade, tint, cells, lines*2)
	rendered := make([]string, lines)
	for line := range lines {
		var out strings.Builder
		for column := range cells {
			out.WriteString(foreground(pixels[line*2][column]) + paperOf(pixels[line*2+1][column]) + "▀")
		}
		out.WriteString(paperOf(background))
		rendered[line] = out.String()
	}
	return rendered
}

// ditherIcon shades an icon pixel by pixel. Each pixel takes one of a few shades of its ink, and
// ordered dithering mixes neighbouring shades where the light falls between them; pixels the item
// does not cover show the background.
func ditherIcon(shade iconShader, tint [3]int, width, height int) [][][3]int {
	pixels := make([][][3]int, height)
	for y := range height {
		pixels[y] = make([][3]int, width)
		for x := range width {
			pixel := sampleIcon(shade, tint, float64(x), float64(y), float64(width), float64(height))
			if !pixel.filled {
				pixels[y][x] = background
				continue
			}
			threshold := (bayer[y%4][x%4] + 0.5) / 16
			level := min(iconShades-1, max(0, int(pixel.light*(iconShades-1)+threshold)))
			pixels[y][x] = shadeOf(pixel.ink, float64(level)/(iconShades-1))
		}
	}
	return pixels
}

// shadeOf lights an ink from deep shadow at 0, through the ink itself at two thirds, to a pale
// highlight at 1.
func shadeOf(ink [3]int, light float64) [3]int {
	shadow, highlight := mix(background, ink, 0.3), mix(ink, [3]int{255, 250, 235}, 0.3)
	if light < 2.0/3 {
		return mix(shadow, ink, light*1.5)
	}
	return mix(ink, highlight, (light-2.0/3)*3)
}

// sampleIcon averages the shade over a pixel, which counts as covered where the item covers most
// of it.
func sampleIcon(shade iconShader, tint [3]int, x, y, width, height float64) iconPixel {
	var sampled iconPixel
	filled := 0
	for sy := range iconSamples {
		for sx := range iconSamples {
			u := (x + (float64(sx)+0.5)/iconSamples) / width
			v := (y + (float64(sy)+0.5)/iconSamples) / height
			if p := shade(u, v, tint); p.filled {
				filled++
				sampled.light += p.light
				sampled.ink = p.ink
			}
		}
	}
	if filled*2 <= iconSamples*iconSamples {
		return iconPixel{}
	}
	sampled.light /= float64(filled)
	sampled.filled = true
	return sampled
}

// The log lies across its icon from the lower left to its cut end at the upper right, lit from
// the upper left, as Old School RuneScape draws it.
var (
	logFoot   = [2]float64{0.17, 0.80}
	logHead   = [2]float64{0.66, 0.36}
	logRadius = 0.27
)

// logShader shades a log: bark in the log's tint, grooved along its length and lit along its top,
// and a cut end of paler wood ringed with growth rings inside a rim of bark.
func logShader(u, v float64, tint [3]int) iconPixel {
	axisU, axisV := logHead[0]-logFoot[0], logHead[1]-logFoot[1]
	length := math.Hypot(axisU, axisV)
	axisU, axisV = axisU/length, axisV/length
	// up points across the log toward its lit upper side.
	upU, upV := axisV, -axisU
	along := (u-logFoot[0])*axisU + (v-logFoot[1])*axisV
	across := ((u-logFoot[0])*upU + (v-logFoot[1])*upV) / logRadius
	// The cut end faces the viewer at an angle, so it shows as an ellipse narrower than the log.
	endAlong := (along - length) / (logRadius * 0.75)
	if ring := math.Hypot(endAlong, across); ring <= 1 {
		if ring > 0.82 {
			return iconPixel{ink: tint, light: 0.3, filled: true}
		}
		// A growth ring shows as a darker circle in the pale wood, around a dark heart.
		light := 1.0
		if math.Abs(ring-0.5) < 0.1 || ring < 0.16 {
			light = 0.2
		}
		return iconPixel{ink: cutEnd(tint), light: light, filled: true}
	}
	startAlong := along / (logRadius * 0.75)
	if math.Abs(across) > 1 || along > length || along < 0 && math.Hypot(startAlong, across) > 1 {
		return iconPixel{}
	}
	// The bark rounds away from the light, darkening toward its lower edge and its far end, and
	// furrows run along it.
	rounding := math.Sqrt(1 - across*across)
	lit := 0.5 + 0.5*across
	furrow := math.Pow(math.Sin(across*math.Pi*2.5+1.2*math.Sin(along*19)), 2)
	light := (0.15 + 0.7*lit) * (0.5 + 0.5*rounding) * (1 - 0.55*furrow)
	if along < 0 {
		light *= 0.5
	}
	return iconPixel{ink: tint, light: light, filled: true}
}
