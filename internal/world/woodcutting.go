package world

import (
	"math"
	"time"
)

// reach is how far a trunk may stand from the hero to be chopped.
const reach = 1.5

// chopTick is one woodcutting attempt; a normal tree falls on the first.
const chopTick = 600 * time.Millisecond

// cell identifies a tree by its place on the tree grid.
type cell struct{ gx, gz int }

type chop struct {
	active  bool
	tree    cell
	elapsed time.Duration
}

// FacedTree finds the nearest standing tree within reach in the direction the hero faces.
func (w *World) FacedTree() (Tree, bool) {
	tree, _, ok := w.facedTree()
	return tree, ok
}

func (w *World) facedTree() (Tree, cell, bool) {
	x, z := w.hero.Position()
	var faced Tree
	var facedCell cell
	found, nearest := false, reach
	for gz := int(math.Floor((z - reach) / TreeSpacing)); gz <= int(math.Floor((z+reach)/TreeSpacing)); gz++ {
		for gx := int(math.Floor((x - reach) / TreeSpacing)); gx <= int(math.Floor((x+reach)/TreeSpacing)); gx++ {
			tree, exists := w.TreeAt(gx, gz)
			if !exists || tree.Stump || FacingToward(tree.X-x, tree.Z-z) != w.hero.Facing() {
				continue
			}
			if distance := math.Hypot(tree.X-x, tree.Z-z); distance <= nearest {
				faced, facedCell, found, nearest = tree, cell{gx, gz}, true, distance
			}
		}
	}
	return faced, facedCell, found
}

// Chop starts chopping the faced tree and reports whether there was one.
func (w *World) Chop() bool {
	_, tree, ok := w.facedTree()
	if !ok {
		return false
	}
	w.chop = chop{active: true, tree: tree}
	return true
}

func (w *World) Chopping() bool { return w.chop.active }

// chopFor works on the chopped tree until it falls, leaving a stump and a log.
func (w *World) chopFor(dt time.Duration) {
	if !w.chop.active {
		return
	}
	w.chop.elapsed += dt
	if w.chop.elapsed < chopTick {
		return
	}
	if w.felled == nil {
		w.felled = make(map[cell]bool)
	}
	w.felled[w.chop.tree] = true
	w.hero.logs++
	w.chop = chop{}
}
