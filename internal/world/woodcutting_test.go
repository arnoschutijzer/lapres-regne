package world_test

import (
	"testing"
	"time"

	"dither-cli/internal/world"
)

// besideTree stands the hero 1.1 left of the seed 7 tree in cell 1, 0 (x 6.0, z 1.7), facing it.
func besideTree() *world.World {
	w := world.New(7)
	for range 3 {
		w.Walk(0, 1, false)
	}
	for range 9 {
		w.Walk(1, 0, false)
	}
	return w
}

func TestNoTreeIsFacedAtTheSpawnClearing(t *testing.T) {
	w := world.New(7)
	if _, ok := w.FacedTree(); ok {
		t.Fatal("the spawn clearing must have no tree within reach")
	}
	if w.Chop() {
		t.Fatal("chopping must not start without a faced tree")
	}
}

func TestTheFacedTreeIsTheOneInFrontWithinReach(t *testing.T) {
	w := besideTree()
	tree, ok := w.FacedTree()
	if !ok || tree.X < 6 || tree.X > 6.1 {
		t.Fatalf("faced tree = %+v, %v; want the tree at x 6.0", tree, ok)
	}
	w.Hero().Face(world.FacingLeft)
	if _, ok := w.FacedTree(); ok {
		t.Fatal("a tree behind the hero must not be faced")
	}
}

func TestChoppingANormalTreeTakesOneTickAndLeavesAStump(t *testing.T) {
	w := besideTree()
	if !w.Chop() {
		t.Fatal("chopping must start at a faced tree")
	}
	w.Advance(0, 0, 599*time.Millisecond, false)
	if !w.Chopping() || w.Hero().Logs() != 0 {
		t.Fatalf("chopping = %v, logs = %d; want still chopping with no logs before the tick", w.Chopping(), w.Hero().Logs())
	}
	w.Advance(0, 0, time.Millisecond, false)
	if w.Chopping() || w.Hero().Logs() != 1 {
		t.Fatalf("chopping = %v, logs = %d; want one log and chopping stopped", w.Chopping(), w.Hero().Logs())
	}
	if tree, _ := w.TreeAt(1, 0); !tree.Stump {
		t.Fatal("a felled tree must leave a stump")
	}
	if _, ok := w.FacedTree(); ok {
		t.Fatal("a stump must not be faced as a tree to chop")
	}
	if w.Chop() {
		t.Fatal("a stump must not be chopped")
	}
}
