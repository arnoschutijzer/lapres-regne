package terminal

import "math"

// Quadrants keep a cell's two columns and two rows solid, without gaps between dots.
var quadrants = [16]rune{' ', '▘', '▝', '▀', '▖', '▌', '▞', '▛', '▗', '▚', '▐', '▜', '▄', '▙', '▟', '█'}

func figureInk(ink int) bool {
	return ink == 9 || ink == axeHandleInk || ink >= defaultLook.skin && ink <= eyeInk
}

// figureCell gives the hero's existing silhouette solid color, leaving the world dithered.
// Two colors per cell keep adjacent materials from being flattened into one tint.
func (c *forestCanvas) figureCell(col, row int) forestCell {
	var colors [4][3]int
	var materials [4]int
	occupied := uint8(0)
	ink := 0
	for q := range colors {
		x, y := col*2+q%2, row*4+(q/2)*2
		a, b := c.pixels[y*c.width+x], c.pixels[(y+1)*c.width+x]
		colors[q] = background
		if !figureInk(a.ink) || figureInk(b.ink) && b.depth < a.depth {
			a, b = b, a
		}
		if !figureInk(a.ink) {
			continue
		}
		light := a.light
		if a.ink == b.ink {
			light = (a.light + b.light) / 2
		}
		// A small number of lighting bands keeps animated surfaces calm.
		light = math.Round(min(1, max(0, light))*16) / 16
		colors[q] = mix(background, forestPalette[a.ink], light)
		if light > 0 {
			ink = a.ink
			materials[q] = a.ink
			occupied |= 1 << q
		}
	}
	// At the silhouette, reserve one color for the background. Prefer clothing and metal
	// over skin so a small item keeps its tint instead of being averaged into the outline.
	if occupied != 15 {
		best := -1
		color := background
		for q, material := range materials {
			if occupied&(1<<q) == 0 {
				continue
			}
			priority := 0
			switch material {
			case defaultLook.undershirt:
				priority = 1
			case defaultLook.shorts, axeHandleInk:
				priority = 2
			case 9:
				priority = 3
			}
			if priority > best {
				best, color, ink = priority, colors[q], material
			}
		}
		return forestCell{bits: occupied, ink: uint8(ink), solid: true, color: color, paper: background}
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
	return forestCell{bits: bits, ink: uint8(ink), solid: true, color: sums[1], paper: sums[0]}
}

func colorDistance(a, b [3]int) int {
	dr, dg, db := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return dr*dr + dg*dg + db*db
}
