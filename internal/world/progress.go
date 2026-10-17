package world

import (
	"fmt"
	"time"
)

// Keep saves the hero's progress, noting now so time away can count toward regrowth.
func (w *World) Keep(progress Progress, now time.Time) error {
	snapshot := Snapshot{
		Seed:      w.seed,
		Map:       w.mapName,
		X:         w.hero.x,
		Z:         w.hero.z,
		Storey:    w.storey,
		Skills:    w.hero.experience(),
		Zoom:      w.zoom,
		Yaw:       w.yaw,
		Inventory: w.hero.inventory.Slots(),
		Stumps:    w.woodcutting.stumps(),
		SavedAt:   now,
	}
	if err := progress.Save(snapshot); err != nil {
		return fmt.Errorf("save progress: %w", err)
	}
	return nil
}

// Resume builds a world from a seed and the saved progress. The hero's skills and inventory, and
// the camera's zoom and yaw, travel to any world; their position and stumps return only to the world they were saved in, with the
// time away since saving counted toward regrowth.
func Resume(seed uint64, progress Progress, now time.Time) (*World, error) {
	return resume(New(seed), progress, now)
}

func resume(w *World, progress Progress, now time.Time) (*World, error) {
	snapshot, saved, err := progress.Load()
	if err != nil {
		return nil, fmt.Errorf("load progress: %w", err)
	}
	if !saved {
		return w, nil
	}
	for skill, xp := range snapshot.Skills {
		w.hero.train(skill, xp)
	}
	w.ZoomBy(snapshot.Zoom)
	w.yaw = snapshot.Yaw
	for index, slot := range snapshot.Inventory[:min(len(snapshot.Inventory), InventorySize)] {
		w.hero.inventory.slots[index] = slot
	}
	if snapshot.Seed != w.seed || snapshot.Map != w.mapName {
		return w, nil
	}
	// A map changed since the save may have built something where the hero stood, or taken away
	// the floor they stood on; they start afresh rather than stuck inside it.
	if snapshot.Storey >= 0 && snapshot.Storey < w.Storeys() {
		w.storey = snapshot.Storey
		if w.passable(snapshot.X, snapshot.Z) {
			w.hero.x, w.hero.z = snapshot.X, snapshot.Z
			w.cameraX, w.cameraZ = snapshot.X, snapshot.Z
		} else {
			w.storey = 0
		}
	}
	// A clock set back before the save counts as no time away.
	away := max(0, now.Sub(snapshot.SavedAt))
	for _, stump := range snapshot.Stumps {
		if stump.Regrow > away {
			w.woodcutting.fell(cell{stump.GX, stump.GZ}, stump.Regrow-away)
		}
	}
	return w, nil
}
