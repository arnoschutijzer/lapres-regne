package world

import (
	"math"
	"testing"
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

func TestOaksGiveTheirXPAndMayGiveSeveralLogsBeforeFalling(t *testing.T) {
	w := New(7)
	oaks, logs := 0, 0
	for gz := -20; gz <= 20 && oaks < 8; gz++ {
		for gx := -20; gx <= 20 && oaks < 8; gx++ {
			tree, ok := w.TreeAt(gx, gz)
			if !ok || tree.Kind != Oak {
				continue
			}
			oaks++
			standBeside(w, tree, levelXP[15])
			before, startXP := w.hero.logs, w.hero.woodcutting.XP()
			if !w.Chop() {
				t.Fatalf("a level 15 hero must chop the oak at x %.1f z %.1f: %s", tree.X, tree.Z, w.Message())
			}
			for range 2000 {
				if !w.Chopping() {
					break
				}
				w.Advance(0, 0, chopTick, false)
			}
			gained := w.hero.logs - before
			if xp := w.hero.woodcutting.XP() - startXP; math.Abs(xp-37.5*float64(gained)) > 1e-9 {
				t.Fatalf("xp = %v for %d oak logs, want 37.5 each", xp, gained)
			}
			logs += gained
		}
	}
	if oaks < 8 || logs <= oaks {
		t.Fatalf("oaks = %d gave logs = %d; want oaks often giving more than one log", oaks, logs)
	}
}
