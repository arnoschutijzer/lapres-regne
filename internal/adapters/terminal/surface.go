package terminal

import "math"

// Quadrants keep a cell's two columns and two rows solid, without gaps between dots.
var quadrants = [16]rune{' ', '▘', '▝', '▀', '▖', '▌', '▞', '▛', '▗', '▚', '▐', '▜', '▄', '▙', '▟', '█'}

func surfacePriority(ink int) int {
	if foremost(ink) {
		return 3
	}
	if prominent(ink) {
		return 2
	}
	return 1
}

// surfaceCell reduces the canvas to four solid quadrants and two terminal colors.
// Thin foreground details take priority over the scenery during downsampling.
func (c *forestCanvas) surfaceCell(col, row int) forestCell {
	var colors [4][3]int
	ink, priority := 0, 0
	for q := range colors {
		x, y := col*2+q%2, row*4+(q/2)*2
		a, b := c.pixels[y*c.width+x], c.pixels[(y+1)*c.width+x]
		if surfacePriority(b.ink) > surfacePriority(a.ink) ||
			surfacePriority(b.ink) == surfacePriority(a.ink) && b.depth < a.depth {
			a, b = b, a
		}
		light := a.light
		if a.ink == b.ink {
			light = (a.light + b.light) / 2
		}
		// A small number of lighting bands keeps animated surfaces calm.
		light = math.Round(min(1, max(0, light))*16) / 16
		colors[q] = mix(background, forestPalette[a.ink], light)
		if p := surfacePriority(a.ink); light > 0 && p > priority {
			ink, priority = a.ink, p
		}
	}
	// Start with the most distinct colors so silhouettes survive color reduction.
	first, second, farthest := 0, 0, 0
	for i := range colors {
		for j := 0; j < i; j++ {
			if distance := colorDistance(colors[i], colors[j]); distance > farthest {
				first, second, farthest = i, j, distance
			}
		}
	}
	var sums [2][3]int
	var counts [2]int
	bits := uint8(0)
	for q, color := range colors {
		group := 0
		if colorDistance(color, colors[first]) < colorDistance(color, colors[second]) {
			group = 1
			bits |= 1 << q
		}
		counts[group]++
		for channel := range color {
			sums[group][channel] += color[channel]
		}
	}
	for group := range sums {
		for channel := range sums[group] {
			sums[group][channel] = (sums[group][channel] + counts[group]/2) / max(1, counts[group])
		}
	}
	return forestCell{bits: bits, ink: uint8(ink), color: sums[1], paper: sums[0]}
}

func colorDistance(a, b [3]int) int {
	dr, dg, db := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return dr*dr + dg*dg + db*db
}
