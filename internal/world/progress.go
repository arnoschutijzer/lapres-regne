package world

import (
	"fmt"
	"time"
)

// Keep saves the hero's progress, noting now so time away can count toward regrowth.
func (w *World) Keep(progress Progress, now time.Time) error {
	snapshot := Snapshot{
		Seed:          w.seed,
		X:             w.hero.x,
		Z:             w.hero.z,
		WoodcuttingXP: w.hero.woodcutting.XP(),
		Inventory:     w.hero.inventory.Slots(),
		SavedAt:       now,
	}
	for tree, regrow := range w.felled {
		snapshot.Stumps = append(snapshot.Stumps, Stump{GX: tree.gx, GZ: tree.gz, Regrow: regrow})
	}
	if err := progress.Save(snapshot); err != nil {
		return fmt.Errorf("save progress: %w", err)
	}
	return nil
}

// Resume builds a world from a seed and the saved progress. The hero's skills and inventory travel
// to any world; their position and stumps return only to the world they were saved in, with the
// time away since saving counted toward regrowth.
func Resume(seed uint64, progress Progress, now time.Time) (*World, error) {
	w := New(seed)
	snapshot, saved, err := progress.Load()
	if err != nil {
		return nil, fmt.Errorf("load progress: %w", err)
	}
	if !saved {
		return w, nil
	}
	w.hero.woodcutting.gain(snapshot.WoodcuttingXP)
	for _, kind := range snapshot.Inventory {
		w.hero.inventory.add(kind)
	}
	if snapshot.Seed != seed {
		return w, nil
	}
	w.hero.x, w.hero.z = snapshot.X, snapshot.Z
	w.cameraX, w.cameraZ = snapshot.X, snapshot.Z
	// A clock set back before the save counts as no time away.
	away := max(0, now.Sub(snapshot.SavedAt))
	for _, stump := range snapshot.Stumps {
		if stump.Regrow > away {
			if w.felled == nil {
				w.felled = make(map[cell]time.Duration)
			}
			w.felled[cell{stump.GX, stump.GZ}] = stump.Regrow - away
		}
	}
	return w, nil
}
