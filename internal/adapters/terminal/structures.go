package terminal

import (
	"math"

	"lapresregne/internal/world"
)

// The inks of a map's built ground and structures.
const (
	stoneInk = 29 + iota
	pathInk
	floorInk
	ironInk
)

// wallHeight is how tall walls stand, well above the hero's head.
const wallHeight = 3.0

// face is a flat rectangle of a structure: from its corner origin along the edges u and v, lit as
// facing along normal.
type face struct {
	origin, u, v, normal vec
}

// fill draws a face whose surface pattern gives the brightness of each point, from its distances
// along u and v in world units. Its edges are left dark, so neighbouring faces read apart in
// braille. Faces turned away from the viewer are skipped.
func (c *forestCanvas) fill(f face, ink int, pattern func(u, v float64) float64) {
	if f.normal.dot(c.viewer()) <= 0 {
		return
	}
	ox, oy, od := c.project(f.origin.x, f.origin.y, f.origin.z)
	ux, uy, ud := c.project(f.origin.x+f.u.x, f.origin.y+f.u.y, f.origin.z+f.u.z)
	vx, vy, vd := c.project(f.origin.x+f.v.x, f.origin.y+f.v.y, f.origin.z+f.v.z)
	ux, uy, ud = ux-ox, uy-oy, ud-od
	vx, vy, vd = vx-ox, vy-oy, vd-od
	determinant := ux*vy - uy*vx
	if math.Abs(determinant) < 1e-9 {
		return
	}
	left := int(math.Floor(min(ox, ox+ux, ox+vx, ox+ux+vx)))
	right := int(math.Ceil(max(ox, ox+ux, ox+vx, ox+ux+vx)))
	top := int(math.Floor(min(oy, oy+uy, oy+vy, oy+uy+vy)))
	bottom := int(math.Ceil(max(oy, oy+uy, oy+vy, oy+uy+vy)))
	lengthU, lengthV := math.Sqrt(f.u.dot(f.u)), math.Sqrt(f.v.dot(f.v))
	light := sunlight(f.normal)
	// Only faces several pixels across get an outline; a thin rail would vanish inside its own.
	edgeU, edgeV := 0.7/c.scale, 0.7/c.scale
	if lengthU < 4*edgeU {
		edgeU = 0
	}
	if lengthV < 4*edgeV {
		edgeV = 0
	}
	for py := max(0, top); py <= min(c.height-1, bottom); py++ {
		for px := max(0, left); px <= min(c.width-1, right); px++ {
			// The pixel's place on the face, as fractions along u and v.
			dx, dy := float64(px)-ox, float64(py)-oy
			s, t := (dx*vy-dy*vx)/determinant, (ux*dy-uy*dx)/determinant
			if s < 0 || s > 1 || t < 0 || t > 1 {
				continue
			}
			alongU, alongV := s*lengthU, t*lengthV
			brightness := light * pattern(alongU, alongV)
			if alongU < edgeU || lengthU-alongU < edgeU || alongV < edgeV || lengthV-alongV < edgeV {
				brightness = 0
			}
			c.plot(px, py, od+s*ud+t*vd-0.01, brightness, ink)
		}
	}
}

// box draws a block standing on the ground from x0, z0 to x1, z1, between heights y0 and y1.
// Sides listed in hidden, as north, east, south, west, are left out where they meet another block.
func (c *forestCanvas) box(x0, z0, x1, z1, y0, y1 float64, hidden [4]bool, ink int, pattern func(u, v float64) float64) {
	width, depth, height := vec{x1 - x0, 0, 0}, vec{0, 0, z1 - z0}, vec{0, y1 - y0, 0}
	c.fill(face{vec{x0, y1, z0}, width, depth, vec{0, 1, 0}}, ink, pattern)
	sides := [4]face{
		{vec{x0, y0, z0}, width, height, vec{0, 0, -1}},
		{vec{x1, y0, z0}, depth, height, vec{1, 0, 0}},
		{vec{x0, y0, z1}, width, height, vec{0, 0, 1}},
		{vec{x0, y0, z0}, depth, height, vec{-1, 0, 0}},
	}
	for side, f := range sides {
		if !hidden[side] {
			c.fill(f, ink, pattern)
		}
	}
}

// stonework lays dressed stone in courses, each block a little lighter or darker than the next,
// with dark mortar between them.
func stonework(u, v float64) float64 {
	course := math.Floor(v / 0.45)
	along := u + math.Mod(course, 2)*0.45
	block := math.Floor(along / 0.9)
	if math.Mod(v, 0.45) < 0.06 || math.Mod(along, 0.9) < 0.06 {
		return 0.45
	}
	return 0.85 + 0.15*wave(block*3.1+course*1.7)
}

// plain gives a structure even brightness.
func plain(u, v float64) float64 { return 1 }

// neighbours are the tile steps north, east, south and west.
var neighbours = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// drawStructures draws the walls, fences, gates and graves standing on the map's tiles within the
// drawn area.
func (r *renderer) drawStructures(c *forestCanvas, left, right, back, front float64) {
	f := r.world
	groundOf := func(col, row int) world.Ground {
		ground, _ := f.GroundAt((float64(col)+0.5)*world.TileSize, (float64(row)+0.5)*world.TileSize)
		return ground
	}
	for row := int(math.Floor(back / world.TileSize)); row <= int(math.Ceil(front/world.TileSize)); row++ {
		for col := int(math.Floor(left / world.TileSize)); col <= int(math.Ceil(right/world.TileSize)); col++ {
			ground, onMap := f.GroundAt((float64(col)+0.5)*world.TileSize, (float64(row)+0.5)*world.TileSize)
			if !onMap || !ground.Blocks() {
				continue
			}
			x, z := float64(col)*world.TileSize, float64(row)*world.TileSize
			var joined [4]bool
			for side, step := range neighbours {
				beside := groundOf(col+step[0], row+step[1])
				switch ground {
				case world.Wall:
					joined[side] = beside == world.Wall
				default:
					joined[side] = beside == world.Fence || beside == world.Gate || beside == world.Wall
				}
			}
			switch ground {
			case world.Wall:
				c.box(x, z, x+world.TileSize, z+world.TileSize, 0, wallHeight, joined, stoneInk, stonework)
			case world.Fence:
				drawRailing(c, x, z, joined, 1.1, []float64{0.45, 0.9}, 4)
			case world.Gate:
				drawRailing(c, x, z, joined, 1.8, []float64{0.3, 0.9, 1.5}, ironInk)
			case world.Grave:
				drawGrave(c, x, z, f.Random(col, row, 41))
			}
		}
	}
}

// drawRailing draws a post in the middle of a tile, with rails at the given heights running from it
// toward each joined side.
func drawRailing(c *forestCanvas, x, z float64, joined [4]bool, height float64, rails []float64, ink int) {
	middleX, middleZ := x+world.TileSize/2, z+world.TileSize/2
	const post, rail = 0.09, 0.05
	c.box(middleX-post, middleZ-post, middleX+post, middleZ+post, 0, height, [4]bool{}, ink, plain)
	for side, step := range neighbours {
		if !joined[side] {
			continue
		}
		// The rail runs from the post to the tile's edge on that side.
		endX, endZ := middleX+float64(step[0])*world.TileSize/2, middleZ+float64(step[1])*world.TileSize/2
		x0, x1 := min(middleX, endX)-rail, max(middleX, endX)+rail
		z0, z1 := min(middleZ, endZ)-rail, max(middleZ, endZ)+rail
		for _, y := range rails {
			c.box(x0, z0, x1, z1, y-rail, y+rail, [4]bool{}, ink, plain)
		}
	}
}

// drawGrave raises a headstone, or for some graves a stone cross, near the north of a tile, sized
// by variety from 0 to 1.
func drawGrave(c *forestCanvas, x, z, variety float64) {
	middleX, stoneZ := x+world.TileSize/2, z+world.TileSize*0.4
	height := 0.75 + variety*0.35
	if variety < 0.3 {
		c.box(middleX-0.09, stoneZ-0.09, middleX+0.09, stoneZ+0.09, 0, height+0.3, [4]bool{}, stoneInk, plain)
		c.box(middleX-0.35, stoneZ-0.08, middleX+0.35, stoneZ+0.08, height-0.2, height-0.04, [4]bool{}, stoneInk, plain)
		return
	}
	c.box(middleX-0.35, stoneZ-0.1, middleX+0.35, stoneZ+0.1, 0, height, [4]bool{}, stoneInk, plain)
}

// flagstones lays square slabs a world unit across, offset row by row, with dark joints between
// them; n varies the stone's brightness.
func flagstones(x, z, n float64) float64 {
	along := x + math.Mod(math.Floor(z), 2)*0.5
	if z-math.Floor(z) < 0.07 || along-math.Floor(along) < 0.07 {
		return 0.1
	}
	return 0.3 + n*0.14
}
