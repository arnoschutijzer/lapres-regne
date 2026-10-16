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
	if _, ok := w.TreeInReach(); ok {
		t.Fatal("the spawn clearing must have no tree within reach")
	}
	if w.Chop() {
		t.Fatal("chopping must not start without a tree in reach")
	}
}

func TestTheTreeInFrontIsInReach(t *testing.T) {
	w := besideTree()
	tree, ok := w.TreeInReach()
	if !ok || tree.X < 6 || tree.X > 6.1 {
		t.Fatalf("tree in reach = %+v, %v; want the tree at x 6.0", tree, ok)
	}
}

func TestOvershootingATreeStillReachesItAndChoppingTurnsTowardIt(t *testing.T) {
	w := besideTree()
	// Four more steps right leave the trunk 1.1 behind the hero.
	for range 4 {
		w.Walk(1, 0, false)
	}
	tree, ok := w.TreeInReach()
	if !ok || tree.X < 6 || tree.X > 6.1 {
		t.Fatalf("tree in reach = %+v, %v; want the tree at x 6.0 behind the hero", tree, ok)
	}
	if !w.Chop() {
		t.Fatal("chopping must start at a tree behind the hero")
	}
	if facing := w.Hero().Facing(); facing != world.FacingLeft {
		t.Fatalf("facing = %v, want the hero to turn left toward the tree", facing)
	}
}

func TestChoppingANormalTreeTakesOneTickAndLeavesAStump(t *testing.T) {
	w := besideTree()
	if !w.Chop() {
		t.Fatal("chopping must start at a tree in reach")
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
	if _, ok := w.TreeInReach(); ok {
		t.Fatal("a stump must not be faced as a tree to chop")
	}
	if w.Chop() {
		t.Fatal("a stump must not be chopped")
	}
}

func TestATreeCanBeChoppedFromTheEdgeOfItsCrown(t *testing.T) {
	w := world.New(7)
	for range 3 {
		w.Walk(0, 1, false)
	}
	// Seven steps right leaves the trunk 2.2 ahead, under the edge of the crown.
	for range 7 {
		w.Walk(1, 0, false)
	}
	if _, ok := w.TreeInReach(); !ok {
		t.Fatal("a tree 2.2 ahead must be within reach")
	}
}

func TestTheChoppedTreeIsKnownWhileChopping(t *testing.T) {
	w := besideTree()
	if _, ok := w.ChoppedTree(); ok {
		t.Fatal("no tree is chopped before chopping starts")
	}
	faced, _ := w.TreeInReach()
	w.Chop()
	if chopped, ok := w.ChoppedTree(); !ok || chopped != faced {
		t.Fatalf("chopped tree = %+v, %v; want the tree in reach %+v", chopped, ok, faced)
	}
	w.Advance(0, 0, time.Second, false)
	if _, ok := w.ChoppedTree(); ok {
		t.Fatal("no tree is chopped once it falls")
	}
}

func TestAChopHoldsTheFacingUntilTheTreeFalls(t *testing.T) {
	w := besideTree()
	w.Chop()
	w.AimAt(0, 0)
	if facing := w.Hero().Facing(); facing != world.FacingRight {
		t.Fatalf("facing = %v, want right toward the tree while chopping", facing)
	}
	w.Advance(0, 0, time.Second, false)
	w.AimAt(0, 0)
	if facing := w.Hero().Facing(); facing != world.FacingLeft {
		t.Fatalf("facing = %v, want aiming to work again once the tree falls", facing)
	}
}

type recordedSounds struct{ chops int }

func (s *recordedSounds) Chop() { s.chops++ }

func TestTheAxeSoundsAsTheChopStartsAndOnEveryTick(t *testing.T) {
	w := besideTree()
	sounds := &recordedSounds{}
	w.SetSounds(sounds)
	w.Chop()
	if sounds.chops != 1 {
		t.Fatalf("chops = %d, want the axe to land as chopping starts", sounds.chops)
	}
	w.Advance(0, 0, 599*time.Millisecond, false)
	if sounds.chops != 1 {
		t.Fatalf("chops = %d, want no strike before the tick", sounds.chops)
	}
	w.Advance(0, 0, time.Millisecond, false)
	if sounds.chops != 2 {
		t.Fatalf("chops = %d, want the felling strike on the tick", sounds.chops)
	}
}

func TestTheLastStrikeIsKnownWhileChopping(t *testing.T) {
	w := besideTree()
	if _, _, ok := w.LastStrike(); ok {
		t.Fatal("no strike is known before chopping")
	}
	w.Chop()
	w.Advance(0, 0, 250*time.Millisecond, false)
	if index, since, ok := w.LastStrike(); !ok || index != 0 || since != 250*time.Millisecond {
		t.Fatalf("last strike = %d, %v, %v; want the first strike 250ms ago", index, since, ok)
	}
}
