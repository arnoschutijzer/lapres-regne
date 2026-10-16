package world

import (
	"math"
	"time"
)

// reach is how far a trunk may stand from the hero to be chopped: about the edge of its crown.
const reach = 2.5

// chopTick is one woodcutting attempt. The axe lands as chopping starts and again on every
// tick, and each tick rolls for a log; a normal tree falls with its first log.
const chopTick = 600 * time.Millisecond

// normalLogXP is the experience a normal tree's log gives.
const normalLogXP = 25.0

// logChance is the chance one tick cuts a normal log, as Old School RuneScape rolls it
// for a bronze axe: 65 in 256 at level 1, rising to 201 in 256 at level 99.
func logChance(level int) float64 {
	const low, high = 64.0, 200.0
	l := float64(level)
	return (1 + math.Floor(low*(maxLevel-l)/(maxLevel-1)+high*(l-1)/(maxLevel-1)+0.5)) / 256
}

// arrival is how close to a trunk an approach ends, comfortably within reach.
const arrival = 1.2

// cell identifies a tree by its place on the tree grid.
type cell struct{ gx, gz int }

type approach struct {
	active bool
	tree   cell
}

type chop struct {
	active  bool
	tree    cell
	elapsed time.Duration
	ticks   int
}

// TreeInReach finds the standing tree the hero acts on: the nearest one in front within reach,
// else the nearest one within reach in any direction, so overshooting a tree still reaches it.
func (w *World) TreeInReach() (Tree, bool) {
	tree, _, ok := w.treeInReach()
	return tree, ok
}

func (w *World) treeInReach() (Tree, cell, bool) {
	x, z := w.hero.Position()
	var nearest Tree
	var nearestCell cell
	found, best := false, math.Inf(1)
	for gz := int(math.Floor((z - reach) / TreeSpacing)); gz <= int(math.Floor((z+reach)/TreeSpacing)); gz++ {
		for gx := int(math.Floor((x - reach) / TreeSpacing)); gx <= int(math.Floor((x+reach)/TreeSpacing)); gx++ {
			tree, exists := w.TreeAt(gx, gz)
			if !exists || tree.Stump {
				continue
			}
			distance := math.Hypot(tree.X-x, tree.Z-z)
			if distance > reach {
				continue
			}
			// Any tree in front outranks every tree beside or behind the hero.
			score := distance
			if FacingToward(tree.X-x, tree.Z-z) != w.hero.Facing() {
				score += reach
			}
			if score < best {
				nearest, nearestCell, found, best = tree, cell{gx, gz}, true, score
			}
		}
	}
	return nearest, nearestCell, found
}

// Chop turns the hero toward the tree in reach and starts chopping it, reporting whether there was one.
func (w *World) Chop() bool {
	tree, _, ok := w.treeInReach()
	if !ok {
		return false
	}
	w.startChop(tree)
	return true
}

func (w *World) startChop(tree Tree) {
	x, z := w.hero.Position()
	w.hero.Face(FacingToward(tree.X-x, tree.Z-z))
	w.approach = approach{}
	w.chop = chop{active: true, tree: tree.cell}
	w.sounds.Chop()
}

// Approach walks the hero straight to a standing tree and chops it on arrival,
// reporting whether the tree still stands.
func (w *World) Approach(tree Tree) bool {
	standing, exists := w.TreeAt(tree.cell.gx, tree.cell.gz)
	if !exists || standing.Stump {
		return false
	}
	w.chop = chop{}
	w.approach = approach{active: true, tree: tree.cell}
	return true
}

// Stop cancels walking to a tree and chopping one.
func (w *World) Stop() {
	w.approach = approach{}
	w.chop = chop{}
}

// approachFor walks toward the approached tree and starts chopping once beside it.
func (w *World) approachFor(dt time.Duration) {
	if !w.approach.active {
		return
	}
	tree, _ := w.TreeAt(w.approach.tree.gx, w.approach.tree.gz)
	x, z := w.hero.Position()
	dx, dz := tree.X-x, tree.Z-z
	remaining := math.Hypot(dx, dz) - arrival
	if remaining > 0 {
		w.hero.advance(dx, dz, math.Min(remaining, w.wading(walkingSpeed*dt.Seconds())))
		remaining = math.Hypot(tree.X-w.hero.x, tree.Z-w.hero.z) - arrival
	}
	if remaining <= 1e-9 {
		w.startChop(tree)
	}
}

func (w *World) Chopping() bool { return w.chop.active }

// AimAt faces the hero toward a world position, except while a chop holds them toward the tree.
func (w *World) AimAt(x, z float64) {
	if w.chop.active {
		return
	}
	w.hero.AimAt(x, z)
}

// LastStrike tells which strike of the chop landed last, counting from zero, and how long ago.
func (w *World) LastStrike() (index int, since time.Duration, ok bool) {
	if !w.chop.active {
		return 0, 0, false
	}
	return int(w.chop.elapsed / chopTick), w.chop.elapsed % chopTick, true
}

// ChoppedTree is the tree being chopped, if any.
func (w *World) ChoppedTree() (Tree, bool) {
	if !w.chop.active {
		return Tree{}, false
	}
	return w.TreeAt(w.chop.tree.gx, w.chop.tree.gz)
}

// chopFor strikes the chopped tree on every tick and rolls for a log; the first log fells it.
func (w *World) chopFor(dt time.Duration) {
	if !w.chop.active {
		return
	}
	w.chop.elapsed += dt
	for w.chop.active && w.chop.elapsed >= time.Duration(w.chop.ticks+1)*chopTick {
		w.chop.ticks++
		w.sounds.Chop()
		if w.rollForLog() {
			w.fell(w.chop.tree)
		}
	}
}

// rollForLog draws this attempt's roll from the world seed and the tree being chopped.
func (w *World) rollForLog() bool {
	const salt = 0x10c5
	roll := w.Random(w.chop.tree.gx, w.chop.tree.gz, salt+uint64(w.hero.attempts))
	w.hero.attempts++
	return roll < logChance(w.hero.woodcutting.Level())
}

// fell turns a tree into a stump and gives the hero its log.
func (w *World) fell(tree cell) {
	if w.felled == nil {
		w.felled = make(map[cell]bool)
	}
	w.felled[tree] = true
	w.hero.logs++
	w.hero.woodcutting.gain(normalLogXP)
	w.chop = chop{}
}
