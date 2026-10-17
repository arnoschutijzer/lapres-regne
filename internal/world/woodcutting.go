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

// inventoryFullMessage is Old School RuneScape's reply when no log fits in the inventory.
const inventoryFullMessage = "Your inventory is too full to carry any more logs."

// arrival is how close to a trunk an approach ends, comfortably within reach.
const arrival = 1.2

// cell identifies a tree by its place on the tree grid.
type cell struct{ gx, gz int }

// chop is the order to chop a tree until it falls or the inventory fills.
type chop struct {
	world   *World
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
	spacing := w.TreeGrid()
	for gz := int(math.Floor((z - reach) / spacing)); gz <= int(math.Floor((z+reach)/spacing)); gz++ {
		for gx := int(math.Floor((x - reach) / spacing)); gx <= int(math.Floor((x+reach)/spacing)); gx++ {
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
	w.order = w.startChop(tree)
	return w.order != nil
}

// startChop turns the hero toward a tree and returns the order to chop it, or nil with a message
// where the tree is above the hero's level or the inventory is full.
func (w *World) startChop(tree Tree) order {
	x, z := w.hero.Position()
	w.hero.Face(FacingToward(tree.X-x, tree.Z-z))
	if required := species[tree.Kind].level; w.hero.woodcutting.Level() < required {
		w.message = fmt.Sprintf("You need woodcutting level %d to chop this %s.", required, tree.Kind)
		return nil
	}
	if w.hero.inventory.Full() {
		w.message = inventoryFullMessage
		return nil
	}
	w.message = ""
	w.sounds.Chop()
	return &chop{world: w, tree: tree.cell, kind: tree.Kind}
}

// Message is the game's reply to the hero's latest action, or empty when there is none.
func (w *World) Message() string { return w.message }

// Approach walks the hero to a standing tree, round whatever blocks the way, and chops it on arrival,
// reporting whether the tree still stands.
func (w *World) Approach(tree Tree) bool {
	standing, exists := w.TreeAt(tree.cell.gx, tree.cell.gz)
	if !exists || standing.Stump {
		return false
	}
	w.Stop()
	route := w.route(standing.X, standing.Z)
	w.order = &closeIn{
		world:     w,
		target:    point{standing.X, standing.Z},
		within:    arrival,
		waypoints: route[:len(route)-1],
		arrive:    func() order { return w.startChop(standing) },
	}
	return true
}

// chopping is the chop in progress, if any.
func (w *World) chopping() (*chop, bool) {
	chop, ok := w.order.(*chop)
	return chop, ok
}

func (w *World) Chopping() bool {
	_, chopping := w.chopping()
	return chopping
}

// AimAt faces the hero toward a world position, except while a chop holds them toward the tree.
func (w *World) AimAt(x, z float64) {
	if w.Chopping() {
		return
	}
	w.hero.AimAt(x, z)
}

// LastStrike tells which strike of the chop landed last, counting from zero, and how long ago.
func (w *World) LastStrike() (index int, since time.Duration, ok bool) {
	chop, chopping := w.chopping()
	if !chopping {
		return 0, 0, false
	}
	return int(chop.elapsed / StrikeInterval), chop.elapsed % StrikeInterval, true
}

// ChoppedTree is the tree being chopped, if any.
func (w *World) ChoppedTree() (Tree, bool) {
	chop, chopping := w.chopping()
	if !chopping {
		return Tree{}, false
	}
	return w.TreeAt(chop.tree.gx, chop.tree.gz)
}

// carryOut swings the axe at its own pace and rolls for a log every four ticks. Trees that give
// several logs wear down their depletion timer while chopped.
func (c *chop) carryOut(dt time.Duration, _ float64) order {
	w := c.world
	c.elapsed += dt
	if !species[c.kind].fallsWithFirstLog {
		if w.depletion == nil {
			w.depletion = make(map[cell]time.Duration)
		}
		w.depletion[c.tree] += dt
	}
	for c.elapsed >= time.Duration(c.strikes+1)*StrikeInterval {
		c.strikes++
		w.sounds.Chop()
	}
	for c.elapsed >= time.Duration(c.rolls+1)*rollInterval {
		c.rolls++
		if w.roll(c.tree, 0x10c5) < species[c.kind].chance(w.hero.woodcutting.Level()) && !w.cutLog(c) {
			return nil
		}
	}
	return c
}

// regrow brings stumps back as trees once their respawn time has passed.
func (w *World) regrow(dt time.Duration) {
	for tree, left := range w.felled {
		if left <= dt {
			delete(w.felled, tree)
		} else {
			w.felled[tree] = left - dt
		}
	}
}

// recoverTimers lets every tree nobody is chopping regain its depletion timer as fast as it wore down.
func (w *World) recoverTimers(dt time.Duration) {
	chop, chopping := w.chopping()
	for tree, worn := range w.depletion {
		if chopping && tree == chop.tree {
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
func (w *World) roll(tree cell, salt uint64) float64 {
	roll := w.Random(tree.gx, tree.gz, salt+uint64(w.hero.attempts))
	w.hero.attempts++
	return roll
}

// cutLog gives the hero a log and its experience, and reports whether chopping goes on. A normal
// tree falls with its first log; other trees fall with the first log after their depletion timer
// runs out.
func (w *World) cutLog(chop *chop) bool {
	kind := species[chop.kind]
	w.hero.inventory.add(kind.logs)
	level := w.hero.woodcutting.Level()
	w.hero.woodcutting.gain(kind.xp)
	if reached := w.hero.woodcutting.Level(); reached > level {
		w.celebrate(level, reached)
	}
	if !kind.fallsWithFirstLog && w.depletion[chop.tree] < kind.lifetime {
		if w.hero.inventory.Full() {
			w.message = inventoryFullMessage
			return false
		}
		return true
	}
	if w.felled == nil {
		w.felled = make(map[cell]time.Duration)
	}
	w.felled[chop.tree] = kind.respawn + time.Duration(w.roll(chop.tree, 0x9e60)*float64(kind.respawnSpread))
	delete(w.depletion, chop.tree)
	w.sounds.Fall()
	return false
}
