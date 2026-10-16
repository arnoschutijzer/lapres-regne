package world

import (
	"fmt"
	"math"
	"time"
)

// reach is how far a trunk may stand from the hero to be chopped: about the edge of its crown.
const reach = 2.5

// gameTick is Old School RuneScape's game tick.
const gameTick = 600 * time.Millisecond

// rollInterval is how often chopping rolls for a log: every 4 game ticks.
const rollInterval = 4 * gameTick

// StrikeInterval is the swing of the axe: it lands as chopping starts and again at this pace,
// independently of the rolls for logs.
const StrikeInterval = 800 * time.Millisecond

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
	kind    TreeKind
	elapsed time.Duration
	strikes int
	rolls   int
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

// Chop turns the hero toward the tree in reach and starts chopping it, reporting whether
// chopping started. A tree above the hero's level leaves a message instead.
func (w *World) Chop() bool {
	tree, _, ok := w.treeInReach()
	if !ok {
		return false
	}
	return w.startChop(tree)
}

func (w *World) startChop(tree Tree) bool {
	x, z := w.hero.Position()
	w.hero.Face(FacingToward(tree.X-x, tree.Z-z))
	w.approach = approach{}
	w.chop = chop{}
	if required := species[tree.Kind].level; w.hero.woodcutting.Level() < required {
		w.message = fmt.Sprintf("You need woodcutting level %d to chop this %s.", required, tree.Kind)
		return false
	}
	w.message = ""
	w.chop = chop{active: true, tree: tree.cell, kind: tree.Kind}
	w.sounds.Chop()
	return true
}

// Message is the game's reply to the hero's latest action, or empty when there is none.
func (w *World) Message() string { return w.message }

// Approach walks the hero straight to a standing tree and chops it on arrival,
// reporting whether the tree still stands.
func (w *World) Approach(tree Tree) bool {
	standing, exists := w.TreeAt(tree.cell.gx, tree.cell.gz)
	if !exists || standing.Stump {
		return false
	}
	w.chop = chop{}
	w.message = ""
	w.approach = approach{active: true, tree: tree.cell}
	return true
}

// Stop cancels walking to a tree and chopping one, and clears the last message.
func (w *World) Stop() {
	w.approach = approach{}
	w.chop = chop{}
	w.message = ""
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
	return int(w.chop.elapsed / StrikeInterval), w.chop.elapsed % StrikeInterval, true
}

// ChoppedTree is the tree being chopped, if any.
func (w *World) ChoppedTree() (Tree, bool) {
	if !w.chop.active {
		return Tree{}, false
	}
	return w.TreeAt(w.chop.tree.gx, w.chop.tree.gz)
}

// chopFor swings the axe at its own pace and rolls for a log every four ticks. Trees that give
// several logs wear down their depletion timer while chopped.
func (w *World) chopFor(dt time.Duration) {
	if !w.chop.active {
		return
	}
	w.chop.elapsed += dt
	if !species[w.chop.kind].fallsWithFirstLog {
		if w.depletion == nil {
			w.depletion = make(map[cell]time.Duration)
		}
		w.depletion[w.chop.tree] += dt
	}
	for w.chop.elapsed >= time.Duration(w.chop.strikes+1)*StrikeInterval {
		w.chop.strikes++
		w.sounds.Chop()
	}
	for w.chop.active && w.chop.elapsed >= time.Duration(w.chop.rolls+1)*rollInterval {
		w.chop.rolls++
		if w.roll(0x10c5) < species[w.chop.kind].chance(w.hero.woodcutting.Level()) {
			w.cutLog()
		}
	}
}

// recoverTimers lets every tree nobody is chopping regain its depletion timer as fast as it wore down.
func (w *World) recoverTimers(dt time.Duration) {
	for tree, worn := range w.depletion {
		if w.chop.active && tree == w.chop.tree {
			continue
		}
		if worn <= dt {
			delete(w.depletion, tree)
		} else {
			w.depletion[tree] = worn - dt
		}
	}
}

// roll draws this attempt's random number from the world seed, the tree being chopped and a salt.
func (w *World) roll(salt uint64) float64 {
	roll := w.Random(w.chop.tree.gx, w.chop.tree.gz, salt+uint64(w.hero.attempts))
	w.hero.attempts++
	return roll
}

// cutLog gives the hero a log and its experience. A normal tree falls with its first log; other
// trees fall with the first log after their depletion timer runs out.
func (w *World) cutLog() {
	kind := species[w.chop.kind]
	w.hero.logs++
	w.hero.woodcutting.gain(kind.xp)
	if !kind.fallsWithFirstLog && w.depletion[w.chop.tree] < kind.lifetime {
		return
	}
	if w.felled == nil {
		w.felled = make(map[cell]bool)
	}
	w.felled[w.chop.tree] = true
	delete(w.depletion, w.chop.tree)
	w.chop = chop{}
}
