package world

import (
	"container/heap"
	"math"
)

// point is a place on the ground.
type point struct{ x, z float64 }

// maxRouteSearch is how many tiles a route search explores before settling for the nearest tile
// it found, so a far-off click never stalls a frame.
const maxRouteSearch = 40000

// route finds the way from the hero to a point as waypoints, leading round whatever blocks the way
// on a map and ending on the point. When the point cannot be reached, the route ends on the
// middle of the reachable tile nearest to it. In the endless forest the way is always straight.
func (w *World) route(x, z float64) []point {
	if w.terrain == nil {
		return []point{{x, z}}
	}
	start := tileOf(w.hero.x, w.hero.z)
	goal := tileOf(x, z)
	tiles := w.here().search(start, goal)
	waypoints := make([]point, 0, len(tiles))
	for _, tile := range tiles[1:] {
		waypoints = append(waypoints, tile.middle())
	}
	if tiles[len(tiles)-1] == goal {
		if len(waypoints) == 0 {
			waypoints = append(waypoints, point{x, z})
		} else {
			waypoints[len(waypoints)-1] = point{x, z}
		}
	}
	return w.here().straighten(point{w.hero.x, w.hero.z}, waypoints)
}

func tileOf(x, z float64) cell {
	return cell{int(math.Floor(x / TileSize)), int(math.Floor(z / TileSize))}
}

func (c cell) middle() point {
	return point{(float64(c.gx) + 0.5) * TileSize, (float64(c.gz) + 0.5) * TileSize}
}

// open tells whether the hero may stand on a tile: nothing blocking is built or stands on it, and
// no wall or fence runs diagonally across it.
func (t *terrain) open(c cell) bool {
	ground, ok := t.tile(c.gx, c.gz)
	if t.aloft && ground == Grass {
		return false
	}
	if fixture, stands := t.fixtures[c]; stands && fixture.Blocks() {
		return false
	}
	return ok && !ground.Blocks() && !t.crossed(c)
}

// crossable tells whether the hero may step straight from a tile to the one beside it: no rail,
// wall, fence or shut door stands in the edge between them.
func (t *terrain) crossable(from, to cell) bool {
	return !t.railed(from, to) && !bars(t.between(from, to)) && !t.shut[edgeBetween(from, to)]
}

// railReach is how far in from a bridge's edge over the water its rail keeps the hero.
const railReach = 0.3

// railed tells whether a rail runs between two tiles side by side: along a bridge over the water.
func (t *terrain) railed(a, b cell) bool {
	first, _ := t.tile(a.gx, a.gz)
	second, _ := t.tile(b.gx, b.gz)
	return first == Bridge && second == Water || first == Water && second == Bridge
}

// cornerReach is how far from a tile's corner walls or fences meeting diagonally across it keep
// the hero: as far as the wall running between them reaches into the tile.
const cornerReach = 0.4

// barrier tells whether a wall or fence stands on a tile, joining any beside it.
func (t *terrain) barrier(c cell) bool {
	ground, ok := t.tile(c.gx, c.gz)
	return ok && (ground == Wall || ground == Battlement || ground == Fence || ground == Gate)
}

// cornered tells whether a point of a tile lies in a corner closed off by walls or fences on the
// two tiles beside it that meet there, which join diagonally across the corner.
func (t *terrain) cornered(here cell, alongX, alongZ float64) bool {
	for _, corner := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		dx, dz := alongX-float64(corner[0])*TileSize, alongZ-float64(corner[1])*TileSize
		if math.Hypot(dx, dz) >= cornerReach {
			continue
		}
		stepX, stepZ := corner[0]*2-1, corner[1]*2-1
		if t.barrier(cell{here.gx + stepX, here.gz}) && t.barrier(cell{here.gx, here.gz + stepZ}) {
			return true
		}
	}
	return false
}

// passable tells whether the hero may stand at a point: on an open tile, out of corners closed by
// walls meeting diagonally, clear of walls and fences along the tiles' edges, and not against the
// rail along a bridge's edge over the water.
func (t *terrain) passable(x, z float64) bool {
	here := tileOf(x, z)
	if !t.open(here) {
		return false
	}
	if t.cornered(here, x-float64(here.gx)*TileSize, z-float64(here.gz)*TileSize) || t.barred(x, z) {
		return false
	}
	if ground, _ := t.tile(here.gx, here.gz); ground != Bridge {
		return true
	}
	alongX, alongZ := x-float64(here.gx)*TileSize, z-float64(here.gz)*TileSize
	sides := [4]struct {
		step    cell
		against bool
	}{
		{cell{0, -1}, alongZ < railReach},
		{cell{1, 0}, alongX > TileSize-railReach},
		{cell{0, 1}, alongZ > TileSize-railReach},
		{cell{-1, 0}, alongX < railReach},
	}
	for _, side := range sides {
		if side.against && t.railed(here, cell{here.gx + side.step.gx, here.gz + side.step.gz}) {
			return false
		}
	}
	return true
}

// steppable tells whether the hero may step from a tile to the open tile beside it, straight
// without crossing a rail, wall or fence, or diagonally without cutting the corner of a blocked
// tile or crossing a rail, wall or fence round it.
func (t *terrain) steppable(from, to cell) bool {
	if !t.open(to) {
		return false
	}
	if from.gx == to.gx || from.gz == to.gz {
		return t.crossable(from, to)
	}
	for _, corner := range [2]cell{{to.gx, from.gz}, {from.gx, to.gz}} {
		if !t.open(corner) || !t.crossable(from, corner) || !t.crossable(corner, to) {
			return false
		}
	}
	return true
}

// search finds the shortest way over open tiles from start to goal, stepping straight or
// diagonally without crossing a rail or cutting the corner of a blocked tile. It returns the tiles from start to goal,
// or to the reached tile nearest the goal when the goal cannot be reached.
func (t *terrain) search(start, goal cell) []cell {
	estimate := func(c cell) float64 {
		dx, dz := math.Abs(float64(c.gx-goal.gx)), math.Abs(float64(c.gz-goal.gz))
		return math.Max(dx, dz) + (math.Sqrt2-1)*math.Min(dx, dz)
	}
	cost := map[cell]float64{start: 0}
	from := map[cell]cell{}
	frontier := &tileQueue{{start, estimate(start)}}
	nearest := start
	for frontier.Len() > 0 && len(cost) < maxRouteSearch {
		current := heap.Pop(frontier).(queuedTile).tile
		if estimate(current) < estimate(nearest) {
			nearest = current
		}
		if current == goal {
			break
		}
		for dz := -1; dz <= 1; dz++ {
			for dx := -1; dx <= 1; dx++ {
				next := cell{current.gx + dx, current.gz + dz}
				if next == current || !t.steppable(current, next) {
					continue
				}
				step := 1.0
				if dx != 0 && dz != 0 {
					step = math.Sqrt2
				}
				if known, seen := cost[next]; seen && known <= cost[current]+step {
					continue
				}
				cost[next] = cost[current] + step
				from[next] = current
				heap.Push(frontier, queuedTile{next, cost[next] + estimate(next)})
			}
		}
	}
	tiles := []cell{nearest}
	for tiles[len(tiles)-1] != start {
		tiles = append(tiles, from[tiles[len(tiles)-1]])
	}
	for i, j := 0, len(tiles)-1; i < j; i, j = i+1, j-1 {
		tiles[i], tiles[j] = tiles[j], tiles[i]
	}
	return tiles
}

// straighten drops every waypoint the hero can walk past in a straight line, so a route runs
// smoothly instead of from tile middle to tile middle.
func (t *terrain) straighten(from point, waypoints []point) []point {
	var straight []point
	for len(waypoints) > 0 {
		farthest := 0
		for i := len(waypoints) - 1; i > 0; i-- {
			if t.clear(from, waypoints[i]) {
				farthest = i
				break
			}
		}
		from = waypoints[farthest]
		straight = append(straight, from)
		waypoints = waypoints[farthest+1:]
	}
	return straight
}

// clear tells whether nothing blocks a straight walk between two points.
func (t *terrain) clear(a, b point) bool {
	steps := int(math.Ceil(math.Hypot(b.x-a.x, b.z-a.z) / 0.1))
	for i := 1; i <= steps; i++ {
		u := float64(i) / float64(steps)
		if !t.passable(a.x+(b.x-a.x)*u, a.z+(b.z-a.z)*u) {
			return false
		}
	}
	return true
}

// follow walks up to step toward the waypoints in turn, dropping each one reached. It returns the
// waypoints still ahead, and false when the hero cannot get any closer to the next one.
func (w *World) follow(waypoints []point, step float64) ([]point, bool) {
	for step > 0 && len(waypoints) > 0 {
		next := waypoints[0]
		dx, dz := next.x-w.hero.x, next.z-w.hero.z
		remaining := math.Hypot(dx, dz)
		if remaining > step {
			return waypoints, w.hero.advance(dx, dz, step) > 0
		}
		w.hero.advance(dx, dz, remaining)
		w.hero.x, w.hero.z = next.x, next.z
		waypoints = waypoints[1:]
		step -= remaining
	}
	return waypoints, true
}

type queuedTile struct {
	tile     cell
	priority float64
}

// tileQueue hands out the tile with the lowest priority first.
type tileQueue []queuedTile

func (q tileQueue) Len() int           { return len(q) }
func (q tileQueue) Less(i, j int) bool { return q[i].priority < q[j].priority }
func (q tileQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *tileQueue) Push(x any)        { *q = append(*q, x.(queuedTile)) }
func (q *tileQueue) Pop() any {
	old := *q
	last := old[len(old)-1]
	*q = old[:len(old)-1]
	return last
}
