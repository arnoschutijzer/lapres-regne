package world

import (
	"math"
	"testing"
	"time"
)

func TestTheChanceToCutALogRisesWithLevel(t *testing.T) {
	if chance := species[NormalTree].chance(1); chance != 65.0/256 {
		t.Fatalf("normal tree chance at level 1 = %v, want 65/256", chance)
	}
	if chance := species[NormalTree].chance(99); chance != 201.0/256 {
		t.Fatalf("normal tree chance at level 99 = %v, want 201/256", chance)
	}
	if chance := species[Oak].chance(15); chance >= species[NormalTree].chance(15) {
		t.Fatalf("oak chance at level 15 = %v, want harder than a normal tree", chance)
	}
	for _, kind := range []TreeKind{NormalTree, Oak, Willow, Yew} {
		for level := 2; level <= 99; level++ {
			if species[kind].chance(level) < species[kind].chance(level-1) {
				t.Fatalf("%v chance at level %d drops below level %d", kind, level, level-1)
			}
		}
	}
}

// standBeside places a hero with enough experience beside a tree, facing it.
func standBeside(w *World, tree Tree, xp float64) {
	w.hero.x, w.hero.z = tree.X-1, tree.Z
	w.hero.Face(FacingRight)
	w.hero.woodcutting = Skill{}
	w.hero.woodcutting.gain(xp)
}

// nearestOaks finds the oaks closest to the spawn point.
func nearestOaks(w *World, count int) []Tree {
	var oaks []Tree
	for radius := 0; len(oaks) < count; radius++ {
		for gz := -radius; gz <= radius; gz++ {
			for gx := -radius; gx <= radius; gx++ {
				if max(abs(gx), abs(gz)) != radius {
					continue
				}
				if tree, ok := w.TreeAt(gx, gz); ok && tree.Kind == Oak && len(oaks) < count {
					oaks = append(oaks, tree)
				}
			}
		}
	}
	return oaks
}

func abs(n int) int { return max(n, -n) }

// chopFor keeps chopping in game ticks until the tree falls or a limit passes, returning the time spent.
func chopFor(w *World, limit time.Duration) time.Duration {
	elapsed := time.Duration(0)
	for w.Chopping() && elapsed < limit {
		w.Advance(0, 0, gameTick, false)
		elapsed += gameTick
	}
	return elapsed
}

func TestOaksGiveTheirXPAndFallOnlyOnceTheirTimerRunsOut(t *testing.T) {
	w := New(7)
	oaks, logs := nearestOaks(w, 8), 0
	for _, tree := range oaks {
		standBeside(w, tree, levelXP[15])
		before, startXP := w.hero.inventory.Len(), w.hero.woodcutting.XP()
		if !w.Chop() {
			t.Fatalf("a level 15 hero must chop the oak at x %.1f z %.1f: %s", tree.X, tree.Z, w.Message())
		}
		if elapsed := chopFor(w, 10*time.Minute); elapsed < 27*time.Second {
			t.Fatalf("the oak at x %.1f z %.1f fell after %v of chopping, want at least 27s", tree.X, tree.Z, elapsed)
		}
		gained := w.hero.inventory.Len() - before
		if xp := w.hero.woodcutting.XP() - startXP; math.Abs(xp-37.5*float64(gained)) > 1e-9 {
			t.Fatalf("xp = %v for %d oak logs, want 37.5 each", xp, gained)
		}
		logs += gained
	}
	if logs <= len(oaks) {
		t.Fatalf("%d oaks gave %d logs; want oaks often giving more than one log", len(oaks), logs)
	}
}

func TestAnOaksTimerRecoversWhileNobodyChopsIt(t *testing.T) {
	w := New(7)
	tree := nearestOaks(w, 1)[0]
	standBeside(w, tree, levelXP[15])
	w.Chop()
	chopFor(w, 20*time.Second)
	w.Stop()
	w.Advance(0, 0, 20*time.Second, false)
	w.Chop()
	if elapsed := chopFor(w, 10*time.Minute); elapsed < 27*time.Second {
		t.Fatalf("the oak fell %v into a fresh chop, want its timer back to the full 27s", elapsed)
	}
}

// nearestOf finds the tree of a kind closest to the spawn point.
func nearestOf(w *World, kind TreeKind) Tree {
	for radius := 0; ; radius++ {
		for gz := -radius; gz <= radius; gz++ {
			for gx := -radius; gx <= radius; gx++ {
				if tree, ok := w.TreeAt(gx, gz); ok && tree.Kind == kind && max(abs(gx), abs(gz)) == radius {
					return tree
				}
			}
		}
	}
}

func TestOaksWillowsAndYewsRegrowAfterTheirRespawnTime(t *testing.T) {
	cases := []struct {
		kind    TreeKind
		respawn time.Duration
	}{
		{Oak, 14 * gameTick}, {Willow, 14 * gameTick}, {Yew, 99 * gameTick},
	}
	for _, c := range cases {
		t.Run(c.kind.String(), func(t *testing.T) {
			w := New(7)
			tree := nearestOf(w, c.kind)
			standBeside(w, tree, levelXP[99])
			if !w.Chop() {
				t.Fatalf("a level 99 hero must chop the %v: %s", c.kind, w.Message())
			}
			chopFor(w, time.Hour)
			if felled, _ := w.TreeAt(tree.cell.gx, tree.cell.gz); !felled.Stump {
				t.Fatalf("the %v must fall", c.kind)
			}
			w.Advance(0, 0, c.respawn-time.Millisecond, false)
			if waiting, _ := w.TreeAt(tree.cell.gx, tree.cell.gz); !waiting.Stump {
				t.Fatalf("the %v regrew before %v", c.kind, c.respawn)
			}
			w.Advance(0, 0, time.Millisecond, false)
			if regrown, _ := w.TreeAt(tree.cell.gx, tree.cell.gz); regrown.Stump {
				t.Fatalf("the %v must regrow after %v", c.kind, c.respawn)
			}
		})
	}
}

// countedFalls counts the trees heard falling.
type countedFalls struct {
	silence
	falls int
}

func (c *countedFalls) Fall() { c.falls++ }

func TestAnOakFallsOnlyOnceAfterGivingSeveralLogs(t *testing.T) {
	w := New(7)
	sounds := &countedFalls{}
	w.SetSounds(sounds)
	tree := nearestOf(w, Oak)
	standBeside(w, tree, levelXP[99])
	w.Chop()
	chopFor(w, time.Hour)
	if logs := w.hero.inventory.Len(); logs < 2 {
		t.Fatalf("logs = %d, want an oak giving several logs before it falls", logs)
	}
	if sounds.falls != 1 {
		t.Fatalf("falls = %d, want the oak heard falling once", sounds.falls)
	}
}

const inventoryFull = "Your inventory is too full to carry any more logs."

// fillInventory packs the hero's inventory with normal logs until a number of slots is used.
func fillInventory(w *World, used int) {
	for w.hero.inventory.Len() < used {
		w.hero.inventory.add(Logs)
	}
}

func TestAFullInventoryRefusesChopping(t *testing.T) {
	w := New(7)
	tree := nearestOf(w, NormalTree)
	standBeside(w, tree, 0)
	fillInventory(w, InventorySize)
	if w.Chop() || w.Chopping() {
		t.Fatal("a full inventory must not start a chop")
	}
	if w.Message() != inventoryFull {
		t.Fatalf("message = %q, want %q", w.Message(), inventoryFull)
	}
}

func TestChoppingStopsOnceTheLastSlotFills(t *testing.T) {
	w := New(7)
	tree := nearestOf(w, Oak)
	standBeside(w, tree, levelXP[99])
	fillInventory(w, InventorySize-1)
	w.Chop()
	chopFor(w, time.Hour)
	if w.Chopping() || !w.hero.inventory.Full() {
		t.Fatalf("chopping = %v, slots = %d; want chopping stopped with a full inventory", w.Chopping(), w.hero.inventory.Len())
	}
	if standing, _ := w.TreeAt(tree.cell.gx, tree.cell.gz); standing.Stump {
		t.Fatal("an oak giving one log must keep standing")
	}
	if w.Message() != inventoryFull {
		t.Fatalf("message = %q, want %q", w.Message(), inventoryFull)
	}
}
