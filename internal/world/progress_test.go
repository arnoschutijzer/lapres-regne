package world_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"lapresregne/internal/world"
)

// memoryProgress keeps one snapshot in memory.
type memoryProgress struct {
	snapshot world.Snapshot
	saved    bool
	err      error
}

func (p *memoryProgress) Load() (world.Snapshot, bool, error) { return p.snapshot, p.saved, p.err }

func (p *memoryProgress) Save(snapshot world.Snapshot) error {
	p.snapshot, p.saved = snapshot, true
	return p.err
}

var quitAt = time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC)

// keptAfterFelling chops the tree beside the hero down and keeps the progress at quitAt.
func keptAfterFelling(t *testing.T) *memoryProgress {
	t.Helper()
	w := besideTree()
	w.Chop()
	chopUntilFelled(t, w)
	progress := &memoryProgress{}
	if err := w.Keep(progress, quitAt); err != nil {
		t.Fatal(err)
	}
	return progress
}

func TestResumingRestoresTheHeroWhereTheyStood(t *testing.T) {
	kept := besideTree()
	kept.Chop()
	chopUntilFelled(t, kept)
	progress := &memoryProgress{}
	kept.Keep(progress, quitAt)
	w, err := world.Resume(7, progress, quitAt.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	x, z := w.Hero().Position()
	keptX, keptZ := kept.Hero().Position()
	if x != keptX || z != keptZ {
		t.Fatalf("position = %v, %v; want %v, %v", x, z, keptX, keptZ)
	}
	if cameraX, cameraZ := w.Camera(); cameraX != x || cameraZ != z {
		t.Fatalf("camera = %v, %v; want it on the hero", cameraX, cameraZ)
	}
	if w.Hero().Inventory().Len() != 1 || w.Hero().Skill(world.Woodcutting).XP() != 25 {
		t.Fatalf("logs = %d, xp = %v; want the kept log and its 25 xp", w.Hero().Inventory().Len(), w.Hero().Skill(world.Woodcutting).XP())
	}
	if tree, _ := w.TreeAt(1, 0); !tree.Stump {
		t.Fatal("a stump must still be a stump 10 seconds later")
	}
}

func TestTimeAwayCountsTowardRegrowth(t *testing.T) {
	progress := keptAfterFelling(t)
	regrow := progress.snapshot.Stumps[0].Regrow
	w, _ := world.Resume(7, progress, quitAt.Add(regrow-time.Second))
	if tree, _ := w.TreeAt(1, 0); !tree.Stump {
		t.Fatal("the stump must wait out the rest of its regrow time")
	}
	w.Advance(0, 0, time.Second, false)
	if tree, _ := w.TreeAt(1, 0); tree.Stump {
		t.Fatal("the stump must regrow once its regrow time has passed, counting time away")
	}
	away, _ := world.Resume(7, progress, quitAt.Add(2*time.Minute))
	if tree, _ := away.TreeAt(1, 0); tree.Stump {
		t.Fatal("a tree must have regrown while the hero was away long enough")
	}
}

func TestAClockSetBackDoesNotStretchRegrowth(t *testing.T) {
	progress := keptAfterFelling(t)
	regrow := progress.snapshot.Stumps[0].Regrow
	w, _ := world.Resume(7, progress, quitAt.Add(-time.Hour))
	w.Advance(0, 0, regrow, false)
	if tree, _ := w.TreeAt(1, 0); tree.Stump {
		t.Fatal("a clock earlier than the save must count as no time away")
	}
}

func TestResumingWithoutSavedProgressStartsFresh(t *testing.T) {
	w, err := world.Resume(7, &memoryProgress{}, quitAt)
	if err != nil {
		t.Fatal(err)
	}
	if x, z := w.Hero().Position(); x != 0 || z != 0 || w.Hero().Inventory().Len() != 0 {
		t.Fatalf("position = %v, %v, logs = %d; want a fresh start", x, z, w.Hero().Inventory().Len())
	}
}

func TestResumingAnotherSeedKeepsOnlySkillsAndLogs(t *testing.T) {
	progress := keptAfterFelling(t)
	w, err := world.Resume(8, progress, quitAt)
	if err != nil {
		t.Fatal(err)
	}
	if x, z := w.Hero().Position(); x != 0 || z != 0 {
		t.Fatalf("position = %v, %v; want the spawn point of the other world", x, z)
	}
	if w.Hero().Inventory().Len() != 1 || w.Hero().Skill(world.Woodcutting).XP() != 25 {
		t.Fatalf("logs = %d, xp = %v; want the hero's logs and xp to travel along", w.Hero().Inventory().Len(), w.Hero().Skill(world.Woodcutting).XP())
	}
	for gz := -3; gz <= 3; gz++ {
		for gx := -3; gx <= 3; gx++ {
			if tree, ok := w.TreeAt(gx, gz); ok && tree.Stump {
				t.Fatalf("stump at %d, %d; want no stumps from another world", gx, gz)
			}
		}
	}
}

func TestResumingReportsProgressThatCannotBeLoaded(t *testing.T) {
	broken := errors.New("unreadable")
	if _, err := world.Resume(7, &memoryProgress{err: broken}, quitAt); !errors.Is(err, broken) {
		t.Fatalf("err = %v, want the load failure", err)
	}
}

func TestKeepingReportsProgressThatCannotBeSaved(t *testing.T) {
	broken := errors.New("disk full")
	if err := besideTree().Keep(&memoryProgress{err: broken}, quitAt); !errors.Is(err, broken) {
		t.Fatalf("err = %v, want the save failure", err)
	}
}

func TestKeepingAndResumingCarryTheInventoryItemByItem(t *testing.T) {
	// Slots stay where they were, gaps included.
	carried := make([]world.Slot, world.InventorySize)
	carried[0] = world.Slot{Item: world.OakLogs, Filled: true}
	carried[2] = world.Slot{Item: world.WillowLogs, Filled: true}
	carried[27] = world.Slot{Item: world.YewLogs, Filled: true}
	progress := &memoryProgress{snapshot: world.Snapshot{Seed: 7, Inventory: carried, SavedAt: quitAt}, saved: true}
	w, err := world.Resume(7, progress, quitAt)
	if err != nil {
		t.Fatal(err)
	}
	if slots := w.Hero().Inventory().Slots(); !reflect.DeepEqual(slots, carried) {
		t.Fatalf("slots = %v, want %v", slots, carried)
	}
	w.Keep(progress, quitAt)
	if !reflect.DeepEqual(progress.snapshot.Inventory, carried) {
		t.Fatalf("kept inventory = %v, want %v", progress.snapshot.Inventory, carried)
	}
}

func TestResumingNeverOverfillsTheInventory(t *testing.T) {
	overfull := make([]world.Slot, world.InventorySize+5)
	for i := range overfull {
		overfull[i] = world.Slot{Item: world.Logs, Filled: true}
	}
	progress := &memoryProgress{snapshot: world.Snapshot{Seed: 7, Inventory: overfull}, saved: true}
	w, _ := world.Resume(7, progress, quitAt)
	if w.Hero().Inventory().Len() != world.InventorySize {
		t.Fatalf("slots = %d, want at most %d", w.Hero().Inventory().Len(), world.InventorySize)
	}
}
