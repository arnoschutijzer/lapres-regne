package progressfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"lapresregne/internal/world"
)

// File keeps progress as JSON at Path.
type File struct {
	Path string
}

// version marks the file layout; a file of any other layout is refused.
const version = 1

// itemNames spell each item in the file, independent of how the domain names them.
var itemNames = map[world.Item]string{
	world.Logs: "logs", world.OakLogs: "oak logs", world.WillowLogs: "willow logs", world.YewLogs: "yew logs",
}

// skillNames spell each skill in the file, independent of how the domain names them.
var skillNames = map[string]string{world.Woodcutting: "woodcutting"}

type progressJSON struct {
	Version int    `json:"version"`
	Seed    uint64 `json:"seed"`
	// Map names the map the hero and respawns belong to; absent for the endless forest.
	Map     string               `json:"map,omitempty"`
	SavedAt time.Time            `json:"savedAt"`
	Hero    heroJSON             `json:"hero"`
	Skills  map[string]skillJSON `json:"skills"`
	// Inventory lists the inventory slot by slot, naming each item and leaving "" for an empty slot.
	Inventory []string `json:"inventory"`
	// Bank lists the piles in the bank in order, naming each item; absent in saves from before the bank.
	Bank []stackJSON `json:"bank"`
	// Respawns lists, by skill, the resources the skill used up that are yet to come back.
	Respawns map[string][]respawnJSON `json:"respawns"`
}

type heroJSON struct {
	X float64 `json:"x"`
	Z float64 `json:"z"`
	// Storey is absent while the hero stands on the ground floor.
	Storey int `json:"storey,omitempty"`
}

type stackJSON struct {
	Item  string `json:"item"`
	Count int    `json:"count"`
}

type skillJSON struct {
	XP float64 `json:"xp"`
}

type respawnJSON struct {
	GX       int   `json:"gx"`
	GZ       int   `json:"gz"`
	InMillis int64 `json:"inMillis"`
}

func (f File) Load() (world.Snapshot, bool, error) {
	data, err := os.ReadFile(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return world.Snapshot{}, false, nil
	}
	if err != nil {
		return world.Snapshot{}, false, err
	}
	var saved progressJSON
	if err := json.Unmarshal(data, &saved); err != nil {
		return world.Snapshot{}, false, fmt.Errorf("read %s: %w", f.Path, err)
	}
	if saved.Version != version {
		return world.Snapshot{}, false, fmt.Errorf("read %s: unknown version %d", f.Path, saved.Version)
	}
	snapshot := world.Snapshot{
		Seed: saved.Seed, Map: saved.Map, X: saved.Hero.X, Z: saved.Hero.Z, Storey: saved.Hero.Storey,
		SavedAt: saved.SavedAt,
	}
	for name, skill := range saved.Skills {
		known, ok := skillNamed(name)
		if !ok {
			return world.Snapshot{}, false, fmt.Errorf("read %s: unknown skill %q", f.Path, name)
		}
		if snapshot.Skills == nil {
			snapshot.Skills = make(map[string]float64)
		}
		snapshot.Skills[known] = skill.XP
	}
	for _, name := range saved.Inventory {
		if name == "" {
			snapshot.Inventory = append(snapshot.Inventory, world.Slot{})
			continue
		}
		item, known := itemNamed(name)
		if !known {
			return world.Snapshot{}, false, fmt.Errorf("read %s: unknown item %q", f.Path, name)
		}
		snapshot.Inventory = append(snapshot.Inventory, world.Slot{Item: item, Filled: true})
	}
	for _, stack := range saved.Bank {
		item, known := itemNamed(stack.Item)
		if !known {
			return world.Snapshot{}, false, fmt.Errorf("read %s: unknown item %q", f.Path, stack.Item)
		}
		snapshot.Bank = append(snapshot.Bank, world.Stack{Item: item, Count: stack.Count})
	}
	for name, respawns := range saved.Respawns {
		known, ok := skillNamed(name)
		if !ok {
			return world.Snapshot{}, false, fmt.Errorf("read %s: unknown skill %q", f.Path, name)
		}
		if snapshot.Respawns == nil {
			snapshot.Respawns = make(map[string][]world.Respawn)
		}
		for _, respawn := range respawns {
			snapshot.Respawns[known] = append(snapshot.Respawns[known], world.Respawn{GX: respawn.GX, GZ: respawn.GZ, In: time.Duration(respawn.InMillis) * time.Millisecond})
		}
	}
	return snapshot, true, nil
}

// Save writes a temporary file beside the save and renames it over the old one, so an
// interrupted save never leaves a half-written file behind.
func (f File) Save(snapshot world.Snapshot) error {
	saved := progressJSON{
		Version: version, Seed: snapshot.Seed, Map: snapshot.Map, SavedAt: snapshot.SavedAt,
		Hero:      heroJSON{X: snapshot.X, Z: snapshot.Z, Storey: snapshot.Storey},
		Skills:    map[string]skillJSON{},
		Inventory: []string{},
		Bank:      []stackJSON{},
		Respawns:  map[string][]respawnJSON{},
	}
	for skill, xp := range snapshot.Skills {
		name, known := skillNames[skill]
		if !known {
			return fmt.Errorf("save %s: no spelling for skill %q", f.Path, skill)
		}
		saved.Skills[name] = skillJSON{XP: xp}
	}
	// Empty slots are saved as "" so items keep their places; trailing empty slots are left off.
	last := -1
	for index, slot := range snapshot.Inventory {
		if slot.Filled {
			last = index
		}
	}
	for _, slot := range snapshot.Inventory[:last+1] {
		name := ""
		if slot.Filled {
			name = itemNames[slot.Item]
		}
		saved.Inventory = append(saved.Inventory, name)
	}
	for _, stack := range snapshot.Bank {
		saved.Bank = append(saved.Bank, stackJSON{Item: itemNames[stack.Item], Count: stack.Count})
	}
	for skill, respawns := range snapshot.Respawns {
		name, known := skillNames[skill]
		if !known {
			return fmt.Errorf("save %s: no spelling for skill %q", f.Path, skill)
		}
		saved.Respawns[name] = []respawnJSON{}
		for _, respawn := range respawns {
			saved.Respawns[name] = append(saved.Respawns[name], respawnJSON{GX: respawn.GX, GZ: respawn.GZ, InMillis: respawn.In.Milliseconds()})
		}
	}
	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(f.Path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".progress-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), f.Path)
}

func skillNamed(name string) (string, bool) {
	for skill, spelled := range skillNames {
		if spelled == name {
			return skill, true
		}
	}
	return "", false
}

func itemNamed(name string) (world.Item, bool) {
	for item, spelled := range itemNames {
		if spelled == name {
			return item, true
		}
	}
	return 0, false
}
