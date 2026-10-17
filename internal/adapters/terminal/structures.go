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
	shadedStoneInk
	shadowGrassInk
	shadowPathInk
	shadowFloorInk
)

// prominent marks the inks of structures, which count double when a braille cell picks its color,
// so walls, fences and doors keep their color among the ground around them when zoomed out.
func prominent(ink int) bool {
	return ink == stoneInk || ink == shadedStoneInk || ink == ironInk || ink == plankInk || furnishing(ink)
}

// inShadow is the ink a patch of ground takes in a structure's shadow: a cooler, darker color, as a
// braille cell shows only one color and sparser dots alone barely read as shade.
func inShadow(ink int) int {
	switch ink {
	case pathInk:
		return shadowPathInk
	case floorInk:
		return shadowFloorInk
	case boardsInk:
		return shadowPathInk
	}
	return shadowGrassInk
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
// facing along normal. A triangle face keeps only the half from origin to the ends of u and v.
type face struct {
	origin, u, v, normal vec
	triangle             bool
	// seamless leaves the face's edges undarkened, for faces that together make one surface.
	seamless bool
}

// surface patterns a face: the brightness of each point and its ink, or keepInk for the face's own,
// from its distances along the face's u and v in world units and its place in the world.
type surface func(u, v float64, at vec) (brightness float64, ink int)

// keepInk leaves a point of a face in the face's own ink.
const keepInk = -1

// fill draws a face patterned by its surface. Its edges are left dark, so neighbouring faces read
// apart in braille. Faces turned away from the viewer are skipped.
func (c *forestCanvas) fill(f face, ink int, pattern surface) {
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
	// Stone and roofs turned away from the sun take a darker color, giving them their form.
	if sunlight(f.normal) < 0.5 {
		ink = shadedOf(ink)
	}
	// Only faces several pixels across get an outline; a thin rail would vanish inside its own.
	edgeU, edgeV := 0.7/c.scale, 0.7/c.scale
	if lengthU < 4*edgeU {
		edgeU = 0
	}
	if lengthV < 4*edgeV || f.seamless {
		edgeV = 0
	}
	if f.seamless {
		edgeU = 0
	}
	for py := max(0, top); py <= min(c.height-1, bottom); py++ {
		for px := max(0, left); px <= min(c.width-1, right); px++ {
			// The pixel's place on the face, as fractions along u and v.
			dx, dy := float64(px)-ox, float64(py)-oy
			s, t := (dx*vy-dy*vx)/determinant, (ux*dy-uy*dx)/determinant
			if s < 0 || s > 1 || t < 0 || t > 1 || f.triangle && s+t > 1 {
				continue
			}
			alongU, alongV := s*lengthU, t*lengthV
			at := vec{f.origin.x + f.u.x*s + f.v.x*t, f.origin.y + f.u.y*s + f.v.y*t, f.origin.z + f.u.z*s + f.v.z*t}
			brightness, pointInk := pattern(alongU, alongV, at)
			brightness *= light
			if pointInk == keepInk {
				pointInk = ink
			}
			if alongU < edgeU || lengthU-alongU < edgeU || alongV < edgeV || lengthV-alongV < edgeV {
				brightness = 0
			}
			c.plot(px, py, od+s*ud+t*vd-0.01, brightness, pointInk)
		}
	}
}

// box draws a block standing on the ground from x0, z0 to x1, z1, between heights y0 and y1.
// Sides listed in hidden, as north, east, south, west, are left out where they meet another block.
func (c *forestCanvas) box(x0, z0, x1, z1, y0, y1 float64, hidden [4]bool, ink int, pattern surface) {
	width, depth, height := vec{x1 - x0, 0, 0}, vec{0, 0, z1 - z0}, vec{0, y1 - y0, 0}
	c.fill(face{origin: vec{x0, y1, z0}, u: width, v: depth, normal: vec{0, 1, 0}}, ink, pattern)
	sides := [4]face{
		{origin: vec{x0, y0, z0}, u: width, v: height, normal: vec{0, 0, -1}},
		{origin: vec{x1, y0, z0}, u: depth, v: height, normal: vec{1, 0, 0}},
		{origin: vec{x0, y0, z1}, u: width, v: height, normal: vec{0, 0, 1}},
		{origin: vec{x0, y0, z0}, u: depth, v: height, normal: vec{-1, 0, 0}},
	}
	for side, f := range sides {
		if !hidden[side] {
			c.fill(f, ink, pattern)
		}
	}
}

// stonework lays dressed stone in courses, each block lighter or darker than the next, with dark
// mortar between them and moss creeping up from the ground in damp patches.
func stonework(u, v float64, at vec) (float64, int) {
	if at.y < 1 && leafClumps(at.x*1.4, at.y*2.2, at.z*1.4) > 0.55+at.y*0.45 {
		return 0.6, mossInk
	}
	course := math.Floor(v / 0.45)
	along := u + math.Mod(course, 2)*0.45
	block := math.Floor(along / 0.9)
	if math.Mod(v, 0.45) < 0.06 || math.Mod(along, 0.9) < 0.06 {
		return 0.45, keepInk
	}
	return 0.72 + 0.28*wave(block*3.1+course*1.7), keepInk
}

// windowed sets a window in stonework, its middle centre along u: a frame of dressed stone around
// dark glass, from sill to lintel.
func windowed(centre float64) surface {
	const halfWidth, sill, lintel, frame = 0.35, 1.15, 2.05, 0.08
	return func(u, v float64, at vec) (float64, int) {
		across := math.Abs(u - centre)
		if across > halfWidth+frame || v < sill-frame || v > lintel+frame {
			return stonework(u, v, at)
		}
		if across > halfWidth || v < sill || v > lintel {
			return 1, keepInk
		}
		// Glazing bars cross the window.
		if math.Abs(u-centre) < 0.03 || math.Abs(v-(sill+lintel)/2) < 0.03 {
			return 0.8, plankInk
		}
		return 0.45, glassInk
	}
}

// The inks of a wall's weathering and its windows, and of wooden floors.
const (
	mossInk = reedInk + 1 + iota
	glassInk
	boardsInk
)

// plain gives a structure even brightness.
func plain(u, v float64, at vec) (float64, int) { return 1, keepInk }

// neighbours are the tile steps north, east, south and west.
var neighbours = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// builtTile is what stands on one tile of the map, and which of its sides join what stands beside
// it: walls join walls and the doors and gates set in them, and fences join anything built.
type builtTile struct {
	ground world.Ground
	onMap  bool
	joined [4]bool
	// walled marks a tile with a wall beside it, so a door there is set in a wall.
	walled bool
	// beside is what lies on the tiles to the north, east, south and west.
	beside [4]world.Ground
	// fixture is the furniture or landmark standing on the tile, if furnished, and alike marks
	// the sides where the same kind stands beside it, so a long table or a pew reads as one.
	fixture   world.Fixture
	furnished bool
	alike     [4]bool
	// besideFixture is the fixture standing on the tiles to the north, east, south and west, where
	// besideFurnished marks one.
	besideFixture   [4]world.Fixture
	besideFurnished [4]bool
}

// windowed tells whether a wall tile is a straight stretch of a building's wall, with a floor
// on one side, where a window may be set.
func (t builtTile) windowed() bool {
	indoors := func(g world.Ground) bool { return g == world.Floor || g == world.Boards }
	north, east, south, west := t.joined[0], t.joined[1], t.joined[2], t.joined[3]
	if east && west && !north && !south {
		return indoors(t.beside[0]) || indoors(t.beside[2])
	}
	if north && south && !east && !west {
		return indoors(t.beside[1]) || indoors(t.beside[3])
	}
	return false
}

// builtArea surveys the tiles within the drawn area once a frame, for drawing structures and the
// shadows they cast.
type builtArea struct {
	col, row, cols, rows int
	tiles                []builtTile
	// roofs are the roofs drawn this frame: none while the hero is indoors.
	roofs []roof
}

// survey records what stands on every tile from left to right and back to front, with a tile's
// margin around them.
func (a *builtArea) survey(f *world.World, left, right, back, front float64) {
	a.col, a.row = int(math.Floor(left/world.TileSize))-1, int(math.Floor(back/world.TileSize))-1
	a.cols = int(math.Ceil(right/world.TileSize)) + 2 - a.col
	a.rows = int(math.Ceil(front/world.TileSize)) + 2 - a.row
	if cap(a.tiles) < a.cols*a.rows {
		a.tiles = make([]builtTile, a.cols*a.rows)
	}
	a.tiles = a.tiles[:a.cols*a.rows]
	a.roofs = a.roofs[:0]
	if !f.Indoors() {
		for _, building := range f.Buildings() {
			a.roofs = append(a.roofs, roofOver(building))
		}
	}
	for i := range a.tiles {
		col, row := a.col+i%a.cols, a.row+i/a.cols
		middleX, middleZ := (float64(col)+0.5)*world.TileSize, (float64(row)+0.5)*world.TileSize
		ground, onMap := f.GroundAt(middleX, middleZ)
		fixture, furnished := f.FixtureAt(middleX, middleZ)
		a.tiles[i] = builtTile{ground: ground, onMap: onMap, fixture: fixture, furnished: furnished}
	}
	for i := range a.tiles {
		tile := &a.tiles[i]
		col, row := a.col+i%a.cols, a.row+i/a.cols
		for side, step := range neighbours {
			beside := a.at(col+step[0], row+step[1]).ground
			tile.beside[side] = beside
			tile.walled = tile.walled || isWall(beside)
			other := a.at(col+step[0], row+step[1])
			tile.besideFixture[side], tile.besideFurnished[side] = other.fixture, other.furnished
			tile.alike[side] = tile.furnished && other.furnished && other.fixture == tile.fixture
			switch tile.ground {
			case world.Wall, world.Battlement:
				tile.joined[side] = isWall(beside) || beside == world.Door || beside == world.Gate
			default:
				tile.joined[side] = beside.Blocks() && beside != world.Grave || beside == world.Door
			}
		}
	}
}

// at is what stands on a tile; nothing beyond the surveyed area.
func (a *builtArea) at(col, row int) builtTile {
	col, row = col-a.col, row-a.row
	if col < 0 || row < 0 || col >= a.cols || row >= a.rows {
		return builtTile{}
	}
	return a.tiles[row*a.cols+col]
}

// tileAt finds the tile holding a world position, and the position within it.
func tileAt(x, z float64) (col, row int, alongX, alongZ float64) {
	col, row = int(math.Floor(x/world.TileSize)), int(math.Floor(z/world.TileSize))
	return col, row, x - float64(col)*world.TileSize, z - float64(row)*world.TileSize
}

// underfoot is the ground to paint at a world position: beside a wall's slab, the ground of the tile
// on that side, so grass and floors run up to the wall's foot.
func (a *builtArea) underfoot(x, z float64, ground world.Ground) world.Ground {
	if !isWall(ground) {
		return ground
	}
	col, row, alongX, alongZ := tileAt(x, z)
	half, middle := wallThickness/2, world.TileSize/2
	stepX, stepZ := 0, 0
	switch {
	case alongZ < middle-half:
		stepZ = -1
	case alongZ > middle+half:
		stepZ = 1
	}
	switch {
	case alongX < middle-half:
		stepX = -1
	case alongX > middle+half:
		stepX = 1
	}
	if stepX == 0 && stepZ == 0 {
		return ground
	}
	for _, step := range [3][2]int{{stepX, stepZ}, {stepX, 0}, {0, stepZ}} {
		if step == [2]int{} {
			continue
		}
		if beside := a.at(col+step[0], row+step[1]); beside.onMap && !beside.ground.Blocks() {
			return beside.ground
		}
	}
	return world.Floor
}

// occupied tells whether a structure fills the point at height y above x, z.
func (a *builtArea) occupied(x, y, z float64) bool {
	col, row, alongX, alongZ := tileAt(x, z)
	tile := a.at(col, row)
	half, middle := wallThickness/2, world.TileSize/2
	across := func(along float64, width float64) bool { return math.Abs(along-middle) <= width }
	north, east, south, west := tile.joined[0], tile.joined[1], tile.joined[2], tile.joined[3]
	switch tile.ground {
	case world.Wall, world.Battlement:
		top := wallHeight + copingHeight
		if tile.ground == world.Battlement && merlonAt(x, z, east || west) {
			top += merlonHeight
		}
		if y > top {
			return false
		}
		if east || west || !north && !south {
			if from, to := span(west, east); alongX >= from && alongX <= to && across(alongZ, half) {
				return true
			}
		}
		if north || south {
			from, to := span(north, south)
			return alongZ >= from && alongZ <= to && across(alongX, half)
		}
	case world.Door:
		if !tile.walled || y > wallHeight+copingHeight {
			return false
		}
		if east || west {
			jamb := alongX < jambWidth || alongX > world.TileSize-jambWidth
			return across(alongZ, half) && (jamb || y >= doorHeight)
		}
		jamb := alongZ < jambWidth || alongZ > world.TileSize-jambWidth
		return across(alongX, half) && (jamb || y >= doorHeight)
	case world.Fence, world.Gate:
		height, rails := 1.1, []float64{0.45, 0.9}
		if tile.ground == world.Gate {
			height, rails = 1.8, []float64{0.3, 0.9, 1.5}
		}
		if y > height {
			return false
		}
		if across(alongX, 0.09) && across(alongZ, 0.09) {
			return true
		}
		for _, rail := range rails {
			if math.Abs(y-rail) > 0.05 {
				continue
			}
			alongRail := east && alongX >= middle && across(alongZ, 0.05) || west && alongX <= middle && across(alongZ, 0.05) ||
				south && alongZ >= middle && across(alongX, 0.05) || north && alongZ <= middle && across(alongX, 0.05)
			if alongRail {
				return true
			}
		}
	case world.Grave:
		return y <= 1.1 && across(alongX, 0.35) && math.Abs(alongZ-world.TileSize*0.4) <= 0.1
	}
	return false
}

// castShadow is how much light reaches ground in a structure's shadow.
const castShadow = 0.45

// The sun stands in the north-west: a point at height h throws its shadow sunX * h east and
// sunZ * h south, as the trees' shadows fall.
const sunX, sunZ = 0.55, 0.3

// shade darkens the ground at x, z lying in a structure's shadow, and a little at the foot of
// one, where little sky reaches.
func (a *builtArea) shade(x, z float64) float64 {
	for height := 0.1; height <= wallHeight+copingHeight; height += 0.2 {
		if a.occupied(x-sunX*height, height, z-sunZ*height) {
			return castShadow
		}
	}
	for _, roof := range a.roofs {
		ridge := roof.ridge()
		// A roof's shadow falls no nearer than its eaves throw it, and no farther than its ridge.
		if x < roof.x0+sunX*eaves || x > roof.x1+sunX*ridge || z < roof.z0+sunZ*eaves || z > roof.z1+sunZ*ridge {
			continue
		}
		for height := eaves; height <= ridge; height += 0.2 {
			if roof.covers(x-sunX*height, height, z-sunZ*height) {
				return castShadow
			}
		}
	}
	shade := 1.0
	for _, step := range [4][2]float64{{0.3, 0}, {-0.3, 0}, {0, 0.3}, {0, -0.3}} {
		if a.occupied(x+step[0], 0.2, z+step[1]) {
			shade -= 0.15
		}
	}
	return shade
}

// drawStructures draws the walls, doors, fences, gates, graves and fixtures standing on the
// surveyed tiles.
func (r *renderer) drawStructures(c *forestCanvas) {
	a := &r.built
	for i, tile := range a.tiles {
		col, row := a.col+i%a.cols, a.row+i/a.cols
		if !tile.onMap {
			continue
		}
		x, z := float64(col)*world.TileSize, float64(row)*world.TileSize
		// Each structure stands on the ground at the middle of its tile.
		c.standOn(x+world.TileSize/2, z+world.TileSize/2)
		switch tile.ground {
		case world.Wall:
			drawWall(c, x, z, tile.joined, tile.windowed() && r.world.Random(col, row, 43) < 0.5)
		case world.Battlement:
			drawWall(c, x, z, tile.joined, false)
			drawMerlons(c, x, z, tile.joined)
		case world.Door:
			if tile.walled {
				drawDoor(c, x, z, tile.joined)
			} else {
				drawOpenGate(c, x, z, tile.joined)
			}
		case world.Fence:
			drawRailing(c, x, z, tile.joined, 1.1, []float64{0.45, 0.9}, plankInk)
		case world.Gate:
			drawRailing(c, x, z, tile.joined, 1.8, []float64{0.3, 0.9, 1.5}, ironInk)
			drawBars(c, x, z, tile.joined, 1.8)
		case world.Grave:
			drawGrave(c, x, z, r.world.Random(col, row, 41))
		case world.Bridge:
			drawBridgeRails(c, x, z, tile.beside)
		}
		if tile.furnished {
			drawFixture(c, x, z, tile, r.world.Random(col, row, 44))
		}
	}
}

// isWall tells whether stone walling stands on a tile, battlemented or not.
func isWall(g world.Ground) bool {
	return g == world.Wall || g == world.Battlement
}

// merlonHeight is how far a battlement's merlons rise above its coping.
const merlonHeight = 0.5

// merlonAt tells whether a battlement's merlon stands over a point, rather than a gap between
// merlons. Merlons and gaps alternate every half unit along the wall, east to west for a wall
// running that way and north to south otherwise, lined up across tiles.
func merlonAt(x, z float64, eastWest bool) bool {
	along := z
	if eastWest {
		along = x
	}
	return along-math.Floor(along) < 0.5
}

// drawMerlons crowns a wall's coping with merlons, along each stretch drawWall raises.
func drawMerlons(c *forestCanvas, x, z float64, joined [4]bool) {
	north, east, south, west := joined[0], joined[1], joined[2], joined[3]
	half := wallThickness / 2
	middleX, middleZ := x+world.TileSize/2, z+world.TileSize/2
	bottom := wallHeight + copingHeight
	merlons := func(from, to float64, at func(a, b float64)) {
		for start := math.Floor(from); start < to; start++ {
			if a, b := max(start, from), min(start+0.5, to); b > a {
				at(a, b)
			}
		}
	}
	if east || west || !north && !south {
		from, to := span(west, east)
		merlons(x+from, x+to, func(a, b float64) {
			c.box(a, middleZ-half, b, middleZ+half, bottom, bottom+merlonHeight, [4]bool{}, stoneInk, plain)
		})
	}
	if north || south {
		from, to := span(north, south)
		merlons(z+from, z+to, func(a, b float64) {
			c.box(middleX-half, a, middleX+half, b, bottom, bottom+merlonHeight, [4]bool{}, stoneInk, plain)
		})
	}
}

// boards lays a floor of wooden boards running east to west, a quarter unit wide, their ends
// staggered from board to board, each board its own shade.
func boards(x, z, n float64) float64 {
	board := math.Floor(z / 0.25)
	along := x + board*0.7
	if z-board*0.25 < 0.03 || along-math.Floor(along/1.6)*1.6 < 0.04 {
		return 0.08
	}
	return 0.24 + n*0.1 + 0.08*wave(board*2.3)
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
// a jutting course of coping stones, with a window through its middle if it has one; a wall joined
// to nothing stands as a pillar.
func drawWall(c *forestCanvas, x, z float64, joined [4]bool, window bool) {
	north, east, south, west := joined[0], joined[1], joined[2], joined[3]
	half := wallThickness / 2
	middleX, middleZ := x+world.TileSize/2, z+world.TileSize/2
	slab := func(x0, z0, x1, z1 float64, hidden [4]bool) {
		pattern := surface(stonework)
		if window {
			// Along the wall the faces start at the slab's west or north end.
			pattern = windowed(max(middleX-x0, middleZ-z0))
		}
		c.box(x0, z0, x1, z1, 0, wallHeight, hidden, stoneInk, pattern)
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
func planks(u, v float64, at vec) (float64, int) {
	if math.Mod(u, 0.22) < 0.04 {
		return 0.55, keepInk
	}
	return 0.9, keepInk
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

// drawBridgeRails rails a bridge's deck along each side of the tile that drops to the water, with
// a post at either end of the side and one in its middle.
func drawBridgeRails(c *forestCanvas, x, z float64, beside [4]world.Ground) {
	const height = 1.0
	rails := []float64{0.5, 0.95}
	post, rail := c.thin(0.09), c.thin(0.05)
	inset := 0.1
	for side, step := range neighbours {
		if beside[side] != world.Water {
			continue
		}
		// The side runs along the tile's edge facing the water, set a little in from it.
		edgeX, edgeZ := x+world.TileSize/2+float64(step[0])*(world.TileSize/2-inset), z+world.TileSize/2+float64(step[1])*(world.TileSize/2-inset)
		alongX, alongZ := float64(step[1]*step[1]), float64(step[0]*step[0])
		for _, along := range []float64{-world.TileSize / 2, 0, world.TileSize / 2} {
			postX, postZ := edgeX+alongX*along, edgeZ+alongZ*along
			c.box(postX-post, postZ-post, postX+post, postZ+post, 0, height, [4]bool{}, plankInk, plain)
		}
		x0, x1 := edgeX-alongX*world.TileSize/2-rail, edgeX+alongX*world.TileSize/2+rail
		z0, z1 := edgeZ-alongZ*world.TileSize/2-rail, edgeZ+alongZ*world.TileSize/2+rail
		for _, y := range rails {
			c.box(x0, z0, x1, z1, y-rail, y+rail, [4]bool{}, plankInk, plain)
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
