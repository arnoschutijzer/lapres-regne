package world_test

import (
	"math"
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

func TestChoppingANormalTreeLeavesAStumpWithItsFirstLog(t *testing.T) {
	w := besideTree()
	if !w.Chop() {
		t.Fatal("chopping must start at a tree in reach")
	}
	w.Advance(0, 0, 599*time.Millisecond, false)
	if !w.Chopping() || w.Hero().Inventory().Len() != 0 {
		t.Fatalf("chopping = %v, logs = %d; want still chopping with no logs before the tick", w.Chopping(), w.Hero().Inventory().Len())
	}
	chopUntilFelled(t, w)
	if w.Chopping() || w.Hero().Inventory().Len() != 1 {
		t.Fatalf("chopping = %v, logs = %d; want one log and chopping stopped", w.Chopping(), w.Hero().Inventory().Len())
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
	chopUntilFelled(t, w)
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
	chopUntilFelled(t, w)
	w.AimAt(0, 0)
	if facing := w.Hero().Facing(); facing != world.FacingLeft {
		t.Fatalf("facing = %v, want aiming to work again once the tree falls", facing)
	}
}

type recordedSounds struct {
	chops int
	// levelUps records each level-up jingle, true when the new level unlocked something.
	levelUps []bool
}

func (s *recordedSounds) Chop()                 { s.chops++ }
func (s *recordedSounds) LevelUp(unlocked bool) { s.levelUps = append(s.levelUps, unlocked) }

func TestTheAxeLandsAsTheChopStartsAndEveryStrikeAfter(t *testing.T) {
	w := besideTree()
	sounds := &recordedSounds{}
	w.SetSounds(sounds)
	w.Chop()
	if sounds.chops != 1 {
		t.Fatalf("chops = %d, want the axe to land as chopping starts", sounds.chops)
	}
	w.Advance(0, 0, world.StrikeInterval-time.Millisecond, false)
	if sounds.chops != 1 {
		t.Fatalf("chops = %d, want no strike before the interval", sounds.chops)
	}
	w.Advance(0, 0, time.Millisecond, false)
	if sounds.chops != 2 {
		t.Fatalf("chops = %d, want the next strike after one interval", sounds.chops)
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

// approachUntilChopping advances the world in frames until the hero chops or a few seconds pass.
func approachUntilChopping(w *world.World) {
	for range 40 {
		if w.Chopping() {
			return
		}
		w.Advance(0, 0, 100*time.Millisecond, false)
	}
}

func TestApproachingATreeWalksUpToItAndStartsChopping(t *testing.T) {
	w := world.New(7)
	tree, _ := w.TreeAt(1, 0)
	if !w.Approach(tree) {
		t.Fatal("a standing tree must be approachable")
	}
	approachUntilChopping(w)
	if !w.Chopping() {
		t.Fatal("the hero must start chopping on arrival")
	}
	x, z := w.Hero().Position()
	if distance := math.Hypot(tree.X-x, tree.Z-z); distance > 1.5 {
		t.Fatalf("distance to trunk = %.2f, want the hero right beside it", distance)
	}
	if facing := w.Hero().Facing(); facing != world.FacingRight {
		t.Fatalf("facing = %v, want right toward the tree", facing)
	}
}

func TestAStumpCannotBeApproached(t *testing.T) {
	w := besideTree()
	w.Chop()
	chopUntilFelled(t, w)
	if stump, _ := w.TreeAt(1, 0); w.Approach(stump) {
		t.Fatal("a stump must not be approached")
	}
}

func TestMovingCancelsAnApproachAndAChop(t *testing.T) {
	cases := []struct {
		name string
		move func(w *world.World)
	}{
		{"key press", func(w *world.World) { w.Walk(-1, 0, false) }},
		{"held key", func(w *world.World) { w.Advance(-1, 0, 10*time.Millisecond, false) }},
		{"stop", func(w *world.World) { w.Stop() }},
	}
	for _, c := range cases {
		t.Run(c.name+" during approach", func(t *testing.T) {
			w := world.New(7)
			tree, _ := w.TreeAt(1, 0)
			w.Approach(tree)
			w.Advance(0, 0, 100*time.Millisecond, false)
			c.move(w)
			approachUntilChopping(w)
			if w.Chopping() {
				t.Fatal("the approach must be cancelled")
			}
		})
		t.Run(c.name+" during chop", func(t *testing.T) {
			w := besideTree()
			w.Chop()
			c.move(w)
			w.Advance(0, 0, time.Second, false)
			if w.Chopping() || w.Hero().Inventory().Len() != 0 {
				t.Fatalf("chopping = %v, logs = %d; want the chop cancelled", w.Chopping(), w.Hero().Inventory().Len())
			}
		})
	}
}

// chopUntilFelled advances tick by tick until the chopped tree falls, returning the ticks taken.
func chopUntilFelled(t *testing.T, w *world.World) int {
	t.Helper()
	for ticks := 1; ticks <= 200; ticks++ {
		w.Advance(0, 0, 600*time.Millisecond, false)
		if !w.Chopping() {
			return ticks
		}
	}
	t.Fatal("the tree did not fall within 200 ticks")
	return 0
}

func TestAChopRollsEveryFourTicksUntilALogComes(t *testing.T) {
	w := besideTree()
	sounds := &recordedSounds{}
	w.SetSounds(sounds)
	w.Chop()
	ticks := chopUntilFelled(t, w)
	if ticks%4 != 0 {
		t.Fatalf("felled after %d ticks, want the log on a roll every 4 ticks", ticks)
	}
	elapsed := time.Duration(ticks) * 600 * time.Millisecond
	if strikes := 1 + int(elapsed/world.StrikeInterval); sounds.chops != strikes {
		t.Fatalf("chops = %d over %v, want %d strikes at their own pace", sounds.chops, elapsed, strikes)
	}
	if w.Hero().Inventory().Len() != 1 || w.Hero().Woodcutting().XP() != 25 {
		t.Fatalf("logs = %d, xp = %v; want one log worth 25 xp", w.Hero().Inventory().Len(), w.Hero().Woodcutting().XP())
	}
	again := besideTree()
	again.Chop()
	if replayed := chopUntilFelled(t, again); replayed != ticks {
		t.Fatalf("ticks = %d, then %d; want the same seed to roll the same chop", ticks, replayed)
	}
}

func TestANormalTreeRegrowsBetween36And60SecondsAfterFalling(t *testing.T) {
	w := besideTree()
	w.Chop()
	chopUntilFelled(t, w)
	w.Advance(0, 0, 36*time.Second-time.Millisecond, false)
	if tree, _ := w.TreeAt(1, 0); !tree.Stump {
		t.Fatal("a normal tree must stay a stump for at least 36 seconds")
	}
	w.Advance(0, 0, 24*time.Second, false)
	if tree, _ := w.TreeAt(1, 0); tree.Stump {
		t.Fatal("a normal tree must regrow within 60 seconds")
	}
	if _, ok := w.TreeInReach(); !ok {
		t.Fatal("a regrown tree must be chopped again")
	}
}
