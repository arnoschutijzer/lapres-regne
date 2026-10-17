package world_test

import (
	"reflect"
	"testing"
	"time"

	"lapresregne/internal/world"
)

// besideTreeWithXP resumes a hero with some woodcutting xp and stands them beside the seed 7 tree at x 6.0.
func besideTreeWithXP(t *testing.T, xp float64) *world.World {
	t.Helper()
	// Another seed's progress brings only the skills along, leaving the hero at the spawn.
	progress := &memoryProgress{saved: true, snapshot: world.Snapshot{Seed: 1, Skills: map[string]float64{world.Woodcutting: xp}}}
	w, err := world.Resume(7, progress, quitAt)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		w.Walk(0, 1, false)
	}
	for range 9 {
		w.Walk(1, 0, false)
	}
	return w
}

func TestNoLevelUpIsCelebratedBeforeOneIsReached(t *testing.T) {
	w := besideTree()
	w.Chop()
	chopUntilFelled(t, w)
	if _, _, ok := w.LatestLevelUp(); ok {
		t.Fatal("25 xp stays at level 1; nothing must be celebrated")
	}
}

func TestALogThatReachesTheNextLevelIsCelebrated(t *testing.T) {
	// Level 2 starts at 83 xp; one more normal log brings 75 xp to 100.
	w := besideTreeWithXP(t, 75)
	if _, _, ok := w.LatestLevelUp(); ok {
		t.Fatal("resuming at a level must not celebrate it")
	}
	w.Chop()
	chopUntilFelled(t, w)
	levelUp, since, ok := w.LatestLevelUp()
	if !ok || levelUp.Skill != "Woodcutting" || levelUp.Level != 2 || len(levelUp.Unlocked) != 0 {
		t.Fatalf("level up = %+v, %v; want woodcutting level 2 unlocking nothing", levelUp, ok)
	}
	if since != 0 {
		t.Fatalf("since = %v, want the level just reached", since)
	}
	w.Advance(0, 0, 1500*time.Millisecond, false)
	if _, since, _ := w.LatestLevelUp(); since != 1500*time.Millisecond {
		t.Fatalf("since = %v, want the time passed since the level was reached", since)
	}
}

func TestALevelThatOpensANewTreeNamesIt(t *testing.T) {
	// Level 15 starts at 2411 xp and lets the hero chop oaks.
	w := besideTreeWithXP(t, 2400)
	w.Chop()
	chopUntilFelled(t, w)
	levelUp, _, ok := w.LatestLevelUp()
	if !ok || levelUp.Level != 15 || !reflect.DeepEqual(levelUp.Unlocked, []world.TreeKind{world.Oak}) {
		t.Fatalf("level up = %+v, %v; want level 15 unlocking oaks", levelUp, ok)
	}
}

func TestALevelUpPlaysItsJingleOnce(t *testing.T) {
	w := besideTreeWithXP(t, 75)
	sounds := &recordedSounds{}
	w.SetSounds(sounds)
	w.Chop()
	chopUntilFelled(t, w)
	w.Advance(0, 0, time.Second, false)
	if !reflect.DeepEqual(sounds.levelUps, []bool{false}) {
		t.Fatalf("level-up jingles = %v, want one plain jingle for level 2", sounds.levelUps)
	}
	unlocking := besideTreeWithXP(t, 2400)
	sounds = &recordedSounds{}
	unlocking.SetSounds(sounds)
	unlocking.Chop()
	chopUntilFelled(t, unlocking)
	if !reflect.DeepEqual(sounds.levelUps, []bool{true}) {
		t.Fatalf("level-up jingles = %v, want the unlocking jingle for level 15", sounds.levelUps)
	}
}

func TestOrdinaryLevelsAreNoMilestone(t *testing.T) {
	w := besideTreeWithXP(t, 75)
	w.Chop()
	chopUntilFelled(t, w)
	if levelUp, _, _ := w.LatestLevelUp(); levelUp.Milestone != world.NoMilestone {
		t.Fatalf("milestone = %v at level 2, want none", levelUp.Milestone)
	}
}

func TestLevel99InTheOnlySkillReachesTheMaxTotalLevel(t *testing.T) {
	// Level 99 starts at 13,034,431 xp; one normal log brings the hero over it.
	w := besideTreeWithXP(t, 13034420)
	w.Chop()
	chopUntilFelled(t, w)
	levelUp, _, ok := w.LatestLevelUp()
	if !ok || levelUp.Level != 99 || levelUp.Milestone != world.TotalMaxed {
		t.Fatalf("level up = %+v, %v; want level 99 reaching the max total level", levelUp, ok)
	}
}
