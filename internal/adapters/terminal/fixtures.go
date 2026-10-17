package terminal

import (
	"math"

	"lapresregne/internal/world"
)

// The inks of furniture and landmarks.
const (
	blueClothInk = shadedSlateInk + 1 + iota
	redClothInk
	whiteClothInk
	goldInk
	emberInk
	flameInk
	waxInk
	burlapInk
)

// furnishing tells whether an ink belongs to furniture, which stands out among the floor around it
// as walls do.
func furnishing(ink int) bool {
	return ink >= blueClothInk && ink <= burlapInk
}

// The proportions of furniture, in world units.
const (
	tableHeight = 0.8
	// tableTop is how thick a table's top is.
	tableTop = 0.12
)

// drawFixture draws the furniture or landmark standing on a tile whose north-west corner is at
// x, z, turned to face into the room. variety, from 0 to 1, varies what it holds.
func drawFixture(c *forestCanvas, x, z float64, tile builtTile, variety float64) {
	back := backOf(tile)
	p := placer{c: c, x: x, z: z, back: back}
	// The sides running on into the same kind, in the fixture's own frame.
	for side := range p.alike {
		p.alike[side] = tile.alike[(side+back)%4]
	}
	switch tile.fixture {
	case world.Table:
		drawTable(c, x, z, tile.alike)
	case world.Chair:
		drawChair(p)
	case world.Throne:
		drawThrone(p)
	case world.Armour:
		drawArmour(p)
	case world.Range:
		drawRange(p)
	case world.Barrel:
		drawBarrel(c, x, z)
	case world.Crate:
		drawCrate(p)
	case world.Sacks:
		drawSacks(p)
	case world.Shelves:
		drawShelves(p, variety)
	case world.Staircase:
		drawStaircase(p)
	case world.Ladder:
		drawLadder(p)
	case world.Torch:
		drawTorch(p)
	case world.Candles:
		drawCandles(p)
	case world.Standard:
		drawStandard(p)
	case world.Altar:
		drawAltar(p)
	case world.Pew:
		drawPew(p)
	case world.Organ:
		drawOrgan(p, variety)
	case world.Coffin:
		drawCoffin(p)
	}
}

// seated tells whether a fixture is sat on, turning its back on the table it stands at.
func seated(f world.Fixture) bool {
	return f == world.Chair || f == world.Throne
}

// backOf is the side, north, east, south or west, a fixture turns its back to: a seat's back is to
// the table's far side; anything else backs onto a wall beside it, but never across a run of its
// own kind, so a row of pews faces along the room. With nothing to back onto it faces south.
func backOf(tile builtTile) int {
	if seated(tile.fixture) {
		for side, beside := range tile.besideFixture {
			if tile.besideFurnished[side] && beside == world.Table {
				return (side + 2) % 4
			}
		}
	}
	alongX, alongZ := tile.alike[1] || tile.alike[3], tile.alike[0] || tile.alike[2]
	for _, side := range [4]int{0, 3, 1, 2} {
		if alongX && side%2 == 1 || alongZ && side%2 == 0 {
			continue
		}
		if isWall(tile.beside[side]) {
			return side
		}
	}
	if alongZ {
		return 3
	}
	return 0
}

// placer draws a fixture's parts in its own frame across a tile, from 0 to TileSize either way with
// its back to the north at 0, turned so its back lies on the tile's side back.
type placer struct {
	c    *forestCanvas
	x, z float64
	back int
	// alike marks the sides of its frame, north, east, south and west, where the same kind
	// stands beside it.
	alike [4]bool
}

// across is the stretch of the frame from west to east a fixture spans, inset from each end
// unless the same kind goes on beyond it.
func (p placer) across(inset float64) (u0, u1 float64) {
	u0, u1 = inset, world.TileSize-inset
	if p.alike[3] {
		u0 = 0
	}
	if p.alike[1] {
		u1 = world.TileSize
	}
	return u0, u1
}

// at turns a point of the fixture's frame into the world.
func (p placer) at(u, v float64) (x, z float64) {
	size := world.TileSize
	switch p.back {
	case 1:
		return p.x + size - v, p.z + u
	case 2:
		return p.x + size - u, p.z + size - v
	case 3:
		return p.x + v, p.z + size - u
	}
	return p.x + u, p.z + v
}

// box draws a block of the fixture from u0, v0 to u1, v1 across its frame, between heights y0
// and y1.
func (p placer) box(u0, v0, u1, v1, y0, y1 float64, ink int, pattern surface) {
	ax, az := p.at(u0, v0)
	bx, bz := p.at(u1, v1)
	p.c.box(min(ax, bx), min(az, bz), max(ax, bx), max(az, bz), y0, y1, [4]bool{}, ink, pattern)
}

// post draws an upright post of a width centred on u, v in the fixture's frame, kept visible
// when zoomed out.
func (p placer) post(u, v, width, y0, y1 float64, ink int) {
	half := p.c.thin(width / 2)
	p.box(u-half, v-half, u+half, v+half, y0, y1, ink, plain)
}

// footprint is the part of a tile a fixture covers, inset from the tile's edges by inset but
// reaching to each edge where the same kind stands beside it.
func footprint(x, z, inset float64, alike [4]bool) (x0, z0, x1, z1 float64) {
	x0, z0, x1, z1 = x+inset, z+inset, x+world.TileSize-inset, z+world.TileSize-inset
	if alike[0] {
		z0 = z
	}
	if alike[1] {
		x1 = x + world.TileSize
	}
	if alike[2] {
		z1 = z + world.TileSize
	}
	if alike[3] {
		x0 = x
	}
	return x0, z0, x1, z1
}

// drawTable stands a wooden table in the middle of a tile, its top running on over the tiles
// beside it where the table goes on, with legs only at its outer corners.
func drawTable(c *forestCanvas, x, z float64, alike [4]bool) {
	x0, z0, x1, z1 := footprint(x, z, 0.3, alike)
	leg := c.thin(0.08)
	corners := [4]struct {
		x, z  float64
		outer bool
	}{
		{x0 + 0.12, z0 + 0.12, !alike[0] && !alike[3]},
		{x1 - 0.12, z0 + 0.12, !alike[0] && !alike[1]},
		{x0 + 0.12, z1 - 0.12, !alike[2] && !alike[3]},
		{x1 - 0.12, z1 - 0.12, !alike[2] && !alike[1]},
	}
	for _, corner := range corners {
		if corner.outer {
			c.box(corner.x-leg, corner.z-leg, corner.x+leg, corner.z+leg, 0, tableHeight-tableTop, [4]bool{}, plankInk, plain)
		}
	}
	c.box(x0, z0, x1, z1, tableHeight-tableTop, tableHeight, alike, plankInk, planks)
}

// drawChair stands a wooden chair with a slatted back.
func drawChair(p placer) {
	for _, leg := range [4][2]float64{{0.68, 0.74}, {1.32, 0.74}, {0.68, 1.3}, {1.32, 1.3}} {
		p.post(leg[0], leg[1], 0.08, 0, 0.42, plankInk)
	}
	p.box(0.62, 0.68, 1.38, 1.36, 0.42, 0.5, plankInk, planks)
	p.box(0.62, 0.68, 1.38, 0.76, 0.5, 1.2, plankInk, planks)
}

// drawThrone raises a gilded throne with a high back, cushioned in Lumbridge blue.
func drawThrone(p placer) {
	p.box(0.42, 0.5, 1.58, 1.4, 0, 0.5, goldInk, plain)
	p.box(0.55, 0.62, 1.45, 1.32, 0.5, 0.6, blueClothInk, plain)
	p.box(0.42, 0.38, 1.58, 0.62, 0.5, 2.0, goldInk, plain)
	p.box(0.58, 0.62, 1.42, 0.66, 0.68, 1.8, blueClothInk, plain)
	p.box(0.42, 0.62, 0.55, 1.4, 0.5, 0.9, goldInk, plain)
	p.box(1.45, 0.62, 1.58, 1.4, 0.5, 0.9, goldInk, plain)
	// A finial crowns each corner of the back.
	p.post(0.48, 0.5, 0.12, 2.0, 2.18, goldInk)
	p.post(1.52, 0.5, 0.12, 2.0, 2.18, goldInk)
}

// drawArmour stands a suit of plate armour on a stone plinth.
func drawArmour(p placer) {
	p.box(0.66, 0.74, 1.34, 1.26, 0, 0.12, stoneInk, plain)
	p.box(0.76, 0.88, 0.96, 1.1, 0.12, 0.88, ironInk, plain)
	p.box(1.04, 0.88, 1.24, 1.1, 0.12, 0.88, ironInk, plain)
	p.box(0.72, 0.84, 1.28, 1.16, 0.88, 1.52, ironInk, plain)
	p.box(0.58, 0.88, 0.72, 1.12, 0.92, 1.48, ironInk, plain)
	p.box(1.28, 0.88, 1.42, 1.12, 0.92, 1.48, ironInk, plain)
	p.box(0.86, 0.86, 1.14, 1.14, 1.54, 1.84, ironInk, plain)
	// A dark slit marks the visor.
	p.box(0.88, 1.14, 1.12, 1.16, 1.68, 1.72, shadedSlateInk, plain)
}

// drawRange builds a stone cooking range against the wall: its iron top, a fire glowing in its
// firebox and a hood rising to carry the smoke away.
func drawRange(p placer) {
	p.box(0.2, 0.2, 1.8, 1.15, 0, 0.85, stoneInk, stonework)
	p.box(0.2, 0.2, 1.8, 1.15, 0.85, 0.95, ironInk, plain)
	p.box(0.55, 1.15, 1.45, 1.18, 0.1, 0.6, emberInk, fire)
	p.box(0.45, 0.2, 1.55, 0.55, 0.95, 2.1, stoneInk, stonework)
}

// fire flickers from embers at its foot to flames above.
func fire(u, v float64, at vec) (float64, int) {
	if v > 0.18+0.1*math.Sin(u*17) {
		return 1.2, flameInk
	}
	return 1, keepInk
}

// drawBarrel stands an iron-hooped barrel in the middle of a tile.
func drawBarrel(c *forestCanvas, x, z float64) {
	middleX, middleZ := x+world.TileSize/2, z+world.TileSize/2
	c.drum(middleX, middleZ, 0.42, 0, 1.0, plankInk, hooped)
}

// hooped binds a barrel's staves with iron hoops near its ends and middle.
func hooped(u, v float64, at vec) (float64, int) {
	for _, hoop := range []float64{0.15, 0.85} {
		if math.Abs(v-hoop) < 0.05 {
			return 0.9, ironInk
		}
	}
	return planks(u, v, at)
}

// drum draws an upright eight-sided drum centred on x, z of a radius, between heights y0 and y1,
// its sides patterned from their foot and its top plain.
func (c *forestCanvas) drum(x, z, radius, y0, y1 float64, ink int, pattern surface) {
	corner := func(i int) vec {
		angle := (float64(i) + 0.5) * math.Pi / 4
		return vec{x + radius*math.Cos(angle), y0, z + radius*math.Sin(angle)}
	}
	for i := range 8 {
		from, to := corner(i), corner(i+1)
		angle := (float64(i) + 1) * math.Pi / 4
		c.fill(face{origin: from, u: vec{to.x - from.x, 0, to.z - from.z}, v: vec{0, y1 - y0, 0}, normal: vec{math.Cos(angle), 0, math.Sin(angle)}}, ink, pattern)
	}
	top := vec{x, y1, z}
	for i := range 8 {
		from, to := corner(i), corner(i+1)
		c.fill(face{origin: top, u: vec{from.x - x, 0, from.z - z}, v: vec{to.x - x, 0, to.z - z}, normal: vec{0, 1, 0}, triangle: true, seamless: true}, ink, plain)
	}
}

// drawCrate stands a wooden crate, braced across its face.
func drawCrate(p placer) {
	p.box(0.5, 0.5, 1.5, 1.5, 0, 0.95, plankInk, braced)
}

// braced runs a crate's planks and frames each face with darker battens.
func braced(u, v float64, at vec) (float64, int) {
	if u < 0.1 || v < 0.1 || u > 0.9 || v > 0.85 {
		return 0.6, keepInk
	}
	return planks(u, v, at)
}

// drawSacks heaps two sacks of grain against each other.
func drawSacks(p placer) {
	p.box(0.42, 0.55, 1.05, 1.25, 0, 0.72, burlapInk, plain)
	p.box(0.5, 0.63, 0.97, 1.17, 0.72, 0.82, burlapInk, plain)
	p.box(0.95, 0.75, 1.58, 1.45, 0, 0.62, burlapInk, plain)
	p.box(1.03, 0.83, 1.5, 1.37, 0.62, 0.7, burlapInk, plain)
}

// drawShelves stands wooden shelves against the wall, laden with jars, pots and bundles whose
// colors vary with variety.
func drawShelves(p placer, variety float64) {
	const width, depth, height = 1.6, 0.45, 2.0
	u0, u1, v0, v1 := 0.2, 0.2+width, 0.12, 0.12+depth
	p.box(u0, v0, u1, v0+0.05, 0, height, plankInk, planks)
	p.box(u0, v0, u0+0.08, v1, 0, height, plankInk, plain)
	p.box(u1-0.08, v0, u1, v1, 0, height, plankInk, plain)
	goods := [...]int{redClothInk, blueClothInk, burlapInk, waxInk, goldInk}
	for shelf := range 4 {
		y := 0.05 + float64(shelf)*0.62
		p.box(u0, v0, u1, v1, y, y+0.06, plankInk, plain)
		if shelf == 3 {
			break
		}
		for item := range 4 {
			pick := int(math.Mod(variety*97+float64(shelf*5+item*3), float64(len(goods))))
			left := u0 + 0.16 + float64(item)*0.36
			p.box(left, v0+0.1, left+0.22, v1-0.06, y+0.06, y+0.3+0.12*float64((item+shelf)%2), goods[pick], plain)
		}
	}
}

// drawStaircase builds stone stairs climbing toward the wall at their back.
func drawStaircase(p placer) {
	const steps = 6
	for step := range steps {
		v1 := world.TileSize - 0.1 - float64(step)*0.3
		p.box(0.3, v1-0.3, 1.7, v1, 0, 0.4*float64(step+1), stoneInk, stonework)
	}
}

// drawLadder leans a wooden ladder against the wall.
func drawLadder(p placer) {
	const height, foot, top = 2.8, 0.9, 0.18
	for _, u := range []float64{0.6, 1.4} {
		for y := 0.0; y < height; y += 0.1 {
			p.post(u, foot-(foot-top)*y/height, 0.08, y, y+0.1, plankInk)
		}
	}
	for y := 0.35; y < height; y += 0.38 {
		v := foot - (foot-top)*y/height
		p.box(0.6, v-0.03, 1.4, v+0.03, y-0.03, y+0.03, plankInk, plain)
	}
}

// drawTorch stands an iron torch on a tall pole, its fire burning in a bowl.
func drawTorch(p placer) {
	p.post(1, 1, 0.1, 0, 1.5, ironInk)
	p.box(0.8, 0.8, 1.2, 1.2, 1.5, 1.62, ironInk, plain)
	p.box(0.86, 0.86, 1.14, 1.14, 1.62, 1.78, emberInk, plain)
	p.box(0.84, 0.84, 1.16, 1.16, 1.76, 2.15, flameInk, plain)
}

// drawCandles stands three lit candles on an iron stand.
func drawCandles(p placer) {
	p.post(1, 1, 0.08, 0, 1.0, ironInk)
	p.box(0.66, 0.9, 1.34, 1.1, 1.0, 1.05, ironInk, plain)
	for _, u := range []float64{0.75, 1.0, 1.25} {
		p.post(u, 1, 0.1, 1.05, 1.36, waxInk)
		p.post(u, 1, 0.06, 1.36, 1.46, flameInk)
	}
}

// drawStandard raises a pole with Lumbridge's blue banner hanging from its crossbar.
func drawStandard(p placer) {
	p.post(1, 0.9, 0.1, 0, 3.4, plankInk)
	p.box(0.45, 0.86, 1.55, 0.94, 3.15, 3.23, plankInk, plain)
	p.box(0.5, 0.94, 1.5, 0.98, 1.9, 3.15, blueClothInk, chevron)
}

// chevron marks a banner with a white chevron pointing down its middle.
func chevron(u, v float64, at vec) (float64, int) {
	if math.Abs(v-0.6-math.Abs(u-0.5)*0.8) < 0.1 {
		return 1, whiteClothInk
	}
	return 1, keepInk
}

// drawAltar raises a stone altar draped in white linen, a golden cross standing on it.
func drawAltar(p placer) {
	u0, u1 := p.across(0.2)
	p.box(u0, 0.55, u1, 1.45, 0, 0.95, stoneInk, stonework)
	p.box(u0, 0.5, u1, 1.5, 0.95, 1.02, whiteClothInk, plain)
	p.box(u0+0.3, 1.5, u1-0.3, 1.52, 0.55, 1.02, whiteClothInk, plain)
	if !p.alike[1] {
		p.post(u1-0.5, 0.75, 0.08, 1.02, 1.6, goldInk)
		p.box(u1-0.68, 0.71, u1-0.32, 0.79, 1.36, 1.44, goldInk, plain)
	}
}

// drawPew lines up a wooden pew, its seat and back running on along the row, closed by carved
// ends.
func drawPew(p placer) {
	u0, u1 := p.across(0.15)
	p.box(u0, 0.75, u1, 1.35, 0.42, 0.5, plankInk, planks)
	p.box(u0, 0.65, u1, 0.75, 0.42, 1.15, plankInk, planks)
	for _, end := range []struct {
		u     float64
		shown bool
	}{{u0, !p.alike[3]}, {u1 - 0.1, !p.alike[1]}} {
		if end.shown {
			p.box(end.u, 0.65, end.u+0.1, 1.35, 0, 1.2, plankInk, plain)
		}
	}
}

// drawOrgan builds a church organ against the wall: a wooden case with its keyboard, and pipes of
// many lengths rising behind.
func drawOrgan(p placer, variety float64) {
	u0, u1 := p.across(0.1)
	p.box(u0, 0.2, u1, 0.95, 0, 1.0, plankInk, planks)
	p.box(u0+0.1, 0.95, u1-0.1, 1.25, 0.78, 0.86, whiteClothInk, keys)
	for u := u0 + 0.12; u < u1-0.1; u += 0.2 {
		rise := 1.2 + 0.9*(0.5+0.5*math.Sin(u*2.1+variety*6))
		p.post(u+0.07, 0.4, 0.13, 1.0, 1.0+rise, ironInk)
	}
}

// keys alternates an organ's white keys with dark gaps.
func keys(u, v float64, at vec) (float64, int) {
	if math.Mod(u, 0.09) < 0.03 {
		return 0.3, plankInk
	}
	return 1, keepInk
}

// drawCoffin lays a closed wooden coffin on the floor.
func drawCoffin(p placer) {
	p.box(0.55, 0.25, 1.45, 1.75, 0, 0.5, plankInk, planks)
	p.box(0.6, 0.3, 1.4, 1.7, 0.5, 0.6, plankInk, plain)
}
