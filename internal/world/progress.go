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
		Inventory: w.hero.inventory.Slots(),
		Bank:      w.bank.Stacks(),
		Respawns:  map[string][]Respawn{},
		SavedAt:   now,
	}
	for skill, depleting := range w.depletingSkills() {
		snapshot.Respawns[skill] = depleting.respawns()
	}
	if err := progress.Save(snapshot); err != nil {
		return fmt.Errorf("save progress: %w", err)
	}
	return nil
}

// Resume builds a world from a seed and the saved progress. The hero's skills, inventory and bank
// travel to any world; their position and the resources used up return only to the world they were
// saved in, with the time away since saving counted toward respawning.
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
	for index, slot := range snapshot.Inventory[:min(len(snapshot.Inventory), InventorySize)] {
		w.hero.inventory.slots[index] = slot
	}
	for _, stack := range snapshot.Bank {
		if stack.Count > 0 {
			w.bank.stacks = append(w.bank.stacks, stack)
		}
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
		} else {
			w.storey = 0
		}
	}
	// A clock set back before the save counts as no time away.
	away := max(0, now.Sub(snapshot.SavedAt))
	depletingSkills := w.depletingSkills()
	for skill, respawns := range snapshot.Respawns {
		depleting, ok := depletingSkills[skill]
		if !ok {
			continue
		}
		for _, respawn := range respawns {
			if respawn.In > away {
				depleting.deplete(Respawn{GX: respawn.GX, GZ: respawn.GZ, In: respawn.In - away})
			}
		}
	}
	return w, nil
}

// depleting is a skill that uses up resources in the world, which come back after a while.
type depleting interface {
	// respawns lists the resources used up and how long each still takes to come back.
	respawns() []Respawn
	// deplete uses up a resource until it respawns.
	deplete(Respawn)
}

// depletingSkills holds the skills that use up resources in the world, by name.
func (w *World) depletingSkills() map[string]depleting {
	return map[string]depleting{Woodcutting: &w.woodcutting}
}
