package terminal

import (
	"math"

	"lapresregne/internal/world"
)

// The inks of thatched and slated roofs, in sunlight and turned from it.
const (
	thatchInk = axeHandleInk + 1 + iota
	shadedThatchInk
	slateInk
	shadedSlateInk
)

// The inks of clay-tiled roofs, in sunlight and turned from it, after the furniture's.
const (
	clayInk = burlapInk + 1 + iota
	shadedClayInk
)

// The proportions of roofs, in world units.
const (
	// eaves is the height the roof rests at, on top of the walls' coping.
	eaves = wallHeight + copingHeight
	// overhang is how far the roof juts out past the walls.
	overhang = 0.25
	// pitch is how steeply each slope rises, in height per unit across.
	pitch = 0.75
)

// roof is the pitched roof over one building, its ridge along the building's longer side.
type roof struct {
	x0, z0, x1, z1 float64
	// alongX marks a ridge running east to west.
	alongX  bool
	roofing world.Roofing
}

// roofOver shapes the roof of a building, reaching from the outer faces of its walls with an
// overhang.
func roofOver(b world.Building) roof {
	outside := world.TileSize/2 - wallThickness/2 - overhang
	x0, z0 := float64(b.Col)*world.TileSize+outside, float64(b.Row)*world.TileSize+outside
	x1 := float64(b.Col+b.Cols)*world.TileSize - outside
	z1 := float64(b.Row+b.Rows)*world.TileSize - outside
	return roof{x0, z0, x1, z1, x1-x0 >= z1-z0, b.Roof}
}

// ridge is the height of the roof's ridge above the ground.
func (r roof) ridge() float64 {
	if r.alongX {
		return eaves + (r.z1-r.z0)/2*pitch
	}
	return eaves + (r.x1-r.x0)/2*pitch
}

// covers tells whether the roof fills the point at height y above x, z.
func (r roof) covers(x, y, z float64) bool {
	if x < r.x0 || x > r.x1 || z < r.z0 || z > r.z1 || y < eaves {
		return false
	}
	// The roof's surface rises from the eaves toward the ridge, a slope's width in.
	toEaves := min(x-r.x0, r.x1-x)
	if r.alongX {
		toEaves = min(z-r.z0, r.z1-z)
	}
	return y <= eaves+toEaves*pitch
}

// drawRoofs roofs every building, unless the hero is indoors: then all roofs are lifted, so the
// rooms around them show.
func (r *renderer) drawRoofs(c *forestCanvas) {
	for _, shape := range r.built.roofs {
		shape.draw(c)
	}
}

// draw lays the roof's two slopes and its stone gable ends.
func (r roof) draw(c *forestCanvas) {
	// The roof rests on walls standing on the ground at the middle of the building.
	c.standOn((r.x0+r.x1)/2, (r.z0+r.z1)/2)
	ink, pattern := slateInk, surface(slates)
	switch r.roofing {
	case world.Thatch:
		ink, pattern = thatchInk, surface(thatch)
	case world.ClayTiles:
		ink, pattern = clayInk, surface(clayTiles)
	}
	rise := r.ridge() - eaves
	if r.alongX {
		run, length := (r.z1-r.z0)/2, r.x1-r.x0
		middle := (r.z0 + r.z1) / 2
		c.fill(face{origin: vec{r.x0, eaves, r.z0}, u: vec{length, 0, 0}, v: vec{0, rise, run}, normal: vec{0, run, -rise}.unit()}, ink, pattern)
		c.fill(face{origin: vec{r.x0, eaves, r.z1}, u: vec{length, 0, 0}, v: vec{0, rise, -run}, normal: vec{0, run, rise}.unit()}, ink, pattern)
		for _, end := range [2]struct{ x, facing float64 }{{r.x0 + overhang, -1}, {r.x1 - overhang, 1}} {
			c.fill(face{origin: vec{end.x, eaves, r.z0 + overhang}, u: vec{0, 0, r.z1 - r.z0 - 2*overhang}, v: vec{0, rise, middle - r.z0 - overhang}, normal: vec{end.facing, 0, 0}, triangle: true}, stoneInk, stonework)
		}
		return
	}
	run, length := (r.x1-r.x0)/2, r.z1-r.z0
	middle := (r.x0 + r.x1) / 2
	c.fill(face{origin: vec{r.x0, eaves, r.z0}, u: vec{0, 0, length}, v: vec{run, rise, 0}, normal: vec{-rise, run, 0}.unit()}, ink, pattern)
	c.fill(face{origin: vec{r.x1, eaves, r.z0}, u: vec{0, 0, length}, v: vec{-run, rise, 0}, normal: vec{rise, run, 0}.unit()}, ink, pattern)
	for _, end := range [2]struct{ z, facing float64 }{{r.z0 + overhang, -1}, {r.z1 - overhang, 1}} {
		c.fill(face{origin: vec{r.x0 + overhang, eaves, end.z}, u: vec{r.x1 - r.x0 - 2*overhang, 0, 0}, v: vec{middle - r.x0 - overhang, rise, 0}, normal: vec{0, 0, end.facing}, triangle: true}, stoneInk, stonework)
	}
}

// slates lays a roof in overlapping rows of slates up its slope, their joints staggered row by row.
func slates(u, v float64, at vec) (float64, int) {
	row := math.Floor(v / 0.3)
	along := u + math.Mod(row, 2)*0.2
	if v-row*0.3 < 0.05 {
		return 0.35, keepInk
	}
	if along-math.Floor(along/0.4)*0.4 < 0.04 {
		return 0.55, keepInk
	}
	return 0.8 + 0.2*wave(row*1.9+math.Floor(along/0.4)*2.7), keepInk
}

// clayTiles lays a roof in rows of curved clay tiles running down its slope, each row overlapping
// the one below.
func clayTiles(u, v float64, at vec) (float64, int) {
	row := math.Floor(v / 0.35)
	if v-row*0.35 < 0.05 {
		return 0.4, keepInk
	}
	curve := 0.5 + 0.5*math.Cos(u/0.25*math.Pi)
	return 0.6 + 0.3*curve + 0.1*wave(row*2.3), keepInk
}

// thatch streaks a roof with straw running down its slope, bound in bands across it.
func thatch(u, v float64, at vec) (float64, int) {
	if math.Mod(v, 0.7) < 0.06 {
		return 0.5, keepInk
	}
	return 0.65 + 0.35*wave(u*11+wave(v*2.3)*1.5), keepInk
}

// shadedOf is the darker ink a surface takes where it turns from the sun, or the ink itself for
// surfaces that keep one ink.
func shadedOf(ink int) int {
	switch ink {
	case stoneInk:
		return shadedStoneInk
	case thatchInk:
		return shadedThatchInk
	case slateInk:
		return shadedSlateInk
	case clayInk:
		return shadedClayInk
	}
	return ink
}
