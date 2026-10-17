package terminal

import (
	"math"

	"lapresregne/internal/world"
)

// walling tells whether stone stands in an edge: a wall, battlements, or a doorway's frame.
func walling(g world.Ground) bool {
	return isWall(g) || g == world.Door
}

// side is what stands along a side of a tile, north, east, south or west.
func (a *builtArea) side(col, row, side int) world.Ground {
	switch side {
	case 0:
		return a.at(col, row).edges.North
	case 1:
		return a.at(col+1, row).edges.West
	case 2:
		return a.at(col, row+1).edges.North
	}
	return a.at(col, row).edges.West
}

// meeting counts the walls and doorways standing in edges that meet at the corner of tiles at
// col, row: the north-west corner of that tile.
func (a *builtArea) meeting(col, row int) int {
	count := 0
	for _, edge := range [...]world.Ground{
		a.at(col, row-1).edges.West, a.at(col, row).edges.West,
		a.at(col-1, row).edges.North, a.at(col, row).edges.North,
		a.at(col-1, row).edges.Rising, a.at(col, row-1).edges.Rising,
		a.at(col, row).edges.Falling, a.at(col-1, row-1).edges.Falling,
	} {
		if walling(edge) {
			count++
		}
	}
	return count
}

// walledDoorway tells whether walls meet a doorway running between two corners of tiles, making it
// a door rather than a gate.
func (a *builtArea) walledDoorway(startCol, startRow, endCol, endRow int) bool {
	return a.meeting(startCol, startRow)+a.meeting(endCol, endRow) > 2
}

// indoorAt tells whether a tile is floored inside.
func (a *builtArea) indoorAt(col, row int) bool {
	ground := a.at(col, row).ground
	return ground == world.Floor || ground == world.Boards
}

// drawEdges draws what stands along the west and north edges of the tile at col, row, whose
// north-west corner is at x, z, and diagonally across it. variety, from 0 to 1, sets windows.
func (a *builtArea) drawEdges(c *forestCanvas, col, row int, x, z, variety float64) {
	edges := a.at(col, row).edges
	size := world.TileSize
	for _, edge := range [2]struct {
		ground             world.Ground
		x0, z0, x1, z1     float64
		startCol, startRow int
		endCol, endRow     int
		// inside and outside are the tiles either side of the edge.
		inside, outside [2]int
		// shut tells whether a doorway's door is shut.
		shut bool
	}{
		{edges.West, x, z, x, z + size, col, row, col, row + 1, [2]int{col, row}, [2]int{col - 1, row}, edges.WestShut},
		{edges.North, x, z, x + size, z, col, row, col + 1, row, [2]int{col, row}, [2]int{col, row - 1}, edges.NorthShut},
	} {
		if edge.ground == world.Grass {
			continue
		}
		c.standOn((edge.x0+edge.x1)/2, (edge.z0+edge.z1)/2)
		startJoined := a.meeting(edge.startCol, edge.startRow) > 1
		endJoined := a.meeting(edge.endCol, edge.endRow) > 1
		switch edge.ground {
		case world.Wall, world.Battlement:
			window := edge.ground == world.Wall && variety < 0.5 &&
				a.indoorAt(edge.inside[0], edge.inside[1]) != a.indoorAt(edge.outside[0], edge.outside[1])
			drawWallAlong(c, edge.x0, edge.z0, edge.x1, edge.z1, startJoined, endJoined, window, edge.ground == world.Battlement)
		case world.Fence:
			drawFenceAlong(c, edge.x0, edge.z0, edge.x1, edge.z1)
		case world.Door:
			// A doorway set in walls has its frame and door; one in a fence is a gate.
			northSouth := edge.x0 == edge.x1
			joined := [4]bool{northSouth, !northSouth, northSouth, !northSouth}
			tileX, tileZ := edge.x0, edge.z0
			if northSouth {
				tileX -= size / 2
			} else {
				tileZ -= size / 2
			}
			if a.walledDoorway(edge.startCol, edge.startRow, edge.endCol, edge.endRow) {
				drawDoor(c, tileX, tileZ, joined, edge.shut)
			} else {
				drawGate(c, tileX, tileZ, joined, edge.shut)
			}
		}
	}
	c.standOn(x+size/2, z+size/2)
	for _, diagonal := range [2]struct {
		ground         world.Ground
		x0, z0, x1, z1 float64
	}{
		{edges.Rising, x, z + size, x + size, z},
		{edges.Falling, x, z, x + size, z + size},
	} {
		switch diagonal.ground {
		case world.Wall, world.Battlement:
			drawDiagonalWall(c, diagonal.x0, diagonal.z0, diagonal.x1, diagonal.z1, diagonal.ground == world.Battlement)
		case world.Fence:
			drawFenceAlong(c, diagonal.x0, diagonal.z0, diagonal.x1, diagonal.z1)
		}
	}
}

// drawWallAlong raises a stone wall along an edge running north to south or east to west, coped
// along its top. It reaches on half its thickness past each end where it joins other walls, so
// their corners meet square. A wall may have a window in its middle, and battlements are crowned
// with merlons.
func drawWallAlong(c *forestCanvas, x0, z0, x1, z1 float64, startJoined, endJoined, window, battlemented bool) {
	half := wallThickness / 2
	from, to := 0.0, 0.0
	if startJoined {
		from = half
	}
	if endJoined {
		to = half
	}
	pattern := surface(stonework)
	if window {
		pattern = windowed(world.TileSize/2 + from)
	}
	top := wallHeight + copingHeight
	if x0 == x1 {
		c.box(x0-half, z0-from, x0+half, z1+to, 0, wallHeight, [4]bool{}, stoneInk, pattern)
		c.box(x0-half-0.08, z0-from-0.08, x0+half+0.08, z1+to+0.08, wallHeight, top, [4]bool{}, stoneInk, plain)
		if battlemented {
			crenellate(z0-from, z1+to, func(a, b float64) {
				c.box(x0-half, a, x0+half, b, top, top+merlonHeight, [4]bool{}, stoneInk, plain)
			})
		}
		return
	}
	c.box(x0-from, z0-half, x1+to, z0+half, 0, wallHeight, [4]bool{}, stoneInk, pattern)
	c.box(x0-from-0.08, z0-half-0.08, x1+to+0.08, z0+half+0.08, wallHeight, top, [4]bool{}, stoneInk, plain)
	if battlemented {
		crenellate(x0-from, x1+to, func(a, b float64) {
			c.box(a, z0-half, b, z0+half, top, top+merlonHeight, [4]bool{}, stoneInk, plain)
		})
	}
}

// crenellate calls merlon for each stretch from one point to another along a wall where a merlon
// stands: alternately half a unit of merlon and half a unit of gap, lined up across the world.
func crenellate(from, to float64, merlon func(a, b float64)) {
	for start := math.Floor(from); start < to; start++ {
		if a, b := max(start, from), min(start+0.5, to); b > a {
			merlon(a, b)
		}
	}
}

// drawDiagonalWall runs a stone wall diagonally from one corner of a tile to the opposite one,
// reaching half its thickness past each to meet the walls there, coped and, for a castle's walls,
// crowned with merlons.
func drawDiagonalWall(c *forestCanvas, x0, z0, x1, z1 float64, battlemented bool) {
	half := wallThickness / 2
	length := math.Hypot(x1-x0, z1-z0)
	dx, dz := (x1-x0)/length*half, (z1-z0)/length*half
	x0, z0, x1, z1 = x0-dx, z0-dz, x1+dx, z1+dz
	top := wallHeight + copingHeight
	c.slab(x0, z0, x1, z1, half, 0, wallHeight, stoneInk, stonework)
	c.slab(x0, z0, x1, z1, half+0.08, wallHeight, top, stoneInk, plain)
	if !battlemented {
		return
	}
	length = math.Hypot(x1-x0, z1-z0)
	crenellate(0, length, func(a, b float64) {
		u0, u1 := a/length, b/length
		c.slab(x0+(x1-x0)*u0, z0+(z1-z0)*u0, x0+(x1-x0)*u1, z0+(z1-z0)*u1, half, top, top+merlonHeight, stoneInk, plain)
	})
}

// drawFenceAlong runs a wooden fence along an edge or diagonally across a tile: a post at each
// end and in the middle, and two rails between.
func drawFenceAlong(c *forestCanvas, x0, z0, x1, z1 float64) {
	const height = 1.1
	post, rail := c.thin(0.09), c.thin(0.05)
	for _, u := range []float64{0, 0.5, 1} {
		postX, postZ := x0+(x1-x0)*u, z0+(z1-z0)*u
		c.box(postX-post, postZ-post, postX+post, postZ+post, 0, height, [4]bool{}, plankInk, plain)
	}
	for _, y := range []float64{0.45, 0.9} {
		c.slab(x0, z0, x1, z1, rail, y-rail, y+rail, plankInk, plain)
	}
}

// edgeOccupied tells whether a wall standing along an edge of a tile, or diagonally across it,
// fills the point alongX, alongZ into the tile at height y.
func (a *builtArea) edgeOccupied(col, row int, alongX, alongZ, y float64) bool {
	half := wallThickness / 2
	size := world.TileSize
	for side, near := range [4]bool{alongZ < half, alongX > size-half, alongZ > size-half, alongX < half} {
		if !near {
			continue
		}
		edge := a.side(col, row, side)
		if !isWall(edge) {
			continue
		}
		top := wallHeight + copingHeight
		if edge == world.Battlement {
			top += merlonHeight
		}
		if y <= top {
			return true
		}
	}
	edges := a.at(col, row).edges
	if isWall(edges.Rising) && fromSegment(alongX, alongZ, 0, size, size, 0) <= half && y <= wallHeight+copingHeight {
		return true
	}
	return isWall(edges.Falling) && fromSegment(alongX, alongZ, 0, 0, size, size) <= half && y <= wallHeight+copingHeight
}
