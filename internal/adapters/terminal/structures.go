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
	plankInk
)

// prominent marks the inks of structures, which count double when a braille cell picks its color,
// so walls, fences and doors keep their color among the ground around them when zoomed out.
func prominent(ink int) bool {
	return ink >= stoneInk && ink != pathInk && ink != floorInk
}

// The proportions of walls and doorways, in world units.
const (
	wallHeight    = 3.0
	wallThickness = 0.7
	// copingHeight is the depth of the capping stones along a wall's top, which jut out a little.
	copingHeight = 0.2
	doorHeight   = 2.1
	// jambWidth is how far the stone either side of a doorway reaches into the door's tile.
	jambWidth = 0.4
)

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
	// Structures are lit a little brighter than leaves, so even a shaded face shows in braille.
	light := 0.45 + 0.55*sunlight(f.normal)
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

// drawStructures draws the walls, doors, fences, gates and graves standing on the map's tiles
// within the drawn area.
func (r *renderer) drawStructures(c *forestCanvas, left, right, back, front float64) {
	f := r.world
	groundOf := func(col, row int) world.Ground {
		ground, _ := f.GroundAt((float64(col)+0.5)*world.TileSize, (float64(row)+0.5)*world.TileSize)
		return ground
	}
	for row := int(math.Floor(back / world.TileSize)); row <= int(math.Ceil(front/world.TileSize)); row++ {
		for col := int(math.Floor(left / world.TileSize)); col <= int(math.Ceil(right/world.TileSize)); col++ {
			ground, onMap := f.GroundAt((float64(col)+0.5)*world.TileSize, (float64(row)+0.5)*world.TileSize)
			if !onMap || !ground.Blocks() && ground != world.Door {
				continue
			}
			x, z := float64(col)*world.TileSize, float64(row)*world.TileSize
			// Walls join walls and the doors and gates set in them; fences join anything built.
			var joined [4]bool
			walled := false
			for side, step := range neighbours {
				beside := groundOf(col+step[0], row+step[1])
				walled = walled || beside == world.Wall
				switch ground {
				case world.Wall:
					joined[side] = beside == world.Wall || beside == world.Door || beside == world.Gate
				default:
					joined[side] = beside.Blocks() && beside != world.Grave || beside == world.Door
				}
			}
			switch ground {
			case world.Wall:
				drawWall(c, x, z, joined)
			case world.Door:
				if walled {
					drawDoor(c, x, z, joined)
				} else {
					drawOpenGate(c, x, z, joined)
				}
			case world.Fence:
				drawRailing(c, x, z, joined, 1.1, []float64{0.45, 0.9}, plankInk)
			case world.Gate:
				drawRailing(c, x, z, joined, 1.8, []float64{0.3, 0.9, 1.5}, ironInk)
				drawBars(c, x, z, joined, 1.8)
			case world.Grave:
				drawGrave(c, x, z, f.Random(col, row, 41))
			}
		}
	}
}

// thin keeps a size in world units at least a pixel across, so slender parts never vanish when the
// view is zoomed out.
func (c *forestCanvas) thin(size float64) float64 {
	return max(size, 0.75/c.scale)
}

// span is the stretch of a tile, from its start at 0 to its far edge, that a wall along it covers:
// from the middle out to each joined end.
func span(joinedStart, joinedEnd bool) (from, to float64) {
	from, to = world.TileSize/2-wallThickness/2, world.TileSize/2+wallThickness/2
	if joinedStart {
		from = 0
	}
	if joinedEnd {
		to = world.TileSize
	}
	return from, to
}

// drawWall raises a stone wall through the middle of a tile toward each joined side, capped with
// a jutting course of coping stones; a wall joined to nothing stands as a pillar.
func drawWall(c *forestCanvas, x, z float64, joined [4]bool) {
	north, east, south, west := joined[0], joined[1], joined[2], joined[3]
	half := wallThickness / 2
	middleX, middleZ := x+world.TileSize/2, z+world.TileSize/2
	slab := func(x0, z0, x1, z1 float64, hidden [4]bool) {
		c.box(x0, z0, x1, z1, 0, wallHeight, hidden, stoneInk, stonework)
		c.box(x0-0.08, z0-0.08, x1+0.08, z1+0.08, wallHeight, wallHeight+copingHeight, hidden, stoneInk, plain)
	}
	if east || west || !north && !south {
		from, to := span(west, east)
		slab(x+from, middleZ-half, x+to, middleZ+half, [4]bool{false, east, false, west})
	}
	if north || south {
		from, to := span(north, south)
		slab(middleX-half, z+from, middleX+half, z+to, [4]bool{north, false, south, false})
	}
}

// drawDoor sets an open wooden door in a wall: stone jambs either side of the doorway, a lintel
// carrying the wall over it, and the door swung back against one jamb.
func drawDoor(c *forestCanvas, x, z float64, joined [4]bool) {
	half := wallThickness / 2
	middleX, middleZ := x+world.TileSize/2, z+world.TileSize/2
	size := world.TileSize
	if joined[1] || joined[3] {
		// The wall runs east to west, so the doorway leads north and south.
		c.box(x, middleZ-half, x+jambWidth, middleZ+half, 0, wallHeight, [4]bool{false, false, false, joined[3]}, stoneInk, stonework)
		c.box(x+size-jambWidth, middleZ-half, x+size, middleZ+half, 0, wallHeight, [4]bool{false, joined[1], false, false}, stoneInk, stonework)
		c.box(x+jambWidth, middleZ-half, x+size-jambWidth, middleZ+half, doorHeight, wallHeight, [4]bool{}, stoneInk, stonework)
		c.box(x-0.08, middleZ-half-0.08, x+size+0.08, middleZ+half+0.08, wallHeight, wallHeight+copingHeight, [4]bool{}, stoneInk, plain)
		hinge := x + jambWidth
		c.box(hinge, middleZ-half-0.9, hinge+c.thin(0.1), middleZ-half, 0.05, doorHeight-0.1, [4]bool{}, plankInk, planks)
		return
	}
	c.box(middleX-half, z, middleX+half, z+jambWidth, 0, wallHeight, [4]bool{joined[0], false, false, false}, stoneInk, stonework)
	c.box(middleX-half, z+size-jambWidth, middleX+half, z+size, 0, wallHeight, [4]bool{false, false, joined[2], false}, stoneInk, stonework)
	c.box(middleX-half, z+jambWidth, middleX+half, z+size-jambWidth, doorHeight, wallHeight, [4]bool{}, stoneInk, stonework)
	c.box(middleX-half-0.08, z-0.08, middleX+half+0.08, z+size+0.08, wallHeight, wallHeight+copingHeight, [4]bool{}, stoneInk, plain)
	hinge := z + jambWidth
	c.box(middleX-half-0.9, hinge, middleX-half, hinge+c.thin(0.1), 0.05, doorHeight-0.1, [4]bool{}, plankInk, planks)
}

// drawOpenGate leaves a gap in a fence between two gateposts, with the gate swung open.
func drawOpenGate(c *forestCanvas, x, z float64, joined [4]bool) {
	post := c.thin(0.1)
	middleX, middleZ := x+world.TileSize/2, z+world.TileSize/2
	if joined[0] || joined[2] {
		for _, postZ := range []float64{z + 0.2, z + world.TileSize - 0.2} {
			c.box(middleX-post, postZ-post, middleX+post, postZ+post, 0, 1.4, [4]bool{}, plankInk, plain)
		}
		c.box(middleX-1.2, z+0.2-post/2, middleX, z+0.2+post/2, 0.3, 1.1, [4]bool{}, plankInk, planks)
		return
	}
	for _, postX := range []float64{x + 0.2, x + world.TileSize - 0.2} {
		c.box(postX-post, middleZ-post, postX+post, middleZ+post, 0, 1.4, [4]bool{}, plankInk, plain)
	}
	c.box(x+0.2-post/2, middleZ-1.2, x+0.2+post/2, middleZ, 0.3, 1.1, [4]bool{}, plankInk, planks)
}

// planks runs dark seams down a door or gate every fifth of a unit, along its width.
func planks(u, v float64) float64 {
	if math.Mod(u, 0.22) < 0.04 {
		return 0.55
	}
	return 0.9
}

// drawRailing draws a post in the middle of a tile, with rails at the given heights running from it
// toward each joined side.
func drawRailing(c *forestCanvas, x, z float64, joined [4]bool, height float64, rails []float64, ink int) {
	middleX, middleZ := x+world.TileSize/2, z+world.TileSize/2
	post, rail := c.thin(0.09), c.thin(0.05)
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

// drawBars stands upright iron bars across a closed gate, from its middle toward each joined side.
func drawBars(c *forestCanvas, x, z float64, joined [4]bool, height float64) {
	middleX, middleZ := x+world.TileSize/2, z+world.TileSize/2
	bar := c.thin(0.04)
	for side, step := range neighbours {
		if !joined[side] {
			continue
		}
		for along := 0.3; along < world.TileSize/2; along += 0.3 {
			barX, barZ := middleX+float64(step[0])*along, middleZ+float64(step[1])*along
			c.box(barX-bar, barZ-bar, barX+bar, barZ+bar, 0, height, [4]bool{}, ironInk, plain)
		}
	}
}

// drawGrave raises a headstone, or for some graves a stone cross, near the north of a tile, sized
// by variety from 0 to 1.
func drawGrave(c *forestCanvas, x, z, variety float64) {
	middleX, stoneZ := x+world.TileSize/2, z+world.TileSize*0.4
	height := 0.75 + variety*0.35
	depth := c.thin(0.1)
	if variety < 0.3 {
		shaft := c.thin(0.09)
		c.box(middleX-shaft, stoneZ-shaft, middleX+shaft, stoneZ+shaft, 0, height+0.3, [4]bool{}, stoneInk, plain)
		c.box(middleX-0.35, stoneZ-depth, middleX+0.35, stoneZ+depth, height-0.2, height-0.2+2*c.thin(0.08), [4]bool{}, stoneInk, plain)
		return
	}
	c.box(middleX-0.35, stoneZ-depth, middleX+0.35, stoneZ+depth, 0, height, [4]bool{}, stoneInk, plain)
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
