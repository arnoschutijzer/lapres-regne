package world

import (
	"math"
	"slices"
)

// UpperFloor is a storey of a map above its ground floor: rows of tiles as the map's Ground, Grass
// standing for open air, with the walls, fixtures and models standing on it.
type UpperFloor struct {
	Ground   [][]Ground
	Edges    Edges
	Fixtures []PlacedFixture
	Models   []PlacedModel
}

// StoreyHeight is how much higher each floor lies than the one below, in world units: the height
// of a wall with its coping.
const StoreyHeight = 3.2

// Flight is a staircase or ladder on the floor the hero stands on, by one of its tiles, and the
// floors it leads to.
type Flight struct {
	Col, Row int
	Kind     Fixture
	Up, Down bool
}

// climbs tells whether a fixture leads from one floor to another.
func (f Fixture) climbs() bool { return f == Staircase || f == Ladder }

// Storey is the floor the hero stands on: 0 for the ground floor, 1 for the first floor above it.
func (w *World) Storey() int { return w.storey }

// Storeys is how many floors the world has, its ground floor included.
func (w *World) Storeys() int { return 1 + len(w.floors) }

// floor is the terrain of a floor, 0 for the ground floor; nil in the endless forest.
func (w *World) floor(storey int) *terrain {
	if storey == 0 {
		return w.terrain
	}
	return w.floors[storey-1]
}

// here is the terrain of the floor the hero stands on; nil in the endless forest.
func (w *World) here() *terrain { return w.floor(w.storey) }

// GroundOn is the ground of a floor at a world position, or false beyond the map's edges, where
// a floor above the ground floor is open to the air, and in the endless forest.
func (w *World) GroundOn(storey int, x, z float64) (Ground, bool) {
	if w.terrain == nil {
		return 0, false
	}
	floor := w.floor(storey)
	ground, ok := floor.at(x, z)
	if storey > 0 && (ground == Grass || floor.outsideDiagonal(x, z)) {
		return 0, false
	}
	return ground, ok
}

// outsideDiagonal tells whether a world position lies on the side of a wall or fence running
// diagonally across its tile where both tiles beside that side are open to the air, so a floor
// above the ground floor ends at the wall there.
func (t *terrain) outsideDiagonal(x, z float64) bool {
	here := tileOf(x, z)
	u, v := x/TileSize-float64(here.gx), z/TileSize-float64(here.gz)
	open := func(dx, dz int) bool {
		ground, ok := t.tile(here.gx+dx, here.gz+dz)
		return !ok || ground == Grass
	}
	edges := t.edgesOf(here)
	if bars(edges.Rising) {
		// The wall runs from the south-west corner to the north-east, between the north-west half
		// and the south-east.
		if u+v < 1 && open(-1, 0) && open(0, -1) || u+v > 1 && open(1, 0) && open(0, 1) {
			return true
		}
	}
	if bars(edges.Falling) {
		// The wall runs from the north-west corner to the south-east, between the north-east half
		// and the south-west.
		if u > v && open(1, 0) && open(0, -1) || u < v && open(-1, 0) && open(0, 1) {
			return true
		}
	}
	return false
}

// EdgesOn is what stands along the edges of the tile at a world position on a floor.
func (w *World) EdgesOn(storey int, x, z float64) TileEdges {
	if w.terrain == nil {
		return TileEdges{}
	}
	return w.floor(storey).edgesOf(tileOf(x, z))
}

// FixtureOn is the fixture standing on the tile at a world position on a floor, or false where
// none stands and in the endless forest.
func (w *World) FixtureOn(storey int, x, z float64) (Fixture, bool) {
	if w.terrain == nil {
		return 0, false
	}
	fixture, ok := w.floor(storey).fixtures[tileOf(x, z)]
	return fixture, ok
}

// ModelsOn lists the models standing on a floor; none in the endless forest.
func (w *World) ModelsOn(storey int) []PlacedModel {
	if w.terrain == nil {
		return nil
	}
	return w.floor(storey).models
}

// ElevationOn is how high a floor lies at a world position: the ground floor as Elevation has it,
// and each floor above level over its tile, a storey above the one below.
func (w *World) ElevationOn(storey int, x, z float64) float64 {
	if storey == 0 || w.terrain == nil {
		return w.Elevation(x, z)
	}
	here := tileOf(x, z)
	return w.terrain.tileHeight(here.gx, here.gz) + float64(storey)*StoreyHeight
}

// flightAt is the staircase or ladder standing on a tile of the hero's floor, leading up where one
// of its tiles meets a staircase or ladder on the floor above, and down where one meets one on the
// floor below.
func (w *World) flightAt(c cell) (Flight, bool) {
	kind, stands := w.here().fixtures[c]
	if !stands || !kind.climbs() {
		return Flight{}, false
	}
	flight := Flight{Col: c.gx, Row: c.gz, Kind: kind}
	for _, tile := range w.here().block(c) {
		if w.storey+1 < w.Storeys() {
			above, ok := w.floor(w.storey + 1).fixtures[tile]
			flight.Up = flight.Up || ok && above.climbs()
		}
		if w.storey > 0 {
			below, ok := w.floor(w.storey - 1).fixtures[tile]
			flight.Down = flight.Down || ok && below.climbs()
		}
	}
	return flight, true
}

// block is the tiles of the fixture standing on a tile: every tile beside it, and beside those, on
// which the same kind of fixture stands.
func (t *terrain) block(start cell) []cell {
	kind := t.fixtures[start]
	tiles, seen := []cell{start}, map[cell]bool{start: true}
	for i := 0; i < len(tiles); i++ {
		for _, step := range [4]cell{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
			next := cell{tiles[i].gx + step.gx, tiles[i].gz + step.gz}
			if other, ok := t.fixtures[next]; ok && other == kind && !seen[next] {
				seen[next] = true
				tiles = append(tiles, next)
			}
		}
	}
	return tiles
}

// standings are the tiles the hero may climb a staircase or ladder from, given its tiles: the open
// tiles beside them, along a side no wall, fence or shut door stands in.
func (t *terrain) standings(block []cell) []cell {
	var found []cell
	for _, tile := range block {
		for _, step := range [4]cell{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
			beside := cell{tile.gx + step.gx, tile.gz + step.gz}
			if t.open(beside) && t.crossable(beside, tile) && !slices.Contains(found, beside) {
				found = append(found, beside)
			}
		}
	}
	return found
}

// FlightInReach finds the staircase or ladder nearest the hero on a tile beside theirs, with no
// wall, fence or shut door between them, or false where none is and in the endless forest.
func (w *World) FlightInReach() (Flight, bool) {
	if w.terrain == nil {
		return Flight{}, false
	}
	return w.flightBeside(tileOf(w.hero.x, w.hero.z))
}

// flightBeside finds the staircase or ladder nearest the hero on a tile beside one, with no wall,
// fence or shut door between.
func (w *World) flightBeside(here cell) (Flight, bool) {
	var nearest Flight
	nearestDistance, found := math.Inf(1), false
	for _, step := range [4]cell{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		tile := cell{here.gx + step.gx, here.gz + step.gz}
		flight, ok := w.flightAt(tile)
		if !ok || !w.here().crossable(here, tile) {
			continue
		}
		middle := tile.middle()
		if distance := math.Hypot(middle.x-w.hero.x, middle.z-w.hero.z); distance < nearestDistance {
			nearest, nearestDistance, found = flight, distance, true
		}
	}
	return nearest, found
}

// Climb climbs up or down the staircase or ladder in reach to the floor it leads to, reporting
// whether the hero climbed. The hero lands where they stood, or on the nearest tile they may stand
// on.
func (w *World) Climb(up bool) bool {
	flight, ok := w.FlightInReach()
	if !ok || up && !flight.Up || !up && !flight.Down {
		return false
	}
	w.Stop()
	middle := cell{flight.Col, flight.Row}.middle()
	w.hero.Face(FacingToward(middle.x-w.hero.x, middle.z-w.hero.z))
	if up {
		w.storey++
	} else {
		w.storey--
	}
	w.land()
	if flight.Kind == Ladder {
		w.play(LadderClimb)
	} else {
		w.play(StairsClimb)
	}
	return true
}

// landingReach is how many tiles from where the hero climbed they may land.
const landingReach = 3

// land moves the hero off whatever keeps them from standing where they climbed to, onto the middle
// of the nearest tile they may stand on.
func (w *World) land() {
	if w.passable(w.hero.x, w.hero.z) {
		return
	}
	here := tileOf(w.hero.x, w.hero.z)
	nearest, nearestDistance := point{w.hero.x, w.hero.z}, math.Inf(1)
	for row := here.gz - landingReach; row <= here.gz+landingReach; row++ {
		for col := here.gx - landingReach; col <= here.gx+landingReach; col++ {
			middle := cell{col, row}.middle()
			distance := math.Hypot(middle.x-w.hero.x, middle.z-w.hero.z)
			if distance < nearestDistance && w.passable(middle.x, middle.z) {
				nearest, nearestDistance = middle, distance
			}
		}
	}
	w.hero.x, w.hero.z = nearest.x, nearest.z
}

// ApproachFlight walks the hero round whatever blocks the way to the nearest tile they may climb a
// staircase or ladder from, beside any of its tiles, and climbs it on arrival where it leads one
// way only. Where it leads both up and down, arriving offers the choice, as OfferedClimb reports.
// Where no such tile can be reached, the hero walks as near to it as they can.
func (w *World) ApproachFlight(flight Flight) {
	if w.terrain == nil {
		return
	}
	tile := cell{flight.Col, flight.Row}
	goal := tile
	start, shortest := tileOf(w.hero.x, w.hero.z), math.Inf(1)
	for _, standing := range w.here().standings(w.here().block(tile)) {
		route := w.here().search(start, standing)
		if route[len(route)-1] != standing {
			continue
		}
		if length := routeLength(route); length < shortest {
			goal, shortest = standing, length
		}
	}
	middle := goal.middle()
	w.walkTo(middle.x, middle.z, func() order { return w.arriveAtFlight(tile) })
}

// routeLength is how many tiles a route of tiles side by side covers, diagonal steps counting as
// the longer way they are.
func routeLength(route []cell) float64 {
	length := 0.0
	for i := 1; i < len(route); i++ {
		length += math.Hypot(float64(route[i].gx-route[i-1].gx), float64(route[i].gz-route[i-1].gz))
	}
	return length
}

// arriveAtFlight climbs the staircase or ladder the hero walked to, once they stop beside it.
func (w *World) arriveAtFlight(tile cell) order {
	if !slices.Contains(w.here().standings(w.here().block(tile)), tileOf(w.hero.x, w.hero.z)) {
		return nil
	}
	flight, ok := w.flightAt(tile)
	switch {
	case !ok:
	case flight.Up && flight.Down:
		w.offeredClimb = true
	case flight.Up || flight.Down:
		w.Climb(flight.Up)
	}
	return nil
}

// OfferedClimb reports, once, that the hero arrived beside a staircase or ladder leading both up
// and down, for the player to choose which way to climb.
func (w *World) OfferedClimb() bool {
	offered := w.offeredClimb
	w.offeredClimb = false
	return offered
}
