package world

import (
	"fmt"
	"math"
	"strings"
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

// woods is what woodcutting needs from the world around it.
type woods interface {
	TreeAt(gx, gz int) (Tree, bool)
	TreeGrid() float64
	Random(x, z int, salt uint64) float64
	Hero() *Character
	// say gives the game's reply to the hero's latest action.
	say(message string)
	play(Effect)
	// train gives the hero experience in a skill, celebrating a new level with what it unlocks.
	train(skill string, xp float64, unlocked unlocks)
}

// woodcutting is the woodcutting skill at work in the woods: the trees it felled and wore down,
// and the rolls it drew.
type woodcutting struct {
	woods woods
	// felled holds the cells whose tree is a stump, with the time left until it regrows.
	felled map[cell]time.Duration
	// depletion is how long each tree that gives several logs has been worn down by chopping.
	depletion map[cell]time.Duration
	// attempts counts every roll for a log, so a restarted chop rolls afresh.
	attempts int
}

// chop is the order to chop a tree until it falls or the inventory fills.
type chop struct {
	woodcutting *woodcutting
	tree        cell
	kind        TreeKind
	elapsed     time.Duration
	strikes     int
	rolls       int
}

// stump tells whether the tree in a cell is felled.
func (wc *woodcutting) stump(tree cell) bool { return wc.felled[tree] > 0 }

// treeInReach finds the standing tree the hero acts on: the nearest one in front within reach,
// else the nearest one within reach in any direction, so overshooting a tree still reaches it.
func (wc *woodcutting) treeInReach() (Tree, bool) {
	hero := wc.woods.Hero()
	x, z := hero.Position()
	var nearest Tree
	found, best := false, math.Inf(1)
	spacing := wc.woods.TreeGrid()
	for gz := int(math.Floor((z - reach) / spacing)); gz <= int(math.Floor((z+reach)/spacing)); gz++ {
		for gx := int(math.Floor((x - reach) / spacing)); gx <= int(math.Floor((x+reach)/spacing)); gx++ {
			tree, exists := wc.woods.TreeAt(gx, gz)
			if !exists || tree.Stump {
				continue
			}
			distance := math.Hypot(tree.X-x, tree.Z-z)
			if distance > reach {
				continue
			}
			// Any tree in front outranks every tree beside or behind the hero.
			score := distance
			if FacingToward(tree.X-x, tree.Z-z) != hero.Facing() {
				score += reach
			}
			if score < best {
				nearest, found, best = tree, true, score
			}
		}
	}
	return nearest, found
}

// chopInReach returns the order to chop the tree in reach, or nil where none is or it cannot be
// chopped.
func (wc *woodcutting) chopInReach() order {
	tree, ok := wc.treeInReach()
	if !ok {
		return nil
	}
	return wc.start(tree)
}

// standing is a tree as it stands now, or false where it is felled.
func (wc *woodcutting) standing(tree Tree) (Tree, bool) {
	standing, exists := wc.woods.TreeAt(tree.cell.gx, tree.cell.gz)
	return standing, exists && !standing.Stump
}

// start turns the hero toward a tree and returns the order to chop it, or nil with a message
// where the tree is above the hero's level or the inventory is full.
func (wc *woodcutting) start(tree Tree) order {
	hero := wc.woods.Hero()
	x, z := hero.Position()
	hero.Face(FacingToward(tree.X-x, tree.Z-z))
	if required := species[tree.Kind].level; hero.Skill(Woodcutting).Level() < required {
		wc.woods.say(fmt.Sprintf("You need woodcutting level %d to chop this %s.", required, tree.Kind))
		return nil
	}
	if hero.inventory.Full() {
		wc.woods.say(inventoryFullMessage)
		return nil
	}
	wc.woods.say("")
	wc.woods.play(AxeStrike)
	return &chop{woodcutting: wc, tree: tree.cell, kind: tree.Kind}
}

// holdsAim keeps the hero facing the tree while chopping it.
func (c *chop) holdsAim() {}

// chopped is the tree being chopped.
func (c *chop) chopped() (Tree, bool) { return c.woodcutting.woods.TreeAt(c.tree.gx, c.tree.gz) }

// lastStrike tells which strike of the chop landed last, counting from zero, and how long ago.
func (c *chop) lastStrike() (index int, since time.Duration) {
	return int(c.elapsed / StrikeInterval), c.elapsed % StrikeInterval
}

// carryOut swings the axe at its own pace and rolls for a log every four ticks. Trees that give
// several logs wear down their depletion timer while chopped.
func (c *chop) carryOut(dt time.Duration, _ float64) order {
	wc := c.woodcutting
	c.elapsed += dt
	if !species[c.kind].fallsWithFirstLog {
		if wc.depletion == nil {
			wc.depletion = make(map[cell]time.Duration)
		}
		wc.depletion[c.tree] += dt
	}
	for c.elapsed >= time.Duration(c.strikes+1)*StrikeInterval {
		c.strikes++
		wc.woods.play(AxeStrike)
	}
	level := wc.woods.Hero().Skill(Woodcutting).Level()
	for c.elapsed >= time.Duration(c.rolls+1)*rollInterval {
		c.rolls++
		if wc.roll(c.tree, 0x10c5) < species[c.kind].chance(level) && !wc.cutLog(c) {
			return nil
		}
	}
	return c
}

// advance regrows stumps once their respawn time has passed, and lets every tree but the one being
// chopped regain its depletion timer as fast as it wore down.
func (wc *woodcutting) advance(dt time.Duration, chopping *chop) {
	for tree, left := range wc.felled {
		if left <= dt {
			delete(wc.felled, tree)
		} else {
			wc.felled[tree] = left - dt
		}
	}
	for tree, worn := range wc.depletion {
		if chopping != nil && tree == chopping.tree {
			continue
		}
		if worn <= dt {
			delete(wc.depletion, tree)
		} else {
			wc.depletion[tree] = worn - dt
		}
	}
}

// roll draws this attempt's random number from the world seed, the tree being chopped and a salt.
func (wc *woodcutting) roll(tree cell, salt uint64) float64 {
	roll := wc.woods.Random(tree.gx, tree.gz, salt+uint64(wc.attempts))
	wc.attempts++
	return roll
}

// cutLog gives the hero a log and its experience, and reports whether chopping goes on. A normal
// tree falls with its first log; other trees fall with the first log after their depletion timer
// runs out.
func (wc *woodcutting) cutLog(chop *chop) bool {
	kind := species[chop.kind]
	hero := wc.woods.Hero()
	hero.inventory.add(kind.logs)
	wc.woods.train(Woodcutting, kind.xp, unlockedTrees)
	if !kind.fallsWithFirstLog && wc.depletion[chop.tree] < kind.lifetime {
		if hero.inventory.Full() {
			wc.woods.say(inventoryFullMessage)
			return false
		}
		return true
	}
	wc.fell(chop.tree, kind.respawn+time.Duration(wc.roll(chop.tree, 0x9e60)*float64(kind.respawnSpread)))
	delete(wc.depletion, chop.tree)
	wc.woods.play(TreeFall)
	return false
}

// fell leaves a stump in a cell that regrows into a tree after regrow.
func (wc *woodcutting) fell(tree cell, regrow time.Duration) {
	if wc.felled == nil {
		wc.felled = make(map[cell]time.Duration)
	}
	wc.felled[tree] = regrow
}

// unlockedTrees names the trees going from one woodcutting level to another lets the hero chop.
func unlockedTrees(from, to int) string {
	var names []string
	for kind, tree := range species {
		if tree.level > from && tree.level <= to {
			names = append(names, TreeKind(kind).String()+"s")
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "You can now chop " + strings.Join(names, " and ") + "."
}

// respawns lists the felled trees and how long each still takes to regrow.
func (wc *woodcutting) respawns() []Respawn {
	var respawns []Respawn
	for tree, regrow := range wc.felled {
		respawns = append(respawns, Respawn{GX: tree.gx, GZ: tree.gz, In: regrow})
	}
	return respawns
}

// deplete fells a tree until it regrows.
func (wc *woodcutting) deplete(respawn Respawn) {
	wc.fell(cell{respawn.GX, respawn.GZ}, respawn.In)
}

// TreeInReach finds the standing tree the hero acts on: the nearest one in front within reach,
// else the nearest one within reach in any direction, so overshooting a tree still reaches it.
func (w *World) TreeInReach() (Tree, bool) { return w.woodcutting.treeInReach() }

// Chop orders the hero to chop the tree in reach, reporting whether chopping started. A tree above
// the hero's level leaves a message instead.
func (w *World) Chop() bool {
	chop := w.woodcutting.chopInReach()
	if chop == nil {
		return false
	}
	w.order = chop
	return true
}

// Approach walks the hero to a standing tree, round whatever blocks the way, and chops it on arrival,
// reporting whether the tree still stands.
func (w *World) Approach(tree Tree) bool {
	standing, ok := w.woodcutting.standing(tree)
	if !ok {
		return false
	}
	w.closeInOn(point{standing.X, standing.Z}, arrival, func() order { return w.woodcutting.start(standing) })
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

// LastStrike tells which strike of the chop landed last, counting from zero, and how long ago.
func (w *World) LastStrike() (index int, since time.Duration, ok bool) {
	chop, chopping := w.chopping()
	if !chopping {
		return 0, 0, false
	}
	index, since = chop.lastStrike()
	return index, since, true
}

// ChoppedTree is the tree being chopped, if any.
func (w *World) ChoppedTree() (Tree, bool) {
	chop, chopping := w.chopping()
	if !chopping {
		return Tree{}, false
	}
	return chop.chopped()
}
